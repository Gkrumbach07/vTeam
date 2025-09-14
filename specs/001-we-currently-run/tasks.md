# Tasks: Multi-Tenant Agentic Platform Migration

**Input**: Design documents from `/specs/001-we-currently-run/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure
2. Load optional design documents:
   → data-model.md: Extract entities → model tasks
   → contracts/: Each file → contract test task
   → research.md: Extract decisions → setup tasks
3. Generate tasks by category:
   → Setup: project init, dependencies, linting
   → Tests: contract tests, integration tests
   → Core: models, services, CLI commands
   → Integration: DB, middleware, logging
   → Polish: unit tests, performance, docs
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests?
   → All entities have models?
   → All endpoints implemented?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Multi-service architecture**: `components/backend/`, `components/operator/`, `components/frontend/`
- **Migration project**: Extend existing files while maintaining backward compatibility
- All paths relative to repository root

## Phase 3.1: Setup & Migration Planning

- [x] T001 Analyze current AgenticSession CRD and create migration strategy document
- [x] T002 [P] Set up Go testing framework in components/backend/ with testify
- [x] T003 [P] Set up Jest/Cypress testing in components/frontend/
- [x] T004 [P] Configure Go linting with golangci-lint in components/operator/
- [x] T005 Create Kubernetes test cluster configuration for integration testing
- [x] T006 Set up test namespaces and RBAC for multi-tenancy testing

## Phase 3.2: Contract Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3

**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Webhook Contract Tests
- [x] T007 [P] Contract test POST /api/v1/webhooks/github in components/backend/tests/contract/webhook_github_test.go
- [x] T008 [P] Contract test POST /api/v1/webhooks/jira in components/backend/tests/contract/webhook_jira_test.go
- [x] T009 [P] Contract test POST /api/v1/webhooks/slack in components/backend/tests/contract/webhook_slack_test.go
- [x] T010 [P] Contract test POST /api/v1/webhooks/{source}/validate in components/backend/tests/contract/webhook_validate_test.go

### Session Management Contract Tests
- [x] T011 [P] Contract test GET /api/v1/namespaces/{ns}/sessions in components/backend/tests/contract/sessions_list_test.go
- [x] T012 [P] Contract test POST /api/v1/namespaces/{ns}/sessions in components/backend/tests/contract/sessions_create_test.go
- [x] T013 [P] Contract test GET /api/v1/namespaces/{ns}/sessions/{id} in components/backend/tests/contract/sessions_get_test.go
- [x] T014 [P] Contract test GET /api/v1/namespaces/{ns}/sessions/{id}/artifacts in components/backend/tests/contract/sessions_artifacts_test.go
- [x] T015 [P] Contract test GET /api/v1/user/namespaces in components/backend/tests/contract/user_namespaces_test.go

### CRD and Operator Integration Tests
- [ ] T016 [P] Integration test Session CRD creation in components/operator/tests/integration/session_creation_test.go
- [ ] T017 [P] Integration test NamespacePolicy validation in components/operator/tests/integration/policy_validation_test.go
- [ ] T018 [P] Integration test Session reconciliation in components/operator/tests/integration/session_reconcile_test.go
- [ ] T019 [P] Integration test multi-tenant isolation in components/operator/tests/integration/namespace_isolation_test.go

### End-to-End Scenario Tests
- [ ] T020 [P] E2E test GitHub webhook → session creation in tests/e2e/github_webhook_test.go
- [ ] T021 [P] E2E test web UI session viewing with RBAC in tests/e2e/ui_session_view_test.go
- [ ] T022 [P] E2E test policy enforcement in tests/e2e/policy_enforcement_test.go
- [ ] T023 [P] E2E test artifact generation and access in tests/e2e/artifact_access_test.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### New CRD Definitions
- [x] T024 [P] Create Session CRD definition in components/manifests/crds/session_v1alpha1.yaml (REFERENCE: data-model.md Session CRD schema lines 13-78)
- [x] T025 [P] Create NamespacePolicy CRD definition in components/manifests/crds/namespacepolicy_v1alpha1.yaml (REFERENCE: data-model.md NamespacePolicy schema lines 88-128)
- [x] T026 [P] Create validation webhooks for Session CRD in components/operator/pkg/webhooks/session_validator.go
- [x] T027 [P] Create validation webhooks for NamespacePolicy in components/operator/pkg/webhooks/policy_validator.go

### Backend API Implementation
- [x] T028 [P] Webhook authentication middleware in components/backend/pkg/middleware/webhook_auth.go
- [x] T029 [P] RBAC middleware with Kubernetes integration in components/backend/pkg/middleware/rbac.go
- [x] T030 [P] Namespace resolver service in components/backend/pkg/services/namespace_resolver.go
- [x] T031 [P] Session manager service in components/backend/pkg/services/session_manager.go
- [x] T032 [P] Artifact indexer service in components/backend/pkg/services/artifact_indexer.go
- [x] T033 POST /api/v1/webhooks/{source} handler in components/backend/pkg/handlers/webhook_handler.go (REFERENCE: contracts/webhooks.yaml OpenAPI spec)
- [x] T034 GET /api/v1/namespaces/{ns}/sessions handler in components/backend/pkg/handlers/session_list_handler.go (REFERENCE: contracts/sessions.yaml lines 12-74)
- [x] T035 POST /api/v1/namespaces/{ns}/sessions handler in components/backend/pkg/handlers/session_create_handler.go (REFERENCE: contracts/sessions.yaml lines 75-134)
- [x] T036 GET /api/v1/namespaces/{ns}/sessions/{id} handler in components/backend/pkg/handlers/session_get_handler.go (REFERENCE: contracts/sessions.yaml lines 135-174)
- [x] T037 GET /api/v1/user/namespaces handler in components/backend/pkg/handlers/user_namespaces_handler.go (REFERENCE: contracts/sessions.yaml lines 258-290)

### Operator Controllers
- [x] T038 Session controller reconciler in components/operator/internal/controllers/session_controller.go
- [x] T039 NamespacePolicy controller in components/operator/internal/controllers/policy_controller.go
- [ ] T040 [P] Session status updater in components/operator/pkg/status/session_status.go
- [ ] T041 [P] Workload creator service in components/operator/pkg/workload/workload_creator.go

### Frontend Multi-tenancy
- [x] T042 [P] Namespace selector component in components/frontend/src/components/ui/namespace-selector.tsx
- [ ] T043 [P] Session type selector component in components/frontend/src/components/ui/session-type-selector.tsx
- [ ] T044 [P] RBAC-aware session list in components/frontend/src/components/session-list.tsx
- [ ] T045 [P] Multi-tenant session detail view in components/frontend/src/components/session-detail.tsx
- [ ] T046 Update session types in components/frontend/src/types/session.ts
- [ ] T047 Update API client for namespace-scoped requests in components/frontend/src/services/api-client.ts
- [ ] T048 Add namespace routing to pages in components/frontend/src/app/namespace/[ns]/page.tsx

## Phase 3.4: Clean Migration (Replace AgenticSession)

### Replace AgenticSession with Session CRD
- [x] T049 Create cleanup script to remove existing AgenticSession CRDs in scripts/cleanup_agentic_sessions.go
- [x] T050 Replace AgenticSession types with Session types in components/backend/pkg/types/session.go
- [x] T051 Update frontend to use new Session CRD exclusively in components/frontend/src/types/session.ts
- [x] T052 Remove old AgenticSession CRD definition from components/manifests/crd.yaml

### Runner Framework Migration
- [x] T053 [P] Make claude-code-runner namespace-aware in components/runners/claude-code-runner/main.py
- [ ] T054 [P] Create generic runner interface in components/runners/pkg/runner_interface.go
- [ ] T055 [P] Framework registry service in components/runners/pkg/registry/framework_registry.go
- [ ] T056 Add artifact storage abstraction in components/runners/pkg/storage/artifact_storage.go

## Phase 3.5: Integration & RBAC

### Kubernetes RBAC Integration
- [ ] T057 Create ClusterRole for multi-tenant access in components/manifests/rbac/multi_tenant_roles.yaml
- [ ] T058 Create namespace-scoped RoleBindings template in components/manifests/rbac/namespace_bindings.yaml
- [ ] T059 OIDC integration for user authentication in components/backend/pkg/auth/oidc.go
- [ ] T060 Service account token validation in components/backend/pkg/auth/service_account.go

### Policy Enforcement
- [ ] T061 Budget tracking service in components/backend/pkg/services/budget_tracker.go
- [ ] T062 Model constraint validator in components/operator/pkg/validators/model_validator.go
- [ ] T063 Tool constraint enforcer in components/operator/pkg/validators/tool_validator.go
- [ ] T064 Notification webhook client in components/backend/pkg/notifications/webhook_client.go

### Observability & Audit
- [ ] T065 [P] Structured logging setup in components/backend/pkg/logging/structured_logger.go
- [ ] T066 [P] Audit trail service in components/backend/pkg/audit/audit_service.go
- [ ] T067 [P] Metrics collection for sessions in components/operator/pkg/metrics/session_metrics.go
- [ ] T068 Frontend logging to backend in components/frontend/src/utils/logger.ts

## Phase 3.6: Polish & Performance

### Unit Tests
- [ ] T069 [P] Unit tests for webhook authentication in components/backend/pkg/middleware/webhook_auth_test.go
- [ ] T070 [P] Unit tests for namespace resolution in components/backend/pkg/services/namespace_resolver_test.go
- [ ] T071 [P] Unit tests for session controller in components/operator/internal/controllers/session_controller_test.go
- [ ] T072 [P] Unit tests for policy validation in components/operator/pkg/webhooks/policy_validator_test.go
- [ ] T073 [P] Unit tests for frontend components in components/frontend/src/components/__tests__/

### Performance & Cleanup
- [ ] T074 Performance testing for 100+ concurrent sessions in tests/performance/concurrent_sessions_test.go
- [ ] T075 Webhook response time optimization (<2s) in components/backend/pkg/handlers/
- [ ] T076 [P] Add caching for RBAC decisions in components/backend/pkg/cache/rbac_cache.go
- [ ] T077 [P] Verify complete removal of AgenticSession references in components/
- [ ] T078 Update documentation and quickstart guide in specs/001-we-currently-run/quickstart.md

### Final Validation
- [ ] T079 Run complete quickstart validation scenarios (EXECUTE: quickstart.md scenarios 1-4, lines 90-202)
- [ ] T080 Verify all tests pass and coverage meets requirements (VALIDATE: All contract tests in tests/contract/ pass)
- [ ] T081 Performance benchmark validation against requirements (TARGET: research.md scalability goals lines 79-85)
- [ ] T082 Security scan and RBAC verification (VALIDATE: quickstart.md RBAC test scenarios lines 206-225)

## Dependencies

**Setup Phase:**
- T001 must complete before all other tasks
- T002-T006 can run in parallel after T001

**Test Phase (TDD Critical Path):**
- T007-T023 (all tests) MUST complete and FAIL before T024-T078
- Contract tests (T007-T015) can run in parallel
- Integration tests (T016-T019) can run in parallel
- E2E tests (T020-T023) can run in parallel

**Implementation Phase:**
- CRD tasks (T024-T027) must complete before controller tasks (T038-T041)
- Service tasks (T028-T032) can run in parallel
- Handler tasks (T033-T037) depend on service tasks
- Frontend tasks (T042-T048) can run mostly in parallel
- Cleanup tasks (T049-T052) can run after core implementation
- Runner tasks (T053-T056) can run in parallel

**Integration Phase:**
- RBAC tasks (T057-T060) can run in parallel after core implementation
- Policy tasks (T061-T064) depend on controller implementation
- Observability tasks (T065-T068) can run in parallel

**Polish Phase:**
- Unit tests (T069-T073) can run in parallel after implementation
- Performance tasks (T074-T076) depend on full implementation
- Cleanup tasks (T077-T078) run after performance validation
- Final validation (T079-T082) runs sequentially at end

## Parallel Execution Examples

### Test Phase Parallel Launch:
```bash
# Contract tests (can run simultaneously):
Task: "Contract test POST /api/v1/webhooks/github in components/backend/tests/contract/webhook_github_test.go"
Task: "Contract test POST /api/v1/webhooks/jira in components/backend/tests/contract/webhook_jira_test.go"
Task: "Contract test GET /api/v1/namespaces/{ns}/sessions in components/backend/tests/contract/sessions_list_test.go"
Task: "Contract test POST /api/v1/namespaces/{ns}/sessions in components/backend/tests/contract/sessions_create_test.go"
```

### Service Implementation Parallel Launch:
```bash
# Core services (different files, can run simultaneously):
Task: "Webhook authentication middleware in components/backend/pkg/middleware/webhook_auth.go"
Task: "RBAC middleware with Kubernetes integration in components/backend/pkg/middleware/rbac.go"
Task: "Namespace resolver service in components/backend/pkg/services/namespace_resolver.go"
Task: "Session manager service in components/backend/pkg/services/session_manager.go"
```

### Frontend Components Parallel Launch:
```bash
# UI components (different files, can run simultaneously):
Task: "Namespace selector component in components/frontend/src/components/ui/namespace-selector.tsx"
Task: "Session type selector component in components/frontend/src/components/ui/session-type-selector.tsx"
Task: "RBAC-aware session list in components/frontend/src/components/session-list.tsx"
```

## Implementation Detail References

**CRITICAL: Each implementation task MUST reference the corresponding design documents**

### CRD Implementation (T024-T027)
- **Session CRD**: Follow exact schema from `data-model.md` lines 13-78
- **NamespacePolicy CRD**: Follow exact schema from `data-model.md` lines 88-128
- **Validation Rules**: Implement constraints from `data-model.md` lines 179-191
- **State Transitions**: Follow state machine from `data-model.md` lines 79-84

### API Handler Implementation (T033-T037)
- **OpenAPI Contracts**: All handlers must match `contracts/*.yaml` specifications exactly
- **Session List**: Implement pagination, filtering per `contracts/sessions.yaml` lines 12-74
- **Webhook Processing**: Follow authentication flow per `contracts/webhooks.yaml`
- **RBAC Integration**: Reference `research.md` OIDC decision lines 16-34

### Validation Testing (T079-T082)
- **Test Scenarios**: Execute all scenarios from `quickstart.md` lines 90-202
- **Success Criteria**: Validate against checklist in `quickstart.md` lines 206-230
- **Performance Targets**: Meet scalability goals from `research.md` lines 79-85
- **RBAC Testing**: Verify namespace isolation per `quickstart.md` lines 206-230

## Notes

- **[P] tasks** = different files, no dependencies, can run in parallel
- **TDD Critical**: T007-T023 must be written and failing before T024-T078
- **Design References**: Each task must reference specific lines in design documents
- **Clean migration**: Replace AgenticSession CRDs with new Session CRDs (no backward compatibility)
- **Multi-tenancy**: All implementation must respect namespace boundaries
- **Testing**: Use real Kubernetes cluster for integration tests
- **Performance**: Target <2s webhook response, 100+ concurrent sessions
- **Security**: All changes must pass RBAC and security validation

## Task Generation Rules Applied

✅ Each contract file → contract test task marked [P]
✅ Each entity in data-model → model/CRD creation task marked [P]
✅ Each endpoint → implementation task (sequential if shared files)
✅ Each user story → integration test marked [P]
✅ Different files = marked [P] for parallel execution
✅ Same file = sequential (no [P])
✅ Tests before implementation (TDD) strictly enforced