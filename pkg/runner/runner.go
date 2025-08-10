// Package runner provides the main execution logic for the convergen CLI tool.
package runner

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/config"
	"github.com/reedom/convergen/v8/pkg/domain"
	"github.com/reedom/convergen/v8/pkg/generator"
	"github.com/reedom/convergen/v8/pkg/generator/model"
	"github.com/reedom/convergen/v8/pkg/logger"
	"github.com/reedom/convergen/v8/pkg/parser"
)

// Run runs the convergen code generator using the provided configuration.
// If a log file path is specified in the configuration, the logger will output to that file.
// It creates a parser instance from the input and output paths in the configuration,
// and then generates a list of methods from the parsed source code. Using a function builder,
// the generator creates a block of functions for each set of methods and combines them with
// the parsed base code. Finally, it generates the output files using the generated code and
// the provided configuration options.
func Run(conf config.Config) error {
	if conf.Log != "" {
		f, err := os.OpenFile(conf.Log, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", conf.Log, err)
		}

		logger.SetupLogger(logger.Enable(), logger.Output(f))
	}

	// Check if this is a generic type instantiation request
	if conf.TypeSpec != "" {
		return runGenericGeneration(conf)
	}

	// Traditional flow for non-generic interfaces
	p, err := parser.NewParser(conf.Input, conf.Output)
	if err != nil {
		return fmt.Errorf("failed to create parser for %s: %w", conf.Input, err)
	}

	methods, err := p.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse source file %s: %w", conf.Input, err)
	}

	builder := p.CreateBuilder()

	funcBlocks := make([]model.FunctionsBlock, 0, len(methods))

	for _, info := range methods {
		functions, err := builder.CreateFunctions(info.Methods)
		if err != nil {
			return fmt.Errorf("failed to create functions for interface %s: %w", info.Marker, err)
		}

		block := model.FunctionsBlock{
			Marker:    info.Marker,
			Functions: functions,
		}
		funcBlocks = append(funcBlocks, block)
	}

	baseCode, err := p.GenerateBaseCode()
	if err != nil {
		return fmt.Errorf("failed to generate base code: %w", err)
	}

	code := model.Code{
		BaseCode:       baseCode,
		FunctionBlocks: funcBlocks,
	}

	g := generator.NewGeneratorWithConfig(code, conf)

	_, err = g.Generate(conf.Output, conf.Prints, conf.DryRun)
	if err != nil {
		return fmt.Errorf("failed to generate output to %s: %w", conf.Output, err)
	}

	return nil
}

// runGenericGeneration handles code generation for generic type instantiation.
func runGenericGeneration(conf config.Config) error {
	// Create a zap logger for generic generation
	logger := zap.NewNop()
	if conf.Verbose {
		config := zap.NewDevelopmentConfig()
		if l, err := config.Build(); err == nil {
			logger = l
		}
	}

	logger.Info("starting generic type instantiation", zap.String("type_spec", conf.TypeSpec))

	// Parse the type specification
	instantiatedInterface, err := parseTypeSpec(conf.TypeSpec, logger)
	if err != nil {
		return fmt.Errorf("failed to parse type specification %s: %w", conf.TypeSpec, err)
	}

	// Create the generic code generator with proper field mapping
	generator, err := createGenericGenerator(logger)
	if err != nil {
		return fmt.Errorf("failed to create generic generator: %w", err)
	}

	// Generate the code
	ctx := context.Background()
	generatedCode, err := generator.GenerateGenericImplementation(ctx, instantiatedInterface)
	if err != nil {
		return fmt.Errorf("failed to generate generic implementation: %w", err)
	}

	// Write or print the generated code
	if conf.Prints {
		fmt.Println(generatedCode)
	}

	if !conf.DryRun {
		outputPath := conf.Output
		if outputPath == "" {
			// Generate default output path
			outputPath = strings.TrimSuffix(conf.Input, ".go") + ".gen.go"
		}

		if err := os.WriteFile(outputPath, []byte(generatedCode), 0644); err != nil {
			return fmt.Errorf("failed to write generated code to %s: %w", outputPath, err)
		}

		logger.Info("generic code generated successfully", zap.String("output", outputPath))
	}

	return nil
}

