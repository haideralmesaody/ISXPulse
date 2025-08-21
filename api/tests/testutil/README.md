# Test Utilities

Common test helpers and utilities used across all test suites.

## Components

- **fixtures/** - Test data fixtures (JSON, CSV files)
- **mocks/** - Generated mocks for interfaces
- **helpers.go** - Common test helper functions
- **assertions.go** - Custom assertion functions
- **context.go** - Test context with tracing

## Mock Generation

Generate mocks for interfaces:

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate mocks
mockgen -source=internal/services/interfaces.go -destination=test/testutil/mocks/services.go
```

## Test Helpers

```go
// Create test context with trace ID
ctx := testutil.TestContext(t)

// Load fixture data
data := testutil.LoadFixture(t, "reports/sample.json")

// Assert error types
testutil.AssertErrorType(t, err, &errors.AppError{})

// Create test logger
logger := testutil.TestLogger(t)
```

## Best Practices

1. Keep test utilities generic and reusable
2. Document complex test setups
3. Use table-driven tests with fixtures
4. Provide cleanup functions for all resources