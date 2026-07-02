/**
 * API Client Wrapper
 *
 * This file wraps the shared @db-backup/api-client and provides
 * a configured instance for the web application.
 */

import { createApiClient } from '@db-backup/api-client';

// Create API client instance with web-specific configuration
export const apiClient = createApiClient({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
  getAuthToken: () => {
    // Get token from localStorage (browser only)
    if (typeof window !== 'undefined') {
      return localStorage.getItem('auth_token');
    }
    return null;
  },
  onUnauthorized: () => {
    // Handle unauthorized access (e.g., redirect to login)
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
  },
});

// Export all methods for backwards compatibility
export const api = {
  // Backups
  listBackups: apiClient.listBackups.bind(apiClient),
  getBackup: apiClient.getBackup.bind(apiClient),
  createBackup: apiClient.createBackup.bind(apiClient),
  deleteBackup: apiClient.deleteBackup.bind(apiClient),
  downloadBackup: apiClient.downloadBackup.bind(apiClient),
  restoreBackup: apiClient.restoreBackup.bind(apiClient),

  // Databases
  listDatabases: apiClient.listDatabases.bind(apiClient),
  getDatabase: apiClient.getDatabase.bind(apiClient),
  createDatabase: apiClient.createDatabase.bind(apiClient),
  updateDatabase: apiClient.updateDatabase.bind(apiClient),
  deleteDatabase: apiClient.deleteDatabase.bind(apiClient),
  testConnection: apiClient.testConnection.bind(apiClient),

  // Schedules
  listSchedules: apiClient.listSchedules.bind(apiClient),
  getSchedule: apiClient.getSchedule.bind(apiClient),
  createSchedule: apiClient.createSchedule.bind(apiClient),
  updateSchedule: apiClient.updateSchedule.bind(apiClient),
  deleteSchedule: apiClient.deleteSchedule.bind(apiClient),

  // Stats
  getStats: apiClient.getStats.bind(apiClient),

  // Health
  healthCheck: apiClient.healthCheck.bind(apiClient),

  // Storage Providers
  listStorageProviders: apiClient.listStorageProviders.bind(apiClient),
  getStorageProvider: apiClient.getStorageProvider.bind(apiClient),
  createStorageProvider: apiClient.createStorageProvider.bind(apiClient),
  updateStorageProvider: apiClient.updateStorageProvider.bind(apiClient),
  deleteStorageProvider: apiClient.deleteStorageProvider.bind(apiClient),
  testStorageProvider: apiClient.testStorageProvider.bind(apiClient),

  // Security
  getSecurityStats: apiClient.getSecurityStats.bind(apiClient),
  listThreatAlerts: apiClient.listThreatAlerts.bind(apiClient),
  getThreatAlert: apiClient.getThreatAlert.bind(apiClient),
};

// Re-export types from shared package
export type {
  Backup,
  BackupListResponse,
  Database,
  DatabaseType,
  Schedule,
  Stats,
  StorageProvider,
  StorageProviderType,
  StorageProviderConfig,
  S3Config,
  GCSConfig,
  AzureConfig,
  MinIOConfig,
  WasabiConfig,
  BackblazeB2Config,
  LocalStorageConfig,
  SecurityStats,
  ThreatAlert,
  ThreatAlertSeverity,
  ThreatAlertStatus,
  ThreatAlertListResponse,
  ThreatAlertFilter,
} from '@db-backup/types';

export default api;
