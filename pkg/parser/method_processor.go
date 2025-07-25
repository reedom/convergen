package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/reedom/convergen/v8/pkg/domain"
	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"
)

// MethodProcessor handles analysis and processing of individual methods
type MethodProcessor struct {
	parser       *ASTParser
	typeResolver *TypeResolver
	logger       *zap.Logger
}

// NewMethodProcessor creates a new method processor
func NewMethodProcessor(parser *ASTParser, typeResolver *TypeResolver, logger *zap.Logger) *MethodProcessor {
	return &MethodProcessor{
		parser:       parser,
		typeResolver: typeResolver,
		logger:       logger,
	}
}

// processMethod analyzes a single method and converts it to domain.Method
func (p *ASTParser) processMethod(ctx context.Context, pkg *packages.Package, file *ast.File, methodObj types.Object, interfaceOpts *domain.InterfaceOptions) (*domain.Method, error) {
	signature, ok := methodObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("expected method signature, got %T", methodObj.Type())
	}

	// Validate method signature
	if err := p.validateMethodSignature(signature, methodObj); err != nil {
		return nil, fmt.Errorf("invalid method signature: %w", err)
	}

	// Extract method annotations
	annotations, err := p.extractMethodAnnotations(file, methodObj)
	if err != nil {
		return nil, fmt.Errorf("failed to extract method annotations: %w", err)
	}

	// Parse method-specific options
	methodOpts, err := p.parseMethodOptions(annotations, interfaceOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse method options: %w", err)
	}

	// Analyze method parameters
	parameters, err := p.analyzeMethodParameters(ctx, signature.Params())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze parameters: %w", err)
	}

	// Analyze return types
	returns, err := p.analyzeMethodReturns(ctx, signature.Results())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze return types: %w", err)
	}

	// Analyze receiver if present
	var receiver *domain.Parameter
	if signature.Recv() != nil {
		recv, err := p.analyzeReceiver(ctx, signature.Recv())
		if err != nil {
			return nil, fmt.Errorf("failed to analyze receiver: %w", err)
		}
		receiver = recv
	}

	// Create field mappings
	fieldMappings, err := p.createFieldMappings(ctx, parameters, returns, methodOpts)
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
		zap.Int("field_mappings", len(fieldMappings)))

	return method, nil
}

// validateMethodSignature ensures the method signature is valid for conversion
func (p *ASTParser) validateMethodSignature(signature *types.Signature, methodObj types.Object) error {
	if signature.Params().Len() == 0 {
		return fmt.Errorf("method %s must have at least one parameter", methodObj.Name())
	}

	if signature.Results().Len() == 0 {
		return fmt.Errorf("method %s must have at least one return value", methodObj.Name())
	}

	// Check if last return is error (optional)
	if signature.Results().Len() > 1 {
		lastReturn := signature.Results().At(signature.Results().Len() - 1)
		if !p.isErrorType(lastReturn.Type()) {
			return fmt.Errorf("method %s: if multiple returns, last must be error type", methodObj.Name())
		}
	}

	return nil
}

// extractMethodAnnotations extracts annotations from method comments
func (p *ASTParser) extractMethodAnnotations(file *ast.File, methodObj types.Object) ([]*Annotation, error) {
	docComment := p.getMethodDocComment(file, methodObj)
	if docComment == nil {
		return nil, nil
	}

	var annotations []*Annotation
	for _, comment := range docComment.List {
		if annotation := p.parseAnnotation(comment); annotation != nil {
			annotations = append(annotations, annotation)
		}
	}

	return annotations, nil
}

// parseMethodOptions converts method annotations to options, inheriting from interface options
func (p *ASTParser) parseMethodOptions(annotations []*Annotation, interfaceOpts *domain.InterfaceOptions) (*domain.MethodOptions, error) {
	// Start with interface options as base
	options := &domain.MethodOptions{
		Style:                interfaceOpts.Style,
		MatchRule:            interfaceOpts.MatchRule,
		CaseSensitive:        interfaceOpts.CaseSensitive,
		UseGetter:            interfaceOpts.UseGetter,
		UseStringer:          interfaceOpts.UseStringer,
		UseTypecast:          interfaceOpts.UseTypecast,
		AllowReverse:         interfaceOpts.AllowReverse,
		SkipFields:           append([]string{}, interfaceOpts.SkipFields...),
		FieldMappings:        p.copyStringMap(interfaceOpts.FieldMappings),
		TypeConverters:       p.copyStringMap(interfaceOpts.TypeConverters),
		LiteralAssignments:   p.copyStringMap(interfaceOpts.LiteralAssignments),
		PreprocessFunction:   interfaceOpts.PreprocessFunction,
		PostprocessFunction:  interfaceOpts.PostprocessFunction,
		CustomValidation:     "",
		ConcurrencyLevel:     1,
		TimeoutDuration:      0,
	}

	// Apply method-specific annotations
	for _, annotation := range annotations {
		if err := p.applyMethodAnnotation(options, annotation); err != nil {
			return nil, fmt.Errorf("failed to apply annotation %s: %w", annotation.Type, err)
		}
	}

	return options, nil
}