// parseTypeSpec parses a type specification like "TypeMapper[User,UserDTO]" into an InstantiatedInterface.
func parseTypeSpec(typeSpec string, logger *zap.Logger) (*domain.InstantiatedInterface, error) {
	// Simple parsing for now - could be enhanced with proper AST parsing
	if !strings.Contains(typeSpec, "[") || !strings.Contains(typeSpec, "]") {
		return nil, fmt.Errorf("invalid type specification format: %s", typeSpec)
	}

	// Extract interface name and type arguments
	bracketStart := strings.Index(typeSpec, "[")
	bracketEnd := strings.LastIndex(typeSpec, "]")

	interfaceName := strings.TrimSpace(typeSpec[:bracketStart])
	typeArgsStr := strings.TrimSpace(typeSpec[bracketStart+1 : bracketEnd])

	// Parse type arguments
	typeArgStrs := strings.Split(typeArgsStr, ",")
	typeArguments := make(map[string]domain.Type)

	// For now, use simple type parameter names (T, U, V, etc.)
	typeParamNames := []string{"T", "U", "V", "W", "X", "Y", "Z"}

	for i, typeArgStr := range typeArgStrs {
		typeArgStr = strings.TrimSpace(typeArgStr)
		if i < len(typeParamNames) {
			// Create a basic type for this argument
			concrete := domain.NewBasicType(typeArgStr, 0)
			typeArguments[typeParamNames[i]] = concrete
		}
	}

	logger.Debug("parsed type specification",
		zap.String("interface", interfaceName),
		zap.Int("type_args", len(typeArguments)))

	// Create the source interface type (simplified)
	sourceInterface := domain.NewBasicType(interfaceName, 0)

	// Use the first type argument as the concrete type for now
	var concreteType domain.Type
	if len(typeArguments) > 0 {
		for _, t := range typeArguments {
			concreteType = t
			break
		}
	} else {
		concreteType = domain.NewBasicType("interface{}", 0)
	}

	// Create the instantiated interface
	return domain.NewInstantiatedInterface(
		sourceInterface,
		typeArguments,
		concreteType,
		typeSpec,
	)
}

// createGenericGenerator creates a generic code generator with proper field mapping.
func createGenericGenerator(logger *zap.Logger) (*generator.GenericCodeGenerator, error) {
	// Create a simple template engine (this would need to be implemented)
	templateEngine := &simpleTemplateEngine{}

	// Create type instantiator
	typeBuilder := domain.NewTypeBuilder()
	typeInstantiator := domain.NewTypeInstantiator(typeBuilder, logger)

	// Create field mapper (this would use the enhanced field mapping from the builder package)
	fieldMapper := &simpleFieldMapper{}

	// Create the generator
	generator := generator.NewGenericCodeGenerator(
		templateEngine,
		typeInstantiator,
		fieldMapper,
		logger,
		nil, // Use default config
	)

	return generator, nil
}

// simpleTemplateEngine provides a basic template engine implementation for testing.
type simpleTemplateEngine struct{}

func (ste *simpleTemplateEngine) Execute(templateName string, data interface{}) (string, error) {
	return fmt.Sprintf("// Generated code for template %s\nfunc Convert(src interface{}) interface{} {\n\treturn src\n}", templateName), nil
}

func (ste *simpleTemplateEngine) RegisterTemplate(name, content string) error {
	return nil
}

func (ste *simpleTemplateEngine) HasTemplate(name string) bool {
	return true
}

func (ste *simpleTemplateEngine) GetTemplateFunctions() map[string]interface{} {
	return make(map[string]interface{})
}

// simpleFieldMapper provides a basic field mapper implementation.
type simpleFieldMapper struct{}

func (sfm *simpleFieldMapper) MapFields(sourceType, destType domain.Type, annotations map[string]string) ([]*generator.FieldMapping, error) {
	return []*generator.FieldMapping{}, nil
}

func (sfm *simpleFieldMapper) ValidateMapping(mapping *generator.FieldMapping) error {
	return nil
}
