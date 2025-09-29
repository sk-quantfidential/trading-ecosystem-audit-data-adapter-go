# Audit Data Adapter (Go)

- **Component**: audit-data-adapter-go
- **Domain**: Event correlation, audit trails, system behavior analysis
- **Epic**: TSE-0001.10 (Audit Infrastructure), TSE-0001.12c (Audit Integration)
- **Tech Stack**: Go, PostgreSQL, Redis
- **Schema Namespace**: `audit`

## Purpose

The Audit Data Adapter provides data persistence services for the audit-correlator-go component, following Clean Architecture principles. It exposes domain-driven APIs for event ingestion, correlation analysis, and audit trail generation while abstracting database implementation details.

## Architecture Compliance

**Clean Architecture**:
- Exposes business domain concepts, not database artifacts
- Provides audit-specific APIs tailored to correlation and compliance needs
- Maintains complete separation from audit-correlator business logic
- Uses shared infrastructure with logical namespace isolation

**Domain Focus**:
- Event ingestion from telemetry, traces, and logs
- Causal chain reconstruction and correlation data
- Timeline analytics and performance metrics
- Regulatory compliance audit trail generation

## Data Requirements

### High-Volume Event Processing
- **Event Ingestion**: Real-time ingestion from all services (telemetry, traces, logs)
- **Correlation Data**: Causal chain reconstruction between events
- **Timeline Analytics**: Performance metrics and behavior analysis
- **Audit Trails**: Regulatory compliance trail generation
- **Event Archives**: Long-term storage with time-series optimization

### Storage Patterns
- **Redis**: Real-time event processing, correlation cache, active query optimization
- **PostgreSQL**: Event archives, audit trails, correlation results, analytics data

## API Design Principles

### Domain-Driven APIs
The adapter exposes audit and correlation concepts, not database implementation:

**Good Examples**:
```go
IngestEvent(event) -> EventID
CorrelateEvents(timeRange, filters) -> CorrelationChain
GenerateAuditTrail(criteria) -> AuditReport
AnalyzeTimeline(scenario) -> TimelineAnalysis
```

**Avoid Database Artifacts**:
```go
// Don't expose these
GetEventTable() -> []EventRow
UpdateCorrelationRecord(id, fields) -> bool
QueryAuditHistory(sql) -> ResultSet
```

## Technology Standards

### Database Conventions
- **PostgreSQL**: snake_case for tables, columns, functions
- **Redis**: kebab-case with `audit:` namespace prefix
- **Go**: PascalCase for public APIs, camelCase for internal functions

### Performance Requirements
- **Event Ingestion**: Handle high-volume event streams (1000+ events/second)
- **Real-time Correlation**: Sub-second correlation for active scenarios
- **Historical Queries**: Efficient time-series queries for audit trails
- **Storage Optimization**: Time-based partitioning for event archives

## Integration Points

### Serves
- **Primary**: audit-correlator-go
- **Integration**: All services emit events to this adapter

### Dependencies
- **Shared Infrastructure**: Single PostgreSQL + Redis instances
- **Protocol Buffers**: Via protobuf-schemas for event definitions
- **Service Discovery**: Via orchestrator-docker configuration

## Development Status

**Repository**: Active (Repository Created)
**Branch**: feature/TSE-0003.0-data-adapter-foundation
**Epic Progress**: TSE-0001.10 (Audit Infrastructure) - Not Started

## Next Steps

1. **Component Configuration**: Add `.claude/` configuration for audit-specific patterns
2. **Schema Design**: Design audit schema in `audit` PostgreSQL namespace
3. **API Definition**: Define event ingestion and correlation APIs
4. **Implementation**: Implement adapter with comprehensive testing
5. **Integration**: Connect with audit-correlator-go component

## Configuration Management

- **Shared Configuration**: project-plan/.claude/ for global architecture patterns
- **Component Configuration**: .claude/ directory for audit-specific settings (to be created)
- **Database Schema**: `audit` namespace with event storage optimization

---

**Epic Context**: TSE-0001 Foundation Services & Infrastructure
**Last Updated**: 2025-09-18
**Architecture**: Clean Architecture with domain-driven data persistence