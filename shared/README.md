# DB Backup Shared Packages

Shared TypeScript packages for the DB Backup platform. These packages provide common types, API clients, and utilities used across all client applications (web, desktop, mobile, extensions).

## Packages

### @db-backup/types

TypeScript type definitions for the DB Backup platform.

```bash
npm install @db-backup/types
```

```typescript
import type { Backup, Database, Schedule } from '@db-backup/types';
```

### @db-backup/api-client

API client for communicating with the DB Backup backend.

```bash
npm install @db-backup/api-client
```

```typescript
import { createApiClient } from '@db-backup/api-client';

const client = createApiClient({
  baseURL: 'http://localhost:8080/api',
  getAuthToken: () => localStorage.getItem('auth_token'),
  onUnauthorized: () => {
    // Handle unauthorized access
  },
});

// Use the client
const backups = await client.listBackups();
```

### @db-backup/utils

Utility functions for the DB Backup platform.

```bash
npm install @db-backup/utils
```

```typescript
import { formatBytes, formatDuration, getDefaultPort } from '@db-backup/utils';

const sizeStr = formatBytes(1048576); // "1 MB"
const durationStr = formatDuration(3600000); // "1h 0m"
const port = getDefaultPort('postgres'); // 5432
```

## Development

### Installation

```bash
npm install
```

### Build

Build all packages:

```bash
npm run build
```

### Test

Run tests for all packages:

```bash
npm run test
```

### Lint

Run TypeScript type checking:

```bash
npm run lint
```

## Project Structure

```
db-backup-shared/
├── package.json          # Root package.json with workspaces
├── tsconfig.json         # Shared TypeScript configuration
├── packages/
│   ├── types/           # TypeScript type definitions
│   ├── api-client/      # API client library
│   └── utils/           # Utility functions
```

## License

MIT
