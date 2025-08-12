package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GenericCompatibilityMatrix provides a structured approach to analyzing
// compatibility between generic types based on their constraints and structure.
type GenericCompatibilityMatrix struct {
	logger               *zap.Logger
	compatibilityChecker *TypeCompatibilityChecker

	// Cache for compatibility results
	cache        map[string]*MatrixCompatibilityResult
	cacheEnabled bool

	// Configuration
	strictConstraintMode bool
	allowPartialMatches  bool
}

// MatrixCompatibilityResult extends CompatibilityResult with matrix-specific information.
type MatrixCompatibilityResult struct {
	*CompatibilityResult

	// Matrix-specific fields
	MatrixAnalysis          *MatrixAnalysis   `json:"matrix_analysis,omitempty"`
	ConstraintCompatibility *ConstraintMatrix `json:"constraint_compatibility,omitempty"`
	TypeParameterMappings   map[string]string `json:"type_parameter_mappings,omitempty"`
}

// MatrixAnalysis provides detailed analysis of type relationships.
type MatrixAnalysis struct {
	SourceStructure       TypeStructure `json:"source_structure"`
	TargetStructure       TypeStructure `json:"target_structure"`
	StructuralMatch       bool          `json:"structural_match"`
	PartialMatchScore     float64       `json:"partial_match_score"`
	RecommendedConversion string        `json:"recommended_conversion,omitempty"`
}

// TypeStructure describes the structural characteristics of a type.
type TypeStructure struct {
	IsGeneric          bool     `json:"is_generic"`
	TypeParameterCount int      `json:"type_parameter_count"`
	ConstraintTypes    []string `json:"constraint_types"`
	UnderlyingType     string   `json:"underlying_type,omitempty"`
	InterfaceMethods   []string `json:"interface_methods,omitempty"`
	GenericComplexity  int      `json:"generic_complexity"`
}

// ConstraintMatrix maps constraint compatibility relationships.
type ConstraintMatrix struct {
	SourceConstraints []ConstraintInfo  `json:"source_constraints"`
	TargetConstraints []ConstraintInfo  `json:"target_constraints"`
	CompatibilityMap  map[string]string `json:"compatibility_map"`
	IncompatiblePairs []ConstraintPair  `json:"incompatible_pairs,omitempty"`
	PartialMatches    []PartialMatch    `json:"partial_matches,omitempty"`
}

// ConstraintInfo provides detailed information about a constraint.
type ConstraintInfo struct {
	ParameterName       string               `json:"parameter_name"`
	ConstraintType      string               `json:"constraint_type"`
	ConstraintSpecifics string               `json:"constraint_specifics"`
	Strictness          ConstraintStrictness `json:"strictness"`
	AllowedTypes        []string             `json:"allowed_types,omitempty"`
}

// ConstraintStrictness represents how strict a constraint is.
type ConstraintStrictness int

const (
	// ConstraintAny allows any type.
	ConstraintAny ConstraintStrictness = iota
	// ConstraintComparable requires comparable types.
	ConstraintComparable
	// ConstraintInterface requires interface implementation.
	ConstraintInterface
	// ConstraintUnion requires one of specific types.
	ConstraintUnion
	// ConstraintUnderlying requires underlying type match.
	ConstraintUnderlying
	// ConstraintExact requires exact type match.
	ConstraintExact
)

// String constants for constraint types to avoid goconst violations.
const (
	anyConstraint             = "any"
	comparableConstraint      = "comparable"
	unionConstraint           = "union"
	underlyingConstraint      = "underlying"
	unionUnderlyingConstraint = "union_underlying"
)

func (cs ConstraintStrictness) String() string {
	switch cs {
	case ConstraintAny:
		return anyConstraint
	case ConstraintComparable:
		return comparableConstraint
	case ConstraintInterface:
		return interfaceKeyword
	case ConstraintUnion:
		return unionConstraint
	case ConstraintUnderlying:
		return underlyingConstraint
	case ConstraintExact:
		return "exact"
	default:
		return UnknownValue
	}
}

// ConstraintPair represents a pair of incompatible constraints.
type ConstraintPair struct {
	SourceConstraint      string `json:"source_constraint"`
	TargetConstraint      string `json:"target_constraint"`
	IncompatibilityReason string `json:"incompatibility_reason"`
}

// PartialMatch represents a partial compatibility match.
type PartialMatch struct {
	SourceParameter    string  `json:"source_parameter"`
	TargetParameter    string  `json:"target_parameter"`
	MatchScore         float64 `json:"match_score"`
	RequiredConversion string  `json:"required_conversion,omitempty"`
}

