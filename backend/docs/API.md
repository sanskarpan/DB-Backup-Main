# DB Backup Utility API Documentation

This directory contains the OpenAPI/Swagger specification for the DB Backup Utility API.

## Overview

The DB Backup Utility provides a comprehensive RESTful API for managing database backups, restores, and schedules. The API supports multiple authentication methods and provides extensive monitoring capabilities.

## Accessing the Documentation

### Swagger UI

Once the server is running, you can access the interactive Swagger UI at:

```
http://localhost:8080/swagger/
```

This provides an interactive interface where you can:
- Browse all API endpoints
- View request/response schemas
- Try out API calls directly
- Download the OpenAPI specification

### Raw OpenAPI Specification

The OpenAPI specification is available in YAML format:

```
http://localhost:8080/swagger.yaml
```

You can also access it locally:
```
docs/swagger.yaml
```

## Authentication

The API supports three authentication methods:

### 1. JWT Authentication

Obtain a JWT token by authenticating with email and password:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

Use the token in subsequent requests:

```bash
curl -X GET http://localhost:8080/api/v1/backups \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 2. OAuth2

Supported providers:
- Google
- GitHub
- Microsoft/Azure AD
- Facebook

Initiate OAuth2 login:
```
GET /api/v1/auth/oauth2/{provider}/login
```

After authentication, you'll receive a JWT token that can be used for API access.

### 3. API Keys

Set the API key in the request header:

```bash
curl -X GET http://localhost:8080/api/v1/backups \
  -H "X-API-Key: YOUR_API_KEY"
```

## Common API Operations

### Create a Backup

```bash
curl -X POST http://localhost:8080/api/v1/backups \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "database_id": "db_123",
    "database_type": "postgres",
    "compression": "zstd",
    "encryption": true,
    "storage_provider": "s3"
  }'
```

### List Backups

```bash
curl -X GET http://localhost:8080/api/v1/backups?limit=10&offset=0 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Restore a Backup

```bash
curl -X POST http://localhost:8080/api/v1/restores \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "backup_id": "bkp_123456",
    "target_database_id": "db_789"
  }'
```

### Create a Backup Schedule

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily PostgreSQL Backup",
    "database_id": "db_123",
    "cron_expression": "0 2 * * *",
    "enabled": true
  }'
```

## Health Monitoring

Check service health:

```bash
curl -X GET http://localhost:8080/api/v1/health
```

Get Prometheus metrics:

```bash
curl -X GET http://localhost:8080/api/v1/metrics
```

## Response Format

All API responses follow a consistent format:

### Success Response

```json
{
  "data": {
    // Response data
  }
}
```

### Error Response

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {
    // Additional error details
  }
}
```

## Rate Limiting

API requests are rate-limited to prevent abuse:
- Default: 100 requests per minute per IP/API key
- Rate limit headers are included in responses:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Remaining`: Remaining requests in current window
  - `X-RateLimit-Reset`: Time when the rate limit resets

## Versioning

The API uses URL-based versioning:

```
/api/v1/...
```

Breaking changes will result in a new API version (e.g., `/api/v2/`).

## Code Generation

You can generate client libraries in various languages using the OpenAPI specification:

### Using OpenAPI Generator

```bash
# Install OpenAPI Generator
npm install @openapitools/openapi-generator-cli -g

# Generate Python client
openapi-generator-cli generate \
  -i docs/swagger.yaml \
  -g python \
  -o clients/python

# Generate TypeScript client
openapi-generator-cli generate \
  -i docs/swagger.yaml \
  -g typescript-fetch \
  -o clients/typescript

# Generate Go client
openapi-generator-cli generate \
  -i docs/swagger.yaml \
  -g go \
  -o clients/go
```

## Support

For API support and questions:
- Email: support@db-backup.example.com
- GitHub Issues: https://github.com/yourorg/db-backup/issues
- Documentation: https://docs.db-backup.example.com

## License

This API is licensed under the MIT License.
