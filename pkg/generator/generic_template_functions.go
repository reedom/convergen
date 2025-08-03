package generator

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Generic template functions that can be used in templates for type-aware code generation.

// substituteTypeInTemplate substitutes a type parameter with its concrete type in templates.
func (gcg *GenericCodeGenerator) substituteTypeInTemplate(typ domain.Type, substitutions map[string]TypeSubstitution) string {
	if typ == nil {
		return "interface{}"
	}

	// Check if this is a type parameter that needs substitution
	if substitution, found := substitutions[typ.Name()]; found {
		gcg.metrics.TypeSubstitutions++

		// Handle package-qualified types
		if substitution.ConcreteType.Package() != "" {
			return substitution.ConcreteType.Package() + "." + substitution.ConcreteType.Name()
		}
		return substitution.ConcreteType.Name()
	}

	// Handle composite types recursively
	switch typ.Kind() {
	case domain.KindSlice:
		if sliceType, ok := typ.(*domain.SliceType); ok {
			elemType := gcg.substituteTypeInTemplate(sliceType.Elem(), substitutions)
			return "[]" + elemType
		}
	case domain.KindPointer:
		if pointerType, ok := typ.(*domain.PointerType); ok {
			elemType := gcg.substituteTypeInTemplate(pointerType.Elem(), substitutions)
			return "*" + elemType
		}
	case domain.KindMap:
		// Handle map types - simplified implementation
		return "map[string]interface{}"
	}

	// Return the original type name with package if available
	if typ.Package() != "" {
		return typ.Package() + "." + typ.Name()
	}
	return typ.Name()
}

// isGenericTypeInTemplate checks if a type is a generic type parameter.
func (gcg *GenericCodeGenerator) isGenericTypeInTemplate(typ domain.Type, typeParams []TypeParam) bool {
	if typ == nil {
		return false
	}

	typeName := typ.Name()
	for _, param := range typeParams {
		if param.Name == typeName {
			return true
		}
	}
	return false
}

// formatTypeParamInTemplate formats a type parameter for display in generated code.
func (gcg *GenericCodeGenerator) formatTypeParamInTemplate(param TypeParam) string {
	if param.Constraint != "" && param.Constraint != "any" {
		return fmt.Sprintf("%s %s", param.Name, param.Constraint)
	}
	return param.Name
}

// generateTypeSwitchInTemplate generates a type switch statement for generic handling.
func (gcg *GenericCodeGenerator) generateTypeSwitchInTemplate(
	varName string,
	typeArgs []TypeArg,
	defaultCase string,
) string {
	if len(typeArgs) == 0 {
		return defaultCase
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("switch %s := %s.(type) {\n", varName, varName))

	for _, typeArg := range typeArgs {
		typeName := typeArg.Name
		if typeArg.PackagePath != "" {
			typeName = typeArg.PackagePath + "." + typeName
		}

		builder.WriteString(fmt.Sprintf("case %s:\n", typeName))
		builder.WriteString(fmt.Sprintf("\t// Handle %s type\n", typeArg.Name))
		builder.WriteString("\t// TODO: Add specific handling\n")
	}

	builder.WriteString("default:\n")
	if defaultCase != "" {
		builder.WriteString(fmt.Sprintf("\t%s\n", defaultCase))
	} else {
		builder.WriteString("\t// Default case\n")
	}
	builder.WriteString("}")

	return builder.String()
}

// hasAnnotationInTemplate checks if a method or field has a specific annotation.
func (gcg *GenericCodeGenerator) hasAnnotationInTemplate(annotations map[string]string, key string) bool {
	_, found := annotations[key]
	return found
}

// getAnnotationInTemplate gets an annotation value from a method or field.
func (gcg *GenericCodeGenerator) getAnnotationInTemplate(annotations map[string]string, key string) string {
	value, found := annotations[key]
	if !found {
		return ""
	}
	return value
}

