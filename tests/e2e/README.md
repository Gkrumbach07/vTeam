# End-to-End Tests

This directory contains end-to-end tests that validate complete user workflows across the multi-tenant agentic platform.

## Test Scenarios

1. **GitHub Webhook Integration**: Tests external tool webhook processing
2. **Web UI Session Management**: Tests RBAC-aware session viewing
3. **Policy Enforcement**: Tests namespace policy constraints
4. **Artifact Management**: Tests artifact generation and access

## Requirements

- Kubernetes cluster with multi-tenancy configured
- External tool webhook endpoints (for integration tests)
- RBAC policies configured for test namespaces

## Running Tests

```bash
go test ./tests/e2e/...
```

## Test Data

Each test creates its own isolated namespace and cleans up after completion.