package tests

import (
	"time"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// Common behavior scenarios that can be reused across test suites

// auditEventLifecycleScenario tests the complete lifecycle of an audit event
func (suite *BehaviorTestSuite) auditEventLifecycleScenario() {
	var (
		eventID = GenerateTestUUID()
		event   *models.AuditEvent
		err     error
	)

	suite.Given("an audit event with pending status", func() {
		event = suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.Status = models.AuditEventStatusPending
		})
	}).When("the event is created in the repository", func() {
		err = suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)
	}).Then("the event should be retrievable", func() {
		retrievedEvent, getErr := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(getErr)
		suite.Require().NotNil(retrievedEvent)
		suite.Equal(eventID, retrievedEvent.ID)
		suite.Equal(models.AuditEventStatusPending, retrievedEvent.Status)
	}).And("the event can be updated to processed status", func() {
		event.Status = models.AuditEventStatusProcessed
		updateErr := suite.adapter.Update(suite.ctx, event)
		suite.Require().NoError(updateErr)
	}).And("the updated status should be persisted", func() {
		updatedEvent, getErr := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(getErr)
		suite.Equal(models.AuditEventStatusProcessed, updatedEvent.Status)
	}).And("the event can be deleted", func() {
		deleteErr := suite.adapter.Delete(suite.ctx, eventID)
		suite.Require().NoError(deleteErr)
	}).And("the deleted event should not be retrievable", func() {
		_, getErr := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Error(getErr, "Should not be able to retrieve deleted event")
	})
}

// serviceDiscoveryLifecycleScenario tests the complete lifecycle of service registration
func (suite *BehaviorTestSuite) serviceDiscoveryLifecycleScenario() {
	var (
		serviceID = GenerateTestID("service")
		service   *models.ServiceRegistration
		err       error
	)

	suite.Given("a service registration with healthy status", func() {
		service = suite.CreateTestServiceRegistration(serviceID, func(s *models.ServiceRegistration) {
			s.Status = "healthy"
			s.Name = "test-lifecycle-service"
		})
	}).When("the service is registered", func() {
		err = suite.adapter.RegisterService(suite.ctx, service)
		suite.Require().NoError(err)
		suite.trackCreatedService(serviceID)
	}).Then("the service should be discoverable", func() {
		retrievedService, getErr := suite.adapter.GetService(suite.ctx, serviceID)
		suite.Require().NoError(getErr)
		suite.Require().NotNil(retrievedService)
		suite.Equal(serviceID, retrievedService.ID)
		suite.Equal("healthy", retrievedService.Status)
	}).And("the service should appear in service list by name", func() {
		services, listErr := suite.adapter.GetServicesByName(suite.ctx, "test-lifecycle-service")
		suite.Require().NoError(listErr)
		suite.Require().NotEmpty(services)

		found := false
		for _, s := range services {
			if s.ID == serviceID {
				found = true
				break
			}
		}
		suite.True(found, "Service should be found in services list")
	}).And("the service heartbeat can be updated", func() {
		updateErr := suite.adapter.UpdateHeartbeat(suite.ctx, serviceID)
		suite.Require().NoError(updateErr)
	}).And("the service can be unregistered", func() {
		unregisterErr := suite.adapter.UnregisterService(suite.ctx, serviceID)
		suite.Require().NoError(unregisterErr)
	}).And("the unregistered service should not be discoverable", func() {
		_, getErr := suite.adapter.GetService(suite.ctx, serviceID)
		suite.Error(getErr, "Should not be able to retrieve unregistered service")
	})
}

// cacheOperationsScenario tests cache functionality
func (suite *BehaviorTestSuite) cacheOperationsScenario() {
	var (
		key   = "test:cache:" + GenerateTestID("key")
		value = map[string]interface{}{
			"test_field": "test_value",
			"numeric":    42,
			"boolean":    true,
		}
		ttl = 30 * time.Second
	)

	suite.Given("a cache key-value pair", func() {
		// Key and value are defined above
	}).When("the value is stored in cache with TTL", func() {
		err := suite.adapter.Set(suite.ctx, key, value, ttl)
		suite.Require().NoError(err)
	}).Then("the value should be retrievable from cache", func() {
		var retrieved map[string]interface{}
		err := suite.adapter.Get(suite.ctx, key, &retrieved)
		suite.Require().NoError(err)
		suite.Equal(value["test_field"], retrieved["test_field"])
		suite.Equal(float64(42), retrieved["numeric"]) // JSON unmarshaling converts numbers to float64
		suite.Equal(true, retrieved["boolean"])
	}).And("the cache should confirm the key exists", func() {
		exists, err := suite.adapter.Exists(suite.ctx, key)
		suite.Require().NoError(err)
		suite.True(exists)
	}).And("the TTL should be set correctly", func() {
		remainingTTL, err := suite.adapter.GetTTL(suite.ctx, key)
		suite.Require().NoError(err)
		suite.Greater(remainingTTL, 20*time.Second, "TTL should be greater than 20 seconds")
		suite.LessOrEqual(remainingTTL, ttl, "TTL should not exceed set value")
	}).And("the key can be deleted from cache", func() {
		err := suite.adapter.Delete(suite.ctx, key)
		suite.Require().NoError(err)
	}).And("the deleted key should not exist", func() {
		exists, err := suite.adapter.Exists(suite.ctx, key)
		suite.Require().NoError(err)
		suite.False(exists)
	})
}

