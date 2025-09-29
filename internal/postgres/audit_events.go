package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/internal/config"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/interfaces"
	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/models"
)

// DBExecutor interface allows using either *sql.DB or *sql.Tx
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// AuditEventPostgresRepository implements AuditEventRepository using PostgreSQL
type AuditEventPostgresRepository struct {
	db     DBExecutor
	logger *logrus.Logger
	config *config.RepositoryConfig
}

// NewAuditEventRepository creates a new PostgreSQL-based audit event repository
func NewAuditEventRepository(db DBExecutor, logger *logrus.Logger, cfg *config.RepositoryConfig) interfaces.AuditEventRepository {
	return &AuditEventPostgresRepository{
		db:     db,
		logger: logger,
		config: cfg,
	}
}

// Create inserts a new audit event
func (r *AuditEventPostgresRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	query := `
		INSERT INTO audit_correlator.audit_events (
			id, trace_id, span_id, service_name, event_type, timestamp, duration,
			status, metadata, tags, correlated_to, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.TraceID, event.SpanID, event.ServiceName, event.EventType,
		event.Timestamp, event.Duration, event.Status, event.Metadata,
		pq.Array(event.Tags), pq.Array(event.CorrelatedTo),
		event.CreatedAt, event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"event_id":     event.ID,
		"trace_id":     event.TraceID,
		"service_name": event.ServiceName,
		"event_type":   event.EventType,
	}).Debug("Audit event created")

	return nil
}

// GetByID retrieves an audit event by ID
func (r *AuditEventPostgresRepository) GetByID(ctx context.Context, id string) (*models.AuditEvent, error) {
	query := `
		SELECT id, trace_id, span_id, service_name, event_type, timestamp, duration,
		       status, metadata, tags, correlated_to, created_at, updated_at
		FROM audit_correlator.audit_events
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	event := &models.AuditEvent{}
	var tags, correlatedTo pq.StringArray

	err := row.Scan(
		&event.ID, &event.TraceID, &event.SpanID, &event.ServiceName, &event.EventType,
		&event.Timestamp, &event.Duration, &event.Status, &event.Metadata,
		&tags, &correlatedTo, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit event not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get audit event: %w", err)
	}

	event.Tags = []string(tags)
	event.CorrelatedTo = []string(correlatedTo)

	return event, nil
}

// Update modifies an existing audit event
func (r *AuditEventPostgresRepository) Update(ctx context.Context, event *models.AuditEvent) error {
	query := `
		UPDATE audit_correlator.audit_events
		SET trace_id = $2, span_id = $3, service_name = $4, event_type = $5,
		    timestamp = $6, duration = $7, status = $8, metadata = $9,
		    tags = $10, correlated_to = $11, updated_at = $12
		WHERE id = $1`

	event.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		event.ID, event.TraceID, event.SpanID, event.ServiceName, event.EventType,
		event.Timestamp, event.Duration, event.Status, event.Metadata,
		pq.Array(event.Tags), pq.Array(event.CorrelatedTo), event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update audit event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("audit event not found: %s", event.ID)
	}

	return nil
}

// Delete removes an audit event
func (r *AuditEventPostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM audit_correlator.audit_events WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete audit event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("audit event not found: %s", id)
	}

	return nil
}

// Query retrieves audit events based on query parameters
func (r *AuditEventPostgresRepository) Query(ctx context.Context, query models.AuditQuery) ([]*models.AuditEvent, error) {
	sqlQuery, args := r.buildQuery(query)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []*models.AuditEvent

	for rows.Next() {
		event := &models.AuditEvent{}
		var tags, correlatedTo pq.StringArray

		err := rows.Scan(
			&event.ID, &event.TraceID, &event.SpanID, &event.ServiceName, &event.EventType,
			&event.Timestamp, &event.Duration, &event.Status, &event.Metadata,
			&tags, &correlatedTo, &event.CreatedAt, &event.UpdatedAt)

		if err != nil {
			r.logger.WithError(err).Warn("Failed to scan audit event row")
			continue
		}

		event.Tags = []string(tags)
		event.CorrelatedTo = []string(correlatedTo)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit event rows: %w", err)
	}

	return events, nil
}

