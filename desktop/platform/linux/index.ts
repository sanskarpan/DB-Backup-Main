/**
 * Linux Platform Integration
 * Provides Linux-specific features: GNOME Shell Extension, KDE Plasma Widget, System Tray
 */

import { invoke } from '@tauri-apps/api/tauri';
import { appWindow } from '@tauri-apps/api/window';

// ============================================================================
// GNOME Shell Extension
// ============================================================================

export interface GnomeExtension {
  uuid: string;
  name: string;
  description: string;
  version: string;
}

export interface GnomePanelButton {
  id: string;
  label: string;
  icon?: string;
  menu?: GnomeMenuItem[];
}

export interface GnomeMenuItem {
  id: string;
  type: 'item' | 'separator' | 'submenu';
  label?: string;
  icon?: string;
  sensitive?: boolean;
  submenu?: GnomeMenuItem[];
}

class GnomeShellExtension {
  private extensionUuid = 'db-backup@example.com';

  /**
   * Install GNOME Shell extension
   */
  async install(): Promise<void> {
    try {
      await invoke('linux_install_gnome_extension', {
        uuid: this.extensionUuid,
      });
    } catch (error) {
      console.error('Failed to install GNOME extension:', error);
      throw error;
    }
  }

  /**
   * Enable GNOME Shell extension
   */
  async enable(): Promise<void> {
    try {
      await invoke('linux_enable_gnome_extension', {
        uuid: this.extensionUuid,
      });
    } catch (error) {
      console.error('Failed to enable GNOME extension:', error);
      throw error;
    }
  }

  /**
   * Disable GNOME Shell extension
   */
  async disable(): Promise<void> {
    try {
      await invoke('linux_disable_gnome_extension', {
        uuid: this.extensionUuid,
      });
    } catch (error) {
      console.error('Failed to disable GNOME extension:', error);
    }
  }

  /**
   * Update panel button
   */
  async updatePanelButton(button: GnomePanelButton): Promise<void> {
    try {
      await invoke('linux_update_gnome_panel_button', { button });
    } catch (error) {
      console.error('Failed to update GNOME panel button:', error);
      throw error;
    }
  }

  /**
   * Set default panel menu
   */
  async setDefaultMenu(): Promise<void> {
    await this.updatePanelButton({
      id: 'db-backup-panel',
      label: 'DB Backup',
      icon: 'drive-harddisk-symbolic',
      menu: [
        {
          id: 'status',
          type: 'item',
          label: 'Status: Idle',
          sensitive: false,
        },
        { id: 'sep1', type: 'separator' },
        {
          id: 'backup_now',
          type: 'item',
          label: 'Backup Now',
          icon: 'document-save-symbolic',
        },
        {
          id: 'recent',
          type: 'submenu',
          label: 'Recent Backups',
          submenu: [
            { id: 'no_backups', type: 'item', label: 'No recent backups' },
          ],
        },
        { id: 'sep2', type: 'separator' },
        {
          id: 'preferences',
          type: 'item',
          label: 'Preferences',
          icon: 'preferences-system-symbolic',
        },
        {
          id: 'quit',
          type: 'item',
          label: 'Quit',
          icon: 'application-exit-symbolic',
        },
      ],
    });
  }

  /**
   * Show notification from extension
   */
  async showNotification(title: string, message: string): Promise<void> {
    try {
      await invoke('linux_gnome_show_notification', { title, message });
    } catch (error) {
      console.error('Failed to show GNOME notification:', error);
    }
  }

  /**
   * Handle menu item activation
   */
  onMenuItemActivated(handler: (itemId: string) => void): void {
    appWindow.listen('gnome-menu-item-activated', (event: any) => {
      handler(event.payload.itemId);
    });
  }

  /**
   * Check if GNOME is running
   */
  async isGnomeRunning(): Promise<boolean> {
    try {
      return await invoke('linux_is_gnome_running');
    } catch (error) {
      return false;
    }
  }
}

