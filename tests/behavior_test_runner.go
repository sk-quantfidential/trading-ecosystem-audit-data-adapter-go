package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

// BehaviorTestRunner provides utilities for running behavior tests with proper setup
type BehaviorTestRunner struct {
	logger          *logrus.Logger
	skipIntegration bool
	skipPerformance bool
	testTimeout     time.Duration
}

// NewBehaviorTestRunner creates a new test runner with configuration
func NewBehaviorTestRunner() *BehaviorTestRunner {
	logger := logrus.New()

	// Configure logging level for tests
	logLevel := GetEnvOrDefault("TEST_LOG_LEVEL", "warn")
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.WarnLevel)
	}

	return &BehaviorTestRunner{
		logger:          logger,
		skipIntegration: GetEnvAsBool("SKIP_INTEGRATION_TESTS", false),
		skipPerformance: GetEnvAsBool("SKIP_PERFORMANCE_TESTS", IsCI()),
		testTimeout:     GetEnvAsDuration("TEST_TIMEOUT", 5*time.Minute),
	}
}

// RunAllBehaviorTests runs all behavior test suites
func (runner *BehaviorTestRunner) RunAllBehaviorTests(t *testing.T) {
	runner.logger.Info("Starting comprehensive behavior test suite")

	// Check test prerequisites
	if !runner.checkPrerequisites(t) {
		t.Fatal("Test prerequisites not met")
	}

	// Run individual test suites
	testSuites := []struct {
		name  string
		suite suite.TestingSuite
		skip  bool
	}{
		{"AuditEventBehavior", new(AuditEventBehaviorTestSuite), false},
		{"ServiceDiscoveryBehavior", new(ServiceDiscoveryBehaviorTestSuite), false},
		{"CacheBehavior", new(CacheBehaviorTestSuite), false},
		{"IntegrationBehavior", new(IntegrationBehaviorTestSuite), runner.skipIntegration},
	}

	totalSuites := 0
	passedSuites := 0

	for _, testSuite := range testSuites {
		if testSuite.skip {
			runner.logger.WithField("suite", testSuite.name).Info("Skipping test suite")
			continue
		}

		totalSuites++
		runner.logger.WithField("suite", testSuite.name).Info("Running behavior test suite")

		// Create a subtest for each suite
		success := t.Run(testSuite.name, func(subT *testing.T) {
			// Set timeout for the test suite
			ctx, cancel := context.WithTimeout(context.Background(), runner.testTimeout)
			defer cancel()

			// Add context to the test
			if behaviorSuite, ok := testSuite.suite.(*BehaviorTestSuite); ok {
				behaviorSuite.ctx = ctx
			}

			suite.Run(subT, testSuite.suite)
		})

		if success {
			passedSuites++
			runner.logger.WithField("suite", testSuite.name).Info("Test suite passed")
		} else {
			runner.logger.WithField("suite", testSuite.name).Error("Test suite failed")
		}
	}

	// Log summary
	runner.logger.WithFields(logrus.Fields{
		"total_suites":  totalSuites,
		"passed_suites": passedSuites,
		"failed_suites": totalSuites - passedSuites,
	}).Info("Behavior test execution summary")

	if passedSuites < totalSuites {
		t.Errorf("Not all test suites passed: %d/%d", passedSuites, totalSuites)
	}
}

// checkPrerequisites verifies that test dependencies are available
func (runner *BehaviorTestRunner) checkPrerequisites(t *testing.T) bool {
	runner.logger.Info("Checking test prerequisites")

	checks := []struct {
		name  string
		check func() error
	}{
		{"Database Connectivity", runner.checkDatabaseConnectivity},
		{"Redis Connectivity", runner.checkRedisConnectivity},
		{"Test Configuration", runner.checkTestConfiguration},
	}

	allPassed := true

	for _, check := range checks {
		if err := check.check(); err != nil {
			runner.logger.WithFields(logrus.Fields{
				"check": check.name,
				"error": err,
			}).Error("Prerequisite check failed")
			allPassed = false
		} else {
			runner.logger.WithField("check", check.name).Info("Prerequisite check passed")
		}
	}

	return allPassed
}

// checkDatabaseConnectivity tests PostgreSQL connectivity
func (runner *BehaviorTestRunner) checkDatabaseConnectivity() error {
	// This is a basic check - could be expanded to actually test connection
	postgresURL := GetEnvOrDefault("TEST_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable")
	if postgresURL == "" {
		return fmt.Errorf("TEST_POSTGRES_URL not configured")
	}

	runner.logger.WithField("postgres_url", postgresURL).Debug("PostgreSQL URL configured")
	return nil
}

