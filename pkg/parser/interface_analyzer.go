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

// InterfaceInfo contains comprehensive information about a convergen interface.
type InterfaceInfo struct {
	Object      types.Object
	Interface   *types.Interface
	Methods     []types.Object
	Options     *domain.InterfaceOptions
	Annotations []*Annotation
	Marker      string
	Position    token.Pos
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
func (p *ASTParser) analyzeInterface(ctx context.Context, pkg *packages.Package, file *ast.File, obj types.Object, iface *types.Interface) (*InterfaceInfo, error) {
	// Generate unique marker for this interface
	marker, err := gonanoid.Nanoid()
	if err != nil {
		return nil, fmt.Errorf("failed to generate marker: %w", err)
	}

	// Extract and parse annotations
	annotations, err := p.extractInterfaceAnnotations(file, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to extract annotations: %w", err)
	}

	// Parse interface-level options
	options, err := p.parseInterfaceOptions(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interface options: %w", err)
	}

	// Get all methods from the interface
	methods := make([]types.Object, 0, iface.NumMethods())

	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		if method.Exported() {
			methods = append(methods, method)
		}
	}

	interfaceInfo := &InterfaceInfo{
		Object:      obj,
		Interface:   iface,
		Methods:     methods,
		Options:     options,
		Annotations: annotations,
		Marker:      marker,
		Position:    obj.Pos(),
	}

	p.logger.Debug("analyzed convergen interface",
		zap.String("name", obj.Name()),
		zap.Int("methods", len(methods)),
		zap.Int("annotations", len(annotations)),
		zap.String("marker", marker))

	return interfaceInfo, nil
}

// extractInterfaceAnnotations extracts all annotations from interface comments.
func (p *ASTParser) extractInterfaceAnnotations(file *ast.File, obj types.Object) ([]*Annotation, error) {
	docComment := p.getDocComment(file, obj)
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

// parseAnnotation parses a single annotation from a comment.
func (p *ASTParser) parseAnnotation(comment *ast.Comment) *Annotation {
	matches := reNotationInterface.FindStringSubmatch(comment.Text)
	if len(matches) < 2 {
		return nil
	}

	annotationType := matches[1]

	argsString := ""
	if len(matches) > 2 {
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
	switch annotation.Type {
	case "convergen":
		// Base annotation, no action needed
		return nil

	case "style":
		if len(annotation.Args) == 0 {
			return ErrStyleAnnotationRequiresArgument
		}

		style, err := p.parseStyle(annotation.Args[0])
		if err != nil {
			return fmt.Errorf("invalid style: %w", err)
		}

		options.Style = style

	case "match":
		if len(annotation.Args) == 0 {
			return ErrMatchAnnotationRequiresArgument
		}

		rule, err := p.parseMatchRule(annotation.Args[0])
		if err != nil {
			return fmt.Errorf("invalid match rule: %w", err)
		}

		options.MatchRule = rule

	case "case":
		options.CaseSensitive = true

	case "case:off":
		options.CaseSensitive = false

	case "getter":
		options.UseGetter = true

	case "getter:off":
		options.UseGetter = false

	case "stringer":
		options.UseStringer = true

	case "stringer:off":
		options.UseStringer = false

	case "typecast":
		options.UseTypecast = true

	case "typecast:off":
		options.UseTypecast = false

	case "recv":
		if len(annotation.Args) == 0 {
			return ErrRecvAnnotationRequiresReceiverName
		}

		if !p.isValidIdentifier(annotation.Args[0]) {
			return fmt.Errorf("%w: %s", ErrInvalidReceiverName, annotation.Args[0])
		}

		options.ReceiverName = annotation.Args[0]

	case "reverse":
		options.AllowReverse = true

	case "skip":
		if len(annotation.Args) == 0 {
			return ErrSkipAnnotationRequiresFieldPattern
		}

		options.SkipFields = append(options.SkipFields, annotation.Args[0])

	case "map":
		if len(annotation.Args) < 2 {
			return ErrMapAnnotationRequiresArguments
		}

		options.FieldMappings[annotation.Args[0]] = annotation.Args[1]

	case "conv":
		if len(annotation.Args) < 2 {
			return ErrConvAnnotationRequiresArguments
		}

		options.TypeConverters[annotation.Args[1]] = annotation.Args[0]

	case "literal":
		if len(annotation.Args) < 2 {
			return ErrLiteralAnnotationRequiresArguments
		}
		// Parse the literal value (may contain spaces)
		value := strings.Join(annotation.Args[1:], " ")
		options.LiteralAssignments[annotation.Args[0]] = value

	case "preprocess":
		if len(annotation.Args) == 0 {
			return ErrPreprocessAnnotationRequiresFunction
		}

		options.PreprocessFunction = annotation.Args[0]

	case "postprocess":
		if len(annotation.Args) == 0 {
			return ErrPostprocessAnnotationRequiresFunction
		}

		options.PostprocessFunction = annotation.Args[0]

	default:
		p.logger.Warn("unknown interface annotation",
			zap.String("type", annotation.Type),
			zap.String("position", p.fileSet.Position(annotation.Position).String()))
	}

	return nil
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
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}

	return true
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
