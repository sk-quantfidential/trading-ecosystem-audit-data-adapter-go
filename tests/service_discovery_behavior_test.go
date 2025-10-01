package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// ServiceDiscoveryBehaviorTestSuite tests the behavior of service discovery repository operations
type ServiceDiscoveryBehaviorTestSuite struct {
	BehaviorTestSuite
}

// TestServiceDiscoveryBehaviorSuite runs the service discovery behavior test suite
func TestServiceDiscoveryBehaviorSuite(t *testing.T) {
	suite.Run(t, new(ServiceDiscoveryBehaviorTestSuite))
}

// TestServiceDiscoveryBasicLifecycle tests basic service lifecycle
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryBasicLifecycle() {
	suite.T().Log("Testing service discovery basic lifecycle")
	suite.RunScenario("service_discovery_lifecycle")
}

// TestServiceDiscoveryMultipleServices tests managing multiple services
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryMultipleServices() {
	var (
		serviceName = "multi-test-service"
		instance1ID = GenerateTestID("instance1")
		instance2ID = GenerateTestID("instance2")
		instance3ID = GenerateTestID("instance3")
	)

	suite.Given("multiple instances of the same service", func() {
		services := []*models.ServiceRegistration{
			suite.CreateTestServiceRegistration(instance1ID, func(s *models.ServiceRegistration) {
				s.Name = serviceName
				s.Host = "host1.example.com"
				s.GRPCPort = 9001
				s.HTTPPort = 8001
				s.Status = "healthy"
			}),
			suite.CreateTestServiceRegistration(instance2ID, func(s *models.ServiceRegistration) {
				s.Name = serviceName
				s.Host = "host2.example.com"
				s.GRPCPort = 9002
				s.HTTPPort = 8002
				s.Status = "healthy"
			}),
			suite.CreateTestServiceRegistration(instance3ID, func(s *models.ServiceRegistration) {
				s.Name = serviceName
				s.Host = "host3.example.com"
				s.GRPCPort = 9003
				s.HTTPPort = 8003
				s.Status = "unhealthy"
			}),
		}

		for _, service := range services {
			err := suite.adapter.RegisterService(suite.ctx, service)
			suite.Require().NoError(err)
			suite.trackCreatedService(service.ID)
		}
	}).When("discovering services by name", func() {
		allServices, err := suite.adapter.GetServicesByName(suite.ctx, serviceName)
		suite.Require().NoError(err)

		suite.Then("all instances should be returned", func() {
			suite.Len(allServices, 3, "Should return all 3 service instances")

			instanceIDs := make([]string, len(allServices))
			for i, service := range allServices {
				instanceIDs[i] = service.ID
				suite.Equal(serviceName, service.Name)
			}

			suite.Contains(instanceIDs, instance1ID)
			suite.Contains(instanceIDs, instance2ID)
			suite.Contains(instanceIDs, instance3ID)
		})
	}).And("when discovering only healthy services", func() {
		healthyServices, err := suite.adapter.GetHealthyServices(suite.ctx, serviceName)
		suite.Require().NoError(err)

		suite.Then("only healthy instances should be returned", func() {
			suite.Len(healthyServices, 2, "Should return only 2 healthy service instances")

			for _, service := range healthyServices {
				suite.Equal("healthy", service.Status)
				suite.Equal(serviceName, service.Name)
			}
		})
	})
}

// TestServiceDiscoveryHeartbeat tests heartbeat functionality
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryHeartbeat() {
	var (
		serviceID   = GenerateTestID("heartbeat-service")
		serviceName = "heartbeat-test-service"
	)

	suite.Given("a registered service", func() {
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
			s.LastSeen = time.Now().Add(-5 * time.Minute) // 5 minutes ago
		})

		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("updating the service heartbeat", func() {
		beforeUpdate := time.Now()
		err := suite.adapter.UpdateHeartbeat(suite.ctx, serviceID)
		suite.Require().NoError(err)
		afterUpdate := time.Now()

		suite.Then("the last seen timestamp should be updated", func() {
			service, err := suite.adapter.GetService(suite.ctx, serviceID)
			suite.Require().NoError(err)

			suite.True(service.LastSeen.After(beforeUpdate.Add(-1*time.Second)),
				"LastSeen should be after update start time")
			suite.True(service.LastSeen.Before(afterUpdate.Add(1*time.Second)),
				"LastSeen should be before update end time")
		})
	})
}

