# Integration Tests

End-to-end integration tests for the ISX Daily Reports Scrapper.

## Test Categories

1. **API Integration Tests**
   - Full HTTP request/response cycle
   - Authentication and authorization
   - Error handling and validation

2. **operation Integration Tests**
   - Complete operation execution
   - step transitions and error recovery
   - WebSocket notifications

3. **Storage Integration Tests**
   - Database operations with real PostgreSQL
   - Transaction handling
   - Concurrent access patterns

## Test Infrastructure

- **Testcontainers** - Spin up real databases for tests
- **Test Fixtures** - Consistent test data
- **Test Helpers** - Common setup and assertions

## Running Tests

```bash
# Run all integration tests
go test ./test/integration/... -tags=integration

# Run with specific database
DATABASE_URL=postgres://... go test ./test/integration/...

# Run with coverage
go test ./test/integration/... -tags=integration -coverprofile=integration.out
```

## Guidelines

1. Use real dependencies (databases, queues) via testcontainers
2. Test complete user workflows, not individual functions
3. Include performance benchmarks for critical paths
4. Clean up all test data after each test