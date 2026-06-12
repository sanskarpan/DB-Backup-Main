# Tests Directory

This directory contains integration tests, end-to-end tests, and test fixtures for the DB Backup Utility.

## Directory Structure

```
tests/
├── integration/         # Integration tests (multi-component)
├── e2e/                # End-to-end tests (full system)
├── fixtures/           # Test data and fixtures
└── README.md          # This file
```

## Test Types

### Unit Tests
Unit tests are placed **next to the source code** following Go conventions:
- `internal/repository/repository_test.go`
- `internal/compression/compression_test.go`
- `internal/encryption/encryption_test.go`
- `internal/api/middleware/middleware_test.go`

### Integration Tests (`integration/`)
Tests that verify multiple components working together:
- Database driver integration tests
- Backup + Storage integration
- API endpoint integration tests
- Scheduler integration tests

### End-to-End Tests (`e2e/`)
Full system tests simulating real-world scenarios:
- Complete backup workflows
- CLI command execution
- API request/response cycles
- Frontend automation tests

### Test Fixtures (`fixtures/`)
Sample data for testing:
- Sample database dumps
- Mock configuration files
- Test backup files
- Sample metadata

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./internal/...

# Run with coverage
go test -cover ./internal/...

# Run specific package
go test ./internal/repository -v
```

### Integration Tests
```bash
# Requires database connections
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/testdb"

# Run integration tests
go test -tags=integration ./tests/integration/... -v
```

### End-to-End Tests
```bash
# Run full system tests
go test -tags=e2e ./tests/e2e/... -v
```

### Frontend Tests
```bash
cd frontend/
npm test                  # Run tests
npm run test:coverage    # With coverage
npm run test:ui          # Interactive UI
```

## Test Coverage Goals

- **Unit Tests**: > 98% coverage per CHECKLIST.md
- **Integration Tests**: All major workflows
- **E2E Tests**: Critical user journeys

## CI/CD Integration

All tests run automatically in GitHub Actions:
- Unit tests on every commit
- Integration tests on PR
- E2E tests before deployment
