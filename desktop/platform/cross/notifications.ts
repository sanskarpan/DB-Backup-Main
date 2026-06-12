/**
 * Cross-Platform Notification System
 * Provides unified notification API across Windows, macOS, and Linux
 */

import { invoke } from '@tauri-apps/api/tauri';
import { appWindow } from '@tauri-apps/api/window';

// ============================================================================
// Types
// ============================================================================

export interface UnifiedNotification {
  id?: string;
  title: string;
  body: string;
  icon?: string;
  urgency?: 'low' | 'normal' | 'critical';
  timeout?: number; // milliseconds, -1 for default, 0 for never
  sound?: boolean;
  actions?: NotificationAction[];
  category?: 'backup' | 'restore' | 'alert' | 'info';
  silent?: boolean;
}

export interface NotificationAction {
  id: string;
  title: string;
}

export interface NotificationResult {
  id: string;
  platform: 'windows' | 'macos' | 'linux' | 'unknown';
  delivered: boolean;
  error?: string;
}

export interface NotificationEvent {
  notificationId: string;
  actionId?: string;
  event: 'clicked' | 'dismissed' | 'action';
}

// ============================================================================
// Cross-Platform Notification Service
// ============================================================================

class CrossPlatformNotifications {
  private platform: 'windows' | 'macos' | 'linux' | 'unknown' = 'unknown';
  private listeners: Map<string, (event: NotificationEvent) => void> = new Map();
  private notificationHistory: Map<string, UnifiedNotification> = new Map();

  constructor() {
    this.detectPlatform();
    this.setupEventListeners();
  }

  /**
   * Detect current platform
   */
  private detectPlatform(): void {
    const userAgent = navigator.userAgent.toLowerCase();
    const platform = navigator.platform.toLowerCase();

    if (platform.indexOf('win') > -1) {
      this.platform = 'windows';
    } else if (platform.indexOf('mac') > -1) {
      this.platform = 'macos';
    } else if (platform.indexOf('linux') > -1) {
      this.platform = 'linux';
    }
  }

  /**
   * Setup event listeners for notification interactions
   */
  private setupEventListeners(): void {
    // Windows
    appWindow.listen('notification-action-windows', (event: any) => {
      this.handleNotificationEvent({
        notificationId: event.payload.notificationId,
        actionId: event.payload.actionId,
        event: 'action',
      });
    });

    // macOS
    appWindow.listen('notification-action-macos', (event: any) => {
      this.handleNotificationEvent({
        notificationId: event.payload.notificationId,
        actionId: event.payload.actionId,
        event: 'action',
      });
    });

    // Linux
    appWindow.listen('notification-action', (event: any) => {
      this.handleNotificationEvent({
        notificationId: event.payload.notificationId,
        actionId: event.payload.actionId,
        event: 'action',
      });
    });

    // Generic click events
    appWindow.listen('notification-clicked', (event: any) => {
      this.handleNotificationEvent({
        notificationId: event.payload.notificationId,
        event: 'clicked',
      });
    });

    appWindow.listen('notification-dismissed', (event: any) => {
      this.handleNotificationEvent({
        notificationId: event.payload.notificationId,
        event: 'dismissed',
      });
    });
  }

  /**
   * Handle notification events
   */
  private handleNotificationEvent(event: NotificationEvent): void {
    this.listeners.forEach(listener => {
      listener(event);
    });
  }

