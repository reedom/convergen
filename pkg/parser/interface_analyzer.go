package parser

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	gonanoid "github.com/matoous/go-nanoid"
	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Static errors for err113 compliance.
var (
	ErrStyleAnnotationRequiresArgument       = errors.New("style annotation requires an argument")
	ErrMatchAnnotationRequiresArgument       = errors.New("match annotation requires an argument")
	ErrRecvAnnotationRequiresReceiverName    = errors.New("recv annotation requires a receiver name")
	ErrInvalidReceiverName                   = errors.New("invalid receiver name")
	ErrSkipAnnotationRequiresFieldPattern    = errors.New("skip annotation requires a field pattern")
	ErrMapAnnotationRequiresArguments        = errors.New("map annotation requires source and destination arguments")
	ErrConvAnnotationRequiresArguments       = errors.New("conv annotation requires converter and type arguments")
	ErrLiteralAnnotationRequiresArguments    = errors.New("literal annotation requires field and value arguments")
	ErrPreprocessAnnotationRequiresFunction  = errors.New("preprocess annotation requires a function name")
	ErrPostprocessAnnotationRequiresFunction = errors.New("postprocess annotation requires a function name")
	ErrUnknownStyle                          = errors.New("unknown style")
	ErrUnknownMatchRule                      = errors.New("unknown match rule")
)

// InstantiatedInterface represents a concrete instantiation of a generic interface.
// Used for caching concrete types when generic interfaces are instantiated with specific types.
type InstantiatedInterface struct {
	TypeArgs     map[string]domain.Type `json:"type_args"`               // Map of type parameter names to concrete types
	Methods      []types.Object         `json:"methods"`                 // Instantiated method signatures
	ResolvedType *types.Interface       `json:"resolved_type,omitempty"` // Fully resolved interface type
	CreatedAt    string                 `json:"created_at"`              // Timestamp of creation
	Validated    bool                   `json:"validated"`               // Whether this instantiation has been validated
}

// NewInstantiatedInterface creates a new instantiated interface with proper validation.
func NewInstantiatedInterface(typeArgs map[string]domain.Type, methods []types.Object) *InstantiatedInterface {
	return &InstantiatedInterface{
		TypeArgs:  typeArgs,
		Methods:   methods,
		CreatedAt: "", // Will be set by time package in real usage
		Validated: false,
	}
}

// NewInterfaceInfo creates a new InterfaceInfo with proper initialization following domain constructor patterns.
func NewInterfaceInfo(
	obj types.Object,
	iface *types.Interface,
	methods []types.Object,
	options *domain.InterfaceOptions,
	annotations []*Annotation,
	marker string,
	position token.Pos,
	typeParams []domain.TypeParam,
) *InterfaceInfo {
	// Determine if this interface is generic
	isGeneric := 0 < len(typeParams)

	return &InterfaceInfo{
		Object:         obj,
		Interface:      iface,
		Methods:        methods,
		Options:        options,
		Annotations:    annotations,
		Marker:         marker,
		Position:       position,
		TypeParams:     typeParams,
		IsGeneric:      isGeneric,
		Instantiations: make(map[string]*InstantiatedInterface),
	}
}

// InterfaceInfo contains comprehensive information about a convergen interface.
type InterfaceInfo struct {
	Object      types.Object             `json:"object"`
	Interface   *types.Interface         `json:"interface"`
	Methods     []types.Object           `json:"methods"`
	Options     *domain.InterfaceOptions `json:"options"`
	Annotations []*Annotation            `json:"annotations"`
	Marker      string                   `json:"marker"`
	Position    token.Pos                `json:"position"`
	TypeParams  []domain.TypeParam       `json:"type_params"`

	// New fields for generic support
	IsGeneric      bool                              `json:"is_generic"`
	Instantiations map[string]*InstantiatedInterface `json:"instantiations,omitempty"`
}

// Annotation represents a parsed annotation from interface or method comments.
type Annotation struct {
	Type     string
	Args     []string
	Position token.Pos
	Raw      string
}

// Interface name constants.
const (
	DefaultInterfaceName = "Convergen"
)

