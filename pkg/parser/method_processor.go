package parser

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrExpectedMethodSignature              = errors.New("expected method signature")
	ErrMethodMustHaveAtLeastOneParameter    = errors.New("method must have at least one parameter")
	ErrMethodMustHaveAtLeastOneReturnValue  = errors.New("method must have at least one return value")
	ErrMultipleReturnsLastMustBeError       = errors.New("if multiple returns, last must be error type")
	ErrValidationAnnotationRequiresFuncName = errors.New("validation annotation requires a function name")
	ErrTimeoutAnnotationRequiresDuration    = errors.New("timeout annotation requires a duration")
)

// MethodProcessor handles analysis and processing of individual methods.
type MethodProcessor struct {
	parser       *ASTParser
	typeResolver *TypeResolver
	logger       *zap.Logger
}

// NewMethodProcessor creates a new method processor.
func NewMethodProcessor(parser *ASTParser, typeResolver *TypeResolver, logger *zap.Logger) *MethodProcessor {
	return &MethodProcessor{
		parser:       parser,
		typeResolver: typeResolver,
		logger:       logger,
	}
}

// processMethod analyzes a single method and converts it to domain.Method.
func (p *ASTParser) processMethod(ctx context.Context, pkg *packages.Package, file *ast.File, methodObj types.Object, interfaceOpts *domain.InterfaceOptions) (*domain.Method, error) {
	signature, ok := methodObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrExpectedMethodSignature, methodObj.Type())
	}

	// Validate method signature
	if err := p.validateMethodSignature(signature, methodObj); err != nil {
		return nil, fmt.Errorf("invalid method signature: %w", err)
	}

	// Extract method annotations
	annotations := p.extractMethodAnnotations(file, methodObj)

	// Parse method-specific options
	methodOpts, err := p.parseMethodOptions(annotations, interfaceOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse method options: %w", err)
	}

	// Check if this is a generic method by examining the interface context
	// Note: In Go, methods cannot have their own type parameters, only the receiver type can be generic
	var interfaceTypeParams []domain.TypeParam
	if signature.Recv() != nil {
		if namedRecv, ok := signature.Recv().Type().(*types.Named); ok {
			// Extract type parameters from the receiver's interface type
			if receiverTypeParams := namedRecv.TypeParams(); receiverTypeParams != nil {
				for i := 0; i < receiverTypeParams.Len(); i++ {
					typeParam := receiverTypeParams.At(i)
					// Convert to domain TypeParam
					domainParam, convertErr := p.convertGoTypeParamToDomainTypeParam(ctx, typeParam)
					if convertErr != nil {
						p.logger.Warn("failed to convert type parameter",
							zap.String("method", methodObj.Name()),
							zap.String("type_param", typeParam.String()),
							zap.Error(convertErr))
						continue
					}
					interfaceTypeParams = append(interfaceTypeParams, *domainParam)
				}
			}
		}
	}

	// Analyze method parameters with generic context
	parameters, err := p.analyzeGenericMethodParameters(ctx, signature.Params(), interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze parameters: %w", err)
	}

	// Analyze return types with generic context
	returns, err := p.analyzeGenericMethodReturns(ctx, signature.Results(), interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze return types: %w", err)
	}

	// Analyze receiver if present (for future use)
	if signature.Recv() != nil {
		_, err := p.analyzeReceiver(ctx, signature.Recv())
		if err != nil {
			return nil, fmt.Errorf("failed to analyze receiver: %w", err)
		}
	}

	// Create field mappings with generic type awareness
	fieldMappings, err := p.createGenericFieldMappings(ctx, parameters, returns, methodOpts, interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create field mappings: %w", err)
	}

	// Create the domain method using the correct constructor
	method, err := domain.NewMethod(methodObj.Name(), parameters[0].Type, returns[0].Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create method: %w", err)
	}

	// Set method parameters and returns
	method.SetSourceParams(parameters)
	method.SetDestinationReturns(returns)

	// Add field mappings to the method
	for _, mapping := range fieldMappings {
		if err := method.AddMapping(mapping); err != nil {
			return nil, fmt.Errorf("failed to add mapping: %w", err)
		}
	}

	p.logger.Debug("processed method",
		zap.String("name", methodObj.Name()),
		zap.Int("parameters", len(parameters)),
		zap.Int("returns", len(returns)),
		zap.Int("field_mappings", len(fieldMappings)),
		zap.Int("type_params", len(interfaceTypeParams)),
		zap.Bool("is_generic", len(interfaceTypeParams) > 0))

	return method, nil
}

