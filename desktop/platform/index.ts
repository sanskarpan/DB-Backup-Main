/**
 * Desktop Platform Integration
 * Main entry point for all platform-specific features
 */

// Platform-specific integrations
import { windowsIntegration } from './windows';
import { macOSIntegration } from './macos';
import { linuxIntegration } from './linux';

// Cross-platform features
import { crossPlatformNotifications } from './cross/notifications';
import { crossPlatformDialogs } from './cross/dialogs';
import { crossPlatformShellIntegration } from './cross/shellIntegration';

// Export platform-specific integrations
export * from './windows';
export * from './macos';
export * from './linux';

// Export cross-platform features
export * from './cross/notifications';
export * from './cross/dialogs';
export * from './cross/shellIntegration';

// ============================================================================
// Platform Detection
// ============================================================================

export type Platform = 'windows' | 'macos' | 'linux' | 'unknown';

export function detectPlatform(): Platform {
  const userAgent = navigator.userAgent.toLowerCase();
  const platform = navigator.platform.toLowerCase();

  if (platform.indexOf('win') > -1 || userAgent.indexOf('windows') > -1) {
    return 'windows';
  } else if (platform.indexOf('mac') > -1 || userAgent.indexOf('mac') > -1) {
    return 'macos';
  } else if (platform.indexOf('linux') > -1 || userAgent.indexOf('linux') > -1) {
    return 'linux';
  }

  return 'unknown';
}

// ============================================================================
// Unified Desktop Integration Manager
// ============================================================================

export interface DesktopIntegrationConfig {
  enableNotifications?: boolean;
  enableShellIntegration?: boolean;
  enablePlatformFeatures?: boolean;
  autoInitialize?: boolean;
}

class DesktopIntegration {
  private platform: Platform;
  private initialized: boolean = false;
  private config: DesktopIntegrationConfig = {
    enableNotifications: true,
    enableShellIntegration: true,
    enablePlatformFeatures: true,
    autoInitialize: false,
  };

  // Platform-specific integrations
  public windows = windowsIntegration;
  public macos = macOSIntegration;
  public linux = linuxIntegration;

  // Cross-platform features
  public notifications = crossPlatformNotifications;
  public dialogs = crossPlatformDialogs;
  public shell = crossPlatformShellIntegration;

  constructor() {
    this.platform = detectPlatform();
  }

  /**
   * Initialize desktop integration based on platform
   */
  async initialize(config?: DesktopIntegrationConfig): Promise<void> {
    if (this.initialized) {
      console.warn('[Desktop] Already initialized');
      return;
    }

    // Merge config
    this.config = { ...this.config, ...config };

    console.log(`[Desktop] Initializing for platform: ${this.platform}`);

    try {
      // Initialize cross-platform features
      if (this.config.enableShellIntegration) {
        await this.shell.initializeDefaults();
        console.log('[Desktop] Shell integration initialized');
      }

      // Initialize platform-specific features
      if (this.config.enablePlatformFeatures) {
        switch (this.platform) {
          case 'windows':
            await this.windows.initialize();
            break;
          case 'macos':
            await this.macos.initialize();
            break;
          case 'linux':
            await this.linux.initialize();
            break;
          default:
            console.warn('[Desktop] Unknown platform, skipping platform-specific initialization');
        }
      }

      this.initialized = true;
      console.log('[Desktop] Integration initialized successfully');
    } catch (error) {
      console.error('[Desktop] Initialization failed:', error);
      throw error;
    }
  }

  /**
   * Check if desktop integration is initialized
   */
  isInitialized(): boolean {
    return this.initialized;
  }

  /**
   * Get current platform
   */
  getPlatform(): Platform {
    return this.platform;
  }

  /**
   * Get platform-specific integration manager
   */
  getPlatformIntegration() {
    switch (this.platform) {
      case 'windows':
        return this.windows;
      case 'macos':
        return this.macos;
      case 'linux':
        return this.linux;
      default:
        return null;
    }
  }

  /**
   * Check if running on Windows
   */
  isWindows(): boolean {
    return this.platform === 'windows';
  }

  /**
   * Check if running on macOS
   */
  isMacOS(): boolean {
    return this.platform === 'macos';
  }

  /**
   * Check if running on Linux
   */
  isLinux(): boolean {
    return this.platform === 'linux';
  }

  /**
   * Get platform capabilities
   */
  async getCapabilities(): Promise<{
    platform: Platform;
    notifications: boolean;
    shellIntegration: boolean;
    platformFeatures: string[];
  }> {
    const platformFeatures: string[] = [];

    switch (this.platform) {
      case 'windows':
        platformFeatures.push(
          'action-center',
          'live-tiles',
          'jump-list',
          'thumbnail-toolbar',
        );
        break;
      case 'macos':
        platformFeatures.push(
          'control-center',
          'menu-bar',
          'quick-look',
          'finder-extension',
        );
        if (await this.macos.hasTouchBar()) {
          platformFeatures.push('touch-bar');
        }
        break;
      case 'linux':
        const de = this.linux.getDesktopEnvironment();
        if (de === 'gnome') {
          platformFeatures.push('gnome-shell-extension');
        } else if (de === 'kde') {
          platformFeatures.push('kde-plasma-widget');
        }
        platformFeatures.push('system-tray', 'dbus-notifications');
        break;
    }

    return {
      platform: this.platform,
      notifications: this.config.enableNotifications ?? true,
      shellIntegration: this.config.enableShellIntegration ?? true,
      platformFeatures,
    };
  }

