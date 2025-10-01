package adapters

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/config"
)

// NewAuditDataAdapterFromEnv creates a new audit data adapter with configuration from environment variables
func NewAuditDataAdapterFromEnv(logger *logrus.Logger) (DataAdapter, error) {
	cfg := config.LoadRepositoryConfig()
	return NewAuditDataAdapter(cfg, logger)
}

// NewAuditDataAdapterWithDefaults creates a new audit data adapter with default configuration for development
func NewAuditDataAdapterWithDefaults(logger *logrus.Logger) (DataAdapter, error) {
	cfg := &config.RepositoryConfig{
		PostgresURL: "postgres://user:password@localhost:5432/trading_ecosystem?sslmode=disable",
		RedisURL:    "redis://localhost:6379/0",
		MongoURL:    "mongodb://localhost:27017/audit_correlator",

		MaxConnections:      25,
		MaxIdleConnections:  10,
		ConnectionTimeout:   30000000000, // 30 seconds in nanoseconds
		IdleTimeout:         600000000000, // 10 minutes in nanoseconds

		MaxRetries:    3,
		RetryInterval: 5000000000, // 5 seconds in nanoseconds

		DefaultTTL: 3600000000000, // 1 hour in nanoseconds

		HealthCheckInterval: 15000000000, // 15 seconds in nanoseconds

		Environment: "development",
	}

	return NewAuditDataAdapter(cfg, logger)
}

// InitializeAndConnect creates and connects a data adapter in one step
func InitializeAndConnect(ctx context.Context, logger *logrus.Logger) (DataAdapter, error) {
	adapter, err := NewAuditDataAdapterFromEnv(logger)
	if err != nil {
		return nil, err
	}

	if err := adapter.Connect(ctx); err != nil {
		return nil, err
	}

	return adapter, nil
}