// validateMethodSignature ensures the method signature is valid for conversion.
func (p *ASTParser) validateMethodSignature(signature *types.Signature, methodObj types.Object) error {
	if signature.Params().Len() == 0 {
		return fmt.Errorf("%w: method %s", ErrMethodMustHaveAtLeastOneParameter, methodObj.Name())
	}

	if signature.Results().Len() == 0 {
		return fmt.Errorf("%w: method %s", ErrMethodMustHaveAtLeastOneReturnValue, methodObj.Name())
	}

	// Check if last return is error (optional)
	if signature.Results().Len() > 1 {
		lastReturn := signature.Results().At(signature.Results().Len() - 1)
		if !isGoErrorType(lastReturn.Type()) {
			return fmt.Errorf("%w: method %s", ErrMultipleReturnsLastMustBeError, methodObj.Name())
		}
	}

	return nil
}

// extractMethodAnnotations extracts annotations from method comments.
func (p *ASTParser) extractMethodAnnotations(file *ast.File, methodObj types.Object) []*Annotation {
	docComment := p.getMethodDocComment(file, methodObj)
	if docComment == nil {
		return nil
	}

	var annotations []*Annotation

	for _, comment := range docComment.List {
		if annotation := p.parseAnnotation(comment); annotation != nil {
			annotations = append(annotations, annotation)
		}
	}

	return annotations
}

// parseMethodOptions converts method annotations to options, inheriting from interface options.
func (p *ASTParser) parseMethodOptions(annotations []*Annotation, interfaceOpts *domain.InterfaceOptions) (*domain.MethodOptions, error) {
	// Start with interface options as base
	options := &domain.MethodOptions{
		Style:               interfaceOpts.Style,
		MatchRule:           interfaceOpts.MatchRule,
		CaseSensitive:       interfaceOpts.CaseSensitive,
		UseGetter:           interfaceOpts.UseGetter,
		UseStringer:         interfaceOpts.UseStringer,
		UseTypecast:         interfaceOpts.UseTypecast,
		AllowReverse:        interfaceOpts.AllowReverse,
		SkipFields:          append([]string{}, interfaceOpts.SkipFields...),
		FieldMappings:       p.copyStringMap(interfaceOpts.FieldMappings),
		TypeConverters:      p.copyStringMap(interfaceOpts.TypeConverters),
		LiteralAssignments:  p.copyStringMap(interfaceOpts.LiteralAssignments),
		PreprocessFunction:  interfaceOpts.PreprocessFunction,
		PostprocessFunction: interfaceOpts.PostprocessFunction,
		CustomValidation:    "",
		ConcurrencyLevel:    1,
		TimeoutDuration:     0,
	}

	// Apply method-specific annotations
	for _, annotation := range annotations {
		if err := p.applyMethodAnnotation(options, annotation); err != nil {
			return nil, fmt.Errorf("failed to apply annotation %s: %w", annotation.Type, err)
		}
	}

	return options, nil
}

// applyMethodAnnotation applies a method-level annotation to options.
func (p *ASTParser) applyMethodAnnotation(options *domain.MethodOptions, annotation *Annotation) error {
	switch annotation.Type {
	case "style", "match", "case", "case:off", "getter", "getter:off",
		"stringer", "stringer:off", "typecast", "typecast:off", "reverse",
		"skip", "map", "conv", "literal", "preprocess", "postprocess":
		// These are handled the same way as interface annotations
		return p.applyInterfaceAnnotationToMethod(options, annotation)

	case "validation":
		if len(annotation.Args) == 0 {
			return ErrValidationAnnotationRequiresFuncName
		}

		options.CustomValidation = annotation.Args[0]

	case "concurrent":
		if len(annotation.Args) > 0 {
			level, err := parseIntArg(annotation.Args[0])
			if err != nil {
				return fmt.Errorf("invalid concurrent level: %w", err)
			}

			options.ConcurrencyLevel = level
		} else {
			options.ConcurrencyLevel = 4 // Default concurrency level
		}

	case "timeout":
		if len(annotation.Args) == 0 {
			return ErrTimeoutAnnotationRequiresDuration
		}

		duration, err := parseDurationArg(annotation.Args[0])
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}

		options.TimeoutDuration = duration

	default:
		p.logger.Warn("unknown method annotation",
			zap.String("type", annotation.Type),
			zap.String("position", p.fileSet.Position(annotation.Position).String()))
	}

	return nil
}

