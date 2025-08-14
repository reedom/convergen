package parser

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrInvalidConstraint           = errors.New("invalid constraint")
	ErrUnsupportedConstraintType   = errors.New("unsupported constraint type")
	ErrEmptyUnionConstraint        = errors.New("union constraint cannot be empty")
	ErrInvalidUnderlyingConstraint = errors.New("invalid underlying type constraint")
	ErrCircularConstraint          = errors.New("circular constraint detected")
)

const (
	comparableConstraint = "comparable"
	anyConstraint        = "any"
	interfaceKeyword     = "interface"
	unionUnderlyingType  = "union_underlying"
)

// ConstraintParser handles parsing of Go type constraints.
// It supports parsing all Go generic constraint syntax including unions,
// underlying types, and interface constraints.
type ConstraintParser struct {
	typeResolver       *TypeResolver
	logger             *zap.Logger
	visitedConstraints map[string]bool // Track visited constraints to detect cycles
	mu                 sync.RWMutex    // Protects visitedConstraints map from concurrent access
}

// ParsedConstraint represents a fully parsed type constraint.
// It provides a structured representation of Go constraint syntax
// that can be easily analyzed and validated.
type ParsedConstraint struct {
	// Core constraint information
	Type         domain.Type `json:"type"`
	IsAny        bool        `json:"is_any"`
	IsComparable bool        `json:"is_comparable"`

	// Union constraint support (~int | ~string | ~float64)
	UnionTypes []domain.Type `json:"union_types,omitempty"`

	// Underlying type constraint support (~string, ~int)
	Underlying *domain.UnderlyingConstraint `json:"underlying,omitempty"`

	// Interface constraint support
	InterfaceType domain.Type `json:"interface_type,omitempty"`

	// Parse metadata
	ConstraintType string        `json:"constraint_type"`
	ParseDuration  time.Duration `json:"parse_duration"`
	Valid          bool          `json:"valid"`
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// NewConstraintParser creates a new constraint parser with the given type resolver and logger.
func NewConstraintParser(typeResolver *TypeResolver, logger *zap.Logger) *ConstraintParser {
	return &ConstraintParser{
		typeResolver:       typeResolver,
		logger:             logger,
		visitedConstraints: make(map[string]bool),
	}
}

// ParseConstraint parses complex constraint combinations and nested expressions.
func (cp *ConstraintParser) ParseConstraint(
	ctx context.Context,
	constraint types.Type,
) (*ParsedConstraint, error) {
	startTime := time.Now()

	// Handle nil constraint (represents 'any')
	if constraint == nil {
		return cp.createAnyConstraint(startTime), nil
	}

	// Check for circular constraint dependencies
	constraintKey := constraint.String()
	if cp.isCircularConstraint(constraintKey) {
		return cp.createAnyConstraintForCircular(constraintKey, startTime), nil
	}

	// Mark constraint as visited and setup cleanup
	cp.markConstraintVisited(constraintKey)
	defer cp.unmarkConstraintVisited(constraintKey)

	cp.logger.Debug("parsing constraint",
		zap.String("constraint", constraint.String()),
		zap.String("constraint_type", fmt.Sprintf("%T", constraint)))

	// Parse constraint by type
	result := &ParsedConstraint{
		ParseDuration: 0, // Will be set at the end
		Valid:         false,
	}

	if err := cp.parseConstraintByType(ctx, constraint, result); err != nil {
		result.ParseDuration = time.Since(startTime)
		return result, err
	}

	// Finalize result
	result.Valid = true
	result.ParseDuration = time.Since(startTime)

	cp.logger.Debug("constraint parsed successfully",
		zap.String("constraint_type", result.ConstraintType),
		zap.Duration("parse_duration", result.ParseDuration),
		zap.Bool("valid", result.Valid))

	return result, nil
}

// parseInterfaceConstraint handles interface constraints including 'comparable' and custom interfaces.
func (cp *ConstraintParser) parseInterfaceConstraint(
	ctx context.Context,
	iface *types.Interface,
	result *ParsedConstraint,
) error {
	// Check for 'comparable' constraint
	if cp.isComparableInterface(iface) {
		result.IsComparable = true
		result.ConstraintType = comparableConstraint
		cp.logger.Debug("detected comparable constraint")
		return nil
	}

	// Check for empty interface (equivalent to 'any')
	if iface.NumMethods() == 0 && iface.NumEmbeddeds() == 0 {
		result.IsAny = true
		result.ConstraintType = anyConstraint
		cp.logger.Debug("detected any constraint (empty interface)")
		return nil
	}

	// Check for union constraints represented as interfaces
	if cp.isUnionInterface(iface) {
		return cp.parseUnionInterfaceConstraint(ctx, iface, result)
	}

	// Handle custom interface constraints
	domainType, err := cp.typeResolver.ResolveType(ctx, iface)
	if err != nil {
		return fmt.Errorf("failed to resolve interface type: %w", err)
	}

	result.Type = domainType
	result.InterfaceType = domainType
	result.ConstraintType = interfaceKeyword

	cp.logger.Debug("detected interface constraint",
		zap.String("interface_name", domainType.String()),
		zap.Int("num_methods", iface.NumMethods()),
		zap.Int("num_embeddeds", iface.NumEmbeddeds()))

	return nil
}

// parseUnionConstraint handles union constraints like ~int | ~string | ~float64.
func (cp *ConstraintParser) parseUnionConstraint(
	ctx context.Context,
	union *types.Union,
	result *ParsedConstraint,
) error {
	if union.Len() == 0 {
		return ErrEmptyUnionConstraint
	}

	unionTypes := make([]domain.Type, 0, union.Len())
	hasUnderlying := false

	for i := 0; i < union.Len(); i++ {
		term := union.Term(i)

		cp.logger.Debug("parsing union term",
			zap.Int("index", i),
			zap.String("type", term.Type().String()),
			zap.Bool("tilde", term.Tilde()))

		// Check if this is an underlying type constraint (~T)
		if term.Tilde() {
			hasUnderlying = true
		}

		// Resolve the domain type
		domainType, err := cp.typeResolver.ResolveType(ctx, term.Type())
		if err != nil {
			return fmt.Errorf("failed to resolve union term %d (%s): %w", i, term.Type().String(), err)
		}

		unionTypes = append(unionTypes, domainType)
	}

	result.UnionTypes = unionTypes
	result.ConstraintType = "union"

	if hasUnderlying {
		result.ConstraintType = unionUnderlyingType
		cp.logger.Debug("detected union constraint with underlying types")
	} else {
		cp.logger.Debug("detected union constraint")
	}

	cp.logger.Debug("parsed union constraint",
		zap.Int("num_types", len(unionTypes)),
		zap.Bool("has_underlying", hasUnderlying))

	return nil
}

// parseNamedConstraint handles named type constraints.
func (cp *ConstraintParser) parseNamedConstraint(
	ctx context.Context,
	named *types.Named,
	result *ParsedConstraint,
) error {
	// Check for predefined constraint types using helper
	if cp.handlePredefinedConstraint(named.Obj().Name(), result, "named") {
		return nil
	}

	// Handle custom named constraints
	domainType, err := cp.typeResolver.ResolveType(ctx, named)
	if err != nil {
		return fmt.Errorf("failed to resolve named constraint: %w", err)
	}

	result.Type = domainType
	result.ConstraintType = "named"

	cp.logger.Debug("detected named constraint",
		zap.String("name", named.Obj().Name()),
		zap.String("package", named.Obj().Pkg().Name()))

	return nil
}

// parseBasicConstraint handles basic type constraints.
func (cp *ConstraintParser) parseBasicConstraint(
	ctx context.Context,
	basic *types.Basic,
	result *ParsedConstraint,
) error {
	domainType, err := cp.typeResolver.ResolveType(ctx, basic)
	if err != nil {
		return fmt.Errorf("failed to resolve basic constraint: %w", err)
	}

	result.Type = domainType
	result.ConstraintType = "basic"

	cp.logger.Debug("detected basic constraint",
		zap.String("name", basic.Name()),
		zap.String("kind", basic.String()))

	return nil
}

// parseAliasConstraint handles alias type constraints (like 'any' in Go 1.18+).
func (cp *ConstraintParser) parseAliasConstraint(
	ctx context.Context,
	alias *types.Alias,
	result *ParsedConstraint,
) error {
	// Check for predefined constraint aliases using helper
	if cp.handlePredefinedConstraint(alias.Obj().Name(), result, "alias") {
		return nil
	}

	// Handle custom alias constraints by resolving the underlying type
	domainType, err := cp.typeResolver.ResolveType(ctx, alias)
	if err != nil {
		return fmt.Errorf("failed to resolve alias constraint: %w", err)
	}

	result.Type = domainType
	result.ConstraintType = "alias"

	cp.logger.Debug("detected alias constraint",
		zap.String("name", alias.Obj().Name()),
		zap.String("package", alias.Obj().Pkg().Name()))

	return nil
}

// handlePredefinedConstraint checks for predefined constraint types and sets the result accordingly.
// Returns true if a predefined constraint was handled, false otherwise.
func (cp *ConstraintParser) handlePredefinedConstraint(constraintName string, result *ParsedConstraint, contextType string) bool {
	switch constraintName {
	case anyConstraint:
		result.IsAny = true
		result.ConstraintType = anyConstraint
		cp.logger.Debug("detected any constraint (" + contextType + ")")
		return true

	case comparableConstraint:
		result.IsComparable = true
		result.ConstraintType = comparableConstraint
		cp.logger.Debug("detected comparable constraint (" + contextType + ")")
		return true

	default:
		return false
	}
}

// isComparableInterface checks if an interface represents the 'comparable' constraint.
func (cp *ConstraintParser) isComparableInterface(iface *types.Interface) bool {
	// Check if this is the built-in 'comparable' interface
	// The comparable interface has specific characteristics in the Go type system
	if iface.NumMethods() == 0 && 0 < iface.NumEmbeddeds() {
		for i := 0; i < iface.NumEmbeddeds(); i++ {
			embedded := iface.EmbeddedType(i)
			if named, ok := embedded.(*types.Named); ok {
				if named.Obj().Name() == comparableConstraint {
					return true
				}
			}
		}
	}

	// Check by string representation as fallback
	return strings.Contains(iface.String(), comparableConstraint)
}

// isUnionInterface checks if an interface represents a union constraint (~int | ~string).
func (cp *ConstraintParser) isUnionInterface(iface *types.Interface) bool {
	// Union constraints appear as interfaces with no methods but embedded types
	if 0 < iface.NumMethods() {
		return false
	}

	// Check if it has embedded types that look like union elements
	if 0 < iface.NumEmbeddeds() {
		// Look for union syntax in string representation
		str := iface.String()
		return strings.Contains(str, "|") || strings.Contains(str, "~")
	}

	return false
}

// parseUnionInterfaceConstraint handles union constraints represented as interfaces.
func (cp *ConstraintParser) parseUnionInterfaceConstraint(
	ctx context.Context,
	iface *types.Interface,
	result *ParsedConstraint,
) error {
	// For union constraints represented as interfaces, we can extract the underlying types
	// by examining the embedded types or analyzing the string representation

	str := iface.String()
	cp.logger.Debug("parsing union interface constraint",
		zap.String("interface_str", str),
		zap.Int("num_embeddeds", iface.NumEmbeddeds()))

	// Check if this is an underlying type constraint (~string)
	if strings.Contains(str, "~") && !strings.Contains(str, "|") {
		return cp.parseUnderlyingInterfaceConstraint(ctx, iface, result)
	}

	// For complex union constraints, we'll parse them as union types
	unionTypes := make([]domain.Type, 0)
	hasUnderlying := strings.Contains(str, "~")

	// Create domain types based on embedded types if available
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embedded := iface.EmbeddedType(i)

		// Handle union types specially
		if union, ok := embedded.(*types.Union); ok {
			// Parse the union constraint directly
			unionResult := &ParsedConstraint{}
			err := cp.parseUnionConstraint(ctx, union, unionResult)
			if err != nil {
				cp.logger.Warn("failed to parse embedded union type",
					zap.Error(err),
					zap.String("union_type", union.String()))
				continue
			}
			// Add all union types to our result
			unionTypes = append(unionTypes, unionResult.UnionTypes...)
			if unionResult.ConstraintType == unionUnderlyingType {
				hasUnderlying = true
			}
		} else {
			// Regular type resolution for non-union types
			domainType, err := cp.typeResolver.ResolveType(ctx, embedded)
			if err != nil {
				cp.logger.Warn("failed to resolve embedded type in union",
					zap.Error(err),
					zap.String("embedded_type", embedded.String()))
				continue
			}
			unionTypes = append(unionTypes, domainType)
		}
	}

	result.UnionTypes = unionTypes
	if hasUnderlying {
		result.ConstraintType = unionUnderlyingType
	} else {
		result.ConstraintType = "union"
	}

	cp.logger.Debug("parsed union interface constraint",
		zap.Int("union_types", len(unionTypes)),
		zap.Bool("has_underlying", hasUnderlying))

	return nil
}

