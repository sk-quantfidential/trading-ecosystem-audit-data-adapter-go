package config

import (
	"os"
	"strconv"
	"time"
)

// RepositoryConfig holds configuration for database repositories
type RepositoryConfig struct {
	// Database connections
	PostgresURL string
	RedisURL    string
	MongoURL    string

	// Connection pooling
	MaxConnections     int
	MaxIdleConnections int
	ConnectionTimeout  time.Duration
	IdleTimeout        time.Duration

	// Retry configuration
	MaxRetries    int
	RetryInterval time.Duration

	// Cache configuration
	DefaultTTL time.Duration

	// Health check configuration
	HealthCheckInterval time.Duration

	// Environment
	Environment string
}

// LoadRepositoryConfig loads configuration from environment variables
func LoadRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{
		// Database connections
		PostgresURL: getEnv("POSTGRES_URL", "postgres://user:password@localhost:5432/trading_ecosystem?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MongoURL:    getEnv("MONGO_URL", "mongodb://localhost:27017/audit_correlator"),

		// Connection pooling
		MaxConnections:    getEnvInt("MAX_CONNECTIONS", 25),
		MaxIdleConnections: getEnvInt("MAX_IDLE_CONNECTIONS", 10),
		ConnectionTimeout: getEnvDuration("CONNECTION_TIMEOUT", 30*time.Second),
		IdleTimeout:       getEnvDuration("IDLE_TIMEOUT", 10*time.Minute),

		// Retry configuration
		MaxRetries:    getEnvInt("MAX_RETRIES", 3),
		RetryInterval: getEnvDuration("RETRY_INTERVAL", 5*time.Second),

		// Cache configuration
		DefaultTTL: getEnvDuration("DEFAULT_TTL", 1*time.Hour),

		// Health check configuration
		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", 15*time.Second),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}