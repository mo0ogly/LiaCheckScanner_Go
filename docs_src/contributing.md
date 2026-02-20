# Contributing

Thank you for your interest in contributing to LiaCheckScanner Go. This page covers the workflow, standards, and expectations for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)
- [Feature Requests](#feature-requests)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a new branch for your feature
4. Make your changes
5. Test your changes
6. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional, for using Makefile commands)
- Platform-specific Fyne/OpenGL dependencies (see [Installation](installation.md))

### Local Development

1. Clone the repository:

    ```bash
    git clone https://github.com/your-username/LiaCheckScanner_Go.git
    cd LiaCheckScanner_Go
    ```

2. Install dependencies:

    ```bash
    go mod download
    ```

3. Run tests:

    ```bash
    go test ./...
    ```

4. Build the application:

    ```bash
    go build -o build/liacheckscanner ./cmd/liacheckscanner
    ```

## Coding Standards

### Go Code Style

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Use `golint` to check for style issues
- Use `go vet` to check for common mistakes

### Code Organization

- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for complex logic
- Follow the existing project structure

### Error Handling

- Always check for errors
- Return errors instead of panicking
- Use meaningful error messages
- Log errors appropriately

### Documentation

- Add doc comments to all exported functions
- Update documentation when adding new features
- Include examples where helpful

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
go test -bench=. ./...
```

### Writing Tests

- Write tests for all new functionality
- Aim for high test coverage
- Use descriptive test names
- Test both success and failure cases
- Use table-driven tests when appropriate

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result := FunctionName(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

## Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** following the coding standards
3. **Write tests** for new functionality
4. **Update documentation** as needed
5. **Run tests** to ensure everything works
6. **Commit your changes** with clear commit messages
7. **Push to your fork** and create a pull request

### Commit Message Format

Use conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

Examples:

- `feat(gui): add pagination to data table`
- `fix(extractor): handle empty IP addresses`
- `docs(readme): update installation instructions`

### Pull Request Guidelines

- Provide a clear description of the changes
- Include any relevant issue numbers
- Add screenshots for UI changes
- Ensure all tests pass
- Update documentation if needed

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

1. **Environment information** -- operating system and version, Go version, application version
2. **Steps to reproduce** -- clear, step-by-step instructions with sample data if applicable
3. **Expected vs actual behavior** -- what you expected to happen and what actually happened
4. **Additional information** -- error messages, logs, or screenshots

### Issue Templates

Use the provided issue templates when creating new issues.

## Feature Requests

When requesting new features:

1. **Describe the feature** clearly
2. **Explain the use case** and why it is needed
3. **Provide examples** of how it would work
4. **Consider implementation** complexity

## Getting Help

- Check the [documentation](index.md) for detailed information
- Look through existing issues and pull requests
- Create a new issue for questions or problems

## License

By contributing to LiaCheckScanner Go, you agree that your contributions will be licensed under the MIT License.