// Annotation regular expressions for interface analyzer.
var (
	reConvergenInterface = regexp.MustCompile(`^\s*//\s*:convergen\b`)
	reNotationInterface  = regexp.MustCompile(`^\s*//\s*:(\S+)\s*(.*)$`)
)

// isConvergenInterface checks if an interface is a convergen target.
func (p *ASTParser) isConvergenInterface(file *ast.File, obj types.Object) bool {
	// Check by name first
	if obj.Name() == DefaultInterfaceName {
		return true
	}

	// Check for :convergen annotation in comments
	docComment := p.getDocComment(file, obj)
	if docComment == nil {
		return false
	}

	for _, comment := range docComment.List {
		if reConvergenInterface.MatchString(comment.Text) {
			return true
		}
	}

	return false
}

// analyzeInterface performs comprehensive analysis of a convergen interface.
func (p *ASTParser) analyzeInterface(ctx context.Context, _ *packages.Package, file *ast.File, obj types.Object, iface *types.Interface) (*InterfaceInfo, error) {
	// Generate unique marker for this interface
	marker, err := gonanoid.Nanoid()
	if err != nil {
		return nil, fmt.Errorf("failed to generate marker: %w", err)
	}

	// Extract and parse annotations
	annotations := p.extractInterfaceAnnotations(file, obj)

	// Parse interface-level options
	options, err := p.parseInterfaceOptions(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interface options: %w", err)
	}

	// Extract type parameters from generic interface declarations
	typeParams, err := p.extractInterfaceTypeParams(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to extract interface type parameters: %w", err)
	}

	// Get all methods from the interface
	methods := make([]types.Object, 0, iface.NumMethods())

	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		if method.Exported() {
			methods = append(methods, method)
		}
	}

	// Create interface info using constructor
	interfaceInfo := NewInterfaceInfo(
		obj,
		iface,
		methods,
		options,
		annotations,
		marker,
		obj.Pos(),
		typeParams,
	)

	p.logger.Debug("analyzed convergen interface",
		zap.String("name", obj.Name()),
		zap.Int("methods", len(methods)),
		zap.Int("annotations", len(annotations)),
		zap.Int("type_params", len(typeParams)),
		zap.String("marker", marker))

	return interfaceInfo, nil
}

// Helper and validation methods for InterfaceInfo

// HasInstantiation checks if the interface has a cached instantiation for the given type signature.
func (info *InterfaceInfo) HasInstantiation(typeSignature string) bool {
	if info.Instantiations == nil {
		return false
	}
	_, exists := info.Instantiations[typeSignature]
	return exists
}

// GetInstantiation retrieves a cached instantiation by type signature.
func (info *InterfaceInfo) GetInstantiation(typeSignature string) (*InstantiatedInterface, bool) {
	if info.Instantiations == nil {
		return nil, false
	}
	instantiation, exists := info.Instantiations[typeSignature]
	return instantiation, exists
}

// AddInstantiation adds a new instantiation to the cache with proper validation.
func (info *InterfaceInfo) AddInstantiation(typeSignature string, instantiation *InstantiatedInterface) error {
	if info.Instantiations == nil {
		info.Instantiations = make(map[string]*InstantiatedInterface)
	}

	if instantiation == nil {
		return fmt.Errorf("instantiation cannot be nil")
	}

	// Validate that this is actually a generic interface
	if !info.IsGeneric {
		return fmt.Errorf("cannot add instantiation to non-generic interface %s", info.Object.Name())
	}

	// Validate type argument count matches type parameter count
	if len(instantiation.TypeArgs) != len(info.TypeParams) {
		return fmt.Errorf("type argument count (%d) does not match type parameter count (%d) for interface %s",
			len(instantiation.TypeArgs), len(info.TypeParams), info.Object.Name())
	}

	info.Instantiations[typeSignature] = instantiation
	return nil
}