// analyzeMethodParameters analyzes all parameters of a method.
func (p *ASTParser) analyzeMethodParameters(ctx context.Context, params *types.Tuple) ([]*domain.Parameter, error) {
	parameters := make([]*domain.Parameter, params.Len())

	for i := 0; i < params.Len(); i++ {
		param := params.At(i)

		// Resolve type with generics support
		resolvedType, err := p.typeResolverPool.Get().ResolveType(ctx, param.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter type: %w", err)
		}

		// Analyze type structure for field mapping
		typeInfo, err := p.analyzeTypeStructure(ctx, resolvedType)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze type structure: %w", err)
		}

		parameters[i] = &domain.Parameter{
			Name:     param.Name(),
			Type:     resolvedType,
			TypeInfo: typeInfo,
			Position: i,
		}
	}

	return parameters, nil
}

// analyzeMethodReturns analyzes return types of a method.
func (p *ASTParser) analyzeMethodReturns(ctx context.Context, results *types.Tuple) ([]*domain.ReturnValue, error) {
	returns := make([]*domain.ReturnValue, results.Len())

	for i := 0; i < results.Len(); i++ {
		result := results.At(i)

		// Resolve type with generics support
		resolvedType, err := p.typeResolverPool.Get().ResolveType(ctx, result.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to resolve return type: %w", err)
		}

		// Check if this is an error type
		isError := p.isErrorType(resolvedType)

		// Analyze type structure for non-error types
		var typeInfo *domain.TypeInfo

		if !isError {
			info, err := p.analyzeTypeStructure(ctx, resolvedType)
			if err != nil {
				return nil, fmt.Errorf("failed to analyze return type structure: %w", err)
			}

			typeInfo = info
		}

		returns[i] = &domain.ReturnValue{
			Name:     result.Name(),
			Type:     resolvedType,
			TypeInfo: typeInfo,
			Position: i,
			IsError:  isError,
		}
	}

	return returns, nil
}

// analyzeReceiver analyzes method receiver.
func (p *ASTParser) analyzeReceiver(ctx context.Context, recv *types.Var) (*domain.Parameter, error) {
	resolvedType, err := p.typeResolverPool.Get().ResolveType(ctx, recv.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve receiver type: %w", err)
	}

	typeInfo, err := p.analyzeTypeStructure(ctx, resolvedType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze receiver type structure: %w", err)
	}

	return &domain.Parameter{
		Name:     recv.Name(),
		Type:     resolvedType,
		TypeInfo: typeInfo,
		Position: -1, // Receiver has special position
	}, nil
}

// createFieldMappings creates field mappings between source and destination types.
func (p *ASTParser) createFieldMappings(_ context.Context, params []*domain.Parameter, returns []*domain.ReturnValue, options *domain.MethodOptions) ([]*domain.FieldMapping, error) {
	var mappings []*domain.FieldMapping

	// For now, we'll create basic mappings between the first parameter and first return
	// In a full implementation, this would be much more sophisticated
	if len(params) > 0 && len(returns) > 0 && !returns[0].IsError {
		sourceParam := params[0]
		destReturn := returns[0]

		if sourceParam.TypeInfo != nil && destReturn.TypeInfo != nil {
			fieldMappings, err := p.matchFields(sourceParam.TypeInfo, destReturn.TypeInfo, options)
			if err != nil {
				return nil, err
			}

			mappings = append(mappings, fieldMappings...)
		}
	}

	return mappings, nil
}

