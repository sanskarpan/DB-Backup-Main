#!/bin/bash
# Database Backup Utility - Initial Setup Script

set -e

echo "🚀 Database Backup Utility - Setup Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env exists
if [ -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file already exists. Backing up to .env.backup${NC}"
    cp .env .env.backup
fi

# Copy .env.example to .env
echo "📝 Creating .env file from .env.example..."
cp .env.example .env

# Generate JWT secret
echo "🔐 Generating secure JWT secret..."
JWT_SECRET=$(openssl rand -base64 48)

# Update .env with generated secret
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s|DBBACKUP_SECURITY_JWT_SECRET=.*|DBBACKUP_SECURITY_JWT_SECRET=$JWT_SECRET|" .env
else
    # Linux
    sed -i "s|DBBACKUP_SECURITY_JWT_SECRET=.*|DBBACKUP_SECURITY_JWT_SECRET=$JWT_SECRET|" .env
fi

echo -e "${GREEN}✅ JWT secret generated and added to .env${NC}"

# Create necessary directories
echo "📁 Creating necessary directories..."
mkdir -p backups
mkdir -p logs
mkdir -p /tmp/backups

echo -e "${GREEN}✅ Directories created${NC}"

# Check for required tools
echo ""
echo "🔍 Checking for required tools..."

check_tool() {
    if command -v $1 &> /dev/null; then
        echo -e "${GREEN}✓${NC} $1 is installed"
        return 0
    else
        echo -e "${RED}✗${NC} $1 is not installed"
        return 1
    fi
}

MISSING_TOOLS=0

# Check Go
if ! check_tool go; then
    echo "  Install from: https://golang.org/dl/"
    MISSING_TOOLS=1
fi

# Check Node.js (for frontend)
if ! check_tool node; then
    echo "  Install from: https://nodejs.org/"
    MISSING_TOOLS=1
fi

# Check database tools (optional)
echo ""
echo "Optional database tools (install as needed):"
check_tool mysqldump || echo "  MySQL backups will not work without mysqldump"
check_tool pg_dump || echo "  PostgreSQL backups will not work without pg_dump"
check_tool mongodump || echo "  MongoDB backups will not work without mongodump"

if [ $MISSING_TOOLS -eq 1 ]; then
    echo ""
    echo -e "${YELLOW}⚠️  Some required tools are missing. Please install them before continuing.${NC}"
    exit 1
fi

echo ""
echo "📦 Installing Go dependencies..."
go mod download
go mod tidy

echo ""
echo "🏗️  Building the application..."
make build || go build -o bin/db-backup-server ./cmd/server

echo ""
echo "📦 Installing frontend dependencies..."
cd frontend && npm install && cd ..

echo ""
echo -e "${GREEN}✅ Setup complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Review and update .env file with your configuration"
echo "  2. Start the server: ./bin/db-backup-server"
echo "  3. Start the frontend: cd frontend && npm run dev"
echo ""
echo "For production deployment, see README.md"