// ============================================================================
// KDE Plasma Widget
// ============================================================================

export interface PlasmaWidget {
  id: string;
  name: string;
  icon: string;
  config?: Record<string, any>;
}

export interface PlasmaWidgetContent {
  title: string;
  subtitle?: string;
  items?: PlasmaWidgetItem[];
  actions?: PlasmaWidgetAction[];
}

export interface PlasmaWidgetItem {
  id: string;
  text: string;
  icon?: string;
  subText?: string;
}

export interface PlasmaWidgetAction {
  id: string;
  text: string;
  icon?: string;
}

class KDEPlasmaWidget {
  private widgetId = 'com.example.dbbackup';

  /**
   * Install Plasma widget
   */
  async install(): Promise<void> {
    try {
      await invoke('linux_install_plasma_widget', { widgetId: this.widgetId });
    } catch (error) {
      console.error('Failed to install Plasma widget:', error);
      throw error;
    }
  }

  /**
   * Update widget content
   */
  async updateContent(content: PlasmaWidgetContent): Promise<void> {
    try {
      await invoke('linux_update_plasma_widget', {
        widgetId: this.widgetId,
        content,
      });
    } catch (error) {
      console.error('Failed to update Plasma widget:', error);
      throw error;
    }
  }

  /**
   * Update with backup status
   */
  async updateWithStatus(status: {
    lastBackup?: string;
    nextScheduled?: string;
    totalBackups: number;
  }): Promise<void> {
    await this.updateContent({
      title: 'DB Backup',
      subtitle: status.lastBackup
        ? `Last: ${status.lastBackup}`
        : 'No backups yet',
      items: [
        {
          id: 'total',
          text: `${status.totalBackups} backups`,
          icon: 'database',
        },
        status.nextScheduled
          ? {
              id: 'next',
              text: `Next: ${status.nextScheduled}`,
              icon: 'clock',
            }
          : null,
      ].filter(Boolean) as PlasmaWidgetItem[],
      actions: [
        { id: 'backup_now', text: 'Backup Now', icon: 'document-save' },
        { id: 'view_backups', text: 'View Backups', icon: 'folder' },
      ],
    });
  }

  /**
   * Handle widget action
   */
  onAction(handler: (actionId: string) => void): void {
    appWindow.listen('plasma-widget-action', (event: any) => {
      handler(event.payload.actionId);
    });
  }

  /**
   * Check if KDE Plasma is running
   */
  async isKDERunning(): Promise<boolean> {
    try {
      return await invoke('linux_is_kde_running');
    } catch (error) {
      return false;
    }
  }
}

// ============================================================================
// System Tray Indicator (Cross-DE)
// ============================================================================

export interface SystemTrayIcon {
  icon: string;
  tooltip: string;
  menu: SystemTrayMenuItem[];
}

export interface SystemTrayMenuItem {
  id: string;
  type: 'item' | 'separator' | 'submenu' | 'checkbox';
  label?: string;
  icon?: string;
  enabled?: boolean;
  checked?: boolean;
  submenu?: SystemTrayMenuItem[];
}

class LinuxSystemTray {
  private menuItems: SystemTrayMenuItem[] = [];

  /**
   * Create system tray icon
   */
  async createTray(icon: SystemTrayIcon): Promise<void> {
    try {
      this.menuItems = icon.menu;
      await invoke('linux_create_system_tray', { icon });
    } catch (error) {
      console.error('Failed to create system tray:', error);
      throw error;
    }
  }

  /**
   * Update tray icon
   */
  async updateIcon(icon: string, tooltip?: string): Promise<void> {
    try {
      await invoke('linux_update_tray_icon', { icon, tooltip });
    } catch (error) {
      console.error('Failed to update tray icon:', error);
    }
  }

  /**
   * Update tray menu
   */
  async updateMenu(menu: SystemTrayMenuItem[]): Promise<void> {
    try {
      this.menuItems = menu;
      await invoke('linux_update_tray_menu', { menu });
    } catch (error) {
      console.error('Failed to update tray menu:', error);
      throw error;
    }
  }