// matchFields matches fields between source and destination types.
func (p *ASTParser) matchFields(source, dest *domain.TypeInfo, options *domain.MethodOptions) ([]*domain.FieldMapping, error) {
	var mappings []*domain.FieldMapping

	// Simple field matching based on name
	for i, sourceField := range source.Fields {
		for j, destField := range dest.Fields {
			if p.fieldsMatch(sourceField, destField, options) {
				// Create field specs
				sourceSpec, err := domain.NewFieldSpec([]string{sourceField.Name}, sourceField.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to create source field spec: %w", err)
				}

				destSpec, err := domain.NewFieldSpec([]string{destField.Name}, destField.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to create dest field spec: %w", err)
				}

				// Create mapping with direct assignment strategy
				strategy := &domain.DirectAssignmentStrategy{}
				mappingID := fmt.Sprintf("field_%d_%d", i, j)

				mapping, err := domain.NewFieldMapping(mappingID, sourceSpec, destSpec, strategy)
				if err != nil {
					return nil, fmt.Errorf("failed to create field mapping: %w", err)
				}

				mappings = append(mappings, mapping)

				break
			}
		}
	}

	return mappings, nil
}

// fieldsMatch determines if two fields match based on options.
func (p *ASTParser) fieldsMatch(source, dest *domain.Field, options *domain.MethodOptions) bool {
	switch options.MatchRule {
	case domain.MatchByName:
		if options.CaseSensitive {
			return source.Name == dest.Name
		}

		return strings.EqualFold(source.Name, dest.Name)
	case domain.MatchByType:
		return source.Type.String() == dest.Type.String()
	case domain.MatchByTag:
		// Tag-based matching would require tag analysis
		return false
	default:
		return false
	}
}

// getMethodDocComment retrieves the documentation comment for a method.
func (p *ASTParser) getMethodDocComment(file *ast.File, methodObj types.Object) *ast.CommentGroup {
	// This is a simplified implementation
	// In practice, you'd need to find the method declaration in the AST
	return nil
}

// determineErrorHandling function removed - was unused

// isErrorType checks if a type is the built-in error type.
func (p *ASTParser) isErrorType(t domain.Type) bool {
	return t.Name() == "error" && t.Kind() == domain.TypeKindInterface
}

// Helper functions

func (p *ASTParser) copyStringMap(original map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range original {
		copy[k] = v
	}

	return copy
}

// processGenericMethod processes a method within a generic interface context.
// This method handles the interface-level type parameters and delegates to processMethod.
func (p *ASTParser) processGenericMethod(ctx context.Context, pkg *packages.Package, file *ast.File, methodObj types.Object, interfaceTypeParams []domain.TypeParam, options *domain.InterfaceOptions) (*domain.Method, error) {
	// Extract the method signature
	signature, ok := methodObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrExpectedMethodSignature, methodObj.Type())
	}

	// Validate method signature with generic context
	if err := p.validateGenericMethodSignature(signature, methodObj, interfaceTypeParams); err != nil {
		return nil, fmt.Errorf("invalid generic method signature: %w", err)
	}

	// Extract method annotations
	annotations := p.extractMethodAnnotations(file, methodObj)

	// Parse method-specific options from interface options
	methodOpts, err := p.parseMethodOptions(annotations, options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse method options: %w", err)
	}

	// Analyze method parameters with generic context
	parameters, err := p.analyzeGenericMethodParameters(ctx, signature.Params(), interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze parameters: %w", err)
	}

	// Analyze return types with generic context
	returns, err := p.analyzeGenericMethodReturns(ctx, signature.Results(), interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze return types: %w", err)
	}

	// Create field mappings with generic type awareness
	fieldMappings, err := p.createGenericFieldMappings(ctx, parameters, returns, methodOpts, interfaceTypeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create field mappings: %w", err)
	}

	// Create the domain method using the correct constructor
	method, err := domain.NewMethod(methodObj.Name(), parameters[0].Type, returns[0].Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create method: %w", err)
	}

	// Set method parameters and returns
	method.SetSourceParams(parameters)
	method.SetDestinationReturns(returns)

	// Add field mappings to the method
	for _, mapping := range fieldMappings {
		if err := method.AddMapping(mapping); err != nil {
			return nil, fmt.Errorf("failed to add mapping: %w", err)
		}
	}

	p.logger.Debug("processed generic method",
		zap.String("name", methodObj.Name()),
		zap.Int("parameters", len(parameters)),
		zap.Int("returns", len(returns)),
		zap.Int("field_mappings", len(fieldMappings)),
		zap.Int("interface_type_params", len(interfaceTypeParams)))

	return method, nil
}

