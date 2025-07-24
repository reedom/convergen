# Contributing to Convergen

First off, thank you for considering contributing to Convergen. Your help is greatly appreciated. This document outlines the coding conventions and best practices to follow when contributing to this project.

## 1. Code Conventions

### Comparison Operators

To maintain consistency and readability, all comparisons should follow the model of a number line. This means:

-   **Use `<` and `<=`**: Always prefer the "less than" and "less than or equal to" operators.
-   **Avoid `>` and `>=`**: Do not use the "greater than" and "greater than or equal to" operators. Re-order the expression to use `<` or `<=` instead.

**Example:**

```go
// Good
if i < 10 {
    // ...
}

// Bad
if 10 > i {
    // ...
}
```

## 2. Effective Go Best Practices

This project adheres to the principles outlined in [Effective Go](https://go.dev/doc/effective_go). Below is a summary of the most important points to keep in mind.

### Formatting

-   **`gofmt`**: All code must be formatted with `gofmt` before committing. This is the single source of truth for formatting.

### Naming

-   **Clarity and Brevity**: Names should be short, descriptive, and meaningful.
-   **`MixedCaps`**: Use `MixedCaps` for multi-word names (e.g., `MyType`, `myVariable`). Do not use underscores.
-   **Package Names**: Package names should be short, lowercase, and single-word.
-   **Exported vs. Unexported**: A name is exported (public) if it begins with a capital letter. Otherwise, it is unexported (private to the package).

### Simplicity and Clarity

-   **Readability First**: Write simple, clear, and straightforward code. Avoid overly clever or obscure solutions.
-   **Comments**: Use comments to explain *why* something is done, not *what* is being done. Explain complex logic or the reasoning behind a particular implementation choice.

### Concurrency

-   **Share by Communicating**: Prefer using channels to communicate between goroutines rather than sharing memory and using locks.
-   **Goroutines**: Keep goroutines lightweight and focused on a single task.

### Error Handling

-   **Explicit Errors**: Functions that can fail must return an `error` as their last return value.
-   **No Exceptions**: Do not use `panic` for normal error handling. `panic` should only be used for truly exceptional situations where the program cannot continue.
-   **Check Every Error**: Always check the returned error value from a function call.

### Interfaces

-   **Implicit Satisfaction**: A type satisfies an interface by implementing its methods. No explicit `implements` declaration is needed.
-   **Define Interfaces Where They Are Used**: It is a best practice for the package that *consumes* an interface to be the one that defines it.

### Control Structures

-   **`defer`**: Use `defer` to schedule cleanup tasks like closing files or releasing resources. It ensures the call is executed just before the function returns.
-   **Semicolons**: The Go lexer automatically inserts semicolons. You should not need to use them in your code.

By following these guidelines, we can ensure that the Convergen codebase remains clean, consistent, and easy to work with for everyone.
