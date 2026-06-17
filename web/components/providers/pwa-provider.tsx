/**
 * PWA Provider Component
 * Manages PWA state and provides context throughout the app
 */

'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import {
  useServiceWorker,
  useInstallPrompt,
  useOnlineStatus,
  usePushNotifications,
  useBackgroundSync,
  usePeriodicSync,
  useBadge,
} from '@/lib/pwa-hooks';
import { offlineDB } from '@/lib/db';

// ============================================================================
// Context Types
// ============================================================================

interface PWAContextValue {
  // Service Worker
  isServiceWorkerRegistered: boolean;
  isUpdateAvailable: boolean;
  updateApp: () => void;
  clearCache: () => Promise<void>;

  // Installation
  isInstalled: boolean;
  isInstallable: boolean;
  promptInstall: () => Promise<boolean>;

  // Online Status
  isOnline: boolean;

  // Push Notifications
  notificationsSupported: boolean;
  notificationPermission: NotificationPermission;
  pushSubscription: any;
  isNotificationLoading: boolean;
  requestNotificationPermission: () => Promise<boolean>;
  subscribeToPush: (userId: string) => Promise<any>;
  unsubscribeFromPush: () => Promise<void>;
  sendTestNotification: () => Promise<boolean>;

  // Background Sync
  backgroundSyncSupported: boolean;
  registerBackgroundSync: (tag: string) => Promise<boolean>;

  // Periodic Sync
  periodicSyncSupported: boolean;
  registerPeriodicSync: (tag: string, interval: number) => Promise<boolean>;
  unregisterPeriodicSync: (tag: string) => Promise<boolean>;

  // Badge
  badgeSupported: boolean;
  setBadge: (count?: number) => Promise<boolean>;
  clearBadge: () => Promise<boolean>;

  // Offline Database
  db: typeof offlineDB;
}

const PWAContext = createContext<PWAContextValue | null>(null);

// ============================================================================
// Provider Component
// ============================================================================

export function PWAProvider({ children }: { children: React.ReactNode }) {
  const [isInitialized, setIsInitialized] = useState(false);

  // Initialize hooks
  const {
    isRegistered,
    isUpdateAvailable,
    update,
    clearCache,
  } = useServiceWorker();

  const {
    isInstalled,
    isInstallable,
    promptInstall,
  } = useInstallPrompt();

  const isOnline = useOnlineStatus();

  const {
    isSupported: notificationsSupported,
    permission: notificationPermission,
    subscription: pushSubscription,
    isLoading: isNotificationLoading,
    requestPermission,
    subscribe,
    unsubscribe,
    sendTestNotification,
  } = usePushNotifications();

  const {
    isSupported: backgroundSyncSupported,
    registerSync,
  } = useBackgroundSync();

  const {
    isSupported: periodicSyncSupported,
    registerPeriodicSync,
    unregisterPeriodicSync,
  } = usePeriodicSync();

  const {
    isSupported: badgeSupported,
    setBadge,
    clearBadge,
  } = useBadge();

  // Initialize offline database
  useEffect(() => {
    offlineDB.init().then(() => {
      setIsInitialized(true);
    });
  }, []);

  // Set up periodic sync for monitoring (if supported)
  useEffect(() => {
    if (periodicSyncSupported && isOnline) {
      // Check monitoring status every 15 minutes
      registerPeriodicSync('monitoring-update', 15 * 60 * 1000);

      // Check compliance every hour
      registerPeriodicSync('compliance-check', 60 * 60 * 1000);
    }
  }, [periodicSyncSupported, isOnline, registerPeriodicSync]);

  // Sync offline data when coming back online
  useEffect(() => {
    if (isOnline && backgroundSyncSupported) {
      registerSync('sync-backups');
      registerSync('sync-monitoring');
    }
  }, [isOnline, backgroundSyncSupported, registerSync]);

  // Store offline status changes
  useEffect(() => {
    if (!isOnline) {
      // Save offline timestamp
      offlineDB.setSetting('last_offline', Date.now());
    } else {
      // Save online timestamp
      offlineDB.setSetting('last_online', Date.now());
    }
  }, [isOnline]);

  // Handle visibility change to update badge
  useEffect(() => {
    const handleVisibilityChange = async () => {
      if (document.visibilityState === 'visible') {
        // Clear badge when app is visible
        await clearBadge();

        // Mark notifications as read
        const unread = await offlineDB.getUnreadNotifications();
        if (unread.length > 0) {
          await offlineDB.markAllNotificationsAsRead();
        }
      } else if (document.visibilityState === 'hidden') {
        // Update badge when app is hidden
        const unread = await offlineDB.getUnreadNotifications();
        await setBadge(unread.length);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [clearBadge, setBadge]);

  // Cleanup old data periodically
  useEffect(() => {
    const cleanup = async () => {
      // Delete old backups (keep 30 days)
      await offlineDB.deleteOldBackups(30);

      // Delete old monitoring data (keep 7 days)
      await offlineDB.deleteOldMonitoringData(168);

      // Delete old notifications (keep 30 days)
      await offlineDB.deleteOldNotifications(30);
    };

    // Run cleanup on mount
    cleanup();

    // Run cleanup daily
    const interval = setInterval(cleanup, 24 * 60 * 60 * 1000);

    return () => clearInterval(interval);
  }, []);

  const value: PWAContextValue = {
    // Service Worker
    isServiceWorkerRegistered: isRegistered,
    isUpdateAvailable,
    updateApp: update,
    clearCache,

    // Installation
    isInstalled,
    isInstallable,
    promptInstall,

    // Online Status
    isOnline,

    // Push Notifications
    notificationsSupported,
    notificationPermission,
    pushSubscription,
    isNotificationLoading,
    requestNotificationPermission: requestPermission,
    subscribeToPush: subscribe,
    unsubscribeFromPush: unsubscribe,
    sendTestNotification,

    // Background Sync
    backgroundSyncSupported,
    registerBackgroundSync: registerSync,

    // Periodic Sync
    periodicSyncSupported,
    registerPeriodicSync,
    unregisterPeriodicSync,

    // Badge
    badgeSupported,
    setBadge,
    clearBadge,

    // Offline Database
    db: offlineDB,
  };

  if (!isInitialized) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="flex flex-col items-center gap-4">
          <div className="w-10 h-10 border-4 border-orange-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-sm text-gray-500 dark:text-gray-400">Initializing...</p>
        </div>
      </div>
    );
  }

  return <PWAContext.Provider value={value}>{children}</PWAContext.Provider>;
}

// ============================================================================
// Hook to use PWA context
// ============================================================================

export function usePWA() {
  const context = useContext(PWAContext);

  if (!context) {
    throw new Error('usePWA must be used within PWAProvider');
  }

  return context;
}
