# Requirements

This document outlines the requirements for the `pkg/builder` package. The package is responsible for building the code generation logic for the `convergen` tool.

## Functional Requirements

The `pkg/builder` package MUST:

*   **REQ-1: Build assignment statements:** The package MUST be able to build assignment statements between two variables.
*   **REQ-2: Handle struct-to-struct assignments:** The package MUST be able to handle assignments between two structs.
*   **REQ-3: Support field converters:** The package MUST support custom field converters.
*   **REQ-4: Support name mappers:** The package MUST support custom name mappers.
*   **REQ-5: Support literal setters:** The package MUST support setting fields to literal values.
*   **REQ-6: Handle slice-to-slice assignments:** The package MUST be able to handle assignments between two slices.
*   **REQ-7: Support type casting:** The package MUST support type casting between assignable types.
*   **REQ-8: Support `stringer` interface:** The package MUST support the `stringer` interface for converting types to strings.
*   **REQ-9: Handle nested structs:** The package MUST be able to handle assignments between nested structs.
*   **REQ-10: Support for additional arguments:** The package MUST support additional arguments in the generated functions.
*   **REQ-11: Support for post-processing:** The package MUST support post-processing of the generated code.
*   **REQ-12: Support for pre-processing:** The package MUST support pre-processing of the generated code.
*   **REQ-13: Support for reversing the direction of the conversion:** The package MUST support reversing the direction of the conversion.

## Non-Functional Requirements

*   **REQ-14: Extensibility:** The package MUST be designed in a way that allows for new types of assignments and matching rules to be added with minimal changes to the existing code.
