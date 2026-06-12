#!/bin/bash
# Database Backup Utility - Test Runner Script

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🧪 Database Backup Utility - Test Suite"
echo "========================================"
echo ""

# Parse arguments
COVERAGE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [-c|--coverage] [-v|--verbose]"
            exit 1
            ;;
    esac
done

# Run backend tests
echo "🔧 Running backend tests..."
if [ "$COVERAGE" = true ]; then
    if [ "$VERBOSE" = true ]; then
        go test -v -cover -coverprofile=coverage.out ./...
    else
        go test -cover -coverprofile=coverage.out ./...
    fi
    echo ""
    echo "📊 Coverage report:"
    go tool cover -func=coverage.out | tail -1
else
    if [ "$VERBOSE" = true ]; then
        go test -v ./...
    else
        go test ./...
    fi
fi

BACKEND_EXIT=$?

# Run frontend tests
echo ""
echo "🎨 Running frontend tests..."
cd frontend
npm test -- --run
FRONTEND_EXIT=$?
cd ..

# Summary
echo ""
echo "=========================================="
if [ $BACKEND_EXIT -eq 0 ] && [ $FRONTEND_EXIT -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    [ $BACKEND_EXIT -ne 0 ] && echo "  - Backend tests failed"
    [ $FRONTEND_EXIT -ne 0 ] && echo "  - Frontend tests failed"
    exit 1
fi
