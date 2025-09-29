package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// AuditEventBehaviorTestSuite tests the behavior of audit event repository operations
type AuditEventBehaviorTestSuite struct {
	BehaviorTestSuite
}

// TestAuditEventBehaviorSuite runs the audit event behavior test suite
func TestAuditEventBehaviorSuite(t *testing.T) {
	suite.Run(t, new(AuditEventBehaviorTestSuite))
}

// TestAuditEventBasicCRUD tests basic CRUD operations for audit events
func (suite *AuditEventBehaviorTestSuite) TestAuditEventBasicCRUD() {
	suite.T().Log("Testing audit event basic CRUD operations")
	suite.RunScenario("audit_event_lifecycle")
}

// TestAuditEventCreateWithValidation tests event creation with validation
func (suite *AuditEventBehaviorTestSuite) TestAuditEventCreateWithValidation() {
	var eventID = GenerateTestID("validation-event")

	suite.Given("an audit event with all required fields", func() {
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.ServiceName = "validation-test-service"
			e.EventType = "validation-test-event"
			e.Status = models.AuditEventStatusPending
		})

		err := suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)
	}).Then("the event should be created successfully", func() {
		suite.AssertEventExists(eventID, models.AuditEventStatusPending)
	}).And("the event should have auto-generated timestamps", func() {
		event, err := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(err)
		suite.False(event.CreatedAt.IsZero(), "CreatedAt should be auto-generated")
		suite.False(event.UpdatedAt.IsZero(), "UpdatedAt should be auto-generated")
	})
}

// TestAuditEventQueryByTraceID tests querying events by trace ID
func (suite *AuditEventBehaviorTestSuite) TestAuditEventQueryByTraceID() {
	var (
		traceID  = GenerateTestID("trace")
		event1ID = GenerateTestID("event1")
		event2ID = GenerateTestID("event2")
		event3ID = GenerateTestID("event3")
	)

	suite.Given("multiple events with the same trace ID", func() {
		// Create events with same trace ID
		for _, eventID := range []string{event1ID, event2ID} {
			event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
				e.TraceID = traceID
				e.ServiceName = "trace-test-service"
			})
			err := suite.adapter.Create(suite.ctx, event)
			suite.Require().NoError(err)
			suite.trackCreatedEvent(eventID)
		}

		// Create event with different trace ID
		event3 := suite.CreateTestAuditEvent(event3ID, func(e *models.AuditEvent) {
			e.TraceID = GenerateTestID("different-trace")
			e.ServiceName = "trace-test-service"
		})
		err := suite.adapter.Create(suite.ctx, event3)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(event3ID)
	}).When("querying events by trace ID", func() {
		events, err := suite.adapter.GetByTraceID(suite.ctx, traceID)
		suite.Require().NoError(err)

		suite.Then("only events with matching trace ID should be returned", func() {
			suite.Len(events, 2, "Should return exactly 2 events with matching trace ID")

			for _, event := range events {
				suite.Equal(traceID, event.TraceID)
			}
		})
	})
}

// TestAuditEventQueryWithFilters tests complex querying with filters
func (suite *AuditEventBehaviorTestSuite) TestAuditEventQueryWithFilters() {
	var (
		serviceName = "filter-test-service"
		eventType   = "filter-test-event"
		startTime   = time.Now().Add(-1 * time.Hour)
		endTime     = time.Now()
		eventIDs    []string
	)

	suite.Given("multiple events with different attributes", func() {
		// Create events that match our filter criteria
		for i := 0; i < 3; i++ {
			eventID := GenerateTestID("filter-match")
			event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
				e.ServiceName = serviceName
				e.EventType = eventType
				e.Status = models.AuditEventStatusPending
				e.Timestamp = time.Now().Add(-30 * time.Minute) // Within time range
			})
			err := suite.adapter.Create(suite.ctx, event)
			suite.Require().NoError(err)
			suite.trackCreatedEvent(eventID)
			eventIDs = append(eventIDs, eventID)
		}

		// Create events that don't match
		for i := 0; i < 2; i++ {
			eventID := GenerateTestID("filter-nomatch")
			event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
				e.ServiceName = "different-service"
				e.EventType = "different-event"
				e.Status = models.AuditEventStatusProcessed
			})
			err := suite.adapter.Create(suite.ctx, event)
			suite.Require().NoError(err)
			suite.trackCreatedEvent(eventID)
		}
	}).When("querying with filters", func() {
		status := models.AuditEventStatusPending
		query := models.AuditQuery{
			ServiceName: &serviceName,
			EventType:   &eventType,
			Status:      &status,
			StartTime:   &startTime,
			EndTime:     &endTime,
			Limit:       10,
			SortBy:      "timestamp",
			SortOrder:   "desc",
		}

		events, err := suite.adapter.Query(suite.ctx, query)
		suite.Require().NoError(err)

		suite.Then("only matching events should be returned", func() {
			suite.Len(events, 3, "Should return exactly 3 matching events")

			for _, event := range events {
				suite.Equal(serviceName, event.ServiceName)
				suite.Equal(eventType, event.EventType)
				suite.Equal(models.AuditEventStatusPending, event.Status)
				suite.True(event.Timestamp.After(startTime))
				suite.True(event.Timestamp.Before(endTime))
			}
		})
	})
}

