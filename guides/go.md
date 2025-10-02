# Go Best Practices Guide

This guide outlines best practices for writing Go code in this project. The goal is to maintain a clean, readable, and maintainable codebase.

## Project Structure

- **Directory per Feature**: Organize code into directories based on features or components. This helps in modularity and separation of concerns. For example, all code related to git operations should be in an `internal/git` package.

## Dependencies and Interfaces

- **Dependency Inversion**: Depend on abstractions (interfaces), not on concrete implementations. This makes code more flexible, testable, and helps avoid circular dependencies.
- **Avoid Circular Dependencies**: A package `A` should not import package `B` if package `B` imports `A`. Using interfaces and proper package organization helps prevent this.
- **Small Interfaces**: Prefer small, single-method interfaces (like `io.Reader`). This follows the Interface Segregation Principle.

## Error Handling

- **Error Wrapping**: When returning an error from a downstream function call, wrap it with additional context using `fmt.Errorf("...: %w", err)`. This preserves the original error and adds context for debugging.

## General Code Style

- **Clarity over Brevity**: Write code that is easy to understand. While Go has many idiomatic shorthands, prioritize readability for other developers.
- **Keep Functions Small**: Functions should be small and do one thing well. This improves readability and testability.
- **Use Table-Driven Tests**: For functions with multiple scenarios, use table-driven tests to keep test code concise and extensible.