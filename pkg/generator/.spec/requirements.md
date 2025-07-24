# Requirements

This document outlines the requirements for the `pkg/generator` package. The package is responsible for generating the Go code for the `convergen` tool.

## Functional Requirements

The `pkg/generator` package MUST:

*   **REQ-1: Generate function signatures:** The package MUST be able to generate function signatures, including the function name, receiver, arguments, and return values.
*   **REQ-2: Generate function bodies:** The package MUST be able to generate function bodies, including variable declarations, assignment statements, and return statements.
*   **REQ-3: Support different destination variable styles:** The package MUST support two styles of destination variables: `return` and `arg`.
*   **REQ-4: Support error handling:** The package MUST be able to generate code that handles errors returned by function calls.
*   **REQ-5: Support pre- and post-processing:** The package MUST be able to generate code that calls pre- and post-processing functions.
*   **REQ-6: Format generated code:** The package MUST format the generated code using `goimports` and `gofmt`.
*   **REQ-7: Write generated code to a file:** The package MUST be able to write the generated code to a file.