// TestAuditEventBulkOperations tests bulk create and update operations
func (suite *AuditEventBehaviorTestSuite) TestAuditEventBulkOperations() {
	suite.T().Log("Testing audit event bulk operations")
	suite.RunScenario("bulk_operations")
}

// TestAuditEventMetadataHandling tests metadata storage and retrieval
func (suite *AuditEventBehaviorTestSuite) TestAuditEventMetadataHandling() {
	var eventID = GenerateTestID("metadata-event")

	suite.Given("an audit event with complex metadata", func() {
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.Metadata = map[string]interface{}{
				"string_field":  "test_value",
				"numeric_field": 42.5,
				"boolean_field": true,
				"nested_object": map[string]interface{}{
					"nested_string": "nested_value",
					"nested_number": 123,
				},
				"array_field": []interface{}{"item1", "item2", "item3"},
			}
		})

		err := suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)
	}).When("the event is retrieved", func() {
		event, err := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(err)

		suite.Then("the metadata should be preserved correctly", func() {
			suite.Equal("test_value", event.Metadata["string_field"])
			suite.Equal(42.5, event.Metadata["numeric_field"])
			suite.Equal(true, event.Metadata["boolean_field"])

			// Check nested object
			nested, ok := event.Metadata["nested_object"].(map[string]interface{})
			suite.True(ok, "Nested object should be preserved")
			suite.Equal("nested_value", nested["nested_string"])
			suite.Equal(float64(123), nested["nested_number"]) // JSON numbers become float64

			// Check array
			array, ok := event.Metadata["array_field"].([]interface{})
			suite.True(ok, "Array should be preserved")
			suite.Len(array, 3)
			suite.Equal("item1", array[0])
		})
	})
}

// TestAuditEventTagsHandling tests tag storage and querying
func (suite *AuditEventBehaviorTestSuite) TestAuditEventTagsHandling() {
	var (
		eventID = GenerateTestID("tags-event")
		tags    = []string{"production", "critical", "security", "api"}
	)

	suite.Given("an audit event with multiple tags", func() {
		event := suite.CreateTestAuditEvent(eventID, func(e *models.AuditEvent) {
			e.Tags = tags
			e.ServiceName = "tags-test-service"
		})

		err := suite.adapter.Create(suite.ctx, event)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(eventID)
	}).When("the event is retrieved", func() {
		event, err := suite.adapter.GetByID(suite.ctx, eventID)
		suite.Require().NoError(err)

		suite.Then("all tags should be preserved", func() {
			suite.ElementsMatch(tags, event.Tags, "All tags should be preserved in correct order")
		})
	}).And("when querying by tags", func() {
		query := models.AuditQuery{
			Tags:  []string{"production", "critical"},
			Limit: 10,
		}

		events, err := suite.adapter.Query(suite.ctx, query)
		suite.Require().NoError(err)

		suite.Then("events with matching tags should be returned", func() {
			found := false
			for _, event := range events {
				if event.ID == eventID {
					found = true
					break
				}
			}
			suite.True(found, "Event with matching tags should be found")
		})
	})
}

