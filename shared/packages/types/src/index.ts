/**
 * DB Backup Platform - Type Definitions
 *
 * Shared TypeScript types for all client applications
 */

// ====================================
// Backup Types
// ====================================

export interface Backup {
  id: string;
  database_id: string;
  database_name?: string;
  filename: string;
  size: number;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  storage_path: string;
  created_at: string;
  completed_at?: string;
  error?: string;
  metadata?: Record<string, string>;
}

export interface BackupListResponse {
  backups: Backup[];
  total: number;
  page: number;
  page_size: number;
}

// ====================================
// Database Types
// ====================================

export type DatabaseType =
  | 'postgres'
  | 'mysql'
  | 'mongodb'
  | 'redis'
  | 'cassandra'
  | 'scylladb'
  | 'elasticsearch'
  | 'opensearch'
  | 'dynamodb'
  | 'influxdb'
  | 'timescaledb';

export interface Database {
  id: string;
  name: string;
  type: DatabaseType;
  host: string;
  port: number;
  username: string;
  created_at: string;
  updated_at: string;
}

// ====================================
// Schedule Types
// ====================================

export interface Schedule {
  id: string;
  database_id: string;
  database_name?: string;
  cron_expression: string;
  retention_days: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run?: string;
  next_run?: string;
}

// ====================================
// Stats Types
// ====================================

export interface Stats {
  total_backups: number;
  successful_backups: number;
  failed_backups: number;
  total_size: number;
  databases: number;
  schedules: number;
}

// ====================================
// Storage Provider Types
// ====================================

export type StorageProviderType = 'local' | 's3' | 'gcs' | 'azure' | 'minio' | 'wasabi' | 'b2';

export interface BaseStorageConfig {
  type: StorageProviderType;
}

export interface LocalStorageConfig extends BaseStorageConfig {
  type: 'local';
  path: string;
}

export interface S3Config extends BaseStorageConfig {
  type: 's3';
  region: string;
  bucket: string;
  access_key: string;
  secret_key: string;
  endpoint?: string;
  use_path_style?: boolean;
}

export interface GCSConfig extends BaseStorageConfig {
  type: 'gcs';
  project: string;
  bucket: string;
  credentials_file: string;
}

export interface AzureConfig extends BaseStorageConfig {
  type: 'azure';
  account_name: string;
  account_key: string;
  container: string;
}

export interface MinIOConfig extends BaseStorageConfig {
  type: 'minio';
  endpoint: string;
  access_key: string;
  secret_key: string;
  bucket: string;
  use_ssl?: boolean;
  region?: string;
  use_path_style?: boolean;
}

export interface WasabiConfig extends BaseStorageConfig {
  type: 'wasabi';
  endpoint: string;
  region: string;
  access_key: string;
  secret_key: string;
  bucket: string;
  use_ssl?: boolean;
  immutable?: boolean;
  retention_days?: number;
}

export interface BackblazeB2Config extends BaseStorageConfig {
  type: 'b2';
  account_id: string;
  application_key: string;
  bucket: string;
  bucket_id?: string;
  endpoint?: string;
  immutable?: boolean;
  retention_days?: number;
}

export type StorageProviderConfig =
  | LocalStorageConfig
  | S3Config
  | GCSConfig
  | AzureConfig
  | MinIOConfig
  | WasabiConfig
  | BackblazeB2Config;

export interface StorageProvider {
  id: string;
  name: string;
  type: StorageProviderType;
  config: StorageProviderConfig;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

// ====================================
// API Response Types
// ====================================

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface HealthCheckResponse {
  status: string;
  timestamp?: string;
  uptime?: number;
}

export interface ConnectionTestResponse {
  success: boolean;
  message: string;
  latency?: number;
}

// ====================================
// Pagination Types
// ====================================

export interface PaginationParams {
  page?: number;
  page_size?: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ====================================
// Filter Types
// ====================================

export interface BackupFilter {
  database_id?: string;
  status?: Backup['status'];
  start_date?: string;
  end_date?: string;
}

export interface ScheduleFilter {
  database_id?: string;
  enabled?: boolean;
}

// ====================================
// User & Auth Types
// ====================================

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'user' | 'viewer';
  created_at: string;
  updated_at: string;
}

export interface AuthToken {
  token: string;
  expires_at: string;
  refresh_token?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  token: AuthToken;
}

// ====================================
// Error Types
// ====================================

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  timestamp: string;
}

export class BackupError extends Error {
  constructor(
    message: string,
    public code: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'BackupError';
  }
}

// ====================================
// Utility Types
// ====================================

export type Nullable<T> = T | null;
export type Optional<T> = T | undefined;
export type Maybe<T> = T | null | undefined;

export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type RequireAtLeastOne<T, Keys extends keyof T = keyof T> = Pick<T, Exclude<keyof T, Keys>> &
  {
    [K in Keys]-?: Required<Pick<T, K>> & Partial<Pick<T, Exclude<Keys, K>>>;
  }[Keys];
