package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	_ "github.com/lib/pq"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/postgres"
	redisImpl "github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/redis"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// AuditDataAdapter implements the DataAdapter interface for audit correlator
type AuditDataAdapter struct {
	// Database connections
	postgresDB  *sql.DB
	redisClient *redis.Client

	// Repository implementations
	auditEvents       AuditEventRepository
	serviceDiscovery  ServiceDiscoveryRepository
	correlations      CorrelationRepository
	cache             CacheRepository

	// Configuration and logging
	config *RepositoryConfig
	logger *logrus.Logger

	// Connection state
	connected bool
}

// NewAuditDataAdapter creates a new audit data adapter
func NewAuditDataAdapter(config *RepositoryConfig, logger *logrus.Logger) (DataAdapter, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	adapter := &AuditDataAdapter{
		config: config,
		logger: logger,
	}

	return adapter, nil
}

// Connect establishes connections to all data sources
func (a *AuditDataAdapter) Connect(ctx context.Context) error {
	if a.connected {
		return nil
	}

	// Connect to PostgreSQL
	if err := a.connectPostgreSQL(ctx); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Connect to Redis
	if err := a.connectRedis(ctx); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize repositories
	a.initializeRepositories()

	a.connected = true
	a.logger.Info("Audit data adapter connected to all data sources")

	return nil
}

// Disconnect closes all database connections
func (a *AuditDataAdapter) Disconnect(ctx context.Context) error {
	if !a.connected {
		return nil
	}

	var errors []error

	// Close PostgreSQL connection
	if a.postgresDB != nil {
		if err := a.postgresDB.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close PostgreSQL: %w", err))
		}
	}

	// Close Redis connection
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	a.connected = false

	if len(errors) > 0 {
		return fmt.Errorf("errors during disconnect: %v", errors)
	}

	a.logger.Info("Audit data adapter disconnected from all data sources")
	return nil
}

// Health checks the health of all data sources
func (a *AuditDataAdapter) Health(ctx context.Context) error {
	if !a.connected {
		return fmt.Errorf("adapter not connected")
	}

	// Check PostgreSQL health
	if err := a.postgresDB.PingContext(ctx); err != nil {
		return fmt.Errorf("PostgreSQL health check failed: %w", err)
	}

	// Check Redis health
	if err := a.cache.Ping(ctx); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// BeginTransaction starts a new transaction
func (a *AuditDataAdapter) BeginTransaction(ctx context.Context) (Transaction, error) {
	if !a.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	tx, err := a.postgresDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &AuditTransaction{
		tx:     tx,
		config: a.config,
		logger: a.logger,
	}, nil
}

// AuditEventRepository methods
func (a *AuditDataAdapter) Create(ctx context.Context, event *models.AuditEvent) error {
	return a.auditEvents.Create(ctx, event)
}

func (a *AuditDataAdapter) GetByID(ctx context.Context, id string) (*models.AuditEvent, error) {
	return a.auditEvents.GetByID(ctx, id)
}

func (a *AuditDataAdapter) Update(ctx context.Context, event *models.AuditEvent) error {
	return a.auditEvents.Update(ctx, event)
}

func (a *AuditDataAdapter) Delete(ctx context.Context, id string) error {
	return a.auditEvents.Delete(ctx, id)
}

func (a *AuditDataAdapter) Query(ctx context.Context, query models.AuditQuery) ([]*models.AuditEvent, error) {
	return a.auditEvents.Query(ctx, query)
}

func (a *AuditDataAdapter) Count(ctx context.Context, query models.AuditQuery) (int64, error) {
	return a.auditEvents.Count(ctx, query)
}

func (a *AuditDataAdapter) GetByTraceID(ctx context.Context, traceID string) ([]*models.AuditEvent, error) {
	return a.auditEvents.GetByTraceID(ctx, traceID)
}

func (a *AuditDataAdapter) GetCorrelatedEvents(ctx context.Context, eventID string) ([]*models.AuditEvent, error) {
	return a.auditEvents.GetCorrelatedEvents(ctx, eventID)
}

func (a *AuditDataAdapter) CreateBatch(ctx context.Context, events []*models.AuditEvent) error {
	return a.auditEvents.CreateBatch(ctx, events)
}

func (a *AuditDataAdapter) UpdateBatch(ctx context.Context, events []*models.AuditEvent) error {
	return a.auditEvents.UpdateBatch(ctx, events)
}

func (a *AuditDataAdapter) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	return a.auditEvents.DeleteOlderThan(ctx, timestamp)
}

