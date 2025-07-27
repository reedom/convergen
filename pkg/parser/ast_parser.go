package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/internal/events"
)

// ASTParser provides event-driven AST parsing with concurrent processing
type ASTParser struct {
	logger   *zap.Logger
	eventBus events.EventBus
	cache    *TypeCache
	config   *ParserConfig
	fileSet  *token.FileSet

	// Concurrent processing
	typeResolverPool *TypeResolverPool
	mutex            sync.RWMutex
}

// ParserConfig configures the parser behavior
type ParserConfig struct {
	BuildTag              string
	MaxConcurrentWorkers  int
	TypeResolutionTimeout time.Duration
	CacheSize             int
	EnableProgress        bool
}

// NewASTParser creates a new event-driven AST parser
func NewASTParser(logger *zap.Logger, eventBus events.EventBus, config *ParserConfig) *ASTParser {
	if config == nil {
		config = &ParserConfig{
			BuildTag:              "convergen",
			MaxConcurrentWorkers:  4,
			TypeResolutionTimeout: 30 * time.Second,
			CacheSize:             1000,
			EnableProgress:        true,
		}
	}

	cache := NewTypeCache(config.CacheSize)
	typeResolverPool := NewTypeResolverPool(config.MaxConcurrentWorkers, cache, logger)

	return &ASTParser{
		logger:           logger,
		eventBus:         eventBus,
		cache:            cache,
		config:           config,
		fileSet:          token.NewFileSet(),
		typeResolverPool: typeResolverPool,
	}
}

// ParseSourceFile parses a source file and emits events throughout the process
func (p *ASTParser) ParseSourceFile(ctx context.Context, sourcePath, destPath string) ([]*domain.Method, string, error) {
	// Emit parse started event
	parseStartedEvent := events.NewParseStartedEvent(ctx, sourcePath)
	if err := p.eventBus.Publish(parseStartedEvent); err != nil {
		p.logger.Warn("failed to publish parse started event", zap.Error(err))
	}

	startTime := time.Now()
	defer func() {
		p.logger.Info("parsing completed",
			zap.String("source_path", sourcePath),
			zap.Duration("duration", time.Since(startTime)))
	}()

	// Load packages with custom configuration
	loadConfig := &packages.Config{
		Mode:       packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		BuildFlags: []string{"-tags", p.config.BuildTag},
		Fset:       p.fileSet,
		ParseFile:  p.createParseFileFunc(sourcePath, destPath),
	}

	// Load package information
	pkgs, err := packages.Load(loadConfig, "file="+sourcePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, "", fmt.Errorf("no packages found for %s", sourcePath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, "", fmt.Errorf("package errors: %v", pkg.Errors)
	}

	// Find the target source file
	var sourceFile *ast.File
	for _, file := range pkg.Syntax {
		if p.fileSet.Position(file.Pos()).Filename == sourcePath {
			sourceFile = file
			break
		}
	}

	if sourceFile == nil {
		return nil, "", fmt.Errorf("source file not found in package")
	}

	// Discover interfaces concurrently
	interfaces, err := p.discoverInterfacesConcurrently(ctx, pkg, sourceFile)
	if err != nil {
		return nil, "", fmt.Errorf("interface discovery failed: %w", err)
	}

	// Process methods concurrently
	methods, err := p.processMethodsConcurrently(ctx, pkg, sourceFile, interfaces)
	if err != nil {
		return nil, "", fmt.Errorf("method processing failed: %w", err)
	}

	// Generate base code
	baseCode, err := p.generateBaseCode(sourceFile, interfaces)
	if err != nil {
		return nil, "", fmt.Errorf("base code generation failed: %w", err)
	}

	// Create parse metrics
	metrics := &events.ParseMetrics{
		ParseDurationMS:      time.Since(startTime).Milliseconds(),
		InterfacesFound:      len(interfaces),
		MethodsProcessed:     len(methods),
		AnnotationsProcessed: p.countAnnotations(interfaces),
		TypesResolved:        p.cache.Size(),
		CacheHitRate:         p.cache.HitRate(),
	}

	// Emit parsed event
	parsedEvent := events.NewParsedEvent(ctx, methods, baseCode)
	parsedEvent.Metrics = metrics
	if err := p.eventBus.Publish(parsedEvent); err != nil {
		p.logger.Warn("failed to publish parsed event", zap.Error(err))
	}

	return methods, baseCode, nil
}

// createParseFileFunc creates a custom parse file function for packages.Load
func (p *ASTParser) createParseFileFunc(sourcePath, destPath string) func(*token.FileSet, string, []byte) (*ast.File, error) {
	return func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		// Skip destination file if it exists
		if filename == destPath {
			return nil, nil
		}

		parseMode := parser.ParseComments
		if filename != sourcePath {
			parseMode = 0 // Skip comments for non-target files
		}

		return parser.ParseFile(fset, filename, src, parseMode)
	}
}