// Count returns the count of audit events matching the query
func (r *AuditEventPostgresRepository) Count(ctx context.Context, query models.AuditQuery) (int64, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM audit_correlator.audit_events
		WHERE 1=1`

	var args []interface{}
	var conditions []string
	argIndex := 1

	// Build WHERE conditions
	if query.ServiceName != nil {
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", argIndex))
		args = append(args, *query.ServiceName)
		argIndex++
	}

	if query.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, *query.EventType)
		argIndex++
	}

	if query.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *query.Status)
		argIndex++
	}

	if query.TraceID != nil {
		conditions = append(conditions, fmt.Sprintf("trace_id = $%d", argIndex))
		args = append(args, *query.TraceID)
		argIndex++
	}

	if query.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *query.StartTime)
		argIndex++
	}

	if query.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *query.EndTime)
		argIndex++
	}

	if len(query.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, pq.Array(query.Tags))
		argIndex++
	}

	if len(conditions) > 0 {
		countQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	return count, nil
}

// GetByTraceID retrieves all audit events for a specific trace
func (r *AuditEventPostgresRepository) GetByTraceID(ctx context.Context, traceID string) ([]*models.AuditEvent, error) {
	query := models.AuditQuery{
		TraceID: &traceID,
		SortBy:  "timestamp",
		SortOrder: "asc",
	}

	return r.Query(ctx, query)
}

// GetCorrelatedEvents retrieves events correlated to a specific event
func (r *AuditEventPostgresRepository) GetCorrelatedEvents(ctx context.Context, eventID string) ([]*models.AuditEvent, error) {
	query := `
		SELECT DISTINCT ae.id, ae.trace_id, ae.span_id, ae.service_name, ae.event_type,
		       ae.timestamp, ae.duration, ae.status, ae.metadata, ae.tags,
		       ae.correlated_to, ae.created_at, ae.updated_at
		FROM audit_correlator.audit_events ae
		WHERE ae.id = ANY(
			SELECT unnest(correlated_to)
			FROM audit_correlator.audit_events
			WHERE id = $1
		)
		OR $1 = ANY(ae.correlated_to)
		ORDER BY ae.timestamp ASC`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get correlated events: %w", err)
	}
	defer rows.Close()

	var events []*models.AuditEvent

	for rows.Next() {
		event := &models.AuditEvent{}
		var tags, correlatedTo pq.StringArray

		err := rows.Scan(
			&event.ID, &event.TraceID, &event.SpanID, &event.ServiceName, &event.EventType,
			&event.Timestamp, &event.Duration, &event.Status, &event.Metadata,
			&tags, &correlatedTo, &event.CreatedAt, &event.UpdatedAt)

		if err != nil {
			r.logger.WithError(err).Warn("Failed to scan correlated event row")
			continue
		}

		event.Tags = []string(tags)
		event.CorrelatedTo = []string(correlatedTo)
		events = append(events, event)
	}

	return events, nil
}

// CreateBatch inserts multiple audit events efficiently
func (r *AuditEventPostgresRepository) CreateBatch(ctx context.Context, events []*models.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO audit_correlator.audit_events (
			id, trace_id, span_id, service_name, event_type, timestamp, duration,
			status, metadata, tags, correlated_to, created_at, updated_at
		) VALUES `

	values := make([]string, len(events))
	args := make([]interface{}, 0, len(events)*13)
	now := time.Now()

	for i, event := range events {
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}
		event.UpdatedAt = now

		startIdx := i * 13
		values[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			startIdx+1, startIdx+2, startIdx+3, startIdx+4, startIdx+5, startIdx+6, startIdx+7,
			startIdx+8, startIdx+9, startIdx+10, startIdx+11, startIdx+12, startIdx+13)

		args = append(args, event.ID, event.TraceID, event.SpanID, event.ServiceName, event.EventType,
			event.Timestamp, event.Duration, event.Status, event.Metadata,
			pq.Array(event.Tags), pq.Array(event.CorrelatedTo), event.CreatedAt, event.UpdatedAt)
	}

	query += strings.Join(values, ", ")

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to create audit events batch: %w", err)
	}

	r.logger.WithField("count", len(events)).Info("Created audit events batch")

	return nil
}