// parseUnderlyingInterfaceConstraint handles underlying type constraints represented as interfaces.
func (cp *ConstraintParser) parseUnderlyingInterfaceConstraint(
	ctx context.Context,
	iface *types.Interface,
	result *ParsedConstraint,
) error {
	// For ~string constraint, try to extract the underlying type
	if 0 < iface.NumEmbeddeds() {
		return cp.parseEmbeddedConstraint(ctx, iface, result)
	}

	// Fallback: treat as regular interface constraint
	domainType, err := cp.typeResolver.ResolveType(ctx, iface)
	if err != nil {
		return fmt.Errorf("failed to resolve interface constraint: %w", err)
	}

	result.Type = domainType
	result.ConstraintType = interfaceKeyword

	cp.logger.Debug("parsed interface constraint (underlying fallback)",
		zap.String("interface_type", domainType.String()))

	return nil
}

// parseEmbeddedConstraint parses embedded constraint types from interface.
func (cp *ConstraintParser) parseEmbeddedConstraint(ctx context.Context, iface *types.Interface, result *ParsedConstraint) error {
	embedded := iface.EmbeddedType(0)

	// Handle union types specially (e.g., for constraints like ~string)
	if union, ok := embedded.(*types.Union); ok {
		return cp.handleUnionConstraint(ctx, union, result)
	}

	// Regular type resolution for non-union embedded types
	domainType, err := cp.typeResolver.ResolveType(ctx, embedded)
	if err != nil {
		return fmt.Errorf("failed to resolve underlying type: %w", err)
	}

	result.Underlying = domain.NewUnderlyingConstraint(domainType, domainType.Package())
	result.Type = domainType
	result.ConstraintType = "underlying"

	cp.logger.Debug("parsed underlying interface constraint",
		zap.String("underlying_type", domainType.String()))

	return nil
}

