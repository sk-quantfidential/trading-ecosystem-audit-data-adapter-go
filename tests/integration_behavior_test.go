package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// IntegrationBehaviorTestSuite tests the behavior of the complete data adapter integration
type IntegrationBehaviorTestSuite struct {
	BehaviorTestSuite
}

// TestIntegrationBehaviorSuite runs the integration behavior test suite
func TestIntegrationBehaviorSuite(t *testing.T) {
	suite.Run(t, new(IntegrationBehaviorTestSuite))
}

// TestFullAuditWorkflow tests a complete audit workflow from event creation to correlation
func (suite *IntegrationBehaviorTestSuite) TestFullAuditWorkflow() {
	var (
		traceID       = GenerateTestID("workflow-trace")
		parentEventID = GenerateTestUUID()
		childEventID  = GenerateTestUUID()
		serviceName   = "workflow-test-service"
		serviceID     = GenerateTestID("workflow-service")
	)

	suite.Given("a service is registered for audit workflow", func() {
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Version = "1.0.0"
			s.Status = "healthy"
		})

		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("a parent audit event is created", func() {
		parentEvent := suite.CreateTestAuditEvent(parentEventID, func(e *models.AuditEvent) {
			e.TraceID = traceID
			e.ServiceName = serviceName
			e.EventType = "workflow-start"
			e.Status = models.AuditEventStatusPending
			e.Metadata = json.RawMessage(`{
				"workflow_id": "wf-123",
				"step": "initialization"
			}`)
		})

		err := suite.adapter.Create(suite.ctx, parentEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(parentEventID)
	}).And("a child event is created with correlation", func() {
		childEvent := suite.CreateTestAuditEvent(childEventID, func(e *models.AuditEvent) {
			e.TraceID = traceID
			e.ServiceName = serviceName
			e.EventType = "workflow-step"
			e.Status = models.AuditEventStatusPending
			e.CorrelatedTo = []string{parentEventID}
			e.Metadata = json.RawMessage(`{
				"workflow_id": "wf-123",
				"step": "processing",
				"parent_step": "initialization"
			}`)
		})

		err := suite.adapter.Create(suite.ctx, childEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(childEventID)
	}).Then("the complete workflow should be traceable", func() {
		// Verify events can be found by trace ID
		traceEvents, err := suite.adapter.GetByTraceID(suite.ctx, traceID)
		suite.Require().NoError(err)
		suite.Len(traceEvents, 2, "Should find both events in trace")

		// Verify correlation
		correlatedEvents, err := suite.adapter.GetCorrelatedEvents(suite.ctx, parentEventID)
		suite.Require().NoError(err)
		suite.GreaterOrEqual(len(correlatedEvents), 1, "Should find correlated events")

		// Find child in correlated events
		var foundChild *models.AuditEvent
		for _, event := range correlatedEvents {
			if event.ID == childEventID {
				foundChild = event
				break
			}
		}
		suite.Require().NotNil(foundChild, "Child event should be in correlated events")
		suite.Contains(foundChild.CorrelatedTo, parentEventID, "Child should reference parent")
	}).And("service metrics can be updated", func() {
		metrics := &models.ServiceMetrics{
			ServiceName:   serviceName,
			InstanceID:    serviceID,
			Timestamp:     time.Now(),
			RequestCount:  100,
			ErrorCount:    5,
			ResponseTime:  200 * time.Millisecond,
			CustomMetrics: map[string]float64{
				"workflow_events": 2,
				"active_traces":   1,
			},
		}

		err := suite.adapter.UpdateServiceMetrics(suite.ctx, metrics)
		suite.Require().NoError(err)

		retrievedMetrics, err := suite.adapter.GetServiceMetrics(suite.ctx, serviceName)
		suite.Require().NoError(err)
		suite.Equal(float64(2), retrievedMetrics.CustomMetrics["workflow_events"])
	}).And("workflow completion can be cached", func() {
		workflowCacheKey := "workflow:wf-123:status"
		workflowStatus := map[string]interface{}{
			"status":      "completed",
			"events":      []string{parentEventID, childEventID},
			"trace_id":    traceID,
			"service":     serviceName,
			"completed_at": time.Now().Format(time.RFC3339),
		}

		err := suite.adapter.Set(suite.ctx, workflowCacheKey, workflowStatus, 1*time.Hour)
		suite.Require().NoError(err)

		var retrievedStatus map[string]interface{}
		err = suite.adapter.Get(suite.ctx, workflowCacheKey, &retrievedStatus)
		suite.Require().NoError(err)
		suite.Equal("completed", retrievedStatus["status"])
	})
}

// TestTransactionConsistency tests transaction behavior across the adapter
func (suite *IntegrationBehaviorTestSuite) TestTransactionConsistency() {
	suite.T().Log("Testing transaction consistency")
	suite.RunScenario("transaction_rollback")
}

// TestConcurrentOperations tests concurrent access to the adapter
func (suite *IntegrationBehaviorTestSuite) TestConcurrentOperations() {
	var (
		serviceName = "concurrent-test-service"
		eventCount  = 20
		serviceID   = GenerateTestID("concurrent-service")
		eventIDs    = make([]string, eventCount)
		errors      = make(chan error, eventCount)
		done        = make(chan bool, eventCount)
	)

	suite.Given("a service for concurrent testing", func() {
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
		})

		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("multiple events are created concurrently", func() {
		// Generate event IDs
		for i := 0; i < eventCount; i++ {
			eventIDs[i] = GenerateTestUUID()
		}

		// Create events concurrently
		for i := 0; i < eventCount; i++ {
			go func(index int) {
				defer func() { done <- true }()

				event := suite.CreateTestAuditEvent(eventIDs[index], func(e *models.AuditEvent) {
					e.ServiceName = serviceName
					e.EventType = "concurrent-test"
					e.Metadata = json.RawMessage(fmt.Sprintf(`{
						"thread_index": %d,
						"timestamp": "%s"
					}`, index, time.Now().Format(time.RFC3339Nano)))
				})

				if err := suite.adapter.Create(suite.ctx, event); err != nil {
					errors <- err
				} else {
					errors <- nil
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < eventCount; i++ {
			<-done
		}

		// Check for errors
		errorCount := 0
		for i := 0; i < eventCount; i++ {
			if err := <-errors; err != nil {
				suite.T().Logf("Concurrent operation error: %v", err)
				errorCount++
			}
		}

		// Track all events for cleanup
		for _, eventID := range eventIDs {
			suite.trackCreatedEvent(eventID)
		}

		suite.Then("all operations should succeed", func() {
			suite.Equal(0, errorCount, "No concurrent operations should fail")
		})
	}).And("all events should be retrievable", func() {
		successCount := 0
		for _, eventID := range eventIDs {
			if event, err := suite.adapter.GetByID(suite.ctx, eventID); err == nil && event != nil {
				suite.Equal(serviceName, event.ServiceName)
				suite.Equal("concurrent-test", event.EventType)
				successCount++
			}
		}

		suite.Equal(eventCount, successCount, "All events should be retrievable")
	}).And("service heartbeat updates should be consistent", func() {
		// Update heartbeat concurrently
		heartbeatErrors := make(chan error, 5)
		heartbeatDone := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func() {
				defer func() { heartbeatDone <- true }()
				if err := suite.adapter.UpdateHeartbeat(suite.ctx, serviceID); err != nil {
					heartbeatErrors <- err
				} else {
					heartbeatErrors <- nil
				}
			}()
		}

		for i := 0; i < 5; i++ {
			<-heartbeatDone
		}

		heartbeatErrorCount := 0
		for i := 0; i < 5; i++ {
			if err := <-heartbeatErrors; err != nil {
				heartbeatErrorCount++
			}
		}

		suite.Equal(0, heartbeatErrorCount, "Concurrent heartbeat updates should succeed")
	})
}

// TestDataConsistencyAcrossComponents tests data consistency between different repositories
func (suite *IntegrationBehaviorTestSuite) TestDataConsistencyAcrossComponents() {
	var (
		serviceName     = "consistency-test-service"
		serviceID       = GenerateTestID("consistency-service")
		eventID         = GenerateTestUUID()
		cacheKey        = "service:status:" + serviceID
		correlationID   = GenerateTestID("correlation")
	)

	suite.Given("data stored across all repository types", func() {
		// 1. Register service
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)

		// 2. Create audit event
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.ServiceName = serviceName
			e.EventType = "consistency-test"
			e.Status = models.AuditEventStatusProcessed
		})
		err = suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)

		// 3. Cache service status
		serviceStatus := map[string]interface{}{
			"service_id":   serviceID,
			"service_name": serviceName,
			"last_event":   eventID,
			"status":       "operational",
			"last_check":   time.Now().Format(time.RFC3339),
		}
		err = suite.adapter.Set(suite.ctx, cacheKey, serviceStatus, 30*time.Minute)
		suite.Require().NoError(err)

		// 4. Create correlation
		correlation := &models.AuditCorrelation{
			ID:              correlationID,
			SourceEventID:   eventID,
			TargetEventID:   eventID, // Self-reference for test
			CorrelationType: "consistency-test",
			Confidence:      0.95,
		}
		err = suite.adapter.CreateCorrelation(suite.ctx, correlation)
		suite.Require().NoError(err)
	}).When("retrieving data from all components", func() {
		// Retrieve service
		retrievedService, err := suite.adapter.GetService(suite.ctx, serviceID)
		suite.Require().NoError(err)

		// Retrieve event
		retrievedEvent, err := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(err)

		// Retrieve cached status
		var retrievedStatus map[string]interface{}
		err = suite.adapter.Get(suite.ctx, cacheKey, &retrievedStatus)
		suite.Require().NoError(err)

		// Retrieve correlation
		correlations, err := suite.adapter.GetCorrelationsByEvent(suite.ctx, eventID)
		suite.Require().NoError(err)

		suite.Then("all data should be consistent", func() {
			// Service data consistency
			suite.Equal(serviceName, retrievedService.Name)
			suite.Equal(serviceID, retrievedService.ID)
			suite.Equal("healthy", retrievedService.Status)

			// Event data consistency
			suite.Equal(eventID, retrievedEvent.ID)
			suite.Equal(serviceName, retrievedEvent.ServiceName)
			suite.Equal(models.AuditEventStatusProcessed, retrievedEvent.Status)

			// Cache data consistency
			suite.Equal(serviceID, retrievedStatus["service_id"])
			suite.Equal(serviceName, retrievedStatus["service_name"])
			suite.Equal(eventID, retrievedStatus["last_event"])

			// Correlation data consistency
			suite.NotEmpty(correlations, "Should have correlations")
			found := false
			for _, corr := range correlations {
				if corr.ID == correlationID {
					found = true
					suite.Equal(eventID, corr.SourceEventID)
					suite.Equal("consistency-test", corr.CorrelationType)
					suite.Equal(0.95, corr.Confidence)
					break
				}
			}
			suite.True(found, "Should find the created correlation")
		})
	})
}