// TestServiceDiscoveryStaleServiceCleanup tests stale service cleanup
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryStaleServiceCleanup() {
	var (
		staleServiceID   = GenerateTestID("stale-service")
		activeServiceID  = GenerateTestID("active-service")
		serviceName      = "cleanup-test-service"
		staleThreshold   = 2 * time.Minute
	)

	suite.Given("stale and active services", func() {
		// Create stale service (last seen > threshold)
		staleService := suite.CreateTestServiceRegistration(staleServiceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
			s.LastSeen = time.Now().Add(-5 * time.Minute) // 5 minutes ago (stale)
		})
		err := suite.adapter.RegisterService(suite.ctx, staleService)
		suite.Require().NoError(err)

		// Create active service (recently seen)
		activeService := suite.CreateTestServiceRegistration(activeServiceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
			s.LastSeen = time.Now().Add(-30 * time.Second) // 30 seconds ago (active)
		})
		err = suite.adapter.RegisterService(suite.ctx, activeService)
		suite.Require().NoError(err)
		suite.trackCreatedService(activeServiceID) // Only track active service for cleanup
	}).When("cleaning up stale services", func() {
		cleanedCount, err := suite.adapter.CleanupStaleServices(suite.ctx, staleThreshold)
		suite.Require().NoError(err)

		suite.Then("stale services should be removed", func() {
			suite.Greater(cleanedCount, int64(0), "Should cleanup at least one stale service")

			// Stale service should be removed
			_, err = suite.adapter.GetService(suite.ctx, staleServiceID)
			suite.Error(err, "Stale service should be removed")

			// Active service should still exist
			_, err = suite.adapter.GetService(suite.ctx, activeServiceID)
			suite.NoError(err, "Active service should still exist")
		})
	})
}

// TestServiceDiscoveryMetrics tests service metrics functionality
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryMetrics() {
	var (
		serviceName  = "metrics-test-service"
		instanceID   = GenerateTestID("metrics-instance")
	)

	suite.Given("a service with metrics to track", func() {
		service := suite.CreateTestServiceRegistration(instanceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
		})

		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(instanceID)
	}).When("updating service metrics", func() {
		metrics := &models.ServiceMetrics{
			ServiceName:   serviceName,
			InstanceID:    instanceID,
			Timestamp:     time.Now(),
			RequestCount:  1000,
			ErrorCount:    50,
			ResponseTime:  250 * time.Millisecond,
			CustomMetrics: map[string]float64{
				"cpu_usage":    75.5,
				"memory_usage": 80.2,
				"disk_usage":   45.0,
			},
		}

		err := suite.adapter.UpdateServiceMetrics(suite.ctx, metrics)
		suite.Require().NoError(err)

		suite.Then("metrics should be retrievable", func() {
			retrievedMetrics, err := suite.adapter.GetServiceMetrics(suite.ctx, serviceName)
			suite.Require().NoError(err)
			suite.Require().NotNil(retrievedMetrics)

			suite.Equal(serviceName, retrievedMetrics.ServiceName)
			suite.Equal(instanceID, retrievedMetrics.InstanceID)
			suite.Equal(int64(1000), retrievedMetrics.RequestCount)
			suite.Equal(int64(50), retrievedMetrics.ErrorCount)
			suite.Equal(250*time.Millisecond, retrievedMetrics.ResponseTime)

			// Check custom metrics
			suite.Equal(75.5, retrievedMetrics.CustomMetrics["cpu_usage"])
			suite.Equal(80.2, retrievedMetrics.CustomMetrics["memory_usage"])
			suite.Equal(45.0, retrievedMetrics.CustomMetrics["disk_usage"])
		})
	})
}

// TestServiceDiscoveryServiceUpdate tests service information updates
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryServiceUpdate() {
	var (
		serviceID   = GenerateTestID("update-service")
		serviceName = "update-test-service"
	)

	suite.Given("a registered service", func() {
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Version = "1.0.0"
			s.Status = "healthy"
			s.Metadata = map[string]string{
				"environment": "development",
				"region":      "us-east-1",
			}
		})

		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("updating service information", func() {
		updatedService := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Version = "1.1.0" // Updated version
			s.Status = "degraded" // Updated status
			s.Metadata = map[string]string{
				"environment": "production", // Updated environment
				"region":      "us-east-1",
				"new_field":   "new_value", // Added field
			}
		})

		err := suite.adapter.UpdateService(suite.ctx, updatedService)
		suite.Require().NoError(err)

		suite.Then("the service should have updated information", func() {
			service, err := suite.adapter.GetService(suite.ctx, serviceID)
			suite.Require().NoError(err)

			suite.Equal("1.1.0", service.Version)
			suite.Equal("degraded", service.Status)
			suite.Equal("production", service.Metadata["environment"])
			suite.Equal("new_value", service.Metadata["new_field"])
		})
	})
}

