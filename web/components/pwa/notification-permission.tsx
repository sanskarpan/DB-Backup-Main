/**
 * Notification Permission Component
 * Handles requesting notification permissions and managing subscriptions
 */

'use client';

import React, { useState, useEffect } from 'react';
import { Bell, BellOff, X } from 'lucide-react';
import { usePWA } from '../providers/pwa-provider';

interface NotificationPermissionProps {
  userId?: string;
  autoPrompt?: boolean;
  delayDays?: number;
}

export function NotificationPermission({
  userId = 'default',
  autoPrompt = true,
  delayDays = 3,
}: NotificationPermissionProps) {
  const {
    notificationsSupported,
    notificationPermission,
    pushSubscription,
    requestNotificationPermission,
    subscribeToPush,
    sendTestNotification,
  } = usePWA();

  const [dismissed, setDismissed] = useState(false);
  const [showPrompt, setShowPrompt] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  // Check if we should show the prompt
  useEffect(() => {
    if (!notificationsSupported || notificationPermission !== 'default') {
      return;
    }

    // Check if user previously dismissed
    const dismissedTime = localStorage.getItem('notification-permission-dismissed');
    if (dismissedTime) {
      const daysSinceDismissed =
        (Date.now() - parseInt(dismissedTime, 10)) / (1000 * 60 * 60 * 24);

      if (daysSinceDismissed < delayDays) {
        setDismissed(true);
        return;
      }

      localStorage.removeItem('notification-permission-dismissed');
    }

    // Auto-prompt if enabled and conditions are met
    if (autoPrompt) {
      // Show prompt after a short delay on first visit
      const timer = setTimeout(() => {
        setShowPrompt(true);
      }, 5000); // 5 second delay

      return () => clearTimeout(timer);
    }
  }, [notificationsSupported, notificationPermission, autoPrompt, delayDays]);

  const handleEnable = async () => {
    setIsLoading(true);

    try {
      // Request permission
      const granted = await requestNotificationPermission();

      if (granted) {
        // Subscribe to push notifications
        await subscribeToPush(userId);

        // Send test notification
        setTimeout(async () => {
          await sendTestNotification();
        }, 1000);

        setShowPrompt(false);
      } else {
        // Permission denied
        handleDismiss();
      }
    } catch (error) {
      console.error('Failed to enable notifications:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDismiss = () => {
    setDismissed(true);
    setShowPrompt(false);
    localStorage.setItem(
      'notification-permission-dismissed',
      Date.now().toString()
    );
  };

  // Don't show if not supported, already granted/denied, or dismissed
  if (
    !notificationsSupported ||
    notificationPermission !== 'default' ||
    dismissed ||
    !showPrompt
  ) {
    return null;
  }

  return (
    <div className="fixed bottom-20 right-4 z-50 max-w-md animate-slide-up">
      <div className="bg-gradient-to-r from-indigo-600 to-indigo-700 text-white rounded-lg shadow-2xl p-4 border border-indigo-500">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0">
            <Bell className="w-6 h-6" />
          </div>

          <div className="flex-1">
            <h3 className="font-semibold text-lg mb-1">Stay Updated</h3>
            <p className="text-sm text-indigo-100 mb-3">
              Enable notifications to get real-time alerts about:
            </p>

            <ul className="text-sm text-indigo-100 space-y-1 mb-3">
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 bg-indigo-300 rounded-full" />
                Backup completion and failures
              </li>
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 bg-indigo-300 rounded-full" />
                System monitoring alerts
              </li>
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 bg-indigo-300 rounded-full" />
                Compliance violations
              </li>
              <li className="flex items-center gap-2">
                <div className="w-1.5 h-1.5 bg-indigo-300 rounded-full" />
                Critical system events
              </li>
            </ul>

            <div className="flex gap-2">
              <button
                onClick={handleEnable}
                disabled={isLoading}
                className="flex items-center gap-2 bg-white text-indigo-700 px-4 py-2 rounded-lg font-medium hover:bg-indigo-50 transition-colors disabled:opacity-50"
              >
                <Bell className="w-4 h-4" />
                {isLoading ? 'Enabling...' : 'Enable Notifications'}
              </button>

              <button
                onClick={handleDismiss}
                className="text-indigo-100 hover:text-white px-3 py-2 rounded-lg hover:bg-indigo-600 transition-colors"
              >
                Not now
              </button>
            </div>
          </div>

          <button
            onClick={handleDismiss}
            className="flex-shrink-0 text-indigo-200 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  );
}
