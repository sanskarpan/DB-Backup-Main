/**
 * macOS Platform Integration
 * Provides macOS-specific features: Control Center, Touch Bar, Menu Bar, Quick Look, Finder Extensions
 */

import { invoke } from '@tauri-apps/api/tauri';
import { appWindow } from '@tauri-apps/api/window';

// ============================================================================
// Control Center Widget
// ============================================================================

export interface ControlCenterWidget {
  title: string;
  subtitle?: string;
  icon?: string;
  actions?: WidgetAction[];
  data?: any;
}

export interface WidgetAction {
  id: string;
  title: string;
  icon?: string;
}

class MacOSControlCenter {
  /**
   * Update Control Center widget
   */
  async updateWidget(widget: ControlCenterWidget): Promise<void> {
    try {
      await invoke('macos_update_control_center_widget', { widget });
    } catch (error) {
      console.error('Failed to update Control Center widget:', error);
      throw error;
    }
  }

  /**
   * Update widget with backup status
   */
  async updateWithStatus(status: {
    lastBackup?: string;
    nextScheduled?: string;
    totalBackups: number;
    status: 'idle' | 'backing_up' | 'error';
  }): Promise<void> {
    await this.updateWidget({
      title: 'DB Backup',
      subtitle: status.lastBackup
        ? `Last backup: ${status.lastBackup}`
        : 'No backups yet',
      icon: status.status === 'error' ? 'error' : 'backup',
      actions: [
        { id: 'backup_now', title: 'Backup Now' },
        { id: 'view_backups', title: 'View Backups' },
      ],
      data: status,
    });
  }

  /**
   * Clear widget
   */
  async clearWidget(): Promise<void> {
    try {
      await invoke('macos_clear_control_center_widget');
    } catch (error) {
      console.error('Failed to clear Control Center widget:', error);
    }
  }
}

// ============================================================================
// Touch Bar Support (MacBook Pro)
// ============================================================================

export interface TouchBarButton {
  id: string;
  label: string;
  icon?: string;
  backgroundColor?: string;
  enabled?: boolean;
}

export interface TouchBarScrubber {
  id: string;
  items: { label: string; icon?: string }[];
  selectedIndex?: number;
}

class MacOSTouchBar {
  private buttons: Map<string, TouchBarButton> = new Map();

  /**
   * Set Touch Bar layout
   */
  async setLayout(items: (TouchBarButton | TouchBarScrubber)[]): Promise<void> {
    try {
      items.forEach(item => {
        if ('label' in item) {
          this.buttons.set(item.id, item);
        }
      });

      await invoke('macos_set_touch_bar_layout', { items });
    } catch (error) {
      console.error('Failed to set Touch Bar layout:', error);
      throw error;
    }
  }

  /**
   * Set backup control Touch Bar
   */
  async setBackupControls(): Promise<void> {
    await this.setLayout([
      {
        id: 'backup_now',
        label: 'Backup Now',
        icon: 'backup',
        backgroundColor: '#007AFF',
      },
      {
        id: 'pause',
        label: 'Pause',
        icon: 'pause',
      },
      {
        id: 'stop',
        label: 'Stop',
        icon: 'stop',
        backgroundColor: '#FF3B30',
      },
      {
        id: 'databases',
        items: [
          { label: 'Production DB' },
          { label: 'Development DB' },
          { label: 'Staging DB' },
        ],
      } as TouchBarScrubber,
    ]);
  }

  /**
   * Update button state
   */
  async updateButton(
    buttonId: string,
    updates: Partial<TouchBarButton>,
  ): Promise<void> {
    const button = this.buttons.get(buttonId);
    if (!button) {
      throw new Error(`Touch Bar button ${buttonId} not found`);
    }

    const updatedButton = { ...button, ...updates };
    this.buttons.set(buttonId, updatedButton);

    await invoke('macos_update_touch_bar_button', {
      buttonId,
      button: updatedButton,
    });
  }

  /**
   * Handle Touch Bar button press
   */
  onButtonPress(handler: (buttonId: string) => void): void {
    appWindow.listen('touch-bar-button-press', (event: any) => {
      handler(event.payload.buttonId);
    });
  }

  /**
   * Clear Touch Bar
   */
  async clear(): Promise<void> {
    try {
      await invoke('macos_clear_touch_bar');
    } catch (error) {
      console.error('Failed to clear Touch Bar:', error);
    }
  }
}

// ============================================================================
// Menu Bar Extra (Status Bar App)
// ============================================================================

export interface MenuBarItem {
  id: string;
  type: 'item' | 'separator' | 'submenu';
  label?: string;
  icon?: string;
  shortcut?: string;
  enabled?: boolean;
  checked?: boolean;
  submenu?: MenuBarItem[];
}

