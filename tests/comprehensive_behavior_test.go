package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// ComprehensiveBehaviorTestSuite runs all behavior tests in a comprehensive manner
type ComprehensiveBehaviorTestSuite struct {
	BehaviorTestSuite
	runner *BehaviorTestRunner
}

// TestComprehensiveBehaviorSuite is the main entry point for all behavior tests
func TestComprehensiveBehaviorSuite(t *testing.T) {
	suite.Run(t, new(ComprehensiveBehaviorTestSuite))
}

// SetupSuite initializes the comprehensive test suite
func (suite *ComprehensiveBehaviorTestSuite) SetupSuite() {
	// Call parent setup
	suite.BehaviorTestSuite.SetupSuite()

	// Initialize test runner
	suite.runner = NewBehaviorTestRunner()
	suite.runner.PrintTestEnvironmentInfo()

	suite.T().Log("=== Starting Comprehensive Behavior Test Suite ===")
}

// TearDownSuite cleans up after all tests
func (suite *ComprehensiveBehaviorTestSuite) TearDownSuite() {
	suite.T().Log("=== Comprehensive Behavior Test Suite Completed ===")

	// Call parent teardown
	suite.BehaviorTestSuite.TearDownSuite()
}

// TestAuditDataAdapterBehavior tests the complete audit data adapter behavior
func (suite *ComprehensiveBehaviorTestSuite) TestAuditDataAdapterBehavior() {
	suite.T().Log("Testing comprehensive audit data adapter behavior")

	// Test basic functionality first
	suite.testBasicFunctionality()

	// Test advanced scenarios
	suite.testAdvancedScenarios()

	// Test error conditions
	suite.testErrorConditions()

	// Test performance characteristics (if enabled)
	if !GetEnvAsBool("SKIP_PERFORMANCE_TESTS", IsCI()) {
		suite.testPerformanceCharacteristics()
	}
}

// testBasicFunctionality tests basic CRUD operations across all repositories
func (suite *ComprehensiveBehaviorTestSuite) testBasicFunctionality() {
	suite.T().Log("Testing basic functionality across all repositories")

	// Test audit events
	suite.Run("BasicAuditEvents", func() {
		suite.RunScenario("audit_event_lifecycle")
	})

	// Test service discovery
	suite.Run("BasicServiceDiscovery", func() {
		suite.RunScenario("service_discovery_lifecycle")
	})

	// Test cache operations
	suite.Run("BasicCacheOperations", func() {
		suite.RunScenario("cache_operations")
	})
}

// testAdvancedScenarios tests complex business scenarios
func (suite *ComprehensiveBehaviorTestSuite) testAdvancedScenarios() {
	suite.T().Log("Testing advanced scenarios")

	// Test bulk operations
	suite.Run("BulkOperations", func() {
		suite.RunScenario("bulk_operations")
	})

	// Test transaction consistency
	suite.Run("TransactionConsistency", func() {
		suite.RunScenario("transaction_rollback")
	})

	// Test cross-repository data consistency
	suite.testCrossRepositoryConsistency()

	// Test complex queries and filtering
	suite.testComplexQueries()
}

// testErrorConditions tests error handling and recovery
func (suite *ComprehensiveBehaviorTestSuite) testErrorConditions() {
	suite.T().Log("Testing error conditions and recovery")

	suite.testInvalidOperations()
	suite.testConnectionRecovery()
	suite.testDataIntegrityUnderErrors()
}

// testPerformanceCharacteristics tests performance aspects
func (suite *ComprehensiveBehaviorTestSuite) testPerformanceCharacteristics() {
	suite.T().Log("Testing performance characteristics")

	suite.testThroughput()
	suite.testLatency()
	suite.testScalability()
}

// testCrossRepositoryConsistency tests data consistency across different repositories
func (suite *ComprehensiveBehaviorTestSuite) testCrossRepositoryConsistency() {
	var (
		serviceName = "cross-repo-test-service"
		serviceID   = GenerateTestID("cross-service")
		eventID     = GenerateTestUUID()
		cacheKey    = "cross:test:" + GenerateTestID("cache")
	)

	suite.Given("data stored across all repositories", func() {
		// Store in service repository
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)

		// Store in audit repository
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.ServiceName = serviceName
			e.EventType = "cross-repo-test"
		})
		err = suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)

		// Store in cache
		cacheData := map[string]interface{}{
			"service_id": serviceID,
			"event_id":   eventID,
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		err = suite.adapter.Set(suite.ctx, cacheKey, cacheData, 10*time.Minute)
		suite.Require().NoError(err)
	}).When("retrieving data from all repositories", func() {
		// This is done in the Then block
	}).Then("all data should be consistent and linked", func() {
		// Verify service data
		service, err := suite.adapter.GetService(suite.ctx, serviceID)
		suite.Require().NoError(err)
		suite.Equal(serviceName, service.Name)

		// Verify event data
		event, err := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(err)
		suite.Equal(serviceName, event.ServiceName)

		// Verify cache data
		var cacheData map[string]interface{}
		err = suite.adapter.Get(suite.ctx, cacheKey, &cacheData)
		suite.Require().NoError(err)
		suite.Equal(serviceID, cacheData["service_id"])
		suite.Equal(eventID, cacheData["event_id"])

		// Verify consistency across repositories
		suite.Equal(service.Name, event.ServiceName, "Service name should match across repositories")
	})
}