  /**
   * Send unified notification across platforms
   */
  async send(notification: UnifiedNotification): Promise<NotificationResult> {
    const id = notification.id || this.generateId();
    const notificationWithId = { ...notification, id };

    this.notificationHistory.set(id, notificationWithId);

    try {
      switch (this.platform) {
        case 'windows':
          await this.sendWindows(notificationWithId);
          break;
        case 'macos':
          await this.sendMacOS(notificationWithId);
          break;
        case 'linux':
          await this.sendLinux(notificationWithId);
          break;
        default:
          await this.sendFallback(notificationWithId);
      }

      return {
        id,
        platform: this.platform,
        delivered: true,
      };
    } catch (error) {
      console.error('Failed to send notification:', error);
      return {
        id,
        platform: this.platform,
        delivered: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * Send Windows notification
   */
  private async sendWindows(notification: UnifiedNotification): Promise<void> {
    await invoke('windows_send_toast_notification', {
      notification: {
        id: notification.id,
        title: notification.title,
        message: notification.body,
        icon: notification.icon,
        actions: notification.actions || [],
        scenario: this.mapUrgencyToScenario(notification.urgency),
        duration: notification.timeout === 0 ? 'long' : 'short',
      },
    });
  }

  /**
   * Send macOS notification
   */
  private async sendMacOS(notification: UnifiedNotification): Promise<void> {
    await invoke('macos_send_notification', {
      notification: {
        id: notification.id,
        title: notification.title,
        body: notification.body,
        subtitle: notification.category,
        sound: notification.sound !== false,
        actions: notification.actions || [],
      },
    });
  }

  /**
   * Send Linux notification
   */
  private async sendLinux(notification: UnifiedNotification): Promise<void> {
    await invoke('linux_send_notification', {
      notification: {
        summary: notification.title,
        body: notification.body,
        icon: notification.icon || 'drive-harddisk',
        urgency: notification.urgency || 'normal',
        timeout: notification.timeout ?? -1,
        actions: notification.actions?.map(a => ({ id: a.id, label: a.title })) || [],
      },
    });
  }

  /**
   * Send fallback notification using browser API
   */
  private async sendFallback(notification: UnifiedNotification): Promise<void> {
    if ('Notification' in window) {
      const permission = await Notification.requestPermission();
      if (permission === 'granted') {
        new Notification(notification.title, {
          body: notification.body,
          icon: notification.icon,
          silent: notification.silent,
        });
      }
    }
  }

  /**
   * Map urgency to Windows scenario
   */
  private mapUrgencyToScenario(
    urgency?: 'low' | 'normal' | 'critical',
  ): 'default' | 'reminder' | 'alarm' {
    switch (urgency) {
      case 'critical':
        return 'alarm';
      case 'low':
        return 'default';
      default:
        return 'reminder';
    }
  }

  /**
   * Generate unique notification ID
   */
  private generateId(): string {
    return `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Close notification
   */
  async close(notificationId: string): Promise<void> {
    try {
      switch (this.platform) {
        case 'windows':
          await invoke('windows_close_notification', { notificationId });
          break;
        case 'macos':
          await invoke('macos_close_notification', { notificationId });
          break;
        case 'linux':
          await invoke('linux_close_notification', { notificationId: parseInt(notificationId) });
          break;
      }

      this.notificationHistory.delete(notificationId);
    } catch (error) {
      console.error('Failed to close notification:', error);
    }
  }

  /**
   * Register notification event listener
   */
  onNotificationEvent(
    listenerId: string,
    handler: (event: NotificationEvent) => void,
  ): void {
    this.listeners.set(listenerId, handler);
  }

  /**
   * Unregister notification event listener
   */
  offNotificationEvent(listenerId: string): void {
    this.listeners.delete(listenerId);
  }

  /**
   * Get notification history
   */
  getHistory(): UnifiedNotification[] {
    return Array.from(this.notificationHistory.values());
  }

  /**
   * Clear notification history
   */
  clearHistory(): void {
    this.notificationHistory.clear();
  }

  /**
   * Get current platform
   */
  getPlatform(): string {
    return this.platform;
  }

  // ============================================================================
  // Convenience Methods for Common Notifications
  // ============================================================================

  /**
   * Send backup complete notification
   */
  async notifyBackupComplete(
    databaseName: string,
    size?: string,
    duration?: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Backup Complete',
      body: `${databaseName} backed up successfully${size ? ` (${size})` : ''}${duration ? ` in ${duration}` : ''}`,
      icon: 'dialog-information',
      urgency: 'normal',
      category: 'backup',
      actions: [
        { id: 'view', title: 'View Details' },
        { id: 'dismiss', title: 'Dismiss' },
      ],
    });
  }

  /**
   * Send backup failed notification
   */
  async notifyBackupFailed(
    databaseName: string,
    error: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Backup Failed',
      body: `${databaseName}: ${error}`,
      icon: 'dialog-error',
      urgency: 'critical',
      category: 'backup',
      timeout: 0, // Don't auto-dismiss
      sound: true,
      actions: [
        { id: 'retry', title: 'Retry Backup' },
        { id: 'view_logs', title: 'View Logs' },
      ],
    });
  }

  /**
   * Send backup in progress notification
   */
  async notifyBackupInProgress(
    databaseName: string,
    progress: number,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Backup In Progress',
      body: `${databaseName}: ${progress}% complete`,
      icon: 'emblem-synchronizing',
      urgency: 'low',
      category: 'backup',
      silent: true,
    });
  }

  /**
   * Send restore complete notification
   */
  async notifyRestoreComplete(
    databaseName: string,
    backupId: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Restore Complete',
      body: `${databaseName} restored from backup ${backupId}`,
      icon: 'dialog-information',
      urgency: 'normal',
      category: 'restore',
      actions: [
        { id: 'view', title: 'View Details' },
        { id: 'dismiss', title: 'Dismiss' },
      ],
    });
  }

  /**
   * Send restore failed notification
   */
  async notifyRestoreFailed(
    databaseName: string,
    error: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Restore Failed',
      body: `${databaseName}: ${error}`,
      icon: 'dialog-error',
      urgency: 'critical',
      category: 'restore',
      timeout: 0,
      sound: true,
      actions: [
        { id: 'retry', title: 'Retry Restore' },
        { id: 'view_logs', title: 'View Logs' },
      ],
    });
  }

  /**
   * Send schedule reminder notification
   */
  async notifyScheduledBackup(
    databaseName: string,
    scheduledTime: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Backup Scheduled',
      body: `${databaseName} backup starting at ${scheduledTime}`,
      icon: 'appointment-soon',
      urgency: 'normal',
      category: 'info',
      actions: [
        { id: 'start_now', title: 'Start Now' },
        { id: 'cancel', title: 'Cancel' },
      ],
    });
  }

  /**
   * Send low storage warning
   */
  async notifyLowStorage(
    availableSpace: string,
    threshold: string,
  ): Promise<NotificationResult> {
    return await this.send({
      title: 'Low Storage Warning',
      body: `Only ${availableSpace} available. Cleanup recommended (threshold: ${threshold})`,
      icon: 'dialog-warning',
      urgency: 'normal',
      category: 'alert',
      actions: [
        { id: 'cleanup', title: 'Clean Up Now' },
        { id: 'settings', title: 'Settings' },
      ],
    });
  }

  /**
   * Send general info notification
   */
  async notifyInfo(
    title: string,
    message: string,
    actions?: NotificationAction[],
  ): Promise<NotificationResult> {
    return await this.send({
      title,
      body: message,
      icon: 'dialog-information',
      urgency: 'low',
      category: 'info',
      actions,
    });
  }

  /**
   * Send general error notification
   */
  async notifyError(
    title: string,
    message: string,
    actions?: NotificationAction[],
  ): Promise<NotificationResult> {
    return await this.send({
      title,
      body: message,
      icon: 'dialog-error',
      urgency: 'critical',
      category: 'alert',
      timeout: 0,
      sound: true,
      actions,
    });
  }
}

// Export singleton instance
export const crossPlatformNotifications = new CrossPlatformNotifications();

// Export class for testing
export { CrossPlatformNotifications };