// validateGenericMethodSignature validates a method signature within a generic interface.
func (p *ASTParser) validateGenericMethodSignature(signature *types.Signature, methodObj types.Object, interfaceTypeParams []domain.TypeParam) error {
	// First perform basic signature validation
	if err := p.validateMethodSignature(signature, methodObj); err != nil {
		return err
	}

	// Additional validation for generic context
	// Check that any type parameters used in the signature are valid
	if err := p.validateTypeParametersInSignature(signature, interfaceTypeParams); err != nil {
		return fmt.Errorf("invalid type parameters in method %s: %w", methodObj.Name(), err)
	}

	return nil
}

// validateTypeParametersInSignature validates that type parameters in method signature are defined.
func (p *ASTParser) validateTypeParametersInSignature(signature *types.Signature, interfaceTypeParams []domain.TypeParam) error {
	// Check parameters
	for i := 0; i < signature.Params().Len(); i++ {
		param := signature.Params().At(i)
		if err := p.validateTypeParameterUsage(param.Type(), interfaceTypeParams); err != nil {
			return fmt.Errorf("parameter %d: %w", i, err)
		}
	}

	// Check return types
	for i := 0; i < signature.Results().Len(); i++ {
		result := signature.Results().At(i)
		if err := p.validateTypeParameterUsage(result.Type(), interfaceTypeParams); err != nil {
			return fmt.Errorf("return %d: %w", i, err)
		}
	}

	return nil
}

// validateTypeParameterUsage validates that a type uses only defined type parameters.
func (p *ASTParser) validateTypeParameterUsage(goType types.Type, interfaceTypeParams []domain.TypeParam) error {
	// Check if this is a type parameter
	if typeParam, ok := goType.(*types.TypeParam); ok {
		// Verify it's defined in the interface
		typeParamName := typeParam.String()
		for _, interfaceParam := range interfaceTypeParams {
			if interfaceParam.Name == typeParamName {
				return nil // Found, it's valid
			}
		}
		return fmt.Errorf("undefined type parameter: %s", typeParamName)
	}

	// Recursively check composite types
	switch t := goType.(type) {
	case *types.Slice:
		return p.validateTypeParameterUsage(t.Elem(), interfaceTypeParams)
	case *types.Pointer:
		return p.validateTypeParameterUsage(t.Elem(), interfaceTypeParams)
	case *types.Map:
		if err := p.validateTypeParameterUsage(t.Key(), interfaceTypeParams); err != nil {
			return err
		}
		return p.validateTypeParameterUsage(t.Elem(), interfaceTypeParams)
	}

	return nil // Other types are valid
}

func (p *ASTParser) applyInterfaceAnnotationToMethod(options *domain.MethodOptions, annotation *Annotation) error {
	// Convert method options to interface options temporarily for reuse
	interfaceOpts := &domain.InterfaceOptions{
		Style:               options.Style,
		MatchRule:           options.MatchRule,
		CaseSensitive:       options.CaseSensitive,
		UseGetter:           options.UseGetter,
		UseStringer:         options.UseStringer,
		UseTypecast:         options.UseTypecast,
		AllowReverse:        options.AllowReverse,
		SkipFields:          options.SkipFields,
		FieldMappings:       options.FieldMappings,
		TypeConverters:      options.TypeConverters,
		LiteralAssignments:  options.LiteralAssignments,
		PreprocessFunction:  options.PreprocessFunction,
		PostprocessFunction: options.PostprocessFunction,
	}

	err := p.applyInterfaceAnnotation(interfaceOpts, annotation)
	if err != nil {
		return err
	}

	// Copy back to method options
	options.Style = interfaceOpts.Style
	options.MatchRule = interfaceOpts.MatchRule
	options.CaseSensitive = interfaceOpts.CaseSensitive
	options.UseGetter = interfaceOpts.UseGetter
	options.UseStringer = interfaceOpts.UseStringer
	options.UseTypecast = interfaceOpts.UseTypecast
	options.AllowReverse = interfaceOpts.AllowReverse
	options.SkipFields = interfaceOpts.SkipFields
	options.FieldMappings = interfaceOpts.FieldMappings
	options.TypeConverters = interfaceOpts.TypeConverters
	options.LiteralAssignments = interfaceOpts.LiteralAssignments
	options.PreprocessFunction = interfaceOpts.PreprocessFunction
	options.PostprocessFunction = interfaceOpts.PostprocessFunction

	return nil
}

