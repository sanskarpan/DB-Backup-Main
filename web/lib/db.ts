/**
 * IndexedDB wrapper for offline data storage
 * Provides persistent storage for backups, monitoring data, and notifications
 */

import { openDB, DBSchema, IDBPDatabase } from 'idb';

// Database schema
interface DBBackupSchema extends DBSchema {
  backups: {
    key: string;
    value: {
      id: string;
      databaseName: string;
      databaseType: string;
      status: string;
      size: number;
      timestamp: number;
      metadata: Record<string, any>;
      synced: boolean;
    };
    indexes: {
      'by-timestamp': number;
      'by-status': string;
      'by-synced': number;
    };
  };
  monitoring: {
    key: string;
    value: {
      id: string;
      metric: string;
      value: number;
      timestamp: number;
      metadata: Record<string, any>;
      synced: boolean;
    };
    indexes: {
      'by-timestamp': number;
      'by-metric': string;
      'by-synced': number;
    };
  };
  notifications: {
    key: string;
    value: {
      id: string;
      title: string;
      body: string;
      type: string;
      timestamp: number;
      read: boolean;
      data: Record<string, any>;
    };
    indexes: {
      'by-timestamp': number;
      'by-read': number;
      'by-type': string;
    };
  };
  offlineQueue: {
    key: string;
    value: {
      id: string;
      url: string;
      method: string;
      headers: Record<string, string>;
      body: any;
      timestamp: number;
      retries: number;
    };
    indexes: {
      'by-timestamp': number;
    };
  };
  settings: {
    key: string;
    value: {
      key: string;
      value: any;
      timestamp: number;
    };
  };
  compliance: {
    key: string;
    value: {
      id: string;
      reportType: string;
      data: Record<string, any>;
      timestamp: number;
      synced: boolean;
    };
    indexes: {
      'by-timestamp': number;
      'by-type': string;
      'by-synced': number;
    };
  };
}

const DB_NAME = 'db-backup-manager';
const DB_VERSION = 1;

class OfflineDB {
  private db: IDBPDatabase<DBBackupSchema> | null = null;

  async init(): Promise<void> {
    if (this.db) return;

    this.db = await openDB<DBBackupSchema>(DB_NAME, DB_VERSION, {
      upgrade(db, oldVersion, newVersion, transaction) {
        // Backups store
        if (!db.objectStoreNames.contains('backups')) {
          const backupStore = db.createObjectStore('backups', {
            keyPath: 'id',
          });
          backupStore.createIndex('by-timestamp', 'timestamp');
          backupStore.createIndex('by-status', 'status');
          backupStore.createIndex('by-synced', 'synced');
        }

        // Monitoring store
        if (!db.objectStoreNames.contains('monitoring')) {
          const monitoringStore = db.createObjectStore('monitoring', {
            keyPath: 'id',
          });
          monitoringStore.createIndex('by-timestamp', 'timestamp');
          monitoringStore.createIndex('by-metric', 'metric');
          monitoringStore.createIndex('by-synced', 'synced');
        }

        // Notifications store
        if (!db.objectStoreNames.contains('notifications')) {
          const notificationStore = db.createObjectStore('notifications', {
            keyPath: 'id',
          });
          notificationStore.createIndex('by-timestamp', 'timestamp');
          notificationStore.createIndex('by-read', 'read');
          notificationStore.createIndex('by-type', 'type');
        }

        // Offline queue store
        if (!db.objectStoreNames.contains('offlineQueue')) {
          const queueStore = db.createObjectStore('offlineQueue', {
            keyPath: 'id',
          });
          queueStore.createIndex('by-timestamp', 'timestamp');
        }

        // Settings store
        if (!db.objectStoreNames.contains('settings')) {
          db.createObjectStore('settings', { keyPath: 'key' });
        }

        // Compliance store
        if (!db.objectStoreNames.contains('compliance')) {
          const complianceStore = db.createObjectStore('compliance', {
            keyPath: 'id',
          });
          complianceStore.createIndex('by-timestamp', 'timestamp');
          complianceStore.createIndex('by-type', 'reportType');
          complianceStore.createIndex('by-synced', 'synced');
        }
      },
    });
  }

  // ============================================================================
  // Backups
  // ============================================================================

