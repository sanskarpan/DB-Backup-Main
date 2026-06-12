/**
 * Cross-Platform Dialog System
 * Provides unified dialog API across Windows, macOS, and Linux
 */

import { invoke } from '@tauri-apps/api/tauri';
import { open, save, message, ask, confirm } from '@tauri-apps/api/dialog';

// ============================================================================
// Types
// ============================================================================

export interface FileDialogOptions {
  title?: string;
  defaultPath?: string;
  filters?: FileFilter[];
  multiple?: boolean;
  directory?: boolean;
}

export interface FileFilter {
  name: string;
  extensions: string[];
}

export interface SaveDialogOptions {
  title?: string;
  defaultPath?: string;
  filters?: FileFilter[];
}

export interface MessageDialogOptions {
  title?: string;
  type?: 'info' | 'warning' | 'error';
}

export interface ConfirmDialogOptions {
  title?: string;
  type?: 'info' | 'warning';
  okLabel?: string;
  cancelLabel?: string;
}

export interface CustomDialogButton {
  id: string;
  label: string;
  default?: boolean;
  destructive?: boolean;
}

export interface CustomDialogOptions {
  title: string;
  message: string;
  buttons: CustomDialogButton[];
  type?: 'info' | 'warning' | 'error' | 'question';
  icon?: string;
}

// ============================================================================
// Cross-Platform Dialog Service
// ============================================================================

class CrossPlatformDialogs {
  private platform: 'windows' | 'macos' | 'linux' | 'unknown' = 'unknown';

