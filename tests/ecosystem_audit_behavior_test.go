package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// TestEcosystemAuditBehaviorSuite tests the ecosystem-wide audit capabilities
func (suite *BehaviorTestSuite) TestEcosystemAuditBehaviorSuite() {
	suite.Run("TestEcosystemAuditTraceCorrelation", suite.TestEcosystemAuditTraceCorrelation)
	suite.Run("TestEcosystemAuditCrossServiceFlow", suite.TestEcosystemAuditCrossServiceFlow)
	suite.Run("TestEcosystemAuditEventChaining", suite.TestEcosystemAuditEventChaining)
	suite.Run("TestEcosystemAuditAggregation", suite.TestEcosystemAuditAggregation)
	suite.Run("TestEcosystemAuditMetricsCollection", suite.TestEcosystemAuditMetricsCollection)
}

// TestEcosystemAuditTraceCorrelation tests correlation of audit events across ecosystem components
func (suite *BehaviorTestSuite) TestEcosystemAuditTraceCorrelation() {
	suite.logger.Info("Testing ecosystem audit trace correlation")

	traceID := GenerateTestUUID()
	correlationID := GenerateTestID("correlation")

	suite.Given("multiple services generating correlated audit events", func() {
		// Simulate trading engine event
		tradingEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.TraceID = traceID
			e.ServiceName = "trading-engine"
			e.EventType = "order_created"
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"order_id": "ord-123",
				"symbol": "BTC/USD",
				"correlation_id": "%s",
				"flow_step": "initiation"
			}`, correlationID))
		})

		// Simulate exchange simulator event
		exchangeEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.TraceID = traceID
			e.ServiceName = "exchange-simulator"
			e.EventType = "order_received"
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"order_id": "ord-123",
				"exchange": "binance",
				"correlation_id": "%s",
				"flow_step": "processing"
			}`, correlationID))
		})

		// Simulate custodian simulator event
		custodianEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.TraceID = traceID
			e.ServiceName = "custodian-simulator"
			e.EventType = "settlement_initiated"
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"order_id": "ord-123",
				"settlement_id": "set-456",
				"correlation_id": "%s",
				"flow_step": "settlement"
			}`, correlationID))
		})

		suite.Require().NoError(suite.auditRepo.Create(context.Background(), tradingEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), exchangeEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), custodianEvent))
	}).When("querying events by trace ID", func() {
		query := &models.AuditQuery{
			TraceID: &traceID,
			Limit:   10,
		}
		events, err := suite.auditRepo.Query(context.Background(), *query)
		suite.Require().NoError(err)

		suite.correlatedEvents = events
	}).Then("all correlated events should be retrievable", func() {
		suite.Require().Len(suite.correlatedEvents, 3, "Should find all 3 correlated events")

		// Verify each service's event is present
		serviceEvents := make(map[string]*models.AuditEvent)
		for _, event := range suite.correlatedEvents {
			serviceEvents[event.ServiceName] = event
		}

		suite.Contains(serviceEvents, "trading-engine")
		suite.Contains(serviceEvents, "exchange-simulator")
		suite.Contains(serviceEvents, "custodian-simulator")

		// Verify correlation ID is preserved across all events
		for serviceName, event := range serviceEvents {
			var metadata map[string]interface{}
			err := json.Unmarshal(event.Metadata, &metadata)
			suite.Require().NoError(err, "Should unmarshal metadata for %s", serviceName)
			suite.Equal(correlationID, metadata["correlation_id"], "Correlation ID should match for %s", serviceName)
		}
	})
}

// TestEcosystemAuditCrossServiceFlow tests audit event flow across multiple ecosystem services
func (suite *BehaviorTestSuite) TestEcosystemAuditCrossServiceFlow() {
	suite.logger.Info("Testing ecosystem audit cross-service flow")

	tradeFlowID := GenerateTestID("trade-flow")

	suite.Given("a complete trading flow across services", func() {
		// Step 1: Market data service receives price update
		marketDataEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "market-data-simulator"
			e.EventType = "price_update"
			e.Tags = []string{"market-data", "price", "real-time"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"symbol": "ETH/USD",
				"price": 2450.50,
				"flow_id": "%s",
				"flow_step": 1,
				"flow_stage": "market_data"
			}`, tradeFlowID))
		})

		// Step 2: Trading engine analyzes and creates order
		tradingEngineEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "trading-engine"
			e.EventType = "order_analysis"
			e.Tags = []string{"trading", "analysis", "order"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"symbol": "ETH/USD",
				"analysis_result": "buy_signal",
				"order_type": "market",
				"flow_id": "%s",
				"flow_step": 2,
				"flow_stage": "analysis"
			}`, tradeFlowID))
		})

		// Step 3: Risk monitor validates the order
		riskMonitorEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "risk-monitor"
			e.EventType = "risk_validation"
			e.Tags = []string{"risk", "validation", "compliance"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"symbol": "ETH/USD",
				"risk_score": 0.15,
				"validation_result": "approved",
				"flow_id": "%s",
				"flow_step": 3,
				"flow_stage": "risk_validation"
			}`, tradeFlowID))
		})

		// Step 4: Exchange simulator executes the order
		exchangeEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "exchange-simulator"
			e.EventType = "order_execution"
			e.Tags = []string{"exchange", "execution", "fill"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"symbol": "ETH/USD",
				"execution_price": 2450.75,
				"quantity": 1.5,
				"flow_id": "%s",
				"flow_step": 4,
				"flow_stage": "execution"
			}`, tradeFlowID))
		})

		// Step 5: Custodian simulator handles settlement
		custodianEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "custodian-simulator"
			e.EventType = "settlement_complete"
			e.Tags = []string{"custodian", "settlement", "final"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"symbol": "ETH/USD",
				"settlement_amount": 3676.125,
				"settlement_status": "completed",
				"flow_id": "%s",
				"flow_step": 5,
				"flow_stage": "settlement"
			}`, tradeFlowID))
		})

		// Create all events
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), marketDataEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), tradingEngineEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), riskMonitorEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), exchangeEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), custodianEvent))
	}).When("reconstructing the complete trading flow", func() {
		// Query all events for this flow
		query := &models.AuditQuery{
			Limit:     10,
			SortBy:    "timestamp",
			SortOrder: "asc",
		}
		allEvents, err := suite.auditRepo.Query(context.Background(), *query)
		suite.Require().NoError(err)

		// Filter events belonging to this flow
		var flowEvents []*models.AuditEvent
		for _, event := range allEvents {
			var metadata map[string]interface{}
			if err := json.Unmarshal(event.Metadata, &metadata); err == nil {
				if flowID, ok := metadata["flow_id"].(string); ok && flowID == tradeFlowID {
					flowEvents = append(flowEvents, event)
				}
			}
		}

		suite.flowEvents = flowEvents
	}).Then("the complete flow should be auditable and traceable", func() {
		suite.Require().Len(suite.flowEvents, 5, "Should have all 5 flow events")

		// Verify each stage is present
		stageMap := make(map[string]*models.AuditEvent)
		for _, event := range suite.flowEvents {
			var metadata map[string]interface{}
			err := json.Unmarshal(event.Metadata, &metadata)
			suite.Require().NoError(err)

			if stage, ok := metadata["flow_stage"].(string); ok {
				stageMap[stage] = event
			}
		}

		expectedStages := []string{"market_data", "analysis", "risk_validation", "execution", "settlement"}
		for _, stage := range expectedStages {
			suite.Contains(stageMap, stage, "Flow should include %s stage", stage)
		}

		// Verify flow continuity - each step should follow the previous
		stepMap := make(map[float64]*models.AuditEvent)
		for _, event := range suite.flowEvents {
			var metadata map[string]interface{}
			err := json.Unmarshal(event.Metadata, &metadata)
			suite.Require().NoError(err)

			if step, ok := metadata["flow_step"].(float64); ok {
				stepMap[step] = event
			}
		}

		for i := 1; i <= 5; i++ {
			suite.Contains(stepMap, float64(i), "Flow should include step %d", i)
		}
	})
}

