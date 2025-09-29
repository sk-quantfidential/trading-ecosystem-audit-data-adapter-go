package adapters

import (
	"context"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/interfaces"
)

// Re-export interfaces for backward compatibility
type AuditEventRepository = interfaces.AuditEventRepository
type ServiceDiscoveryRepository = interfaces.ServiceDiscoveryRepository
type CorrelationRepository = interfaces.CorrelationRepository
type CacheRepository = interfaces.CacheRepository

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

