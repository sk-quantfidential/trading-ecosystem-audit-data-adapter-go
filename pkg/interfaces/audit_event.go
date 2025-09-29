package interfaces

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