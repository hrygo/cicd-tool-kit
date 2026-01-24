# Testing

## Running Tests

### All Tests

```bash
go test ./...
```

### Specific Package

```bash
go test ./pkg/runner
```

### With Coverage

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Structure

```
test/
├── unit/              # Unit tests (alongside source code)
├── integration/       # Integration tests
├── e2e/              # End-to-end tests
└── fixtures/         # Test data
    ├── diffs/        # Sample diffs
    └── expected/     # Expected results
```

## Unit Tests

Unit tests are placed alongside source code:

```go
// pkg/runner/lifecycle_test.go
package runner

func TestRunnerBootstrap(t *testing.T) {
    r := New()
    err := r.Bootstrap(context.Background())
    assert.NoError(t, err)
}
```

## Integration Tests

```bash
# Requires environment setup
export INTEGRATION_TEST=true
go test ./test/integration/...
```

## Writing Tests

1. **Arrange**: Set up test data
2. **Act**: Execute the code under test
3. **Assert**: Verify expected behavior

```go
func TestSkillExecution(t *testing.T) {
    // Arrange
    skill := &Skill{Name: "test"}
    input := "test input"

    // Act
    result, err := skill.Execute(context.Background(), input)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```
