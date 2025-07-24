# Requirements

This document outlines the requirements for the `pkg/parser` package. The package is responsible for parsing the Go source code and extracting the information needed to generate the conversion functions.

## Functional Requirements

The `pkg/parser` package MUST:

*   **REQ-1: Parse `Convergen` interfaces:** The package MUST be able to find and parse `Convergen` interfaces in the source code.
*   **REQ-2: Parse method signatures:** The package MUST be able to parse the method signatures of the `Convergen` interfaces.
*   **REQ-3: Parse notations:** The package MUST be able to parse the notations in the comments of the `Convergen` interfaces and methods.
*   **REQ-4: Resolve types:** The package MUST be able to resolve the types of the arguments and return values of the methods.
*   **REQ-5: Resolve converters:** The package MUST be able to resolve the converter functions.
*   **REQ-6: Resolve manipulators:** The package MUST be able to resolve the manipulator functions.
*   **REQ-7: Generate base code:** The package MUST be able to generate the base code for the conversion functions.