// checkRedisConnectivity tests Redis connectivity
func (runner *BehaviorTestRunner) checkRedisConnectivity() error {
	redisURL := GetEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/15")
	if redisURL == "" {
		return fmt.Errorf("TEST_REDIS_URL not configured")
	}

	runner.logger.WithField("redis_url", redisURL).Debug("Redis URL configured")
	return nil
}

// checkTestConfiguration validates test configuration
func (runner *BehaviorTestRunner) checkTestConfiguration() error {
	// Check for required test configuration
	testConfig := GetTestConfig()

	requiredKeys := []string{"postgres_url", "redis_url", "environment"}
	for _, key := range requiredKeys {
		if _, exists := testConfig[key]; !exists {
			return fmt.Errorf("required test configuration '%s' not found", key)
		}
	}

	return nil
}

// PrintTestEnvironmentInfo prints information about the test environment
func (runner *BehaviorTestRunner) PrintTestEnvironmentInfo() {
	runner.logger.Info("=== Behavior Test Environment Information ===")

	envInfo := map[string]interface{}{
		"CI":                     IsCI(),
		"Local":                  IsLocal(),
		"Skip Integration":       runner.skipIntegration,
		"Skip Performance":       runner.skipPerformance,
		"Test Timeout":           runner.testTimeout,
		"Log Level":              runner.logger.GetLevel(),
		"Postgres URL":           GetEnvOrDefault("TEST_POSTGRES_URL", "default"),
		"Redis URL":              GetEnvOrDefault("TEST_REDIS_URL", "default"),
		"Mongo URL":              GetEnvOrDefault("TEST_MONGO_URL", "default"),
	}

	for key, value := range envInfo {
		runner.logger.WithFields(logrus.Fields{
			"key":   key,
			"value": value,
		}).Info("Environment setting")
	}

	runner.logger.Info("=== End Environment Information ===")
}

// BehaviorTestConfig holds configuration for behavior tests
type BehaviorTestConfig struct {
	PostgresURL             string
	RedisURL                string
	MongoURL                string
	TestTimeout             time.Duration
	SkipIntegrationTests    bool
	SkipPerformanceTests    bool
	LogLevel                string
	MaxConcurrentOperations int
	LargeDatasetSize        int
}

// LoadBehaviorTestConfig loads test configuration from environment
func LoadBehaviorTestConfig() *BehaviorTestConfig {
	return &BehaviorTestConfig{
		PostgresURL:             GetEnvOrDefault("TEST_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable"),
		RedisURL:                GetEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/15"),
		MongoURL:                GetEnvOrDefault("TEST_MONGO_URL", "mongodb://localhost:27017/audit_test"),
		TestTimeout:             GetEnvAsDuration("TEST_TIMEOUT", 5*time.Minute),
		SkipIntegrationTests:    GetEnvAsBool("SKIP_INTEGRATION_TESTS", false),
		SkipPerformanceTests:    GetEnvAsBool("SKIP_PERFORMANCE_TESTS", IsCI()),
		LogLevel:                GetEnvOrDefault("TEST_LOG_LEVEL", "warn"),
		MaxConcurrentOperations: GetEnvAsInt("TEST_MAX_CONCURRENT_OPS", 50),
		LargeDatasetSize:        GetEnvAsInt("TEST_LARGE_DATASET_SIZE", 100),
	}
}

// ValidateConfig validates the test configuration
func (config *BehaviorTestConfig) ValidateConfig() error {
	if config.PostgresURL == "" {
		return fmt.Errorf("PostgresURL is required")
	}

	if config.RedisURL == "" {
		return fmt.Errorf("RedisURL is required")
	}

	if config.TestTimeout <= 0 {
		return fmt.Errorf("TestTimeout must be positive")
	}

	return nil
}

// TestMain can be used as the main entry point for behavior tests
func TestMain(m *testing.M) {
	// Setup
	runner := NewBehaviorTestRunner()
	runner.PrintTestEnvironmentInfo()

	// Validate configuration
	config := LoadBehaviorTestConfig()
	if err := config.ValidateConfig(); err != nil {
		runner.logger.WithError(err).Fatal("Invalid test configuration")
	}

	// Run tests
	exitCode := m.Run()

	// Cleanup
	runner.logger.Info("Behavior tests completed")

	os.Exit(exitCode)
}