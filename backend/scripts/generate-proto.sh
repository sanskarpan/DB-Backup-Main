#!/bin/bash

# Script to generate Go code from Protocol Buffer definitions
# This script generates gRPC service code, gRPC gateway code, and OpenAPI v2 specs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Ensure protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}ERROR: protoc not found. Install it with: brew install protobuf${NC}"
    exit 1
fi

# Ensure Go protoc plugins are installed
if [ ! -f "$HOME/go/bin/protoc-gen-go" ]; then
    echo -e "${YELLOW}Installing protoc-gen-go...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if [ ! -f "$HOME/go/bin/protoc-gen-go-grpc" ]; then
    echo -e "${YELLOW}Installing protoc-gen-go-grpc...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

if [ ! -f "$HOME/go/bin/protoc-gen-grpc-gateway" ]; then
    echo -e "${YELLOW}Installing protoc-gen-grpc-gateway...${NC}"
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
fi

if [ ! -f "$HOME/go/bin/protoc-gen-openapiv2" ]; then
    echo -e "${YELLOW}Installing protoc-gen-openapiv2...${NC}"
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
fi

# Add Go bin to PATH
export PATH="$PATH:$HOME/go/bin"

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

# Create output directories
mkdir -p api/proto/gen/go/dbbackup/v1
mkdir -p api/proto/gen/openapiv2

echo -e "${GREEN}Generating Go code from Protocol Buffers...${NC}"

# Generate Go code for all proto files
protoc \
  --go_out=api/proto/gen/go \
  --go_opt=paths=source_relative \
  --go-grpc_out=api/proto/gen/go \
  --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=api/proto/gen/go \
  --grpc-gateway_opt=paths=source_relative \
  --grpc-gateway_opt=generate_unbound_methods=true \
  --openapiv2_out=api/proto/gen/openapiv2 \
  --openapiv2_opt=allow_merge=true \
  --openapiv2_opt=merge_file_name=dbbackup \
  -I. \
  -Iapi/proto \
  api/proto/common.proto \
  api/proto/backup_service.proto \
  api/proto/restore_service.proto \
  api/proto/monitoring_service.proto

echo -e "${GREEN}✓ Protocol Buffer code generation complete!${NC}"
echo ""
echo "Generated files:"
echo "  - Go service code: api/proto/gen/go/api/proto/*.pb.go"
echo "  - gRPC service code: api/proto/gen/go/api/proto/*_grpc.pb.go"
echo "  - gRPC gateway code: api/proto/gen/go/api/proto/*.pb.gw.go"
echo "  - OpenAPI v2 spec: api/proto/gen/openapiv2/dbbackup.swagger.json"
