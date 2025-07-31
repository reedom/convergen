package parser

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/reedom/convergen/v8/pkg/builder/model"
	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/option"
)

// Static errors for err113 compliance.
var (
	ErrMethodProcessingPanic     = errors.New("method processing panic")
	ErrExpectedSignature         = errors.New("expected signature but got different type")
	ErrMethodMustHaveParameters  = errors.New("method must have one or more parameters as copy source")
	ErrMethodMustHaveReturnValues = errors.New("method must have one or more return values as copy destination")
	ErrAllMethodsFailed          = errors.New("all methods failed to process")
	ErrPartialProcessingFailure  = errors.New("partial processing failure")
)

// MethodProcessingResult contains the result of processing a method.
type MethodProcessingResult struct {
	Method         *model.MethodEntry
	DomainMethod   *domain.Method
	Index          int
	Error          error
	ProcessingTime time.Duration
}

// ConcurrentMethodProcessor handles concurrent method processing with error recovery.
type ConcurrentMethodProcessor struct {
	parser     *Parser
	maxWorkers int
	timeout    time.Duration
	logger     *zap.Logger
	metrics    *ProcessingMetrics
}

// ProcessingMetrics tracks method processing performance.
type ProcessingMetrics struct {
	TotalMethods        int
	SuccessfulMethods   int
	FailedMethods       int
	TotalProcessingTime time.Duration
	AverageMethodTime   time.Duration
	mutex               sync.RWMutex
}

// NewConcurrentMethodProcessor creates a new concurrent method processor.
func NewConcurrentMethodProcessor(parser *Parser, maxWorkers int, timeout time.Duration, logger *zap.Logger) *ConcurrentMethodProcessor {
	return &ConcurrentMethodProcessor{
		parser:     parser,
		maxWorkers: maxWorkers,
		timeout:    timeout,
		logger:     logger,
		metrics:    &ProcessingMetrics{},
	}
}

// ProcessMethodsConcurrent processes methods concurrently with error recovery.
func (cmp *ConcurrentMethodProcessor) ProcessMethodsConcurrent(ctx context.Context, intf *intfEntry) ([]*model.MethodEntry, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
	mset := types.NewMethodSet(iface)

	if mset.Len() == 0 {
		return []*model.MethodEntry{}, nil
	}

	// Initialize metrics
	cmp.metrics.mutex.Lock()
	cmp.metrics.TotalMethods = mset.Len()
	cmp.metrics.SuccessfulMethods = 0
	cmp.metrics.FailedMethods = 0
	startTime := time.Now()
	cmp.metrics.mutex.Unlock()

	// Create results channel and error group
	results := make([]*MethodProcessingResult, mset.Len())
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cmp.maxWorkers)

	// Process methods concurrently
	for i := 0; i < mset.Len(); i++ {
		// Capture loop variable
		methodObj := mset.At(i).Obj()

		g.Go(func() error {
			result := cmp.processMethodWithRecovery(gctx, methodObj, intf.opts, i)
			results[i] = result

			// Update metrics
			cmp.updateMetrics(result)

			// Log processing result
			if result.Error != nil {
				cmp.logger.Warn("Method processing failed",
					zap.String("method", methodObj.Name()),
					zap.Error(result.Error),
					zap.Duration("processing_time", result.ProcessingTime))
			} else {
				cmp.logger.Debug("Method processed successfully",
					zap.String("method", methodObj.Name()),
					zap.Duration("processing_time", result.ProcessingTime))
			}

			return nil // Don't fail the entire group for individual method failures
		})
	}

	// Wait for all methods to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("method processing failed: %w", err)
	}

	// Collect successful results and handle errors
	return cmp.collectResults(results, startTime)
}

// processMethodWithRecovery processes a single method with error recovery.
func (cmp *ConcurrentMethodProcessor) processMethodWithRecovery(ctx context.Context, methodObj types.Object, opts option.Options, index int) *MethodProcessingResult {
	startTime := time.Now()
	result := &MethodProcessingResult{
		Index: index,
	}

	// Create timeout context for individual method processing
	methodCtx, cancel := context.WithTimeout(ctx, cmp.timeout)
	defer cancel()

	// Process method with panic recovery
	defer func() {
		result.ProcessingTime = time.Since(startTime)
		if r := recover(); r != nil {
			result.Error = fmt.Errorf("%w: %v", ErrMethodProcessingPanic, r)
			cmp.logger.Error("Method processing panic recovered",
				zap.String("method", methodObj.Name()),
				zap.Any("panic", r),
				zap.Duration("processing_time", result.ProcessingTime))
		}
	}()

	// Process the method
	method, err := cmp.processMethodSafely(methodCtx, methodObj, opts)
	if err != nil {
		result.Error = fmt.Errorf("failed to process method %s: %w", methodObj.Name(), err)
		return result
	}

	result.Method = method

	return result
}

