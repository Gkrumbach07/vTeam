# Implementation Plan: Multi-Tenant Agentic Platform with External Tool Integration

**Branch**: `001-we-currently-run` | **Date**: 2025-09-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-we-currently-run/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
4. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
5. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, or `GEMINI.md` for Gemini CLI).
6. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
7. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
8. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
**MIGRATION PROJECT**: Transform existing `AgenticSession` CRD (website analysis focused) into generic multi-tenant agentic platform. Current system has single-namespace website analysis with Claude - needs evolution to support external tool webhooks (Jira, Slack, GitHub), namespace-based isolation, RBAC access control, and generic Session CRD supporting multiple agentic frameworks. Implements zero-touch and full-login interaction modes with append-only audit trails.

## Technical Context
**Language/Version**: Go 1.21+ (backend/operator), Node.js 18+ (frontend), Python 3.11+ (runners)
**Primary Dependencies**: Kubernetes client-go, Operator SDK, NextJS, existing agentic runner frameworks
**Storage**: Kubernetes etcd (CRDs), S3/PVC (artifacts), append-only logging
**Testing**: Go testing, Jest/Cypress, pytest
**Target Platform**: OpenShift/Kubernetes cluster
**Project Type**: web (existing components/ structure: backend + operator + frontend + runners)
**Performance Goals**: 100+ concurrent sessions, <2s webhook response, multi-tenant isolation
**Constraints**: Kubernetes RBAC compliance, OpenShift security policies, immutable audit logs, extend existing components/ architecture
**Scale/Scope**: 50+ namespaces, 1000+ users, multiple agentic frameworks

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: 4 (extending existing: backend, operator, frontend, runners) - exceeds max 3 but unavoidable with existing architecture
- Using framework directly? Yes - Kubernetes client-go, NextJS (existing choices)
- Single data model? Yes - Session CRD with framework-agnostic references
- Avoiding patterns? Using existing Kubernetes operator patterns

**Architecture**:
- EVERY feature as library? Kubernetes operators are services, not libraries
- Libraries listed: webhook-auth, namespace-resolver, session-manager, artifact-indexer
- CLI per library: kubectl plugin for session management
- Library docs: Will include llms.txt for AI context

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? Yes
- Git commits show tests before implementation? Yes
- Order: Contract→Integration→E2E→Unit strictly followed? Yes
- Real dependencies used? Yes - actual Kubernetes cluster for testing
- Integration tests for: CRD changes, webhook contracts, RBAC policies
- FORBIDDEN: Implementation before test, skipping RED phase

**Observability**:
- Structured logging included? Yes - JSON structured logs
- Frontend logs → backend? Yes - centralized logging
- Error context sufficient? Yes - with namespace/session context

**Versioning**:
- Version number assigned? 0.1.0 (new feature)
- BUILD increments on every change? Yes
- Breaking changes handled? CRD versioning strategy required

## Project Structure

### Documentation (this feature)
```
specs/001-we-currently-run/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
# CURRENT STATE: Single-namespace website analysis platform
components/
├── backend/             # Go API - currently single namespace, no webhooks
├── operator/            # Kubernetes operator - AgenticSession CRD (website-specific)
├── frontend/            # NextJS UI - single session type, no namespace selector
├── runners/             # Single Claude runner for website analysis
│   └── claude-code-runner/
└── manifests/           # Basic K8s manifests - no RBAC, single namespace
    ├── crd.yaml         # AgenticSession CRD (website analysis)
    ├── rbac.yaml
    ├── operator-deployment.yaml
    └── backend/frontend deployments

# MIGRATION TARGET: Multi-tenant generic agentic platform
components/
├── backend/             # MIGRATE: Add webhook endpoints, RBAC, multi-tenancy
├── operator/            # MIGRATE: Generic Session CRD, namespace policies
├── frontend/            # MIGRATE: Namespace selector, session type support
├── runners/             # MIGRATE: Framework-agnostic runner architecture
└── manifests/           # MIGRATE: Multi-tenant RBAC, policies, new CRDs

# LEGACY (not modified in this feature)
demos/                   # Keep as-is, legacy demos
tools/                   # Keep as-is, legacy tooling
```

**Structure Decision**: Migration from website-analysis to generic multi-tenant platform

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - OAuth provider selection (corporate SSO vs GitHub OAuth vs custom)
   - Session/artifact retention policies (30d, 90d, 1yr?)
   - Error notification mechanisms (email, Slack, webhooks?)
   - Scalability targets (concurrent sessions, users, namespaces)
   - Policy configuration approach (YAML, UI, API?)

2. **Generate and dispatch research agents**:
   ```
   Task: "Research OAuth integration options for Kubernetes RBAC"
   Task: "Find best practices for multi-tenant Kubernetes operators"
   Task: "Research session retention patterns for audit compliance"
   Task: "Find CRD versioning strategies for breaking changes"
   Task: "Research webhook authentication patterns (API keys vs signatures)"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Session CRD: spec, status, history, artifacts
   - Namespace: policies, access controls, users
   - Policy: constraints, budgets, approvals
   - User: RBAC bindings, namespace memberships
   - Audit: immutable log entries with context

2. **Generate API contracts** from functional requirements:
   - Webhook endpoints: POST /webhooks/{source}
   - Session management: GET/POST /api/v1/namespaces/{ns}/sessions
   - RBAC endpoints: GET /api/v1/user/namespaces, permissions
   - Output OpenAPI schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - Webhook authentication tests (must fail initially)
   - Session CRUD with RBAC tests
   - Namespace isolation tests
   - CRD reconciliation tests

4. **Extract test scenarios** from user stories:
   - Zero-touch webhook → session creation
   - UI session viewing with namespace scoping
   - Policy enforcement during session execution
   - Artifact generation and discovery

5. **Update agent file incrementally** (O(1) operation):
   - Update CLAUDE.md with Session CRD context
   - Add webhook authentication patterns
   - Include Kubernetes operator best practices
   - Document multi-tenancy design decisions

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, CLAUDE.md

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `/templates/tasks-template.md` as base
- CRD definition and validation webhook tasks
- Webhook authentication and routing implementation
- RBAC integration with Kubernetes
- Session reconciliation controller
- Frontend namespace-aware components
- Integration tests for multi-tenancy

**Ordering Strategy**:
- CRD → Operator → Backend → Frontend
- Tests before each implementation phase
- Parallel tasks: [P] CRD tests, webhook tests, UI components

**Estimated Output**: 35-40 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 4th project (runners) | Existing architecture has runners; extending with framework support | Cannot consolidate without breaking existing deployment |
| Service architecture | Kubernetes operators must be services; extending existing | Library approach incompatible with existing K8s patterns |

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (with documented complexity)
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*