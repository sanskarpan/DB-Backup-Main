/**
 * Windows Platform Integration
 * Provides Windows-specific features: Action Center, Live Tiles, Jump Lists, Thumbnail Toolbar
 */

import { invoke } from '@tauri-apps/api/tauri';
import { appWindow } from '@tauri-apps/api/window';

// ============================================================================
// Action Center Integration
// ============================================================================

export interface WindowsNotification {
  title: string;
  message: string;
  icon?: string;
  actions?: NotificationAction[];
  scenario?: 'default' | 'reminder' | 'alarm' | 'incomingCall';
  duration?: 'short' | 'long';
}

export interface NotificationAction {
  action: string;
  title: string;
  arguments?: string;
}

class WindowsActionCenter {
  /**
   * Send notification to Windows Action Center
   */
  async sendNotification(notification: WindowsNotification): Promise<void> {
    try {
      await invoke('windows_send_toast_notification', {
        notification: {
          title: notification.title,
          message: notification.message,
          icon: notification.icon,
          actions: notification.actions || [],
          scenario: notification.scenario || 'default',
          duration: notification.duration || 'short',
        },
      });
    } catch (error) {
      console.error('Failed to send Windows notification:', error);
      throw error;
    }
  }

  /**
   * Send backup complete notification
   */
  async notifyBackupComplete(databaseName: string, size: string): Promise<void> {
    await this.sendNotification({
      title: 'Backup Complete',
      message: `${databaseName} backed up successfully (${size})`,
      scenario: 'default',
      actions: [
        { action: 'view', title: 'View Details' },
        { action: 'dismiss', title: 'Dismiss' },
      ],
    });
  }

  /**
   * Send backup failed notification
   */
  async notifyBackupFailed(databaseName: string, error: string): Promise<void> {
    await this.sendNotification({
      title: 'Backup Failed',
      message: `${databaseName}: ${error}`,
      scenario: 'reminder',
      duration: 'long',
      actions: [
        { action: 'retry', title: 'Retry Backup' },
        { action: 'view', title: 'View Logs' },
      ],
    });
  }

  /**
   * Send scheduled backup reminder
   */
  async sendScheduleReminder(databaseName: string, time: string): Promise<void> {
    await this.sendNotification({
      title: 'Backup Scheduled',
      message: `${databaseName} backup starting at ${time}`,
      scenario: 'reminder',
      actions: [
        { action: 'start_now', title: 'Start Now' },
        { action: 'cancel', title: 'Cancel' },
      ],
    });
  }
}

// ============================================================================
// Live Tiles (Windows 10/11)
// ============================================================================

export interface LiveTileData {
  type: 'small' | 'medium' | 'wide' | 'large';
  title: string;
  content: string;
  backgroundImage?: string;
  badge?: number | 'activity' | 'alert' | 'available' | 'busy';
}

class WindowsLiveTiles {
  /**
   * Update Live Tile
   */
  async updateTile(tileData: LiveTileData): Promise<void> {
    try {
      await invoke('windows_update_live_tile', { tileData });
    } catch (error) {
      console.error('Failed to update Live Tile:', error);
      throw error;
    }
  }

  /**
   * Update tile with backup stats
   */
  async updateWithStats(stats: {
    totalBackups: number;
    lastBackup: string;
    nextScheduled?: string;
  }): Promise<void> {
    await this.updateTile({
      type: 'medium',
      title: 'DB Backup',
      content: `${stats.totalBackups} backups\nLast: ${stats.lastBackup}`,
      badge: stats.totalBackups,
    });
  }

  /**
   * Update tile with progress
   */
  async updateWithProgress(
    operation: string,
    progress: number,
  ): Promise<void> {
    await this.updateTile({
      type: 'medium',
      title: 'DB Backup',
      content: `${operation}\n${progress}% complete`,
      badge: 'activity',
    });
  }

  /**
   * Clear Live Tile
   */
  async clearTile(): Promise<void> {
    try {
      await invoke('windows_clear_live_tile');
    } catch (error) {
      console.error('Failed to clear Live Tile:', error);
    }
  }

  /**
   * Enable badge notifications
   */
  async setBadge(value: number | string): Promise<void> {
    try {
      await invoke('windows_set_tile_badge', { value });
    } catch (error) {
      console.error('Failed to set tile badge:', error);
    }
  }
}

// ============================================================================
// Jump Lists (Taskbar)
// ============================================================================

export interface JumpListItem {
  type: 'task' | 'recent' | 'frequent';
  title: string;
  description?: string;
  path?: string;
  arguments?: string;
  iconPath?: string;
}

class WindowsJumpList {
  /**
   * Update Jump List
   */
  async updateJumpList(items: JumpListItem[]): Promise<void> {
    try {
      await invoke('windows_update_jump_list', { items });
    } catch (error) {
      console.error('Failed to update Jump List:', error);
      throw error;
    }
  }

  /**
   * Add recent backup to Jump List
   */
  async addRecentBackup(backupName: string, backupId: string): Promise<void> {
    await this.updateJumpList([
      {
        type: 'recent',
        title: backupName,
        description: 'Open recent backup',
        arguments: `--open-backup=${backupId}`,
      },
    ]);
  }

