# Convergen Codebase Design Overview

This document provides a comprehensive overview of the `convergen` codebase, its architecture, and the data flow. It is intended to serve as a guide for future development and refactoring efforts.

## 1. Overall Architecture and Data Flow

The `convergen` tool operates in a pipeline architecture, where the output of one stage becomes the input for the next. The process can be summarized as follows:

1.  **Runner (`pkg/runner`):** The entry point of the application. It orchestrates the entire process, initializing and calling the other components in sequence.
2.  **Parser (`pkg/parser`):** Reads the target Go source file, parses its Abstract Syntax Tree (AST), and identifies `Convergen` interfaces and their methods. It extracts all `//:` notations and populates configuration objects.
3.  **Builder (`pkg/builder`):** Takes the parsed information and builds an intermediate, abstract representation of the functions to be generated. This is where the core logic for field mapping, type conversion, and assignment generation resides.
4.  **Generator (`pkg/generator`):** Takes the intermediate representation from the builder and generates the final Go source code as a string. It handles the final formatting and file output.

The data flows through these stages:
`Source File` -> **Parser** -> `MethodsInfo[]` -> **Builder** -> `gmodel.Function[]` -> **Generator** -> `Generated Go File`

---

## 2. Package Breakdown

### `pkg/runner`
*   **Responsibility:** Main application driver.
*   **Key Logic:**
    *   Initializes configuration (`config.Config`).
    *   Sets up the logger (`pkg/logger`).
    *   Instantiates the `parser.Parser`.
    *   Calls the parser to get method definitions (`p.Parse()`).
    *   Instantiates the `builder.FunctionBuilder`.
    *   Calls the builder to create function models (`builder.CreateFunctions()`).
    *   Gets the base source code (with `Convergen` interfaces removed) from the parser (`p.GenerateBaseCode()`).
    *   Instantiates the `generator.Generator`.
    *   Calls the generator to produce the final output file (`g.Generate()`).

### `pkg/parser`
*   **Responsibility:** Source code analysis and notation parsing.
*   **Key Logic:**
    *   Uses `golang.org/x/tools/go/packages` to load the source code, its AST, and type information.
    *   **`interface.go`**: Scans the AST for `type ... interface` declarations that are either named `Convergen` or have a `// :convergen` comment.
    *   **`method.go`**: For each found interface, it parses the method signatures.
    *   **`comment.go`**: Parses `//:` notations from the doc comments of interfaces and methods. It uses regular expressions to extract commands and arguments.
    *   It resolves types and looks up functions specified in `:conv`, `:preprocess`, and `:postprocess` notations.
*   **Key Output:** A slice of `model.MethodsInfo` objects, which contain the parsed methods and their associated `option.Options`.

### `pkg/option`
*   **Responsibility:** Defines all configuration structures.
*   **Key Logic:**
    *   `Options`: A central struct holding all configuration flags for a given method (e.g., `Style`, `Rule`, `ExactCase`).
    *   `*Matcher` (`NameMatcher`, `PatternMatcher`, `IdentMatcher`): Structs that handle the logic for matching field names, patterns, and identifiers based on the parsed notations.
    *   `*Setter`/`*Converter` (`LiteralSetter`, `FieldConverter`): Structs that hold the details for custom assignment rules.

### `pkg/builder`
*   **Responsibility:** Translates the parsed information into a concrete, high-level representation of the functions to be generated.
*   **Key Logic:**
    *   `FunctionBuilder`: The main entry point. It iterates through `model.MethodEntry` from the parser.
    *   `assignmentBuilder`: The core component that determines how to assign a source field to a destination field. It handles struct-to-struct, slice-to-slice, and primitive assignments.
    *   `model/node.go`: Defines the `Node` interface and its implementations (`StructFieldNode`, `ConverterNode`, etc.). This forms an expression tree for the right-hand side (RHS) of an assignment, allowing for complex chains like `src.User.GetStatus().String()`.
*   **Key Output:** A slice of `gmodel.Function` objects.

### `pkg/generator`
*   **Responsibility:** Generates the final Go code from the high-level representation.
*   **Key Logic:**
    *   `Generator`: The main struct. Its `Generate` method orchestrates the final steps.
    *   `function.go` (`FuncToString`): Contains the logic to build the function signature and body as a string from a `gmodel.Function`.
    *   `assignment.go` (`AssignmentToString`): Contains the logic to convert different `gmodel.Assignment` types (like `SimpleField`, `NestStruct`) into their corresponding Go code strings.
    *   The generator replaces markers in the base code with the generated function strings.
    *   It uses `go/format` and `golang.org/x/tools/imports` to format the final output beautifully.

### `pkg/util` & `pkg/logger`
*   **`pkg/util`**: A collection of helper functions for working with the Go AST (`ast.go`), types (`types.go`), and imports (`import.go`). This is a crucial support package.
*   **`pkg/logger`**: A simple configurable logger.

---

## 3. Future Refactoring Plan (`pkg/builder`)

The current implementation in `pkg/builder`, specifically within `assignmentBuilder`, is functional but complex. The logic for deciding which assignment rule to apply (e.g., `:map`, `:conv`, name match) is intertwined.

**Proposed Enhancement: Chain of Responsibility Pattern**

To improve modularity and extensibility, the `assignmentBuilder` will be refactored to use the **Chain of Responsibility** design pattern.

1.  **`AssignmentHandler` Interface:** An interface will be defined:
    ```go
    type AssignmentHandler interface {
        Handle(lhs, rhs bmodel.Node) (gmodel.Assignment, error)
        SetNext(handler AssignmentHandler)
    }
    ```

2.  **Concrete Handlers:** A series of handlers will be created, each responsible for one specific matching strategy:
    *   `SkipHandler`: Checks for `:skip`.
    *   `LiteralSetterHandler`: Checks for `:literal`.
    *   `ConverterHandler`: Checks for `:conv`.
    *   `NameMapperHandler`: Checks for `:map`.
    *   `StructFieldMatchHandler`: Handles the default name/getter matching.
    *   `SliceHandler`: Handles slice-to-slice assignments.
    *   ... and so on.

3.  **Handler Chain:** These handlers will be linked in a chain. The `assignmentBuilder` will simply invoke the first handler in the chain. Each handler will attempt to process the assignment. If it cannot, it will pass the request to the next handler in the chain.

This approach will make the code cleaner, easier to test, and significantly easier to extend with new matching rules in the future without modifying existing logic.