// applyMethodAnnotation applies a method-level annotation to options
func (p *ASTParser) applyMethodAnnotation(options *domain.MethodOptions, annotation *Annotation) error {
	switch annotation.Type {
	case "style", "match", "case", "case:off", "getter", "getter:off",
		 "stringer", "stringer:off", "typecast", "typecast:off", "reverse",
		 "skip", "map", "conv", "literal", "preprocess", "postprocess":
		// These are handled the same way as interface annotations
		return p.applyInterfaceAnnotationToMethod(options, annotation)

	case "validation":
		if len(annotation.Args) == 0 {
			return fmt.Errorf("validation annotation requires a function name")
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
			return fmt.Errorf("timeout annotation requires a duration")
		}
		duration, err := parseDurationArg(annotation.Args[0])
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		options.TimeoutDuration = duration

	default:
		p.logger.Warn("unknown method annotation",
			zap.String("type", annotation.Type),
			zap.String("position", p.parser.fileSet.Position(annotation.Position).String()))
	}

	return nil
}

// analyzeMethodParameters analyzes all parameters of a method
func (p *ASTParser) analyzeMethodParameters(ctx context.Context, params *types.Tuple) ([]*domain.Parameter, error) {
	parameters := make([]*domain.Parameter, params.Len())

	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		
		// Resolve type with generics support
		resolvedType, err := p.typeResolver.ResolveType(ctx, param.Type())
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

// analyzeMethodReturns analyzes return types of a method
func (p *ASTParser) analyzeMethodReturns(ctx context.Context, results *types.Tuple) ([]*domain.ReturnValue, error) {
	returns := make([]*domain.ReturnValue, results.Len())

	for i := 0; i < results.Len(); i++ {
		result := results.At(i)
		
		// Resolve type with generics support
		resolvedType, err := p.typeResolver.ResolveType(ctx, result.Type())
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

// analyzeReceiver analyzes method receiver
func (p *ASTParser) analyzeReceiver(ctx context.Context, recv *types.Var) (*domain.Parameter, error) {
	resolvedType, err := p.typeResolver.ResolveType(ctx, recv.Type())
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

// createFieldMappings creates field mappings between source and destination types
func (p *ASTParser) createFieldMappings(ctx context.Context, params []*domain.Parameter, returns []*domain.ReturnValue, options *domain.MethodOptions) ([]*domain.FieldMapping, error) {
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

// matchFields matches fields between source and destination types
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

// fieldsMatch determines if two fields match based on options
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

// getMethodDocComment retrieves the documentation comment for a method
func (p *ASTParser) getMethodDocComment(file *ast.File, methodObj types.Object) *ast.CommentGroup {
	// This is a simplified implementation
	// In practice, you'd need to find the method declaration in the AST
	return nil
}

// determineErrorHandling determines the error handling strategy for the method
func (p *ASTParser) determineErrorHandling(returns []*domain.ReturnValue) domain.ErrorHandlingMethod {
	for _, ret := range returns {
		if ret.IsError {
			return domain.ErrorHandlingReturn
		}
	}
	return domain.ErrorHandlingNone
}

// isErrorType checks if a type is the built-in error type
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

func (p *ASTParser) applyInterfaceAnnotationToMethod(options *domain.MethodOptions, annotation *Annotation) error {
	// Convert method options to interface options temporarily for reuse
	interfaceOpts := &domain.InterfaceOptions{
		Style:                options.Style,
		MatchRule:            options.MatchRule,
		CaseSensitive:        options.CaseSensitive,
		UseGetter:            options.UseGetter,
		UseStringer:          options.UseStringer,
		UseTypecast:          options.UseTypecast,
		AllowReverse:         options.AllowReverse,
		SkipFields:           options.SkipFields,
		FieldMappings:        options.FieldMappings,
		TypeConverters:       options.TypeConverters,
		LiteralAssignments:   options.LiteralAssignments,
		PreprocessFunction:   options.PreprocessFunction,
		PostprocessFunction:  options.PostprocessFunction,
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