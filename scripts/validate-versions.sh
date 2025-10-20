#!/bin/bash
# Version validation script to prevent misconfiguration
# Validates that all Go and Docker versions across the project are consistent and real

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0

echo "=== Version Validation ==="
echo ""

# Extract versions from files
GO_MOD_VERSION=$(grep '^go ' go.mod | awk '{print $2}')
DOCKERFILE_GO_IMAGE=$(grep '^FROM golang:' Dockerfile | head -1 | awk '{print $2}' | cut -d' ' -f1)
DOCKERFILE_ALPINE_IMAGE=$(grep '^FROM alpine:' Dockerfile | awk '{print $2}')
CI_GO_VERSIONS=$(grep "go-version: \[" .github/workflows/ci.yml | sed "s/.*\[\(.*\)\].*/\1/" | tr -d "'" | tr -d ' ')

echo "📋 Detected Versions:"
echo "  go.mod:              go ${GO_MOD_VERSION}"
echo "  Dockerfile (Go):     ${DOCKERFILE_GO_IMAGE}"
echo "  Dockerfile (Alpine): ${DOCKERFILE_ALPINE_IMAGE}"
echo "  CI workflow:         [${CI_GO_VERSIONS}]"
echo ""

# Check 1: go.mod version format
echo "✓ Checking go.mod version format..."
if ! echo "$GO_MOD_VERSION" | grep -qE '^[0-9]+\.[0-9]+(\.[0-9]+)?$'; then
    echo -e "${RED}✗ FAIL: Invalid go.mod version format: $GO_MOD_VERSION${NC}"
    echo "  Expected format: X.Y or X.Y.Z (e.g., 1.23 or 1.24.0)"
    ((ERRORS++))
else
    echo -e "${GREEN}  Valid format: $GO_MOD_VERSION${NC}"
fi

# Check 2: Dockerfile Go image tag format
echo "✓ Checking Dockerfile Go image tag..."
GO_IMAGE_TAG=$(echo "$DOCKERFILE_GO_IMAGE" | cut -d: -f2)
if ! echo "$GO_IMAGE_TAG" | grep -qE '^[0-9]+\.[0-9]+'; then
    echo -e "${RED}✗ FAIL: Invalid Dockerfile Go image tag: $GO_IMAGE_TAG${NC}"
    echo "  Expected format: golang:X.Y-alpine or golang:X.Y.Z-alpine"
    ((ERRORS++))
else
    echo -e "${GREEN}  Valid format: $GO_IMAGE_TAG${NC}"
fi

# Check 3: Dockerfile Alpine image tag format
echo "✓ Checking Dockerfile Alpine image tag..."
ALPINE_TAG=$(echo "$DOCKERFILE_ALPINE_IMAGE" | cut -d: -f2)
if ! echo "$ALPINE_TAG" | grep -qE '^[0-9]+\.[0-9]+'; then
    echo -e "${RED}✗ FAIL: Invalid Dockerfile Alpine tag: $ALPINE_TAG${NC}"
    echo "  Expected format: alpine:X.Y"
    ((ERRORS++))
else
    echo -e "${GREEN}  Valid format: $ALPINE_TAG${NC}"
fi

# Check 4: No toolchain directive in go.mod (causes issues)
echo "✓ Checking for toolchain directive..."
if grep -q '^toolchain ' go.mod; then
    TOOLCHAIN_VERSION=$(grep '^toolchain ' go.mod | awk '{print $2}')
    echo -e "${YELLOW}⚠ WARNING: Toolchain directive found: $TOOLCHAIN_VERSION${NC}"
    echo "  This can cause build issues. Consider removing it."
    echo "  The Go toolchain will auto-select based on 'go' version."
fi

