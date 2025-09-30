package tests

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// GetEnv gets environment variable with empty default
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvOrDefault gets environment variable with default fallback
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvAsInt gets environment variable as integer with default fallback
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsDuration gets environment variable as duration with default fallback
func GetEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvAsBool gets environment variable as boolean with default fallback
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GenerateTestID generates a unique test ID
func GenerateTestID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GenerateTestUUID generates a unique UUID for test purposes
func GenerateTestUUID() string {
	return uuid.New().String()
}

// GetTestConfig returns test configuration based on environment
func GetTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"postgres_url": GetEnvOrDefault("TEST_POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable"),
		"redis_url":    GetEnvOrDefault("TEST_REDIS_URL", "redis://localhost:6379/15"),
		"mongo_url":    GetEnvOrDefault("TEST_MONGO_URL", "mongodb://localhost:27017/audit_test"),
		"environment":  "test",
		"log_level":    GetEnvOrDefault("TEST_LOG_LEVEL", "warn"),
	}
}

// IsCI returns true if running in CI environment
func IsCI() bool {
	return GetEnvAsBool("CI", false) || GetEnvAsBool("GITHUB_ACTIONS", false)
}

// IsLocal returns true if running locally (not in CI)
func IsLocal() bool {
	return !IsCI()
}

// GetGoroutineCount returns current number of goroutines
func GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration, checkInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return lastErr
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	counter int64
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{
		counter: time.Now().UnixNano(),
	}
}

// NextID generates a unique ID
func (g *TestDataGenerator) NextID(prefix string) string {
	g.counter++
	return fmt.Sprintf("%s-%d", prefix, g.counter)
}

// NextUUID generates a unique UUID
func (g *TestDataGenerator) NextUUID() string {
	return uuid.New().String()
}

// NextEmail generates a unique email address
func (g *TestDataGenerator) NextEmail() string {
	g.counter++
	return fmt.Sprintf("test-%d@example.com", g.counter)
}

// NextHost generates a unique hostname
func (g *TestDataGenerator) NextHost() string {
	g.counter++
	return fmt.Sprintf("host-%d.example.com", g.counter)
}

// TestTimestamps provides utilities for time-based testing
type TestTimestamps struct {
	baseTime time.Time
}

// NewTestTimestamps creates a new test timestamps utility
func NewTestTimestamps() *TestTimestamps {
	return &TestTimestamps{
		baseTime: time.Now().Truncate(time.Second), // Remove sub-second precision for easier testing
	}
}

// Now returns the base timestamp
func (t *TestTimestamps) Now() time.Time {
	return t.baseTime
}

// MinutesAgo returns a timestamp N minutes before base time
func (t *TestTimestamps) MinutesAgo(minutes int) time.Time {
	return t.baseTime.Add(-time.Duration(minutes) * time.Minute)
}

// MinutesLater returns a timestamp N minutes after base time
func (t *TestTimestamps) MinutesLater(minutes int) time.Time {
	return t.baseTime.Add(time.Duration(minutes) * time.Minute)
}

// HoursAgo returns a timestamp N hours before base time
func (t *TestTimestamps) HoursAgo(hours int) time.Time {
	return t.baseTime.Add(-time.Duration(hours) * time.Hour)
}

// HoursLater returns a timestamp N hours after base time
func (t *TestTimestamps) HoursLater(hours int) time.Time {
	return t.baseTime.Add(time.Duration(hours) * time.Hour)
}

// DaysAgo returns a timestamp N days before base time
func (t *TestTimestamps) DaysAgo(days int) time.Time {
	return t.baseTime.Add(-time.Duration(days) * 24 * time.Hour)
}

// DaysLater returns a timestamp N days after base time
func (t *TestTimestamps) DaysLater(days int) time.Time {
	return t.baseTime.Add(time.Duration(days) * 24 * time.Hour)
}