// testComplexQueries tests complex querying scenarios
func (suite *ComprehensiveBehaviorTestSuite) testComplexQueries() {
	var (
		serviceName = "complex-query-service"
		eventCount  = 15
		eventIDs    []string
	)

	suite.Given("a variety of audit events for complex querying", func() {
		timestamps := NewTestTimestamps()

		// Create events with different characteristics
		for i := 0; i < eventCount; i++ {
			eventID := GenerateTestUUID()
			event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
				e.ServiceName = serviceName
				e.EventType = fmt.Sprintf("event-type-%d", i%3) // 3 different types
				e.Status = []models.AuditEventStatus{
					models.AuditEventStatusPending,
					models.AuditEventStatusProcessed,
					models.AuditEventStatusFailed,
				}[i%3] // Rotate through statuses
				e.Timestamp = timestamps.MinutesAgo(i * 10) // Spread over time
				e.Tags = []string{
					fmt.Sprintf("priority-%d", i%2), // priority-0 or priority-1
					fmt.Sprintf("category-%d", i%4), // category-0 to category-3
				}
				e.Metadata = json.RawMessage(fmt.Sprintf(`{
					"batch_id": %d,
					"priority": %d,
					"size": %d
				}`, i/5, i%2, (i+1)*100))
			})

			err := suite.adapter.Create(suite.ctx, event)
			suite.Require().NoError(err)
			eventIDs = append(eventIDs, eventID)
			suite.trackCreatedEvent(eventID)
		}
	}).When("performing complex queries", func() {
		// Query 1: Events of specific type with specific status
		eventType := "event-type-1"
		status := models.AuditEventStatusProcessed
		query1 := models.AuditQuery{
			ServiceName: &serviceName,
			EventType:   &eventType,
			Status:      &status,
			Limit:       10,
		}

		results1, err := suite.adapter.Query(suite.ctx, query1)
		suite.Require().NoError(err)

		suite.Then("type and status filtered query should return correct results", func() {
			suite.NotEmpty(results1, "Should find events matching type and status")
			for _, event := range results1 {
				suite.Equal(eventType, event.EventType)
				suite.Equal(status, event.Status)
			}
		})

		// Query 2: Time range query
		timestamps := NewTestTimestamps()
		startTime := timestamps.MinutesAgo(100)
		endTime := timestamps.MinutesAgo(50)
		query2 := models.AuditQuery{
			ServiceName: &serviceName,
			StartTime:   &startTime,
			EndTime:     &endTime,
			Limit:       20,
			SortBy:      "timestamp",
			SortOrder:   "desc",
		}

		results2, err := suite.adapter.Query(suite.ctx, query2)
		suite.Require().NoError(err)

		suite.And("time range query should return events within the range", func() {
			for _, event := range results2 {
				suite.True(event.Timestamp.After(startTime) || event.Timestamp.Equal(startTime))
				suite.True(event.Timestamp.Before(endTime) || event.Timestamp.Equal(endTime))
			}
		})

		// Query 3: Tag-based query
		query3 := models.AuditQuery{
			ServiceName: &serviceName,
			Tags:        []string{"priority-1"},
			Limit:       10,
		}

		results3, err := suite.adapter.Query(suite.ctx, query3)
		suite.Require().NoError(err)

		suite.And("tag-based query should return events with matching tags", func() {
			for _, event := range results3 {
				suite.Contains(event.Tags, "priority-1")
			}
		})
	})
}

// testInvalidOperations tests handling of invalid operations
func (suite *ComprehensiveBehaviorTestSuite) testInvalidOperations() {
	suite.Given("invalid operation scenarios", func() {
		// Test invalid audit event creation
		invalidEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ID = "" // Invalid empty ID
		})

		err := suite.adapter.Create(suite.ctx, invalidEvent)
		suite.Then("invalid event creation should fail gracefully", func() {
			suite.Error(err, "Should return error for invalid event")
		})

		// Test non-existent event retrieval
		_, err = suite.adapter.GetByID(suite.ctx, "non-existent-event-id")
		suite.And("non-existent event retrieval should fail gracefully", func() {
			suite.Error(err, "Should return error for non-existent event")
		})

		// Test invalid cache operations
		err = suite.adapter.Get(suite.ctx, "non-existent-cache-key", nil)
		suite.And("invalid cache operations should fail gracefully", func() {
			suite.Error(err, "Should return error for non-existent cache key")
		})
	})
}