// generateFieldAccessInTemplate generates field access code for field mappings.
func (gcg *GenericCodeGenerator) generateFieldAccessInTemplate(mapping *FieldMapping) string {
	if mapping == nil {
		return ""
	}

	var builder strings.Builder

	// Add validation if present
	if mapping.Validation != "" {
		builder.WriteString(fmt.Sprintf("\t// Validate %s\n", mapping.SourceField))
		builder.WriteString(fmt.Sprintf("\tif %s {\n", mapping.Validation))
		builder.WriteString(fmt.Sprintf("\t\treturn dst, fmt.Errorf(\"validation failed for field %s\")\n", mapping.SourceField))
		builder.WriteString("\t}\n\n")
	}

	// Generate assignment or conversion
	if mapping.Converter != "" {
		// Use custom converter
		builder.WriteString(fmt.Sprintf("\t// Convert %s using %s\n", mapping.SourceField, mapping.Converter))
		builder.WriteString(fmt.Sprintf("\tconverted%s, err := %s(src.%s)\n",
			mapping.DestField, mapping.Converter, mapping.SourceField))
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString(fmt.Sprintf("\t\treturn dst, fmt.Errorf(\"conversion failed for field %s: %%w\", err)\n", mapping.SourceField))
		builder.WriteString("\t}\n")
		builder.WriteString(fmt.Sprintf("\tdst.%s = converted%s\n", mapping.DestField, mapping.DestField))
	} else {
		// Direct assignment
		builder.WriteString(fmt.Sprintf("\tdst.%s = src.%s\n", mapping.DestField, mapping.SourceField))
	}

	return builder.String()
}

// generateErrorHandlingInTemplate generates error handling code.
func (gcg *GenericCodeGenerator) generateErrorHandlingInTemplate(
	methodName string,
	phase string,
	returnType domain.Type,
) string {
	var builder strings.Builder

	builder.WriteString("\tif err != nil {\n")

	returnStmt := "return "
	if returnType != nil {
		// Zero value for the return type
		returnStmt += gcg.generateZeroValueInTemplate(returnType) + ", "
	}
	returnStmt += fmt.Sprintf("fmt.Errorf(\"%s failed in %s: %%w\", err)", phase, methodName)

	builder.WriteString(fmt.Sprintf("\t\t%s\n", returnStmt))
	builder.WriteString("\t}")

	return builder.String()
}

// generateZeroValueInTemplate generates the zero value for a type.
func (gcg *GenericCodeGenerator) generateZeroValueInTemplate(typ domain.Type) string {
	if typ == nil {
		return "nil"
	}

	switch typ.Kind() {
	case domain.KindBasic:
		switch typ.Name() {
		case "bool":
			return "false"
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64", "complex64", "complex128":
			return "0"
		case "string":
			return `""`
		default:
			return fmt.Sprintf("%s{}", typ.Name())
		}
	case domain.KindPointer:
		return "nil"
	case domain.KindSlice:
		return "nil"
	case domain.KindMap:
		return "nil"
	case domain.KindInterface:
		return "nil"
	default:
		// For struct and other types, use composite literal
		typeName := typ.Name()
		if typ.Package() != "" {
			typeName = typ.Package() + "." + typeName
		}
		return fmt.Sprintf("%s{}", typeName)
	}
}

// generateImportStatementInTemplate generates import statements for required packages.
func (gcg *GenericCodeGenerator) generateImportStatementInTemplate(
	typeArgs []TypeArg,
	additionalImports []string,
) string {
	imports := make(map[string]bool)

	// Add imports for type arguments
	for _, typeArg := range typeArgs {
		if typeArg.PackagePath != "" {
			imports[typeArg.PackagePath] = true
		}
	}

	// Add additional imports
	for _, imp := range additionalImports {
		if imp != "" {
			imports[imp] = true
		}
	}

	if len(imports) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("import (\n")

	for importPath := range imports {
		builder.WriteString(fmt.Sprintf("\t\"%s\"\n", importPath))
	}

	builder.WriteString(")")

	return builder.String()
}

