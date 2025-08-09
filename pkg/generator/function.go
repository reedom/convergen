package generator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/reedom/convergen/v8/pkg/generator/model"
)

// FuncToString generates the string representation of a given Function.
// The generated string can be used to represent the Function as Go code.
// The function generates a doc comment (if any), the function signature,
// the variable declarations (if any), the assignment statements, and the return statement.
// The function uses ManipulatorToString to generate the string representation of manipulators.
func (g *Generator) FuncToString(f *model.Function) string {
	// Determine output style based on CLI flags and function configuration
	initialOutputStyle := g.determineOutputStyle(f)

	// Apply struct literal fallback detection
	actualOutputStyle, fallbackApplied := g.applyStructLiteralFallback(f, initialOutputStyle)

	// Add verbose comment if requested
	if g.config.IsVerboseMode() {
		f.Comments = append([]string{fmt.Sprintf("// Generated with output style: %s", actualOutputStyle)}, f.Comments...)
		if fallbackApplied {
			f.Comments = append(f.Comments, fmt.Sprintf("// Note: Fallback from %s to %s: %s", initialOutputStyle, actualOutputStyle, g.getFallbackReason(f)))
		}
	}

	// Generate code based on the determined output style
	if actualOutputStyle == model.OutputStyleStructLiteral {
		return g.generateStructLiteralFunction(f)
	}

	// Traditional assignment generation (existing code)
	return g.generateTraditionalFunction(f)
}

