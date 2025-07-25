package domain

import (
	"fmt"
	"time"
)

// Method represents a conversion method to be generated
type Method struct {
	Name        string         `json:"name"`
	SourceType  Type           `json:"source_type"`
	DestType    Type           `json:"dest_type"`
	Config      *MethodConfig  `json:"config"`
	Mappings    []*FieldMapping `json:"mappings"`
	Signature   *MethodSignature `json:"signature"`
}

// NewMethod creates a validated method
func NewMethod(name string, sourceType, destType Type) (*Method, error) {
	if name == "" {
		return nil, fmt.Errorf("method name cannot be empty")
	}
	if sourceType == nil {
		return nil, fmt.Errorf("source type cannot be nil")
	}
	if destType == nil {
		return nil, fmt.Errorf("destination type cannot be nil")
	}
	
	return &Method{
		Name:       name,
		SourceType: sourceType,
		DestType:   destType,
		Config:     NewMethodConfig(),
		Mappings:   make([]*FieldMapping, 0),
		Signature:  nil, // Will be set by signature analysis
	}, nil
}

// AddMapping adds a field mapping to the method
func (m *Method) AddMapping(mapping *FieldMapping) error {
	if mapping == nil {
		return fmt.Errorf("mapping cannot be nil")
	}
	
	// Check for duplicate mapping IDs
	for _, existing := range m.Mappings {
		if existing.ID == mapping.ID {
			return fmt.Errorf("mapping with ID %s already exists", mapping.ID)
		}
	}
	
	m.Mappings = append(m.Mappings, mapping)
	return nil
}

// GetMappingByID retrieves a mapping by its ID
func (m *Method) GetMappingByID(id string) (*FieldMapping, bool) {
	for _, mapping := range m.Mappings {
		if mapping.ID == id {
			return mapping, true
		}
	}
	return nil, false
}

// MethodConfig holds method-level configuration
type MethodConfig struct {
	Style         StyleConfig        `json:"style"`          // return vs arg style
	Receiver      *ReceiverConfig    `json:"receiver"`       // optional receiver
	Reverse       bool               `json:"reverse"`        // reverse copy direction
	CaseSensitive bool               `json:"case_sensitive"` // case-sensitive matching
	UseGetters    bool               `json:"use_getters"`    // include getters
	UseStringers  bool               `json:"use_stringers"`  // use String() methods
	TypeCasting   bool               `json:"type_casting"`   // allow type casting
	PreProcess    []*ManipulatorFunc `json:"pre_process"`    // pre-processing functions
	PostProcess   []*ManipulatorFunc `json:"post_process"`   // post-processing functions
}

// NewMethodConfig creates a default method configuration
func NewMethodConfig() *MethodConfig {
	return &MethodConfig{
		Style:         StyleReturn,
		Receiver:      nil,
		Reverse:       false,
		CaseSensitive: true,
		UseGetters:    false,
		UseStringers:  false,
		TypeCasting:   false,
		PreProcess:    make([]*ManipulatorFunc, 0),
		PostProcess:   make([]*ManipulatorFunc, 0),
	}
}

// StyleConfig defines the generated function signature style
type StyleConfig int

const (
	StyleReturn StyleConfig = iota // func Convert(src *Source) (dst *Dest)
	StyleArg                       // func Convert(dst *Dest, src *Source)
)

func (s StyleConfig) String() string {
	switch s {
	case StyleReturn:
		return "return"
	case StyleArg:
		return "arg"
	default:
		return "unknown"
	}
}

// ReceiverConfig specifies receiver method generation
type ReceiverConfig struct {
	Variable string `json:"variable"`
	Type     Type   `json:"type"`
}

// ManipulatorFunc represents pre/post processing functions
type ManipulatorFunc struct {
	Name       string   `json:"name"`
	Package    string   `json:"package"`
	ImportPath string   `json:"import_path"`
	Args       []string `json:"args"`
	ReturnsErr bool     `json:"returns_err"`
}

// MethodSignature describes the generated method signature
type MethodSignature struct {
	Name     string       `json:"name"`
	Receiver *Receiver    `json:"receiver"`
	Params   []*Parameter `json:"params"`
	Results  []*Parameter `json:"results"`
	HasError bool         `json:"has_error"`
}