// isGoErrorType checks if a Go types.Type is the built-in error interface.
func isGoErrorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name() == "error" && named.Obj().Pkg() == nil
	}

	return false
}

// ===========================================
// TASK-006: Generic Method Processing Functions
// ===========================================

// convertGoTypeParamToDomainTypeParam converts a Go types.TypeParam to domain.TypeParam.
func (p *ASTParser) convertGoTypeParamToDomainTypeParam(ctx context.Context, goTypeParam *types.TypeParam) (*domain.TypeParam, error) {
	constraint := goTypeParam.Constraint()
	name := goTypeParam.String()
	index := goTypeParam.Index()

	// Parse the constraint to determine its type
	if constraint != nil {
		// Use constraint parser if available, or fallback to basic parsing
		if constraintParser := p.getConstraintParser(); constraintParser != nil {
			parsedConstraint, err := constraintParser.ParseConstraint(ctx, constraint)
			if err != nil {
				p.logger.Warn("failed to parse constraint, using basic parsing",
					zap.String("type_param", name),
					zap.Error(err))
				return p.parseBasicConstraint(name, constraint, index)
			}
			return p.convertParsedConstraintToDomainTypeParam(name, index, parsedConstraint)
		}
	}

	// Fallback to basic constraint parsing
	return p.parseBasicConstraint(name, constraint, index)
}

// parseBasicConstraint handles basic constraint parsing when advanced parser is not available.
func (p *ASTParser) parseBasicConstraint(name string, constraint types.Type, index int) (*domain.TypeParam, error) {
	if constraint == nil {
		// No constraint specified, default to 'any'
		return domain.NewAnyTypeParam(name, index), nil
	}

	// Check for basic constraint types
	constraintStr := constraint.String()
	switch constraintStr {
	case "any", "interface{}":
		return domain.NewAnyTypeParam(name, index), nil
	case "comparable":
		return domain.NewComparableTypeParam(name, index), nil
	default:
		// For other constraints, resolve the constraint type and create a basic type param
		constraintType, err := p.typeResolverPool.Get().ResolveType(context.Background(), constraint)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve constraint type: %w", err)
		}
		return domain.NewTypeParam(name, constraintType, index), nil
	}
}

// getConstraintParser returns the constraint parser if available.
func (p *ASTParser) getConstraintParser() *ConstraintParser {
	// This would be injected or configured based on available parsers
	// For now, return nil to use fallback parsing
	return nil
}

// convertParsedConstraintToDomainTypeParam converts a parsed constraint to domain.TypeParam.
func (p *ASTParser) convertParsedConstraintToDomainTypeParam(name string, index int, parsed *ParsedConstraint) (*domain.TypeParam, error) {
	if parsed.IsAny {
		return domain.NewAnyTypeParam(name, index), nil
	}
	if parsed.IsComparable {
		return domain.NewComparableTypeParam(name, index), nil
	}
	if len(parsed.UnionTypes) > 0 {
		return domain.NewUnionTypeParam(name, parsed.UnionTypes, index), nil
	}
	if parsed.Underlying != nil {
		return domain.NewUnderlyingTypeParam(name, parsed.Underlying, index), nil
	}
	if parsed.Type != nil {
		return domain.NewTypeParam(name, parsed.Type, index), nil
	}

	// Default to any constraint
	return domain.NewAnyTypeParam(name, index), nil
}

