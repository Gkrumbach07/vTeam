# Research Findings: Multi-Tenant Agentic Platform

**Feature**: Multi-Tenant Agentic Platform with External Tool Integration
**Phase**: 0 - Research & Resolution of Unknowns
**Date**: 2025-09-14

## Research Scope
This research resolves the NEEDS CLARIFICATION items identified in the feature specification:
- FR-013: OAuth provider selection
- FR-014: Session/artifact retention policies
- FR-015: Error notification mechanisms
- FR-016: Scalability targets
- FR-017: Policy configuration approach

## Decision 1: OAuth Provider Integration

**Decision**: Kubernetes OIDC integration with pluggable provider support

**Rationale**:
- Kubernetes has native OIDC support that integrates directly with RBAC
- Allows flexibility for different enterprise identity providers
- Supports both user authentication (web UI) and service account authentication (webhooks)
- Maintains compatibility with existing Kubernetes security model

**Alternatives considered**:
- Custom JWT implementation: Rejected due to complexity and security risks
- GitHub OAuth only: Too limiting for enterprise deployments
- Service mesh auth (Istio): Adds unnecessary infrastructure complexity

**Implementation approach**:
- OIDC configuration in Kubernetes API server
- Service accounts for webhook authentication with bound tokens
- Web UI uses OIDC flow with namespace-scoped token exchange

## Decision 2: Session and Artifact Retention Policies

**Decision**: Configurable retention with 90-day default, compliance-aware deletion

**Rationale**:
- 90 days balances operational needs with storage costs
- Configurable per-namespace to support different compliance requirements
- Immutable audit logs with separate retention (7 years for compliance)
- Artifact cleanup separate from session history retention

**Alternatives considered**:
- Fixed 30-day retention: Too short for debugging and compliance
- Indefinite retention: Storage costs and GDPR concerns
- User-controlled deletion: Conflicts with audit requirements

**Implementation approach**:
- Namespace policy specifies retention periods
- Background controller for cleanup with safety checks
- Audit trail retention separate from operational data
- S3 lifecycle policies for artifact archival

## Decision 3: Error Notification Mechanisms

**Decision**: Pluggable notification system with webhook callbacks

**Rationale**:
- Webhook callbacks allow integration with existing alerting systems
- Supports multiple notification channels without platform lock-in
- Can route back to originating external tool (Jira comment, Slack thread)
- Maintains audit trail of notifications sent

**Alternatives considered**:
- Email notifications: Requires SMTP configuration, hard to manage
- Built-in Slack integration: Too opinionated, doesn't scale to other tools
- Kubernetes events only: Not visible outside cluster

**Implementation approach**:
- Namespace policy configures notification webhook URLs
- Structured JSON payloads with session context
- Retry logic with exponential backoff
- Notification delivery status in session audit trail

## Decision 4: Scalability Targets

**Decision**:
- 200 concurrent sessions per cluster
- 100 namespaces per cluster
- 2000 users across all namespaces
- <2s webhook response time
- 99.5% availability for session creation

**Rationale**:
- Conservative targets based on Kubernetes etcd limits
- Horizontal scaling via multiple clusters if needed
- Allows for growth while maintaining performance
- Matches enterprise team structures (20 users average per namespace)

**Alternatives considered**:
- Higher targets: Would require complex sharding and caching
- Lower targets: Insufficient for medium enterprise deployments
- Per-node limits: Too dependent on cluster configuration

**Implementation approach**:
- Resource quotas per namespace
- Connection pooling for external services
- Caching for RBAC decisions
- Metrics and alerting for capacity planning

## Decision 5: Policy Configuration Approach

**Decision**: GitOps-based YAML configuration with validation webhooks

**Rationale**:
- Infrastructure-as-code approach familiar to Kubernetes operators
- Version control and approval workflows for policy changes
- Validation webhooks prevent invalid configurations
- Can be integrated with existing Kubernetes deployment pipelines

**Alternatives considered**:
- Admin web UI: Requires additional auth, harder to audit changes
- Direct API configuration: No version control or approval workflow
- ConfigMaps only: No validation, easy to misconfigure

**Implementation approach**:
- Custom resource definitions for namespace policies
- Admission webhooks for policy validation
- Operator watches for policy changes and enforces constraints
- Policy templates for common patterns

## Additional Technical Decisions

### Webhook Authentication Strategy
**Decision**: API key authentication with namespace-scoped tokens

**Rationale**:
- Simpler than cryptographic signatures for MVP
- API keys can be rotated independently per integration
- Namespace resolution based on API key prevents privilege escalation
- Compatible with most external tool webhook systems

### Session CRD Design
**Decision**: Generic status/spec pattern with pluggable artifact references

**Rationale**:
- Follows Kubernetes controller patterns
- Framework-agnostic artifact storage (S3, PVC, external URLs)
- Status subresource allows optimistic concurrency
- Can accommodate different agentic framework requirements

### Multi-tenancy Implementation
**Decision**: Kubernetes namespace-native isolation with RBAC

**Rationale**:
- Leverages existing Kubernetes security boundaries
- No custom authorization layer needed
- Standard kubectl/RBAC tooling works out of box
- Clear audit trail via Kubernetes API server logs

## Risk Assessment

**High Impact Risks**:
1. **RBAC complexity**: Mitigation via policy templates and validation
2. **Webhook rate limits**: Mitigation via async processing and queuing
3. **Artifact storage scaling**: Mitigation via S3 tiering and cleanup

**Medium Impact Risks**:
1. **Session CRD evolution**: Mitigation via Kubernetes CRD versioning
2. **External tool changes**: Mitigation via pluggable webhook adapters

## Next Phase Requirements

All NEEDS CLARIFICATION items are now resolved:
- ✅ OAuth provider: Kubernetes OIDC
- ✅ Retention policies: 90-day configurable
- ✅ Error notifications: Webhook callbacks
- ✅ Scalability: 200 sessions, 100 namespaces, 2000 users
- ✅ Policy configuration: GitOps YAML with validation

**Ready for Phase 1**: Design & Contracts