#!/bin/bash

# Validation script to check if the test environment can be set up
# Run this before the full test suite

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Multi-Tenant Platform Test Environment Validation ===${NC}"
echo ""

VALIDATION_FAILED=0

# Check prerequisites
check_command() {
    local cmd=$1
    local install_msg=$2

    if command -v $cmd &> /dev/null; then
        VERSION=$($cmd version --short 2>/dev/null || $cmd --version 2>/dev/null | head -1 || echo "unknown")
        echo -e "${GREEN}✓ $cmd${NC} ($VERSION)"
        return 0
    else
        echo -e "${RED}✗ $cmd is not installed${NC}"
        echo "  Install: $install_msg"
        VALIDATION_FAILED=1
        return 1
    fi
}

echo "=== Prerequisites Check ==="
# Check for container runtime - prefer podman over docker
if command -v podman &> /dev/null; then
    VERSION=$(podman version --format "{{.Client.Version}}" 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✓ podman${NC} ($VERSION)"
    export CONTAINER_RUNTIME="podman"
elif command -v docker &> /dev/null; then
    VERSION=$(docker version --format '{{.Client.Version}}' 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✓ docker${NC} ($VERSION)"
    export CONTAINER_RUNTIME="docker"
else
    echo -e "${RED}✗ Neither podman nor docker is installed${NC}"
    echo "  Install: brew install podman"
    VALIDATION_FAILED=1
fi

check_command "kind" "brew install kind"
check_command "kubectl" "brew install kubectl"
check_command "go" "brew install go"

echo ""

# Check if Go modules are ready
echo "=== Go Modules Check ==="
cd /Users/gkrumbac/Documents/vTeam/components

if [ -f "backend/go.mod" ]; then
    echo -e "${GREEN}✓ Backend go.mod exists${NC}"
    cd backend
    if go mod verify > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Backend dependencies verified${NC}"
    else
        echo -e "${YELLOW}⚠ Running go mod tidy for backend...${NC}"
        go mod tidy
    fi
    cd ..
else
    echo -e "${RED}✗ Backend go.mod not found${NC}"
    VALIDATION_FAILED=1
fi

if [ -f "operator/go.mod" ]; then
    echo -e "${GREEN}✓ Operator go.mod exists${NC}"
    cd operator
    if go mod verify > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Operator dependencies verified${NC}"
    else
        echo -e "${YELLOW}⚠ Running go mod tidy for operator...${NC}"
        go mod tidy
    fi
    cd ..
else
    echo -e "${RED}✗ Operator go.mod not found${NC}"
    VALIDATION_FAILED=1
fi

echo ""

# Check if CRDs exist
echo "=== CRDs Check ==="
if [ -f "manifests/crds/session_v1alpha1.yaml" ]; then
    echo -e "${GREEN}✓ Session CRD definition exists${NC}"
else
    echo -e "${RED}✗ Session CRD definition not found${NC}"
    echo "  Expected: manifests/crds/session_v1alpha1.yaml"
    VALIDATION_FAILED=1
fi

if [ -f "manifests/crds/namespacepolicy_v1alpha1.yaml" ]; then
    echo -e "${GREEN}✓ NamespacePolicy CRD definition exists${NC}"
else
    echo -e "${RED}✗ NamespacePolicy CRD definition not found${NC}"
    echo "  Expected: manifests/crds/namespacepolicy_v1alpha1.yaml"
    VALIDATION_FAILED=1
fi

echo ""

# Check if backend builds
echo "=== Backend Build Check ==="
cd backend
if go build -o /tmp/test-backend . > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Backend builds successfully${NC}"
    rm -f /tmp/test-backend
else
    echo -e "${RED}✗ Backend build failed${NC}"
    echo "  Try: cd backend && go mod tidy && go build"
    VALIDATION_FAILED=1
fi
cd ..

# Check if operator builds
echo "=== Operator Build Check ==="
cd operator
if go build -o /tmp/test-operator . > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Operator builds successfully${NC}"
    rm -f /tmp/test-operator
else
    echo -e "${RED}✗ Operator build failed${NC}"
    echo "  Try: cd operator && go mod tidy && go build"
    VALIDATION_FAILED=1
fi
cd ..

echo ""

# Check container runtime daemon
echo "=== Container Runtime Check ==="
if [ "${CONTAINER_RUNTIME}" = "podman" ]; then
    if podman info > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Podman is running${NC}"
    else
        echo -e "${RED}✗ Podman is not running${NC}"
        echo "  Start podman machine: podman machine start"
        VALIDATION_FAILED=1
    fi
elif [ "${CONTAINER_RUNTIME}" = "docker" ]; then
    if docker info > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Docker daemon is running${NC}"
    else
        echo -e "${RED}✗ Docker daemon is not running${NC}"
        echo "  Start Docker Desktop or docker service"
        VALIDATION_FAILED=1
    fi
fi

echo ""

# Check existing kind clusters
echo "=== Kind Clusters Check ==="
if kind get clusters 2>/dev/null | grep -q "ambient-platform-test"; then
    echo -e "${YELLOW}⚠ Kind cluster 'ambient-platform-test' already exists${NC}"
    echo "  The test script will ask if you want to recreate it"
else
    echo -e "${GREEN}✓ No conflicting kind clusters found${NC}"
fi

echo ""

# Test basic unit tests
echo "=== Unit Tests Check ==="
cd backend
if go test ./tests/contract/ -run TestGitHubWebhook > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Basic unit tests pass${NC}"
else
    echo -e "${RED}✗ Basic unit tests fail${NC}"
    echo "  Check test configuration"
    VALIDATION_FAILED=1
fi
cd ..

echo ""

# Summary
echo "=== Validation Summary ==="
if [ $VALIDATION_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All validations passed! Ready to run tests.${NC}"
    echo ""
    echo "Run the full test suite:"
    echo "  ./run-kind-tests.sh"
    echo ""
    echo "Or set up the environment manually:"
    echo "  ./setup-kind-test.sh"
else
    echo -e "${RED}❌ Some validations failed. Please fix the issues above.${NC}"
    echo ""
    echo "Common fixes:"
    echo "  1. Install missing tools: brew install docker kind kubectl go"
    echo "  2. Start Docker Desktop"
    echo "  3. Run go mod tidy in backend/ and operator/ directories"
    echo "  4. Check that all required files exist"
fi

echo ""
echo "=== File Structure Check ==="
echo "Current working directory: $(pwd)"
echo ""
echo "Required files:"
echo "- backend/main.go: $([ -f backend/main.go ] && echo "✓" || echo "✗")"
echo "- backend/Dockerfile: $([ -f backend/Dockerfile ] && echo "✓" || echo "✗")"
echo "- operator/main.go: $([ -f operator/main.go ] && echo "✓" || echo "✗")"
echo "- manifests/crds/: $([ -d manifests/crds ] && echo "✓" || echo "✗")"
echo "- setup-kind-test.sh: $([ -f setup-kind-test.sh ] && echo "✓" || echo "✗")"
echo "- run-kind-tests.sh: $([ -f run-kind-tests.sh ] && echo "✓" || echo "✗")"

exit $VALIDATION_FAILED