  async addBackup(backup: DBBackupSchema['backups']['value']): Promise<void> {
    await this.init();
    await this.db!.add('backups', backup);
  }

  async getBackup(id: string): Promise<DBBackupSchema['backups']['value'] | undefined> {
    await this.init();
    return await this.db!.get('backups', id);
  }

  async getAllBackups(): Promise<DBBackupSchema['backups']['value'][]> {
    await this.init();
    return await this.db!.getAll('backups');
  }

  async getUnsyncedBackups(): Promise<DBBackupSchema['backups']['value'][]> {
    await this.init();
    return await this.db!.getAllFromIndex('backups', 'by-synced', 0);
  }

  async updateBackup(
    id: string,
    updates: Partial<DBBackupSchema['backups']['value']>
  ): Promise<void> {
    await this.init();
    const backup = await this.getBackup(id);
    if (backup) {
      await this.db!.put('backups', { ...backup, ...updates });
    }
  }

  async deleteBackup(id: string): Promise<void> {
    await this.init();
    await this.db!.delete('backups', id);
  }

  async deleteOldBackups(daysToKeep: number = 30): Promise<void> {
    await this.init();
    const cutoffTime = Date.now() - daysToKeep * 24 * 60 * 60 * 1000;
    const allBackups = await this.getAllBackups();

    for (const backup of allBackups) {
      if (backup.timestamp < cutoffTime && backup.synced) {
        await this.deleteBackup(backup.id);
      }
    }
  }

  // ============================================================================
  // Monitoring
  // ============================================================================

  async addMonitoringData(
    data: DBBackupSchema['monitoring']['value']
  ): Promise<void> {
    await this.init();
    await this.db!.add('monitoring', data);
  }

  async getMonitoringData(id: string): Promise<DBBackupSchema['monitoring']['value'] | undefined> {
    await this.init();
    return await this.db!.get('monitoring', id);
  }

  async getAllMonitoringData(): Promise<DBBackupSchema['monitoring']['value'][]> {
    await this.init();
    return await this.db!.getAll('monitoring');
  }

  async getMonitoringByMetric(
    metric: string
  ): Promise<DBBackupSchema['monitoring']['value'][]> {
    await this.init();
    return await this.db!.getAllFromIndex('monitoring', 'by-metric', metric);
  }

  async getRecentMonitoringData(
    hours: number = 24
  ): Promise<DBBackupSchema['monitoring']['value'][]> {
    await this.init();
    const cutoffTime = Date.now() - hours * 60 * 60 * 1000;
    const allData = await this.getAllMonitoringData();
    return allData.filter((data) => data.timestamp >= cutoffTime);
  }

  async deleteOldMonitoringData(hoursToKeep: number = 168): Promise<void> {
    await this.init();
    const cutoffTime = Date.now() - hoursToKeep * 60 * 60 * 1000;
    const allData = await this.getAllMonitoringData();

    for (const data of allData) {
      if (data.timestamp < cutoffTime && data.synced) {
        await this.db!.delete('monitoring', data.id);
      }
    }
  }

  // ============================================================================
  // Notifications
  // ============================================================================

  async addNotification(
    notification: DBBackupSchema['notifications']['value']
  ): Promise<void> {
    await this.init();
    await this.db!.add('notifications', notification);
  }

  async getNotification(
    id: string
  ): Promise<DBBackupSchema['notifications']['value'] | undefined> {
    await this.init();
    return await this.db!.get('notifications', id);
  }

  async getAllNotifications(): Promise<DBBackupSchema['notifications']['value'][]> {
    await this.init();
    const notifications = await this.db!.getAll('notifications');
    return notifications.sort((a, b) => b.timestamp - a.timestamp);
  }

  async getUnreadNotifications(): Promise<DBBackupSchema['notifications']['value'][]> {
    await this.init();
    return await this.db!.getAllFromIndex('notifications', 'by-read', 0);
  }

  async markNotificationAsRead(id: string): Promise<void> {
    await this.init();
    const notification = await this.getNotification(id);
    if (notification) {
      await this.db!.put('notifications', { ...notification, read: true });
    }
  }

  async markAllNotificationsAsRead(): Promise<void> {
    await this.init();
    const unread = await this.getUnreadNotifications();
    for (const notification of unread) {
      await this.markNotificationAsRead(notification.id);
    }
  }