# Check 5: CI includes go.mod version
echo "✓ Checking CI includes go.mod version..."
GO_MAJOR_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1-2)
if ! echo "$CI_GO_VERSIONS" | grep -q "$GO_MAJOR_MINOR"; then
    echo -e "${RED}✗ FAIL: CI workflow does not test go.mod version${NC}"
    echo "  go.mod specifies: $GO_MAJOR_MINOR"
    echo "  CI tests: [$CI_GO_VERSIONS]"
    echo "  Add '$GO_MAJOR_MINOR' to CI workflow matrix"
    ((ERRORS++))
else
    echo -e "${GREEN}  CI includes $GO_MAJOR_MINOR${NC}"
fi

# Check 6: Verify Docker images exist (optional, requires docker)
if command -v docker &> /dev/null; then
    echo "✓ Checking if Docker images exist..."

    # Try to pull/verify Go image
    if docker pull "$DOCKERFILE_GO_IMAGE" &>/dev/null; then
        echo -e "${GREEN}  ✓ $DOCKERFILE_GO_IMAGE exists${NC}"
    else
        echo -e "${RED}✗ FAIL: Docker image not found: $DOCKERFILE_GO_IMAGE${NC}"
        echo "  This image doesn't exist on Docker Hub"
        echo "  Use a valid golang:<version>-alpine tag"
        ((ERRORS++))
    fi

    # Try to pull/verify Alpine image
    if docker pull "$DOCKERFILE_ALPINE_IMAGE" &>/dev/null; then
        echo -e "${GREEN}  ✓ $DOCKERFILE_ALPINE_IMAGE exists${NC}"
    else
        echo -e "${RED}✗ FAIL: Docker image not found: $DOCKERFILE_ALPINE_IMAGE${NC}"
        echo "  This image doesn't exist on Docker Hub"
        echo "  Use a valid alpine:<version> tag"
        ((ERRORS++))
    fi
else
    echo -e "${YELLOW}⚠ Skipping Docker image existence check (docker not available)${NC}"
fi

# Check 7: Verify go version is reasonable
echo "✓ Checking go.mod version is reasonable..."
GO_MAJOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f2)

if [ "$GO_MAJOR" -ne 1 ]; then
    echo -e "${RED}✗ FAIL: Unexpected Go major version: $GO_MAJOR${NC}"
    echo "  Go is currently on version 1.x"
    ((ERRORS++))
elif [ "$GO_MINOR" -gt 30 ]; then
    echo -e "${YELLOW}⚠ WARNING: Go minor version seems high: 1.$GO_MINOR${NC}"
    echo "  Latest stable Go versions are typically 1.20-1.25"
    echo "  Verify this version exists: https://go.dev/dl/"
fi

# Check 8: README documentation consistency
echo "✓ Checking README documentation..."
README_GO_VERSION=$(grep -o 'golang:[0-9.]*-alpine' README.md | head -1 | cut -d: -f2 | cut -d- -f1)
README_ALPINE_VERSION=$(grep -o 'alpine:[0-9.]*' README.md | head -1 | cut -d: -f2)

if [ -n "$README_GO_VERSION" ] && [ "$README_GO_VERSION" != "$GO_IMAGE_TAG" ]; then
    if ! echo "$GO_IMAGE_TAG" | grep -q "$README_GO_VERSION"; then
        echo -e "${YELLOW}⚠ WARNING: README mentions different Go version${NC}"
        echo "  README: golang:$README_GO_VERSION"
        echo "  Dockerfile: $GO_IMAGE_TAG"
    fi
fi

echo ""
echo "=== Summary ==="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ All version checks passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Found $ERRORS error(s)${NC}"
    echo ""
    echo "Version Management Guidelines:"
    echo "  1. Use released Go versions: https://go.dev/dl/"
    echo "  2. Use available Docker tags: https://hub.docker.com/_/golang/tags"
    echo "  3. Use available Alpine tags: https://hub.docker.com/_/alpine/tags"
    echo "  4. Keep CI matrix in sync with go.mod"
    echo "  5. Run 'make validate-versions' before committing"
    exit 1
fi