// handleUnionConstraint handles union constraint types.
func (cp *ConstraintParser) handleUnionConstraint(ctx context.Context, union *types.Union, result *ParsedConstraint) error {
	// For underlying constraints, we expect a single term with tilde
	if union.Len() == 1 {
		term := union.Term(0)
		if term.Tilde() {
			domainType, err := cp.typeResolver.ResolveType(ctx, term.Type())
			if err != nil {
				return fmt.Errorf("failed to resolve underlying type: %w", err)
			}

			result.Underlying = domain.NewUnderlyingConstraint(domainType, domainType.Package())
			result.Type = domainType
			result.ConstraintType = "underlying"

			cp.logger.Debug("parsed underlying interface constraint from union",
				zap.String("underlying_type", domainType.String()))

			return nil
		}
	}
	// If it's a multi-term union, treat as union constraint
	return cp.parseUnionConstraint(ctx, union, result)
}

// ValidateConstraint validates that a constraint is well-formed and supported.
func (cp *ConstraintParser) ValidateConstraint(constraint *ParsedConstraint) error {
	if constraint == nil {
		return fmt.Errorf("%w: constraint is nil", ErrInvalidConstraint)
	}

	if !constraint.Valid {
		return fmt.Errorf("%w: %s", ErrInvalidConstraint, constraint.ErrorMessage)
	}

	// Validate constraint type consistency
	constraintCount := 0
	if constraint.IsAny {
		constraintCount++
	}
	if constraint.IsComparable {
		constraintCount++
	}
	if 0 < len(constraint.UnionTypes) {
		constraintCount++
	}
	if constraint.Underlying != nil {
		constraintCount++
	}
	if constraint.InterfaceType != nil {
		constraintCount++
	}

	if 1 < constraintCount {
		return fmt.Errorf("%w: multiple constraint types detected", ErrInvalidConstraint)
	}

	if constraintCount == 0 && constraint.Type == nil {
		return fmt.Errorf("%w: no constraint type detected", ErrInvalidConstraint)
	}

	return nil
}