// ValidateGenericConsistency validates that the interface's generic configuration is consistent.
func (info *InterfaceInfo) ValidateGenericConsistency() error {
	// Check IsGeneric flag consistency
	hasTypeParams := 0 < len(info.TypeParams)
	if info.IsGeneric != hasTypeParams {
		return fmt.Errorf("IsGeneric flag (%v) inconsistent with TypeParams length (%d) for interface %s",
			info.IsGeneric, len(info.TypeParams), info.Object.Name())
	}

	// Validate type parameters
	for i, typeParam := range info.TypeParams {
		if !typeParam.IsValid() {
			return fmt.Errorf("invalid type parameter at index %d (%s) for interface %s: %s",
				i, typeParam.Name, info.Object.Name(), typeParam.GetConstraintType())
		}
	}

	// Validate instantiations
	if info.IsGeneric && info.Instantiations != nil {
		for signature, instantiation := range info.Instantiations {
			if instantiation == nil {
				return fmt.Errorf("nil instantiation found for signature %s in interface %s",
					signature, info.Object.Name())
			}

			// Validate type argument count
			if len(instantiation.TypeArgs) != len(info.TypeParams) {
				return fmt.Errorf("instantiation %s has wrong type argument count (%d vs %d) for interface %s",
					signature, len(instantiation.TypeArgs), len(info.TypeParams), info.Object.Name())
			}
		}
	} else if !info.IsGeneric && info.Instantiations != nil && 0 < len(info.Instantiations) {
		return fmt.Errorf("non-generic interface %s should not have instantiations", info.Object.Name())
	}

	return nil
}

// GetTypeParameterByName finds a type parameter by name.
func (info *InterfaceInfo) GetTypeParameterByName(name string) (*domain.TypeParam, bool) {
	for i := range info.TypeParams {
		if info.TypeParams[i].Name == name {
			return &info.TypeParams[i], true
		}
	}
	return nil, false
}

// GetTypeParameterByIndex finds a type parameter by index.
func (info *InterfaceInfo) GetTypeParameterByIndex(index int) (*domain.TypeParam, bool) {
	if index < 0 || len(info.TypeParams) <= index {
		return nil, false
	}
	return &info.TypeParams[index], true
}

// GetInstantiationCount returns the number of cached instantiations.
func (info *InterfaceInfo) GetInstantiationCount() int {
	if info.Instantiations == nil {
		return 0
	}
	return len(info.Instantiations)
}

// ClearInstantiations removes all cached instantiations.
func (info *InterfaceInfo) ClearInstantiations() {
	if info.Instantiations != nil {
		info.Instantiations = make(map[string]*InstantiatedInterface)
	}
}

// GetTypeParameterNames returns the names of all type parameters.
func (info *InterfaceInfo) GetTypeParameterNames() []string {
	names := make([]string, len(info.TypeParams))
	for i, param := range info.TypeParams {
		names[i] = param.Name
	}
	return names
}

// Helper methods for InstantiatedInterface

// IsValid checks if the instantiated interface is valid.
func (inst *InstantiatedInterface) IsValid() bool {
	if inst == nil {
		return false
	}

	// Should have type arguments
	if inst.TypeArgs == nil || len(inst.TypeArgs) == 0 {
		return false
	}

	// Type arguments should not be nil
	for name, typ := range inst.TypeArgs {
		if name == "" || typ == nil {
			return false
		}
	}

	return true
}

// GetTypeArgument retrieves a type argument by parameter name.
func (inst *InstantiatedInterface) GetTypeArgument(paramName string) (domain.Type, bool) {
	if inst.TypeArgs == nil {
		return nil, false
	}
	typ, exists := inst.TypeArgs[paramName]
	return typ, exists
}

// GetTypeArgumentNames returns the names of all type arguments.
func (inst *InstantiatedInterface) GetTypeArgumentNames() []string {
	if inst.TypeArgs == nil {
		return nil
	}

	names := make([]string, 0, len(inst.TypeArgs))
	for name := range inst.TypeArgs {
		names = append(names, name)
	}
	return names
}

