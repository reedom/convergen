# Refactoring Plan for Convergen

This document outlines the tasks for the planned refactoring of the `convergen` codebase. The primary goal is to improve modularity, maintainability, and extensibility, starting with the most complex component.

## Phase 1: Refactor the `pkg/builder` Package

This is the core of the refactoring effort. The goal is to replace the complex, monolithic logic in `assignmentBuilder` with a more flexible Chain of Responsibility pattern.

-   **Task 1.1: Define the `AssignmentHandler` Interface**
    -   Create a new file `pkg/builder/handler.go`.
    -   Define the `AssignmentHandler` interface with a `Handle` method and a `SetNext` method.

-   **Task 1.2: Implement Concrete `AssignmentHandler`s**
    -   Create handlers for each of the existing assignment strategies. Each handler will be a separate struct implementing the `AssignmentHandler` interface.
        -   `SkipHandler` (for `:skip`)
        -   `LiteralSetterHandler` (for `:literal`)
        -   `ConverterHandler` (for `:conv`)
        -   `NameMapperHandler` (for `:map`)
        -   `StructFieldMatchHandler` (for default name matching)
        -   `SliceHandler` (for slice assignments)

-   **Task 1.3: Integrate the Handler Chain**
    -   Modify `assignmentBuilder` to create and chain the handlers together.
    -   Replace the large `if/else` or `switch` blocks in `matchStructFieldAndStruct` with a single call to the first handler in the chain.

-   **Task 1.4: Verify Functionality**
    -   Ensure all existing tests for the `builder` package pass after the refactoring.
    -   Add new unit tests for each individual handler to ensure they work in isolation.

## Phase 2: Review and Potentially Refactor Dependent Packages

After the `builder` is refactored, review the packages that interact with it to see if they can be simplified.

-   **Task 2.1: Review `pkg/generator`**
    -   Analyze the `gmodel.Function` and `gmodel.Assignment` structures.
    -   Determine if the new, cleaner logic in the `builder` allows for a simpler or more expressive model to be passed to the generator.
    -   If so, refactor the generator to take advantage of the improved model.

-   **Task 2.2: Review `pkg/runner` and `pkg/parser`**
    -   Ensure that the integration with the newly refactored `builder` is clean.
    -   Make any necessary adjustments to the `runner`'s orchestration logic.
    -   The `parser` should require minimal to no changes, but its interaction with the `builder` should be double-checked.

## Phase 3: Final Integration and Testing

-   **Task 3.1: End-to-End Testing**
    -   Run the full test suite for the entire application to ensure that the refactoring has not introduced any regressions.
    -   Manually run the generator on all existing use cases in the `tests/fixtures` directory to confirm the output is identical to the pre-refactoring output.