  /**
   * Set default tray menu
   */
  async setDefaultTray(): Promise<void> {
    await this.createTray({
      icon: 'drive-harddisk',
      tooltip: 'DB Backup - Idle',
      menu: [
        {
          id: 'status',
          type: 'item',
          label: 'Status: Idle',
          enabled: false,
        },
        { id: 'sep1', type: 'separator' },
        {
          id: 'backup_now',
          type: 'item',
          label: 'Backup Now',
          icon: 'document-save',
        },
        {
          id: 'restore',
          type: 'item',
          label: 'Restore Database',
          icon: 'edit-undo',
        },
        { id: 'sep2', type: 'separator' },
        {
          id: 'auto_backup',
          type: 'checkbox',
          label: 'Auto Backup',
          checked: false,
        },
        { id: 'sep3', type: 'separator' },
        {
          id: 'preferences',
          type: 'item',
          label: 'Preferences',
          icon: 'preferences-system',
        },
        {
          id: 'about',
          type: 'item',
          label: 'About',
          icon: 'help-about',
        },
        { id: 'sep4', type: 'separator' },
        {
          id: 'quit',
          type: 'item',
          label: 'Quit',
          icon: 'application-exit',
        },
      ],
    });
  }

  /**
   * Show notification from tray
   */
  async showNotification(title: string, message: string, icon?: string): Promise<void> {
    try {
      await invoke('linux_tray_show_notification', { title, message, icon });
    } catch (error) {
      console.error('Failed to show tray notification:', error);
    }
  }

  /**
   * Handle tray menu click
   */
  onMenuClick(handler: (itemId: string) => void): void {
    appWindow.listen('tray-menu-click', (event: any) => {
      handler(event.payload.itemId);
    });
  }

  /**
   * Handle tray icon click
   */
  onTrayClick(handler: () => void): void {
    appWindow.listen('tray-icon-click', () => {
      handler();
    });
  }

  /**
   * Remove tray icon
   */
  async removeTray(): Promise<void> {
    try {
      await invoke('linux_remove_system_tray');
    } catch (error) {
      console.error('Failed to remove system tray:', error);
    }
  }
}

// ============================================================================
// Desktop Notifications (D-Bus)
// ============================================================================

export interface LinuxNotification {
  summary: string;
  body: string;
  icon?: string;
  urgency?: 'low' | 'normal' | 'critical';
  timeout?: number; // milliseconds, -1 for default, 0 for never
  actions?: NotificationAction[];
  hints?: Record<string, any>;
}

export interface NotificationAction {
  id: string;
  label: string;
}

class LinuxNotifications {
  /**
   * Send desktop notification via D-Bus
   */
  async send(notification: LinuxNotification): Promise<number> {
    try {
      return await invoke('linux_send_notification', { notification });
    } catch (error) {
      console.error('Failed to send Linux notification:', error);
      throw error;
    }
  }

  /**
   * Close notification
   */
  async close(notificationId: number): Promise<void> {
    try {
      await invoke('linux_close_notification', { notificationId });
    } catch (error) {
      console.error('Failed to close Linux notification:', error);
    }
  }

  /**
   * Handle notification action
   */
  onAction(handler: (notificationId: number, actionId: string) => void): void {
    appWindow.listen('notification-action', (event: any) => {
      handler(event.payload.notificationId, event.payload.actionId);
    });
  }

  /**
   * Send backup complete notification
   */
  async notifyBackupComplete(databaseName: string): Promise<number> {
    return await this.send({
      summary: 'Backup Complete',
      body: `${databaseName} backed up successfully`,
      icon: 'dialog-information',
      urgency: 'normal',
      actions: [
        { id: 'view', label: 'View Details' },
        { id: 'dismiss', label: 'Dismiss' },
      ],
    });
  }

