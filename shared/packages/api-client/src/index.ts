/**
 * DB Backup Platform - API Client
 *
 * Axios-based API client for communicating with the backend
 */

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import type {
  Backup,
  BackupListResponse,
  Database,
  Schedule,
  Stats,
  StorageProvider,
  HealthCheckResponse,
  ConnectionTestResponse,
  BackupFilter,
  PaginationParams,
  SecurityStats,
  ThreatAlert,
  ThreatAlertListResponse,
  ThreatAlertFilter,
} from '@db-backup/types';

/**
 * Envelope used by endpoints that wrap their payload in { success, data }.
 */
interface SuccessEnvelope<T> {
  success: boolean;
  message?: string;
  data: T;
}

// ====================================
// API Client Configuration
// ====================================

export interface ApiClientConfig {
  baseURL: string;
  timeout?: number;
  headers?: Record<string, string>;
  onUnauthorized?: () => void;
  getAuthToken?: () => string | null;
}

// ====================================
// API Client Class
// ====================================

export class DbBackupApiClient {
  private client: AxiosInstance;
  private config: ApiClientConfig;

  constructor(config: ApiClientConfig) {
    this.config = config;
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
        ...config.headers,
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor - add auth token
    this.client.interceptors.request.use(
      (config) => {
        const token = this.config.getAuthToken?.();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor - handle errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          this.config.onUnauthorized?.();
        }
        return Promise.reject(error);
      }
    );
  }

  // ====================================
  // Backup Operations
  // ====================================

  async listBackups(
    params?: PaginationParams & { database_id?: string }
  ): Promise<BackupListResponse> {
    const { data } = await this.client.get<BackupListResponse>('/backups', { params });
    return data;
  }

  async getBackup(id: string): Promise<Backup> {
    const { data } = await this.client.get<Backup>(`/backups/${id}`);
    return data;
  }

  async createBackup(config: {
    database_id?: string;
    database_type?: string;
    database?: string;
    host?: string;
    port?: number;
    [key: string]: unknown;
  }): Promise<Backup> {
    const { data } = await this.client.post<Backup>('/backups', config);
    return data;
  }

  async deleteBackup(id: string): Promise<void> {
    await this.client.delete(`/backups/${id}`);
  }

  async downloadBackup(id: string): Promise<Blob> {
    const { data } = await this.client.get(`/backups/${id}/download`, {
      responseType: 'blob',
    });
    return data;
  }

  async restoreBackup(id: string, targetDatabaseId?: string): Promise<void> {
    await this.client.post(`/backups/${id}/restore`, {
      target_database_id: targetDatabaseId,
    });
  }

  // ====================================
  // Database Operations
  // NOTE: The /databases endpoints below are not yet implemented in the backend.
  // The backend currently supports /backups, /schedules, /stats, /security, and /catalog.
  // These methods are kept here for use by web/mobile clients pending backend implementation.
  // ====================================

  async listDatabases(): Promise<Database[]> {
    const { data } = await this.client.get<Database[]>('/databases');
    return data;
  }

  async getDatabase(id: string): Promise<Database> {
    const { data } = await this.client.get<Database>(`/databases/${id}`);
    return data;
  }

  async createDatabase(database: Partial<Database>): Promise<Database> {
    const { data } = await this.client.post<Database>('/databases', database);
    return data;
  }

  async updateDatabase(id: string, database: Partial<Database>): Promise<Database> {
    const { data } = await this.client.put<Database>(`/databases/${id}`, database);
    return data;
  }

  async deleteDatabase(id: string): Promise<void> {
    await this.client.delete(`/databases/${id}`);
  }

  async testConnection(id: string): Promise<ConnectionTestResponse> {
    const { data } = await this.client.post<ConnectionTestResponse>(`/databases/${id}/test`);
    return data;
  }

  // ====================================
  // Schedule Operations
  // ====================================

  async listSchedules(): Promise<Schedule[]> {
    const { data } = await this.client.get<Schedule[]>('/schedules');
    return data;
  }

  async getSchedule(id: string): Promise<Schedule> {
    const { data } = await this.client.get<Schedule>(`/schedules/${id}`);
    return data;
  }

  async createSchedule(schedule: Partial<Schedule>): Promise<Schedule> {
    const { data } = await this.client.post<Schedule>('/schedules', schedule);
    return data;
  }

  async updateSchedule(id: string, schedule: Partial<Schedule>): Promise<Schedule> {
    const { data } = await this.client.put<Schedule>(`/schedules/${id}`, schedule);
    return data;
  }

  async deleteSchedule(id: string): Promise<void> {
    await this.client.delete(`/schedules/${id}`);
  }

  // ====================================
  // Stats Operations
  // ====================================

  async getStats(): Promise<Stats> {
    const { data } = await this.client.get<Stats>('/stats');
    return data;
  }

  // ====================================
  // Health Operations
  // ====================================

  async healthCheck(): Promise<HealthCheckResponse> {
    const { data } = await this.client.get<HealthCheckResponse>('/health');
    return data;
  }

  // ====================================
  // Storage Provider Operations
  // ====================================

  async listStorageProviders(): Promise<StorageProvider[]> {
    const { data } = await this.client.get<StorageProvider[]>('/security/storage/providers');
    return data;
  }

  async getStorageProvider(id: string): Promise<StorageProvider> {
    const { data } = await this.client.get<StorageProvider>(`/security/storage/providers/${id}`);
    return data;
  }

  async createStorageProvider(provider: Partial<StorageProvider>): Promise<StorageProvider> {
    const { data } = await this.client.post<StorageProvider>('/security/storage/providers', provider);
    return data;
  }

  async updateStorageProvider(
    id: string,
    provider: Partial<StorageProvider>
  ): Promise<StorageProvider> {
    const { data } = await this.client.put<StorageProvider>(`/security/storage/providers/${id}`, provider);
    return data;
  }

  async deleteStorageProvider(id: string): Promise<void> {
    await this.client.delete(`/security/storage/providers/${id}`);
  }

  async testStorageProvider(id: string): Promise<ConnectionTestResponse> {
    const { data } = await this.client.post<ConnectionTestResponse>(
      `/security/storage/providers/${id}/test`
    );
    return data;
  }

  // ====================================
  // Security Operations
  //
  // These map to the real backend endpoints GET /security/stats and
  // GET /security/alerts. Both wrap their payload in a { success, data }
  // envelope, so we unwrap `data.data` here.
  // ====================================

  async getSecurityStats(): Promise<SecurityStats> {
    const { data } = await this.client.get<SuccessEnvelope<SecurityStats>>('/security/stats');
    return data.data;
  }

  async listThreatAlerts(params?: ThreatAlertFilter): Promise<ThreatAlertListResponse> {
    const { data } = await this.client.get<SuccessEnvelope<ThreatAlertListResponse>>(
      '/security/alerts',
      { params }
    );
    return data.data;
  }

  async getThreatAlert(id: string): Promise<ThreatAlert> {
    const { data } = await this.client.get<SuccessEnvelope<ThreatAlert>>(
      `/security/alerts/${id}`
    );
    return data.data;
  }

  // ====================================
  // Custom Request Method
  // ====================================

  async request<T = any>(config: AxiosRequestConfig): Promise<AxiosResponse<T>> {
    return this.client.request<T>(config);
  }
}

// ====================================
// Factory Function
// ====================================

export function createApiClient(config: ApiClientConfig): DbBackupApiClient {
  return new DbBackupApiClient(config);
}

// ====================================
// Default Export
// ====================================

export default DbBackupApiClient;
