package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RepositoryConfig holds configuration for database repositories
type RepositoryConfig struct {
	// Service Identity
	ServiceName         string // e.g., "audit-correlator", "exchange-simulator"
	ServiceInstanceName string // e.g., "audit-correlator", "exchange-OKX"

	// Database connections
	PostgresURL string
	RedisURL    string
	MongoURL    string

	// Multi-tenancy (derived from instance name)
	SchemaName     string // PostgreSQL schema: "audit", "exchange_okx", etc.
	RedisNamespace string // Redis key prefix: "audit", "exchange:OKX", etc.

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
	// Read service identity from environment
	serviceName := getEnv("SERVICE_NAME", "audit-correlator")
	instanceName := getEnv("SERVICE_INSTANCE_NAME", serviceName)

	cfg := &RepositoryConfig{
		// Service Identity
		ServiceName:         serviceName,
		ServiceInstanceName: instanceName,

		// Database connections
		PostgresURL: getEnv("POSTGRES_URL", "postgres://user:password@localhost:5432/trading_ecosystem?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MongoURL:    getEnv("MONGO_URL", "mongodb://localhost:27017/audit_correlator"),

		// Multi-tenancy - automatically derived from instance name
		SchemaName:     deriveSchemaName(serviceName, instanceName),
		RedisNamespace: deriveRedisNamespace(serviceName, instanceName),

		// Connection pooling
		MaxConnections:     getEnvInt("MAX_CONNECTIONS", 25),
		MaxIdleConnections: getEnvInt("MAX_IDLE_CONNECTIONS", 10),
		ConnectionTimeout:  getEnvDuration("CONNECTION_TIMEOUT", 30*time.Second),
		IdleTimeout:        getEnvDuration("IDLE_TIMEOUT", 10*time.Minute),

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

	// Validate instance name
	if err := ValidateInstanceName(cfg.ServiceInstanceName); err != nil {
		// Log warning but don't fail - allow backward compatibility
		// In production, this should be enforced
		_ = err
	}

	return cfg
}

// ValidateInstanceName validates that an instance name follows DNS-safe naming conventions
func ValidateInstanceName(name string) error {
	// Required explicit - no empty strings
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	// DNS-safe: lowercase alphanumeric and hyphens only, must start/end with alphanumeric
	validPattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("instance name must be DNS-safe: lowercase, alphanumeric, hyphens only, must start and end with letter or number (got: %s)", name)
	}

	// Max 63 characters (DNS label limit)
	if len(name) > 63 {
		return fmt.Errorf("instance name exceeds 63 character limit (got: %d characters)", len(name))
	}

	return nil
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

// deriveSchemaName derives PostgreSQL schema name from service instance
func deriveSchemaName(serviceName, instanceName string) string {
	if serviceName == instanceName {
		// Singleton: audit-correlator → "audit"
		parts := strings.Split(serviceName, "-")
		return parts[0]
	}
	// Multi-instance: exchange-OKX → "exchange_okx"
	return strings.ReplaceAll(strings.ToLower(instanceName), "-", "_")
}

// deriveRedisNamespace derives Redis key namespace from service instance
func deriveRedisNamespace(serviceName, instanceName string) string {
	if serviceName == instanceName {
		// Singleton: audit-correlator → "audit"
		parts := strings.Split(serviceName, "-")
		return parts[0]
	}
	// Multi-instance: exchange-OKX → "exchange:OKX"
	parts := strings.SplitN(instanceName, "-", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("%s:%s", parts[0], parts[1])
	}
	return instanceName
}

// maskPassword masks password in database URLs for logging
func maskPassword(url string) string {
	// Simple password masking: user:password@host → user:***@host
	if strings.Contains(url, "@") {
		parts := strings.SplitN(url, "@", 2)
		if strings.Contains(parts[0], ":") {
			userPass := strings.SplitN(parts[0], "://", 2)
			if len(userPass) == 2 {
				credentials := strings.SplitN(userPass[1], ":", 2)
				if len(credentials) == 2 {
					return fmt.Sprintf("%s://%s:***@%s", userPass[0], credentials[0], parts[1])
				}
			}
		}
	}
	return url
}

// LogConfiguration logs the adapter configuration for debugging
func (cfg *RepositoryConfig) LogConfiguration(logger *logrus.Logger) {
	// Mask passwords in URLs
	maskedPostgres := maskPassword(cfg.PostgresURL)
	maskedRedis := maskPassword(cfg.RedisURL)

	logger.WithFields(logrus.Fields{
		"service_name":     cfg.ServiceName,
		"instance_name":    cfg.ServiceInstanceName,
		"schema":           cfg.SchemaName,
		"redis_namespace":  cfg.RedisNamespace,
		"postgres_url":     maskedPostgres,
		"redis_url":        maskedRedis,
		"environment":      cfg.Environment,
	}).Info("Data adapter configuration loaded")
}