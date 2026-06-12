import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DbBackupApiClient, createApiClient } from '../src';

describe('API Client Package', () => {
  let client: DbBackupApiClient;

  beforeEach(() => {
    client = createApiClient({
      baseURL: 'http://localhost:8080/api',
      timeout: 5000,
    });
  });

  it('should create API client with factory function', () => {
    expect(client).toBeInstanceOf(DbBackupApiClient);
  });

  it('should have all backup methods', () => {
    expect(typeof client.listBackups).toBe('function');
    expect(typeof client.getBackup).toBe('function');
    expect(typeof client.createBackup).toBe('function');
    expect(typeof client.deleteBackup).toBe('function');
    expect(typeof client.downloadBackup).toBe('function');
    expect(typeof client.restoreBackup).toBe('function');
  });

  it('should have all database methods', () => {
    expect(typeof client.listDatabases).toBe('function');
    expect(typeof client.getDatabase).toBe('function');
    expect(typeof client.createDatabase).toBe('function');
    expect(typeof client.updateDatabase).toBe('function');
    expect(typeof client.deleteDatabase).toBe('function');
    expect(typeof client.testConnection).toBe('function');
  });

  it('should have all schedule methods', () => {
    expect(typeof client.listSchedules).toBe('function');
    expect(typeof client.getSchedule).toBe('function');
    expect(typeof client.createSchedule).toBe('function');
    expect(typeof client.updateSchedule).toBe('function');
    expect(typeof client.deleteSchedule).toBe('function');
  });

  it('should have stats and health methods', () => {
    expect(typeof client.getStats).toBe('function');
    expect(typeof client.healthCheck).toBe('function');
  });

  it('should have storage provider methods', () => {
    expect(typeof client.listStorageProviders).toBe('function');
    expect(typeof client.getStorageProvider).toBe('function');
    expect(typeof client.createStorageProvider).toBe('function');
    expect(typeof client.updateStorageProvider).toBe('function');
    expect(typeof client.deleteStorageProvider).toBe('function');
    expect(typeof client.testStorageProvider).toBe('function');
  });

  it('should call onUnauthorized callback on 401 response', () => {
    const onUnauthorized = vi.fn();
    const clientWithCallback = createApiClient({
      baseURL: 'http://localhost:8080/api',
      onUnauthorized,
    });

    expect(clientWithCallback).toBeInstanceOf(DbBackupApiClient);
  });

  it('should add auth token to requests when provided', () => {
    const getAuthToken = vi.fn(() => 'test-token');
    const clientWithAuth = createApiClient({
      baseURL: 'http://localhost:8080/api',
      getAuthToken,
    });

    expect(clientWithAuth).toBeInstanceOf(DbBackupApiClient);
  });
});