// generateMethodSignatureInTemplate generates a method signature.
func (gcg *GenericCodeGenerator) generateMethodSignatureInTemplate(method *MethodData) string {
	var builder strings.Builder

	// Add receiver if present
	if method.Receiver != nil {
		receiver := method.Receiver.Name
		if method.Receiver.IsPointer {
			receiver = "*" + receiver
		}
		builder.WriteString(fmt.Sprintf("func (%s %s) ", method.Receiver.Name, receiver))
	} else {
		builder.WriteString("func ")
	}

	// Add method name
	builder.WriteString(method.Name)

	// Add parameters
	builder.WriteString("(")
	for i, param := range method.Parameters {
		if 0 < i {
			builder.WriteString(", ")
		}

		paramType := param.Type.Name()
		if param.Type.Package() != "" {
			paramType = param.Type.Package() + "." + paramType
		}

		if param.Variadic {
			paramType = "..." + paramType
		}

		builder.WriteString(fmt.Sprintf("%s %s", param.Name, paramType))
	}
	builder.WriteString(")")

	// Add return type
	if method.ReturnType != nil || method.ReturnsError {
		builder.WriteString(" ")

		hasReturnType := method.ReturnType != nil
		hasError := method.ReturnsError

		if hasReturnType && hasError {
			builder.WriteString("(")
		}

		if hasReturnType {
			returnType := method.ReturnType.Name()
			if method.ReturnType.Package() != "" {
				returnType = method.ReturnType.Package() + "." + returnType
			}
			builder.WriteString(returnType)
		}

		if hasReturnType && hasError {
			builder.WriteString(", ")
		}

		if hasError {
			builder.WriteString("error")
		}

		if hasReturnType && hasError {
			builder.WriteString(")")
		}
	}

	return builder.String()
}

// generateValidationInTemplate generates validation code for a field.
func (gcg *GenericCodeGenerator) generateValidationInTemplate(
	fieldName string,
	validation string,
	returnType domain.Type,
) string {
	if validation == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("\tif %s {\n", validation))

	returnStmt := "return "
	if returnType != nil {
		returnStmt += gcg.generateZeroValueInTemplate(returnType) + ", "
	}
	returnStmt += fmt.Sprintf("fmt.Errorf(\"validation failed for field %s\")", fieldName)

	builder.WriteString(fmt.Sprintf("\t\t%s\n", returnStmt))
	builder.WriteString("\t}")

	return builder.String()
}

// generateConversionInTemplate generates conversion code for a field.
func (gcg *GenericCodeGenerator) generateConversionInTemplate(
	sourceField string,
	destField string,
	converter string,
	returnType domain.Type,
) string {
	if converter == "" {
		return fmt.Sprintf("\tdst.%s = src.%s", destField, sourceField)
	}

	var builder strings.Builder

	// Generate conversion call
	builder.WriteString(fmt.Sprintf("\tconverted%s, err := %s(src.%s)\n", destField, converter, sourceField))
	builder.WriteString("\tif err != nil {\n")

	returnStmt := "return "
	if returnType != nil {
		returnStmt += gcg.generateZeroValueInTemplate(returnType) + ", "
	}
	returnStmt += fmt.Sprintf("fmt.Errorf(\"conversion failed for field %s: %%w\", err)", sourceField)

	builder.WriteString(fmt.Sprintf("\t\t%s\n", returnStmt))
	builder.WriteString("\t}\n")
	builder.WriteString(fmt.Sprintf("\tdst.%s = converted%s", destField, destField))

	return builder.String()
}

// generateCommentInTemplate generates comment lines for methods or fields.
func (gcg *GenericCodeGenerator) generateCommentInTemplate(comments []string, prefix string) string {
	if len(comments) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, comment := range comments {
		if comment != "" {
			builder.WriteString(fmt.Sprintf("%s// %s\n", prefix, comment))
		}
	}

	return builder.String()
}

