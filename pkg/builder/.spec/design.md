# Design

This document outlines the design of the `pkg/builder` package. The package is responsible for building the code generation logic for the `convergen` tool.

## Architecture

The `pkg/builder` package is composed of the following components:

*   **`FunctionBuilder`:** This is the main entry point for building a function. It takes a `MethodEntry` and returns a `gmodel.Function`.
*   **`assignmentBuilder`:** This component is responsible for building a single assignment between two variables.
*   **`bmodel.Node`:** This is an interface that represents a node in the expression tree. There are several implementations of this interface, such as `RootNode`, `ScalarNode`, `StructFieldNode`, `StructMethodNode`, `ConverterNode`, `TypecastEntry`, and `StringerEntry`.

## `assignmentBuilder`

The `assignmentBuilder` is responsible for building a single assignment between two variables. It uses a dispatch mechanism to determine the type of assignment to generate. The following assignment types are supported:

*   **Struct-to-struct:** This is the most common type of assignment. The `assignmentBuilder` iterates over the fields of the destination struct and tries to find a matching field in the source struct. If a match is found, it generates an assignment statement.
*   **Slice-to-slice:** This type of assignment is used to copy elements from a source slice to a destination slice.

## `bmodel.Node`

The `bmodel.Node` interface represents a node in the expression tree. The expression tree is used to represent the right-hand side of an assignment. For example, the expression `dst.User.Name` would be represented by a tree of `StructFieldNode`s.

The `bmodel.Node` interface has the following methods:

*   **`AssignExpr()`:** This method returns the expression to be used on the right-hand side of the assignment.
*   **`ExprType()`:** This method returns the type of the expression.
*   **`ReturnsError()`:** This method returns `true` if the expression returns an error.

## Field Matching

The `assignmentBuilder` uses a chain of responsibility pattern to match fields. The chain is composed of a series of `AssignmentHandler`s. Each handler is responsible for a specific matching strategy. The following handlers are used:

*   **`LiteralSetterHandler`:** This handler is responsible for handling literal setters.
*   **`ConverterHandler`:** This handler is responsible for handling field converters.
*   **`NameMapperHandler`:** This handler is responsible for handling name mappers.
*   **`StructFieldMatchHandler`:** This handler is responsible for matching fields by name.

If a handler can handle the assignment, it returns an `Assignment`. Otherwise, it returns `nil` and the assignment is passed to the next handler in the chain.