func (a *AuditDataAdapter) ArchiveOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	return a.auditEvents.ArchiveOlderThan(ctx, timestamp)
}

// ServiceDiscoveryRepository methods
func (a *AuditDataAdapter) RegisterService(ctx context.Context, service *models.ServiceRegistration) error {
	return a.serviceDiscovery.RegisterService(ctx, service)
}

func (a *AuditDataAdapter) UpdateService(ctx context.Context, service *models.ServiceRegistration) error {
	return a.serviceDiscovery.UpdateService(ctx, service)
}

func (a *AuditDataAdapter) UnregisterService(ctx context.Context, serviceID string) error {
	return a.serviceDiscovery.UnregisterService(ctx, serviceID)
}

func (a *AuditDataAdapter) GetService(ctx context.Context, serviceID string) (*models.ServiceRegistration, error) {
	return a.serviceDiscovery.GetService(ctx, serviceID)
}

func (a *AuditDataAdapter) GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error) {
	return a.serviceDiscovery.GetServicesByName(ctx, serviceName)
}

func (a *AuditDataAdapter) GetAllServices(ctx context.Context) ([]*models.ServiceRegistration, error) {
	return a.serviceDiscovery.GetAllServices(ctx)
}

func (a *AuditDataAdapter) GetHealthyServices(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error) {
	return a.serviceDiscovery.GetHealthyServices(ctx, serviceName)
}

func (a *AuditDataAdapter) UpdateHeartbeat(ctx context.Context, serviceID string) error {
	return a.serviceDiscovery.UpdateHeartbeat(ctx, serviceID)
}

func (a *AuditDataAdapter) CleanupStaleServices(ctx context.Context, ttl time.Duration) (int64, error) {
	return a.serviceDiscovery.CleanupStaleServices(ctx, ttl)
}

func (a *AuditDataAdapter) GetServiceMetrics(ctx context.Context, serviceName string) (*models.ServiceMetrics, error) {
	return a.serviceDiscovery.GetServiceMetrics(ctx, serviceName)
}

func (a *AuditDataAdapter) UpdateServiceMetrics(ctx context.Context, metrics *models.ServiceMetrics) error {
	return a.serviceDiscovery.UpdateServiceMetrics(ctx, metrics)
}

// CacheRepository methods
func (a *AuditDataAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return a.cache.Set(ctx, key, value, ttl)
}

func (a *AuditDataAdapter) Get(ctx context.Context, key string, dest interface{}) error {
	return a.cache.Get(ctx, key, dest)
}

func (a *AuditDataAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.cache.Exists(ctx, key)
}

func (a *AuditDataAdapter) SetMany(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	return a.cache.SetMany(ctx, items, ttl)
}

func (a *AuditDataAdapter) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	return a.cache.GetMany(ctx, keys)
}

func (a *AuditDataAdapter) DeleteMany(ctx context.Context, keys []string) error {
	return a.cache.DeleteMany(ctx, keys)
}

func (a *AuditDataAdapter) GetKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	return a.cache.GetKeysByPattern(ctx, pattern)
}

func (a *AuditDataAdapter) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	return a.cache.DeleteByPattern(ctx, pattern)
}

func (a *AuditDataAdapter) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	return a.cache.SetTTL(ctx, key, ttl)
}

func (a *AuditDataAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return a.cache.GetTTL(ctx, key)
}

func (a *AuditDataAdapter) Ping(ctx context.Context) error {
	return a.cache.Ping(ctx)
}

func (a *AuditDataAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return a.cache.GetStats(ctx)
}

// CorrelationRepository methods (placeholder implementation for future use)
func (a *AuditDataAdapter) CreateCorrelation(ctx context.Context, correlation *models.AuditCorrelation) error {
	if a.correlations != nil {
		return a.correlations.CreateCorrelation(ctx, correlation)
	}
	// For now, store in cache as fallback
	key := fmt.Sprintf("correlation:%s:%s", correlation.SourceEventID, correlation.TargetEventID)
	return a.cache.Set(ctx, key, correlation, a.config.DefaultTTL)
}