// TestAuditEventCorrelation tests event correlation functionality
func (suite *AuditEventBehaviorTestSuite) TestAuditEventCorrelation() {
	var (
		parentEventID = GenerateTestID("parent-event")
		childEventID  = GenerateTestID("child-event")
	)

	suite.Given("a parent audit event", func() {
		parentEvent := suite.CreateTestAuditEvent(parentEventID, func(e *models.AuditEvent) {
			e.ServiceName = "correlation-test-service"
			e.EventType = "parent-event"
		})

		err := suite.adapter.Create(suite.ctx, parentEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(parentEventID)
	}).And("a child event correlated to the parent", func() {
		childEvent := suite.CreateTestAuditEvent(childEventID, func(e *models.AuditEvent) {
			e.ServiceName = "correlation-test-service"
			e.EventType = "child-event"
			e.CorrelatedTo = []string{parentEventID}
		})

		err := suite.adapter.Create(suite.ctx, childEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(childEventID)
	}).When("querying for correlated events", func() {
		correlatedEvents, err := suite.adapter.GetCorrelatedEvents(suite.ctx, parentEventID)
		suite.Require().NoError(err)

		suite.Then("the child event should be in the correlated events", func() {
			suite.NotEmpty(correlatedEvents, "Should find correlated events")

			found := false
			for _, event := range correlatedEvents {
				if event.ID == childEventID {
					found = true
					suite.Contains(event.CorrelatedTo, parentEventID, "Child event should reference parent")
					break
				}
			}
			suite.True(found, "Child event should be found in correlated events")
		})
	})
}

// TestAuditEventPerformance tests performance characteristics
func (suite *AuditEventBehaviorTestSuite) TestAuditEventPerformance() {
	var eventIDs []string

	suite.Given("performance test configuration", func() {
		// Skip if in CI to avoid flaky tests
		if IsCI() {
			suite.T().Skip("Skipping performance test in CI environment")
		}
	}).When("creating events individually", func() {
		suite.AssertPerformance("create 100 events individually", 10*time.Second, func() {
			for i := 0; i < 100; i++ {
				eventID := GenerateTestID("perf-event")
				event := suite.CreateTestAuditEvent(eventID)
				err := suite.adapter.Create(suite.ctx, event)
				suite.Require().NoError(err)
				eventIDs = append(eventIDs, eventID)
			}
		})

		// Track for cleanup
		for _, eventID := range eventIDs {
			suite.trackCreatedEvent(eventID)
		}
	}).Then("query performance should be acceptable", func() {
		suite.AssertPerformance("query 100 events", 2*time.Second, func() {
			query := models.AuditQuery{
				Limit:     100,
				SortBy:    "timestamp",
				SortOrder: "desc",
			}
			_, err := suite.adapter.Query(suite.ctx, query)
			suite.Require().NoError(err)
		})
	})
}

// TestAuditEventCleanupOperations tests cleanup and archival operations
func (suite *AuditEventBehaviorTestSuite) TestAuditEventCleanupOperations() {
	var (
		oldEventID = GenerateTestID("old-event")
		newEventID = GenerateTestID("new-event")
		cutoffTime = time.Now().Add(-1 * time.Hour)
	)

	suite.Given("old and new audit events", func() {
		// Create old event
		oldEvent := suite.CreateTestAuditEvent(oldEventID, func(e *models.AuditEvent) {
			e.Timestamp = time.Now().Add(-2 * time.Hour) // 2 hours ago
			e.ServiceName = "cleanup-test-service"
		})
		err := suite.adapter.Create(suite.ctx, oldEvent)
		suite.Require().NoError(err)

		// Create new event
		newEvent := suite.CreateTestAuditEvent(newEventID, func(e *models.AuditEvent) {
			e.Timestamp = time.Now().Add(-30 * time.Minute) // 30 minutes ago
			e.ServiceName = "cleanup-test-service"
		})
		err = suite.adapter.Create(suite.ctx, newEvent)
		suite.Require().NoError(err)
		suite.trackCreatedEvent(newEventID) // Only track new event for cleanup
	}).When("deleting events older than cutoff time", func() {
		deletedCount, err := suite.adapter.DeleteOlderThan(suite.ctx, cutoffTime)
		suite.Require().NoError(err)

		suite.Then("only old events should be deleted", func() {
			suite.Greater(deletedCount, int64(0), "Should delete at least one old event")

			// Old event should be deleted
			_, err = suite.adapter.GetByID(suite.ctx, oldEventID)
			suite.Error(err, "Old event should be deleted")

			// New event should still exist
			_, err = suite.adapter.GetByID(suite.ctx, newEventID)
			suite.NoError(err, "New event should still exist")
		})
	})
}