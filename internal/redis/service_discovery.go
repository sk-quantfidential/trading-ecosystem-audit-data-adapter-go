package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/interfaces"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// ServiceDiscoveryRedisRepository implements ServiceDiscoveryRepository using Redis
type ServiceDiscoveryRedisRepository struct {
	client *redis.Client
	logger *logrus.Logger
	config *config.RepositoryConfig
}

// NewServiceDiscoveryRepository creates a new Redis-based service discovery repository
func NewServiceDiscoveryRepository(client *redis.Client, logger *logrus.Logger, cfg *config.RepositoryConfig) interfaces.ServiceDiscoveryRepository {
	return &ServiceDiscoveryRedisRepository{
		client: client,
		logger: logger,
		config: cfg,
	}
}

// RegisterService registers a service in Redis
func (r *ServiceDiscoveryRedisRepository) RegisterService(ctx context.Context, service *models.ServiceRegistration) error {
	service.RegisteredAt = time.Now()
	service.LastSeen = time.Now()

	// Serialize service data
	serviceData, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service registration: %w", err)
	}

	// Store in Redis with TTL
	serviceKey := r.getServiceKey(service.Name, service.ID)
	ttl := r.config.HealthCheckInterval * 3 // Allow 3 missed heartbeats

	err = r.client.Set(ctx, serviceKey, serviceData, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Add to service name index
	indexKey := r.getServiceIndexKey(service.Name)
	err = r.client.SAdd(ctx, indexKey, service.ID).Err()
	if err != nil {
		r.logger.WithError(err).Warn("Failed to update service index")
	}

	r.logger.WithFields(logrus.Fields{
		"service_name": service.Name,
		"service_id":   service.ID,
		"host":         service.Host,
		"grpc_port":    service.GRPCPort,
	}).Info("Service registered in discovery")

	return nil
}