// TestServiceDiscoveryLoadBalancing tests service selection for load balancing
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryLoadBalancing() {
	var (
		serviceName = "loadbalancer-test-service"
		instanceIDs []string
	)

	suite.Given("multiple healthy service instances", func() {
		// Create 5 healthy instances
		for i := 0; i < 5; i++ {
			instanceID := GenerateTestID("lb-instance")
			service := suite.CreateTestServiceRegistration(instanceID, func(s *models.ServiceRegistration) {
				s.Name = serviceName
				s.Host = fmt.Sprintf("host%d.example.com", i+1)
				s.GRPCPort = 9000 + i
				s.HTTPPort = 8000 + i
				s.Status = "healthy"
			})

			err := suite.adapter.RegisterService(suite.ctx, service)
			suite.Require().NoError(err)
			suite.trackCreatedService(instanceID)
			instanceIDs = append(instanceIDs, instanceID)
		}
	}).When("discovering healthy services for load balancing", func() {
		healthyServices, err := suite.adapter.GetHealthyServices(suite.ctx, serviceName)
		suite.Require().NoError(err)

		suite.Then("all healthy instances should be available", func() {
			suite.Len(healthyServices, 5, "Should return all 5 healthy instances")

			// Verify all instances are present
			foundIDs := make(map[string]bool)
			for _, service := range healthyServices {
				foundIDs[service.ID] = true
				suite.Equal("healthy", service.Status)
				suite.Equal(serviceName, service.Name)
			}

			for _, expectedID := range instanceIDs {
				suite.True(foundIDs[expectedID], "Instance %s should be found", expectedID)
			}
		}).And("services should have different endpoints", func() {
			hostPorts := make(map[string]bool)
			for _, service := range healthyServices {
				hostPort := fmt.Sprintf("%s:%d", service.Host, service.GRPCPort)
				suite.False(hostPorts[hostPort], "Each service should have unique host:port combination")
				hostPorts[hostPort] = true
			}
		})
	})
}

// TestServiceDiscoveryVersioning tests service versioning support
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryVersioning() {
	var (
		serviceName = "versioning-test-service"
		v1InstanceID = GenerateTestID("v1-instance")
		v2InstanceID = GenerateTestID("v2-instance")
	)

	suite.Given("services with different versions", func() {
		// Register v1.0.0 instance
		v1Service := suite.CreateTestServiceRegistration(v1InstanceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Version = "1.0.0"
			s.Host = "v1.example.com"
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, v1Service)
		suite.Require().NoError(err)
		suite.trackCreatedService(v1InstanceID)

		// Register v2.0.0 instance
		v2Service := suite.CreateTestServiceRegistration(v2InstanceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Version = "2.0.0"
			s.Host = "v2.example.com"
			s.Status = "healthy"
		})
		err = suite.adapter.RegisterService(suite.ctx, v2Service)
		suite.Require().NoError(err)
		suite.trackCreatedService(v2InstanceID)
	}).When("discovering all service versions", func() {
		allServices, err := suite.adapter.GetServicesByName(suite.ctx, serviceName)
		suite.Require().NoError(err)

		suite.Then("both versions should be available", func() {
			suite.Len(allServices, 2, "Should return both service versions")

			versions := make(map[string]string) // version -> instanceID
			for _, service := range allServices {
				versions[service.Version] = service.ID
			}

			suite.Equal(v1InstanceID, versions["1.0.0"], "v1.0.0 instance should be found")
			suite.Equal(v2InstanceID, versions["2.0.0"], "v2.0.0 instance should be found")
		})
	})
}

// TestServiceDiscoveryPerformance tests performance of service discovery operations
func (suite *ServiceDiscoveryBehaviorTestSuite) TestServiceDiscoveryPerformance() {
	var (
		serviceName = "performance-test-service"
		serviceIDs  []string
	)

	suite.Given("performance test configuration", func() {
		// Skip if in CI to avoid flaky tests
		if IsCI() {
			suite.T().Skip("Skipping performance test in CI environment")
		}
	}).When("registering many services", func() {
		suite.AssertPerformance("register 50 services", 5*time.Second, func() {
			for i := 0; i < 50; i++ {
				serviceID := GenerateTestID("perf-service")
				service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
					s.Name = serviceName
					s.Host = fmt.Sprintf("host%d.example.com", i)
					s.GRPCPort = 9000 + i
					s.Status = "healthy"
				})

				err := suite.adapter.RegisterService(suite.ctx, service)
				suite.Require().NoError(err)
				serviceIDs = append(serviceIDs, serviceID)
			}
		})

		// Track for cleanup
		for _, serviceID := range serviceIDs {
			suite.trackCreatedService(serviceID)
		}
	}).Then("discovery performance should be acceptable", func() {
		suite.AssertPerformance("discover 50 services", 1*time.Second, func() {
			_, err := suite.adapter.GetServicesByName(suite.ctx, serviceName)
			suite.Require().NoError(err)
		})
	}).And("individual service lookup should be fast", func() {
		suite.AssertPerformance("lookup individual service", 100*time.Millisecond, func() {
			_, err := suite.adapter.GetService(suite.ctx, serviceIDs[0])
			suite.Require().NoError(err)
		})
	})
}