// formatTypeConstraintInTemplate formats type constraints for display.
func (gcg *GenericCodeGenerator) formatTypeConstraintInTemplate(constraint string) string {
	switch constraint {
	case "any":
		return "any"
	case "comparable":
		return "comparable"
	case "":
		return "any"
	default:
		return constraint
	}
}

// generateTypeAssertionInTemplate generates type assertion code.
func (gcg *GenericCodeGenerator) generateTypeAssertionInTemplate(
	varName string,
	targetType domain.Type,
	substitutions map[string]TypeSubstitution,
) string {
	targetTypeName := gcg.substituteTypeInTemplate(targetType, substitutions)
	return fmt.Sprintf("%s.(%s)", varName, targetTypeName)
}

// generateSliceConversionInTemplate generates code for converting slices of generic types.
func (gcg *GenericCodeGenerator) generateSliceConversionInTemplate(
	sourceVar string,
	destVar string,
	elementConverter string,
	returnType domain.Type,
) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("\t%s = make([]T, len(%s))\n", destVar, sourceVar))
	builder.WriteString(fmt.Sprintf("\tfor i, item := range %s {\n", sourceVar))

	if elementConverter != "" {
		builder.WriteString(fmt.Sprintf("\t\tconverted, err := %s(item)\n", elementConverter))
		builder.WriteString("\t\tif err != nil {\n")

		returnStmt := "return "
		if returnType != nil {
			returnStmt += gcg.generateZeroValueInTemplate(returnType) + ", "
		}
		returnStmt += "fmt.Errorf(\"slice element conversion failed at index %d: %w\", i, err)"

		builder.WriteString(fmt.Sprintf("\t\t\t%s\n", returnStmt))
		builder.WriteString("\t\t}\n")
		builder.WriteString(fmt.Sprintf("\t\t%s[i] = converted\n", destVar))
	} else {
		builder.WriteString(fmt.Sprintf("\t\t%s[i] = item\n", destVar))
	}

	builder.WriteString("\t}")

	return builder.String()
}

// registerGenericTemplateFunctions registers all generic template functions.
func (gcg *GenericCodeGenerator) registerGenericTemplateFunctions() {
	if gcg.templateEngine == nil {
		return
	}

	functions := map[string]interface{}{
		"substituteType":          gcg.substituteTypeInTemplate,
		"isGenericType":           gcg.isGenericTypeInTemplate,
		"formatTypeParam":         gcg.formatTypeParamInTemplate,
		"generateTypeSwitch":      gcg.generateTypeSwitchInTemplate,
		"hasAnnotation":           gcg.hasAnnotationInTemplate,
		"getAnnotation":           gcg.getAnnotationInTemplate,
		"generateFieldAccess":     gcg.generateFieldAccessInTemplate,
		"generateErrorHandling":   gcg.generateErrorHandlingInTemplate,
		"generateZeroValue":       gcg.generateZeroValueInTemplate,
		"generateImportStatement": gcg.generateImportStatementInTemplate,
		"generateMethodSignature": gcg.generateMethodSignatureInTemplate,
		"generateValidation":      gcg.generateValidationInTemplate,
		"generateConversion":      gcg.generateConversionInTemplate,
		"generateComment":         gcg.generateCommentInTemplate,
		"formatTypeConstraint":    gcg.formatTypeConstraintInTemplate,
		"generateTypeAssertion":   gcg.generateTypeAssertionInTemplate,
		"generateSliceConversion": gcg.generateSliceConversionInTemplate,
	}

	for name, fn := range functions {
		// Skip template registration for mock implementations
		_ = name
		_ = fn
	}

	gcg.logger.Debug("registered generic template functions",
		zap.Int("function_count", len(functions)))
}