  constructor() {
    this.detectPlatform();
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

  // ============================================================================
  // File Dialogs
  // ============================================================================

  /**
   * Open file dialog
   */
  async openFile(options: FileDialogOptions = {}): Promise<string | string[] | null> {
    try {
      const selected = await open({
        title: options.title,
        defaultPath: options.defaultPath,
        filters: options.filters,
        multiple: options.multiple || false,
        directory: options.directory || false,
      });

      return selected;
    } catch (error) {
      console.error('Failed to open file dialog:', error);
      return null;
    }
  }

  /**
   * Open single file dialog
   */
  async openSingleFile(options: FileDialogOptions = {}): Promise<string | null> {
    const result = await this.openFile({ ...options, multiple: false });
    return typeof result === 'string' ? result : null;
  }

  /**
   * Open multiple files dialog
   */
  async openMultipleFiles(options: FileDialogOptions = {}): Promise<string[] | null> {
    const result = await this.openFile({ ...options, multiple: true });
    return Array.isArray(result) ? result : null;
  }

  /**
   * Open directory dialog
   */
  async openDirectory(options: FileDialogOptions = {}): Promise<string | null> {
    const result = await this.openFile({ ...options, directory: true, multiple: false });
    return typeof result === 'string' ? result : null;
  }

  /**
   * Save file dialog
   */
  async saveFile(options: SaveDialogOptions = {}): Promise<string | null> {
    try {
      const selected = await save({
        title: options.title,
        defaultPath: options.defaultPath,
        filters: options.filters,
      });

      return selected;
    } catch (error) {
      console.error('Failed to open save dialog:', error);
      return null;
    }
  }

  // ============================================================================
  // Message Dialogs
  // ============================================================================

  /**
   * Show message dialog
   */
  async showMessage(
    message: string,
    options: MessageDialogOptions = {},
  ): Promise<void> {
    try {
      await message(message, {
        title: options.title,
        type: options.type,
      });
    } catch (error) {
      console.error('Failed to show message:', error);
    }
  }

  /**
   * Show info message
   */
  async showInfo(message: string, title?: string): Promise<void> {
    await this.showMessage(message, { title, type: 'info' });
  }

  /**
   * Show warning message
   */
  async showWarning(message: string, title?: string): Promise<void> {
    await this.showMessage(message, { title, type: 'warning' });
  }

  /**
   * Show error message
   */
  async showError(message: string, title?: string): Promise<void> {
    await this.showMessage(message, { title, type: 'error' });
  }

  /**
   * Show confirmation dialog
   */
  async showConfirm(
    message: string,
    options: ConfirmDialogOptions = {},
  ): Promise<boolean> {
    try {
      return await confirm(message, {
        title: options.title,
        type: options.type,
        okLabel: options.okLabel,
        cancelLabel: options.cancelLabel,
      });
    } catch (error) {
      console.error('Failed to show confirmation:', error);
      return false;
    }
  }

  /**
   * Ask yes/no question
   */
  async askYesNo(
    question: string,
    title?: string,
  ): Promise<boolean> {
    try {
      return await ask(question, { title, type: 'info' });
    } catch (error) {
      console.error('Failed to ask question:', error);
      return false;
    }
  }

  /**
   * Show custom dialog with multiple buttons
   */
  async showCustomDialog(options: CustomDialogOptions): Promise<string | null> {
    try {
      const result = await invoke<string>('show_custom_dialog', {
        dialog: {
          title: options.title,
          message: options.message,
          buttons: options.buttons,
          type: options.type || 'info',
          icon: options.icon,
        },
      });

      return result;
    } catch (error) {
      console.error('Failed to show custom dialog:', error);
      return null;
    }
  }

  // ============================================================================
  // Platform-Specific Dialogs
  // ============================================================================

  /**
   * Show native Windows task dialog (Windows only)
   */
  async showWindowsTaskDialog(options: {
    title: string;
    instruction: string;
    content: string;
    buttons: CustomDialogButton[];
    icon?: 'information' | 'warning' | 'error' | 'shield';
    expandedInfo?: string;
  }): Promise<string | null> {
    if (this.platform !== 'windows') {
      console.warn('Task dialog is Windows-only, using custom dialog instead');
      return await this.showCustomDialog({
        title: options.title,
        message: `${options.instruction}\n\n${options.content}`,
        buttons: options.buttons,
        type: options.icon === 'error' ? 'error' :
              options.icon === 'warning' ? 'warning' : 'info',
      });
    }

    try {
      return await invoke<string>('windows_show_task_dialog', { options });
    } catch (error) {
      console.error('Failed to show Windows task dialog:', error);
      return null;
    }
  }

  /**
   * Show macOS sheet dialog (macOS only)
   */
  async showMacOSSheet(options: {
    title: string;
    message: string;
    buttons: CustomDialogButton[];
    alertStyle?: 'informational' | 'warning' | 'critical';
  }): Promise<string | null> {
    if (this.platform !== 'macos') {
      console.warn('Sheet dialog is macOS-only, using custom dialog instead');
      return await this.showCustomDialog({
        title: options.title,
        message: options.message,
        buttons: options.buttons,
        type: options.alertStyle === 'critical' ? 'error' :
              options.alertStyle === 'warning' ? 'warning' : 'info',
      });
    }

    try {
      return await invoke<string>('macos_show_sheet', { options });
    } catch (error) {
      console.error('Failed to show macOS sheet:', error);
      return null;
    }
  }

  // ============================================================================
  // Convenience Methods for Common Dialogs
  // ============================================================================

  /**
   * Show backup file selection dialog
   */
  async selectBackupFile(): Promise<string | null> {
    return await this.openSingleFile({
      title: 'Select Backup File',
      filters: [
        { name: 'Backup Files', extensions: ['bak', 'sql', 'dump', 'gz', 'tar'] },
        { name: 'All Files', extensions: ['*'] },
      ],
    });
  }

  /**
   * Show backup save location dialog
   */
  async selectBackupSaveLocation(
    defaultName: string = 'backup',
  ): Promise<string | null> {
    return await this.saveFile({
      title: 'Save Backup As',
      defaultPath: `${defaultName}_${new Date().toISOString().split('T')[0]}.bak`,
      filters: [
        { name: 'Backup Files', extensions: ['bak', 'sql', 'dump'] },
        { name: 'Compressed', extensions: ['gz', 'tar', 'zip'] },
      ],
    });
  }

  /**
   * Show database selection dialog
   */
  async selectDatabaseFile(): Promise<string | null> {
    return await this.openSingleFile({
      title: 'Select Database File',
      filters: [
        { name: 'Database Files', extensions: ['db', 'sqlite', 'sqlite3', 'sql'] },
        { name: 'All Files', extensions: ['*'] },
      ],
    });
  }

  /**
   * Show export directory dialog
   */
  async selectExportDirectory(): Promise<string | null> {
    return await this.openDirectory({
      title: 'Select Export Directory',
    });
  }

  /**
   * Confirm backup deletion
   */
  async confirmBackupDeletion(backupName: string): Promise<boolean> {
    return await this.showConfirm(
      `Are you sure you want to delete backup "${backupName}"? This action cannot be undone.`,
      {
        title: 'Confirm Deletion',
        type: 'warning',
        okLabel: 'Delete',
        cancelLabel: 'Cancel',
      },
    );
  }

  /**
   * Confirm database restore
   */
  async confirmDatabaseRestore(
    databaseName: string,
    backupName: string,
  ): Promise<boolean> {
    return await this.showConfirm(
      `Restore "${databaseName}" from backup "${backupName}"?\n\nThis will overwrite the current database. Make sure to create a backup of the current state if needed.`,
      {
        title: 'Confirm Restore',
        type: 'warning',
        okLabel: 'Restore',
        cancelLabel: 'Cancel',
      },
    );
  }

  /**
   * Confirm schedule cancellation
   */
  async confirmScheduleCancellation(scheduleName: string): Promise<boolean> {
    return await this.showConfirm(
      `Cancel the scheduled backup "${scheduleName}"?`,
      {
        title: 'Confirm Cancellation',
        type: 'info',
        okLabel: 'Yes, Cancel',
        cancelLabel: 'No',
      },
    );
  }

  /**
   * Show backup progress dialog
   */
  async showBackupProgress(
    databaseName: string,
    onCancel?: () => void,
  ): Promise<void> {
    // This would typically open a progress dialog with a cancel button
    // Implementation depends on whether you want a modal or non-modal progress indicator
    try {
      await invoke('show_progress_dialog', {
        title: 'Backup In Progress',
        message: `Backing up ${databaseName}...`,
        cancellable: !!onCancel,
      });
    } catch (error) {
      console.error('Failed to show progress dialog:', error);
    }
  }

  /**
   * Show backup complete dialog
   */
  async showBackupCompleteDialog(
    databaseName: string,
    backupSize: string,
    duration: string,
  ): Promise<void> {
    await this.showInfo(
      `Backup of "${databaseName}" completed successfully.\n\nSize: ${backupSize}\nDuration: ${duration}`,
      'Backup Complete',
    );
  }

  /**
   * Show backup failed dialog
   */
  async showBackupFailedDialog(
    databaseName: string,
    error: string,
  ): Promise<boolean> {
    const retry = await this.showCustomDialog({
      title: 'Backup Failed',
      message: `Failed to backup "${databaseName}".\n\nError: ${error}\n\nWould you like to retry?`,
      type: 'error',
      buttons: [
        { id: 'retry', label: 'Retry', default: true },
        { id: 'cancel', label: 'Cancel' },
        { id: 'view_logs', label: 'View Logs' },
      ],
    });

    return retry === 'retry';
  }

  /**
   * Get current platform
   */
  getPlatform(): string {
    return this.platform;
  }
}

// Export singleton instance
export const crossPlatformDialogs = new CrossPlatformDialogs();

// Export class for testing
export { CrossPlatformDialogs };
