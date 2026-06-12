/**
 * React Hooks for Desktop Platform Integration
 * Provides easy-to-use hooks for React components
 */

import { useEffect, useState, useCallback, useRef } from 'react';
import {
  desktopIntegration,
  Platform,
  detectPlatform,
  DesktopIntegrationConfig,
} from './index';
import type {
  UnifiedNotification,
  NotificationEvent,
  NotificationResult,
} from './cross/notifications';
import type { FileDialogOptions, SaveDialogOptions } from './cross/dialogs';
import type { ContextMenuItem, ProtocolHandler } from './cross/shellIntegration';

// ============================================================================
// Platform Detection Hook
// ============================================================================

export interface UsePlatformResult {
  platform: Platform;
  isWindows: boolean;
  isMacOS: boolean;
  isLinux: boolean;
  isUnknown: boolean;
  capabilities: {
    notifications: boolean;
    shellIntegration: boolean;
    platformFeatures: string[];
  } | null;
}

/**
 * Hook to detect current platform and capabilities
 */
export function usePlatform(): UsePlatformResult {
  const [platform] = useState<Platform>(detectPlatform());
  const [capabilities, setCapabilities] = useState<UsePlatformResult['capabilities']>(null);

  useEffect(() => {
    desktopIntegration.getCapabilities().then(setCapabilities);
  }, []);

  return {
    platform,
    isWindows: platform === 'windows',
    isMacOS: platform === 'macos',
    isLinux: platform === 'linux',
    isUnknown: platform === 'unknown',
    capabilities,
  };
}

// ============================================================================
// Desktop Integration Hook
// ============================================================================

export interface UseDesktopIntegrationResult {
  initialized: boolean;
  platform: Platform;
  initialize: (config?: DesktopIntegrationConfig) => Promise<void>;
  cleanup: () => Promise<void>;
  updateStatus: (status: {
    state: 'idle' | 'backing_up' | 'restoring' | 'error';
    message?: string;
    progress?: number;
  }) => Promise<void>;
}

/**
 * Hook to manage desktop integration lifecycle
 */
export function useDesktopIntegration(
  autoInitialize: boolean = true,
  config?: DesktopIntegrationConfig,
): UseDesktopIntegrationResult {
  const [initialized, setInitialized] = useState(false);
  const [platform] = useState<Platform>(detectPlatform());

  const initialize = useCallback(async (initConfig?: DesktopIntegrationConfig) => {
    try {
      await desktopIntegration.initialize(initConfig || config);
      setInitialized(true);
    } catch (error) {
      console.error('Failed to initialize desktop integration:', error);
    }
  }, [config]);

  const cleanup = useCallback(async () => {
    try {
      await desktopIntegration.cleanup();
      setInitialized(false);
    } catch (error) {
      console.error('Failed to cleanup desktop integration:', error);
    }
  }, []);

  const updateStatus = useCallback(async (status: {
    state: 'idle' | 'backing_up' | 'restoring' | 'error';
    message?: string;
    progress?: number;
  }) => {
    await desktopIntegration.updateStatus(status);
  }, []);

  useEffect(() => {
    if (autoInitialize && !initialized) {
      initialize();
    }

    return () => {
      if (initialized) {
        cleanup();
      }
    };
  }, [autoInitialize, initialized, initialize, cleanup]);

  return {
    initialized,
    platform,
    initialize,
    cleanup,
    updateStatus,
  };
}

// ============================================================================
// Notifications Hook
// ============================================================================

export interface UseNotificationsResult {
  send: (notification: UnifiedNotification) => Promise<NotificationResult>;
  close: (notificationId: string) => Promise<void>;
  notifyBackupComplete: (
    databaseName: string,
    size?: string,
    duration?: string,
  ) => Promise<NotificationResult>;
  notifyBackupFailed: (
    databaseName: string,
    error: string,
  ) => Promise<NotificationResult>;
  notifyBackupInProgress: (
    databaseName: string,
    progress: number,
  ) => Promise<NotificationResult>;
  notifyRestoreComplete: (
    databaseName: string,
    backupId: string,
  ) => Promise<NotificationResult>;
  notifyRestoreFailed: (
    databaseName: string,
    error: string,
  ) => Promise<NotificationResult>;
  onNotificationEvent: (handler: (event: NotificationEvent) => void) => () => void;
  history: UnifiedNotification[];
}

/**
 * Hook for cross-platform notifications
 */