// NewGenericCompatibilityMatrix creates a new generic compatibility matrix.
func NewGenericCompatibilityMatrix(logger *zap.Logger) *GenericCompatibilityMatrix {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &GenericCompatibilityMatrix{
		logger:               logger,
		cache:                make(map[string]*MatrixCompatibilityResult),
		cacheEnabled:         true,
		strictConstraintMode: false,
		allowPartialMatches:  true,
	}
}

// SetCompatibilityChecker sets the compatibility checker for advanced operations.
func (gcm *GenericCompatibilityMatrix) SetCompatibilityChecker(checker *TypeCompatibilityChecker) {
	gcm.compatibilityChecker = checker
	gcm.logger.Debug("compatibility checker configured for matrix")
}

// AnalyzeCompatibility performs comprehensive generic type compatibility analysis.
func (gcm *GenericCompatibilityMatrix) AnalyzeCompatibility(
	ctx context.Context,
	sourceType, targetType Type,
) (*MatrixCompatibilityResult, error) {
	startTime := time.Now()

	gcm.logger.Debug("analyzing generic compatibility with matrix",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()))

	// Generate cache key
	cacheKey := fmt.Sprintf("%s<->%s", sourceType.String(), targetType.String())

	// Check cache first
	if gcm.cacheEnabled {
		if cached, found := gcm.cache[cacheKey]; found {
			gcm.logger.Debug("matrix cache hit",
				zap.String("cache_key", cacheKey))
			return cached, nil
		}
	}

	// Perform structural analysis
	sourceStructure := gcm.analyzeTypeStructure(sourceType)
	targetStructure := gcm.analyzeTypeStructure(targetType)

	// Create matrix analysis
	matrixAnalysis := &MatrixAnalysis{
		SourceStructure:   sourceStructure,
		TargetStructure:   targetStructure,
		StructuralMatch:   gcm.checkStructuralMatch(sourceStructure, targetStructure),
		PartialMatchScore: gcm.calculatePartialMatchScore(sourceStructure, targetStructure),
	}

	// Analyze constraint compatibility
	constraintMatrix := gcm.analyzeConstraintCompatibility(sourceType, targetType)

	// Get basic compatibility result
	var baseResult *CompatibilityResult
	var err error
	if gcm.compatibilityChecker != nil {
		baseResult, err = gcm.compatibilityChecker.CheckAssignabilityWithContext(ctx, sourceType, targetType)
		if err != nil {
			return nil, fmt.Errorf("base compatibility check failed: %w", err)
		}
	} else {
		// Fallback compatibility check
		baseResult = gcm.performBasicCompatibilityCheck(sourceType, targetType)
	}

	// Create matrix result
	result := &MatrixCompatibilityResult{
		CompatibilityResult:     baseResult,
		MatrixAnalysis:          matrixAnalysis,
		ConstraintCompatibility: constraintMatrix,
		TypeParameterMappings:   gcm.generateTypeParameterMappings(sourceType, targetType, constraintMatrix),
	}

	// Add recommendation if needed
	if !result.Compatible && gcm.allowPartialMatches {
		matrixAnalysis.RecommendedConversion = gcm.generateConversionRecommendation(matrixAnalysis, constraintMatrix)
	}

	// Update timing
	result.CheckDurationMS = time.Since(startTime).Milliseconds()

	// Cache result
	if gcm.cacheEnabled {
		gcm.cache[cacheKey] = result
	}

	gcm.logger.Debug("matrix compatibility analysis completed",
		zap.String("source_type", sourceType.String()),
		zap.String("target_type", targetType.String()),
		zap.Bool("compatible", result.Compatible),
		zap.Bool("structural_match", matrixAnalysis.StructuralMatch),
		zap.Float64("partial_match_score", matrixAnalysis.PartialMatchScore),
		zap.Int64("duration_ms", result.CheckDurationMS))

	return result, nil
}

