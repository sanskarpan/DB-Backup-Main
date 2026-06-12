/**
 * DB Backup Platform - Utilities
 *
 * Shared utility functions for all client applications
 */

import type { Backup, DatabaseType } from '@db-backup/types';

// ====================================
// Format Utilities
// ====================================

/**
 * Format bytes to human-readable size
 */
export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

/**
 * Format duration in milliseconds to human-readable string
 */
export function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${minutes % 60}m`;
  if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
  return `${seconds}s`;
}

/**
 * Format date to relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(date: string | Date): string {
  const now = new Date();
  const then = typeof date === 'string' ? new Date(date) : date;
  const diff = now.getTime() - then.getTime();

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  if (years > 0) return `${years} year${years > 1 ? 's' : ''} ago`;
  if (months > 0) return `${months} month${months > 1 ? 's' : ''} ago`;
  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`;
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
  return 'just now';
}

/**
 * Format date to ISO string for API
 */
export function formatDateForApi(date: Date): string {
  return date.toISOString();
}

/**
 * Parse API date string to Date object
 */
export function parseDateFromApi(dateString: string): Date {
  return new Date(dateString);
}

// ====================================
// Validation Utilities
// ====================================

/**
 * Validate email format
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

/**
 * Validate cron expression format (basic validation)
 */
export function isValidCronExpression(cron: string): boolean {
  const parts = cron.trim().split(/\s+/);
  return parts.length >= 5 && parts.length <= 7;
}

/**
 * Validate URL format
 */
export function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate port number
 */
export function isValidPort(port: number): boolean {
  return Number.isInteger(port) && port > 0 && port <= 65535;
}

// ====================================
// Database Utilities
// ====================================

/**
 * Get default port for database type
 */
export function getDefaultPort(type: DatabaseType): number {
  const ports: Record<DatabaseType, number> = {
    postgres: 5432,
    mysql: 3306,
    mongodb: 27017,
    redis: 6379,
    cassandra: 9042,
    scylladb: 9042,
    elasticsearch: 9200,
    opensearch: 9200,
    dynamodb: 8000,
    influxdb: 8086,
    timescaledb: 5432,
  };
  return ports[type] || 0;
}

/**
 * Get database type display name
 */
export function getDatabaseTypeName(type: DatabaseType): string {
  const names: Record<DatabaseType, string> = {
    postgres: 'PostgreSQL',
    mysql: 'MySQL',
    mongodb: 'MongoDB',
    redis: 'Redis',
    cassandra: 'Apache Cassandra',
    scylladb: 'ScyllaDB',
    elasticsearch: 'Elasticsearch',
    opensearch: 'OpenSearch',
    dynamodb: 'Amazon DynamoDB',
    influxdb: 'InfluxDB',
    timescaledb: 'TimescaleDB',
  };
  return names[type] || type;
}

// ====================================
// Backup Utilities
// ====================================

/**
 * Get backup status color/variant
 */
export function getBackupStatusColor(status: Backup['status']): string {
  const colors: Record<Backup['status'], string> = {
    pending: 'gray',
    in_progress: 'blue',
    completed: 'green',
    failed: 'red',
  };
  return colors[status] || 'gray';
}

/**
 * Check if backup is in progress
 */
export function isBackupInProgress(backup: Backup): boolean {
  return backup.status === 'in_progress' || backup.status === 'pending';
}

/**
 * Check if backup is completed successfully
 */
export function isBackupCompleted(backup: Backup): boolean {
  return backup.status === 'completed';
}

/**
 * Check if backup has failed
 */
export function isBackupFailed(backup: Backup): boolean {
  return backup.status === 'failed';
}

/**
 * Calculate backup success rate
 */
export function calculateSuccessRate(successful: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((successful / total) * 100);
}

// ====================================
// String Utilities
// ====================================

/**
 * Truncate string to specified length
 */
export function truncate(str: string, length: number, suffix: string = '...'): string {
  if (str.length <= length) return str;
  return str.substring(0, length - suffix.length) + suffix;
}

/**
 * Capitalize first letter
 */
export function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Convert camelCase to Title Case
 */
export function camelToTitle(str: string): string {
  return str
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (char) => char.toUpperCase())
    .trim();
}

// ====================================
// Array Utilities
// ====================================

/**
 * Group array items by key
 */
export function groupBy<T>(array: T[], key: keyof T): Record<string, T[]> {
  return array.reduce(
    (result, item) => {
      const groupKey = String(item[key]);
      if (!result[groupKey]) {
        result[groupKey] = [];
      }
      result[groupKey].push(item);
      return result;
    },
    {} as Record<string, T[]>
  );
}

/**
 * Sort array by key
 */
export function sortBy<T>(array: T[], key: keyof T, order: 'asc' | 'desc' = 'asc'): T[] {
  return [...array].sort((a, b) => {
    const aVal = a[key];
    const bVal = b[key];

    if (aVal < bVal) return order === 'asc' ? -1 : 1;
    if (aVal > bVal) return order === 'asc' ? 1 : -1;
    return 0;
  });
}

// ====================================
// Object Utilities
// ====================================

/**
 * Deep clone object
 */
export function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj));
}

/**
 * Check if object is empty
 */
export function isEmpty(obj: object): boolean {
  return Object.keys(obj).length === 0;
}

/**
 * Remove undefined/null values from object
 */
export function compact<T extends Record<string, any>>(obj: T): Partial<T> {
  return Object.entries(obj).reduce(
    (acc, [key, value]) => {
      if (value !== undefined && value !== null) {
        acc[key as keyof T] = value;
      }
      return acc;
    },
    {} as Partial<T>
  );
}

// ====================================
// Debounce & Throttle
// ====================================

/**
 * Debounce function
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

  return (...args: Parameters<T>) => {
    if (timeout) clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

/**
 * Throttle function
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean = false;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

// ====================================
// Local Storage Utilities
// ====================================

/**
 * Safe localStorage getItem with JSON parsing
 */
export function getLocalStorage<T>(key: string, defaultValue: T): T {
  if (typeof window === 'undefined') return defaultValue;

  try {
    const item = window.localStorage.getItem(key);
    return item ? JSON.parse(item) : defaultValue;
  } catch {
    return defaultValue;
  }
}

/**
 * Safe localStorage setItem with JSON stringifying
 */
export function setLocalStorage<T>(key: string, value: T): void {
  if (typeof window === 'undefined') return;

  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch (error) {
    console.error('Error saving to localStorage:', error);
  }
}

/**
 * Remove item from localStorage
 */
export function removeLocalStorage(key: string): void {
  if (typeof window === 'undefined') return;

  try {
    window.localStorage.removeItem(key);
  } catch (error) {
    console.error('Error removing from localStorage:', error);
  }
}