export function useNotifications(): UseNotificationsResult {
  const [history, setHistory] = useState<UnifiedNotification[]>([]);
  const listenerIdRef = useRef<string>(`listener_${Date.now()}`);

  useEffect(() => {
    // Update history
    setHistory(desktopIntegration.notifications.getHistory());

    return () => {
      desktopIntegration.notifications.offNotificationEvent(listenerIdRef.current);
    };
  }, []);

  const onNotificationEvent = useCallback(
    (handler: (event: NotificationEvent) => void) => {
      const listenerId = listenerIdRef.current;
      desktopIntegration.notifications.onNotificationEvent(listenerId, handler);

      return () => {
        desktopIntegration.notifications.offNotificationEvent(listenerId);
      };
    },
    [],
  );

  return {
    send: desktopIntegration.notifications.send.bind(desktopIntegration.notifications),
    close: desktopIntegration.notifications.close.bind(desktopIntegration.notifications),
    notifyBackupComplete: desktopIntegration.notifications.notifyBackupComplete.bind(
      desktopIntegration.notifications,
    ),
    notifyBackupFailed: desktopIntegration.notifications.notifyBackupFailed.bind(
      desktopIntegration.notifications,
    ),
    notifyBackupInProgress: desktopIntegration.notifications.notifyBackupInProgress.bind(
      desktopIntegration.notifications,
    ),
    notifyRestoreComplete: desktopIntegration.notifications.notifyRestoreComplete.bind(
      desktopIntegration.notifications,
    ),
    notifyRestoreFailed: desktopIntegration.notifications.notifyRestoreFailed.bind(
      desktopIntegration.notifications,
    ),
    onNotificationEvent,
    history,
  };
}

// ============================================================================
// Dialogs Hook
// ============================================================================

export interface UseDialogsResult {
  openFile: (options?: FileDialogOptions) => Promise<string | string[] | null>;
  openSingleFile: (options?: FileDialogOptions) => Promise<string | null>;
  openMultipleFiles: (options?: FileDialogOptions) => Promise<string[] | null>;
  openDirectory: (options?: FileDialogOptions) => Promise<string | null>;
  saveFile: (options?: SaveDialogOptions) => Promise<string | null>;
  showInfo: (message: string, title?: string) => Promise<void>;
  showWarning: (message: string, title?: string) => Promise<void>;
  showError: (message: string, title?: string) => Promise<void>;
  showConfirm: (
    message: string,
    options?: { title?: string; okLabel?: string; cancelLabel?: string },
  ) => Promise<boolean>;
  askYesNo: (question: string, title?: string) => Promise<boolean>;
  selectBackupFile: () => Promise<string | null>;
  selectBackupSaveLocation: (defaultName?: string) => Promise<string | null>;
  confirmBackupDeletion: (backupName: string) => Promise<boolean>;
  confirmDatabaseRestore: (
    databaseName: string,
    backupName: string,
  ) => Promise<boolean>;
}

/**
 * Hook for cross-platform dialogs
 */
export function useDialogs(): UseDialogsResult {
  return {
    openFile: desktopIntegration.dialogs.openFile.bind(desktopIntegration.dialogs),
    openSingleFile: desktopIntegration.dialogs.openSingleFile.bind(
      desktopIntegration.dialogs,
    ),
    openMultipleFiles: desktopIntegration.dialogs.openMultipleFiles.bind(
      desktopIntegration.dialogs,
    ),
    openDirectory: desktopIntegration.dialogs.openDirectory.bind(
      desktopIntegration.dialogs,
    ),
    saveFile: desktopIntegration.dialogs.saveFile.bind(desktopIntegration.dialogs),
    showInfo: desktopIntegration.dialogs.showInfo.bind(desktopIntegration.dialogs),
    showWarning: desktopIntegration.dialogs.showWarning.bind(desktopIntegration.dialogs),
    showError: desktopIntegration.dialogs.showError.bind(desktopIntegration.dialogs),
    showConfirm: desktopIntegration.dialogs.showConfirm.bind(desktopIntegration.dialogs),
    askYesNo: desktopIntegration.dialogs.askYesNo.bind(desktopIntegration.dialogs),
    selectBackupFile: desktopIntegration.dialogs.selectBackupFile.bind(
      desktopIntegration.dialogs,
    ),
    selectBackupSaveLocation:
      desktopIntegration.dialogs.selectBackupSaveLocation.bind(
        desktopIntegration.dialogs,
      ),
    confirmBackupDeletion: desktopIntegration.dialogs.confirmBackupDeletion.bind(
      desktopIntegration.dialogs,
    ),
    confirmDatabaseRestore: desktopIntegration.dialogs.confirmDatabaseRestore.bind(
      desktopIntegration.dialogs,
    ),
  };
}

// ============================================================================
// Shell Integration Hook
// ============================================================================

export interface UseShellIntegrationResult {
  registerContextMenu: (items: ContextMenuItem[]) => Promise<boolean>;
  registerProtocol: (handler: ProtocolHandler) => Promise<boolean>;
  onContextMenuAction: (
    handler: (data: {
      itemId: string;
      filePaths: string[];
      item: ContextMenuItem;
    }) => void,
  ) => void;
  onProtocolInvoked: (
    handler: (data: {
      protocol: string;
      url: string;
      args: Record<string, string>;
    }) => void,
  ) => void;
  platform: Platform;
}

/**
 * Hook for shell integration (context menus, protocols, etc.)
 */