// generateTraditionalFunction generates the string representation of a Function using traditional assignment syntax.
// This is the original generation method for backward compatibility and complex cases.
func (g *Generator) generateTraditionalFunction(f *model.Function) string {
	var sb strings.Builder

	// doc comment
	for i := range f.Comments {
		sb.WriteString(f.Comments[i])
		sb.WriteString("\n")
	}

	// "func"
	sb.WriteString("func ")

	if f.IsReceiverMethod() {
		// "func (r *MyStruct)"
		sb.WriteString("(")

		// Use new fields if available, otherwise fall back to old behavior
		if f.ReceiverVar != "" && f.ReceiverType != "" {
			// New behavior: use parsed receiver variable and type
			sb.WriteString(f.ReceiverVar)
			sb.WriteString(" ")
			sb.WriteString(f.ReceiverType)
		} else {
			// Backward compatibility: use original receiver and source type
			sb.WriteString(f.Receiver)
			sb.WriteString(" ")
			sb.WriteString(f.Src.FullType())
		}

		sb.WriteString(") ")
	}

	// "func (r *SrcModel) Name("
	sb.WriteString(f.Name)
	sb.WriteString("(")

	if f.DstVarStyle == model.DstVarArg {
		// "func Name(dst *DstModel"
		sb.WriteString(f.Dst.Name)
		sb.WriteString(" *")
		sb.WriteString(f.Dst.PtrLessFullType())

		if !f.IsReceiverMethod() {
			// "func Name(dst *DstModel, "
			sb.WriteString(", ")
		}
	}

	if !f.IsReceiverMethod() {
		// "func Name(dst *DstModel, src *SrcModel"
		sb.WriteString(f.Src.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
	}

	for _, args := range f.AdditionalArgs {
		fullType := args.FullType()

		if strings.Contains(args.Type, "/") {
			re := regexp.MustCompile(`^([^a-zA-Z0-9]*)([a-zA-Z0-9].*/)(.+)$`)
			fullType = re.ReplaceAllString(fullType, "$1$3")
		}

		sb.WriteString(", ")
		sb.WriteString(args.Name)
		sb.WriteString(" ")
		sb.WriteString(fullType)
	}

	// "func Name(dst *DstModel, src *SrcModel)"
	sb.WriteString(") ")

	if f.DstVarStyle == model.DstVarReturn {
		writeReturnSignature(f, &sb)

		if f.Dst.Pointer {
			writePointerInitialization(f, &sb)

			sb.WriteString(f.Dst.PtrLessFullType())
			sb.WriteString("{}\n")
		}
	} else {
		if f.RetError {
			// "func Name(dst *DstModel, src *SrcModel) (err error) {"
			sb.WriteString("(err error) {\n")
		} else {
			// "func Name(dst *DstModel, src *SrcModel) {"
			sb.WriteString("{\n")
		}
	}

	if f.PreProcess != nil {
		sb.WriteString(g.ManipulatorToString(f.PreProcess, f.Src, f.Dst, f.AdditionalArgs))
	}

	for i := range f.Assignments {
		sb.WriteString(AssignmentToString(f, f.Assignments[i]))
	}

	if f.PostProcess != nil {
		sb.WriteString(g.ManipulatorToString(f.PostProcess, f.Src, f.Dst, f.AdditionalArgs))
	}

	if f.RetError || f.DstVarStyle == model.DstVarReturn {
		sb.WriteString("\nreturn\n")
	}

	sb.WriteString("}\n\n")

	return sb.String()
}

// writeReturnSignature writes the return signature part of the function.
func writeReturnSignature(f *model.Function, sb *strings.Builder) {
	// "func Name(src *SrcModel) (dst *DstModel"
	sb.WriteString("(")
	sb.WriteString(f.Dst.Name)
	sb.WriteString(" ")
	sb.WriteString(f.Dst.FullType())

	if f.RetError {
		// "func Name(src *SrcModel) (dst *DstModel, err error"
		sb.WriteString(", err error")
	}

	// "func Name(src *SrcModel) (dst *DstModel) {"
	sb.WriteString(") {\n")
}

// writePointerInitialization writes the pointer initialization part.
func writePointerInitialization(f *model.Function, sb *strings.Builder) {
	// "dst = &DstModel{}"
	sb.WriteString(f.Dst.Name)
	sb.WriteString(" = ")

	if f.Dst.Pointer {
		sb.WriteString("&")
	}
}

// determineOutputStyle determines the output style based on CLI flags and other factors.
// Priority: CLI flags (highest) > method annotations > interface annotations > default.
func (g *Generator) determineOutputStyle(f *model.Function) model.OutputStyle {
	// Priority 1: CLI flags (highest priority)
	if g.config.IsStructLiteralDisabled() {
		return model.OutputStyleTraditional
	}
	if g.config.IsStructLiteralExplicitlyEnabled() {
		return model.OutputStyleStructLiteral
	}

	// Priority 2: Function-level output style (if set)
	if f.OutputStyle != "" {
		return f.OutputStyle
	}

	// Priority 3: Default behavior (for now, traditional assignment to maintain compatibility)
	// TODO: Once struct literal is fully implemented and tested, consider defaulting to OutputStyleAuto
	return model.OutputStyleTraditional
}

// applyStructLiteralFallback applies automatic fallback detection for struct literal compatibility.
// Returns the final output style to use and whether a fallback was applied.
func (g *Generator) applyStructLiteralFallback(f *model.Function, requestedStyle model.OutputStyle) (model.OutputStyle, bool) {
	// If traditional assignment is explicitly requested, no fallback needed
	if requestedStyle == model.OutputStyleTraditional {
		return requestedStyle, false
	}

	// If struct literal is explicitly forced via CLI, validate compatibility and fail if incompatible
	if requestedStyle == model.OutputStyleStructLiteral {
		if !g.canUseStructLiteral(f) {
			// For now, we'll apply fallback even for forced struct literal to avoid breaking builds
			// TODO: Consider making this configurable or logging a warning
			return model.OutputStyleTraditional, true
		}
		return requestedStyle, false
	}

	// For OutputStyleAuto, apply intelligent fallback detection
	if requestedStyle == model.OutputStyleAuto {
		if g.canUseStructLiteral(f) {
			return model.OutputStyleStructLiteral, false
		}
		return model.OutputStyleTraditional, true
	}

	// Default case: fallback to traditional assignment for unknown styles
	return model.OutputStyleTraditional, true
}

// canUseStructLiteral determines whether a function can be generated using struct literal syntax.
// It performs compatibility analysis to detect incompatible features that require traditional assignment.
func (g *Generator) canUseStructLiteral(f *model.Function) bool {
	// Check for incompatible features that require imperative execution
	if f.PreProcess != nil || f.PostProcess != nil {
		return false
	}

	// Check for incompatible destination variable style
	// :style arg methods modify the passed argument, incompatible with struct literal return
	if f.DstVarStyle == model.DstVarArg {
		return false
	}

	// Check all assignments for compatibility
	for _, assignment := range f.Assignments {
		if !g.canAssignInStructLiteral(assignment) {
			return false
		}
	}

	return true
}

// canAssignInStructLiteral determines whether an assignment can be included in a struct literal.
// Returns false for assignments that require imperative execution or error handling.
func (g *Generator) canAssignInStructLiteral(assignment model.Assignment) bool {
	// Error-returning assignments need imperative style for error checking
	if assignment.RetError() {
		return false
	}

	// Complex assignments that generate multiple statements need imperative style
	if g.isComplexAssignment(assignment) {
		return false
	}

	// Skip and NoMatch assignments are not included in struct literals
	// but they don't prevent struct literal usage
	switch assignment.(type) {
	case model.SkipField, model.NoMatchField:
		return true // These don't appear in struct literal but don't prevent it
	default:
		return true // Simple value assignments are compatible
	}
}

// isComplexAssignment determines if an assignment generates complex imperative code.
// Complex assignments include loops, conditional blocks, and multiple statements.
func (g *Generator) isComplexAssignment(assignment model.Assignment) bool {
	switch assignment.(type) {
	case model.SimpleField:
		// Simple field assignments are not complex
		return false
	case model.SkipField, model.NoMatchField:
		// Skip/NoMatch don't generate complex code (just comments)
		return false
	case model.NestStruct:
		// Nested structs generate conditional blocks and are complex
		return true
	case model.SliceAssignment, model.SliceLoopAssignment, model.SliceTypecastAssignment:
		// All slice assignments generate conditional blocks and loops
		return true
	default:
		// Unknown assignment types are considered complex for safety
		return true
	}
}

// getFallbackReason returns a human-readable explanation of why struct literal cannot be used.
// This is used for verbose mode reporting and debugging.
func (g *Generator) getFallbackReason(f *model.Function) string {
	if f.PreProcess != nil {
		return "preprocess annotation requires imperative execution before struct creation"
	}
	if f.PostProcess != nil {
		return "postprocess annotation requires imperative execution after struct creation"
	}
	if f.DstVarStyle == model.DstVarArg {
		return ":style arg annotation modifies passed argument, incompatible with struct literal return"
	}

	// Check assignments for specific reasons
	for i, assignment := range f.Assignments {
		if assignment.RetError() {
			return fmt.Sprintf("assignment %d returns error, requires imperative error handling", i+1)
		}
		if g.isComplexAssignment(assignment) {
			switch assignment.(type) {
			case model.NestStruct:
				return fmt.Sprintf("assignment %d contains nested struct with conditional logic", i+1)
			case model.SliceAssignment, model.SliceLoopAssignment, model.SliceTypecastAssignment:
				return fmt.Sprintf("assignment %d contains slice operations requiring loops and conditionals", i+1)
			default:
				return fmt.Sprintf("assignment %d contains complex logic requiring imperative execution", i+1)
			}
		}
	}

	// This should not happen if canUseStructLiteral works correctly
	return "unknown compatibility issue detected"
}

// ErrIncompatibleStructLiteral is returned when struct literal is forced but incompatible features are detected.
var ErrIncompatibleStructLiteral = errors.New("method is forced to use struct literal but has incompatible features")

// validateStructLiteralCompatibility performs pre-generation validation for forced struct literal usage.
// Returns an error if struct literal is forced but incompatible features are detected.
func (g *Generator) validateStructLiteralCompatibility(f *model.Function) error {
	// Check if struct literal is explicitly forced through configuration
	// TODO: This will need to check configuration flags when they're integrated
	if f.OutputStyle == model.OutputStyleStructLiteral && !g.canUseStructLiteral(f) {
		return fmt.Errorf("method %s: %w: %s",
			f.Name, ErrIncompatibleStructLiteral, g.getFallbackReason(f))
	}

	return nil
}

// generateStructLiteralFunction generates the string representation of a Function using struct literal syntax.
// This method is used when the output style is OutputStyleStructLiteral and the function is compatible.
func (g *Generator) generateStructLiteralFunction(f *model.Function) string {
	var sb strings.Builder

	// doc comment
	for i := range f.Comments {
		sb.WriteString(f.Comments[i])
		sb.WriteString("\n")
	}

	// "func"
	sb.WriteString("func ")

	if f.IsReceiverMethod() {
		// "func (r *MyStruct)"
		sb.WriteString("(")

		// Use new fields if available, otherwise fall back to old behavior
		if f.ReceiverVar != "" && f.ReceiverType != "" {
			// New behavior: use parsed receiver variable and type
			sb.WriteString(f.ReceiverVar)
			sb.WriteString(" ")
			sb.WriteString(f.ReceiverType)
		} else {
			// Backward compatibility: use original receiver and source type
			sb.WriteString(f.Receiver)
			sb.WriteString(" ")
			sb.WriteString(f.Src.FullType())
		}

		sb.WriteString(") ")
	}

	// "func (r *SrcModel) Name("
	sb.WriteString(f.Name)
	sb.WriteString("(")

	// Add source parameter (never dst arg for struct literal)
	if !f.IsReceiverMethod() {
		// "func Name(src *SrcModel"
		sb.WriteString(f.Src.Name)
		sb.WriteString(" ")
		sb.WriteString(f.Src.FullType())
	}

	// Add additional args
	for _, args := range f.AdditionalArgs {
		fullType := args.FullType()

		if strings.Contains(args.Type, "/") {
			re := regexp.MustCompile(`^([^a-zA-Z0-9]*)([a-zA-Z0-9].*/)(.+)$`)
			fullType = re.ReplaceAllString(fullType, "$1$3")
		}

		sb.WriteString(", ")
		sb.WriteString(args.Name)
		sb.WriteString(" ")
		sb.WriteString(fullType)
	}

	// "func Name(src *SrcModel)"
	sb.WriteString(") ")

	// Write return signature for struct literal style
	g.writeStructLiteralReturnSignature(f, &sb)

	if f.PreProcess != nil {
		sb.WriteString(g.ManipulatorToString(f.PreProcess, f.Src, f.Dst, f.AdditionalArgs))
	}

	// Generate the struct literal return statement
	g.writeStructLiteralReturn(f, &sb)

	if f.PostProcess != nil {
		sb.WriteString(g.ManipulatorToString(f.PostProcess, f.Src, f.Dst, f.AdditionalArgs))
	}

	sb.WriteString("}\n\n")

	return sb.String()
}

// writeStructLiteralReturnSignature writes the return signature for struct literal functions.
func (g *Generator) writeStructLiteralReturnSignature(f *model.Function, sb *strings.Builder) {
	// "func Name(src *SrcModel) (dst DstModel"
	sb.WriteString("(")
	sb.WriteString(f.Dst.Name)
	sb.WriteString(" ")
	sb.WriteString(f.Dst.FullType())

	if f.RetError {
		// "func Name(src *SrcModel) (dst DstModel, err error"
		sb.WriteString(", err error")
	}

	// "func Name(src *SrcModel) (dst DstModel) {\n"
	sb.WriteString(") {\n")
}

// writeStructLiteralReturn writes the struct literal return statement.
func (g *Generator) writeStructLiteralReturn(f *model.Function, sb *strings.Builder) {
	sb.WriteString("\treturn ")

	// Handle pointer types
	if f.Dst.Pointer {
		sb.WriteString("&")
	}

	// Write the struct literal
	sb.WriteString(f.Dst.PtrLessFullType())
	sb.WriteString("{\n")

	// Generate field assignments for the struct literal
	for _, assignment := range f.Assignments {
		if g.shouldIncludeInStructLiteral(assignment) {
			g.writeStructLiteralAssignment(assignment, sb)
		} else {
			// Write skip/nomatch comments outside the struct literal
			g.writeStructLiteralComment(assignment, sb)
		}
	}

	sb.WriteString("\t}\n")
}

// shouldIncludeInStructLiteral determines if an assignment should be included in the struct literal.
func (g *Generator) shouldIncludeInStructLiteral(assignment model.Assignment) bool {
	switch assignment.(type) {
	case model.SkipField, model.NoMatchField:
		return false // These become comments outside the struct literal
	case model.SimpleField:
		return !assignment.RetError() // Only include simple fields without errors
	default:
		return false // Complex assignments are not supported in struct literals
	}
}

// writeStructLiteralAssignment writes a field assignment within the struct literal.
func (g *Generator) writeStructLiteralAssignment(assignment model.Assignment, sb *strings.Builder) {
	if simpleField, ok := assignment.(model.SimpleField); ok {
		sb.WriteString("\t\t")
		sb.WriteString(simpleField.LHS)
		sb.WriteString(": ")
		sb.WriteString(simpleField.RHS)
		sb.WriteString(",\n")
	}
}

// writeStructLiteralComment writes a comment for skip/nomatch fields outside the struct literal.
func (g *Generator) writeStructLiteralComment(assignment model.Assignment, sb *strings.Builder) {
	switch assign := assignment.(type) {
	case model.SkipField:
		sb.WriteString("\t// skip: ")
		sb.WriteString(assign.LHS)
		sb.WriteString("\n")
	case model.NoMatchField:
		sb.WriteString("\t// no match: ")
		sb.WriteString(assign.LHS)
		sb.WriteString("\n")
	}
}
