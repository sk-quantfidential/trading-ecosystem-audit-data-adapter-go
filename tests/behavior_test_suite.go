package tests

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/adapters"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// BehaviorTestSuite provides the base test suite for behavior-driven testing
type BehaviorTestSuite struct {
	suite.Suite
	ctx    context.Context
	logger *logrus.Logger
	config *config.RepositoryConfig

	// Test infrastructure
	adapter         adapters.DataAdapter
	postgresDB      *sql.DB
	redisClient     *redis.Client

	// Test state
	createdEvents   []string
	createdServices []string

	// Test scenarios
	scenarios map[string]func()
}

// SetupSuite runs once before all tests in the suite
func (suite *BehaviorTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup logger for tests
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	// Setup test configuration
	suite.config = &config.RepositoryConfig{
		PostgresURL: getTestEnv("TEST_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable"),
		RedisURL:    getTestEnv("TEST_REDIS_URL", "redis://localhost:6379/15"), // Use DB 15 for tests
		MongoURL:    getTestEnv("TEST_MONGO_URL", "mongodb://localhost:27017/audit_test"),

		MaxConnections:     5,
		MaxIdleConnections: 2,
		ConnectionTimeout:  10 * time.Second,
		IdleTimeout:        5 * time.Minute,

		MaxRetries:    2,
		RetryInterval: time.Second,

		DefaultTTL: 10 * time.Minute,

		HealthCheckInterval: 30 * time.Second,
		Environment:         "test",
	}

	// Initialize test scenarios
	suite.scenarios = make(map[string]func())
	suite.initializeScenarios()
}

// SetupTest runs before each test method
func (suite *BehaviorTestSuite) SetupTest() {
	var err error

	// Create adapter for this test
	suite.adapter, err = adapters.NewAuditDataAdapter(suite.config, suite.logger)
	suite.Require().NoError(err, "Failed to create test adapter")

	// Connect to databases
	err = suite.adapter.Connect(suite.ctx)
	suite.Require().NoError(err, "Failed to connect adapter")

	// Reset test state
	suite.createdEvents = []string{}
	suite.createdServices = []string{}

	// Clean up test data
	suite.cleanupTestData()
}

// TearDownTest runs after each test method
func (suite *BehaviorTestSuite) TearDownTest() {
	if suite.adapter != nil {
		// Clean up test data
		suite.cleanupTestData()

		// Disconnect adapter
		err := suite.adapter.Disconnect(suite.ctx)
		if err != nil {
			suite.logger.WithError(err).Warn("Failed to disconnect adapter in teardown")
		}
	}
}

// TearDownSuite runs once after all tests in the suite
func (suite *BehaviorTestSuite) TearDownSuite() {
	// Final cleanup
	suite.logger.Info("Behavior test suite completed")
}

// Helper methods for behavior testing

// Given sets up the initial state for a test scenario
func (suite *BehaviorTestSuite) Given(description string, setup func()) *BehaviorTestSuite {
	suite.logger.WithField("given", description).Info("Setting up test scenario")
	setup()
	return suite
}

// When executes the action being tested
func (suite *BehaviorTestSuite) When(description string, action func()) *BehaviorTestSuite {
	suite.logger.WithField("when", description).Info("Executing test action")
	action()
	return suite
}

// Then validates the expected outcome
func (suite *BehaviorTestSuite) Then(description string, validation func()) *BehaviorTestSuite {
	suite.logger.WithField("then", description).Info("Validating test outcome")
	validation()
	return suite
}

// And chains additional conditions or actions
func (suite *BehaviorTestSuite) And(description string, step func()) *BehaviorTestSuite {
	suite.logger.WithField("and", description).Info("Additional test step")
	step()
	return suite
}

// Test data helpers

// CreateTestAuditEvent creates a test audit event with common defaults
func (suite *BehaviorTestSuite) CreateTestAuditEvent(id string, overrides ...func(*models.AuditEvent)) *models.AuditEvent {
	event := &models.AuditEvent{
		ID:          id,
		TraceID:     fmt.Sprintf("trace-%s", id),
		SpanID:      fmt.Sprintf("span-%s", id),
		ServiceName: "test-service",
		EventType:   "test-event",
		Timestamp:   time.Now(),
		Duration:    100 * time.Millisecond,
		Status:      models.AuditEventStatusPending,
		Metadata: map[string]interface{}{
			"test_key": "test_value",
			"numeric":  42,
		},
		Tags:         []string{"test", "behavior"},
		CorrelatedTo: []string{},
	}

	// Apply any overrides
	for _, override := range overrides {
		override(event)
	}

	return event
}