  async deleteNotification(id: string): Promise<void> {
    await this.init();
    await this.db!.delete('notifications', id);
  }

  async deleteOldNotifications(daysToKeep: number = 30): Promise<void> {
    await this.init();
    const cutoffTime = Date.now() - daysToKeep * 24 * 60 * 60 * 1000;
    const allNotifications = await this.getAllNotifications();

    for (const notification of allNotifications) {
      if (notification.timestamp < cutoffTime && notification.read) {
        await this.deleteNotification(notification.id);
      }
    }
  }

  // ============================================================================
  // Offline Queue
  // ============================================================================

  async addToQueue(
    item: DBBackupSchema['offlineQueue']['value']
  ): Promise<void> {
    await this.init();
    await this.db!.add('offlineQueue', item);
  }

  async getQueueItem(
    id: string
  ): Promise<DBBackupSchema['offlineQueue']['value'] | undefined> {
    await this.init();
    return await this.db!.get('offlineQueue', id);
  }

  async getAllQueueItems(): Promise<DBBackupSchema['offlineQueue']['value'][]> {
    await this.init();
    const items = await this.db!.getAll('offlineQueue');
    return items.sort((a, b) => a.timestamp - b.timestamp);
  }

  async updateQueueItem(
    id: string,
    updates: Partial<DBBackupSchema['offlineQueue']['value']>
  ): Promise<void> {
    await this.init();
    const item = await this.getQueueItem(id);
    if (item) {
      await this.db!.put('offlineQueue', { ...item, ...updates });
    }
  }

  async deleteQueueItem(id: string): Promise<void> {
    await this.init();
    await this.db!.delete('offlineQueue', id);
  }

  async clearQueue(): Promise<void> {
    await this.init();
    const tx = this.db!.transaction('offlineQueue', 'readwrite');
    await tx.objectStore('offlineQueue').clear();
    await tx.done;
  }

  // ============================================================================
  // Settings
  // ============================================================================

  async getSetting(key: string): Promise<any> {
    await this.init();
    const setting = await this.db!.get('settings', key);
    return setting?.value;
  }

  async setSetting(key: string, value: any): Promise<void> {
    await this.init();
    await this.db!.put('settings', {
      key,
      value,
      timestamp: Date.now(),
    });
  }

  async deleteSetting(key: string): Promise<void> {
    await this.init();
    await this.db!.delete('settings', key);
  }

  // ============================================================================
  // Compliance
  // ============================================================================

  async addComplianceReport(
    report: DBBackupSchema['compliance']['value']
  ): Promise<void> {
    await this.init();
    await this.db!.add('compliance', report);
  }

  async getComplianceReport(
    id: string
  ): Promise<DBBackupSchema['compliance']['value'] | undefined> {
    await this.init();
    return await this.db!.get('compliance', id);
  }

  async getAllComplianceReports(): Promise<DBBackupSchema['compliance']['value'][]> {
    await this.init();
    return await this.db!.getAll('compliance');
  }

  async getComplianceReportsByType(
    type: string
  ): Promise<DBBackupSchema['compliance']['value'][]> {
    await this.init();
    return await this.db!.getAllFromIndex('compliance', 'by-type', type);
  }

  async deleteComplianceReport(id: string): Promise<void> {
    await this.init();
    await this.db!.delete('compliance', id);
  }

  // ============================================================================
  // Utility
  // ============================================================================

  async clearAll(): Promise<void> {
    await this.init();
    const stores: Array<keyof DBBackupSchema> = [
      'backups',
      'monitoring',
      'notifications',
      'offlineQueue',
      'settings',
      'compliance',
    ];

    for (const store of stores) {
      const tx = this.db!.transaction(store as any, 'readwrite');
      await (tx.objectStore(store as any) as any).clear();
      await tx.done;
    }
  }

  async getStorageEstimate(): Promise<StorageEstimate> {
    if ('storage' in navigator && 'estimate' in navigator.storage) {
      return await navigator.storage.estimate();
    }
    return { usage: 0, quota: 0 };
  }

  async close(): Promise<void> {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
  }
}

// Export singleton instance
export const offlineDB = new OfflineDB();

// Export types
export type {
  DBBackupSchema,
};
