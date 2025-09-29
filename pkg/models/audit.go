package models

import (
	"time"
)

// AuditEvent represents a correlation event in the audit system
type AuditEvent struct {
	ID           string                 `json:"id" bson:"_id,omitempty"`
	TraceID      string                 `json:"trace_id" bson:"trace_id"`
	SpanID       string                 `json:"span_id" bson:"span_id"`
	ServiceName  string                 `json:"service_name" bson:"service_name"`
	EventType    string                 `json:"event_type" bson:"event_type"`
	Timestamp    time.Time              `json:"timestamp" bson:"timestamp"`
	Duration     time.Duration          `json:"duration" bson:"duration"`
	Status       AuditEventStatus       `json:"status" bson:"status"`
	Metadata     map[string]interface{} `json:"metadata" bson:"metadata"`
	Tags         []string               `json:"tags" bson:"tags"`
	CorrelatedTo []string               `json:"correlated_to" bson:"correlated_to"`
	CreatedAt    time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" bson:"updated_at"`
}

// AuditEventStatus represents the status of an audit event
type AuditEventStatus string

const (
	AuditEventStatusPending   AuditEventStatus = "pending"
	AuditEventStatusProcessed AuditEventStatus = "processed"
	AuditEventStatusFailed    AuditEventStatus = "failed"
	AuditEventStatusSkipped   AuditEventStatus = "skipped"
)

// ServiceRegistration represents a service in the discovery registry
type ServiceRegistration struct {
	ID          string            `json:"id" redis:"id"`
	Name        string            `json:"name" redis:"name"`
	Version     string            `json:"version" redis:"version"`
	Host        string            `json:"host" redis:"host"`
	GRPCPort    int               `json:"grpc_port" redis:"grpc_port"`
	HTTPPort    int               `json:"http_port" redis:"http_port"`
	Status      string            `json:"status" redis:"status"`
	Metadata    map[string]string `json:"metadata" redis:"metadata"`
	LastSeen    time.Time         `json:"last_seen" redis:"last_seen"`
	RegisteredAt time.Time        `json:"registered_at" redis:"registered_at"`
}

// AuditQuery represents query parameters for audit events
type AuditQuery struct {
	ServiceName *string
	EventType   *string
	Status      *AuditEventStatus
	TraceID     *string
	StartTime   *time.Time
	EndTime     *time.Time
	Tags        []string
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string // "asc" or "desc"
}

// AuditCorrelation represents a correlation between audit events
type AuditCorrelation struct {
	ID             string    `json:"id" bson:"_id,omitempty"`
	SourceEventID  string    `json:"source_event_id" bson:"source_event_id"`
	TargetEventID  string    `json:"target_event_id" bson:"target_event_id"`
	CorrelationType string   `json:"correlation_type" bson:"correlation_type"`
	Confidence     float64   `json:"confidence" bson:"confidence"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}

// ServiceMetrics represents metrics data for services
type ServiceMetrics struct {
	ServiceName    string            `redis:"service_name"`
	InstanceID     string            `redis:"instance_id"`
	Timestamp      time.Time         `redis:"timestamp"`
	RequestCount   int64             `redis:"request_count"`
	ErrorCount     int64             `redis:"error_count"`
	ResponseTime   time.Duration     `redis:"response_time"`
	CustomMetrics  map[string]float64 `redis:"custom_metrics"`
}