// transactionRollbackScenario tests transaction rollback behavior
func (suite *BehaviorTestSuite) transactionRollbackScenario() {
	var (
		eventID1 = GenerateTestUUID()
		eventID2 = GenerateTestUUID()
		event1   *models.AuditEvent
		event2   *models.AuditEvent
	)

	suite.Given("two audit events for transaction testing", func() {
		event1 = suite.CreateTestAuditEvent(eventID1)
		event2 = suite.CreateTestAuditEvent(eventID2, func(e *models.AuditEvent) {
			e.ID = "" // Invalid ID to cause failure
		})
	}).When("a transaction is started and first event is created", func() {
		tx, err := suite.adapter.BeginTransaction(suite.ctx)
		suite.Require().NoError(err)

		// Create first event successfully
		err = tx.AuditEvents().Create(suite.ctx, event1)
		suite.Require().NoError(err)

		// Try to create second event (should fail due to empty ID)
		err = tx.AuditEvents().Create(suite.ctx, event2)
		suite.Require().Error(err, "Second event creation should fail")

		// Rollback transaction
		rollbackErr := tx.Rollback(suite.ctx)
		suite.Require().NoError(rollbackErr)
	}).Then("neither event should exist after rollback", func() {
		_, err := suite.adapter.GetByID(suite.ctx, eventID1)
		suite.Error(err, "First event should not exist after rollback")

		_, err = suite.adapter.GetByID(suite.ctx, eventID2)
		suite.Error(err, "Second event should not exist after rollback")
	})
}

// bulkOperationsScenario tests bulk operations performance and correctness
func (suite *BehaviorTestSuite) bulkOperationsScenario() {
	var (
		eventCount = 10
		events     []*models.AuditEvent
		eventIDs   []string
	)

	suite.Given("multiple audit events for bulk testing", func() {
		events = make([]*models.AuditEvent, eventCount)
		eventIDs = make([]string, eventCount)

		for i := 0; i < eventCount; i++ {
			eventID := GenerateTestUUID()
			eventIDs[i] = eventID
			events[i] = suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
				e.ServiceName = "bulk-test-service"
				e.EventType = "bulk-test-event"
			})
		}
	}).When("events are created in bulk", func() {
		suite.AssertPerformance("bulk create", 5*time.Second, func() {
			err := suite.adapter.CreateBatch(suite.ctx, events)
			suite.Require().NoError(err)
		})

		// Track all events for cleanup
		for _, eventID := range eventIDs {
			suite.trackCreatedEvent(eventID)
		}
	}).Then("all events should be retrievable", func() {
		for _, eventID := range eventIDs {
			event, err := suite.adapter.GetByID(suite.ctx, eventID)
			suite.Require().NoError(err)
			suite.Require().NotNil(event)
			suite.Equal("bulk-test-service", event.ServiceName)
		}
	}).And("events can be queried by service name", func() {
		query := models.AuditQuery{
			ServiceName: stringPtr("bulk-test-service"),
			Limit:       20,
		}

		results, err := suite.adapter.Query(suite.ctx, query)
		suite.Require().NoError(err)
		suite.GreaterOrEqual(len(results), eventCount)
	}).And("events can be updated in bulk", func() {
		// Update all events to processed status
		for _, event := range events {
			event.Status = models.AuditEventStatusProcessed
		}

		suite.AssertPerformance("bulk update", 5*time.Second, func() {
			err := suite.adapter.UpdateBatch(suite.ctx, events)
			suite.Require().NoError(err)
		})
	}).And("all events should have updated status", func() {
		for _, eventID := range eventIDs {
			event, err := suite.adapter.GetByID(suite.ctx, eventID)
			suite.Require().NoError(err)
			suite.Equal(models.AuditEventStatusProcessed, event.Status)
		}
	})
}

// Helper function to create string pointer for optional query fields
func stringPtr(s string) *string {
	return &s
}