  /**
   * Set default tasks in Jump List
   */
  async setDefaultTasks(): Promise<void> {
    await this.updateJumpList([
      {
        type: 'task',
        title: 'New Backup',
        description: 'Create a new backup',
        arguments: '--new-backup',
      },
      {
        type: 'task',
        title: 'Restore Database',
        description: 'Restore from backup',
        arguments: '--restore',
      },
      {
        type: 'task',
        title: 'View Backups',
        description: 'View all backups',
        arguments: '--view-backups',
      },
      {
        type: 'task',
        title: 'Settings',
        description: 'Open settings',
        arguments: '--settings',
      },
    ]);
  }

  /**
   * Clear Jump List
   */
  async clearJumpList(): Promise<void> {
    try {
      await invoke('windows_clear_jump_list');
    } catch (error) {
      console.error('Failed to clear Jump List:', error);
    }
  }
}

// ============================================================================
// Thumbnail Toolbar Buttons
// ============================================================================

export interface ThumbnailButton {
  id: string;
  tooltip: string;
  icon: string;
  flags?: ('enabled' | 'disabled' | 'dismissonclick' | 'nobackground' | 'hidden' | 'noninteractive')[];
}

class WindowsThumbnailToolbar {
  private buttons: Map<string, ThumbnailButton> = new Map();

  /**
   * Add thumbnail toolbar buttons
   */
  async setButtons(buttons: ThumbnailButton[]): Promise<void> {
    try {
      buttons.forEach(btn => this.buttons.set(btn.id, btn));
      await invoke('windows_set_thumbnail_buttons', { buttons });
    } catch (error) {
      console.error('Failed to set thumbnail buttons:', error);
      throw error;
    }
  }

  /**
   * Update button state
   */
  async updateButton(buttonId: string, updates: Partial<ThumbnailButton>): Promise<void> {
    const button = this.buttons.get(buttonId);
    if (!button) {
      throw new Error(`Button ${buttonId} not found`);
    }

    const updatedButton = { ...button, ...updates };
    this.buttons.set(buttonId, updatedButton);

    await invoke('windows_update_thumbnail_button', {
      buttonId,
      button: updatedButton,
    });
  }

  /**
   * Set default backup control buttons
   */
  async setBackupControls(): Promise<void> {
    await this.setButtons([
      {
        id: 'backup_now',
        tooltip: 'Backup Now',
        icon: 'icons/backup.ico',
        flags: ['enabled'],
      },
      {
        id: 'pause',
        tooltip: 'Pause Backup',
        icon: 'icons/pause.ico',
        flags: ['enabled'],
      },
      {
        id: 'stop',
        tooltip: 'Stop Backup',
        icon: 'icons/stop.ico',
        flags: ['enabled'],
      },
    ]);
  }

  /**
   * Handle thumbnail button click
   */
  onButtonClick(handler: (buttonId: string) => void): void {
    // Register event listener in Tauri
    appWindow.listen('thumbnail-button-click', (event: any) => {
      handler(event.payload.buttonId);
    });
  }
}

// ============================================================================
// Windows Integration Manager
// ============================================================================

class WindowsIntegration {
  public actionCenter: WindowsActionCenter;
  public liveTiles: WindowsLiveTiles;
  public jumpList: WindowsJumpList;
  public thumbnailToolbar: WindowsThumbnailToolbar;

  constructor() {
    this.actionCenter = new WindowsActionCenter();
    this.liveTiles = new WindowsLiveTiles();
    this.jumpList = new WindowsJumpList();
    this.thumbnailToolbar = new WindowsThumbnailToolbar();
  }

  /**
   * Initialize all Windows features
   */
  async initialize(): Promise<void> {
    try {
      // Set default Jump List tasks
      await this.jumpList.setDefaultTasks();

      // Set default thumbnail toolbar buttons
      await this.thumbnailToolbar.setBackupControls();

      console.log('[Windows] Integration initialized');
    } catch (error) {
      console.error('[Windows] Initialization error:', error);
    }
  }

  /**
   * Check if running on Windows
   */
  static isWindows(): boolean {
    return navigator.platform.indexOf('Win') > -1;
  }

  /**
   * Get Windows version
   */
  async getWindowsVersion(): Promise<string> {
    try {
      return await invoke('windows_get_version');
    } catch (error) {
      return 'unknown';
    }
  }

  /**
   * Check if Windows 10/11 features are available
   */
  async supportsModernFeatures(): Promise<boolean> {
    try {
      const version = await this.getWindowsVersion();
      return version.startsWith('10') || version.startsWith('11');
    } catch (error) {
      return false;
    }
  }
}

// Export singleton instance
export const windowsIntegration = new WindowsIntegration();

// Export individual services
export const windowsActionCenter = windowsIntegration.actionCenter;
export const windowsLiveTiles = windowsIntegration.liveTiles;
export const windowsJumpList = windowsIntegration.jumpList;
export const windowsThumbnailToolbar = windowsIntegration.thumbnailToolbar;
