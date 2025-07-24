# Requirements

This document outlines the requirements for the `pkg/parser` package. The package is responsible for parsing Go source code to extract the information needed for generating conversion functions.

## Functional Requirements

The `pkg/parser` package MUST:

*   **REQ-1: Parse `Convergen` interfaces:** The package MUST be able to find and parse interfaces named `Convergen` or interfaces annotated with `// :convergen`.
*   **REQ-2: Parse method signatures:** The package MUST be able to parse the method signatures within the `Convergen` interfaces.
*   **REQ-3: Parse notations:** The package MUST parse the following notations from the doc comments of interfaces and methods to configure the generation process:
    *   `:match <name|none>`: Sets the field matching algorithm.
    *   `:style <return|arg>`: Sets the destination variable style.
    *   `:recv <var>`: Specifies the source as a receiver.
    *   `:reverse`: Reverses the copy direction.
    *   `:case` / `:case:off`: Toggles case-sensitive name matching.
    *   `:getter` / `:getter:off`: Toggles the use of getter methods for field matching.
    *   `:stringer` / `:stringer:off`: Toggles the use of `String()` methods for `string` conversions.
    *   `:typecast` / `:typecast:off`: Toggles automatic type casting for assignable types.
    *   `:skip <dst field pattern>`: Skips copying to a specified destination field. Supports regex.
    *   `:map <src> <dst field>`: Defines an explicit mapping between a source and a destination.
    *   `:conv <func> <src> [dst field]`: Uses a custom function to convert and assign a value.
    *   `:literal <dst> <literal>`: Assigns a literal value to a destination field.
    *   `:preprocess <func>`: Specifies a function to call before the conversion logic.
    *   `:postprocess <func>`: Specifies a function to call after the conversion logic.
*   **REQ-4: Resolve types:** The package MUST be able to resolve the types of method arguments, return values, and types referenced in notations.
*   **REQ-5: Resolve converters:** The package MUST be able to look up and validate functions specified in `:conv` notations.
*   **REQ-6: Resolve manipulators:** The package MUST be able to look up and validate functions specified in `:preprocess` and `:postprocess` notations.
*   **REQ-7: Generate base code:** The package MUST be able to produce a version of the source code with all `Convergen` interfaces and related comments removed, replacing them with markers for the generator.