// TestEcosystemAuditEventChaining tests chaining of related audit events
func (suite *BehaviorTestSuite) TestEcosystemAuditEventChaining() {
	suite.logger.Info("Testing ecosystem audit event chaining")

	parentEventID := GenerateTestUUID()
	chainID := GenerateTestID("chain")

	suite.Given("a chain of related audit events", func() {
		// Parent event
		parentEvent := suite.CreateTestAuditEvent(parentEventID, func(e *models.AuditEvent) {
			e.ServiceName = "orchestrator"
			e.EventType = "workflow_started"
			e.Tags = []string{"workflow", "parent", "orchestration"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"workflow_id": "%s",
				"workflow_type": "order_processing",
				"chain_position": "root"
			}`, chainID))
		})

		// Child events that reference the parent
		childEvent1 := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "trading-engine"
			e.EventType = "order_validation"
			e.CorrelatedTo = []string{parentEventID}
			e.Tags = []string{"workflow", "child", "validation"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"workflow_id": "%s",
				"parent_event_id": "%s",
				"chain_position": "child",
				"validation_stage": "preliminary"
			}`, chainID, parentEventID))
		})

		childEvent2 := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "risk-monitor"
			e.EventType = "risk_assessment"
			e.CorrelatedTo = []string{parentEventID}
			e.Tags = []string{"workflow", "child", "risk"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"workflow_id": "%s",
				"parent_event_id": "%s",
				"chain_position": "child",
				"assessment_type": "comprehensive"
			}`, chainID, parentEventID))
		})

		// Grandchild event
		grandChildEvent := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
			e.ServiceName = "exchange-simulator"
			e.EventType = "order_executed"
			e.CorrelatedTo = []string{childEvent1.ID, childEvent2.ID}
			e.Tags = []string{"workflow", "grandchild", "execution"}
			e.Metadata = json.RawMessage(fmt.Sprintf(`{
				"workflow_id": "%s",
				"parent_events": ["%s", "%s"],
				"chain_position": "grandchild",
				"execution_result": "successful"
			}`, chainID, childEvent1.ID, childEvent2.ID))
		})

		suite.Require().NoError(suite.auditRepo.Create(context.Background(), parentEvent))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), childEvent1))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), childEvent2))
		suite.Require().NoError(suite.auditRepo.Create(context.Background(), grandChildEvent))
	}).When("querying the event chain", func() {
		// Query events related to the parent
		relatedEvents, err := suite.auditRepo.GetCorrelatedEvents(context.Background(), parentEventID)
		suite.Require().NoError(err)

		suite.chainedEvents = relatedEvents
	}).Then("the complete event chain should be discoverable", func() {
		suite.Require().GreaterOrEqual(len(suite.chainedEvents), 2, "Should find related events")

		// Verify that child events reference the parent
		var foundChildEvents int
		for _, event := range suite.chainedEvents {
			for _, correlatedID := range event.CorrelatedTo {
				if correlatedID == parentEventID {
					foundChildEvents++
					break
				}
			}
		}

		suite.GreaterOrEqual(foundChildEvents, 2, "Should find at least 2 child events referencing parent")
	})
}

// TestEcosystemAuditAggregation tests aggregation capabilities across ecosystem
func (suite *BehaviorTestSuite) TestEcosystemAuditAggregation() {
	suite.logger.Info("Testing ecosystem audit aggregation capabilities")

	sessionID := GenerateTestID("session")

	suite.Given("multiple audit events from different services", func() {
		services := []string{"trading-engine", "exchange-simulator", "custodian-simulator", "risk-monitor", "market-data-simulator"}
		eventTypes := []string{"order_created", "price_update", "risk_check", "settlement", "execution"}

		for i, service := range services {
			event := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
				e.ServiceName = service
				e.EventType = eventTypes[i]
				e.Tags = []string{"aggregation", "session", "ecosystem"}
				e.Metadata = json.RawMessage(fmt.Sprintf(`{
					"session_id": "%s",
					"service_instance": "%s-001",
					"aggregation_category": "trading_session",
					"sequence": %d
				}`, sessionID, service, i+1))
			})
			suite.Require().NoError(suite.auditRepo.Create(context.Background(), event))
		}
	}).When("performing ecosystem-wide aggregation queries", func() {
		// Query all events for the session
		query := &models.AuditQuery{
			Limit:     20,
			SortBy:    "timestamp",
			SortOrder: "asc",
		}
		allEvents, err := suite.auditRepo.Query(context.Background(), *query)
		suite.Require().NoError(err)

		// Filter events for this session
		var sessionEvents []*models.AuditEvent
		for _, event := range allEvents {
			var metadata map[string]interface{}
			if err := json.Unmarshal(event.Metadata, &metadata); err == nil {
				if sesID, ok := metadata["session_id"].(string); ok && sesID == sessionID {
					sessionEvents = append(sessionEvents, event)
				}
			}
		}

		suite.aggregatedEvents = sessionEvents
	}).Then("aggregated data should provide ecosystem insights", func() {
		suite.Require().Len(suite.aggregatedEvents, 5, "Should aggregate all session events")

		// Verify service distribution
		serviceCount := make(map[string]int)
		eventTypeCount := make(map[string]int)

		for _, event := range suite.aggregatedEvents {
			serviceCount[event.ServiceName]++
			eventTypeCount[event.EventType]++
		}

		suite.Equal(5, len(serviceCount), "Should have events from 5 different services")
		suite.Equal(5, len(eventTypeCount), "Should have 5 different event types")

		// Each service should contribute exactly one event
		for service, count := range serviceCount {
			suite.Equal(1, count, "Service %s should have exactly 1 event", service)
		}
	})
}

// TestEcosystemAuditMetricsCollection tests metrics collection across the ecosystem
func (suite *BehaviorTestSuite) TestEcosystemAuditMetricsCollection() {
	suite.logger.Info("Testing ecosystem audit metrics collection")

	suite.Given("audit events with performance metrics", func() {
		// Create events with different performance characteristics
		performanceEvents := []struct {
			service  string
			duration time.Duration
			status   models.AuditEventStatus
		}{
			{"trading-engine", 50 * time.Millisecond, models.AuditEventStatusProcessed},
			{"exchange-simulator", 200 * time.Millisecond, models.AuditEventStatusProcessed},
			{"custodian-simulator", 100 * time.Millisecond, models.AuditEventStatusProcessed},
			{"risk-monitor", 25 * time.Millisecond, models.AuditEventStatusProcessed},
			{"market-data-simulator", 10 * time.Millisecond, models.AuditEventStatusFailed},
		}

		for i, perf := range performanceEvents {
			event := suite.CreateTestAuditEvent("", func(e *models.AuditEvent) {
				e.ServiceName = perf.service
				e.EventType = "performance_test"
				e.Duration = perf.duration
				e.Status = perf.status
				e.Tags = []string{"performance", "metrics", "monitoring"}
				e.Metadata = json.RawMessage(fmt.Sprintf(`{
					"performance_test_id": "perf-test-ecosystem",
					"test_sequence": %d,
					"expected_duration_ms": %d,
					"actual_duration_ms": %d,
					"status": "%s"
				}`, i+1, int(perf.duration.Milliseconds()), int(perf.duration.Milliseconds()), perf.status))
			})
			suite.Require().NoError(suite.auditRepo.Create(context.Background(), event))
		}

		// Allow time for events to be processed
		time.Sleep(100 * time.Millisecond)
	}).When("collecting ecosystem performance metrics", func() {
		// Query performance test events
		query := &models.AuditQuery{
			EventType: stringPtr("performance_test"),
			Limit:     10,
		}
		events, err := suite.auditRepo.Query(context.Background(), *query)
		suite.Require().NoError(err)

		suite.metricsEvents = events
	}).Then("metrics should provide ecosystem health insights", func() {
		suite.Require().Len(suite.metricsEvents, 5, "Should collect metrics from all services")

		// Calculate aggregate metrics
		var totalDuration time.Duration
		var processedEvents, failedEvents int
		serviceMetrics := make(map[string]time.Duration)

		for _, event := range suite.metricsEvents {
			totalDuration += event.Duration
			serviceMetrics[event.ServiceName] = event.Duration

			switch event.Status {
			case models.AuditEventStatusProcessed:
				processedEvents++
			case models.AuditEventStatusFailed:
				failedEvents++
			}
		}

		// Verify metrics collection
		avgDuration := totalDuration / time.Duration(len(suite.metricsEvents))
		successRate := float64(processedEvents) / float64(len(suite.metricsEvents)) * 100

		suite.logger.WithFields(map[string]interface{}{
			"total_events":    len(suite.metricsEvents),
			"avg_duration_ms": avgDuration.Milliseconds(),
			"success_rate":    successRate,
			"failed_events":   failedEvents,
		}).Info("Ecosystem performance metrics collected")

		suite.Equal(4, processedEvents, "Should have 4 successful events")
		suite.Equal(1, failedEvents, "Should have 1 failed event")
		suite.Equal(5, len(serviceMetrics), "Should have metrics from all 5 services")

		// Verify fastest and slowest services
		fastestDuration := time.Hour
		slowestDuration := time.Duration(0)
		for _, duration := range serviceMetrics {
			if duration < fastestDuration {
				fastestDuration = duration
			}
			if duration > slowestDuration {
				slowestDuration = duration
			}
		}

		suite.Equal(10*time.Millisecond, fastestDuration, "Fastest service should be market-data-simulator")
		suite.Equal(200*time.Millisecond, slowestDuration, "Slowest service should be exchange-simulator")
	})
}

