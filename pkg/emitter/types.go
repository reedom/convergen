package emitter

import (
	"strings"
	"time"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// OptimizationLevel defines the level of code optimization
type OptimizationLevel int

const (
	OptimizationNone OptimizationLevel = iota
	OptimizationBasic
	OptimizationAggressive
	OptimizationMaximal
)

func (ol OptimizationLevel) String() string {
	switch ol {
	case OptimizationNone:
		return "none"
	case OptimizationBasic:
		return "basic"
	case OptimizationAggressive:
		return "aggressive"
	case OptimizationMaximal:
		return "maximal"
	default:
		return "unknown"
	}
}

// ConstructionStrategy defines how code should be constructed
type ConstructionStrategy int

const (
	StrategyCompositeLiteral ConstructionStrategy = iota
	StrategyAssignmentBlock
	StrategyMixedApproach
	StrategyCustomTemplate
)

func (cs ConstructionStrategy) String() string {
	switch cs {
	case StrategyCompositeLiteral:
		return "composite_literal"
	case StrategyAssignmentBlock:
		return "assignment_block"
	case StrategyMixedApproach:
		return "mixed_approach"
	case StrategyCustomTemplate:
		return "custom_template"
	default:
		return "unknown"
	}
}

// GeneratedCode represents the complete generated Go code
type GeneratedCode struct {
	PackageName    string                 `json:"package_name"`
	Imports        *ImportDeclaration     `json:"imports"`
	Methods        []*MethodCode          `json:"methods"`
	BaseCode       string                 `json:"base_code"`
	Source         string                 `json:"source"`         // Final formatted source
	Metadata       *GenerationMetadata    `json:"metadata"`
	Metrics        *GenerationMetrics     `json:"metrics"`
}

// MethodCode represents generated code for a single method
type MethodCode struct {
	Name           string                 `json:"name"`
	Signature      string                 `json:"signature"`
	Body           string                 `json:"body"`
	ErrorHandling  string                 `json:"error_handling"`
	Documentation  string                 `json:"documentation"`
	Imports        []*Import              `json:"imports"`
	Complexity     *ComplexityMetrics     `json:"complexity"`
	Strategy       ConstructionStrategy   `json:"strategy"`
	Fields         []*FieldCode           `json:"fields"`
}

// FieldCode represents generated code for a single field
type FieldCode struct {
	Name           string                 `json:"name"`
	Assignment     string                 `json:"assignment"`
	Declaration    string                 `json:"declaration"`
	ErrorCheck     string                 `json:"error_check"`
	Imports        []*Import              `json:"imports"`
	Dependencies   []string               `json:"dependencies"`
	Order          int                    `json:"order"`
	Strategy       string                 `json:"strategy"`
}

// ErrorCode represents generated error handling code
type ErrorCode struct {
	CheckCode      string                 `json:"check_code"`
	HandlingCode   string                 `json:"handling_code"`
	ReturnCode     string                 `json:"return_code"`
	WrapperCode    string                 `json:"wrapper_code"`
	Imports        []*Import              `json:"imports"`
}

// ImportDeclaration represents the complete import section
type ImportDeclaration struct {
	Imports        []*Import              `json:"imports"`
	StandardLibs   []*Import              `json:"standard_libs"`
	ThirdPartyLibs []*Import              `json:"third_party_libs"`
	LocalImports   []*Import              `json:"local_imports"`
	Source         string                 `json:"source"`         // Formatted import block
}

// Import represents a single import statement
type Import struct {
	Path      string                    `json:"path"`
	Alias     string                    `json:"alias"`
	Used      bool                      `json:"used"`
	Standard  bool                      `json:"standard"`
	Local     bool                      `json:"local"`
	Required  bool                      `json:"required"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	GenerationTime     time.Time            `json:"generation_time"`
	CompletionTime     time.Time            `json:"completion_time"`
	GenerationDuration time.Duration        `json:"generation_duration"`
	EmitterVersion     string               `json:"emitter_version"`
	ConfigHash         string               `json:"config_hash"`
	SourceFileHash     string               `json:"source_file_hash"`
	Environment        map[string]string    `json:"environment"`
}

// GenerationMetrics contains detailed metrics about the generation process
type GenerationMetrics struct {
	MethodsGenerated       int                  `json:"methods_generated"`
	FieldsGenerated        int                  `json:"fields_generated"`
	LinesGenerated         int                  `json:"lines_generated"`
	ImportsGenerated       int                  `json:"imports_generated"`
	TotalGenerationTime    time.Duration        `json:"total_generation_time"`
	AverageMethodTime      time.Duration        `json:"average_method_time"`
	OptimizationTime       time.Duration        `json:"optimization_time"`
	FormattingTime         time.Duration        `json:"formatting_time"`
	ValidationTime         time.Duration        `json:"validation_time"`
	MemoryUsage            int64                `json:"memory_usage"`
	ConcurrencyLevel       int                  `json:"concurrency_level"`
	OptimizationsApplied   int                  `json:"optimizations_applied"`
	ErrorsEncountered      int                  `json:"errors_encountered"`
	WarningsGenerated      int                  `json:"warnings_generated"`
}

// ComplexityMetrics analyzes the complexity of generated code
type ComplexityMetrics struct {
	FieldCount          int                  `json:"field_count"`
	ErrorFields         int                  `json:"error_fields"`
	ConverterFields     int                  `json:"converter_fields"`
	NestedFields        int                  `json:"nested_fields"`
	ComplexityScore     float64              `json:"complexity_score"`
	LinesGenerated      int                  `json:"lines_generated"`
	CyclomaticComplexity int                 `json:"cyclomatic_complexity"`
	RecommendedStrategy ConstructionStrategy `json:"recommended_strategy"`
	GenerationTime      time.Duration        `json:"generation_time"`
}

// ImportAnalysis contains the results of import analysis
type ImportAnalysis struct {
	RequiredImports    []*Import            `json:"required_imports"`
	ConflictingNames   map[string][]*Import `json:"conflicting_names"`
	UnusedImports      []*Import            `json:"unused_imports"`
	StandardLibs       []*Import            `json:"standard_libs"`
	ThirdPartyLibs     []*Import            `json:"third_party_libs"`
	LocalImports       []*Import            `json:"local_imports"`
	OptimizationsSuggested []string        `json:"optimizations_suggested"`
}

// EmitterMetrics tracks overall emitter performance
type EmitterMetrics struct {
	TotalGenerations     int64                `json:"total_generations"`
	TotalMethods         int64                `json:"total_methods"`
	TotalFields          int64                `json:"total_fields"`
	TotalLines           int64                `json:"total_lines"`
	TotalGenerationTime  time.Duration        `json:"total_generation_time"`
	AverageGenerationTime time.Duration       `json:"average_generation_time"`
	ThroughputPerSecond  float64              `json:"throughput_per_second"`
	
	// Strategy usage statistics
	StrategyUsage        map[string]int64     `json:"strategy_usage"`
	StrategyPerformance  map[string]time.Duration `json:"strategy_performance"`
	
	// Optimization statistics
	OptimizationsApplied map[string]int64     `json:"optimizations_applied"`
	OptimizationTime     time.Duration        `json:"optimization_time"`
	
	// Error statistics
	ErrorsEncountered    int64                `json:"errors_encountered"`
	ErrorsByType         map[string]int64     `json:"errors_by_type"`
	
	// Memory statistics
	PeakMemoryUsage      int64                `json:"peak_memory_usage"`
	AverageMemoryUsage   int64                `json:"average_memory_usage"`
	
	// Performance history
	PerformanceHistory   []PerformanceSnapshot `json:"performance_history"`
	
	// Timing
	StartTime            time.Time            `json:"start_time"`
	LastGeneration       time.Time            `json:"last_generation"`
}

// PerformanceSnapshot captures performance at a specific point in time
type PerformanceSnapshot struct {
	Timestamp           time.Time            `json:"timestamp"`
	GenerationsPerSecond float64             `json:"generations_per_second"`
	MethodsPerSecond    float64              `json:"methods_per_second"`
	LinesPerSecond      float64              `json:"lines_per_second"`
	MemoryUsage         int64                `json:"memory_usage"`
	ConcurrencyLevel    int                  `json:"concurrency_level"`
	ErrorRate           float64              `json:"error_rate"`
}

// GenerationRequest represents a request for code generation
type GenerationRequest struct {
	ExecutionResults    *domain.ExecutionResults `json:"execution_results"`
	Config              *EmitterConfig           `json:"config"`
	Context             map[string]interface{}   `json:"context"`
	RequestID           string                   `json:"request_id"`
	Timestamp           time.Time                `json:"timestamp"`
}

// GenerationResponse represents the response from code generation
type GenerationResponse struct {
	GeneratedCode       *GeneratedCode           `json:"generated_code"`
	Success             bool                     `json:"success"`
	Errors              []GenerationError        `json:"errors"`
	Warnings            []GenerationWarning      `json:"warnings"`
	RequestID           string                   `json:"request_id"`
	ProcessingTime      time.Duration            `json:"processing_time"`
}

// GenerationError represents an error during code generation
type GenerationError struct {
	Phase       string                       `json:"phase"`
	Method      string                       `json:"method"`
	Field       string                       `json:"field"`
	Message     string                       `json:"message"`
	Code        string                       `json:"code"`
	Severity    string                       `json:"severity"`
	Context     map[string]interface{}       `json:"context"`
	Timestamp   time.Time                    `json:"timestamp"`
	Recoverable bool                         `json:"recoverable"`
}

// GenerationWarning represents a warning during code generation
type GenerationWarning struct {
	Phase       string                       `json:"phase"`
	Method      string                       `json:"method"`
	Field       string                       `json:"field"`
	Message     string                       `json:"message"`
	Code        string                       `json:"code"`
	Context     map[string]interface{}       `json:"context"`
	Timestamp   time.Time                    `json:"timestamp"`
	Suggestion  string                       `json:"suggestion"`
}

// TemplateData represents data passed to code templates
type TemplateData struct {
	Method          *domain.MethodResult     `json:"method"`
	Fields          []*domain.FieldResult    `json:"fields"`
	Package         string                   `json:"package"`
	Imports         []*Import                `json:"imports"`
	Config          *EmitterConfig           `json:"config"`
	Metadata        map[string]interface{}   `json:"metadata"`
	HelperFunctions map[string]interface{}   `json:"helper_functions"`
}

// OrderedBuffer maintains insertion order for deterministic output
type OrderedBuffer struct {
	items    []*BufferItem
	itemMap  map[int]*BufferItem
	maxOrder int
	sorted   bool
}

// BufferItem represents an item in the ordered buffer
type BufferItem struct {
	Order   int
	Content string
	Type    string
	Context map[string]interface{}
}

// ValidationResult represents the result of code validation
type ValidationResult struct {
	Valid       bool                         `json:"valid"`
	Errors      []ValidationError            `json:"errors"`
	Warnings    []ValidationWarning          `json:"warnings"`
	Suggestions []ValidationSuggestion       `json:"suggestions"`
	Metrics     *ValidationMetrics           `json:"metrics"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Line        int                          `json:"line"`
	Column      int                          `json:"column"`
	Message     string                       `json:"message"`
	Code        string                       `json:"code"`
	Severity    string                       `json:"severity"`
	Context     string                       `json:"context"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Line        int                          `json:"line"`
	Column      int                          `json:"column"`
	Message     string                       `json:"message"`
	Code        string                       `json:"code"`
	Context     string                       `json:"context"`
	Suggestion  string                       `json:"suggestion"`
}

// ValidationSuggestion represents a validation suggestion
type ValidationSuggestion struct {
	Line        int                          `json:"line"`
	Column      int                          `json:"column"`
	Message     string                       `json:"message"`
	Replacement string                       `json:"replacement"`
	Benefit     string                       `json:"benefit"`
}

// ValidationMetrics contains validation performance metrics
type ValidationMetrics struct {
	ValidationTime   time.Duration           `json:"validation_time"`
	LinesValidated   int                     `json:"lines_validated"`
	ErrorsFound      int                     `json:"errors_found"`
	WarningsFound    int                     `json:"warnings_found"`
	SuggestionsFound int                     `json:"suggestions_found"`
}

// TemplateSystem manages code generation templates
type TemplateSystem interface {
	Execute(template string, data interface{}) (string, error)
	HasTemplate(name string) bool
	RegisterTemplate(name, content string) error
}

// CodeValidator validates generated code
type CodeValidator interface {
	Validate(code string) error
	ValidateMethod(method *MethodCode) error
	ValidateMethodCode(methodCode *MethodCode) error
}

// SimpleTemplateSystem provides basic template functionality
type SimpleTemplateSystem struct{}

func NewTemplateSystem() TemplateSystem {
	return &SimpleTemplateSystem{}
}

func (s *SimpleTemplateSystem) Execute(template string, data interface{}) (string, error) {
	// Simple template execution - in practice this would use text/template
	return template, nil
}

func (s *SimpleTemplateSystem) HasTemplate(name string) bool {
	return false
}

func (s *SimpleTemplateSystem) RegisterTemplate(name, content string) error {
	// Simple registration - in practice this would store templates
	return nil
}

func NewCustomTemplate(name, content string) interface{} {
	return content
}

// NewEmitterMetrics creates default emitter metrics
func NewEmitterMetrics() *EmitterMetrics {
	return &EmitterMetrics{
		StartTime:        time.Now(),
		TotalMethods:     0,
		MethodsGenerated: 0,
		MethodsFailed:    0,
		LinesGenerated:   0,
		ImportsGenerated: 0,
		BytesGenerated:   0,
	}
}

// RecordGeneration records generation metrics
func (m *EmitterMetrics) RecordGeneration(methodCode *MethodCode, packageName string, methods []*MethodCode) {
	m.MethodsGenerated++
	if methodCode != nil {
		m.LinesGenerated += len(strings.Split(methodCode.Assignment, "\n"))
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *EmitterMetrics) GetSnapshot() *EmitterMetrics {
	return &EmitterMetrics{
		StartTime:        m.StartTime,
		EndTime:          m.EndTime,
		TotalMethods:     m.TotalMethods,
		MethodsGenerated: m.MethodsGenerated,
		MethodsFailed:    m.MethodsFailed,
		LinesGenerated:   m.LinesGenerated,
		ImportsGenerated: m.ImportsGenerated,
		BytesGenerated:   m.BytesGenerated,
	}
}

// Event type constants
const (
	EventEmitStarted      = "emit.started"
	EventEmitCompleted    = "emit.completed"
	EventEmitFailed       = "emit.failed"
	EventValidationFailed = "emit.validation.failed"
)

// Default values and constants
const (
	DefaultIndentStyle     = "\t"
	DefaultLineWidth       = 120
	DefaultMaxConcurrency  = 8
	DefaultGenerationTimeout = 30 * time.Second
	
	MaxFieldsCompositeLiteral = 10
	MaxComplexityScore        = 100.0
	MaxCyclomaticComplexity   = 20
	
	// Template names
	TemplateCompositeLiteral  = "composite_literal"
	TemplateAssignmentBlock   = "assignment_block"
	TemplateErrorHandling     = "error_handling"
	TemplateMethodSignature   = "method_signature"
	TemplateImportDeclaration = "import_declaration"
)

// Helper functions for type conversion and validation

// NewGenerationMetrics creates a new GenerationMetrics instance
func NewGenerationMetrics() *GenerationMetrics {
	return &GenerationMetrics{
		StrategyUsage:        make(map[string]int64),
		StrategyPerformance:  make(map[string]time.Duration),
		OptimizationsApplied: make(map[string]int64),
		ErrorsByType:         make(map[string]int64),
		PerformanceHistory:   make([]PerformanceSnapshot, 0, 1000),
	}
}

// NewComplexityMetrics creates a new ComplexityMetrics instance
func NewComplexityMetrics() *ComplexityMetrics {
	return &ComplexityMetrics{
		RecommendedStrategy: StrategyCompositeLiteral,
	}
}

// IsComplex returns true if the complexity metrics indicate complex code
func (cm *ComplexityMetrics) IsComplex() bool {
	return cm.ComplexityScore > 50.0 || 
		   cm.ErrorFields > 0 || 
		   cm.NestedFields > 2 ||
		   cm.CyclomaticComplexity > 10
}

// ShouldUseComposite returns true if composite literal strategy is recommended
func (cm *ComplexityMetrics) ShouldUseComposite() bool {
	return cm.RecommendedStrategy == StrategyCompositeLiteral
}

// Add method adds a new item to the ordered buffer
func (ob *OrderedBuffer) Add(order int, content, itemType string) {
	item := &BufferItem{
		Order:   order,
		Content: content,
		Type:    itemType,
		Context: make(map[string]interface{}),
	}
	
	if ob.itemMap == nil {
		ob.itemMap = make(map[int]*BufferItem)
	}
	
	ob.items = append(ob.items, item)
	ob.itemMap[order] = item
	
	if order > ob.maxOrder {
		ob.maxOrder = order
	}
	
	ob.sorted = false
}

// Generate returns the ordered content as a string
func (ob *OrderedBuffer) Generate() string {
	if !ob.sorted {
		ob.sort()
	}
	
	var result string
	for _, item := range ob.items {
		result += item.Content
	}
	
	return result
}

// sort sorts the buffer items by order
func (ob *OrderedBuffer) sort() {
	// Simple insertion sort for small collections
	for i := 1; i < len(ob.items); i++ {
		key := ob.items[i]
		j := i - 1
		
		for j >= 0 && ob.items[j].Order > key.Order {
			ob.items[j+1] = ob.items[j]
			j--
		}
		
		ob.items[j+1] = key
	}
	
	ob.sorted = true
}