// GetConstraintTypeString returns a human-readable string representation of the constraint type.
func (cp *ConstraintParser) GetConstraintTypeString(constraint *ParsedConstraint) string {
	if constraint == nil {
		return "unknown"
	}

	if constraint.IsAny {
		return anyConstraint
	}
	if constraint.IsComparable {
		return comparableConstraint
	}
	if 0 < len(constraint.UnionTypes) {
		types := make([]string, len(constraint.UnionTypes))
		for i, t := range constraint.UnionTypes {
			if constraint.ConstraintType == unionUnderlyingType {
				types[i] = "~" + t.String()
			} else {
				types[i] = t.String()
			}
		}
		return strings.Join(types, " | ")
	}
	if constraint.Underlying != nil {
		return "~" + constraint.Underlying.Type.String()
	}
	if constraint.InterfaceType != nil {
		return constraint.InterfaceType.String()
	}
	if constraint.Type != nil {
		return constraint.Type.String()
	}

	return "unknown"
}

// ConvertToDomainTypeParam converts a ParsedConstraint to a domain.TypeParam.
// This provides integration with the enhanced TypeParam structure from TASK-001.
func (cp *ConstraintParser) ConvertToDomainTypeParam(
	name string,
	index int,
	constraint *ParsedConstraint,
) (*domain.TypeParam, error) {
	if constraint == nil {
		return nil, fmt.Errorf("%w: constraint is nil", ErrInvalidConstraint)
	}

	if err := cp.ValidateConstraint(constraint); err != nil {
		return nil, fmt.Errorf("invalid constraint for type param conversion: %w", err)
	}

	// Create appropriate TypeParam based on constraint type
	if constraint.IsAny {
		return domain.NewAnyTypeParam(name, index), nil
	}

	if constraint.IsComparable {
		return domain.NewComparableTypeParam(name, index), nil
	}

	if 0 < len(constraint.UnionTypes) {
		if constraint.ConstraintType == unionUnderlyingType {
			return domain.NewUnionUnderlyingTypeParam(name, constraint.UnionTypes, index), nil
		}
		return domain.NewUnionTypeParam(name, constraint.UnionTypes, index), nil
	}

	if constraint.Underlying != nil {
		return domain.NewUnderlyingTypeParam(name, constraint.Underlying, index), nil
	}

	// Handle interface or basic constraints
	if constraint.Type != nil {
		return domain.NewTypeParam(name, constraint.Type, index), nil
	}

	return nil, fmt.Errorf("%w: no valid constraint found", ErrInvalidConstraint)
}

