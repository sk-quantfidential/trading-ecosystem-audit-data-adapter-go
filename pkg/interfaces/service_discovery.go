package interfaces

import (
	"context"
	"time"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

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