// analyzeGenericMethodParameters analyzes method parameters with generic type context.
func (p *ASTParser) analyzeGenericMethodParameters(ctx context.Context, params *types.Tuple, interfaceTypeParams []domain.TypeParam) ([]*domain.Parameter, error) {
	parameters := make([]*domain.Parameter, params.Len())

	for i := 0; i < params.Len(); i++ {
		param := params.At(i)

		// Check if parameter type contains generic type parameters
		paramType := param.Type()
		// Note: Go types.Tuple doesn't have Variadic() method, variadic handling needs signature-level check
		isVariadic := false // TODO: Implement proper variadic detection from method signature

		// Handle variadic parameters
		if isVariadic {
			if sliceType, ok := paramType.(*types.Slice); ok {
				paramType = sliceType.Elem()
			}
		}

		// Resolve type with generics support
		resolvedType, err := p.resolveGenericType(ctx, paramType, interfaceTypeParams)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter type: %w", err)
		}

		// Restore variadic slice type if needed
		if isVariadic {
			resolvedType = domain.NewSliceType(resolvedType, resolvedType.Package())
		}

		// Analyze type structure for field mapping
		typeInfo, err := p.analyzeTypeStructure(ctx, resolvedType)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze type structure: %w", err)
		}

		parameters[i] = &domain.Parameter{
			Name:     param.Name(),
			Type:     resolvedType,
			TypeInfo: typeInfo,
			Position: i,
		}
	}

	return parameters, nil
}

// analyzeGenericMethodReturns analyzes return types with generic type context.
func (p *ASTParser) analyzeGenericMethodReturns(ctx context.Context, results *types.Tuple, interfaceTypeParams []domain.TypeParam) ([]*domain.ReturnValue, error) {
	returns := make([]*domain.ReturnValue, results.Len())

	for i := 0; i < results.Len(); i++ {
		result := results.At(i)

		// Resolve type with generics support
		resolvedType, err := p.resolveGenericType(ctx, result.Type(), interfaceTypeParams)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve return type: %w", err)
		}

		// Check if this is an error type
		isError := p.isErrorType(resolvedType)

		// Analyze type structure for non-error types
		var typeInfo *domain.TypeInfo

		if !isError {
			info, err := p.analyzeTypeStructure(ctx, resolvedType)
			if err != nil {
				return nil, fmt.Errorf("failed to analyze return type structure: %w", err)
			}

			typeInfo = info
		}

		returns[i] = &domain.ReturnValue{
			Name:     result.Name(),
			Type:     resolvedType,
			TypeInfo: typeInfo,
			Position: i,
			IsError:  isError,
		}
	}

	return returns, nil
}

// resolveGenericType resolves a types.Type with awareness of generic type parameters.
func (p *ASTParser) resolveGenericType(ctx context.Context, goType types.Type, interfaceTypeParams []domain.TypeParam) (domain.Type, error) {
	// Check if this is a type parameter
	if typeParam, ok := goType.(*types.TypeParam); ok {
		// Find matching interface type parameter
		for _, interfaceParam := range interfaceTypeParams {
			if interfaceParam.Name == typeParam.String() {
				// Return a generic type representing this type parameter
				return domain.NewGenericType(interfaceParam.Name, interfaceParam.Constraint, interfaceParam.Index, ""), nil
			}
		}
		// If not found in interface params, treat as a regular type
	}

	// Check for composite types containing type parameters
	switch t := goType.(type) {
	case *types.Slice:
		elemType, err := p.resolveGenericType(ctx, t.Elem(), interfaceTypeParams)
		if err != nil {
			return nil, err
		}
		return domain.NewSliceType(elemType, ""), nil

	case *types.Pointer:
		elemType, err := p.resolveGenericType(ctx, t.Elem(), interfaceTypeParams)
		if err != nil {
			return nil, err
		}
		return domain.NewPointerType(elemType, ""), nil

	case *types.Map:
		keyType, err := p.resolveGenericType(ctx, t.Key(), interfaceTypeParams)
		if err != nil {
			return nil, err
		}
		valueType, err := p.resolveGenericType(ctx, t.Elem(), interfaceTypeParams)
		if err != nil {
			return nil, err
		}
		return domain.NewMapType(keyType, valueType), nil

	default:
		// For other types, use the regular type resolver
		return p.typeResolverPool.Get().ResolveType(ctx, goType)
	}
}

// createGenericFieldMappings creates field mappings with generic type awareness.
func (p *ASTParser) createGenericFieldMappings(ctx context.Context, params []*domain.Parameter, returns []*domain.ReturnValue, options *domain.MethodOptions, interfaceTypeParams []domain.TypeParam) ([]*domain.FieldMapping, error) {
	var mappings []*domain.FieldMapping

	// For now, we'll create basic mappings between the first parameter and first return
	// In a full implementation, this would be much more sophisticated and handle type substitution
	if len(params) > 0 && len(returns) > 0 && !returns[0].IsError {
		sourceParam := params[0]
		destReturn := returns[0]

		if sourceParam.TypeInfo != nil && destReturn.TypeInfo != nil {
			// Enhanced field mapping that considers generic types
			fieldMappings, err := p.matchGenericFields(sourceParam.TypeInfo, destReturn.TypeInfo, options, interfaceTypeParams)
			if err != nil {
				return nil, err
			}

			mappings = append(mappings, fieldMappings...)
		}
	}

	return mappings, nil
}

