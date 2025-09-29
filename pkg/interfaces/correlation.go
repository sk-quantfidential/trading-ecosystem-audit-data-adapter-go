package interfaces

import (
	"context"
	"time"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

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