export function useShellIntegration(): UseShellIntegrationResult {
  const [platform] = useState<Platform>(detectPlatform());

  const registerContextMenu = useCallback(
    async (items: ContextMenuItem[]) => {
      return await desktopIntegration.shell.registerContextMenuItems(items);
    },
    [],
  );

  const registerProtocol = useCallback(async (handler: ProtocolHandler) => {
    return await desktopIntegration.shell.registerProtocolHandler(handler);
  }, []);

  const onContextMenuAction = useCallback(
    (
      handler: (data: {
        itemId: string;
        filePaths: string[];
        item: ContextMenuItem;
      }) => void,
    ) => {
      desktopIntegration.shell.onContextMenuAction(handler);
    },
    [],
  );

  const onProtocolInvoked = useCallback(
    (
      handler: (data: {
        protocol: string;
        url: string;
        args: Record<string, string>;
      }) => void,
    ) => {
      desktopIntegration.shell.onProtocolInvoked(handler);
    },
    [],
  );

  return {
    registerContextMenu,
    registerProtocol,
    onContextMenuAction,
    onProtocolInvoked,
    platform,
  };
}

// ============================================================================
// Backup Operations Hook
// ============================================================================

export interface UseBackupOperationsResult {
  notifyBackupStarted: (databaseName: string) => Promise<void>;
  notifyBackupProgress: (databaseName: string, progress: number) => Promise<void>;
  notifyBackupComplete: (
    databaseName: string,
    size?: string,
    duration?: string,
  ) => Promise<void>;
  notifyBackupFailed: (databaseName: string, error: string) => Promise<void>;
  notifyRestoreComplete: (databaseName: string, backupId: string) => Promise<void>;
  notifyRestoreFailed: (databaseName: string, error: string) => Promise<void>;
  confirmBackupDeletion: (backupName: string) => Promise<boolean>;
  confirmDatabaseRestore: (
    databaseName: string,
    backupName: string,
  ) => Promise<boolean>;
  selectBackupFile: () => Promise<string | null>;
  selectBackupSaveLocation: (defaultName?: string) => Promise<string | null>;
}

/**
 * Hook combining notifications and dialogs for backup operations
 */
export function useBackupOperations(): UseBackupOperationsResult {
  const notifications = useNotifications();
  const dialogs = useDialogs();
  const { updateStatus } = useDesktopIntegration(false);

  const notifyBackupStarted = useCallback(
    async (databaseName: string) => {
      await updateStatus({
        state: 'backing_up',
        message: `Backing up ${databaseName}`,
        progress: 0,
      });
    },
    [updateStatus],
  );

  const notifyBackupProgress = useCallback(
    async (databaseName: string, progress: number) => {
      await updateStatus({
        state: 'backing_up',
        message: `Backing up ${databaseName}`,
        progress,
      });
      await notifications.notifyBackupInProgress(databaseName, progress);
    },
    [updateStatus, notifications],
  );

  const notifyBackupComplete = useCallback(
    async (databaseName: string, size?: string, duration?: string) => {
      await desktopIntegration.notifyBackupComplete(databaseName, size, duration);
    },
    [],
  );

  const notifyBackupFailed = useCallback(
    async (databaseName: string, error: string) => {
      await desktopIntegration.notifyBackupFailed(databaseName, error);
    },
    [],
  );

  const notifyRestoreComplete = useCallback(
    async (databaseName: string, backupId: string) => {
      await notifications.notifyRestoreComplete(databaseName, backupId);
      await updateStatus({
        state: 'idle',
        message: `${databaseName} restored`,
      });
    },
    [notifications, updateStatus],
  );

  const notifyRestoreFailed = useCallback(
    async (databaseName: string, error: string) => {
      await notifications.notifyRestoreFailed(databaseName, error);
      await updateStatus({
        state: 'error',
        message: `Restore failed: ${error}`,
      });
    },
    [notifications, updateStatus],
  );

  return {
    notifyBackupStarted,
    notifyBackupProgress,
    notifyBackupComplete,
    notifyBackupFailed,
    notifyRestoreComplete,
    notifyRestoreFailed,
    confirmBackupDeletion: dialogs.confirmBackupDeletion,
    confirmDatabaseRestore: dialogs.confirmDatabaseRestore,
    selectBackupFile: dialogs.selectBackupFile,
    selectBackupSaveLocation: dialogs.selectBackupSaveLocation,
  };
}

// ============================================================================
// Complete Integration Hook
// ============================================================================

export interface UseCompleteDesktopIntegrationResult {
  platform: UsePlatformResult;
  integration: UseDesktopIntegrationResult;
  notifications: UseNotificationsResult;
  dialogs: UseDialogsResult;
  shell: UseShellIntegrationResult;
  backup: UseBackupOperationsResult;
}

/**
 * Hook providing access to all desktop integration features
 */
export function useCompleteDesktopIntegration(
  autoInitialize: boolean = true,
  config?: DesktopIntegrationConfig,
): UseCompleteDesktopIntegrationResult {
  const platform = usePlatform();
  const integration = useDesktopIntegration(autoInitialize, config);
  const notifications = useNotifications();
  const dialogs = useDialogs();
  const shell = useShellIntegration();
  const backup = useBackupOperations();

  return {
    platform,
    integration,
    notifications,
    dialogs,
    shell,
    backup,
  };
}