// CreateTestServiceRegistration creates a test service registration
func (suite *BehaviorTestSuite) CreateTestServiceRegistration(id string, overrides ...func(*models.ServiceRegistration)) *models.ServiceRegistration {
	service := &models.ServiceRegistration{
		ID:       id,
		Name:     "test-service",
		Version:  "1.0.0",
		Host:     "localhost",
		GRPCPort: 9000,
		HTTPPort: 8000,
		Status:   "healthy",
		Metadata: map[string]string{
			"environment": "test",
			"region":      "local",
		},
		LastSeen:     time.Now(),
		RegisteredAt: time.Now(),
	}

	// Apply any overrides
	for _, override := range overrides {
		override(service)
	}

	return service
}

// Scenario helpers

// RunScenario executes a named test scenario
func (suite *BehaviorTestSuite) RunScenario(name string) {
	if scenario, exists := suite.scenarios[name]; exists {
		suite.logger.WithField("scenario", name).Info("Running behavior scenario")
		scenario()
	} else {
		suite.Fail("Unknown scenario: " + name)
	}
}

// initializeScenarios sets up common test scenarios
func (suite *BehaviorTestSuite) initializeScenarios() {
	suite.scenarios["audit_event_lifecycle"] = suite.auditEventLifecycleScenario
	suite.scenarios["service_discovery_lifecycle"] = suite.serviceDiscoveryLifecycleScenario
	suite.scenarios["cache_operations"] = suite.cacheOperationsScenario
	suite.scenarios["transaction_rollback"] = suite.transactionRollbackScenario
	suite.scenarios["bulk_operations"] = suite.bulkOperationsScenario
}

// Cleanup helpers

// cleanupTestData removes any test data created during tests
func (suite *BehaviorTestSuite) cleanupTestData() {
	// Clean up audit events
	for _, eventID := range suite.createdEvents {
		_ = suite.adapter.Delete(suite.ctx, eventID)
	}

	// Clean up service registrations
	for _, serviceID := range suite.createdServices {
		_ = suite.adapter.UnregisterService(suite.ctx, serviceID)
	}

	// Clean up cache entries
	pattern := "test:*"
	keys, err := suite.adapter.GetKeysByPattern(suite.ctx, pattern)
	if err == nil {
		_ = suite.adapter.DeleteMany(suite.ctx, keys)
	}
}

// trackCreatedEvent adds an event ID to the cleanup list
func (suite *BehaviorTestSuite) trackCreatedEvent(eventID string) {
	suite.createdEvents = append(suite.createdEvents, eventID)
}

// trackCreatedService adds a service ID to the cleanup list
func (suite *BehaviorTestSuite) trackCreatedService(serviceID string) {
	suite.createdServices = append(suite.createdServices, serviceID)
}

// Utility functions

// getTestEnv gets environment variable with fallback
func getTestEnv(key, defaultValue string) string {
	if value := GetEnv(key); value != "" {
		return value
	}
	return defaultValue
}

// Custom assertion helpers

// AssertEventExists checks if an audit event exists and has expected properties
func (suite *BehaviorTestSuite) AssertEventExists(eventID string, expectedStatus models.AuditEventStatus) {
	event, err := suite.adapter.GetByID(suite.ctx, eventID)
	suite.Require().NoError(err, "Failed to retrieve event %s", eventID)
	suite.Require().NotNil(event, "Event %s should exist", eventID)
	suite.Equal(expectedStatus, event.Status, "Event %s should have status %s", eventID, expectedStatus)
}

// AssertServiceRegistered checks if a service is properly registered
func (suite *BehaviorTestSuite) AssertServiceRegistered(serviceID string, expectedStatus string) {
	service, err := suite.adapter.GetService(suite.ctx, serviceID)
	suite.Require().NoError(err, "Failed to retrieve service %s", serviceID)
	suite.Require().NotNil(service, "Service %s should be registered", serviceID)
	suite.Equal(expectedStatus, service.Status, "Service %s should have status %s", serviceID, expectedStatus)
}

// AssertCacheContains checks if cache contains expected key-value pair
func (suite *BehaviorTestSuite) AssertCacheContains(key string, expectedValue interface{}) {
	var actualValue interface{}
	err := suite.adapter.Get(suite.ctx, key, &actualValue)
	suite.Require().NoError(err, "Failed to get cache value for key %s", key)
	suite.Equal(expectedValue, actualValue, "Cache should contain expected value for key %s", key)
}

// Performance helpers

// MeasureExecutionTime measures and logs execution time of a function
func (suite *BehaviorTestSuite) MeasureExecutionTime(description string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	suite.logger.WithFields(logrus.Fields{
		"operation": description,
		"duration":  duration,
	}).Info("Performance measurement")
	return duration
}

// AssertPerformance checks if operation completes within expected time
func (suite *BehaviorTestSuite) AssertPerformance(description string, maxDuration time.Duration, fn func()) {
	duration := suite.MeasureExecutionTime(description, fn)
	suite.LessOrEqual(duration, maxDuration,
		"Operation '%s' took %v but should complete within %v", description, duration, maxDuration)
}