// testConnectionRecovery tests connection recovery scenarios
func (suite *ComprehensiveBehaviorTestSuite) testConnectionRecovery() {
	// This is a placeholder for connection recovery testing
	// In a real scenario, you might simulate connection failures
	suite.Given("connection recovery scenarios", func() {
		// Test health check
		err := suite.adapter.Health(suite.ctx)
		suite.Then("health check should pass under normal conditions", func() {
			suite.NoError(err, "Health check should pass")
		})
	})
}

// testDataIntegrityUnderErrors tests data integrity when errors occur
func (suite *ComprehensiveBehaviorTestSuite) testDataIntegrityUnderErrors() {
	var (
		validEventID   = GenerateTestUUID()
		invalidEventID = GenerateTestUUID()
	)

	suite.Given("mixed valid and invalid operations", func() {
		// Create valid event
		validEvent := suite.CreateTestAuditEvent(validEventID, func(e *models.AuditEvent) {
			e.ServiceName = "integrity-test-service"
			e.EventType = "integrity-test"
		})
		err := suite.adapter.Create(suite.ctx, validEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(validEventID)

		// Attempt invalid operation
		invalidEvent := suite.CreateTestAuditEvent(invalidEventID, func(e *models.AuditEvent) {
			e.ID = "" // Make it invalid by clearing the ID
			e.ServiceName = "integrity-test-service"
			e.EventType = "integrity-test"
		})
		err = suite.adapter.Create(suite.ctx, invalidEvent)
		suite.Require().Error(err, "Invalid operation should fail")

		suite.Then("valid data should remain intact after invalid operations", func() {
			// Verify valid event still exists and is correct
			retrievedEvent, err := suite.adapter.GetByID(suite.ctx, validEventID)
			suite.Require().NoError(err)
			suite.Equal(validEventID, retrievedEvent.ID)
			suite.Equal("integrity-test-service", retrievedEvent.ServiceName)

			// Verify system is still healthy
			err = suite.adapter.Health(suite.ctx)
			suite.NoError(err, "System should remain healthy after error")
		})
	})
}

// testThroughput tests throughput characteristics
func (suite *ComprehensiveBehaviorTestSuite) testThroughput() {
	var eventIDs []string

	suite.Given("throughput testing scenario", func() {
		eventCount := GetEnvAsInt("TEST_THROUGHPUT_SIZE", 25)

		suite.AssertPerformance("create events for throughput test", 5*time.Second, func() {
			for i := 0; i < eventCount; i++ {
				eventID := GenerateTestUUID()
				event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
					e.ServiceName = "throughput-test-service"
					e.EventType = "throughput-test"
				})

				err := suite.adapter.Create(suite.ctx, event)
				suite.Require().NoError(err)
				eventIDs = append(eventIDs, eventID)
			}
		})

		// Track for cleanup
		for _, eventID := range eventIDs {
			suite.trackCreatedEvent(eventID)
		}

		suite.Then("throughput should meet acceptable levels", func() {
			suite.T().Logf("Created %d events for throughput testing", len(eventIDs))
		})
	})
}

// testLatency tests latency characteristics
func (suite *ComprehensiveBehaviorTestSuite) testLatency() {
	var eventID = GenerateTestUUID()

	suite.Given("latency testing scenario", func() {
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.ServiceName = "latency-test-service"
			e.EventType = "latency-test"
		})

		suite.AssertPerformance("single event creation latency", 100*time.Millisecond, func() {
			err := suite.adapter.Create(suite.ctx, event)
			suite.Require().NoError(err)
		})

		suite.trackCreatedEvent(eventID)

		suite.AssertPerformance("single event retrieval latency", 50*time.Millisecond, func() {
			_, err := suite.adapter.GetByID(suite.ctx, eventID)
			suite.Require().NoError(err)
		})

		suite.Then("latency should be within acceptable bounds", func() {
			suite.T().Log("Latency tests completed within acceptable bounds")
		})
	})
}

// testScalability tests scalability characteristics
func (suite *ComprehensiveBehaviorTestSuite) testScalability() {
	suite.Given("scalability testing scenario", func() {
		// Test with increasing loads
		sizes := []int{10, 25, 50}

		for _, size := range sizes {
			suite.Run(fmt.Sprintf("Scalability_%d_events", size), func() {
				var eventIDs []string

				suite.AssertPerformance(fmt.Sprintf("create %d events", size),
					time.Duration(size)*50*time.Millisecond, func() {
					for i := 0; i < size; i++ {
						eventID := GenerateTestUUID()
						event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
							e.ServiceName = "scalability-test-service"
							e.EventType = "scalability-test"
							e.Metadata = json.RawMessage(fmt.Sprintf(`{
								"batch_size": %d,
								"index": %d
							}`, size, i))
						})

						err := suite.adapter.Create(suite.ctx, event)
						suite.Require().NoError(err)
						eventIDs = append(eventIDs, eventID)
					}
				})

				// Track for cleanup
				for _, eventID := range eventIDs {
					suite.trackCreatedEvent(eventID)
				}

				suite.T().Logf("Scalability test with %d events completed", size)
			})
		}

		suite.Then("scalability should be acceptable across different loads", func() {
			suite.T().Log("Scalability tests completed successfully")
		})
	})
}