// extractInterfaceAnnotations extracts all annotations from interface comments.
func (p *ASTParser) extractInterfaceAnnotations(file *ast.File, obj types.Object) []*Annotation {
	docComment := p.getDocComment(file, obj)
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

// parseAnnotation parses a single annotation from a comment.
func (p *ASTParser) parseAnnotation(comment *ast.Comment) *Annotation {
	matches := reNotationInterface.FindStringSubmatch(comment.Text)
	if len(matches) < 2 {
		return nil
	}

	annotationType := matches[1]

	argsString := ""
	if 2 < len(matches) {
		argsString = strings.TrimSpace(matches[2])
	}

	var args []string
	if argsString != "" {
		args = strings.Fields(argsString)
	}

	return &Annotation{
		Type:     annotationType,
		Args:     args,
		Position: comment.Pos(),
		Raw:      comment.Text,
	}
}

// parseInterfaceOptions converts annotations to interface options.
func (p *ASTParser) parseInterfaceOptions(annotations []*Annotation) (*domain.InterfaceOptions, error) {
	options := &domain.InterfaceOptions{
		Style:               domain.StyleCamelCase,
		MatchRule:           domain.MatchByName,
		CaseSensitive:       false,
		UseGetter:           false,
		UseStringer:         false,
		UseTypecast:         false,
		ReceiverName:        "",
		AllowReverse:        false,
		SkipFields:          make([]string, 0),
		FieldMappings:       make(map[string]string),
		TypeConverters:      make(map[string]string),
		LiteralAssignments:  make(map[string]string),
		PreprocessFunction:  "",
		PostprocessFunction: "",
	}

	// Apply annotations to options
	for _, annotation := range annotations {
		if err := p.applyInterfaceAnnotation(options, annotation); err != nil {
			return nil, fmt.Errorf("failed to apply annotation %s: %w", annotation.Type, err)
		}
	}

	return options, nil
}

// applyInterfaceAnnotation applies a single annotation to interface options.
func (p *ASTParser) applyInterfaceAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	// Handle simple boolean flags first
	if p.applyInterfaceBooleanFlags(options, annotation) {
		return nil
	}

	// Handle complex annotations that require argument processing
	return p.applyInterfaceComplexAnnotations(options, annotation)
}

// applyInterfaceBooleanFlags handles simple boolean flag annotations for interfaces.
func (p *ASTParser) applyInterfaceBooleanFlags(options *domain.InterfaceOptions, annotation *Annotation) bool {
	switch annotation.Type {
	case "convergen":
		// Base annotation, no action needed
		return true
	case "case", "case:off":
		options.CaseSensitive = annotation.Type == "case"
		return true
	case "getter", "getter:off":
		options.UseGetter = annotation.Type == "getter"
		return true
	case "stringer", "stringer:off":
		options.UseStringer = annotation.Type == "stringer"
		return true
	case "typecast", "typecast:off":
		options.UseTypecast = annotation.Type == "typecast"
		return true
	case "reverse":
		options.AllowReverse = true
		return true
	default:
		return false
	}
}

// applyInterfaceComplexAnnotations handles annotations that require argument processing.
func (p *ASTParser) applyInterfaceComplexAnnotations(options *domain.InterfaceOptions, annotation *Annotation) error {
	switch annotation.Type {
	case "style":
		return p.applyStyleAnnotation(options, annotation)
	case "match":
		return p.applyMatchAnnotation(options, annotation)
	case "recv":
		return p.applyRecvAnnotation(options, annotation)
	case "skip":
		return p.applySkipAnnotation(options, annotation)
	case "map":
		return p.applyMapAnnotation(options, annotation)
	case "conv":
		return p.applyConvAnnotation(options, annotation)
	case "literal":
		return p.applyLiteralAnnotation(options, annotation)
	case "preprocess":
		return p.applyPreprocessAnnotation(options, annotation)
	case "postprocess":
		return p.applyPostprocessAnnotation(options, annotation)
	default:
		p.logUnknownAnnotation(annotation)
		return nil
	}
}

// applyStyleAnnotation applies style annotation to interface options.
func (p *ASTParser) applyStyleAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrStyleAnnotationRequiresArgument
	}

	style, err := p.parseStyle(annotation.Args[0])
	if err != nil {
		return fmt.Errorf("invalid style: %w", err)
	}

	options.Style = style
	return nil
}

// applyMatchAnnotation applies match annotation to interface options.
func (p *ASTParser) applyMatchAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrMatchAnnotationRequiresArgument
	}

	rule, err := p.parseMatchRule(annotation.Args[0])
	if err != nil {
		return fmt.Errorf("invalid match rule: %w", err)
	}

	options.MatchRule = rule
	return nil
}