// processMethodSafely processes a method with enhanced error handling.
func (cmp *ConcurrentMethodProcessor) processMethodSafely(ctx context.Context, methodObj types.Object, opts option.Options) (*model.MethodEntry, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("method processing context cancelled: %w", ctx.Err())
	default:
	}

	// Validate method signature
	signature, ok := methodObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%w but got %T", ErrExpectedSignature, methodObj.Type())
	}

	// Enhanced validation with specific error messages
	if signature.Params().Len() == 0 {
		return nil, fmt.Errorf("%w: method %s", ErrMethodMustHaveParameters, methodObj.Name())
	}

	if signature.Results().Len() == 0 {
		return nil, fmt.Errorf("%w: method %s", ErrMethodMustHaveReturnValues, methodObj.Name())
	}

	// Process method using the original parser logic with error recovery
	method, err := cmp.parser.parseMethod(methodObj, opts)
	if err != nil {
		// Try to provide more context for common errors
		if isTypeResolutionError(err) {
			return nil, fmt.Errorf("type resolution failed for method %s: %w (suggestion: check import statements and type definitions)", methodObj.Name(), err)
		}

		if isAnnotationError(err) {
			return nil, fmt.Errorf("annotation processing failed for method %s: %w (suggestion: check annotation syntax)", methodObj.Name(), err)
		}

		return nil, err
	}

	return method, nil
}

// collectResults collects successful results and handles partial failures.
func (cmp *ConcurrentMethodProcessor) collectResults(results []*MethodProcessingResult, startTime time.Time) ([]*model.MethodEntry, error) {
	var successfulMethods []*model.MethodEntry

	var errors []error

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else if result.Method != nil {
			successfulMethods = append(successfulMethods, result.Method)
		}
	}

	// Update final metrics
	cmp.metrics.mutex.Lock()

	cmp.metrics.TotalProcessingTime = time.Since(startTime)
	if cmp.metrics.SuccessfulMethods > 0 {
		cmp.metrics.AverageMethodTime = cmp.metrics.TotalProcessingTime / time.Duration(cmp.metrics.SuccessfulMethods)
	}
	cmp.metrics.mutex.Unlock()

	// Log processing summary
	cmp.logger.Info("Method processing completed",
		zap.Int("total_methods", len(results)),
		zap.Int("successful_methods", len(successfulMethods)),
		zap.Int("failed_methods", len(errors)),
		zap.Duration("total_time", cmp.metrics.TotalProcessingTime),
		zap.Duration("average_method_time", cmp.metrics.AverageMethodTime))

	// Decide how to handle partial failures based on success rate
	successRate := float64(len(successfulMethods)) / float64(len(results))

	if successRate == 0 {
		// Complete failure
		return nil, fmt.Errorf("%w: %d methods", ErrAllMethodsFailed, len(results))
	}

	if successRate < 0.5 {
		// More than half failed - this might indicate a systemic issue
		cmp.logger.Warn("Low method processing success rate",
			zap.Float64("success_rate", successRate),
			zap.Int("failed_count", len(errors)))

		// Return partial results with warning
		return successfulMethods, fmt.Errorf("%w: %d of %d methods failed", ErrPartialProcessingFailure, len(errors), len(results))
	}

	if len(errors) > 0 {
		// Some failures but majority succeeded
		cmp.logger.Warn("Some methods failed to process",
			zap.Int("failed_count", len(errors)),
			zap.Float64("success_rate", successRate))
	}

	return successfulMethods, nil
}

// updateMetrics updates processing metrics in a thread-safe manner.
func (cmp *ConcurrentMethodProcessor) updateMetrics(result *MethodProcessingResult) {
	cmp.metrics.mutex.Lock()
	defer cmp.metrics.mutex.Unlock()

	if result.Error != nil {
		cmp.metrics.FailedMethods++
	} else {
		cmp.metrics.SuccessfulMethods++
	}
}

// GetMetrics returns current processing metrics.
func (cmp *ConcurrentMethodProcessor) GetMetrics() ProcessingMetrics {
	cmp.metrics.mutex.RLock()
	defer cmp.metrics.mutex.RUnlock()

	// Return a copy without the mutex to avoid copying locks
	return ProcessingMetrics{
		TotalMethods:        cmp.metrics.TotalMethods,
		SuccessfulMethods:   cmp.metrics.SuccessfulMethods,
		FailedMethods:       cmp.metrics.FailedMethods,
		TotalProcessingTime: cmp.metrics.TotalProcessingTime,
		AverageMethodTime:   cmp.metrics.AverageMethodTime,
	}
}

// Helper functions for error classification.
func isTypeResolutionError(err error) bool {
	// Check if error is related to type resolution
	return false // Placeholder - implement based on actual error types
}

func isAnnotationError(err error) bool {
	// Check if error is related to annotation processing
	return false // Placeholder - implement based on actual error types
}
