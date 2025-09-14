#!/bin/bash

# Setup script for kind test environment
# Creates a local Kubernetes cluster for testing the multi-tenant platform

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Multi-Tenant Agentic Platform - Kind Test Environment Setup ===${NC}"
echo ""

# Check prerequisites
check_prerequisite() {
    local cmd=$1
    local install_msg=$2

    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}✗ $cmd is not installed${NC}"
        echo "  $install_msg"
        return 1
    else
        echo -e "${GREEN}✓ $cmd is installed${NC}"
        return 0
    fi
}

echo "Checking prerequisites..."
PREREQ_FAILED=0

# Check for container runtime
if command -v podman &> /dev/null; then
    echo -e "${GREEN}✓ podman is installed${NC}"
elif command -v docker &> /dev/null; then
    echo -e "${GREEN}✓ docker is installed${NC}"
else
    echo -e "${RED}✗ Neither podman nor docker is installed${NC}"
    echo "  Install: brew install podman"
    PREREQ_FAILED=1
fi
check_prerequisite "kind" "Install kind: brew install kind OR go install sigs.k8s.io/kind@latest" || PREREQ_FAILED=1
check_prerequisite "kubectl" "Install kubectl: brew install kubectl" || PREREQ_FAILED=1

if [ $PREREQ_FAILED -eq 1 ]; then
    echo ""
    echo -e "${RED}Please install missing prerequisites and run again${NC}"
    exit 1
fi

echo ""

# Create kind cluster configuration
echo "Creating kind cluster configuration..."
cat > /tmp/kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ambient-platform-test
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
    protocol: TCP
  - containerPort: 30443
    hostPort: 8443
    protocol: TCP
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
networking:
  apiServerPort: 6443
EOF

# Check if cluster already exists
if kind get clusters | grep -q "ambient-platform-test"; then
    echo -e "${YELLOW}Cluster 'ambient-platform-test' already exists${NC}"
    read -p "Delete and recreate? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting existing cluster..."
        kind delete cluster --name ambient-platform-test
    else
        echo "Using existing cluster..."
    fi
fi

# Create kind cluster
if ! kind get clusters | grep -q "ambient-platform-test"; then
    echo -e "${BLUE}Creating kind cluster 'ambient-platform-test'...${NC}"
    kind create cluster --config /tmp/kind-config.yaml --wait 60s

    echo "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=60s
fi

# Set kubectl context
echo "Setting kubectl context..."
kubectl cluster-info --context kind-ambient-platform-test

echo ""
echo -e "${GREEN}✓ Kind cluster is ready!${NC}"
echo ""

# Create namespaces
echo -e "${BLUE}Creating namespaces...${NC}"
kubectl create namespace ambient-system --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace team-alpha --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace team-beta --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✓ Namespaces created${NC}"
echo ""

# Apply CRDs
echo -e "${BLUE}Applying CRDs...${NC}"
if [ -f "manifests/crds/session_v1alpha1.yaml" ]; then
    kubectl apply -f manifests/crds/session_v1alpha1.yaml
    echo -e "${GREEN}✓ Session CRD applied${NC}"
else
    echo -e "${YELLOW}⚠ Session CRD not found${NC}"
fi

if [ -f "manifests/crds/namespacepolicy_v1alpha1.yaml" ]; then
    kubectl apply -f manifests/crds/namespacepolicy_v1alpha1.yaml
    echo -e "${GREEN}✓ NamespacePolicy CRD applied${NC}"
else
    echo -e "${YELLOW}⚠ NamespacePolicy CRD not found${NC}"
fi
echo ""

# Create a sample NamespacePolicy
echo -e "${BLUE}Creating sample NamespacePolicy for team-alpha...${NC}"
cat <<EOF | kubectl apply -f -
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
      monthly: "100.00"
      currency: "USD"
  tools:
    allowed:
      - "bash"
      - "edit"
      - "read"
      - "write"
    blocked:
      - "exec"
  retention:
    sessions: "90d"
    artifacts: "30d"
    auditLogs: "7y"
  webhookAuth:
    apiKeys:
      github: "test-api-key-123"
      jira: "test-api-key-456"
EOF
echo -e "${GREEN}✓ NamespacePolicy created${NC}"
echo ""

# Create secrets for runners
echo -e "${BLUE}Creating runner secrets...${NC}"
kubectl create secret generic runner-secrets \
    --from-literal=anthropic-api-key=test-key-123 \
    --namespace=team-alpha \
    --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✓ Runner secrets created${NC}"
echo ""

# Deploy backend as NodePort service for testing
echo -e "${BLUE}Creating backend deployment for testing...${NC}"
cat <<EOF | kubectl apply -f -
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
---
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
        image: golang:1.21
        command: ["/bin/sh"]
        args: ["-c", "echo 'Backend placeholder - build and deploy actual image' && sleep 3600"]
        ports:
        - containerPort: 8080
EOF
echo -e "${YELLOW}Note: Backend deployment is a placeholder. Build and deploy the actual backend image.${NC}"
echo ""

# Create test Session
echo -e "${BLUE}Creating test Session...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: ambient.ai/v1alpha1
kind: Session
metadata:
  name: test-session-001
  namespace: team-alpha
spec:
  trigger:
    source: "manual"
    event: "test"
    payload:
      test: true
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
status:
  phase: "Pending"
EOF
echo -e "${GREEN}✓ Test session created${NC}"
echo ""

# Verification
echo -e "${BLUE}=== Cluster Status ===${NC}"
echo ""
echo "Nodes:"
kubectl get nodes
echo ""
echo "Namespaces:"
kubectl get namespaces | grep -E "(ambient-system|team-)"
echo ""
echo "CRDs:"
kubectl get crds | grep ambient.ai || echo "No Ambient CRDs found"
echo ""
echo "Sessions in team-alpha:"
kubectl get sessions -n team-alpha 2>/dev/null || echo "Session CRD not yet available"
echo ""
echo "NamespacePolicies in team-alpha:"
kubectl get namespacepolicies -n team-alpha 2>/dev/null || echo "NamespacePolicy CRD not yet available"
echo ""

# Instructions for next steps
echo -e "${BLUE}=== Next Steps ===${NC}"
echo ""
echo "1. Build the backend container image:"
echo "   cd backend"
echo "   # Using podman:"
echo "   podman build -t ambient-backend:test ."
echo "   podman save ambient-backend:test | kind load image-archive --name ambient-platform-test /dev/stdin"
echo "   # Or using docker:"
echo "   docker build -t ambient-backend:test ."
echo "   kind load docker-image ambient-backend:test --name ambient-platform-test"
echo ""
echo "2. Run the operator locally (for development):"
echo "   cd operator"
echo "   export KUBECONFIG=~/.kube/config"
echo "   go run main.go"
echo ""
echo "3. Test webhook endpoint (backend must be running):"
echo "   curl -X POST http://localhost:8080/api/v1/webhooks/github \\"
echo "     -H 'X-API-Key: test-api-key-123' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"action\":\"opened\",\"pull_request\":{\"id\":123}}'"
echo ""
echo "4. Watch session creation:"
echo "   kubectl get sessions -n team-alpha -w"
echo ""
echo "5. Check operator logs:"
echo "   kubectl logs -n ambient-system deployment/ambient-operator -f"
echo ""
echo "6. Clean up when done:"
echo "   kind delete cluster --name ambient-platform-test"
echo ""
echo -e "${GREEN}✓ Kind test environment setup complete!${NC}"
echo ""
echo "Cluster: ambient-platform-test"
echo "Context: kind-ambient-platform-test"
echo "Backend: http://localhost:8080"
echo "