// applyRecvAnnotation applies receiver annotation to interface options.
func (p *ASTParser) applyRecvAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrRecvAnnotationRequiresReceiverName
	}

	if !p.isValidIdentifier(annotation.Args[0]) {
		return fmt.Errorf("%w: %s", ErrInvalidReceiverName, annotation.Args[0])
	}

	options.ReceiverName = annotation.Args[0]
	return nil
}

// applySkipAnnotation applies skip annotation to interface options.
func (p *ASTParser) applySkipAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrSkipAnnotationRequiresFieldPattern
	}

	options.SkipFields = append(options.SkipFields, annotation.Args[0])
	return nil
}

// applyMapAnnotation applies map annotation to interface options.
func (p *ASTParser) applyMapAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) < 2 {
		return ErrMapAnnotationRequiresArguments
	}

	options.FieldMappings[annotation.Args[0]] = annotation.Args[1]
	return nil
}

// applyConvAnnotation applies conv annotation to interface options.
func (p *ASTParser) applyConvAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) < 2 {
		return ErrConvAnnotationRequiresArguments
	}

	options.TypeConverters[annotation.Args[1]] = annotation.Args[0]
	return nil
}

// applyLiteralAnnotation applies literal annotation to interface options.
func (p *ASTParser) applyLiteralAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) < 2 {
		return ErrLiteralAnnotationRequiresArguments
	}
	// Parse the literal value (may contain spaces)
	value := strings.Join(annotation.Args[1:], " ")
	options.LiteralAssignments[annotation.Args[0]] = value
	return nil
}

// applyPreprocessAnnotation applies preprocess annotation to interface options.
func (p *ASTParser) applyPreprocessAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrPreprocessAnnotationRequiresFunction
	}

	options.PreprocessFunction = annotation.Args[0]
	return nil
}

// applyPostprocessAnnotation applies postprocess annotation to interface options.
func (p *ASTParser) applyPostprocessAnnotation(options *domain.InterfaceOptions, annotation *Annotation) error {
	if len(annotation.Args) == 0 {
		return ErrPostprocessAnnotationRequiresFunction
	}

	options.PostprocessFunction = annotation.Args[0]
	return nil
}

// logUnknownAnnotation logs a warning for unknown annotations.
func (p *ASTParser) logUnknownAnnotation(annotation *Annotation) {
	p.logger.Warn("unknown interface annotation",
		zap.String("type", annotation.Type),
		zap.String("position", p.fileSet.Position(annotation.Position).String()))
}

// parseStyle converts string to domain.VariableStyle.
func (p *ASTParser) parseStyle(styleStr string) (domain.VariableStyle, error) {
	switch strings.ToLower(styleStr) {
	case "camel", "camelcase":
		return domain.StyleCamelCase, nil
	case "snake", "snakecase":
		return domain.StyleSnakeCase, nil
	case "pascal", "pascalcase":
		return domain.StylePascalCase, nil
	default:
		return domain.StyleCamelCase, fmt.Errorf("%w: %s", ErrUnknownStyle, styleStr)
	}
}

// parseMatchRule converts string to domain.MatchRule.
func (p *ASTParser) parseMatchRule(ruleStr string) (domain.MatchRule, error) {
	switch strings.ToLower(ruleStr) {
	case "name", "byname":
		return domain.MatchByName, nil
	case "type", "bytype":
		return domain.MatchByType, nil
	case "tag", "bytag":
		return domain.MatchByTag, nil
	default:
		return domain.MatchByName, fmt.Errorf("%w: %s", ErrUnknownMatchRule, ruleStr)
	}
}

// isValidIdentifier checks if a string is a valid Go identifier.
func (p *ASTParser) isValidIdentifier(id string) bool {
	if id == "" {
		return false
	}

	for i, r := range id {
		if i == 0 {
			if !p.isValidFirstChar(r) {
				return false
			}
		} else {
			if !p.isValidSubsequentChar(r) {
				return false
			}
		}
	}

	return true
}

// isValidFirstChar checks if a rune is valid as the first character of an identifier.
func (p *ASTParser) isValidFirstChar(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || r == '_'
}

// isValidSubsequentChar checks if a rune is valid as a subsequent character of an identifier.
func (p *ASTParser) isValidSubsequentChar(r rune) bool {
	return p.isValidFirstChar(r) || ('0' <= r && r <= '9')
}

