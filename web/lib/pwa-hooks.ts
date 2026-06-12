/**
 * PWA Hooks for managing Progressive Web App features
 */

'use client';

import { useState, useEffect, useCallback } from 'react';
import { Workbox } from 'workbox-window';

// ============================================================================
// Types
// ============================================================================

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

interface PushSubscription {
  endpoint: string;
  keys: {
    p256dh: string;
    auth: string;
  };
}

// ============================================================================
// useServiceWorker Hook
// ============================================================================

export function useServiceWorker() {
  const [wb, setWb] = useState<Workbox | null>(null);
  const [isUpdateAvailable, setIsUpdateAvailable] = useState(false);
  const [isRegistered, setIsRegistered] = useState(false);

  useEffect(() => {
    if (
      typeof window !== 'undefined' &&
      'serviceWorker' in navigator &&
      process.env.NODE_ENV === 'production'
    ) {
      const workbox = new Workbox('/sw.js', { scope: '/' });

      workbox.addEventListener('waiting', () => {
        setIsUpdateAvailable(true);
      });

      workbox.addEventListener('activated', () => {
        setIsUpdateAvailable(false);
      });

      workbox.addEventListener('installed', (event) => {
        if (!event.isUpdate) {
          setIsRegistered(true);
        }
      });

      workbox.register().then(() => {
        setIsRegistered(true);
      });

      setWb(workbox);
    }
  }, []);

  const update = useCallback(() => {
    if (wb) {
      wb.messageSkipWaiting();
      window.location.reload();
    }
  }, [wb]);

  const clearCache = useCallback(async () => {
    if (wb && wb.messageSW) {
      await wb.messageSW({ type: 'CLEAR_CACHE' });
    }
  }, [wb]);

  return {
    isRegistered,
    isUpdateAvailable,
    update,
    clearCache,
    workbox: wb,
  };
}

// ============================================================================
// useInstallPrompt Hook
// ============================================================================

export function useInstallPrompt() {
  const [installPrompt, setInstallPrompt] =
    useState<BeforeInstallPromptEvent | null>(null);
  const [isInstalled, setIsInstalled] = useState(false);
  const [isInstallable, setIsInstallable] = useState(false);

  useEffect(() => {
    // Check if already installed
    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstalled(true);
    }

    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setInstallPrompt(e as BeforeInstallPromptEvent);
      setIsInstallable(true);
    };

    const handleAppInstalled = () => {
      setIsInstalled(true);
      setInstallPrompt(null);
      setIsInstallable(false);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.addEventListener('appinstalled', handleAppInstalled);

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, []);

  const promptInstall = useCallback(async () => {
    if (!installPrompt) return false;

    await installPrompt.prompt();
    const { outcome } = await installPrompt.userChoice;

    if (outcome === 'accepted') {
      setInstallPrompt(null);
      setIsInstallable(false);
      return true;
    }

    return false;
  }, [installPrompt]);

  return {
    isInstalled,
    isInstallable,
    promptInstall,
  };
}

// ============================================================================
// useOnlineStatus Hook
// ============================================================================

export function useOnlineStatus() {
  const [isOnline, setIsOnline] = useState(
    typeof navigator !== 'undefined' ? navigator.onLine : true
  );

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return isOnline;
}

// ============================================================================
// usePushNotifications Hook
// ============================================================================

export function usePushNotifications() {
  const [subscription, setSubscription] = useState<PushSubscription | null>(null);
  const [isSupported, setIsSupported] = useState(false);
  const [permission, setPermission] = useState<NotificationPermission>('default');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if ('Notification' in window && 'serviceWorker' in navigator) {
      setIsSupported(true);
      setPermission(Notification.permission);
    }
  }, []);

  const requestPermission = useCallback(async () => {
    if (!isSupported) return false;

    const result = await Notification.requestPermission();
    setPermission(result);
    return result === 'granted';
  }, [isSupported]);

  const subscribe = useCallback(
    async (userId: string) => {
      if (!isSupported || permission !== 'granted') {
        return null;
      }

      setIsLoading(true);

      try {
        const registration = await navigator.serviceWorker.ready;

        // Get VAPID public key from server
        const response = await fetch('/api/push/public-key');
        const { publicKey } = await response.json();

        // Subscribe to push notifications
        const pushSubscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(publicKey) as BufferSource,
        });

        // Convert to our format
        const subscription: PushSubscription = {
          endpoint: pushSubscription.endpoint,
          keys: {
            p256dh: arrayBufferToBase64(pushSubscription.getKey('p256dh')),
            auth: arrayBufferToBase64(pushSubscription.getKey('auth')),
          },
        };

        // Send subscription to server
        await fetch('/api/push/subscribe', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            user_id: userId,
            endpoint: subscription.endpoint,
            keys: subscription.keys,
            device_info: {
              userAgent: navigator.userAgent,
              platform: navigator.platform,
            },
          }),
        });

        setSubscription(subscription);
        return subscription;
      } catch (error) {
        console.error('Failed to subscribe to push notifications:', error);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [isSupported, permission]
  );

  const unsubscribe = useCallback(async () => {
    if (!subscription) return;

    setIsLoading(true);

    try {
      const registration = await navigator.serviceWorker.ready;
      const pushSubscription = await registration.pushManager.getSubscription();

      if (pushSubscription) {
        await pushSubscription.unsubscribe();
      }

      // Notify server
      await fetch('/api/push/unsubscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endpoint: subscription.endpoint }),
      });

      setSubscription(null);
    } catch (error) {
      console.error('Failed to unsubscribe from push notifications:', error);
    } finally {
      setIsLoading(false);
    }
  }, [subscription]);

  const sendTestNotification = useCallback(async () => {
    if (!subscription) return false;

    try {
      const response = await fetch('/api/push/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endpoint: subscription.endpoint }),
      });

      return response.ok;
    } catch (error) {
      console.error('Failed to send test notification:', error);
      return false;
    }
  }, [subscription]);

  return {
    isSupported,
    permission,
    subscription,
    isLoading,
    requestPermission,
    subscribe,
    unsubscribe,
    sendTestNotification,
  };
}

