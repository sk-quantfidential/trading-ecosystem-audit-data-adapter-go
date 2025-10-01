# Audit Data Adapter Behavior Tests

This directory contains comprehensive behavior tests for the audit-data-adapter-go project. The tests are designed to verify the behavior of the data adapter across different scenarios, including normal operations, error conditions, and performance characteristics.

## Test Structure

### Test Suites

1. **AuditEventBehaviorTestSuite** - Tests audit event repository operations
   - CRUD operations for audit events
   - Complex querying and filtering
   - Metadata and tag handling
   - Event correlation
   - Bulk operations
   - Performance characteristics

2. **ServiceDiscoveryBehaviorTestSuite** - Tests service discovery repository operations
   - Service registration and discovery
   - Heartbeat management
   - Service metrics tracking
   - Multi-instance management
   - Load balancing scenarios
   - Stale service cleanup

3. **CacheBehaviorTestSuite** - Tests cache repository operations
   - Basic key-value operations
   - Complex data type caching
   - TTL (Time-To-Live) management
   - Bulk operations
   - Pattern-based operations
   - Performance testing

4. **IntegrationBehaviorTestSuite** - Tests complete system integration
   - Cross-repository data consistency
   - Full workflow scenarios
   - Transaction consistency
   - Concurrent operations
   - Error recovery
   - Large dataset operations

5. **ComprehensiveBehaviorTestSuite** - Runs all tests in a comprehensive manner
   - Complete system validation
   - Performance benchmarking
   - Scalability testing
   - Error condition handling

### Test Framework Features

- **Behavior-Driven Testing**: Uses Given/When/Then pattern for clear test scenarios
- **Automatic Cleanup**: Tracks created resources and cleans them up automatically
- **Performance Assertions**: Built-in performance testing with configurable thresholds
- **Environment Configuration**: Flexible configuration through environment variables
- **CI/CD Ready**: Automatically adapts behavior for CI environments

## Prerequisites

Before running the tests, ensure you have:

1. **PostgreSQL**: Running instance for audit event storage
   - Default: `postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable`
   - Configure with `TEST_POSTGRES_URL` environment variable

2. **Redis**: Running instance for service discovery and caching
   - Default: `redis://localhost:6379/15` (uses database 15 for tests)
   - Configure with `TEST_REDIS_URL` environment variable

3. **MongoDB** (Optional): For future correlation features
   - Default: `mongodb://localhost:27017/audit_test`
   - Configure with `TEST_MONGO_URL` environment variable

## Running Tests

### Quick Start

```bash
# Run all behavior tests
go test -v ./tests

# Run specific test suite
go test -v ./tests -run TestAuditEventBehaviorSuite
go test -v ./tests -run TestServiceDiscoveryBehaviorSuite
go test -v ./tests -run TestCacheBehaviorSuite
go test -v ./tests -run TestIntegrationBehaviorSuite
go test -v ./tests -run TestComprehensiveBehaviorSuite
```

### Environment Configuration

Configure tests using environment variables:

```bash
# Database connections
export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/audit_test?sslmode=disable"
export TEST_REDIS_URL="redis://localhost:6379/15"
export TEST_MONGO_URL="mongodb://localhost:27017/audit_test"

# Test behavior
export TEST_LOG_LEVEL="info"                # debug, info, warn, error
export TEST_TIMEOUT="10m"                   # Test suite timeout
export SKIP_INTEGRATION_TESTS="false"       # Skip integration tests
export SKIP_PERFORMANCE_TESTS="false"       # Skip performance tests

# Performance testing
export TEST_THROUGHPUT_SIZE="50"            # Number of events for throughput tests
export TEST_MAX_CONCURRENT_OPS="25"         # Max concurrent operations
export TEST_LARGE_DATASET_SIZE="100"        # Large dataset size for testing

# Run tests
go test -v ./tests
```

### Docker Setup

For easy testing with Docker:

```bash
# Start test databases
docker run -d --name audit-test-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=audit_test \
  -p 5432:5432 postgres:17-alpine

docker run -d --name audit-test-redis \
  -p 6379:6379 redis:8-alpine

# Wait for containers to be ready
sleep 5

# Run tests
go test -v ./tests

# Cleanup
docker rm -f audit-test-postgres audit-test-redis
```

### CI/CD Configuration

The tests automatically detect CI environments and adjust behavior:

- Skip performance tests by default in CI
- Use shorter timeouts
- Reduce dataset sizes
- Enable more verbose logging

Example GitHub Actions workflow:

```yaml
name: Behavior Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: audit_test
        options: --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:8-alpine
        options: --health-cmd "redis-cli ping" --health-interval 10s --health-timeout 5s --health-retries 5
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Run Behavior Tests
      env:
        TEST_POSTGRES_URL: postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable
        TEST_REDIS_URL: redis://localhost:6379/15
        TEST_LOG_LEVEL: info
      run: go test -v ./tests
```

## Test Scenarios

### Core Scenarios

The tests cover these key scenarios:

1. **Audit Event Lifecycle**
   - Create → Read → Update → Delete
   - Validation and error handling
   - Timestamp management

2. **Service Discovery Lifecycle**
   - Register → Discover → Update → Unregister
   - Heartbeat management
   - Health status tracking

3. **Cache Operations**
   - Set → Get → Update → Delete
   - TTL management
   - Pattern operations

4. **Bulk Operations**
   - Batch creation and updates
   - Performance optimization
   - Error handling in batches

5. **Transaction Rollback**
   - Transaction consistency
   - Rollback on errors
   - Data integrity

### Advanced Scenarios

1. **Full Audit Workflow**
   - Service registration
   - Event creation with correlation
   - Metrics tracking
   - Workflow completion caching

2. **Concurrent Operations**
   - Multiple simultaneous operations
   - Thread safety validation
   - Consistency under concurrency

3. **Data Consistency**
   - Cross-repository consistency
   - Complex queries
   - Error recovery

4. **Performance Testing**
   - Throughput measurements
   - Latency validation
   - Scalability testing

## Test Output

### Successful Run Example

```
=== RUN   TestComprehensiveBehaviorSuite
=== Starting Comprehensive Behavior Test Suite ===
=== Behavior Test Environment Information ===
INFO[0000] Environment setting                           key=CI value=false
INFO[0000] Environment setting                           key="Skip Performance" value=false
INFO[0000] Environment setting                           key="Test Timeout" value=5m0s
=== End Environment Information ===
=== RUN   TestComprehensiveBehaviorSuite/TestAuditDataAdapterBehavior
Testing comprehensive audit data adapter behavior
=== RUN   TestComprehensiveBehaviorSuite/TestAuditDataAdapterBehavior/BasicAuditEvents
Running behavior scenario: audit_event_lifecycle
=== RUN   TestComprehensiveBehaviorSuite/TestAuditDataAdapterBehavior/BasicServiceDiscovery
Running behavior scenario: service_discovery_lifecycle
=== Comprehensive Behavior Test Suite Completed ===
--- PASS: TestComprehensiveBehaviorSuite (2.45s)
PASS
ok      github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/tests 2.451s
```

### Performance Metrics

The tests provide performance metrics:

```
INFO[0001] Performance measurement                       duration=145.123ms operation="create 100 events individually"
INFO[0001] Performance measurement                       duration=89.456ms operation="query 100 events"
INFO[0002] Performance measurement                       duration=234.789ms operation="bulk create 100 events"
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   ```
   Error: Failed to connect to PostgreSQL
   Solution: Ensure PostgreSQL is running and TEST_POSTGRES_URL is correct
   ```

2. **Redis Connection Errors**
   ```
   Error: Failed to connect to Redis
   Solution: Ensure Redis is running and TEST_REDIS_URL is correct
   ```

3. **Test Timeouts**
   ```
   Error: Test timed out
   Solution: Increase TEST_TIMEOUT or run tests individually
   ```

4. **Performance Test Failures**
   ```
   Error: Operation took longer than expected
   Solution: Run with SKIP_PERFORMANCE_TESTS=true or increase thresholds
   ```

### Debug Mode

Enable debug logging for detailed test execution:

```bash
export TEST_LOG_LEVEL=debug
go test -v ./tests -run TestComprehensiveBehaviorSuite
```

### Individual Test Debugging

Run specific test methods:

```bash
# Run only audit event tests
go test -v ./tests -run TestAuditEventBehaviorSuite/TestAuditEventBasicCRUD

# Run only integration tests
go test -v ./tests -run TestIntegrationBehaviorSuite/TestFullAuditWorkflow
```

## Contributing

When adding new behavior tests:

1. Follow the Given/When/Then pattern
2. Use the test framework helpers
3. Ensure proper cleanup with tracking methods
4. Add performance assertions where appropriate
5. Update this README with new test scenarios

## Test Coverage

The behavior tests provide comprehensive coverage of:

- ✅ All repository interfaces
- ✅ CRUD operations
- ✅ Complex queries
- ✅ Bulk operations
- ✅ Transaction handling
- ✅ Error conditions
- ✅ Performance characteristics
- ✅ Concurrent operations
- ✅ Data consistency
- ✅ Integration scenarios

For code coverage analysis:

```bash
go test -v -coverprofile=coverage.out ./tests
go tool cover -html=coverage.out -o coverage.html
```