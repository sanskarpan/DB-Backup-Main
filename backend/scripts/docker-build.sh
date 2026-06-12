#!/bin/bash
# Database Backup Utility - Docker Build Script

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🐳 Database Backup Utility - Docker Build"
echo "=========================================="
echo ""

# Parse arguments
VERSION=${1:-latest}
REGISTRY=${2:-""}

echo "Building version: $VERSION"
if [ -n "$REGISTRY" ]; then
    echo "Registry: $REGISTRY"
fi
echo ""

# Build backend image
echo "🔧 Building backend Docker image..."
if [ -n "$REGISTRY" ]; then
    docker build -t ${REGISTRY}/db-backup:${VERSION} .
else
    docker build -t db-backup:${VERSION} .
fi

echo ""
echo -e "${GREEN}✅ Docker image built successfully!${NC}"
echo ""

# Show image info
echo "📦 Image information:"
if [ -n "$REGISTRY" ]; then
    docker images ${REGISTRY}/db-backup:${VERSION}
else
    docker images db-backup:${VERSION}
fi

echo ""
echo "To run the image:"
if [ -n "$REGISTRY" ]; then
    echo "  docker run -p 8080:8080 --env-file .env ${REGISTRY}/db-backup:${VERSION}"
else
    echo "  docker run -p 8080:8080 --env-file .env db-backup:${VERSION}"
fi

echo ""
echo "To push to registry:"
if [ -n "$REGISTRY" ]; then
    echo "  docker push ${REGISTRY}/db-backup:${VERSION}"
else
    echo "  Please specify a registry:"
    echo "  ./scripts/docker-build.sh ${VERSION} your-registry.com"
fi
