# Design

This document outlines the design of the `pkg/generator` package. The package is responsible for generating the Go code for the `convergen` tool.

## Architecture

The `pkg/generator` package is composed of the following components:

*   **`Generator`:** This is the main entry point for generating code. It takes a `model.Code` and generates the Go code.
*   **`model.Code`:** This is the data model that represents the code to be generated.
*   **String Functions:** A set of functions for converting the `model` objects to their string representation in Go code.

## `Generator`

The `Generator` is responsible for generating the Go code. It uses the `model.Code` to generate the code. The `Generator` has the following methods:

*   **`Generate()`:** This method generates the Go code and writes it to a file.

## `model.Code`

The `model.Code` is the data model that represents the code to be generated. It contains the following information:

*   **`BaseCode`:** The base code to which the generated functions will be added.
*   **`FunctionBlocks`:** A list of function blocks to be generated.

## String Functions

A set of functions are used to convert the `model` objects to their string representation in Go code. These functions are:

*   **`FuncToString()`:** This function converts a `model.Function` to its string representation.
*   **`AssignmentToString()`:** This function converts a `model.Assignment` to its string representation.
*   **`ManipulatorToString()`:** This function converts a `model.Manipulator` to its string representation.
