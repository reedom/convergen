# Design

This document outlines the design of the `pkg/parser` package. The package is responsible for parsing the Go source code and extracting the information needed to generate the conversion functions.

## Architecture

The `pkg/parser` package is composed of the following components:

*   **`Parser`:** This is the main entry point for parsing the source code. It takes the path to the source file and returns a list of `model.MethodsInfo` objects.
*   **`intfEntry`:** This struct represents a `Convergen` interface.
*   **`comment.go`:** This file contains the logic for parsing the notations in the comments.
*   **`interface.go`:** This file contains the logic for finding and parsing the `Convergen` interfaces.
*   **`method.go`:** This file contains the logic for parsing the methods of the `Convergen` interfaces.

## `Parser`

The `Parser` is responsible for parsing the source code. It has the following methods:

*   **`Parse()`:** This method parses the source code and returns a list of `model.MethodsInfo` objects.
*   **`CreateBuilder()`:** This method creates a new `builder.FunctionBuilder`.
*   **`GenerateBaseCode()`:** This method generates the base code for the conversion functions.

## `intfEntry`

The `intfEntry` struct represents a `Convergen` interface. It contains the following information:

*   **`intf`:** The `types.Object` of the interface.
*   **`opts`:** The `option.Options` of the interface.
*   **`marker`:** The marker of the interface.
