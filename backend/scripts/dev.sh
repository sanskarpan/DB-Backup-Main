#!/bin/bash
# Database Backup Utility - Development Server Script

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔧 Database Backup Utility - Development Mode"
echo "=============================================="
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env file not found. Running setup first...${NC}"
    ./scripts/setup.sh
fi

# Start backend in development mode
echo "🚀 Starting backend server..."
go run ./cmd/server &
BACKEND_PID=$!

# Wait a bit for backend to start
sleep 2

# Start frontend in development mode
echo "🎨 Starting frontend development server..."
cd frontend && npm run dev &
FRONTEND_PID=$!

echo ""
echo -e "${GREEN}✅ Development servers started!${NC}"
echo ""
echo "  Backend:  http://localhost:8080"
echo "  Frontend: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop all servers"

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Stopping servers..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    echo "✅ Servers stopped"
    exit 0
}

# Trap Ctrl+C
trap cleanup INT

# Wait for processes
wait