func (a *AuditDataAdapter) CreateCorrelations(ctx context.Context, correlations []*models.AuditCorrelation) error {
	if a.correlations != nil {
		return a.correlations.CreateCorrelations(ctx, correlations)
	}
	// Fallback implementation
	items := make(map[string]interface{})
	for _, correlation := range correlations {
		key := fmt.Sprintf("correlation:%s:%s", correlation.SourceEventID, correlation.TargetEventID)
		items[key] = correlation
	}
	return a.cache.SetMany(ctx, items, a.config.DefaultTTL)
}

func (a *AuditDataAdapter) GetCorrelationsByEvent(ctx context.Context, eventID string) ([]*models.AuditCorrelation, error) {
	if a.correlations != nil {
		return a.correlations.GetCorrelationsByEvent(ctx, eventID)
	}
	// Fallback implementation
	return []*models.AuditCorrelation{}, nil
}

func (a *AuditDataAdapter) GetCorrelationsByType(ctx context.Context, correlationType string) ([]*models.AuditCorrelation, error) {
	if a.correlations != nil {
		return a.correlations.GetCorrelationsByType(ctx, correlationType)
	}
	// Fallback implementation
	return []*models.AuditCorrelation{}, nil
}

func (a *AuditDataAdapter) FindSimilarEvents(ctx context.Context, eventID string, threshold float64) ([]*models.AuditCorrelation, error) {
	if a.correlations != nil {
		return a.correlations.FindSimilarEvents(ctx, eventID, threshold)
	}
	// Fallback implementation
	return []*models.AuditCorrelation{}, nil
}

func (a *AuditDataAdapter) GetCorrelationChain(ctx context.Context, eventID string, maxDepth int) ([]*models.AuditCorrelation, error) {
	if a.correlations != nil {
		return a.correlations.GetCorrelationChain(ctx, eventID, maxDepth)
	}
	// Fallback implementation
	return []*models.AuditCorrelation{}, nil
}

func (a *AuditDataAdapter) DeleteCorrelationsOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	if a.correlations != nil {
		return a.correlations.DeleteCorrelationsOlderThan(ctx, timestamp)
	}
	// Fallback implementation
	return 0, nil
}

// Private helper methods
func (a *AuditDataAdapter) connectPostgreSQL(ctx context.Context) error {
	db, err := sql.Open("postgres", a.config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(a.config.MaxConnections)
	db.SetMaxIdleConns(a.config.MaxIdleConnections)
	db.SetConnMaxLifetime(a.config.IdleTimeout)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	a.postgresDB = db
	a.logger.Info("Connected to PostgreSQL")

	return nil
}

func (a *AuditDataAdapter) connectRedis(ctx context.Context) error {
	opt, err := redis.ParseURL(a.config.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure connection pool
	opt.PoolSize = a.config.MaxConnections
	opt.MinIdleConns = a.config.MaxIdleConnections
	opt.DialTimeout = a.config.ConnectionTimeout
	opt.ReadTimeout = a.config.ConnectionTimeout
	opt.WriteTimeout = a.config.ConnectionTimeout

	client := redis.NewClient(opt)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	a.redisClient = client
	a.logger.Info("Connected to Redis")

	return nil
}

func (a *AuditDataAdapter) initializeRepositories() {
	// Initialize PostgreSQL-based repositories
	a.auditEvents = postgres.NewAuditEventRepository(a.postgresDB, a.logger, a.config)

	// Initialize Redis-based repositories
	a.serviceDiscovery = redisImpl.NewServiceDiscoveryRepository(a.redisClient, a.logger, a.config)
	a.cache = redisImpl.NewCacheRepository(a.redisClient, a.logger, a.config)

	// Note: correlations repository can be added later when needed
	a.correlations = nil

	a.logger.Info("Initialized all repository implementations")
}

// AuditTransaction implements the Transaction interface
type AuditTransaction struct {
	tx     *sql.Tx
	config *RepositoryConfig
	logger *logrus.Logger
}

func (t *AuditTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *AuditTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

func (t *AuditTransaction) AuditEvents() AuditEventRepository {
	return postgres.NewAuditEventRepository(t.tx, t.logger, t.config)
}

func (t *AuditTransaction) Correlations() CorrelationRepository {
	// For now, return nil since correlations are not implemented yet
	return nil
}