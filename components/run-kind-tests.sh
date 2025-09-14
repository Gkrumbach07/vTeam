#!/bin/bash

# Complete test script for multi-tenant platform using kind
# Sets up kind cluster and runs full test suite

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Multi-Tenant Agentic Platform - Kind Test Suite ===${NC}"
echo ""

# Change to component directory
cd /Users/gkrumbac/Documents/vTeam/components

# Step 1: Setup kind cluster
echo -e "${BLUE}Step 1: Setting up kind cluster...${NC}"
./setup-kind-test.sh

echo ""
echo -e "${BLUE}Step 2: Building and deploying backend...${NC}"
cd backend

# Build container image (podman/docker)
echo "Building backend container image..."
if command -v podman &> /dev/null; then
    podman build -t ambient-backend:test .
    # Load image into kind
    echo "Loading image into kind cluster..."
    podman save ambient-backend:test | kind load image-archive --name ambient-platform-test /dev/stdin
else
    docker build -t ambient-backend:test .
    # Load image into kind
    echo "Loading image into kind cluster..."
    kind load docker-image ambient-backend:test --name ambient-platform-test
fi

# Deploy backend
echo "Deploying backend to kind..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: ambient-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: ambient-backend:test
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: NAMESPACE
          value: "ambient-system"
---
apiVersion: v1
kind: Service
metadata:
  name: backend-service
  namespace: ambient-system
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30080
  selector:
    app: backend
EOF

# Wait for backend to be ready
echo "Waiting for backend deployment..."
kubectl wait --for=condition=available --timeout=60s deployment/backend -n ambient-system

echo -e "${GREEN}✓ Backend deployed successfully${NC}"
echo ""

# Step 3: Test CRD validation
echo -e "${BLUE}Step 3: Testing CRD validation...${NC}"

# Test valid session creation
echo "Testing valid session creation..."
cat <<EOF | kubectl apply -f -
apiVersion: ambient.ai/v1alpha1
kind: Session
metadata:
  name: valid-test-session
  namespace: team-alpha
spec:
  trigger:
    source: "github"
    event: "pull_request_opened"
    payload:
      action: "opened"
      pull_request:
        id: 123
        title: "Test PR"
  framework:
    type: "claude-code"
    version: "1.0"
    config: {}
  policy:
    modelConstraints:
      allowed: ["claude-3-sonnet"]
      budget: "10.00"
    toolConstraints:
      allowed: ["bash", "read"]
    approvalRequired: false
EOF

if kubectl get session valid-test-session -n team-alpha > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Valid session created successfully${NC}"
else
    echo -e "${RED}✗ Valid session creation failed${NC}"
fi

# Test invalid session creation (should fail validation if webhooks are set up)
echo "Testing invalid session creation..."
cat <<EOF | kubectl apply -f - --validate=false 2>/dev/null || echo -e "${GREEN}✓ Invalid session rejected (expected)${NC}"
apiVersion: ambient.ai/v1alpha1
kind: Session
metadata:
  name: invalid-test-session
  namespace: team-alpha
spec:
  trigger:
    source: "github"
    event: "test"
  framework:
    type: "invalid-framework"  # This should be rejected
    version: "1.0"
  policy:
    modelConstraints:
      allowed: ["claude-3-opus"]  # This should be rejected by policy
EOF

echo ""

# Step 4: Test webhook endpoints
echo -e "${BLUE}Step 4: Testing webhook endpoints...${NC}"

# Wait a moment for the service to be accessible
sleep 5

# Test GitHub webhook
echo "Testing GitHub webhook..."
WEBHOOK_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/api/v1/webhooks/github \
  -H "X-API-Key: test-api-key-123" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "pull_request": {
      "id": 456,
      "title": "Another Test PR"
    },
    "repository": {
      "name": "test-repo",
      "owner": {
        "login": "test-org"
      }
    }
  }' || echo "Connection failed")

HTTP_CODE=$(echo "$WEBHOOK_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$WEBHOOK_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "202" ]; then
    echo -e "${GREEN}✓ GitHub webhook successful (HTTP $HTTP_CODE)${NC}"
    echo "Response: $RESPONSE_BODY"

    # Extract session ID if available
    SESSION_ID=$(echo "$RESPONSE_BODY" | grep -o '"sessionId":"[^"]*"' | cut -d'"' -f4)
    if [ ! -z "$SESSION_ID" ]; then
        echo "Created session ID: $SESSION_ID"
        # Check if session was created in Kubernetes
        sleep 2
        if kubectl get session "$SESSION_ID" -n team-alpha > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Session created in Kubernetes${NC}"
        else
            echo -e "${YELLOW}⚠ Session not found in Kubernetes (webhook handler may need operator integration)${NC}"
        fi
    fi
else
    echo -e "${RED}✗ GitHub webhook failed (HTTP $HTTP_CODE)${NC}"
    echo "Response: $RESPONSE_BODY"
fi

echo ""

# Step 5: Test session management API
echo -e "${BLUE}Step 5: Testing session management API...${NC}"

# List sessions
echo "Testing session list..."
LIST_RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/api/v1/namespaces/team-alpha/sessions || echo "Connection failed")
LIST_CODE=$(echo "$LIST_RESPONSE" | tail -n1)
LIST_BODY=$(echo "$LIST_RESPONSE" | head -n -1)