// Receiver represents a method receiver
type Receiver struct {
	Name    string `json:"name"`
	Type    Type   `json:"type"`
	Pointer bool   `json:"pointer"`
}

// Parameter represents a function parameter or result
type Parameter struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// ExecutionPlan defines how to execute field conversions concurrently
type ExecutionPlan struct {
	Methods      map[string]*MethodPlan `json:"methods"`
	GlobalLimits *ResourceLimits        `json:"global_limits"`
	Strategy     ExecutionStrategy      `json:"strategy"`
	Metrics      *PlanMetrics           `json:"metrics"`
}

// MethodPlan represents the execution plan for a single method
type MethodPlan struct {
	Method    *Method             `json:"method"`
	Mappings  []*FieldMapping     `json:"mappings"`
	Batches   []*ConcurrentBatch  `json:"batches"`
	Resources *ResourceAllocation `json:"resources"`
}

// ConcurrentBatch groups fields that can be processed in parallel
type ConcurrentBatch struct {
	ID          string           `json:"id"`
	Fields      []*FieldMapping  `json:"fields"`
	DependsOn   []string         `json:"depends_on"` // Batch IDs this batch depends on
	Complexity  int              `json:"complexity"`
	EstimatedMS int64            `json:"estimated_ms"`
}

// NewConcurrentBatch creates a validated concurrent batch
func NewConcurrentBatch(id string, fields []*FieldMapping) (*ConcurrentBatch, error) {
	if id == "" {
		return nil, fmt.Errorf("batch ID cannot be empty")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("batch must contain at least one field")
	}
	
	return &ConcurrentBatch{
		ID:          id,
		Fields:      append([]*FieldMapping(nil), fields...), // defensive copy
		DependsOn:   make([]string, 0),
		Complexity:  calculateBatchComplexity(fields),
		EstimatedMS: estimateBatchTime(fields),
	}, nil
}

// AddDependency adds a batch dependency
func (cb *ConcurrentBatch) AddDependency(batchID string) error {
	if batchID == "" {
		return fmt.Errorf("dependency batch ID cannot be empty")
	}
	if batchID == cb.ID {
		return fmt.Errorf("batch cannot depend on itself")
	}
	
	// Check if dependency already exists
	for _, dep := range cb.DependsOn {
		if dep == batchID {
			return nil // Already exists
		}
	}
	
	cb.DependsOn = append(cb.DependsOn, batchID)
	return nil
}

// ResourceLimits defines execution constraints
type ResourceLimits struct {
	MaxGoroutines       int           `json:"max_goroutines"`
	MaxMemoryMB         int           `json:"max_memory_mb"`
	TimeoutMS           int           `json:"timeout_ms"`
	MaxConcurrentFields int           `json:"max_concurrent_fields"`
}

// NewResourceLimits creates default resource limits
func NewResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxGoroutines:       10,
		MaxMemoryMB:         100,
		TimeoutMS:           30000, // 30 seconds
		MaxConcurrentFields: 50,
	}
}

// ResourceAllocation represents allocated resources for execution
type ResourceAllocation struct {
	MaxConcurrentBatches int                 `json:"max_concurrent_batches"`
	MemoryLimitMB        int                 `json:"memory_limit_mb"`
	TimeoutMS            int                 `json:"timeout_ms"`
	GoroutinePool        *GoroutinePoolConfig `json:"goroutine_pool"`
}

// GoroutinePoolConfig configures the worker goroutine pool  
type GoroutinePoolConfig struct {
	MinWorkers  int `json:"min_workers"`
	MaxWorkers  int `json:"max_workers"`
	QueueSize   int `json:"queue_size"`
	IdleTimeout int `json:"idle_timeout_ms"`
}

// ExecutionStrategy determines how to balance performance vs resources
type ExecutionStrategy int

const (
	StrategySequential ExecutionStrategy = iota
	StrategyBatched
	StrategyFullyConcurrent
	StrategyAdaptive
)

func (s ExecutionStrategy) String() string {
	switch s {
	case StrategySequential:
		return "sequential"
	case StrategyBatched:
		return "batched"
	case StrategyFullyConcurrent:
		return "fully_concurrent"
	case StrategyAdaptive:
		return "adaptive"
	default:
		return "unknown"
	}
}