class MacOSMenuBar {
  private menuItems: MenuBarItem[] = [];

  /**
   * Update menu bar extra
   */
  async updateMenu(items: MenuBarItem[]): Promise<void> {
    try {
      this.menuItems = items;
      await invoke('macos_update_menu_bar', { items });
    } catch (error) {
      console.error('Failed to update menu bar:', error);
      throw error;
    }
  }

  /**
   * Set menu bar icon
   */
  async setIcon(icon: string, template?: boolean): Promise<void> {
    try {
      await invoke('macos_set_menu_bar_icon', { icon, template });
    } catch (error) {
      console.error('Failed to set menu bar icon:', error);
    }
  }

  /**
   * Set menu bar title
   */
  async setTitle(title: string): Promise<void> {
    try {
      await invoke('macos_set_menu_bar_title', { title });
    } catch (error) {
      console.error('Failed to set menu bar title:', error);
    }
  }

  /**
   * Set default menu structure
   */
  async setDefaultMenu(): Promise<void> {
    await this.updateMenu([
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
        shortcut: 'Cmd+B',
      },
      {
        id: 'recent_backups',
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
        label: 'Preferences...',
        shortcut: 'Cmd+,',
      },
      { id: 'sep3', type: 'separator' },
      {
        id: 'quit',
        type: 'item',
        label: 'Quit DB Backup',
        shortcut: 'Cmd+Q',
      },
    ]);
  }

  /**
   * Update menu item
   */
  async updateMenuItem(
    itemId: string,
    updates: Partial<MenuBarItem>,
  ): Promise<void> {
    try {
      await invoke('macos_update_menu_bar_item', { itemId, updates });
    } catch (error) {
      console.error('Failed to update menu bar item:', error);
    }
  }

  /**
   * Handle menu item click
   */
  onMenuItemClick(handler: (itemId: string) => void): void {
    appWindow.listen('menu-bar-item-click', (event: any) => {
      handler(event.payload.itemId);
    });
  }
}

// ============================================================================
// Quick Look Plugin
// ============================================================================

export interface QuickLookMetadata {
  displayName: string;
  contentType: string;
  size: number;
  createdAt: Date;
  modifiedAt: Date;
  customFields?: Record<string, string>;
}

class MacOSQuickLook {
  /**
   * Register Quick Look plugin for backup files
   */
  async registerPlugin(): Promise<void> {
    try {
      await invoke('macos_register_quick_look_plugin');
    } catch (error) {
      console.error('Failed to register Quick Look plugin:', error);
      throw error;
    }
  }

  /**
   * Generate Quick Look preview for backup file
   */
  async generatePreview(
    filePath: string,
    metadata: QuickLookMetadata,
  ): Promise<string> {
    try {
      return await invoke('macos_generate_quick_look_preview', {
        filePath,
        metadata,
      });
    } catch (error) {
      console.error('Failed to generate Quick Look preview:', error);
      throw error;
    }
  }

  /**
   * Generate thumbnail for backup file
   */
  async generateThumbnail(filePath: string, size: number): Promise<string> {
    try {
      return await invoke('macos_generate_quick_look_thumbnail', {
        filePath,
        size,
      });
    } catch (error) {
      console.error('Failed to generate Quick Look thumbnail:', error);
      throw error;
    }
  }
}

// ============================================================================
// Finder Extensions
// ============================================================================

export interface FinderBadge {
  path: string;
  badge: 'sync' | 'uploading' | 'uploaded' | 'error' | 'none';
}

export interface FinderContextMenuItem {
  id: string;
  title: string;
  icon?: string;
  enabled?: boolean;
}

class MacOSFinderExtension {
  /**
   * Set Finder badge for file/folder
   */
  async setBadge(path: string, badge: FinderBadge['badge']): Promise<void> {
    try {
      await invoke('macos_set_finder_badge', { path, badge });
    } catch (error) {
      console.error('Failed to set Finder badge:', error);
      throw error;
    }
  }

  /**
   * Clear Finder badge
   */
  async clearBadge(path: string): Promise<void> {
    await this.setBadge(path, 'none');
  }

  /**
   * Add context menu items to Finder
   */
  async addContextMenuItems(items: FinderContextMenuItem[]): Promise<void> {
    try {
      await invoke('macos_add_finder_context_menu_items', { items });
    } catch (error) {
      console.error('Failed to add Finder context menu items:', error);
      throw error;
    }
  }

