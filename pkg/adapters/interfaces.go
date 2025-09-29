package adapters

import (
	"context"
	"time"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// AuditEventRepository defines the interface for audit event persistence
type AuditEventRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, event *models.AuditEvent) error
	GetByID(ctx context.Context, id string) (*models.AuditEvent, error)
	Update(ctx context.Context, event *models.AuditEvent) error
	Delete(ctx context.Context, id string) error

	// Query operations
	Query(ctx context.Context, query models.AuditQuery) ([]*models.AuditEvent, error)
	Count(ctx context.Context, query models.AuditQuery) (int64, error)

	// Trace operations
	GetByTraceID(ctx context.Context, traceID string) ([]*models.AuditEvent, error)
	GetCorrelatedEvents(ctx context.Context, eventID string) ([]*models.AuditEvent, error)

	// Bulk operations
	CreateBatch(ctx context.Context, events []*models.AuditEvent) error
	UpdateBatch(ctx context.Context, events []*models.AuditEvent) error

	// Cleanup operations
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
	ArchiveOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// ServiceDiscoveryRepository defines the interface for service discovery
type ServiceDiscoveryRepository interface {
	// Service registration
	RegisterService(ctx context.Context, service *models.ServiceRegistration) error
	UpdateService(ctx context.Context, service *models.ServiceRegistration) error
	UnregisterService(ctx context.Context, serviceID string) error

	// Service discovery
	GetService(ctx context.Context, serviceID string) (*models.ServiceRegistration, error)
	GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error)
	GetAllServices(ctx context.Context) ([]*models.ServiceRegistration, error)
	GetHealthyServices(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error)

	// Health management
	UpdateHeartbeat(ctx context.Context, serviceID string) error
	CleanupStaleServices(ctx context.Context, ttl time.Duration) (int64, error)

	// Metrics
	GetServiceMetrics(ctx context.Context, serviceName string) (*models.ServiceMetrics, error)
	UpdateServiceMetrics(ctx context.Context, metrics *models.ServiceMetrics) error
}

// CorrelationRepository defines the interface for event correlation
type CorrelationRepository interface {
	// Create correlations
	CreateCorrelation(ctx context.Context, correlation *models.AuditCorrelation) error
	CreateCorrelations(ctx context.Context, correlations []*models.AuditCorrelation) error

	// Query correlations
	GetCorrelationsByEvent(ctx context.Context, eventID string) ([]*models.AuditCorrelation, error)
	GetCorrelationsByType(ctx context.Context, correlationType string) ([]*models.AuditCorrelation, error)

	// Analysis operations
	FindSimilarEvents(ctx context.Context, eventID string, threshold float64) ([]*models.AuditCorrelation, error)
	GetCorrelationChain(ctx context.Context, eventID string, maxDepth int) ([]*models.AuditCorrelation, error)

	// Cleanup
	DeleteCorrelationsOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Key-Value operations
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Bulk operations
	SetMany(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	GetMany(ctx context.Context, keys []string) (map[string]interface{}, error)
	DeleteMany(ctx context.Context, keys []string) error

	// Pattern operations
	GetKeysByPattern(ctx context.Context, pattern string) ([]string, error)
	DeleteByPattern(ctx context.Context, pattern string) (int64, error)

	// Expiration
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Health and stats
	Ping(ctx context.Context) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// DataAdapter combines all repository interfaces for audit correlator
type DataAdapter interface {
	AuditEventRepository
	ServiceDiscoveryRepository
	CorrelationRepository
	CacheRepository

	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Health(ctx context.Context) error

	// Transaction support
	BeginTransaction(ctx context.Context) (Transaction, error)
}

// Transaction defines the interface for transactional operations
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	// Repository access within transaction
	AuditEvents() AuditEventRepository
	Correlations() CorrelationRepository
}

// RepositoryConfig defines configuration for data adapters
type RepositoryConfig struct {
	// Database connections
	PostgresURL string
	RedisURL    string
	MongoURL    string

	// Connection pooling
	MaxConnections    int
	MaxIdleConnections int
	ConnectionTimeout time.Duration
	IdleTimeout       time.Duration

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