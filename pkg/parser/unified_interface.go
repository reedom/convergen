package parser

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/reedom/convergen/v8/pkg/builder/model"
	"github.com/reedom/convergen/v8/pkg/domain"
)

// ParseStrategy defines different parsing approaches
type ParseStrategy int

const (
	StrategyLegacy ParseStrategy = iota // Traditional synchronous parsing
	StrategyModern                      // Event-driven concurrent parsing
	StrategyAuto                        // Automatically choose based on input complexity
)

// ParseResult contains comprehensive parsing results
type ParseResult struct {
	Methods        []*model.MethodEntry   `json:"methods"`
	DomainMethods  []*domain.Method       `json:"domain_methods,omitempty"`
	BaseCode       string                 `json:"base_code"`
	Metrics        *ParseMetrics          `json:"metrics"`
	Interfaces     []*ParsedInterfaceInfo `json:"interfaces"`
	Errors         []ParseError           `json:"errors,omitempty"`
	Warnings       []ParseWarning         `json:"warnings,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Strategy       ParseStrategy          `json:"strategy"`
}

// ParseMetrics provides detailed parsing performance information
type ParseMetrics struct {
	TotalFiles         int           `json:"total_files"`
	TotalInterfaces    int           `json:"total_interfaces"`
	TotalMethods       int           `json:"total_methods"`
	ProcessedMethods   int           `json:"processed_methods"`
	FailedMethods      int           `json:"failed_methods"`
	ParsingTime        time.Duration `json:"parsing_time"`
	TypeResolutionTime time.Duration `json:"type_resolution_time"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	ConcurrencyLevel   int           `json:"concurrency_level"`
	MemoryUsagePeakMB  float64       `json:"memory_usage_peak_mb"`
}

// ParsedInterfaceInfo contains information about discovered interfaces for unified parser results
type ParsedInterfaceInfo struct {
	Name        string   `json:"name"`
	PackageName string   `json:"package_name"`
	MethodCount int      `json:"method_count"`
	IsGeneric   bool     `json:"is_generic"`
	Annotations []string `json:"annotations"`
	Location    string   `json:"location"`
}

// ParseError represents a parsing error with rich context
type ParseError struct {
	Code        string                 `json:"code"`
	Phase       ProcessingPhase        `json:"phase"`
	Message     string                 `json:"message"`
	Location    string                 `json:"location,omitempty"`
	Interface   string                 `json:"interface,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ParseWarning represents a non-fatal parsing issue
type ParseWarning struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Location   string                 `json:"location,omitempty"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ProcessingPhase represents different phases of parsing
type ProcessingPhase string

const (
	PhasePackageLoading       ProcessingPhase = "package_loading"
	PhaseInterfaceDiscovery   ProcessingPhase = "interface_discovery"
	PhaseMethodAnalysis       ProcessingPhase = "method_analysis"
	PhaseTypeResolution       ProcessingPhase = "type_resolution"
	PhaseAnnotationProcessing ProcessingPhase = "annotation_processing"
	PhaseValidation           ProcessingPhase = "validation"
	PhaseCodeGeneration       ProcessingPhase = "code_generation"
)

// ConvergenParser defines the unified interface for all parser implementations
type ConvergenParser interface {
	// ParseSourceFile parses a source file and returns comprehensive results
	ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error)

	// ParseSourceFiles parses multiple source files concurrently
	ParseSourceFiles(ctx context.Context, files []SourceFile) ([]*ParseResult, error)

	// SetConfig updates the parser configuration
	SetConfig(config *ParserConfig) error

	// GetConfig returns the current parser configuration
	GetConfig() *ParserConfig

	// GetMetrics returns current parsing metrics
	GetMetrics() *ParseMetrics

	// Validate performs validation without full parsing
	Validate(ctx context.Context, sourcePath string) ([]ParseError, []ParseWarning, error)

	// Close releases resources and performs cleanup
	Close() error

	// GetStrategy returns the parsing strategy being used
	GetStrategy() ParseStrategy
}

