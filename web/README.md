# DB Backup Web

Next.js web frontend for the DB Backup platform.

## Features

- Modern React 18 with Next.js 14 App Router
- Progressive Web App (PWA) support with offline capabilities
- Real-time backup monitoring and management
- Database configuration and testing
- Schedule management with cron expressions
- Storage provider configuration
- Advanced analytics and visualizations
- Responsive design with Tailwind CSS
- Accessibility compliant (WCAG 2.1)

## Tech Stack

- **Framework**: Next.js 14 with App Router
- **UI**: React 18, Tailwind CSS, Lucide Icons
- **State Management**: TanStack Query (React Query)
- **Forms**: React Hook Form + Zod validation
- **Charts**: Recharts
- **Testing**: Vitest + React Testing Library
- **PWA**: @ducanh2912/next-pwa

## Shared Packages

This project uses shared packages from `@db-backup/*`:

- `@db-backup/types` - TypeScript type definitions
- `@db-backup/api-client` - API client library
- `@db-backup/utils` - Utility functions

## Development

### Prerequisites

- Node.js 18+
- npm 9+
- Backend API running at `http://localhost:8080`

### Installation

```bash
npm install
```

### Environment Variables

Create a `.env.local` file:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

### Run Development Server

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Build for Production

```bash
npm run build
npm start
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run specific test suites
npm run test:pwa
npm run test:a11y
npm run test:performance
npm run test:integration

# Type check
npm run type-check
```

### Linting

```bash
npm run lint
```

## Project Structure

```
db-backup-web/
├── app/                    # Next.js App Router pages
│   ├── page.tsx           # Dashboard
│   ├── backups/           # Backup management
│   ├── databases/         # Database configuration
│   ├── schedules/         # Schedule management
│   ├── monitoring/        # Real-time monitoring
│   ├── security/          # Security features
│   └── ...
├── components/            # React components
│   ├── ui/               # Base UI components
│   ├── dashboard/        # Dashboard components
│   ├── layout/           # Layout components
│   └── ...
├── lib/                  # Utilities and helpers
│   ├── api.ts           # API client wrapper
│   └── ...
├── public/              # Static assets
├── tests/               # Test files
└── package.json
```

## Features

### Dashboard
- Real-time backup statistics
- Recent backups overview
- System health monitoring
- Quick actions

### Backup Management
- List all backups with filtering
- Create manual backups
- Download backup files
- Restore backups
- Delete old backups

### Database Configuration
- Add/edit database connections
- Support for 11 database types
- Test connections
- View connection details

### Schedule Management
- Create backup schedules with cron expressions
- Enable/disable schedules
- View next run time
- Configure retention policies

### Storage Providers
- Configure multiple storage backends
- Support for S3, GCS, Azure, MinIO, Wasabi, B2
- Test storage connectivity
- View storage statistics

### Monitoring
- Real-time backup progress
- System metrics and alerts
- Audit logs
- Performance analytics

### Security
- JWT authentication
- Role-based access control
- Encrypted credentials
- Security audit logs

## PWA Features

- Offline support
- Install as desktop/mobile app
- Background sync
- Push notifications
- Cached API responses

## Accessibility

- WCAG 2.1 Level AA compliant
- Keyboard navigation
- Screen reader support
- High contrast mode
- Focus management

## License

MIT
