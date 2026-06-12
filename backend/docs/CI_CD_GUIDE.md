# CI/CD Pipeline Guide

## Overview

This document describes the complete CI/CD pipeline for the Database Backup Utility.

## Table of Contents

- [Pipeline Architecture](#pipeline-architecture)
- [GitHub Actions Workflows](#github-actions-workflows)
- [GitLab CI Pipeline](#gitlab-ci-pipeline)
- [Configuration](#configuration)
- [Secrets Management](#secrets-management)
- [Deployment Strategies](#deployment-strategies)
- [Monitoring and Alerts](#monitoring-and-alerts)
- [Troubleshooting](#troubleshooting)

## Pipeline Architecture

```
┌──────────────┐
│  Code Push   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Lint & Format                 │
│  • golangci-lint                         │
│  • gofmt, goimports                      │
│  • go mod tidy check                     │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│              Test Suite                  │
│  • Unit tests (multiple OS/Go versions)  │
│  • Integration tests (with databases)    │
│  • E2E tests                             │
│  • Benchmark tests                       │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│          Security Scanning               │
│  • CodeQL (SAST)                         │
│  • Gosec (Go security)                   │
│  • Trivy (vulnerabilities)               │
│  • Snyk (dependencies)                   │
│  • Secret scanning (TruffleHog/Gitleaks) │
│  • License compliance                    │
│  • SBOM generation                       │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│         Build & Package                  │
│  • Build binaries (multi-platform)       │
│  • Build Docker images (multi-arch)      │
│  • Package Helm chart                    │
│  • Sign artifacts (Cosign)               │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Deployment                    │
│  • Development (auto on develop)         │
│  • Staging (manual on main)              │
│  • Production (manual on tags)           │
│  • Canary deployment (production)        │
└──────┬───────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│             Release                      │
│  • Create GitHub/GitLab release          │
│  • Publish Docker images                 │
│  • Publish Helm chart                    │
│  • Generate changelog                    │
│  • Notify stakeholders                   │
└──────────────────────────────────────────┘
```

## GitHub Actions Workflows

### 1. Main CI/CD Pipeline (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main`, `develop`
- Pull requests to `main`, `develop`
- Tags matching `v*`

**Jobs:**
- `lint-and-test`: Code quality and unit tests
- `build`: Multi-platform binary builds
- `docker`: Docker image builds and push
- `security`: Container security scanning
- `release`: Create GitHub release (tags only)
- `deploy`: Deploy to Kubernetes (main branch)

### 2. Test Suite (`.github/workflows/test.yml`)

**Triggers:**
- Push to any branch
- Pull requests
- Daily schedule (2 AM UTC)

**Jobs:**
- `unit-tests`: Cross-platform testing (Ubuntu, macOS, Windows)
- `integration-tests`: Testing with live databases
- `e2e-tests`: End-to-end testing with Docker Compose
- `benchmark-tests`: Performance benchmarking

### 3. Code Quality (`.github/workflows/code-quality.yml`)

**Jobs:**
- `golangci-lint`: Comprehensive Go linting
- `gosec`: Security-focused static analysis
- `staticcheck`: Advanced Go linter
- `govulncheck`: Vulnerability checking
- `sonarcloud`: Code quality and coverage
- `codeclimate`: Test coverage reporting
- `dependency-review`: Dependency vulnerability scanning
- `go-mod-tidy`: Module tidiness check
- `format-check`: Code formatting verification
- `spell-check`: Documentation spell checking

### 4. Security Scanning (`.github/workflows/security.yml`)

**Jobs:**
- `codeql`: GitHub's code scanning
- `gosec`: Go security checker
- `trivy-fs`: Filesystem vulnerability scan
- `trivy-config`: Configuration security scan
- `snyk`: Dependency vulnerability scan
- `nancy`: Go dependency checker
- `osv-scanner`: Open Source Vulnerabilities scanner
- `semgrep`: Pattern-based security scanning
- `docker-scout`: Container image security
- `secrets-scan`: Secret detection (TruffleHog + Gitleaks)
- `license-check`: License compliance
- `sbom-generation`: Software Bill of Materials

### 5. Deployment (`.github/workflows/deploy.yml`)

**Manual Workflow:**
- Environment selection (development, staging, production)
- Version/tag selection
- Helm-based deployment
- Smoke testing
- Canary deployment (production only)
- Rollback capability
- Slack notifications

### 6. Release (`.github/workflows/release.yml`)

**Triggered on tags** (v*)

**Jobs:**
- `create-release`: Create GitHub release with changelog
- `build-binaries`: Build release binaries for all platforms
- `build-docker`: Build and push multi-arch Docker images
- `publish-helm`: Package and publish Helm chart
- `update-documentation`: Update docs with new version
- `notify-release`: Slack notification

## GitLab CI Pipeline

### Stages

1. **lint**: Code quality checks
2. **test**: Unit, integration, E2E, and benchmark tests
3. **security**: Security and vulnerability scanning
4. **build**: Binary and Docker image builds
5. **deploy**: Environment deployments
6. **release**: Release creation and artifact publishing

### Key Features

- Matrix builds for multi-platform binaries
- Service containers for integration tests
- Parallel job execution
- Docker layer caching
- Helm-based Kubernetes deployments
- Manual approval for production
- Comprehensive artifact management

## Configuration

### Required Secrets (GitHub Actions)

```bash
# Container Registry
GITHUB_TOKEN              # Automatic (provided by GitHub)

# Code Quality
CODECOV_TOKEN             # Codecov upload token
SONAR_TOKEN               # SonarCloud authentication
CC_TEST_REPORTER_ID       # CodeClimate reporter ID

# Security
SNYK_TOKEN                # Snyk API token

# Kubernetes Deployments
KUBECONFIG_DEVELOPMENT    # Base64 encoded kubeconfig
KUBECONFIG_STAGING        # Base64 encoded kubeconfig
KUBECONFIG_PRODUCTION     # Base64 encoded kubeconfig

# Notifications
SLACK_WEBHOOK_URL         # Slack webhook for notifications
```

### Required Variables (GitLab CI)

```bash
# Container Registry
CI_REGISTRY               # GitLab container registry
CI_REGISTRY_USER          # Registry username
CI_REGISTRY_PASSWORD      # Registry password

# Kubernetes
KUBECONFIG_DEVELOPMENT    # Base64 encoded kubeconfig
KUBECONFIG_STAGING        # Base64 encoded kubeconfig
KUBECONFIG_PRODUCTION     # Base64 encoded kubeconfig

# Helm
HELM_VERSION              # Helm version to use
```

### Configuration Files

| File | Purpose |
|------|---------|
| `.golangci.yml` | golangci-lint configuration |
| `.cspell.json` | Spell checker configuration |
| `.github/dependabot.yml` | Dependency updates |
| `sonar-project.properties` | SonarCloud configuration |

## Secrets Management

### GitHub Actions Secrets

Set secrets via:
```bash
gh secret set SECRET_NAME --body "secret-value"
```

Or via GitHub UI:
Settings → Secrets and variables → Actions → New repository secret

### GitLab CI Variables

Set variables via:
- GitLab UI: Settings → CI/CD → Variables
- Protect production secrets (Protected + Masked)

### Kubernetes Secrets

Convert kubeconfig to base64:
```bash
cat ~/.kube/config | base64 | tr -d '\n'
```

## Deployment Strategies

### 1. Automatic Deployments

- **Development**: Auto-deploy on push to `develop` branch
- **Staging**: Manual approval on push to `main` branch
- **Production**: Manual approval on release tags

### 2. Canary Deployments (Production)

```
1. Deploy canary with 10% traffic
2. Monitor for 5 minutes
3. Increase to 50% traffic
4. Monitor for 5 minutes
5. Promote to 100% or rollback
```

### 3. Blue-Green Deployments

Use Istio VirtualService to switch traffic:

```bash
# Switch to new version
kubectl patch virtualservice db-backup -n db-backup \
  --type=json \
  -p='[{"op": "replace", "path": "/spec/http/0/route/0/destination/subset", "value": "v2"}]'
```

### 4. Rolling Updates

Helm automatically performs rolling updates:

```bash
helm upgrade db-backup ./helm/db-backup \
  --set image.tag=v1.2.0 \
  --wait
```

## Monitoring and Alerts

### Pipeline Monitoring

**GitHub Actions:**
- View runs: `https://github.com/{org}/{repo}/actions`
- Workflow insights: Actions → Workflows → Select workflow

**GitLab CI:**
- View pipelines: CI/CD → Pipelines
- Pipeline analytics: Analytics → CI/CD Analytics

### Alerts

**Slack Notifications:**
- Deployment success/failure
- Release creation
- Security scan findings
- Test failures

**Email Notifications:**
- GitHub: Settings → Notifications
- GitLab: User Settings → Notifications

### Metrics

Track:
- Build success rate
- Average build time
- Test coverage trends
- Deployment frequency
- Mean time to recovery (MTTR)

## Troubleshooting

### Build Failures

**Check logs:**
```bash
# GitHub Actions
gh run view <run-id> --log

# GitLab CI
gitlab-runner exec docker <job-name>
```

**Common issues:**
1. **Lint failures**: Run locally: `golangci-lint run`
2. **Test failures**: Run with same env vars: `go test ./...`
3. **Docker build failures**: Build locally: `docker build .`

### Deployment Failures

**Check deployment status:**
```bash
kubectl rollout status deployment/db-backup -n db-backup
kubectl get pods -n db-backup
kubectl logs -n db-backup deployment/db-backup
```

**Rollback:**
```bash
# Helm
helm rollback db-backup -n db-backup

# Kubernetes
kubectl rollout undo deployment/db-backup -n db-backup
```

### Security Scan Failures

**False positives:**
Add to ignore list in respective configs:
- `.trivyignore` for Trivy
- `gosec.json` for Gosec
- `.snyk` for Snyk

**Real vulnerabilities:**
1. Update dependencies: `go get -u ./...`
2. Review security advisories
3. Apply patches or wait for upstream fixes

## Best Practices

### 1. Branch Protection

Enable on `main` and `develop`:
- Require status checks to pass
- Require pull request reviews
- Require signed commits
- Restrict force pushes

### 2. PR Workflow

```
1. Create feature branch
2. Make changes
3. Run tests locally
4. Create PR
5. Wait for CI checks
6. Address review comments
7. Merge to develop
8. Auto-deploy to development
9. Test in development
10. Create PR to main
11. Manual deploy to staging
12. UAT in staging
13. Merge to main
14. Create release tag
15. Manual deploy to production
```

### 3. Version Management

Use semantic versioning:
- `v1.0.0`: Major release
- `v1.1.0`: Minor release (new features)
- `v1.1.1`: Patch release (bug fixes)
- `v1.2.0-rc.1`: Release candidate
- `v1.2.0-beta.1`: Beta release

### 4. Cache Management

**GitHub Actions:**
- Go modules are cached automatically
- Docker layers use BuildKit cache

**GitLab CI:**
- Configure cache in `.gitlab-ci.yml`
- Use cache:pull-push policy

### 5. Security

- Never commit secrets
- Use secret scanning
- Rotate credentials regularly
- Use minimal permissions
- Enable branch protection
- Require signed commits

## Support

- Documentation: https://backup.example.com/docs/ci-cd
- CI/CD Issues: https://github.com/your-org/db-backup/labels/ci-cd
- Slack: #db-backup-ci-cd