// UpdateService updates an existing service registration
func (r *ServiceDiscoveryRedisRepository) UpdateService(ctx context.Context, service *models.ServiceRegistration) error {
	service.LastSeen = time.Now()

	serviceData, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service update: %w", err)
	}

	serviceKey := r.getServiceKey(service.Name, service.ID)
	ttl := r.config.HealthCheckInterval * 3

	err = r.client.Set(ctx, serviceKey, serviceData, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// UnregisterService removes a service from Redis
func (r *ServiceDiscoveryRedisRepository) UnregisterService(ctx context.Context, serviceID string) error {
	// Find the service to get its name
	pattern := fmt.Sprintf("services:*:%s", serviceID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find service keys: %w", err)
	}

	if len(keys) == 0 {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	// Delete service key
	serviceKey := keys[0]
	err = r.client.Del(ctx, serviceKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Extract service name from key and remove from index
	// Key format: services:serviceName:serviceID
	keyParts := serviceKey[9:] // Remove "services:" prefix
	serviceName := keyParts[:len(keyParts)-len(serviceID)-1] // Remove ":serviceID" suffix

	indexKey := r.getServiceIndexKey(serviceName)
	err = r.client.SRem(ctx, indexKey, serviceID).Err()
	if err != nil {
		r.logger.WithError(err).Warn("Failed to update service index on unregister")
	}

	r.logger.WithFields(logrus.Fields{
		"service_id":   serviceID,
		"service_name": serviceName,
	}).Info("Service unregistered from discovery")

	return nil
}

// GetService retrieves a specific service by ID
func (r *ServiceDiscoveryRedisRepository) GetService(ctx context.Context, serviceID string) (*models.ServiceRegistration, error) {
	pattern := fmt.Sprintf("services:*:%s", serviceID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	serviceData, err := r.client.Get(ctx, keys[0]).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service data: %w", err)
	}

	var service models.ServiceRegistration
	if err := json.Unmarshal([]byte(serviceData), &service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service data: %w", err)
	}

	return &service, nil
}

// GetServicesByName retrieves all services with a specific name
func (r *ServiceDiscoveryRedisRepository) GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error) {
	pattern := fmt.Sprintf("services:%s:*", serviceName)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	var services []*models.ServiceRegistration

	for _, key := range keys {
		serviceData, err := r.client.Get(ctx, key).Result()
		if err != nil {
			r.logger.WithError(err).WithField("key", key).Warn("Failed to get service data")
			continue
		}

		var service models.ServiceRegistration
		if err := json.Unmarshal([]byte(serviceData), &service); err != nil {
			r.logger.WithError(err).WithField("key", key).Warn("Failed to unmarshal service data")
			continue
		}

		services = append(services, &service)
	}

	return services, nil
}

// GetAllServices retrieves all registered services
func (r *ServiceDiscoveryRedisRepository) GetAllServices(ctx context.Context) ([]*models.ServiceRegistration, error) {
	keys, err := r.client.Keys(ctx, "services:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all services: %w", err)
	}

	var services []*models.ServiceRegistration

	for _, key := range keys {
		serviceData, err := r.client.Get(ctx, key).Result()
		if err != nil {
			r.logger.WithError(err).WithField("key", key).Warn("Failed to get service data")
			continue
		}

		var service models.ServiceRegistration
		if err := json.Unmarshal([]byte(serviceData), &service); err != nil {
			r.logger.WithError(err).WithField("key", key).Warn("Failed to unmarshal service data")
			continue
		}

		services = append(services, &service)
	}

	return services, nil
}

// GetHealthyServices retrieves only healthy services with a specific name
func (r *ServiceDiscoveryRedisRepository) GetHealthyServices(ctx context.Context, serviceName string) ([]*models.ServiceRegistration, error) {
	services, err := r.GetServicesByName(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	var healthyServices []*models.ServiceRegistration
	now := time.Now()

	for _, service := range services {
		// Check if service is considered healthy based on last seen time
		if service.Status == "healthy" && now.Sub(service.LastSeen) < r.config.HealthCheckInterval*3 {
			healthyServices = append(healthyServices, service)
		}
	}

	return healthyServices, nil
}

// UpdateHeartbeat updates the last seen timestamp for a service
func (r *ServiceDiscoveryRedisRepository) UpdateHeartbeat(ctx context.Context, serviceID string) error {
	service, err := r.GetService(ctx, serviceID)
	if err != nil {
		return err
	}

	service.LastSeen = time.Now()
	return r.UpdateService(ctx, service)
}

// CleanupStaleServices removes services that haven't been seen for the specified TTL
func (r *ServiceDiscoveryRedisRepository) CleanupStaleServices(ctx context.Context, ttl time.Duration) (int64, error) {
	services, err := r.GetAllServices(ctx)
	if err != nil {
		return 0, err
	}

	var cleanedCount int64
	now := time.Now()

	for _, service := range services {
		if now.Sub(service.LastSeen) > ttl {
			if err := r.UnregisterService(ctx, service.ID); err != nil {
				r.logger.WithError(err).WithField("service_id", service.ID).Warn("Failed to cleanup stale service")
			} else {
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		r.logger.WithField("cleaned_count", cleanedCount).Info("Cleaned up stale services")
	}

	return cleanedCount, nil
}

// GetServiceMetrics retrieves metrics for a service
func (r *ServiceDiscoveryRedisRepository) GetServiceMetrics(ctx context.Context, serviceName string) (*models.ServiceMetrics, error) {
	metricsKey := r.getServiceMetricsKey(serviceName)

	metricsData, err := r.client.HGetAll(ctx, metricsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service metrics: %w", err)
	}

	if len(metricsData) == 0 {
		return nil, fmt.Errorf("metrics not found for service: %s", serviceName)
	}

	metrics := &models.ServiceMetrics{
		ServiceName:   serviceName,
		CustomMetrics: make(map[string]float64),
	}

	// Parse metrics data
	if instanceID, exists := metricsData["instance_id"]; exists {
		metrics.InstanceID = instanceID
	}

	if timestampStr, exists := metricsData["timestamp"]; exists {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			metrics.Timestamp = timestamp
		}
	}

	if requestCountStr, exists := metricsData["request_count"]; exists {
		if requestCount, err := strconv.ParseInt(requestCountStr, 10, 64); err == nil {
			metrics.RequestCount = requestCount
		}
	}

	if errorCountStr, exists := metricsData["error_count"]; exists {
		if errorCount, err := strconv.ParseInt(errorCountStr, 10, 64); err == nil {
			metrics.ErrorCount = errorCount
		}
	}

	if responseTimeStr, exists := metricsData["response_time"]; exists {
		if responseTime, err := time.ParseDuration(responseTimeStr); err == nil {
			metrics.ResponseTime = responseTime
		}
	}

	// Parse custom metrics
	for key, value := range metricsData {
		if key != "instance_id" && key != "timestamp" && key != "request_count" &&
		   key != "error_count" && key != "response_time" {
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				metrics.CustomMetrics[key] = floatVal
			}
		}
	}

	return metrics, nil
}

// UpdateServiceMetrics updates metrics for a service
func (r *ServiceDiscoveryRedisRepository) UpdateServiceMetrics(ctx context.Context, metrics *models.ServiceMetrics) error {
	metricsKey := r.getServiceMetricsKey(metrics.ServiceName)

	// Prepare metrics data
	metricsData := map[string]interface{}{
		"instance_id":    metrics.InstanceID,
		"timestamp":      metrics.Timestamp.Format(time.RFC3339),
		"request_count":  metrics.RequestCount,
		"error_count":    metrics.ErrorCount,
		"response_time":  metrics.ResponseTime.String(),
	}

	// Add custom metrics
	for key, value := range metrics.CustomMetrics {
		metricsData[key] = value
	}

	err := r.client.HMSet(ctx, metricsKey, metricsData).Err()
	if err != nil {
		return fmt.Errorf("failed to update service metrics: %w", err)
	}

	// Set TTL for metrics
	err = r.client.Expire(ctx, metricsKey, r.config.DefaultTTL).Err()
	if err != nil {
		r.logger.WithError(err).Warn("Failed to set TTL for metrics")
	}

	return nil
}

// Helper methods
func (r *ServiceDiscoveryRedisRepository) getServiceKey(serviceName, serviceID string) string {
	return fmt.Sprintf("services:%s:%s", serviceName, serviceID)
}

func (r *ServiceDiscoveryRedisRepository) getServiceIndexKey(serviceName string) string {
	return fmt.Sprintf("service_index:%s", serviceName)
}

func (r *ServiceDiscoveryRedisRepository) getServiceMetricsKey(serviceName string) string {
	return fmt.Sprintf("metrics:%s", serviceName)
}