// analyzeTypeStructure analyzes the structural characteristics of a type.
func (gcm *GenericCompatibilityMatrix) analyzeTypeStructure(t Type) TypeStructure {
	structure := TypeStructure{
		IsGeneric:          t.Generic(),
		TypeParameterCount: len(t.TypeParams()),
		ConstraintTypes:    make([]string, 0),
		GenericComplexity:  0,
	}

	if t.Underlying() != nil {
		structure.UnderlyingType = t.Underlying().String()
	}

	// Analyze type parameters
	if t.Generic() {
		params := t.TypeParams()
		structure.ConstraintTypes = make([]string, len(params))

		for i, param := range params {
			structure.ConstraintTypes[i] = param.GetConstraintType()

			// Calculate complexity based on constraint type
			switch param.GetConstraintType() {
			case anyConstraint:
				structure.GenericComplexity++
			case comparableConstraint:
				structure.GenericComplexity += 2
			case unionConstraint, unionUnderlyingConstraint:
				structure.GenericComplexity += 4
			case interfaceKeyword:
				structure.GenericComplexity += 3
			case underlyingConstraint:
				structure.GenericComplexity += 3
			default:
				structure.GenericComplexity += 5 // Custom constraints are most complex
			}
		}
	}

	// Analyze interface methods if applicable
	if t.Kind() == KindInterface {
		// In a full implementation, you would extract method signatures
		// For now, we'll use a simplified approach
		structure.InterfaceMethods = []string{} // Placeholder
	}

	return structure
}

// checkStructuralMatch checks if two type structures are compatible.
func (gcm *GenericCompatibilityMatrix) checkStructuralMatch(source, target TypeStructure) bool {
	// Basic structural requirements
	if source.IsGeneric != target.IsGeneric {
		return false
	}

	if source.TypeParameterCount != target.TypeParameterCount {
		return false
	}

	// For non-generic types, check underlying compatibility
	if !source.IsGeneric && !target.IsGeneric {
		return source.UnderlyingType == target.UnderlyingType ||
			source.UnderlyingType == "" ||
			target.UnderlyingType == ""
	}

	// For generic types, we need more sophisticated matching
	return gcm.checkConstraintStructuralMatch(source.ConstraintTypes, target.ConstraintTypes)
}

// checkConstraintStructuralMatch checks if constraint structures are compatible.
func (gcm *GenericCompatibilityMatrix) checkConstraintStructuralMatch(sourceConstraints, targetConstraints []string) bool {
	if len(sourceConstraints) != len(targetConstraints) {
		return false
	}

	for i, sourceConstraint := range sourceConstraints {
		targetConstraint := targetConstraints[i]

		if !gcm.areConstraintsStructurallyCompatible(sourceConstraint, targetConstraint) {
			return false
		}
	}

	return true
}

// areConstraintsStructurallyCompatible checks structural compatibility of constraints.
func (gcm *GenericCompatibilityMatrix) areConstraintsStructurallyCompatible(source, target string) bool {
	// Exact match
	if source == target {
		return true
	}

	// 'any' is compatible with everything
	if source == anyConstraint || target == anyConstraint {
		return true
	}

	// Specific compatibility rules
	switch source {
	case comparableConstraint:
		return gcm.isComparableCompatible(target)
	case unionConstraint, unionUnderlyingConstraint:
		return strings.HasPrefix(target, unionConstraint)
	case underlyingConstraint:
		return target == underlyingConstraint || strings.HasPrefix(target, unionConstraint)
	case interfaceKeyword:
		return target == interfaceKeyword
	default:
		// For custom constraints, require exact match in strict mode
		if gcm.strictConstraintMode {
			return source == target
		}
		return true // Allow in non-strict mode
	}
}

// isComparableCompatible checks if a constraint is compatible with 'comparable'.
func (gcm *GenericCompatibilityMatrix) isComparableCompatible(constraint string) bool {
	comparableConstraints := []string{
		comparableConstraint, anyConstraint, "int", "string", "bool", "float64", "int64",
	}

	for _, comp := range comparableConstraints {
		if constraint == comp {
			return true
		}
	}

	return false
}

// calculatePartialMatchScore calculates a score representing partial compatibility.
func (gcm *GenericCompatibilityMatrix) calculatePartialMatchScore(source, target TypeStructure) float64 {
	if source.IsGeneric != target.IsGeneric {
		return 0.0
	}

	if !source.IsGeneric {
		// For non-generic types, score based on underlying type similarity
		if source.UnderlyingType == target.UnderlyingType {
			return 1.0
		}
		return 0.5 // Some compatibility might exist
	}

	// For generic types, calculate based on constraint compatibility
	if source.TypeParameterCount != target.TypeParameterCount {
		return 0.0
	}

	if len(source.ConstraintTypes) == 0 || len(target.ConstraintTypes) == 0 {
		return 0.5
	}

	matchingConstraints := 0
	totalConstraints := len(source.ConstraintTypes)

	for i, sourceConstraint := range source.ConstraintTypes {
		if i < len(target.ConstraintTypes) {
			targetConstraint := target.ConstraintTypes[i]
			if gcm.areConstraintsStructurallyCompatible(sourceConstraint, targetConstraint) {
				matchingConstraints++
			}
		}
	}

	return float64(matchingConstraints) / float64(totalConstraints)
}

