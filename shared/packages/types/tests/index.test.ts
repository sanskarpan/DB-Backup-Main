import { describe, it, expect } from 'vitest';
import type { Backup, Database, Schedule, Stats } from '../src';

describe('Types Package', () => {
  it('should export Backup type correctly', () => {
    const backup: Backup = {
      id: 'test-123',
      database_id: 'db-1',
      filename: 'backup.sql',
      size: 1024,
      status: 'completed',
      storage_path: '/backups/backup.sql',
      created_at: '2024-01-01T00:00:00Z',
    };

    expect(backup.id).toBe('test-123');
    expect(backup.status).toBe('completed');
  });

  it('should export Database type correctly', () => {
    const database: Database = {
      id: 'db-1',
      name: 'production',
      type: 'postgres',
      host: 'localhost',
      port: 5432,
      username: 'admin',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    };

    expect(database.type).toBe('postgres');
    expect(database.port).toBe(5432);
  });

  it('should export Schedule type correctly', () => {
    const schedule: Schedule = {
      id: 'sched-1',
      database_id: 'db-1',
      cron_expression: '0 0 * * *',
      retention_days: 30,
      enabled: true,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    };

    expect(schedule.enabled).toBe(true);
    expect(schedule.retention_days).toBe(30);
  });

  it('should export Stats type correctly', () => {
    const stats: Stats = {
      total_backups: 100,
      successful_backups: 95,
      failed_backups: 5,
      total_size: 1024000,
      databases: 10,
      schedules: 5,
    };

    expect(stats.total_backups).toBe(100);
    expect(stats.successful_backups).toBe(95);
  });
});