// PlanMetrics track planning performance and characteristics
type PlanMetrics struct {
	PlanningDurationMS    int64   `json:"planning_duration_ms"`
	MethodsPlanned        int     `json:"methods_planned"`
	TotalFields           int     `json:"total_fields"`
	ConcurrentBatches     int     `json:"concurrent_batches"`
	ParallelizationRatio  float64 `json:"parallelization_ratio"`
	EstimatedSpeedupRatio float64 `json:"estimated_speedup_ratio"`
}

// GenerationResult represents the outcome of processing
type GenerationResult struct {
	Methods     []*Method             `json:"methods"`
	BaseCode    string                `json:"base_code"`
	Generated   *GeneratedFunction    `json:"generated"`
	Errors      []GenerationError     `json:"errors"`
	Metrics     *ProcessingMetrics    `json:"metrics"`
	Diagnostics []Diagnostic          `json:"diagnostics"`
}

// GeneratedFunction represents a complete generated function
type GeneratedFunction struct {
	Name        string               `json:"name"`
	Signature   string               `json:"signature"`
	Body        string               `json:"body"`
	Imports     []Import             `json:"imports"`
	Comments    []Comment            `json:"comments"`
	Metadata    *FunctionMetadata    `json:"metadata"`
}

// Comment represents a code comment
type Comment struct {
	Text     string `json:"text"`
	Position int    `json:"position"`
	Type     string `json:"type"` // "line", "block", "doc"
}

// FunctionMetadata contains metadata about the generated function
type FunctionMetadata struct {
	GeneratedAt     time.Time `json:"generated_at"`
	Version         string    `json:"version"`
	SourceMethod    string    `json:"source_method"`
	Optimizations   []string  `json:"optimizations"`
	PerformanceHint string    `json:"performance_hint"`
}

// FieldResult represents the result of processing a single field
type FieldResult struct {
	FieldID     string           `json:"field_id"`
	Code        *GeneratedCode   `json:"code"`
	Error       *GenerationError `json:"error"`
	ProcessedAt time.Time        `json:"processed_at"`
	DurationMS  int64            `json:"duration_ms"`
}

// ProcessingMetrics track performance and resource usage
type ProcessingMetrics struct {
	TotalDurationMS   int64   `json:"total_duration_ms"`
	ConcurrentFields  int     `json:"concurrent_fields"`
	MaxGoroutines     int     `json:"max_goroutines"`
	MemoryUsageMB     int     `json:"memory_usage_mb"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
}

// Diagnostic represents processing diagnostics and warnings
type Diagnostic struct {
	Level   DiagnosticLevel `json:"level"`
	Message string          `json:"message"`
	Method  string          `json:"method"`
	Field   string          `json:"field"`
	Code    string          `json:"code"`
}

// DiagnosticLevel represents the severity of a diagnostic
type DiagnosticLevel int

const (
	DiagnosticInfo DiagnosticLevel = iota
	DiagnosticWarning
	DiagnosticError
)

func (d DiagnosticLevel) String() string {
	switch d {
	case DiagnosticInfo:
		return "info"
	case DiagnosticWarning:
		return "warning"
	case DiagnosticError:
		return "error"
	default:
		return "unknown"
	}
}

// Helper functions for complexity and time estimation

// calculateBatchComplexity estimates the complexity of processing a batch
func calculateBatchComplexity(fields []*FieldMapping) int {
	complexity := 0
	for _, field := range fields {
		// Base complexity for each field
		complexity += 1
		
		// Additional complexity based on strategy
		switch field.StrategyName {
		case "direct":
			complexity += 0
		case "typecast":
			complexity += 1
		case "method":
			complexity += 2
		case "converter":
			complexity += 3
		case "literal":
			complexity += 0
		default:
			complexity += 2
		}
		
		// Additional complexity for dependencies
		complexity += len(field.Dependencies)
	}
	
	return complexity
}

// estimateBatchTime estimates processing time for a batch
func estimateBatchTime(fields []*FieldMapping) int64 {
	baseTimeMS := int64(len(fields)) // 1ms per field base time
	
	for _, field := range fields {
		// Add time based on strategy complexity
		switch field.StrategyName {
		case "direct":
			baseTimeMS += 0
		case "typecast":
			baseTimeMS += 1
		case "method":
			baseTimeMS += 2
		case "converter":
			baseTimeMS += 5
		case "literal":
			baseTimeMS += 0
		default:
			baseTimeMS += 3
		}
	}
	
	return baseTimeMS
}