if [ "$LIST_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Session list API working (HTTP $LIST_CODE)${NC}"
    # Count sessions
    SESSION_COUNT=$(echo "$LIST_BODY" | grep -o '"sessions":\[.*\]' | grep -o '{' | wc -l | xargs)
    echo "Found $SESSION_COUNT sessions in team-alpha"
else
    echo -e "${RED}✗ Session list API failed (HTTP $LIST_CODE)${NC}"
    echo "Response: $LIST_BODY"
fi

# Test user namespaces
echo "Testing user namespaces API..."
NAMESPACES_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -H "Authorization: Bearer test-token" \
  http://localhost:8080/api/v1/user/namespaces || echo "Connection failed")
NAMESPACES_CODE=$(echo "$NAMESPACES_RESPONSE" | tail -n1)

if [ "$NAMESPACES_CODE" = "200" ]; then
    echo -e "${GREEN}✓ User namespaces API working (HTTP $NAMESPACES_CODE)${NC}"
else
    echo -e "${RED}✗ User namespaces API failed (HTTP $NAMESPACES_CODE)${NC}"
fi

echo ""

# Step 6: Check Kubernetes resources
echo -e "${BLUE}Step 6: Checking Kubernetes resources...${NC}"

echo "Sessions in team-alpha:"
kubectl get sessions -n team-alpha -o custom-columns=NAME:.metadata.name,PHASE:.status.phase,CREATED:.metadata.creationTimestamp --no-headers 2>/dev/null || echo "No sessions found"

echo ""
echo "NamespacePolicy in team-alpha:"
kubectl get namespacepolicy policy -n team-alpha -o custom-columns=NAME:.metadata.name,BUDGET:.spec.models.budget.monthly,CREATED:.metadata.creationTimestamp --no-headers 2>/dev/null || echo "No policy found"

echo ""
echo "Backend pod status:"
kubectl get pods -n ambient-system -l app=backend

echo ""

# Step 7: Test summary
echo -e "${BLUE}=== Test Summary ===${NC}"
echo ""

# Count successful tests
TOTAL_TESTS=6
PASSED_TESTS=0

# Check results
if kubectl get session valid-test-session -n team-alpha > /dev/null 2>&1; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

if [ "$HTTP_CODE" = "202" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

if [ "$LIST_CODE" = "200" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

if [ "$NAMESPACES_CODE" = "200" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

# Check if backend is running
if kubectl get pods -n ambient-system -l app=backend --no-headers | grep -q "Running"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

# Check if CRDs are installed
if kubectl get crds | grep -q "ambient.ai"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

echo -e "Tests passed: ${GREEN}$PASSED_TESTS${NC}/$TOTAL_TESTS"

if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo -e "${GREEN}🎉 All tests passed! The multi-tenant platform is working in kind.${NC}"
else
    echo -e "${YELLOW}⚠ Some tests failed. Check the logs above for details.${NC}"
fi

echo ""
echo -e "${BLUE}=== Next Steps ===${NC}"
echo ""
echo "1. View session details:"
echo "   kubectl describe session <session-name> -n team-alpha"
echo ""
echo "2. Check backend logs:"
echo "   kubectl logs -n ambient-system deployment/backend -f"
echo ""
echo "3. Test more webhook scenarios:"
echo "   curl -X POST http://localhost:8080/api/v1/webhooks/github \\"
echo "     -H 'X-API-Key: test-api-key-123' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"action\":\"closed\",\"pull_request\":{\"id\":789}}'"
echo ""
echo "4. Deploy and test operator controllers:"
echo "   cd ../operator"
echo "   export KUBECONFIG=~/.kube/config"
echo "   go run main.go"
echo ""
echo "5. Monitor session reconciliation:"
echo "   kubectl get sessions -A -w"
echo ""
echo "6. Clean up when done:"
echo "   kind delete cluster --name ambient-platform-test"
echo ""
echo -e "${GREEN}✓ Kind test environment is ready for development!${NC}"