// analyzeConstraintCompatibility analyzes constraint compatibility in detail.
func (gcm *GenericCompatibilityMatrix) analyzeConstraintCompatibility(sourceType, targetType Type) *ConstraintMatrix {
	matrix := &ConstraintMatrix{
		SourceConstraints: make([]ConstraintInfo, 0),
		TargetConstraints: make([]ConstraintInfo, 0),
		CompatibilityMap:  make(map[string]string),
		IncompatiblePairs: make([]ConstraintPair, 0),
		PartialMatches:    make([]PartialMatch, 0),
	}

	// Analyze source constraints
	if sourceType.Generic() {
		for _, param := range sourceType.TypeParams() {
			constraintInfo := gcm.analyzeConstraint(param)
			matrix.SourceConstraints = append(matrix.SourceConstraints, constraintInfo)
		}
	}

	// Analyze target constraints
	if targetType.Generic() {
		for _, param := range targetType.TypeParams() {
			constraintInfo := gcm.analyzeConstraint(param)
			matrix.TargetConstraints = append(matrix.TargetConstraints, constraintInfo)
		}
	}

	// Build compatibility map
	gcm.buildCompatibilityMap(matrix)

	return matrix
}

// analyzeConstraint analyzes a single constraint in detail.
func (gcm *GenericCompatibilityMatrix) analyzeConstraint(param TypeParam) ConstraintInfo {
	constraintType := param.GetConstraintType()

	info := ConstraintInfo{
		ParameterName:       param.Name,
		ConstraintType:      constraintType,
		ConstraintSpecifics: constraintType,
		AllowedTypes:        make([]string, 0),
	}

	// Determine strictness
	switch constraintType {
	case anyConstraint:
		info.Strictness = ConstraintAny
		info.AllowedTypes = []string{anyConstraint}
	case comparableConstraint:
		info.Strictness = ConstraintComparable
		info.AllowedTypes = []string{"int", "string", "bool", "float64", "int64", "uint64", comparableConstraint}
	case unionConstraint, unionUnderlyingConstraint:
		info.Strictness = ConstraintUnion
		if 0 < len(param.UnionTypes) {
			for _, unionType := range param.UnionTypes {
				info.AllowedTypes = append(info.AllowedTypes, unionType.String())
			}
		}
	case underlyingConstraint:
		info.Strictness = ConstraintUnderlying
		if param.Underlying != nil {
			info.AllowedTypes = []string{param.Underlying.Type.String()}
		}
	case interfaceKeyword:
		info.Strictness = ConstraintInterface
		if param.Constraint != nil {
			info.AllowedTypes = []string{param.Constraint.String()}
		}
	default:
		info.Strictness = ConstraintExact
		info.AllowedTypes = []string{constraintType}
	}

	return info
}

// buildCompatibilityMap builds the compatibility mapping between constraints.
func (gcm *GenericCompatibilityMatrix) buildCompatibilityMap(matrix *ConstraintMatrix) {
	// Map source constraints to target constraints
	for i, sourceConstraint := range matrix.SourceConstraints {
		bestMatch := ""
		bestScore := 0.0

		for j, targetConstraint := range matrix.TargetConstraints {
			if i == j { // Same position
				score := gcm.calculateConstraintCompatibilityScore(sourceConstraint, targetConstraint)
				if bestScore < score {
					bestMatch = targetConstraint.ParameterName
					bestScore = score
				}
			}
		}

		if bestMatch != "" {
			matrix.CompatibilityMap[sourceConstraint.ParameterName] = bestMatch

			if bestScore < 1.0 {
				// Add partial match
				matrix.PartialMatches = append(matrix.PartialMatches, PartialMatch{
					SourceParameter: sourceConstraint.ParameterName,
					TargetParameter: bestMatch,
					MatchScore:      bestScore,
				})
			}
		} else {
			// Add incompatible pair
			if i < len(matrix.TargetConstraints) {
				matrix.IncompatiblePairs = append(matrix.IncompatiblePairs, ConstraintPair{
					SourceConstraint:      sourceConstraint.ConstraintType,
					TargetConstraint:      matrix.TargetConstraints[i].ConstraintType,
					IncompatibilityReason: "constraints are structurally incompatible",
				})
			}
		}
	}
}

