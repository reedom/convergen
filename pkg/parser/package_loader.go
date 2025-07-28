package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"
)

// PackageLoadResult contains the result of loading a package
type PackageLoadResult struct {
	Package    *packages.Package
	File       *ast.File
	FileSet    *token.FileSet
	SourcePath string
	Error      error
}

// PackageLoader provides concurrent package loading capabilities
type PackageLoader struct {
	maxWorkers int
	timeout    time.Duration
	pool       *sync.Pool
	mutex      sync.RWMutex
	cache      map[string]*PackageLoadResult
}

// NewPackageLoader creates a new concurrent package loader
func NewPackageLoader(maxWorkers int, timeout time.Duration) *PackageLoader {
	return &PackageLoader{
		maxWorkers: maxWorkers,
		timeout:    timeout,
		cache:      make(map[string]*PackageLoadResult),
		pool: &sync.Pool{
			New: func() interface{} {
				return &packages.Config{
					Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
						packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
				}
			},
		},
	}
}

// LoadPackageConcurrent loads a single package with concurrent optimization
func (pl *PackageLoader) LoadPackageConcurrent(ctx context.Context, sourcePath, destPath string) (*PackageLoadResult, error) {
	// Check cache first
	pl.mutex.RLock()
	if cached, exists := pl.cache[sourcePath]; exists {
		pl.mutex.RUnlock()
		return cached, cached.Error
	}
	pl.mutex.RUnlock()

	// Create context with timeout
	loadCtx, cancel := context.WithTimeout(ctx, pl.timeout)
	defer cancel()

	result := &PackageLoadResult{
		SourcePath: sourcePath,
	}

	// Load package information
	if err := pl.loadPackageInfo(loadCtx, sourcePath, destPath, result); err != nil {
		result.Error = err
		return result, err
	}

	// Cache successful result
	pl.mutex.Lock()
	pl.cache[sourcePath] = result
	pl.mutex.Unlock()

	return result, nil
}

// LoadPackagesConcurrent loads multiple packages concurrently
func (pl *PackageLoader) LoadPackagesConcurrent(ctx context.Context, paths []string) ([]*PackageLoadResult, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no packages to load")
	}

	// Use errgroup for concurrent loading with limited workers
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pl.maxWorkers)

	results := make([]*PackageLoadResult, len(paths))

	for i, path := range paths {
		i, path := i, path // Capture loop variables
		g.Go(func() error {
			result, err := pl.LoadPackageConcurrent(gctx, path, "")
			results[i] = result
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return results, fmt.Errorf("failed to load packages concurrently: %w", err)
	}

	return results, nil
}

// loadPackageInfo performs the actual package loading
func (pl *PackageLoader) loadPackageInfo(ctx context.Context, sourcePath, destPath string, result *PackageLoadResult) error {
	// Check file stats for optimization decisions
	srcStat, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", sourcePath, err)
	}

	// Prepare file set and config
	result.FileSet = token.NewFileSet()
	cfg := pl.pool.Get().(*packages.Config)
	defer pl.pool.Put(cfg)

	// Configure package loading with concurrent-safe settings
	cfg.Context = ctx
	cfg.Fset = result.FileSet
	cfg.ParseFile = func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Parse with concurrent-optimized settings
		file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
		}
		result.File = file
		return file, nil
	}

	// Load package with timeout protection
	loadChan := make(chan error, 1)
	go func() {
		pkgs, err := packages.Load(cfg, "file="+sourcePath)
		if err != nil {
			loadChan <- fmt.Errorf("failed to load package info for %s: %w", sourcePath, err)
			return
		}
		if len(pkgs) == 0 {
			loadChan <- fmt.Errorf("no packages found for %s", sourcePath)
			return
		}
		result.Package = pkgs[0]
		loadChan <- nil
	}()

	select {
	case err := <-loadChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("package loading timeout for %s: %w", sourcePath, ctx.Err())
	}

	// Validate loaded package
	if result.Package == nil {
		return fmt.Errorf("failed to load package information for %s", sourcePath)
	}
	if result.File == nil {
		return fmt.Errorf("failed to parse source file %s", sourcePath)
	}

	// Perform optimization checks
	if srcStat.Size() > 1024*1024 { // Files > 1MB
		// Log large file warning (can be enhanced with proper logging later)
		fmt.Printf("Large file detected (%d bytes): %s - consider using incremental parsing\n", srcStat.Size(), sourcePath)
	}

	return nil
}

// ClearCache clears the package loading cache
func (pl *PackageLoader) ClearCache() {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	pl.cache = make(map[string]*PackageLoadResult)
}

// GetCacheStats returns cache statistics
func (pl *PackageLoader) GetCacheStats() (hits, misses int) {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	return len(pl.cache), 0 // Simplified stats for now
}
