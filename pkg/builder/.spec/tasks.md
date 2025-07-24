# `pkg/builder` Refactoring Tasks

This document outlines the specific tasks required to refactor the `pkg/builder` package. The goal is to replace the current monolithic assignment logic with a more modular and extensible Chain of Responsibility pattern.

### Task 1: Establish the Handler Framework

-   **1.1. Create `pkg/builder/handler.go`:** This new file will house the core handler interface.
-   **1.2. Define `AssignmentHandler` interface:** The interface will be defined as follows:
    ```go
    type AssignmentHandler interface {
        SetNext(handler AssignmentHandler)
        Handle(lhs, rhs bmodel.Node, additionalArgs []bmodel.Node) (gmodel.Assignment, error)
    }
    ```
-   **1.3. Create a base handler struct:** Implement a small, embeddable struct that provides a default implementation for `SetNext` to reduce boilerplate in concrete handlers.

### Task 2: Implement Concrete Handlers

For each of the following, create a new struct that implements the `AssignmentHandler` interface. Each handler should focus *only* on its specific logic.

-   **2.1. `SkipHandler`:**
    -   Logic: Checks `opts.ShouldSkip()`.
    -   Output: Returns a `gmodel.SkipField` assignment if the field should be skipped.

-   **2.2. `LiteralSetterHandler`:**
    -   Logic: Iterates through `opts.Literals` and checks for a match.
    -   Output: Returns a `gmodel.SimpleField` with the literal value.

-   **2.3. `ConverterHandler`:**
    -   Logic: Iterates through `opts.Converters` and checks for a match.
    -   Output: Returns a `gmodel.SimpleField` with the converter function call.

-   **2.4. `NameMapperHandler`:**
    -   Logic: Iterates through `opts.NameMapper` and `opts.TemplatedNameMapper` and checks for a match.
    -   Output: Returns a `gmodel.SimpleField` with the mapped field.

-   **2.5. `SliceHandler`:**
    -   Logic: Checks if both LHS and RHS are slice types.
    -   Output: Returns a `gmodel.SliceAssignment`, `gmodel.SliceLoopAssignment`, or `gmodel.SliceTypecastAssignment`.

-   **2.6. `StructFieldMatchHandler`:**
    -   Logic: This will contain the existing logic for direct field-to-field or getter-to-field name matching.
    -   Output: Returns a `gmodel.SimpleField` for direct assignments or a `gmodel.NestStruct` for nested struct assignments.

### Task 3: Integrate the Handler Chain

-   **3.1. Modify `assignmentBuilder.build`:**
    -   Remove the direct call to `dispatch`.
    -   Instantiate and assemble the chain of handlers in the correct order of precedence (e.g., `SkipHandler` -> `LiteralSetterHandler` -> `ConverterHandler` -> ...).
-   **3.2. Modify `assignmentBuilder.structToStruct`:**
    -   Instead of calling `matchStructFieldAndStruct`, this method will now iterate through the LHS fields and invoke the handler chain for each field.
-   **3.3. Deprecate `matchStructFieldAndStruct` and `dispatch`:** Once the handler chain is fully integrated, these complex methods can be removed.

### Task 4: Testing and Verification

-   **4.1. Create `pkg/builder/handler_test.go`:**
    -   Add new unit tests for each individual handler to verify its logic in isolation.
-   **4.2. Run Existing Tests:**
    -   Execute the full test suite for the `pkg/builder` package to ensure that the refactoring has not introduced any regressions in functionality.
