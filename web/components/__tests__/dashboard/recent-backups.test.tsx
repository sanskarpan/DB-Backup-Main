import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { RecentBackups } from '../../dashboard/recent-backups'

describe('RecentBackups', () => {
  it('should display message when no backups', () => {
    render(<RecentBackups backups={[]} />)
    expect(screen.getByText('No backups yet')).toBeInTheDocument()
  })

  it('should render list of backups', () => {
    const mockBackups = [
      {
        id: '1',
        database_id: 'db-1',
        database_name: 'mydb',
        filename: 'backup1.sql',
        size: 1024000,
        status: 'completed' as const,
        storage_path: '/path/to/backup1',
        created_at: '2025-12-28T10:00:00Z',
      },
      {
        id: '2',
        database_id: 'db-2',
        database_name: 'testdb',
        filename: 'backup2.sql',
        size: 2048000,
        status: 'in_progress' as const,
        storage_path: '/path/to/backup2',
        created_at: '2025-12-28T11:00:00Z',
      },
    ]

    render(<RecentBackups backups={mockBackups} />)

    expect(screen.getByText('mydb')).toBeInTheDocument()
    expect(screen.getByText('testdb')).toBeInTheDocument()
    expect(screen.getByText('backup1.sql')).toBeInTheDocument()
    expect(screen.getByText('backup2.sql')).toBeInTheDocument()
  })

  it('should show correct status icons', () => {
    const completedBackup = [{
      id: '1',
      database_id: 'db-1',
      database_name: 'mydb',
      filename: 'backup.sql',
      size: 1024,
      status: 'completed' as const,
      storage_path: '/path/to/backup',
      created_at: '2025-12-28T10:00:00Z',
    }]

    const { container } = render(<RecentBackups backups={completedBackup} />)

    // Check for green status indicator (completed)
    expect(container.querySelector('.text-green-800')).toBeInTheDocument()
  })

  it('should format file sizes correctly', () => {
    const backups = [{
      id: '1',
      database_id: 'db-1',
      database_name: 'mydb',
      filename: 'backup.sql',
      size: 1048576, // 1 MB
      status: 'completed' as const,
      storage_path: '/path',
      created_at: '2025-12-28T10:00:00Z',
    }]

    render(<RecentBackups backups={backups} />)
    expect(screen.getByText(/1 MB/)).toBeInTheDocument()
  })

  it('should format dates correctly', () => {
    const backups = [{
      id: '1',
      database_id: 'db-1',
      database_name: 'mydb',
      filename: 'backup.sql',
      size: 1024,
      status: 'completed' as const,
      storage_path: '/path',
      created_at: '2025-12-28T10:30:00Z',
    }]

    render(<RecentBackups backups={backups} />)
    // Date format should include Dec 28
    expect(screen.getByText(/Dec 28/)).toBeInTheDocument()
  })
})