// extractInterfaceTypeParams extracts type parameters from generic interface declarations.
// Handles interfaces like: type Converter[T any] interface { ... } and type Mapper[T, U any] interface { ... }
func (p *ASTParser) extractInterfaceTypeParams(
	ctx context.Context,
	obj types.Object,
) ([]domain.TypeParam, error) {
	// Check if this is a named type with type parameters
	named, ok := obj.Type().(*types.Named)
	if !ok {
		// Non-named types cannot have type parameters
		p.logger.Debug("interface is not a named type, no type parameters",
			zap.String("interface_name", obj.Name()))
		return []domain.TypeParam{}, nil
	}

	// Check if it has type parameters
	typeParams := named.TypeParams()
	if typeParams == nil || typeParams.Len() == 0 {
		p.logger.Debug("interface has no type parameters",
			zap.String("interface_name", obj.Name()))
		return []domain.TypeParam{}, nil
	}

	p.logger.Debug("extracting type parameters from interface",
		zap.String("interface_name", obj.Name()),
		zap.Int("type_param_count", typeParams.Len()))

	// Create constraint parser for parsing type parameter constraints
	typeResolver := NewTypeResolver(p.cache, p.logger)
	constraintParser := NewConstraintParser(typeResolver, p.logger)

	// Extract and parse each type parameter
	domainTypeParams := make([]domain.TypeParam, 0, typeParams.Len())

	for i := 0; i < typeParams.Len(); i++ {
		typeParam := typeParams.At(i)
		paramName := typeParam.Obj().Name()

		p.logger.Debug("processing type parameter",
			zap.String("interface_name", obj.Name()),
			zap.Int("param_index", i),
			zap.String("param_name", paramName),
			zap.String("constraint", typeParam.Constraint().String()))

		// Parse the constraint using the constraint parser from TASK-002
		constraint, err := constraintParser.ParseConstraint(ctx, typeParam.Constraint())
		if err != nil {
			p.logger.Error("failed to parse type parameter constraint",
				zap.String("interface_name", obj.Name()),
				zap.String("param_name", paramName),
				zap.Error(err))
			return nil, fmt.Errorf("failed to parse constraint for type parameter %s: %w", paramName, err)
		}

		// Convert parsed constraint to domain type parameter
		domainTypeParam, err := constraintParser.ConvertToDomainTypeParam(paramName, i, constraint)
		if err != nil {
			p.logger.Error("failed to convert constraint to domain type parameter",
				zap.String("interface_name", obj.Name()),
				zap.String("param_name", paramName),
				zap.Error(err))
			return nil, fmt.Errorf("failed to convert constraint for type parameter %s: %w", paramName, err)
		}

		// Validate the type parameter
		if !domainTypeParam.IsValid() {
			p.logger.Error("invalid type parameter configuration",
				zap.String("interface_name", obj.Name()),
				zap.String("param_name", paramName),
				zap.String("constraint_type", domainTypeParam.GetConstraintType()))
			return nil, fmt.Errorf("invalid type parameter %s with constraint type %s", paramName, domainTypeParam.GetConstraintType())
		}

		domainTypeParams = append(domainTypeParams, *domainTypeParam)

		p.logger.Debug("successfully extracted type parameter",
			zap.String("interface_name", obj.Name()),
			zap.String("param_name", paramName),
			zap.String("constraint_type", domainTypeParam.GetConstraintType()),
			zap.String("constraint_string", domainTypeParam.String()))
	}

	p.logger.Info("extracted type parameters from interface",
		zap.String("interface_name", obj.Name()),
		zap.Int("type_param_count", len(domainTypeParams)))

	return domainTypeParams, nil
}

// getDocComment retrieves the documentation comment for an object.
func (p *ASTParser) getDocComment(file *ast.File, obj types.Object) *ast.CommentGroup {
	// Find the AST node corresponding to the object
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == obj.Name() {
						if genDecl.Doc != nil {
							return genDecl.Doc
						}

						if typeSpec.Doc != nil {
							return typeSpec.Doc
						}
					}
				}
			}
		}
	}

	return nil
}