// calculateConstraintCompatibilityScore calculates compatibility score between constraints.
func (gcm *GenericCompatibilityMatrix) calculateConstraintCompatibilityScore(source, target ConstraintInfo) float64 {
	// Exact match
	if source.ConstraintType == target.ConstraintType {
		return 1.0
	}

	// 'any' compatibility
	if source.ConstraintType == anyConstraint || target.ConstraintType == anyConstraint {
		return 0.9
	}

	// Strictness-based compatibility
	if source.Strictness == target.Strictness {
		return 0.7
	}

	// Check allowed types overlap
	overlap := 0
	for _, sourceType := range source.AllowedTypes {
		for _, targetType := range target.AllowedTypes {
			if sourceType == targetType {
				overlap++
			}
		}
	}

	if 0 < overlap {
		maxTypes := len(source.AllowedTypes)
		if len(target.AllowedTypes) > maxTypes {
			maxTypes = len(target.AllowedTypes)
		}
		return float64(overlap) / float64(maxTypes) * 0.8
	}

	return 0.0
}

// generateTypeParameterMappings generates mappings between type parameters.
func (gcm *GenericCompatibilityMatrix) generateTypeParameterMappings(
	sourceType, targetType Type,
	matrix *ConstraintMatrix,
) map[string]string {
	mappings := make(map[string]string)

	// Use compatibility map as base
	for source, target := range matrix.CompatibilityMap {
		mappings[source] = target
	}

	// Add direct positional mappings for unmatched parameters
	sourceParams := sourceType.TypeParams()
	targetParams := targetType.TypeParams()

	minParams := len(sourceParams)
	if len(targetParams) < minParams {
		minParams = len(targetParams)
	}

	for i := 0; i < minParams; i++ {
		sourceParam := sourceParams[i].Name
		targetParam := targetParams[i].Name

		// Only add if not already mapped
		if _, exists := mappings[sourceParam]; !exists {
			mappings[sourceParam] = targetParam
		}
	}

	return mappings
}

// generateConversionRecommendation generates a recommendation for type conversion.
func (gcm *GenericCompatibilityMatrix) generateConversionRecommendation(
	analysis *MatrixAnalysis,
	matrix *ConstraintMatrix,
) string {
	if analysis.PartialMatchScore > 0.7 {
		return "Consider type assertion or explicit conversion"
	}

	if analysis.PartialMatchScore > 0.5 {
		return "Partial compatibility detected, conversion may be possible with constraint relaxation"
	}

	if 0 < len(matrix.PartialMatches) {
		return "Partial parameter matches found, consider generic type instantiation"
	}

	return "Types appear incompatible, consider refactoring or alternative approaches"
}

// performBasicCompatibilityCheck provides fallback compatibility checking.
func (gcm *GenericCompatibilityMatrix) performBasicCompatibilityCheck(sourceType, targetType Type) *CompatibilityResult {
	// This is a simplified fallback when no compatibility checker is available
	result := &CompatibilityResult{
		Level:      CompatibilityNone,
		Compatible: false,
	}

	// Basic checks
	if sourceType.String() == targetType.String() {
		result.Level = CompatibilityIdentical
		result.Compatible = true
		return result
	}

	if sourceType.Kind() == targetType.Kind() {
		result.Level = CompatibilityWithConversion
		result.Compatible = true
		result.RequiresConversion = true
		return result
	}

	result.ErrorMessage = fmt.Sprintf("basic compatibility check: %s not compatible with %s",
		sourceType.String(), targetType.String())
	return result
}

// ClearCache clears the compatibility cache.
func (gcm *GenericCompatibilityMatrix) ClearCache() {
	gcm.cache = make(map[string]*MatrixCompatibilityResult)
	gcm.logger.Debug("matrix compatibility cache cleared")
}

// GetCacheSize returns the current cache size.
func (gcm *GenericCompatibilityMatrix) GetCacheSize() int {
	return len(gcm.cache)
}

// SetStrictConstraintMode enables or disables strict constraint checking.
func (gcm *GenericCompatibilityMatrix) SetStrictConstraintMode(strict bool) {
	gcm.strictConstraintMode = strict
	gcm.logger.Debug("strict constraint mode updated",
		zap.Bool("strict_mode", strict))
}

// SetAllowPartialMatches enables or disables partial match analysis.
func (gcm *GenericCompatibilityMatrix) SetAllowPartialMatches(allow bool) {
	gcm.allowPartialMatches = allow
	gcm.logger.Debug("partial matches setting updated",
		zap.Bool("allow_partial_matches", allow))
}
