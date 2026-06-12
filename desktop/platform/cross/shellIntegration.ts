/**
 * Cross-Platform Shell Integration
 * Provides file associations, context menus, and protocol handlers
 */

import { invoke } from '@tauri-apps/api/tauri';
import { appWindow } from '@tauri-apps/api/window';

// ============================================================================
// Types
// ============================================================================

export interface FileAssociation {
  extension: string;
  mimeType?: string;
  description: string;
  icon?: string;
}

export interface ContextMenuItem {
  id: string;
  label: string;
  icon?: string;
  extensions?: string[]; // File extensions this item applies to
  directories?: boolean; // Show for directories
  files?: boolean; // Show for files
  multiple?: boolean; // Show when multiple items selected
  separator?: boolean;
  submenu?: ContextMenuItem[];
}

export interface ProtocolHandler {
  protocol: string; // e.g., 'dbbackup'
  description: string;
}

export interface ShellCommand {
  id: string;
  command: string;
  name: string;
  description: string;
}

export interface DragDropHandler {
  id: string;
  extensions?: string[];
  handler: (files: string[]) => Promise<void>;
}

// ============================================================================
// Cross-Platform Shell Integration Service
// ============================================================================

class CrossPlatformShellIntegration {
  private platform: 'windows' | 'macos' | 'linux' | 'unknown' = 'unknown';
  private contextMenuItems: Map<string, ContextMenuItem> = new Map();
  private dragDropHandlers: Map<string, DragDropHandler> = new Map();
  private protocolHandlers: Set<string> = new Set();

  constructor() {
    this.detectPlatform();
    this.setupEventListeners();
  }