  /**
   * Send backup failed notification
   */
  async notifyBackupFailed(databaseName: string, error: string): Promise<number> {
    return await this.send({
      summary: 'Backup Failed',
      body: `${databaseName}: ${error}`,
      icon: 'dialog-error',
      urgency: 'critical',
      timeout: 0, // Don't auto-dismiss
      actions: [
        { id: 'retry', label: 'Retry' },
        { id: 'view_logs', label: 'View Logs' },
      ],
    });
  }
}

// ============================================================================
// Linux Integration Manager
// ============================================================================

class LinuxIntegration {
  public gnome: GnomeShellExtension;
  public kde: KDEPlasmaWidget;
  public systemTray: LinuxSystemTray;
  public notifications: LinuxNotifications;
  private desktopEnvironment: 'gnome' | 'kde' | 'other' | null = null;

  constructor() {
    this.gnome = new GnomeShellExtension();
    this.kde = new KDEPlasmaWidget();
    this.systemTray = new LinuxSystemTray();
    this.notifications = new LinuxNotifications();
  }

  /**
   * Initialize Linux features based on desktop environment
   */
  async initialize(): Promise<void> {
    try {
      // Detect desktop environment
      this.desktopEnvironment = await this.detectDesktopEnvironment();

      // Initialize based on DE
      switch (this.desktopEnvironment) {
        case 'gnome':
          await this.initializeGnome();
          break;
        case 'kde':
          await this.initializeKDE();
          break;
        default:
          await this.initializeGeneric();
          break;
      }

      console.log(
        `[Linux] Integration initialized for ${this.desktopEnvironment}`,
      );
    } catch (error) {
      console.error('[Linux] Initialization error:', error);
    }
  }

  /**
   * Initialize GNOME-specific features
   */
  private async initializeGnome(): Promise<void> {
    try {
      await this.gnome.install();
      await this.gnome.enable();
      await this.gnome.setDefaultMenu();
    } catch (error) {
      console.error('[Linux] GNOME initialization failed:', error);
      // Fallback to system tray
      await this.systemTray.setDefaultTray();
    }
  }

  /**
   * Initialize KDE-specific features
   */
  private async initializeKDE(): Promise<void> {
    try {
      await this.kde.install();
    } catch (error) {
      console.error('[Linux] KDE initialization failed:', error);
      // Fallback to system tray
      await this.systemTray.setDefaultTray();
    }
  }

  /**
   * Initialize generic Linux features (system tray)
   */
  private async initializeGeneric(): Promise<void> {
    await this.systemTray.setDefaultTray();
  }

  /**
   * Detect desktop environment
   */
  async detectDesktopEnvironment(): Promise<'gnome' | 'kde' | 'other'> {
    try {
      const isGnome = await this.gnome.isGnomeRunning();
      if (isGnome) return 'gnome';

      const isKDE = await this.kde.isKDERunning();
      if (isKDE) return 'kde';

      return 'other';
    } catch (error) {
      return 'other';
    }
  }

  /**
   * Check if running on Linux
   */
  static isLinux(): boolean {
    return navigator.platform.indexOf('Linux') > -1;
  }

  /**
   * Get desktop environment name
   */
  getDesktopEnvironment(): 'gnome' | 'kde' | 'other' | null {
    return this.desktopEnvironment;
  }

  /**
   * Get Linux distribution info
   */
  async getDistributionInfo(): Promise<{
    name: string;
    version: string;
    id: string;
  }> {
    try {
      return await invoke('linux_get_distribution_info');
    } catch (error) {
      return { name: 'unknown', version: 'unknown', id: 'unknown' };
    }
  }
}

// Export singleton instance
export const linuxIntegration = new LinuxIntegration();

// Export individual services
export const gnomeShell = linuxIntegration.gnome;
export const kdePlasma = linuxIntegration.kde;
export const linuxSystemTray = linuxIntegration.systemTray;
export const linuxNotifications = linuxIntegration.notifications;