  /**
   * Set default context menu items
   */
  async setDefaultContextMenu(): Promise<void> {
    await this.addContextMenuItems([
      {
        id: 'backup_to_db',
        title: 'Backup to Database',
        icon: 'backup',
      },
      {
        id: 'restore_from_backup',
        title: 'Restore from Backup',
        icon: 'restore',
      },
      {
        id: 'view_backup_info',
        title: 'View Backup Info',
        icon: 'info',
      },
    ]);
  }

  /**
   * Handle context menu item click
   */
  onContextMenuClick(handler: (itemId: string, path: string) => void): void {
    appWindow.listen('finder-context-menu-click', (event: any) => {
      handler(event.payload.itemId, event.payload.path);
    });
  }

  /**
   * Register Finder Sync extension
   */
  async registerSyncExtension(): Promise<void> {
    try {
      await invoke('macos_register_finder_sync_extension');
    } catch (error) {
      console.error('Failed to register Finder Sync extension:', error);
      throw error;
    }
  }
}

// ============================================================================
// Notification Center Widget
// ============================================================================

export interface NotificationWidget {
  title: string;
  content: string;
  actions?: { id: string; title: string }[];
}

class MacOSNotificationCenter {
  /**
   * Add widget to Notification Center
   */
  async addWidget(widget: NotificationWidget): Promise<void> {
    try {
      await invoke('macos_add_notification_center_widget', { widget });
    } catch (error) {
      console.error('Failed to add Notification Center widget:', error);
      throw error;
    }
  }

  /**
   * Update widget content
   */
  async updateWidget(widgetId: string, content: string): Promise<void> {
    try {
      await invoke('macos_update_notification_center_widget', {
        widgetId,
        content,
      });
    } catch (error) {
      console.error('Failed to update Notification Center widget:', error);
    }
  }

  /**
   * Remove widget
   */
  async removeWidget(widgetId: string): Promise<void> {
    try {
      await invoke('macos_remove_notification_center_widget', { widgetId });
    } catch (error) {
      console.error('Failed to remove Notification Center widget:', error);
    }
  }
}

// ============================================================================
// macOS Integration Manager
// ============================================================================

class MacOSIntegration {
  public controlCenter: MacOSControlCenter;
  public touchBar: MacOSTouchBar;
  public menuBar: MacOSMenuBar;
  public quickLook: MacOSQuickLook;
  public finderExtension: MacOSFinderExtension;
  public notificationCenter: MacOSNotificationCenter;

  constructor() {
    this.controlCenter = new MacOSControlCenter();
    this.touchBar = new MacOSTouchBar();
    this.menuBar = new MacOSMenuBar();
    this.quickLook = new MacOSQuickLook();
    this.finderExtension = new MacOSFinderExtension();
    this.notificationCenter = new MacOSNotificationCenter();
  }

  /**
   * Initialize all macOS features
   */
  async initialize(): Promise<void> {
    try {
      // Set default menu bar
      await this.menuBar.setDefaultMenu();

      // Set Touch Bar controls (if available)
      if (await this.hasTouchBar()) {
        await this.touchBar.setBackupControls();
      }

      // Register Quick Look plugin
      await this.quickLook.registerPlugin();

      // Set default Finder context menu
      await this.finderExtension.setDefaultContextMenu();

      console.log('[macOS] Integration initialized');
    } catch (error) {
      console.error('[macOS] Initialization error:', error);
    }
  }

  /**
   * Check if running on macOS
   */
  static isMacOS(): boolean {
    return navigator.platform.indexOf('Mac') > -1;
  }

  /**
   * Get macOS version
   */
  async getMacOSVersion(): Promise<string> {
    try {
      return await invoke('macos_get_version');
    } catch (error) {
      return 'unknown';
    }
  }

  /**
   * Check if device has Touch Bar
   */
  async hasTouchBar(): Promise<boolean> {
    try {
      return await invoke('macos_has_touch_bar');
    } catch (error) {
      return false;
    }
  }

  /**
   * Check if modern macOS features are available (macOS 11+)
   */
  async supportsModernFeatures(): Promise<boolean> {
    try {
      const version = await this.getMacOSVersion();
      const majorVersion = parseInt(version.split('.')[0]);
      return majorVersion >= 11;
    } catch (error) {
      return false;
    }
  }
}

// Export singleton instance
export const macOSIntegration = new MacOSIntegration();

// Export individual services
export const macOSControlCenter = macOSIntegration.controlCenter;
export const macOSTouchBar = macOSIntegration.touchBar;
export const macOSMenuBar = macOSIntegration.menuBar;
export const macOSQuickLook = macOSIntegration.quickLook;
export const macOSFinderExtension = macOSIntegration.finderExtension;
export const macOSNotificationCenter = macOSIntegration.notificationCenter;