// ============================================================================
// useBackgroundSync Hook
// ============================================================================

export function useBackgroundSync() {
  const [isSupported, setIsSupported] = useState(false);

  useEffect(() => {
    if ('serviceWorker' in navigator && 'sync' in ServiceWorkerRegistration.prototype) {
      setIsSupported(true);
    }
  }, []);

  const registerSync = useCallback(
    async (tag: string) => {
      if (!isSupported) return false;

      try {
        const registration = await navigator.serviceWorker.ready;
        // @ts-ignore - SyncManager is not yet in TypeScript DOM types
        await registration.sync.register(tag);
        return true;
      } catch (error) {
        console.error('Failed to register background sync:', error);
        return false;
      }
    },
    [isSupported]
  );

  return {
    isSupported,
    registerSync,
  };
}

// ============================================================================
// usePeriodicSync Hook
// ============================================================================

export function usePeriodicSync() {
  const [isSupported, setIsSupported] = useState(false);

  useEffect(() => {
    if (
      'serviceWorker' in navigator &&
      'periodicSync' in ServiceWorkerRegistration.prototype
    ) {
      setIsSupported(true);
    }
  }, []);

  const registerPeriodicSync = useCallback(
    async (tag: string, minInterval: number) => {
      if (!isSupported) return false;

      try {
        const registration = await navigator.serviceWorker.ready;
        // @ts-ignore - periodicSync is not in TypeScript definitions yet
        await registration.periodicSync.register(tag, {
          minInterval,
        });
        return true;
      } catch (error) {
        console.error('Failed to register periodic sync:', error);
        return false;
      }
    },
    [isSupported]
  );

  const unregisterPeriodicSync = useCallback(
    async (tag: string) => {
      if (!isSupported) return false;

      try {
        const registration = await navigator.serviceWorker.ready;
        // @ts-ignore
        await registration.periodicSync.unregister(tag);
        return true;
      } catch (error) {
        console.error('Failed to unregister periodic sync:', error);
        return false;
      }
    },
    [isSupported]
  );

  return {
    isSupported,
    registerPeriodicSync,
    unregisterPeriodicSync,
  };
}

// ============================================================================
// useBadge Hook
// ============================================================================

export function useBadge() {
  const [isSupported, setIsSupported] = useState(false);

  useEffect(() => {
    if ('setAppBadge' in navigator) {
      setIsSupported(true);
    }
  }, []);

  const setBadge = useCallback(
    async (count?: number) => {
      if (!isSupported) return false;

      try {
        // @ts-ignore
        if (count !== undefined) {
          // @ts-ignore
          await navigator.setAppBadge(count);
        } else {
          // @ts-ignore
          await navigator.setAppBadge();
        }
        return true;
      } catch (error) {
        console.error('Failed to set badge:', error);
        return false;
      }
    },
    [isSupported]
  );

  const clearBadge = useCallback(async () => {
    if (!isSupported) return false;

    try {
      // @ts-ignore
      await navigator.clearAppBadge();
      return true;
    } catch (error) {
      console.error('Failed to clear badge:', error);
      return false;
    }
  }, [isSupported]);

  return {
    isSupported,
    setBadge,
    clearBadge,
  };
}

// ============================================================================
// Utility Functions
// ============================================================================

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

function arrayBufferToBase64(buffer: ArrayBuffer | null): string {
  if (!buffer) return '';

  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return window.btoa(binary);
}