// discoverInterfacesConcurrently discovers convergen interfaces using concurrent processing
func (p *ASTParser) discoverInterfacesConcurrently(ctx context.Context, pkg *packages.Package, file *ast.File) ([]*InterfaceInfo, error) {
	var interfaces []*InterfaceInfo
	var mutex sync.Mutex

	// Create worker group for interface discovery
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(p.config.MaxConcurrentWorkers)

	scope := pkg.Types.Scope()
	names := scope.Names()

	// Progress tracking with completion signal
	var progressDone chan struct{}
	if p.config.EnableProgress {
		progressDone = make(chan struct{})
		go p.trackProgress(ctx, domain.PhaseParsing, len(names), "Discovering interfaces", progressDone)
	}

	processed := 0
	for _, name := range names {
		name := name // Capture for goroutine
		g.Go(func() error {
			obj := scope.Lookup(name)
			if obj == nil {
				return nil
			}

			// Check if it's an interface in our source file
			iface, ok := obj.Type().Underlying().(*types.Interface)
			if !ok {
				return nil
			}

			objPos := p.fileSet.Position(obj.Pos())
			if objPos.Filename != p.fileSet.Position(file.Pos()).Filename {
				return nil
			}

			// Check if it's a convergen interface
			if p.isConvergenInterface(file, obj) {
				interfaceInfo, err := p.analyzeInterface(gctx, pkg, file, obj, iface)
				if err != nil {
					return fmt.Errorf("failed to analyze interface %s: %w", name, err)
				}

				mutex.Lock()
				interfaces = append(interfaces, interfaceInfo)
				processed++
				mutex.Unlock()

				p.logger.Debug("discovered convergen interface",
					zap.String("name", name),
					zap.String("position", objPos.String()))
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Signal progress completion before returning error
		if progressDone != nil {
			close(progressDone)
		}
		return nil, err
	}

	// Signal progress completion
	if progressDone != nil {
		close(progressDone)
	}

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no convergen interfaces found")
	}

	p.logger.Info("interface discovery completed",
		zap.Int("interfaces_found", len(interfaces)),
		zap.Int("total_scanned", len(names)))

	return interfaces, nil
}

// processMethodsConcurrently processes methods from discovered interfaces
func (p *ASTParser) processMethodsConcurrently(ctx context.Context, pkg *packages.Package, file *ast.File, interfaces []*InterfaceInfo) ([]*domain.Method, error) {
	var allMethods []*domain.Method
	var mutex sync.Mutex

	// Create worker group for method processing
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(p.config.MaxConcurrentWorkers)

	totalMethods := 0
	for _, iface := range interfaces {
		totalMethods += len(iface.Methods)
	}

	// Progress tracking with completion signal
	var progressDone chan struct{}
	if p.config.EnableProgress {
		progressDone = make(chan struct{})
		go p.trackProgress(ctx, domain.PhaseParsing, totalMethods, "Processing methods", progressDone)
	}

	for _, iface := range interfaces {
		iface := iface // Capture for goroutine
		for _, methodObj := range iface.Methods {
			methodObj := methodObj // Capture for goroutine
			g.Go(func() error {
				method, err := p.processMethod(gctx, pkg, file, methodObj, iface.Options)
				if err != nil {
					return fmt.Errorf("failed to process method %s: %w", methodObj.Name(), err)
				}

				mutex.Lock()
				allMethods = append(allMethods, method)
				mutex.Unlock()

				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		// Signal progress completion before returning error
		if progressDone != nil {
			close(progressDone)
		}
		return nil, err
	}

	// Signal progress completion
	if progressDone != nil {
		close(progressDone)
	}

	// Resolve cross-references between methods
	if err := p.resolveCrossReferences(ctx, allMethods); err != nil {
		return nil, fmt.Errorf("cross-reference resolution failed: %w", err)
	}

	return allMethods, nil
}

// trackProgress emits progress events during processing with adaptive frequency
func (p *ASTParser) trackProgress(ctx context.Context, phase domain.ProcessingPhase, total int, message string, done <-chan struct{}) {
	// Adaptive reporting frequency based on operation complexity
	interval := p.calculateProgressInterval(total)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastProgressTime := time.Now()
	reportCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			// Operation completed, emit final progress event
			if reportCount > 0 { // Only emit final if we've been reporting
				finalEvent := events.NewProgressEvent(ctx, phase, total, total, message+" (completed)")
				if err := p.eventBus.Publish(finalEvent); err != nil {
					p.logger.Debug("failed to publish final progress event", zap.Error(err))
				}
			}
			return
		case <-ticker.C:
			// Adaptive progress reporting with throttling
			if p.shouldReportProgress(lastProgressTime, reportCount, total) {
				progressEvent := events.NewProgressEvent(ctx, phase, 0, total, message)
				if err := p.eventBus.Publish(progressEvent); err != nil {
					// Demote to debug level to reduce noise
					p.logger.Debug("failed to publish progress event", zap.Error(err))
				}
				lastProgressTime = time.Now()
				reportCount++

				// Dynamically adjust frequency for long-running operations
				if reportCount > 10 && total > 100 {
					ticker.Reset(interval * 2) // Slow down reporting for very long operations
				}
			}
		}
	}
}

// calculateProgressInterval determines the optimal progress reporting interval
func (p *ASTParser) calculateProgressInterval(total int) time.Duration {
	switch {
	case total <= 5:
		// Very small operations - no progress tracking needed
		return 1 * time.Hour // Effectively disable
	case total <= 20:
		// Small operations - infrequent reporting
		return 500 * time.Millisecond
	case total <= 100:
		// Medium operations - moderate reporting
		return 200 * time.Millisecond
	case total <= 500:
		// Large operations - frequent reporting
		return 100 * time.Millisecond
	default:
		// Very large operations - very frequent initial reporting
		return 50 * time.Millisecond
	}
}

// shouldReportProgress determines if progress should be reported based on various factors
func (p *ASTParser) shouldReportProgress(lastReport time.Time, reportCount int, total int) bool {
	// For very small operations, don't report at all
	if total <= 5 {
		return false
	}

	// For the first few reports, always report
	if reportCount < 3 {
		return true
	}

	// For long-running operations, throttle reporting
	if total > 100 && reportCount > 20 {
		// Report less frequently for very long operations
		return time.Since(lastReport) > 1*time.Second
	}

	// Standard reporting
	return true
}

// countAnnotations counts total annotations across all interfaces
func (p *ASTParser) countAnnotations(interfaces []*InterfaceInfo) int {
	count := 0
	for _, iface := range interfaces {
		count += len(iface.Annotations)
		for _, method := range iface.Methods {
			// Count method-level annotations
			count += p.countMethodAnnotations(method)
		}
	}
	return count
}

// countMethodAnnotations counts annotations for a specific method
func (p *ASTParser) countMethodAnnotations(method types.Object) int {
	// This would analyze the method's documentation for annotations
	// Implementation details depend on the annotation format
	return 0 // Placeholder
}

// Close releases resources used by the parser
func (p *ASTParser) Close() error {
	if p.typeResolverPool != nil {
		return p.typeResolverPool.Close()
	}
	return nil
}