// UpdateBatch updates multiple audit events efficiently
func (r *AuditEventPostgresRepository) UpdateBatch(ctx context.Context, events []*models.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Check if we already have a transaction
	var executor DBExecutor
	var tx *sql.Tx
	var err error

	if db, ok := r.db.(*sql.DB); ok {
		// We have a DB, need to start a transaction
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		executor = tx
	} else {
		// We already have a transaction
		executor = r.db
	}

	query := `
		UPDATE audit_correlator.audit_events
		SET trace_id = $2, span_id = $3, service_name = $4, event_type = $5,
		    timestamp = $6, duration = $7, status = $8, metadata = $9,
		    tags = $10, correlated_to = $11, updated_at = $12
		WHERE id = $1`

	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()

	for _, event := range events {
		event.UpdatedAt = now

		_, err := stmt.ExecContext(ctx,
			event.ID, event.TraceID, event.SpanID, event.ServiceName, event.EventType,
			event.Timestamp, event.Duration, event.Status, event.Metadata,
			pq.Array(event.Tags), pq.Array(event.CorrelatedTo), event.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to update audit event %s: %w", event.ID, err)
		}
	}

	// Only commit if we created the transaction
	if tx != nil {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit batch update: %w", err)
		}
	}

	r.logger.WithField("count", len(events)).Info("Updated audit events batch")

	return nil
}

// DeleteOlderThan removes audit events older than the specified timestamp
func (r *AuditEventPostgresRepository) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM audit_correlator.audit_events WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit events: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	if deletedCount > 0 {
		r.logger.WithFields(logrus.Fields{
			"deleted_count": deletedCount,
			"before":        timestamp,
		}).Info("Deleted old audit events")
	}

	return deletedCount, nil
}

// ArchiveOlderThan moves old audit events to archive table
func (r *AuditEventPostgresRepository) ArchiveOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	// First, copy to archive table
	archiveQuery := `
		INSERT INTO audit_correlator.audit_events_archive
		SELECT * FROM audit_correlator.audit_events
		WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, archiveQuery, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to archive audit events: %w", err)
	}

	archivedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get archived count: %w", err)
	}

	if archivedCount > 0 {
		// Delete from main table
		deleteQuery := `DELETE FROM audit_correlator.audit_events WHERE created_at < $1`
		_, err = r.db.ExecContext(ctx, deleteQuery, timestamp)
		if err != nil {
			return 0, fmt.Errorf("failed to delete archived events: %w", err)
		}

		r.logger.WithFields(logrus.Fields{
			"archived_count": archivedCount,
			"before":         timestamp,
		}).Info("Archived old audit events")
	}

	return archivedCount, nil
}

// buildQuery constructs SQL query and arguments from AuditQuery
func (r *AuditEventPostgresRepository) buildQuery(query models.AuditQuery) (string, []interface{}) {
	sqlQuery := `
		SELECT id, trace_id, span_id, service_name, event_type, timestamp, duration,
		       status, metadata, tags, correlated_to, created_at, updated_at
		FROM audit_correlator.audit_events
		WHERE 1=1`

	var args []interface{}
	var conditions []string
	argIndex := 1

	// Build WHERE conditions
	if query.ServiceName != nil {
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", argIndex))
		args = append(args, *query.ServiceName)
		argIndex++
	}

	if query.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, *query.EventType)
		argIndex++
	}

	if query.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *query.Status)
		argIndex++
	}

	if query.TraceID != nil {
		conditions = append(conditions, fmt.Sprintf("trace_id = $%d", argIndex))
		args = append(args, *query.TraceID)
		argIndex++
	}

	if query.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *query.StartTime)
		argIndex++
	}

	if query.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *query.EndTime)
		argIndex++
	}

	if len(query.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, pq.Array(query.Tags))
		argIndex++
	}

	if len(conditions) > 0 {
		sqlQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	if query.SortBy != "" {
		sortOrder := "ASC"
		if query.SortOrder == "desc" {
			sortOrder = "DESC"
		}
		sqlQuery += fmt.Sprintf(" ORDER BY %s %s", query.SortBy, sortOrder)
	} else {
		sqlQuery += " ORDER BY timestamp DESC"
	}

	// Add pagination
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
		argIndex++
	}

	return sqlQuery, args
}