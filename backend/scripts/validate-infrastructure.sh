#!/bin/bash

# Infrastructure Validation Script for DB Backup System
# This script validates that all components are properly configured and running

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Helper functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

check_command() {
    if command -v $1 &> /dev/null; then
        print_success "$1 is installed"
        return 0
    else
        print_error "$1 is not installed"
        return 1
    fi
}

check_port() {
    if nc -z localhost $1 2>/dev/null; then
        print_success "Port $1 is open ($2)"
        return 0
    else
        print_error "Port $1 is not accessible ($2)"
        return 1
    fi
}

# Main validation
print_header "System Requirements"
check_command "go"
check_command "docker"
check_command "kubectl"
check_command "helm"
check_command "node"
check_command "npm"

print_header "Go Environment"
if [ -n "$(go version 2>/dev/null)" ]; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go version: $GO_VERSION"
else
    print_error "Go not found"
fi

print_header "Docker Status"
if docker info &> /dev/null; then
    print_success "Docker daemon is running"

    # Check for required images
    if docker images | grep -q "postgres"; then
        print_success "PostgreSQL image available"
    else
        print_warning "PostgreSQL image not found"
    fi

    if docker images | grep -q "mongo"; then
        print_success "MongoDB image available"
    else
        print_warning "MongoDB image not found"
    fi

    if docker images | grep -q "redis"; then
        print_success "Redis image available"
    else
        print_warning "Redis image not found"
    fi
else
    print_error "Docker daemon is not running"
fi

print_header "Database Connectivity"
check_port 5432 "PostgreSQL"
check_port 27017 "MongoDB"
check_port 6379 "Redis"
check_port 3306 "MySQL"
check_port 9042 "Cassandra"
check_port 9200 "Elasticsearch"

print_header "Storage Providers"
check_port 9000 "MinIO"

# Check if storage directories exist
if [ -d "/tmp/db-backups" ]; then
    print_success "Local backup directory exists"
else
    print_warning "Local backup directory not found"
    mkdir -p /tmp/db-backups && print_success "Created local backup directory"
fi

print_header "Kubernetes"
if kubectl cluster-info &> /dev/null; then
    print_success "Kubernetes cluster is accessible"

    # Check for operator
    if kubectl get namespace db-backup-system &> /dev/null; then
        print_success "db-backup-system namespace exists"
    else
        print_warning "db-backup-system namespace not found"
    fi

    # Check for CRDs
    if kubectl get crd backuppolicies.backup.db.sanskarpan.com &> /dev/null; then
        print_success "BackupPolicy CRD is installed"
    else
        print_warning "BackupPolicy CRD not found"
    fi
else
    print_warning "Kubernetes cluster not accessible"
fi

print_header "Frontend Dependencies"
if [ -f "frontend/package.json" ]; then
    print_success "Frontend package.json found"

    if [ -d "frontend/node_modules" ]; then
        print_success "Frontend dependencies installed"
    else
        print_warning "Frontend dependencies not installed (run: npm install)"
    fi
else
    print_error "Frontend package.json not found"
fi

print_header "Backend Build"
if [ -f "go.mod" ]; then
    print_success "go.mod found"

    # Try to build
    if go build -o /tmp/db-backup-test ./cmd/server &> /dev/null; then
        print_success "Backend builds successfully"
        rm -f /tmp/db-backup-test
    else
        print_error "Backend build failed"
    fi
else
    print_error "go.mod not found"
fi

print_header "Configuration Files"
CONFIG_FILES=(
    "config/config.yaml"
    "config/databases.yaml"
    "config/storage.yaml"
    ".env.example"
)

for file in "${CONFIG_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_warning "$file not found"
    fi
done

print_header "Test Files"
TEST_COUNT=$(find . -name "*_test.go" | wc -l)
print_success "Found $TEST_COUNT test files"

# Run quick tests
if go test -short ./... &> /dev/null; then
    print_success "Unit tests pass"
else
    print_error "Unit tests failed"
fi

print_header "Documentation"
DOC_FILES=(
    "README.md"
    "ARCHITECTURE.md"
    "API_DOCUMENTATION.md"
    "DEPLOYMENT.md"
)

for file in "${DOC_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_warning "$file not found"
    fi
done

# Summary
print_header "Validation Summary"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All critical checks passed!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some critical checks failed. Please review and fix.${NC}"
    exit 1
fi
