import { describe, it, expect } from 'vitest';
import {
  formatBytes,
  formatDuration,
  isValidEmail,
  isValidPort,
  getDefaultPort,
  getDatabaseTypeName,
  getBackupStatusColor,
  truncate,
  capitalize,
  groupBy,
  sortBy,
  isEmpty,
  compact,
} from '../src';

describe('Utils Package', () => {
  describe('Format Utilities', () => {
    it('should format bytes correctly', () => {
      expect(formatBytes(0)).toBe('0 Bytes');
      expect(formatBytes(1024)).toBe('1 KB');
      expect(formatBytes(1048576)).toBe('1 MB');
      expect(formatBytes(1073741824)).toBe('1 GB');
    });

    it('should format duration correctly', () => {
      expect(formatDuration(1000)).toBe('1s');
      expect(formatDuration(60000)).toBe('1m 0s');
      expect(formatDuration(3600000)).toBe('1h 0m');
    });
  });

  describe('Validation Utilities', () => {
    it('should validate email correctly', () => {
      expect(isValidEmail('test@example.com')).toBe(true);
      expect(isValidEmail('invalid-email')).toBe(false);
      expect(isValidEmail('test@')).toBe(false);
    });

    it('should validate port correctly', () => {
      expect(isValidPort(80)).toBe(true);
      expect(isValidPort(5432)).toBe(true);
      expect(isValidPort(0)).toBe(false);
      expect(isValidPort(65536)).toBe(false);
      expect(isValidPort(3.14)).toBe(false);
    });
  });

  describe('Database Utilities', () => {
    it('should return correct default ports', () => {
      expect(getDefaultPort('postgres')).toBe(5432);
      expect(getDefaultPort('mysql')).toBe(3306);
      expect(getDefaultPort('mongodb')).toBe(27017);
      expect(getDefaultPort('redis')).toBe(6379);
    });

    it('should return correct database type names', () => {
      expect(getDatabaseTypeName('postgres')).toBe('PostgreSQL');
      expect(getDatabaseTypeName('mysql')).toBe('MySQL');
      expect(getDatabaseTypeName('mongodb')).toBe('MongoDB');
    });
  });

  describe('Backup Utilities', () => {
    it('should return correct status colors', () => {
      expect(getBackupStatusColor('completed')).toBe('green');
      expect(getBackupStatusColor('failed')).toBe('red');
      expect(getBackupStatusColor('in_progress')).toBe('blue');
      expect(getBackupStatusColor('pending')).toBe('gray');
    });
  });

  describe('String Utilities', () => {
    it('should truncate strings correctly', () => {
      expect(truncate('Hello World', 5)).toBe('He...');
      expect(truncate('Hi', 10)).toBe('Hi');
    });

    it('should capitalize strings correctly', () => {
      expect(capitalize('hello')).toBe('Hello');
      expect(capitalize('WORLD')).toBe('WORLD');
    });
  });

  describe('Array Utilities', () => {
    it('should group items correctly', () => {
      const items = [
        { id: 1, type: 'a' },
        { id: 2, type: 'b' },
        { id: 3, type: 'a' },
      ];
      const grouped = groupBy(items, 'type');
      expect(grouped['a']).toHaveLength(2);
      expect(grouped['b']).toHaveLength(1);
    });

    it('should sort items correctly', () => {
      const items = [{ id: 3 }, { id: 1 }, { id: 2 }];
      const sorted = sortBy(items, 'id', 'asc');
      expect(sorted[0].id).toBe(1);
      expect(sorted[2].id).toBe(3);
    });
  });

  describe('Object Utilities', () => {
    it('should check if object is empty', () => {
      expect(isEmpty({})).toBe(true);
      expect(isEmpty({ a: 1 })).toBe(false);
    });

    it('should compact objects correctly', () => {
      const obj = { a: 1, b: null, c: undefined, d: 'test' };
      const compacted = compact(obj);
      expect(compacted).toEqual({ a: 1, d: 'test' });
    });
  });
});