// TestHealthAndMonitoring tests health and monitoring capabilities
func (suite *IntegrationBehaviorTestSuite) TestHealthAndMonitoring() {
	suite.Given("a fully operational data adapter", func() {
		// Create some test data to ensure systems are working
		testService := suite.CreateTestServiceRegistration(GenerateTestID("health-service"), func(s *models.ServiceRegistration) {
			s.Name = "health-test-service"
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, testService)
		suite.Require().NoError(err)
		suite.trackCreatedService(testService.ID)

		testEvent := suite.CreateTestAuditEvent(GenerateTestUUID(), func(e *models.AuditEvent) {
			e.ServiceName = "health-test-service"
			e.EventType = "health-check"
		})
		err = suite.adapter.Create(suite.ctx, testEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(testEvent.ID)

		cacheKey := "health:test:" + GenerateTestID("key")
		err = suite.adapter.Set(suite.ctx, cacheKey, "health-value", time.Minute)
		suite.Require().NoError(err)
	}).When("checking overall health", func() {
		err := suite.adapter.Health(suite.ctx)
		suite.Require().NoError(err)

		suite.Then("all components should be healthy", func() {
			// Health check passed - all components are responding
		})
	}).And("when checking individual component health", func() {
		// Check cache health
		err := suite.adapter.Ping(suite.ctx)
		suite.Require().NoError(err)

		suite.Then("cache should be responsive", func() {
			// Cache ping succeeded
		})
	}).And("when getting cache statistics", func() {
		stats, err := suite.adapter.GetStats(suite.ctx)
		suite.Require().NoError(err)

		suite.Then("statistics should be available", func() {
			suite.NotEmpty(stats, "Should have cache statistics")
		})
	})
}

// TestErrorRecovery tests error recovery and resilience
func (suite *IntegrationBehaviorTestSuite) TestErrorRecovery() {
	var (
		validEventID   = GenerateTestUUID()
		invalidEventID = "" // Invalid ID to trigger error
		serviceID      = GenerateTestID("recovery-service")
	)

	suite.Given("a mix of valid and invalid operations", func() {
		// Register a valid service first
		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = "recovery-test-service"
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("attempting operations with mixed validity", func() {
		// Valid operation
		validEvent := suite.CreateTestAuditEvent(validEventID, func(e *models.AuditEvent) {
			e.ServiceName = "recovery-test-service"
			e.EventType = "recovery-test"
		})
		err := suite.adapter.Create(suite.ctx, validEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(validEventID)

		// Invalid operation (should fail)
		invalidEvent := suite.CreateTestAuditEvent(invalidEventID, func(e *models.AuditEvent) {
			e.ServiceName = "recovery-test-service"
			e.EventType = "recovery-test"
		})
		err = suite.adapter.Create(suite.ctx, invalidEvent)
		suite.Require().Error(err, "Invalid event creation should fail")

		suite.Then("valid operations should succeed despite invalid ones", func() {
			// Verify valid event was created
			retrievedEvent, err := suite.adapter.GetByID(suite.ctx, validEventID)
			suite.Require().NoError(err)
			suite.Equal(validEventID, retrievedEvent.ID)

			// Verify service is still accessible
			retrievedService, err := suite.adapter.GetService(suite.ctx, serviceID)
			suite.Require().NoError(err)
			suite.Equal(serviceID, retrievedService.ID)

			// Verify adapter is still healthy
			err = suite.adapter.Health(suite.ctx)
			suite.Require().NoError(err)
		})
	})
}

// TestLargeDatasetOperations tests operations with larger datasets
func (suite *IntegrationBehaviorTestSuite) TestLargeDatasetOperations() {
	var (
		serviceName    = "large-dataset-service"
		serviceID      = GenerateTestID("large-service")
		eventCount     = 50 // Moderate size for integration test
		batchSize      = 10
		eventIDs       []string
	)

	suite.Given("a service for large dataset testing", func() {
		// Skip if in CI to avoid timeout issues
		if IsCI() {
			suite.T().Skip("Skipping large dataset test in CI environment")
		}

		service := suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Name = serviceName
			s.Status = "healthy"
		})
		err := suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).When("creating a large number of audit events", func() {
		suite.AssertPerformance("create 50 events in batches", 10*time.Second, func() {
			for batch := 0; batch < eventCount; batch += batchSize {
				batchEvents := make([]*models.AuditEvent, 0, batchSize)

				for i := 0; i < batchSize && batch+i < eventCount; i++ {
					eventID := GenerateTestUUID()
					event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
						e.ServiceName = serviceName
						e.EventType = "large-dataset-test"
						e.Metadata = json.RawMessage(fmt.Sprintf(`{
							"batch_number": %d,
							"event_index": %d,
							"dataset_size": %d
						}`, batch/batchSize, i, eventCount))
					})
					batchEvents = append(batchEvents, event)
					eventIDs = append(eventIDs, eventID)
				}

				err := suite.adapter.CreateBatch(suite.ctx, batchEvents)
				suite.Require().NoError(err)
			}
		})

		// Track all events for cleanup
		for _, eventID := range eventIDs {
			suite.trackCreatedEvent(eventID)
		}
	}).Then("all events should be queryable efficiently", func() {
		suite.AssertPerformance("query large dataset", 3*time.Second, func() {
			query := models.AuditQuery{
				ServiceName: &serviceName,
				EventType:   stringPtr("large-dataset-test"),
				Limit:       eventCount + 10, // Slightly more than created
				SortBy:      "timestamp",
				SortOrder:   "desc",
			}

			results, err := suite.adapter.Query(suite.ctx, query)
			suite.Require().NoError(err)
			suite.GreaterOrEqual(len(results), eventCount, "Should find all created events")
		})
	}).And("service metrics should reflect the activity", func() {
		metrics := &models.ServiceMetrics{
			ServiceName:   serviceName,
			InstanceID:    serviceID,
			Timestamp:     time.Now(),
			RequestCount:  int64(eventCount),
			ErrorCount:    0,
			ResponseTime:  100 * time.Millisecond,
			CustomMetrics: map[string]float64{
				"events_processed": float64(eventCount),
				"batch_operations": float64(eventCount / batchSize),
			},
		}

		err := suite.adapter.UpdateServiceMetrics(suite.ctx, metrics)
		suite.Require().NoError(err)

		retrievedMetrics, err := suite.adapter.GetServiceMetrics(suite.ctx, serviceName)
		suite.Require().NoError(err)
		suite.Equal(float64(eventCount), retrievedMetrics.CustomMetrics["events_processed"])
	})
}