// Helper methods for ParseConstraint refactoring

// createAnyConstraint creates a ParsedConstraint for 'any' constraints.
func (cp *ConstraintParser) createAnyConstraint(startTime time.Time) *ParsedConstraint {
	cp.logger.Debug("parsing nil constraint (any)")
	return &ParsedConstraint{
		Type:           nil,
		IsAny:          true,
		IsComparable:   false,
		ConstraintType: anyConstraint,
		ParseDuration:  time.Since(startTime),
		Valid:          true,
	}
}

// isCircularConstraint checks if a constraint creates a circular dependency.
func (cp *ConstraintParser) isCircularConstraint(constraintKey string) bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.visitedConstraints[constraintKey]
}

// createAnyConstraintForCircular creates an 'any' constraint for circular dependencies.
func (cp *ConstraintParser) createAnyConstraintForCircular(constraintKey string, startTime time.Time) *ParsedConstraint {
	cp.logger.Warn("circular constraint detected, treating as 'any'",
		zap.String("constraint", constraintKey))

	return &ParsedConstraint{
		Type:           nil,
		IsAny:          true,
		IsComparable:   false,
		ConstraintType: anyConstraint,
		ParseDuration:  time.Since(startTime),
		Valid:          true,
	}
}

// markConstraintVisited marks a constraint as visited to detect cycles.
func (cp *ConstraintParser) markConstraintVisited(constraintKey string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.visitedConstraints[constraintKey] = true
}

