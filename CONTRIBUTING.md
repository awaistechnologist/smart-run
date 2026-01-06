# Contributing to SmartRun

Thank you for your interest in contributing to SmartRun! This document provides guidelines for contributing to the project.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
   ```bash
   git clone https://github.com/YOUR_USERNAME/smart-run.git
   cd smart-run
   ```
3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- A text editor or IDE

### Building the Project

```bash
# Build the CLI
go build -o smart-run ./cmd/smart-run

# Build the server
go build -o smartrund ./cmd/smartrund

# Run tests
go test ./...

# Run specific package tests
go test ./internal/engine
```

### Running Locally

```bash
# Run the web server
./smartrund --port 8080

# Or use the quick start script
scripts/start.sh
```

## Code Guidelines

### Go Code Style

- Follow standard Go conventions and formatting
- Run `go fmt` before committing
- Run `go vet` to catch common mistakes
- Use meaningful variable and function names
- Add comments for exported functions and types

### Project Structure

```
smart-run/
├── cmd/                  # Command-line applications
│   ├── smart-run/       # CLI tool
│   └── smartrund/       # Web server
├── internal/            # Internal packages (not importable)
│   ├── engine/         # Core scheduling algorithm
│   ├── prices/         # Price fetching (Octopus API)
│   ├── weather/        # Weather fetching (Open-Meteo)
│   ├── store/          # Database layer (SQLite)
│   └── uiapi/          # HTTP API server
├── web/                # Frontend files
│   ├── index.html
│   └── static/
└── config.example.yaml # Example configuration
```

### Making Changes

1. **Small, focused commits** - Each commit should do one thing
2. **Descriptive commit messages** - Explain what and why, not just what
3. **Test your changes** - Make sure existing tests pass and add new tests if needed
4. **Update documentation** - If you change behavior, update README.md

### Commit Message Format

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem this commit solves and why you chose
this particular solution.

Fixes #123
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./internal/engine
```

### Writing Tests

- Place tests in the same package as the code they test
- Name test files `*_test.go`
- Use table-driven tests when appropriate
- Test edge cases and error conditions

Example:
```go
func TestBestWindows(t *testing.T) {
    tests := []struct {
        name    string
        input   []PriceSlot
        want    []Recommendation
        wantErr bool
    }{
        // Test cases here
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := BestWindows(tt.input, ...)
            // Assertions here
        })
    }
}
```

## Pull Request Process

1. **Update documentation** if needed (README.md, code comments)
2. **Run all tests** and ensure they pass
3. **Format your code** with `go fmt`
4. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```
5. **Create a Pull Request** on GitHub
6. **Describe your changes** in the PR description:
   - What problem does this solve?
   - How did you solve it?
   - Any breaking changes?
   - Related issues?

### PR Checklist

- [ ] Code follows Go conventions and is properly formatted
- [ ] Tests pass (`go test ./...`)
- [ ] New code has tests
- [ ] Documentation updated (if needed)
- [ ] Commit messages are clear and descriptive
- [ ] No sensitive data (API keys, personal info, etc.)

## Areas for Contribution

### Good First Issues

- Documentation improvements
- Bug fixes
- UI/UX enhancements
- Test coverage improvements

### Feature Ideas

- Support for additional energy providers
- Mobile app
- Smart home integration (Home Assistant, etc.)
- Additional scheduling algorithms
- Carbon intensity optimization
- Cost prediction and analytics

### Bug Reports

When reporting bugs, please include:

1. **Description** - What happened?
2. **Expected behavior** - What should have happened?
3. **Steps to reproduce** - How can we reproduce it?
4. **Environment** - OS, Go version, etc.
5. **Logs/Screenshots** - If applicable

## Code Review Process

1. Maintainers will review your PR
2. They may suggest changes or improvements
3. Make requested changes and push updates
4. Once approved, a maintainer will merge your PR

## Privacy and Security

**IMPORTANT:** Do not commit:
- API keys or secrets
- Personal data (addresses, coordinates, names)
- Database files (*.db)
- Configuration files with personal settings

These are already in `.gitignore`, but please verify before committing.

## Questions?

- Open an issue for discussion
- Check existing issues and PRs
- Review the README.md for project overview

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Thank You!

Your contributions make SmartRun better for everyone. We appreciate your time and effort!
