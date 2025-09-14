# Data Model: Multi-Tenant Agentic Platform

**Feature**: Multi-Tenant Agentic Platform with External Tool Integration
**Phase**: 1 - Design & Contracts
**Date**: 2025-09-14

## Core Entities

### 1. Session CRD (Primary Entity)
**Purpose**: Represents a durable agentic execution unit with append-only history

**Spec Fields**:
```yaml
apiVersion: ambient.ai/v1alpha1
kind: Session
metadata:
  name: session-uuid
  namespace: team-alpha
spec:
  trigger:
    source: "github"                    # webhook source
    event: "pull_request"               # event type
    payload: {}                         # original webhook payload
  framework:
    type: "claude-code"                 # runner framework type
    version: "1.0"                      # framework version
    config: {}                          # framework-specific config
  policy:
    modelConstraints:
      allowed: ["claude-3-sonnet"]
      budget: "100.00"
    toolConstraints:
      allowed: ["bash", "edit", "read"]
    approvalRequired: false
  artifacts:
    references: []                      # framework-agnostic artifact refs
    storage:
      type: "s3"                        # s3, pvc, external
      location: "bucket/path/"
```

**Status Fields**:
```yaml
status:
  phase: "Running"                      # Pending, Running, Completed, Failed
  conditions:
    - type: "PolicyValidated"
      status: "True"
      lastTransitionTime: "2025-09-14T10:00:00Z"
    - type: "WorkloadCreated"
      status: "True"
      lastTransitionTime: "2025-09-14T10:01:00Z"
  history:
    - timestamp: "2025-09-14T10:00:00Z"
      event: "SessionCreated"
      data: {}
    - timestamp: "2025-09-14T10:01:00Z"
      event: "WorkloadStarted"
      data: {"podName": "session-uuid-runner"}
  artifacts:
    generated:
      - type: "log"
        location: "s3://bucket/sessions/uuid/logs/session.log"
        createdAt: "2025-09-14T10:01:00Z"
      - type: "output"
        location: "s3://bucket/sessions/uuid/outputs/result.json"
        createdAt: "2025-09-14T10:05:00Z"
  workload:
    podName: "session-uuid-runner"
    jobName: "session-uuid-job"
```

**Validation Rules**:
- `spec.framework.type` must match available runner types
- `spec.policy` must comply with namespace policy constraints
- `metadata.namespace` must be resolved server-side from webhook authentication
- `status.history` is append-only, never modified or deleted

**State Transitions**:
1. `Pending` → `Running`: Policy validated, workload created
2. `Running` → `Completed`: Workload finished successfully
3. `Running` → `Failed`: Workload failed or exceeded constraints
4. Any state → `Failed`: Policy violation detected

### 2. NamespacePolicy CRD
**Purpose**: Defines constraints and access controls for a namespace

```yaml
apiVersion: ambient.ai/v1alpha1
kind: NamespacePolicy
metadata:
  name: policy
  namespace: team-alpha
spec:
  models:
    allowed:
      - "claude-3-sonnet"
      - "claude-3-haiku"
    budget:
      monthly: "1000.00"
      currency: "USD"
  tools:
    allowed:
      - "bash"
      - "edit"
      - "read"
      - "write"
    blocked:
      - "exec"
  approval:
    required: false
    approvers:
      - "team-alpha-leads"
  retention:
    sessions: "90d"
    artifacts: "30d"
    auditLogs: "7y"
  notifications:
    webhooks:
      - url: "https://hooks.slack.com/services/..."
        events: ["session.failed", "session.completed"]
      - url: "https://api.jira.com/webhooks/..."
        events: ["session.failed"]
  webhookAuth:
    apiKeys:
      github: "api-key-hash-123"
      jira: "api-key-hash-456"
```

### 3. WebhookEvent (Runtime Entity)
**Purpose**: Represents incoming webhook requests for processing

**Fields**:
- `source`: External tool identifier (github, jira, slack)
- `headers`: HTTP headers from webhook request
- `payload`: Raw webhook payload
- `authentication`: API key or token used
- `timestamp`: When webhook was received
- `processed`: Whether event was successfully processed
- `targetNamespace`: Resolved namespace (server-side)
- `errors`: Any processing errors

### 4. AuditLog (Storage Entity)
**Purpose**: Immutable log entries for compliance and debugging

**Fields**:
- `timestamp`: ISO 8601 timestamp
- `namespace`: Target namespace
- `sessionId`: Related session UUID
- `actor`: User or service account
- `action`: Action performed (create, update, delete, view)
- `resource`: Resource affected (session, policy, etc.)
- `outcome`: Success or failure
- `details`: Additional context (JSON)
- `traceId`: Request tracing identifier

## Relationships

### Session → NamespacePolicy
- **Type**: Many-to-One
- **Constraint**: Session must comply with policy in its namespace
- **Enforcement**: Admission webhook validates against policy

### Session → Artifacts
- **Type**: One-to-Many
- **Constraint**: Artifacts scoped to session's namespace
- **Implementation**: Framework-agnostic references in session status

### WebhookEvent → Session
- **Type**: One-to-One (typically)
- **Flow**: WebhookEvent processed → Session created
- **Constraint**: Namespace resolved from webhook authentication

### AuditLog → Session
- **Type**: Many-to-One
- **Purpose**: Complete audit trail for each session
- **Retention**: Longer retention than session data

## Validation Rules

### Cross-Entity Validation
1. **Session creation**: Must pass NamespacePolicy validation
2. **Artifact generation**: Must be within namespace storage limits
3. **Webhook processing**: Must resolve to valid namespace via API key
4. **Audit logging**: Must capture all state transitions

### Business Rules
1. **Append-only history**: Session history and audit logs never modified
2. **Namespace isolation**: Sessions cannot access other namespaces
3. **Policy enforcement**: All actions validated against current policy
4. **Retention compliance**: Automated cleanup based on policy settings

## Storage Patterns

### Kubernetes Resources
- **Sessions**: Stored as CRDs in etcd
- **Policies**: Stored as CRDs in etcd
- **RBAC**: Native Kubernetes RoleBindings

### External Storage
- **Artifacts**: S3/PVC with namespace prefixing
- **Audit Logs**: Structured logging to external system
- **Webhook Payloads**: Temporary storage, cleaned up after processing

### Framework Integration
- **Generic References**: Session status contains artifact pointers
- **Framework Storage**: Each runner manages its own artifact format
- **Unified Discovery**: Session provides single point of artifact access

## Performance Considerations

### Indexing Strategy
- **Namespace**: Primary index for RBAC filtering
- **Timestamp**: Secondary index for time-based queries
- **Status**: Index for operational dashboards

### Caching Strategy
- **RBAC decisions**: Cache user namespace permissions
- **Policy validation**: Cache policy rules per namespace
- **Artifact metadata**: Cache frequently accessed artifact info

### Scaling Limits
- **Sessions per namespace**: 50 concurrent (configurable)
- **History entries**: 1000 per session (with archival)
- **Artifacts per session**: 100 (with cleanup policies)

## Next Phase Requirements

Data model supports all functional requirements:
- ✅ Generic Session CRD for multiple frameworks
- ✅ Namespace-based isolation and RBAC
- ✅ Append-only audit trail
- ✅ Policy enforcement and constraints
- ✅ Framework-agnostic artifact storage
- ✅ Webhook authentication and processing

**Ready for Contract Generation**: API endpoints and schemas