// unmarkConstraintVisited removes a constraint from the visited set.
func (cp *ConstraintParser) unmarkConstraintVisited(constraintKey string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	delete(cp.visitedConstraints, constraintKey)
}

// parseConstraintByType parses the constraint based on its Go type.
func (cp *ConstraintParser) parseConstraintByType(ctx context.Context, constraint types.Type, result *ParsedConstraint) error {
	switch constraintType := constraint.(type) {
	case *types.Interface:
		err := cp.parseInterfaceConstraint(ctx, constraintType, result)
		if err != nil {
			result.ErrorMessage = err.Error()
			cp.logger.Error("failed to parse interface constraint", zap.Error(err))
			return fmt.Errorf("failed to parse interface constraint: %w", err)
		}

	case *types.Union:
		err := cp.parseUnionConstraint(ctx, constraintType, result)
		if err != nil {
			result.ErrorMessage = err.Error()
			cp.logger.Error("failed to parse union constraint", zap.Error(err))
			return fmt.Errorf("failed to parse union constraint: %w", err)
		}

	case *types.Named:
		err := cp.parseNamedConstraint(ctx, constraintType, result)
		if err != nil {
			result.ErrorMessage = err.Error()
			cp.logger.Error("failed to parse named constraint", zap.Error(err))
			return fmt.Errorf("failed to parse named constraint: %w", err)
		}

	case *types.Basic:
		err := cp.parseBasicConstraint(ctx, constraintType, result)
		if err != nil {
			result.ErrorMessage = err.Error()
			cp.logger.Error("failed to parse basic constraint", zap.Error(err))
			return fmt.Errorf("failed to parse basic constraint: %w", err)
		}

	case *types.Alias:
		err := cp.parseAliasConstraint(ctx, constraintType, result)
		if err != nil {
			result.ErrorMessage = err.Error()
			cp.logger.Error("failed to parse alias constraint", zap.Error(err))
			return fmt.Errorf("failed to parse alias constraint: %w", err)
		}

	default:
		err := fmt.Errorf("%w: %T", ErrUnsupportedConstraintType, constraint)
		result.ErrorMessage = err.Error()
		cp.logger.Error("unsupported constraint type",
			zap.String("type", fmt.Sprintf("%T", constraint)),
			zap.String("constraint", constraint.String()))
		return err
	}

	return nil
}
