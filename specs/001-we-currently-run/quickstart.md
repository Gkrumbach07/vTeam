# Quickstart Guide: Multi-Tenant Agentic Platform

**Feature**: Multi-Tenant Agentic Platform with External Tool Integration
**Phase**: 1 - Design & Contracts
**Date**: 2025-09-14

## Prerequisites
- Kubernetes cluster with OIDC configured
- kubectl access with cluster-admin permissions
- External tools (GitHub, Jira, Slack) with webhook capabilities

## Setup Steps

### 1. Deploy Platform Components

```bash
# Deploy CRDs and RBAC
kubectl apply -f components/manifests/crds/
kubectl apply -f components/manifests/rbac/

# Deploy operator
kubectl apply -f components/manifests/operator/

# Deploy backend service
kubectl apply -f components/manifests/backend/

# Deploy frontend
kubectl apply -f components/manifests/frontend/
```

### 2. Create Team Namespace and Policy

```bash
# Create namespace for team-alpha
kubectl create namespace team-alpha

# Apply namespace policy
cat <<EOF | kubectl apply -f -
apiVersion: ambient.ai/v1alpha1
kind: NamespacePolicy
metadata:
  name: policy
  namespace: team-alpha
spec:
  models:
    allowed: ["claude-3-sonnet"]
    budget:
      monthly: "100.00"
      currency: "USD"
  tools:
    allowed: ["bash", "edit", "read", "write"]
  retention:
    sessions: "90d"
    artifacts: "30d"
  webhookAuth:
    apiKeys:
      github: "test-api-key-123"
EOF
```

### 3. Set Up RBAC for Team Users

```bash
# Create role for team-alpha editors
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: session-editor
  namespace: team-alpha
rules:
- apiGroups: ["ambient.ai"]
  resources: ["sessions"]
  verbs: ["get", "list", "create", "watch"]
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
EOF

# Bind users to role
kubectl create rolebinding team-alpha-editors \
  --role=session-editor \
  --user=alice@company.com \
  --user=bob@company.com \
  --namespace=team-alpha
```

## Test Scenarios

### Scenario 1: GitHub Webhook → Session Creation

**Test Setup:**
1. Configure GitHub webhook to point to `https://platform.example.com/api/v1/webhooks/github`
2. Set webhook secret to API key from namespace policy
3. Create a pull request in monitored repository

**Expected Flow:**
1. GitHub sends webhook to platform
2. Platform authenticates using API key
3. Namespace resolved to `team-alpha` based on API key
4. Session CRD created in `team-alpha` namespace
5. Operator creates runner pod/job
6. Session status updated as execution proceeds

**Verification Commands:**
```bash
# Check webhook was received
curl -H "X-API-Key: test-api-key-123" \
     -H "Content-Type: application/json" \
     -d '{"action":"opened","pull_request":{"id":123}}' \
     https://platform.example.com/api/v1/webhooks/github

# Expected response:
# {
#   "sessionId": "uuid",
#   "namespace": "team-alpha",
#   "status": "accepted"
# }

# Verify session was created
kubectl get sessions -n team-alpha

# Watch session progress
kubectl get session <session-id> -n team-alpha -o yaml
```

### Scenario 2: Web UI Session Management

**Test Setup:**
1. Access web UI at `https://platform.example.com`
2. Authenticate with OIDC provider
3. Navigate to team-alpha namespace

**Expected Behavior:**
1. User sees namespace selector with `team-alpha` option
2. Session list shows only team-alpha sessions
3. User can view session details and logs
4. User with editor role can create manual sessions

**Verification Steps:**
```bash
# Test API access directly
export TOKEN=$(kubectl get secret user-token -o jsonpath='{.data.token}' | base64 -d)

curl -H "Authorization: Bearer $TOKEN" \
     https://platform.example.com/api/v1/namespaces/team-alpha/sessions

# Expected: List of sessions visible to user
```

### Scenario 3: Policy Enforcement

**Test Setup:**
1. Modify namespace policy to block certain models
2. Attempt to create session with blocked model

**Test Command:**
```bash
# Update policy to block claude-3-opus
kubectl patch namespacepolicy policy -n team-alpha --type=merge -p='{
  "spec": {
    "models": {
      "allowed": ["claude-3-sonnet"],
      "blocked": ["claude-3-opus"]
    }
  }
}'

# Try to create session with blocked model
curl -H "X-API-Key: test-api-key-123" \
     -H "Content-Type: application/json" \
     -d '{
       "framework": {"type": "claude-code", "config": {"model": "claude-3-opus"}}
     }' \
     https://platform.example.com/api/v1/namespaces/team-alpha/sessions
```

**Expected Result:**
- HTTP 403 Forbidden
- Error message about policy violation

### Scenario 4: Artifact Generation and Access

**Test Setup:**
1. Create session that generates artifacts
2. Verify artifacts are scoped to namespace
3. Test artifact download with proper permissions

**Verification:**
```bash
# Get session with artifacts
SESSION_ID=$(kubectl get sessions -n team-alpha -o jsonpath='{.items[0].metadata.name}')

# List artifacts
curl -H "Authorization: Bearer $TOKEN" \
     https://platform.example.com/api/v1/namespaces/team-alpha/sessions/$SESSION_ID/artifacts

# Expected: List of artifacts with download URLs

# Verify artifact is namespace-scoped
kubectl get session $SESSION_ID -n team-alpha -o yaml | grep artifacts -A 10
```

## Success Criteria Validation

### ✅ Namespace Isolation
- Sessions created in correct namespace based on webhook authentication
- Users only see sessions in namespaces they have access to
- Artifacts are scoped to namespace

### ✅ RBAC Integration
- Kubernetes RBAC controls session access
- Viewer vs editor permissions enforced
- Unauthorized access properly rejected

### ✅ Policy Enforcement
- Namespace policies validated during session creation
- Budget limits enforced
- Model and tool constraints applied

### ✅ Webhook Processing
- External tool webhooks processed correctly
- API key authentication working
- Namespace resolution from server-side policy

### ✅ Audit Trail
- All actions logged with context
- Session history append-only
- Policy changes audited

## Troubleshooting

### Common Issues

**Webhook returns 401 Unauthorized:**
- Check API key matches namespace policy
- Verify webhook headers include X-API-Key

**Session stuck in Pending:**
- Check operator logs: `kubectl logs -n ambient-system deployment/ambient-operator`
- Verify runner images are available
- Check resource quotas in namespace

**User cannot see sessions:**
- Verify RBAC bindings: `kubectl get rolebindings -n team-alpha`
- Check user identity in JWT token
- Confirm OIDC configuration

**Policy violations not enforced:**
- Check admission webhook status: `kubectl get validatingwebhookconfigurations`
- Verify webhook endpoints are reachable
- Review webhook logs for errors

### Debug Commands

```bash
# Check CRD status
kubectl get crds | grep ambient

# View operator logs
kubectl logs -n ambient-system -l app=ambient-operator

# Check webhook configuration
kubectl get validatingwebhookconfigurations ambient-session-webhook -o yaml

# Test policy validation
kubectl auth can-i create sessions --as=alice@company.com -n team-alpha

# View session controller events
kubectl get events -n team-alpha --field-selector involvedObject.kind=Session
```

## Next Steps

After successful quickstart validation:
1. Configure production OIDC provider
2. Set up monitoring and alerting
3. Implement backup/restore for session data
4. Scale testing with multiple namespaces
5. Add additional agentic framework runners

This quickstart demonstrates all core functionality and validates the multi-tenant architecture works end-to-end.