  /**
   * Detect current platform
   */
  private detectPlatform(): void {
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
   * Setup event listeners
   */
  private setupEventListeners(): void {
    // Context menu activation
    appWindow.listen('context-menu-activated', (event: any) => {
      this.handleContextMenuActivation(event.payload);
    });

    // File drop events
    appWindow.listen('tauri://file-drop', (event: any) => {
      this.handleFileDrop(event.payload);
    });

    // Protocol handler events
    appWindow.listen('protocol-invoked', (event: any) => {
      this.handleProtocolInvocation(event.payload);
    });
  }

  /**
   * Handle context menu activation
   */
  private async handleContextMenuActivation(payload: {
    itemId: string;
    filePaths: string[];
  }): Promise<void> {
    const item = this.contextMenuItems.get(payload.itemId);
    if (!item) {
      console.warn(`Context menu item ${payload.itemId} not found`);
      return;
    }

    // Trigger custom event for the application to handle
    appWindow.emit('shell-context-menu-action', {
      itemId: payload.itemId,
      filePaths: payload.filePaths,
      item,
    });
  }

  /**
   * Handle file drop
   */
  private async handleFileDrop(files: string[]): Promise<void> {
    for (const [, handler] of this.dragDropHandlers) {
      try {
        // Check if handler accepts these file types
        if (handler.extensions) {
          const matchingFiles = files.filter(file => {
            const ext = file.split('.').pop()?.toLowerCase();
            return ext && handler.extensions!.includes(ext);
          });

          if (matchingFiles.length > 0) {
            await handler.handler(matchingFiles);
          }
        } else {
          // Handler accepts all files
          await handler.handler(files);
        }
      } catch (error) {
        console.error(`Drag-drop handler ${handler.id} failed:`, error);
      }
    }
  }

  /**
   * Handle protocol invocation
   */
  private handleProtocolInvocation(payload: {
    protocol: string;
    url: string;
    args: Record<string, string>;
  }): void {
    appWindow.emit('shell-protocol-invoked', payload);
  }

  // ============================================================================
  // File Associations
  // ============================================================================

  /**
   * Register file association
   */
  async registerFileAssociation(association: FileAssociation): Promise<boolean> {
    try {
      await invoke('register_file_association', {
        association: {
          extension: association.extension,
          mimeType: association.mimeType,
          description: association.description,
          icon: association.icon,
          platform: this.platform,
        },
      });

      return true;
    } catch (error) {
      console.error('Failed to register file association:', error);
      return false;
    }
  }

  /**
   * Register multiple file associations
   */
  async registerFileAssociations(
    associations: FileAssociation[],
  ): Promise<void> {
    for (const assoc of associations) {
      await this.registerFileAssociation(assoc);
    }
  }

  /**
   * Register default backup file associations
   */
  async registerDefaultAssociations(): Promise<void> {
    const associations: FileAssociation[] = [
      {
        extension: 'bak',
        mimeType: 'application/x-backup',
        description: 'Database Backup File',
        icon: 'backup',
      },
      {
        extension: 'dbbackup',
        mimeType: 'application/x-dbbackup',
        description: 'DB Backup Archive',
        icon: 'backup',
      },
      {
        extension: 'sql',
        mimeType: 'application/sql',
        description: 'SQL Backup File',
        icon: 'database',
      },
    ];

    await this.registerFileAssociations(associations);
  }

  /**
   * Unregister file association
   */
  async unregisterFileAssociation(extension: string): Promise<boolean> {
    try {
      await invoke('unregister_file_association', { extension });
      return true;
    } catch (error) {
      console.error('Failed to unregister file association:', error);
      return false;
    }
  }

  // ============================================================================
  // Context Menu Integration
  // ============================================================================

  /**
   * Register context menu items
   */
  async registerContextMenuItems(items: ContextMenuItem[]): Promise<boolean> {
    try {
      items.forEach(item => {
        this.contextMenuItems.set(item.id, item);
      });

      await invoke('register_context_menu_items', {
        items,
        platform: this.platform,
      });

      return true;
    } catch (error) {
      console.error('Failed to register context menu items:', error);
      return false;
    }
  }

  /**
   * Register default context menu items
   */
  async registerDefaultContextMenu(): Promise<void> {
    const items: ContextMenuItem[] = [
      {
        id: 'backup_file',
        label: 'Backup with DB Backup',
        icon: 'backup',
        files: true,
        extensions: ['db', 'sqlite', 'sqlite3', 'sql'],
      },
      {
        id: 'restore_from_backup',
        label: 'Restore from Backup',
        icon: 'restore',
        files: true,
        extensions: ['bak', 'sql', 'dump', 'dbbackup'],
      },
      {
        id: 'verify_backup',
        label: 'Verify Backup Integrity',
        icon: 'check',
        files: true,
        extensions: ['bak', 'dbbackup'],
      },
      {
        id: 'separator_1',
        label: '',
        separator: true,
        files: true,
        extensions: ['bak', 'dbbackup', 'db', 'sqlite'],
      },
      {
        id: 'backup_info',
        label: 'View Backup Info',
        icon: 'info',
        files: true,
        extensions: ['bak', 'dbbackup'],
      },
      {
        id: 'compress_backup',
        label: 'Compress Backup',
        icon: 'compress',
        files: true,
        extensions: ['bak', 'sql', 'dump'],
      },
      {
        id: 'encrypt_backup',
        label: 'Encrypt Backup',
        icon: 'lock',
        files: true,
        extensions: ['bak', 'sql', 'dump', 'dbbackup'],
      },
    ];

    await this.registerContextMenuItems(items);
  }

  /**
   * Update context menu item
   */
  async updateContextMenuItem(
    itemId: string,
    updates: Partial<ContextMenuItem>,
  ): Promise<boolean> {
    const item = this.contextMenuItems.get(itemId);
    if (!item) {
      console.error(`Context menu item ${itemId} not found`);
      return false;
    }

    const updatedItem = { ...item, ...updates };
    this.contextMenuItems.set(itemId, updatedItem);

    try {
      await invoke('update_context_menu_item', {
        itemId,
        item: updatedItem,
      });
      return true;
    } catch (error) {
      console.error('Failed to update context menu item:', error);
      return false;
    }
  }

  /**
   * Remove context menu item
   */
  async removeContextMenuItem(itemId: string): Promise<boolean> {
    try {
      await invoke('remove_context_menu_item', { itemId });
      this.contextMenuItems.delete(itemId);
      return true;
    } catch (error) {
      console.error('Failed to remove context menu item:', error);
      return false;
    }
  }

  /**
   * Listen for context menu actions
   */
  onContextMenuAction(
    handler: (data: {
      itemId: string;
      filePaths: string[];
      item: ContextMenuItem;
    }) => void,
  ): void {
    appWindow.listen('shell-context-menu-action', (event: any) => {
      handler(event.payload);
    });
  }

  // ============================================================================
  // Protocol Handlers
  // ============================================================================

  /**
   * Register protocol handler (e.g., dbbackup://)
   */
  async registerProtocolHandler(handler: ProtocolHandler): Promise<boolean> {
    try {
      await invoke('register_protocol_handler', {
        protocol: handler.protocol,
        description: handler.description,
        platform: this.platform,
      });

      this.protocolHandlers.add(handler.protocol);
      return true;
    } catch (error) {
      console.error('Failed to register protocol handler:', error);
      return false;
    }
  }

  /**
   * Register default protocol handler
   */
  async registerDefaultProtocol(): Promise<void> {
    await this.registerProtocolHandler({
      protocol: 'dbbackup',
      description: 'DB Backup Protocol',
    });
  }

  /**
   * Unregister protocol handler
   */
  async unregisterProtocolHandler(protocol: string): Promise<boolean> {
    try {
      await invoke('unregister_protocol_handler', { protocol });
      this.protocolHandlers.delete(protocol);
      return true;
    } catch (error) {
      console.error('Failed to unregister protocol handler:', error);
      return false;
    }
  }

  /**
   * Listen for protocol invocations
   */
  onProtocolInvoked(
    handler: (data: {
      protocol: string;
      url: string;
      args: Record<string, string>;
    }) => void,
  ): void {
    appWindow.listen('shell-protocol-invoked', (event: any) => {
      handler(event.payload);
    });
  }

  // ============================================================================
  // Drag and Drop
  // ============================================================================

  /**
   * Register drag-drop handler
   */
  registerDragDropHandler(handler: DragDropHandler): void {
    this.dragDropHandlers.set(handler.id, handler);
  }

  /**
   * Unregister drag-drop handler
   */
  unregisterDragDropHandler(handlerId: string): void {
    this.dragDropHandlers.delete(handlerId);
  }

  /**
   * Register default drag-drop handlers
   */
  registerDefaultDragDropHandlers(): void {
    // Handler for database files
    this.registerDragDropHandler({
      id: 'database_files',
      extensions: ['db', 'sqlite', 'sqlite3', 'sql'],
      handler: async (files: string[]) => {
        appWindow.emit('database-files-dropped', { files });
      },
    });

    // Handler for backup files
    this.registerDragDropHandler({
      id: 'backup_files',
      extensions: ['bak', 'dbbackup', 'dump'],
      handler: async (files: string[]) => {
        appWindow.emit('backup-files-dropped', { files });
      },
    });
  }

  // ============================================================================
  // Shell Commands
  // ============================================================================

  /**
   * Register shell command (for terminal integration)
   */
  async registerShellCommand(command: ShellCommand): Promise<boolean> {
    try {
      await invoke('register_shell_command', {
        command: {
          id: command.id,
          command: command.command,
          name: command.name,
          description: command.description,
        },
      });

      return true;
    } catch (error) {
      console.error('Failed to register shell command:', error);
      return false;
    }
  }

  /**
   * Unregister shell command
   */
  async unregisterShellCommand(commandId: string): Promise<boolean> {
    try {
      await invoke('unregister_shell_command', { commandId });
      return true;
    } catch (error) {
      console.error('Failed to unregister shell command:', error);
      return false;
    }
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  /**
   * Get current platform
   */
  getPlatform(): string {
    return this.platform;
  }

  /**
   * Get registered context menu items
   */
  getContextMenuItems(): ContextMenuItem[] {
    return Array.from(this.contextMenuItems.values());
  }

  /**
   * Get registered protocol handlers
   */
  getProtocolHandlers(): string[] {
    return Array.from(this.protocolHandlers);
  }

  /**
   * Initialize all default shell integrations
   */
  async initializeDefaults(): Promise<void> {
    await this.registerDefaultAssociations();
    await this.registerDefaultContextMenu();
    await this.registerDefaultProtocol();
    this.registerDefaultDragDropHandlers();
  }
}

// Export singleton instance
export const crossPlatformShellIntegration = new CrossPlatformShellIntegration();

// Export class for testing
export { CrossPlatformShellIntegration };