// matchGenericFields matches fields between source and destination types with generic awareness.
func (p *ASTParser) matchGenericFields(source, dest *domain.TypeInfo, options *domain.MethodOptions, interfaceTypeParams []domain.TypeParam) ([]*domain.FieldMapping, error) {
	var mappings []*domain.FieldMapping

	// Simple field matching based on name, with generic type compatibility checking
	for i, sourceField := range source.Fields {
		for j, destField := range dest.Fields {
			if p.genericFieldsMatch(sourceField, destField, options, interfaceTypeParams) {
				// Create field specs
				sourceSpec, err := domain.NewFieldSpec([]string{sourceField.Name}, sourceField.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to create source field spec: %w", err)
				}

				destSpec, err := domain.NewFieldSpec([]string{destField.Name}, destField.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to create dest field spec: %w", err)
				}

				// Create mapping with appropriate strategy for generic types
				strategy := p.selectGenericMappingStrategy(sourceField.Type, destField.Type, interfaceTypeParams)
				mappingID := fmt.Sprintf("generic_field_%d_%d", i, j)

				mapping, err := domain.NewFieldMapping(mappingID, sourceSpec, destSpec, strategy)
				if err != nil {
					return nil, fmt.Errorf("failed to create field mapping: %w", err)
				}

				mappings = append(mappings, mapping)

				break
			}
		}
	}

	return mappings, nil
}

// genericFieldsMatch determines if two fields match considering generic types.
func (p *ASTParser) genericFieldsMatch(source, dest *domain.Field, options *domain.MethodOptions, interfaceTypeParams []domain.TypeParam) bool {
	// First check basic field matching
	if !p.fieldsMatch(source, dest, options) {
		return false
	}

	// Additional generic type compatibility checking
	return p.areGenericTypesCompatible(source.Type, dest.Type, interfaceTypeParams)
}

// areGenericTypesCompatible checks if two types are compatible in a generic context.
func (p *ASTParser) areGenericTypesCompatible(sourceType, destType domain.Type, interfaceTypeParams []domain.TypeParam) bool {
	// If both types are generic, they should be compatible through type parameters
	if sourceType.Generic() && destType.Generic() {
		// Check if they refer to the same type parameter
		return sourceType.String() == destType.String()
	}

	// If one is generic and one is concrete, check constraint satisfaction
	if sourceType.Generic() && !destType.Generic() {
		return p.typeConstraintSatisfied(sourceType, destType, interfaceTypeParams)
	}

	if !sourceType.Generic() && destType.Generic() {
		return p.typeConstraintSatisfied(destType, sourceType, interfaceTypeParams)
	}

	// Both are concrete types, use regular assignability
	return sourceType.AssignableTo(destType)
}

// typeConstraintSatisfied checks if a concrete type satisfies a generic type's constraints.
func (p *ASTParser) typeConstraintSatisfied(genericType, concreteType domain.Type, interfaceTypeParams []domain.TypeParam) bool {
	// Find the type parameter for the generic type
	for _, param := range interfaceTypeParams {
		if param.Name == genericType.Name() {
			return param.SatisfiesConstraint(concreteType)
		}
	}

	// If type parameter not found, default to allowing the mapping
	return true
}

// selectGenericMappingStrategy selects the appropriate mapping strategy for generic types.
func (p *ASTParser) selectGenericMappingStrategy(sourceType, destType domain.Type, interfaceTypeParams []domain.TypeParam) domain.MappingStrategy {
	// For generic types, we typically use direct assignment with type substitution
	if sourceType.Generic() || destType.Generic() {
		return &domain.GenericDirectAssignmentStrategy{
			InterfaceTypeParams: interfaceTypeParams,
		}
	}

	// For concrete types, use the standard direct assignment
	return &domain.DirectAssignmentStrategy{}
}