// SourceFile represents a source file to be parsed
type SourceFile struct {
	Path       string        `json:"path"`
	DestPath   string        `json:"dest_path,omitempty"`
	Priority   int           `json:"priority,omitempty"`
	MaxMethods int           `json:"max_methods,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
}

// ParserFactory creates parser instances based on strategy
type ParserFactory struct {
	defaultConfig *ParserConfig
}

// NewParserFactory creates a new parser factory
func NewParserFactory(defaultConfig *ParserConfig) *ParserFactory {
	return &ParserFactory{
		defaultConfig: EnsureValidConfig(defaultConfig),
	}
}

// CreateParser creates a parser instance using the specified strategy
func (pf *ParserFactory) CreateParser(strategy ParseStrategy) (ConvergenParser, error) {
	switch strategy {
	case StrategyLegacy:
		return NewLegacyParser(pf.defaultConfig), nil
	case StrategyModern:
		return NewModernParser(pf.defaultConfig), nil
	case StrategyAuto:
		return NewAdaptiveParser(pf.defaultConfig), nil
	default:
		return NewLegacyParser(pf.defaultConfig), nil
	}
}

// CreateParserWithConfig creates a parser with custom configuration
func (pf *ParserFactory) CreateParserWithConfig(strategy ParseStrategy, config *ParserConfig) (ConvergenParser, error) {
	switch strategy {
	case StrategyLegacy:
		return NewLegacyParser(config), nil
	case StrategyModern:
		return NewModernParser(config), nil
	case StrategyAuto:
		return NewAdaptiveParser(config), nil
	default:
		return NewLegacyParser(config), nil
	}
}

// GetSupportedStrategies returns all supported parsing strategies
func (pf *ParserFactory) GetSupportedStrategies() []ParseStrategy {
	return []ParseStrategy{StrategyLegacy, StrategyModern, StrategyAuto}
}

// GetStrategyName returns the human-readable name of a parsing strategy
func GetStrategyName(strategy ParseStrategy) string {
	switch strategy {
	case StrategyLegacy:
		return "Legacy Sequential Parser"
	case StrategyModern:
		return "Modern Concurrent Parser"
	case StrategyAuto:
		return "Adaptive Parser"
	default:
		return "Unknown Strategy"
	}
}

// GetRecommendedStrategy returns the recommended strategy based on input characteristics
func GetRecommendedStrategy(fileCount int, averageMethodsPerInterface int, totalMethods int) ParseStrategy {
	// Simple heuristics for strategy selection
	if totalMethods > 50 || fileCount > 3 {
		return StrategyModern // Use concurrent parsing for complex scenarios
	}

	if averageMethodsPerInterface > 20 {
		return StrategyModern // Use concurrent parsing for interfaces with many methods
	}

	return StrategyLegacy // Use legacy parsing for simple scenarios
}

// Error implements the error interface for ParseError
func (pe *ParseError) Error() string {
	if pe.Location != "" {
		return pe.Location + ": " + pe.Message
	}
	return pe.Message
}

// Error implements the error interface for ParseWarning
func (pw *ParseWarning) Error() string {
	if pw.Location != "" {
		return pw.Location + ": " + pw.Message
	}
	return pw.Message
}

// LegacyParser wraps the existing Parser for backward compatibility
type LegacyParser struct {
	config   *ParserConfig
	metrics  *ParseMetrics
	parser   *Parser
	strategy ParseStrategy
}

// NewLegacyParser creates a new legacy parser instance
func NewLegacyParser(config *ParserConfig) *LegacyParser {
	validConfig := EnsureValidConfig(config)
	// Legacy parser should never have concurrency enabled
	validConfig.EnableConcurrentLoading = false
	validConfig.EnableMethodConcurrency = false

	return &LegacyParser{
		config:   validConfig,
		metrics:  &ParseMetrics{},
		strategy: StrategyLegacy,
	}
}

// ParseSourceFile implements ConvergenParser interface
func (lp *LegacyParser) ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error) {
	startTime := time.Now()

	// Create parser instance
	parser, err := NewParserWithConfig(sourcePath, destPath, lp.config)
	if err != nil {
		return &ParseResult{
			Errors:         []ParseError{{Code: "PARSER_CREATE_FAILED", Message: err.Error(), Phase: PhasePackageLoading}},
			ProcessingTime: time.Since(startTime),
			Strategy:       lp.strategy,
		}, err
	}

	lp.parser = parser

	// Parse the source file
	methodsInfo, err := parser.Parse()
	if err != nil {
		return &ParseResult{
			Errors:         []ParseError{{Code: "PARSE_FAILED", Message: err.Error(), Phase: PhaseMethodAnalysis}},
			ProcessingTime: time.Since(startTime),
			Strategy:       lp.strategy,
		}, err
	}

	// Generate base code
	baseCode, err := parser.GenerateBaseCode()
	if err != nil {
		return &ParseResult{
			Warnings:       []ParseWarning{{Code: "BASE_CODE_GENERATION_FAILED", Message: err.Error()}},
			ProcessingTime: time.Since(startTime),
			Strategy:       lp.strategy,
		}, nil // Don't fail for base code generation issues
	}

	// Collect all methods
	var allMethods []*model.MethodEntry
	var interfaces []*ParsedInterfaceInfo

	for _, info := range methodsInfo {
		allMethods = append(allMethods, info.Methods...)
		interfaces = append(interfaces, &ParsedInterfaceInfo{
			Name:        "convergen", // Legacy parsers don't track interface names
			MethodCount: len(info.Methods),
			Location:    sourcePath,
		})
	}

	// Update metrics
	lp.updateMetrics(len(methodsInfo), len(allMethods), time.Since(startTime))

	return &ParseResult{
		Methods:        allMethods,
		BaseCode:       baseCode,
		Metrics:        lp.metrics,
		Interfaces:     interfaces,
		ProcessingTime: time.Since(startTime),
		Strategy:       lp.strategy,
	}, nil
}

// ParseSourceFiles implements ConvergenParser interface
func (lp *LegacyParser) ParseSourceFiles(ctx context.Context, files []SourceFile) ([]*ParseResult, error) {
	results := make([]*ParseResult, len(files))

	for i, file := range files {
		result, err := lp.ParseSourceFile(ctx, file.Path, file.DestPath)
		if err != nil {
			result = &ParseResult{
				Errors:         []ParseError{{Code: "FILE_PARSE_FAILED", Message: err.Error(), Location: file.Path}},
				ProcessingTime: 0,
				Strategy:       lp.strategy,
			}
		}
		results[i] = result
	}

	return results, nil
}

// SetConfig implements ConvergenParser interface
func (lp *LegacyParser) SetConfig(config *ParserConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	lp.config = config
	return nil
}

// GetConfig implements ConvergenParser interface
func (lp *LegacyParser) GetConfig() *ParserConfig {
	return lp.config
}

// GetMetrics implements ConvergenParser interface
func (lp *LegacyParser) GetMetrics() *ParseMetrics {
	return lp.metrics
}

// Validate implements ConvergenParser interface
func (lp *LegacyParser) Validate(ctx context.Context, sourcePath string) ([]ParseError, []ParseWarning, error) {
	// Basic validation - check if file exists and is readable
	if _, err := os.Stat(sourcePath); err != nil {
		return []ParseError{{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("source file not found: %s", sourcePath),
			Phase:   PhasePackageLoading,
		}}, nil, err
	}

	return nil, nil, nil
}

// Close implements ConvergenParser interface
func (lp *LegacyParser) Close() error {
	// Legacy parser doesn't require cleanup
	return nil
}

// GetStrategy implements ConvergenParser interface
func (lp *LegacyParser) GetStrategy() ParseStrategy {
	return lp.strategy
}

// updateMetrics updates internal metrics
func (lp *LegacyParser) updateMetrics(interfaceCount, methodCount int, processingTime time.Duration) {
	lp.metrics.TotalInterfaces = interfaceCount
	lp.metrics.TotalMethods = methodCount
	lp.metrics.ProcessedMethods = methodCount
	lp.metrics.ParsingTime = processingTime
	lp.metrics.ConcurrencyLevel = 1 // Legacy is always sequential
}

// ModernParser uses concurrent processing and enhanced features
type ModernParser struct {
	config        *ParserConfig
	metrics       *ParseMetrics
	parser        *Parser
	packageLoader *PackageLoader
	strategy      ParseStrategy
}

// NewModernParser creates a new modern parser instance with concurrent capabilities
func NewModernParser(config *ParserConfig) *ModernParser {
	validConfig := EnsureValidConfig(config)
	// Modern parser should always have concurrency enabled
	validConfig.EnableConcurrentLoading = true
	validConfig.EnableMethodConcurrency = true

	return &ModernParser{
		config:        validConfig,
		metrics:       &ParseMetrics{},
		packageLoader: NewPackageLoader(validConfig.MaxConcurrentWorkers, validConfig.TypeResolutionTimeout),
		strategy:      StrategyModern,
	}
}

// ParseSourceFile implements ConvergenParser interface with concurrent processing
func (mp *ModernParser) ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error) {
	startTime := time.Now()

	// Create parser instance with concurrent loading enabled
	parser, err := NewParserWithConfig(sourcePath, destPath, mp.config)
	if err != nil {
		return &ParseResult{
			Errors:         []ParseError{{Code: "PARSER_CREATE_FAILED", Message: err.Error(), Phase: PhasePackageLoading}},
			ProcessingTime: time.Since(startTime),
			Strategy:       mp.strategy,
		}, err
	}

	mp.parser = parser

	// Parse with concurrent processing
	methodsInfo, err := parser.Parse()
	if err != nil {
		return &ParseResult{
			Errors:         []ParseError{{Code: "PARSE_FAILED", Message: err.Error(), Phase: PhaseMethodAnalysis}},
			ProcessingTime: time.Since(startTime),
			Strategy:       mp.strategy,
		}, err
	}

	// Generate base code
	baseCode, err := parser.GenerateBaseCode()
	if err != nil {
		return &ParseResult{
			Warnings:       []ParseWarning{{Code: "BASE_CODE_GENERATION_FAILED", Message: err.Error()}},
			ProcessingTime: time.Since(startTime),
			Strategy:       mp.strategy,
		}, nil
	}

	// Collect all methods and enhanced interface information
	var allMethods []*model.MethodEntry
	var interfaces []*ParsedInterfaceInfo

	for _, info := range methodsInfo {
		allMethods = append(allMethods, info.Methods...)
		interfaces = append(interfaces, &ParsedInterfaceInfo{
			Name:        "convergen",
			MethodCount: len(info.Methods),
			Location:    sourcePath,
		})
	}

	// Get package loader metrics for enhanced reporting
	cacheHits, cacheMisses := mp.packageLoader.GetCacheStats()
	cacheHitRate := 0.0
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses)
	}

	// Update metrics with concurrent processing information
	mp.updateMetrics(len(methodsInfo), len(allMethods), time.Since(startTime), cacheHitRate)

	return &ParseResult{
		Methods:        allMethods,
		BaseCode:       baseCode,
		Metrics:        mp.metrics,
		Interfaces:     interfaces,
		ProcessingTime: time.Since(startTime),
		Strategy:       mp.strategy,
	}, nil
}

// ParseSourceFiles implements ConvergenParser interface with enhanced concurrent processing
func (mp *ModernParser) ParseSourceFiles(ctx context.Context, files []SourceFile) ([]*ParseResult, error) {
	results := make([]*ParseResult, len(files))

	// Process files concurrently if we have multiple files
	if len(files) > 1 && mp.config.MaxConcurrentWorkers > 1 {
		return mp.parseSourceFilesConcurrent(ctx, files)
	}

	// Sequential processing for single file or when concurrency is limited
	for i, file := range files {
		result, err := mp.ParseSourceFile(ctx, file.Path, file.DestPath)
		if err != nil {
			result = &ParseResult{
				Errors:         []ParseError{{Code: "FILE_PARSE_FAILED", Message: err.Error(), Location: file.Path}},
				ProcessingTime: 0,
				Strategy:       mp.strategy,
			}
		}
		results[i] = result
	}

	return results, nil
}

// parseSourceFilesConcurrent processes multiple files concurrently
func (mp *ModernParser) parseSourceFilesConcurrent(ctx context.Context, files []SourceFile) ([]*ParseResult, error) {
	results := make([]*ParseResult, len(files))

	// Use errgroup for concurrent processing with limited concurrency
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(mp.config.MaxConcurrentWorkers)

	for i, file := range files {
		i, file := i, file // Capture loop variables
		g.Go(func() error {
			result, err := mp.ParseSourceFile(gctx, file.Path, file.DestPath)
			if err != nil {
				result = &ParseResult{
					Errors:         []ParseError{{Code: "FILE_PARSE_FAILED", Message: err.Error(), Location: file.Path}},
					ProcessingTime: 0,
					Strategy:       mp.strategy,
				}
			}
			results[i] = result
			return nil // Don't fail the entire group for individual file failures
		})
	}

	if err := g.Wait(); err != nil {
		return results, fmt.Errorf("concurrent file processing failed: %w", err)
	}

	return results, nil
}

// SetConfig implements ConvergenParser interface
func (mp *ModernParser) SetConfig(config *ParserConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Ensure concurrency is enabled for modern parser
	config.EnableConcurrentLoading = true
	config.EnableMethodConcurrency = true

	mp.config = config

	// Update package loader with new configuration
	mp.packageLoader = NewPackageLoader(config.MaxConcurrentWorkers, config.TypeResolutionTimeout)

	return nil
}

// GetConfig implements ConvergenParser interface
func (mp *ModernParser) GetConfig() *ParserConfig {
	return mp.config
}

// GetMetrics implements ConvergenParser interface
func (mp *ModernParser) GetMetrics() *ParseMetrics {
	return mp.metrics
}

// Validate implements ConvergenParser interface with enhanced validation
func (mp *ModernParser) Validate(ctx context.Context, sourcePath string) ([]ParseError, []ParseWarning, error) {
	var errors []ParseError
	var warnings []ParseWarning

	// Check if file exists and is readable
	if _, err := os.Stat(sourcePath); err != nil {
		errors = append(errors, ParseError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("source file not found: %s", sourcePath),
			Phase:   PhasePackageLoading,
		})
		return errors, warnings, err
	}

	// Enhanced validation using package loader
	result, err := mp.packageLoader.LoadPackageConcurrent(ctx, sourcePath, "")
	if err != nil {
		errors = append(errors, ParseError{
			Code:    "PACKAGE_LOAD_FAILED",
			Message: fmt.Sprintf("package loading failed: %s", err.Error()),
			Phase:   PhasePackageLoading,
		})
		return errors, warnings, err
	}

	if result.Package == nil {
		warnings = append(warnings, ParseWarning{
			Code:    "PACKAGE_INFO_MISSING",
			Message: "package information could not be loaded",
		})
	}

	if result.File == nil {
		warnings = append(warnings, ParseWarning{
			Code:    "FILE_PARSE_INCOMPLETE",
			Message: "file parsing incomplete",
		})
	}

	return errors, warnings, nil
}

// Close implements ConvergenParser interface
func (mp *ModernParser) Close() error {
	if mp.packageLoader != nil {
		mp.packageLoader.ClearCache()
	}
	return nil
}

// GetStrategy implements ConvergenParser interface
func (mp *ModernParser) GetStrategy() ParseStrategy {
	return mp.strategy
}

// updateMetrics updates internal metrics with concurrent processing information
func (mp *ModernParser) updateMetrics(interfaceCount, methodCount int, processingTime time.Duration, cacheHitRate float64) {
	mp.metrics.TotalInterfaces = interfaceCount
	mp.metrics.TotalMethods = methodCount
	mp.metrics.ProcessedMethods = methodCount
	mp.metrics.ParsingTime = processingTime
	mp.metrics.ConcurrencyLevel = mp.config.MaxConcurrentWorkers
	mp.metrics.CacheHitRate = cacheHitRate
}

// AdaptiveParser dynamically chooses between legacy and modern strategies
type AdaptiveParser struct {
	config           *ParserConfig
	metrics          *ParseMetrics
	currentParser    ConvergenParser
	strategy         ParseStrategy
	adaptiveStrategy ParseStrategy
}

// NewAdaptiveParser creates a new adaptive parser instance
func NewAdaptiveParser(config *ParserConfig) *AdaptiveParser {
	validConfig := EnsureValidConfig(config)
	// Adaptive parser starts with concurrency disabled and enables it as needed
	validConfig.EnableConcurrentLoading = false
	validConfig.EnableMethodConcurrency = false

	return &AdaptiveParser{
		config:   validConfig,
		metrics:  &ParseMetrics{},
		strategy: StrategyAuto,
	}
}

// ParseSourceFile implements ConvergenParser interface with adaptive strategy selection
func (ap *AdaptiveParser) ParseSourceFile(ctx context.Context, sourcePath, destPath string) (*ParseResult, error) {
	startTime := time.Now()

	// Analyze input to determine optimal strategy
	strategy := ap.determineStrategy(sourcePath, destPath)

	// Create appropriate parser based on determined strategy
	parser, err := ap.createParserForStrategy(strategy)
	if err != nil {
		return &ParseResult{
			Errors:         []ParseError{{Code: "ADAPTIVE_PARSER_CREATE_FAILED", Message: err.Error(), Phase: PhasePackageLoading}},
			ProcessingTime: time.Since(startTime),
			Strategy:       ap.strategy,
		}, err
	}

	ap.currentParser = parser
	ap.adaptiveStrategy = strategy

	// Parse using selected strategy
	result, err := parser.ParseSourceFile(ctx, sourcePath, destPath)
	if err != nil {
		return result, err
	}

	// Update result to reflect adaptive strategy
	result.Strategy = ap.strategy

	// Collect metrics from underlying parser
	ap.updateMetricsFromParser(parser, time.Since(startTime))

	return result, nil
}

// ParseSourceFiles implements ConvergenParser interface with adaptive multi-file strategy
func (ap *AdaptiveParser) ParseSourceFiles(ctx context.Context, files []SourceFile) ([]*ParseResult, error) {
	if len(files) == 0 {
		return []*ParseResult{}, nil
	}

	// Analyze files to determine optimal strategy
	strategy := ap.determineMultiFileStrategy(files)

	// Create appropriate parser
	parser, err := ap.createParserForStrategy(strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create adaptive parser: %w", err)
	}

	ap.currentParser = parser
	ap.adaptiveStrategy = strategy

	// Parse using selected strategy
	results, err := parser.ParseSourceFiles(ctx, files)
	if err != nil {
		return results, err
	}

	// Update all results to reflect adaptive strategy
	for _, result := range results {
		if result != nil {
			result.Strategy = ap.strategy
		}
	}

	return results, nil
}

// SetConfig implements ConvergenParser interface
func (ap *AdaptiveParser) SetConfig(config *ParserConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	ap.config = config

	// Update current parser if it exists
	if ap.currentParser != nil {
		return ap.currentParser.SetConfig(config)
	}

	return nil
}

// GetConfig implements ConvergenParser interface
func (ap *AdaptiveParser) GetConfig() *ParserConfig {
	return ap.config
}

// GetMetrics implements ConvergenParser interface
func (ap *AdaptiveParser) GetMetrics() *ParseMetrics {
	if ap.currentParser != nil {
		// Get metrics from current parser and merge with adaptive metrics
		currentMetrics := ap.currentParser.GetMetrics()
		ap.metrics.TotalFiles = currentMetrics.TotalFiles
		ap.metrics.TotalInterfaces = currentMetrics.TotalInterfaces
		ap.metrics.TotalMethods = currentMetrics.TotalMethods
		ap.metrics.ProcessedMethods = currentMetrics.ProcessedMethods
		ap.metrics.FailedMethods = currentMetrics.FailedMethods
		ap.metrics.ParsingTime = currentMetrics.ParsingTime
		ap.metrics.TypeResolutionTime = currentMetrics.TypeResolutionTime
		ap.metrics.CacheHitRate = currentMetrics.CacheHitRate
		ap.metrics.ConcurrencyLevel = currentMetrics.ConcurrencyLevel
		ap.metrics.MemoryUsagePeakMB = currentMetrics.MemoryUsagePeakMB
	}
	return ap.metrics
}

// Validate implements ConvergenParser interface
func (ap *AdaptiveParser) Validate(ctx context.Context, sourcePath string) ([]ParseError, []ParseWarning, error) {
	// Use modern parser for validation as it has more comprehensive validation
	modernParser := NewModernParser(ap.config)
	defer modernParser.Close()

	return modernParser.Validate(ctx, sourcePath)
}

// Close implements ConvergenParser interface
func (ap *AdaptiveParser) Close() error {
	if ap.currentParser != nil {
		return ap.currentParser.Close()
	}
	return nil
}

// GetStrategy implements ConvergenParser interface
func (ap *AdaptiveParser) GetStrategy() ParseStrategy {
	return ap.strategy
}

// GetAdaptiveStrategy returns the actual strategy being used by the adaptive parser
func (ap *AdaptiveParser) GetAdaptiveStrategy() ParseStrategy {
	return ap.adaptiveStrategy
}

// determineStrategy analyzes input characteristics to determine optimal parsing strategy
func (ap *AdaptiveParser) determineStrategy(sourcePath, destPath string) ParseStrategy {
	// Quick heuristics for single file analysis

	// Check file size - large files benefit from concurrent processing
	if stat, err := os.Stat(sourcePath); err == nil {
		fileSize := stat.Size()
		if fileSize > 100*1024 { // Files larger than 100KB
			return StrategyModern
		}
	}

	// For small files, use legacy parser for simplicity
	return StrategyLegacy
}

// determineMultiFileStrategy analyzes multiple files to determine optimal strategy
func (ap *AdaptiveParser) determineMultiFileStrategy(files []SourceFile) ParseStrategy {
	fileCount := len(files)

	// Use recommendation system
	// For multi-file scenarios, estimate average methods per interface
	estimatedMethods := fileCount * 5 // Conservative estimate
	averageMethodsPerInterface := 5   // Conservative estimate

	return GetRecommendedStrategy(fileCount, averageMethodsPerInterface, estimatedMethods)
}

// createParserForStrategy creates the appropriate parser for the given strategy
func (ap *AdaptiveParser) createParserForStrategy(strategy ParseStrategy) (ConvergenParser, error) {
	// Clone config to avoid modifying the original
	configCopy := CloneConfig(ap.config)

	switch strategy {
	case StrategyLegacy:
		return NewLegacyParser(configCopy), nil
	case StrategyModern:
		return NewModernParser(configCopy), nil
	default:
		// Default to legacy for unknown strategies
		return NewLegacyParser(configCopy), nil
	}
}

// updateMetricsFromParser updates adaptive parser metrics from underlying parser
func (ap *AdaptiveParser) updateMetricsFromParser(parser ConvergenParser, totalTime time.Duration) {
	if parser == nil {
		return
	}

	metrics := parser.GetMetrics()
	if metrics == nil {
		return
	}

	// Copy metrics from underlying parser
	ap.metrics = metrics

	// Add adaptive-specific information
	// Total processing time includes strategy selection overhead
	ap.metrics.ParsingTime = totalTime
}
