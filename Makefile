# DB-Backup-Main - Monorepo Makefile
# Orchestrates builds, tests, and operations across all sub-projects

.PHONY: all build test clean deps help \
        backend-build backend-test backend-lint backend-deps \
        web-build web-test web-lint web-deps \
        shared-build shared-deps \
        desktop-build desktop-test desktop-deps \
        mobile-test mobile-deps \
        docker-up docker-down

# Default
.DEFAULT_GOAL := help

## help: Show available targets
help:
	@echo "DB-Backup-Main - Available targets:"
	@echo ""
	@grep -E '^## .*:' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""

# =============================================================================
# Top-level targets
# =============================================================================

## all: Build and test everything
all: deps build test

## deps: Install all dependencies
deps: backend-deps shared-deps web-deps mobile-deps desktop-deps

## build: Build all projects
build: backend-build shared-build web-build

## test: Run all tests
test: backend-test web-test mobile-test

## clean: Clean all build artifacts
clean: backend-clean web-clean desktop-clean
	@echo "All clean."

## lint: Lint all projects
lint: backend-lint web-lint

# =============================================================================
# Backend (Go)
# =============================================================================

## backend-deps: Install Go dependencies
backend-deps:
	cd backend && go mod download && go mod tidy

## backend-build: Build all Go binaries
backend-build:
	cd backend && make build

## backend-test: Run Go tests
backend-test:
	cd backend && go test -short -timeout 300s ./...

## backend-lint: Lint Go code
backend-lint:
	cd backend && go vet ./... && go fmt ./...

## backend-clean: Clean Go build artifacts
backend-clean:
	cd backend && make clean

## backend-run: Run the API server
backend-run:
	cd backend && go run cmd/server/main.go

# =============================================================================
# Shared (TypeScript packages)
# =============================================================================

## shared-deps: Install shared package dependencies
shared-deps:
	cd shared && npm ci --ignore-scripts 2>/dev/null || cd shared && npm install

## shared-build: Build shared packages
shared-build:
	cd shared && npm run build --if-present

# =============================================================================
# Web (Next.js)
# =============================================================================

## web-deps: Install web dependencies
web-deps: shared-build
	cd web && npm ci --ignore-scripts 2>/dev/null || cd web && npm install

## web-build: Build Next.js frontend
web-build: web-deps
	cd web && npm run build

## web-test: Run web tests
web-test:
	cd web && npm test -- --run

## web-lint: Lint web code
web-lint:
	cd web && npm run lint

## web-clean: Clean web build artifacts
web-clean:
	rm -rf web/.next

## web-dev: Start web dev server
web-dev:
	cd web && npm run dev

# =============================================================================
# Mobile (React Native)
# =============================================================================

## mobile-deps: Install mobile dependencies
mobile-deps: shared-build
	cd mobile && npm ci --ignore-scripts 2>/dev/null || cd mobile && npm install

## mobile-test: Run mobile tests
mobile-test:
	cd mobile && npm test -- --passWithNoTests

# =============================================================================
# Desktop (Tauri)
# =============================================================================

## desktop-deps: Install desktop dependencies
desktop-deps: shared-build
	cd desktop && npm ci --ignore-scripts 2>/dev/null || cd desktop && npm install

## desktop-build: Build desktop frontend
desktop-build: desktop-deps
	cd desktop && npm run build

## desktop-test: Run desktop tests
desktop-test:
	cd desktop && npm test -- --run

## desktop-clean: Clean desktop build artifacts
desktop-clean:
	rm -rf desktop/dist

# =============================================================================
# Docker / Infrastructure
# =============================================================================

## docker-up: Start development infrastructure
docker-up:
	cd backend && docker-compose up -d

## docker-down: Stop development infrastructure
docker-down:
	cd backend && docker-compose down
