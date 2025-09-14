# Migration Strategy: AgenticSession → Multi-Tenant Session Platform

**Date**: 2025-09-14
**Task**: T001 - Analyze current AgenticSession CRD and create migration strategy

## Current State Analysis

### Existing AgenticSession CRD
- **API Group**: `vteam.ambient-code`
- **Kind**: `AgenticSession`
- **Scope**: Namespaced (but single-tenant usage)
- **Purpose**: Website analysis with Claude
- **Required Fields**: `prompt`, `websiteURL`
- **LLM Settings**: Fixed Claude model configuration
- **Status Tracking**: Phase-based with detailed execution history

### Current Backend Architecture
- **Language**: Go with Gin framework
- **Kubernetes Integration**: Dynamic client for CRD operations
- **Single Namespace**: Uses environment variable or "default"
- **API Endpoints**:
  - `/api/agentic-sessions` (CRUD operations)
  - No webhook support
  - No RBAC integration
  - No external tool integration

### Current Frontend
- **Technology**: NextJS with TypeScript
- **Session Types**: `AgenticSession` with website-specific fields
- **UI Features**: Single session type, no namespace selector
- **Authentication**: No OIDC or RBAC integration

## Migration Requirements

### 1. CRD Evolution: AgenticSession → Session
**Breaking Changes Required:**
- Change API group from `vteam.ambient-code` to `ambient.ai`
- Replace website-specific fields with generic framework support
- Add multi-tenancy support with namespace policies
- Add webhook trigger support
- Add framework-agnostic artifact references

**Migration Path:**
- **Phase 1**: Deploy new Session CRD alongside existing
- **Phase 2**: Migrate existing data if needed (or clean start)
- **Phase 3**: Remove old AgenticSession CRD entirely

### 2. Backend Transformation: Single → Multi-Tenant
**Current Limitations:**
- Single namespace operation
- No webhook endpoints
- No RBAC middleware
- No policy enforcement

**Required Changes:**
- Add webhook authentication middleware
- Implement namespace-scoped operations
- Add RBAC integration with Kubernetes
- Add policy validation
- Add external tool integration endpoints

### 3. Frontend Evolution: Simple → Multi-Tenant UI
**Current Limitations:**
- Single session type support
- No namespace awareness
- No user authentication
- Fixed session creation flow

**Required Changes:**
- Add namespace selector component
- Add RBAC-aware UI elements
- Add multiple session type support
- Add OIDC authentication flow

## Migration Strategy

### Strategy 1: Clean Break (RECOMMENDED)
**Approach**: Replace existing system entirely with new multi-tenant platform

**Benefits:**
- Clean architecture from start
- No compatibility complexity
- Faster development
- Purpose-built for multi-tenancy

**Drawbacks:**
- Existing data will be lost
- Requires redeployment
- Users need to recreate sessions

**Implementation:**
1. Build new Session CRD and controllers
2. Build new multi-tenant backend
3. Build new namespace-aware frontend
4. Deploy alongside existing system
5. Migrate/recreate important sessions manually
6. Switch over and remove old system

### Strategy 2: Gradual Migration (NOT RECOMMENDED)
**Approach**: Support both CRDs during transition period

**Benefits:**
- Preserves existing data
- Gradual user migration
- Rollback capability

**Drawbacks:**
- Complex dual-CRD support
- Compatibility layer maintenance
- Delayed multi-tenancy benefits
- Technical debt accumulation

**Rejected Because:**
- User explicitly stated no backward compatibility needed
- Clean architecture is more important than data preservation
- Current system has limited production usage

## Implementation Plan

### Phase 1: Foundation (Tasks T002-T006)
- Set up testing frameworks
- Configure linting and development tools
- Prepare test clusters with multi-tenancy support

### Phase 2: TDD Test Suite (Tasks T007-T023)
- Write failing tests for all new functionality
- Contract tests for webhook endpoints
- Integration tests for multi-tenancy
- E2E tests for complete workflows

### Phase 3: Core Implementation (Tasks T024-T048)
- New Session CRD with generic framework support
- New NamespacePolicy CRD for constraints
- Multi-tenant backend with webhook support
- Namespace-aware frontend with RBAC

### Phase 4: Clean Migration (Tasks T049-T056)
- Remove old AgenticSession CRD entirely
- Update all references to new Session type
- Migrate runner architecture to be framework-agnostic
- Clean up any remaining legacy code

### Phase 5: Integration & Polish (Tasks T057-T082)
- RBAC and policy enforcement
- Observability and audit trails
- Performance optimization
- Final validation and documentation

## Risk Assessment

### High-Risk Items
1. **Data Loss**: Existing AgenticSessions will be lost
   - **Mitigation**: Document important sessions before migration
   - **Acceptance**: User confirmed no backward compatibility needed

2. **Multi-tenancy Complexity**: New architectural patterns
   - **Mitigation**: Thorough testing with real Kubernetes cluster
   - **Validation**: Integration tests for namespace isolation

3. **RBAC Integration**: Complex Kubernetes permission model
   - **Mitigation**: Start with simple viewer/editor roles
   - **Testing**: Dedicated RBAC validation scenarios

### Medium-Risk Items
1. **Performance**: 100+ concurrent sessions target
   - **Mitigation**: Performance testing and optimization tasks
   - **Monitoring**: Metrics collection for session execution

2. **Framework Migration**: Making runners generic
   - **Mitigation**: Gradual runner refactoring
   - **Testing**: Multiple framework compatibility tests

## Success Criteria

### Technical
- [ ] All tests pass (contract, integration, e2e, unit)
- [ ] Performance targets met (<2s webhook response, 100+ sessions)
- [ ] Multi-tenancy working with namespace isolation
- [ ] RBAC integration functional
- [ ] External tool webhooks processing correctly

### Functional
- [ ] Users can trigger sessions from external tools
- [ ] Namespace-scoped session visibility working
- [ ] Policy enforcement preventing violations
- [ ] Artifact generation and access working
- [ ] Audit trail capturing all actions

### Operational
- [ ] Clean removal of all AgenticSession references
- [ ] Documentation updated for new architecture
- [ ] Deployment manifests ready for production
- [ ] Monitoring and logging configured

## Timeline Estimate

**Total Tasks**: 82 tasks across 6 phases
**Estimated Duration**: 3-4 weeks (depending on testing thoroughness)
**Critical Path**: TDD tests → Core implementation → Integration testing

**Phase Breakdown:**
- Phase 1 (Setup): 2-3 days
- Phase 2 (TDD Tests): 4-5 days
- Phase 3 (Core Implementation): 8-10 days
- Phase 4 (Clean Migration): 2-3 days
- Phase 5 (Integration & Polish): 5-6 days

**Parallel Execution Opportunities:**
- 47 tasks marked [P] can run concurrently
- Contract tests can be written in parallel
- Frontend and backend implementation can overlap
- Unit tests and documentation can be done in parallel

## Next Steps

1. ✅ **T001 Complete**: Migration strategy documented
2. **T002-T006**: Set up testing frameworks and development environment
3. **T007-T023**: Begin TDD phase with failing tests
4. Execute remaining tasks following dependency order

This migration represents a significant architectural evolution from a simple website analysis tool to a comprehensive multi-tenant agentic platform. The clean break approach ensures we build the right architecture from the start rather than being constrained by legacy design decisions.