  /**
   * Send notification using best available method for platform
   */
  async sendNotification(
    title: string,
    message: string,
    options?: {
      urgency?: 'low' | 'normal' | 'critical';
      actions?: Array<{ id: string; title: string }>;
    },
  ): Promise<void> {
    await this.notifications.send({
      title,
      body: message,
      urgency: options?.urgency,
      actions: options?.actions,
    });
  }

  /**
   * Update platform-specific status indicator
   */
  async updateStatus(status: {
    state: 'idle' | 'backing_up' | 'restoring' | 'error';
    message?: string;
    progress?: number;
  }): Promise<void> {
    switch (this.platform) {
      case 'windows':
        // Update Live Tile
        if (status.progress !== undefined) {
          await this.windows.liveTiles.updateWithProgress(
            status.message || 'Backup',
            status.progress,
          );
        }
        break;

      case 'macos':
        // Update Control Center widget
        await this.macos.controlCenter.updateWithStatus({
          lastBackup: status.message,
          nextScheduled: undefined,
          totalBackups: 0,
          status: status.state,
        });

        // Update menu bar
        await this.macos.menuBar.updateMenuItem('status', {
          label: `Status: ${status.state}`,
        });
        break;

      case 'linux':
        // Update system tray
        const icon =
          status.state === 'error'
            ? 'dialog-error'
            : status.state === 'backing_up'
            ? 'emblem-synchronizing'
            : 'drive-harddisk';

        await this.linux.systemTray.updateIcon(
          icon,
          `DB Backup - ${status.state}`,
        );

        // Update system tray menu
        await this.linux.systemTray.updateMenu([
          {
            id: 'status',
            type: 'item',
            label: `Status: ${status.state}`,
            enabled: false,
          },
          { id: 'sep1', type: 'separator' },
          {
            id: 'backup_now',
            type: 'item',
            label: 'Backup Now',
            icon: 'document-save',
          },
        ]);
        break;
    }
  }

  /**
   * Show backup complete notification
   */
  async notifyBackupComplete(
    databaseName: string,
    size?: string,
    duration?: string,
  ): Promise<void> {
    await this.notifications.notifyBackupComplete(databaseName, size, duration);

    // Update platform status
    await this.updateStatus({
      state: 'idle',
      message: `${databaseName} backed up${size ? ` (${size})` : ''}`,
    });
  }

  /**
   * Show backup failed notification
   */
  async notifyBackupFailed(databaseName: string, error: string): Promise<void> {
    await this.notifications.notifyBackupFailed(databaseName, error);

    // Update platform status
    await this.updateStatus({
      state: 'error',
      message: `Backup failed: ${error}`,
    });
  }

  /**
   * Update backup progress
   */
  async updateBackupProgress(
    databaseName: string,
    progress: number,
  ): Promise<void> {
    await this.updateStatus({
      state: 'backing_up',
      message: databaseName,
      progress,
    });
  }

  /**
   * Cleanup and remove integrations
   */
  async cleanup(): Promise<void> {
    try {
      switch (this.platform) {
        case 'windows':
          await this.windows.liveTiles.clearTile();
          await this.windows.jumpList.clearJumpList();
          break;

        case 'macos':
          await this.macos.controlCenter.clearWidget();
          await this.macos.touchBar.clear();
          break;

        case 'linux':
          await this.linux.systemTray.removeTray();
          if (this.linux.getDesktopEnvironment() === 'gnome') {
            await this.linux.gnome.disable();
          }
          break;
      }

      this.initialized = false;
      console.log('[Desktop] Cleanup complete');
    } catch (error) {
      console.error('[Desktop] Cleanup failed:', error);
    }
  }
}

// Export singleton instance
export const desktopIntegration = new DesktopIntegration();

// Export class for testing
export { DesktopIntegration };

// ============================================================================
// Convenience Functions
// ============================================================================

/**
 * Initialize desktop integration with default config
 */
export async function initializeDesktopIntegration(
  config?: DesktopIntegrationConfig,
): Promise<void> {
  await desktopIntegration.initialize(config);
}

/**
 * Get current platform
 */
export function getCurrentPlatform(): Platform {
  return desktopIntegration.getPlatform();
}

/**
 * Check if desktop integration is ready
 */
export function isDesktopIntegrationReady(): boolean {
  return desktopIntegration.isInitialized();
}

/**
 * Send cross-platform notification
 */
export async function sendDesktopNotification(
  title: string,
  message: string,
  options?: {
    urgency?: 'low' | 'normal' | 'critical';
    actions?: Array<{ id: string; title: string }>;
  },
): Promise<void> {
  await desktopIntegration.sendNotification(title, message, options);
}

/**
 * Show cross-platform dialog
 */
export async function showDesktopDialog(
  title: string,
  message: string,
  type: 'info' | 'warning' | 'error' = 'info',
): Promise<void> {
  switch (type) {
    case 'error':
      await desktopIntegration.dialogs.showError(message, title);
      break;
    case 'warning':
      await desktopIntegration.dialogs.showWarning(message, title);
      break;
    default:
      await desktopIntegration.dialogs.showInfo(message, title);
  }
}

/**
 * Get platform capabilities
 */
export async function getDesktopCapabilities() {
  return await desktopIntegration.getCapabilities();
}
