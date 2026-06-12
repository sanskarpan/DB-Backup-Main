/**
 * PWA Settings Panel Component
 * Comprehensive settings panel for managing all PWA features
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Bell,
  BellOff,
  Download,
  RefreshCw,
  Trash2,
  Database,
  Wifi,
  WifiOff,
  Cloud,
  Check,
  X,
  Settings,
  Smartphone,
} from 'lucide-react';
import { usePWA } from '../providers/pwa-provider';

export function PWASettings() {
  const {
    isServiceWorkerRegistered,
    isUpdateAvailable,
    updateApp,
    clearCache,
    isInstalled,
    isInstallable,
    promptInstall,
    isOnline,
    notificationsSupported,
    notificationPermission,
    pushSubscription,
    requestNotificationPermission,
    subscribeToPush,
    unsubscribeFromPush,
    sendTestNotification,
    backgroundSyncSupported,
    periodicSyncSupported,
    badgeSupported,
    db,
  } = usePWA();

  const [storageUsage, setStorageUsage] = useState({ usage: 0, quota: 0 });
  const [isClearing, setIsClearing] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);

  // Get storage estimate
  useEffect(() => {
    const getStorage = async () => {
      const estimate = await db.getStorageEstimate();
      setStorageUsage({ usage: estimate.usage || 0, quota: estimate.quota || 0 });
    };

    getStorage();
  }, [db]);

  const handleClearCache = async () => {
    setIsClearing(true);
    try {
      await clearCache();
      await db.clearAll();

      // Refresh storage estimate
      const estimate = await db.getStorageEstimate();
      setStorageUsage({ usage: estimate.usage || 0, quota: estimate.quota || 0 });

      showSuccessMessage();
    } catch (error) {
      console.error('Failed to clear cache:', error);
    } finally {
      setIsClearing(false);
    }
  };

  const handleUpdate = async () => {
    setIsUpdating(true);
    updateApp();
  };

  const handleInstall = async () => {
    await promptInstall();
  };

  const handleEnableNotifications = async () => {
    const granted = await requestNotificationPermission();
    if (granted) {
      await subscribeToPush('default-user');
    }
  };

  const handleDisableNotifications = async () => {
    await unsubscribeFromPush();
  };

  const handleTestNotification = async () => {
    setIsTesting(true);
    try {
      await sendTestNotification();
    } finally {
      setIsTesting(false);
    }
  };

  const showSuccessMessage = () => {
    setShowSuccess(true);
    setTimeout(() => setShowSuccess(false), 3000);
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  const storagePercent =
    storageUsage.quota > 0
      ? ((storageUsage.usage / storageUsage.quota) * 100).toFixed(1)
      : 0;

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <Settings className="w-5 h-5 text-gray-700" />
        <h2 className="text-xl font-semibold text-gray-900">PWA Settings</h2>
      </div>

      {/* Success Message */}
      {showSuccess && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 flex items-center gap-3">
          <Check className="w-5 h-5 text-green-600" />
          <span className="text-green-700 font-medium">Settings saved successfully!</span>
        </div>
      )}

      {/* Installation Status */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900 flex items-center gap-2">
          <Smartphone className="w-5 h-5" />
          Installation
        </h3>

        <div className="bg-gray-50 rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-gray-900">App Status</p>
              <p className="text-sm text-gray-600">
                {isInstalled
                  ? 'App is installed'
                  : isInstallable
                  ? 'App is installable'
                  : 'App is running in browser'}
              </p>
            </div>
            <div
              className={`px-3 py-1 rounded-full text-sm font-medium ${
                isInstalled
                  ? 'bg-green-100 text-green-700'
                  : 'bg-gray-100 text-gray-700'
              }`}
            >
              {isInstalled ? 'Installed' : 'Browser'}
            </div>
          </div>

          {isInstallable && !isInstalled && (
            <button
              onClick={handleInstall}
              className="w-full flex items-center justify-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Download className="w-4 h-4" />
              Install App
            </button>
          )}
        </div>
      </div>

      {/* Service Worker */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900 flex items-center gap-2">
          <Cloud className="w-5 h-5" />
          Service Worker
        </h3>

        <div className="bg-gray-50 rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-gray-900">Status</p>
              <p className="text-sm text-gray-600">
                {isServiceWorkerRegistered ? 'Active and running' : 'Not registered'}
              </p>
            </div>
            <div
              className={`px-3 py-1 rounded-full text-sm font-medium ${
                isServiceWorkerRegistered
                  ? 'bg-green-100 text-green-700'
                  : 'bg-red-100 text-red-700'
              }`}
            >
              {isServiceWorkerRegistered ? 'Active' : 'Inactive'}
            </div>
          </div>

          {isUpdateAvailable && (
            <div className="bg-purple-50 border border-purple-200 rounded-lg p-3">
              <p className="text-sm text-purple-700 mb-2">
                A new version is available
              </p>
              <button
                onClick={handleUpdate}
                disabled={isUpdating}
                className="w-full flex items-center justify-center gap-2 bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50"
              >
                <RefreshCw
                  className={`w-4 h-4 ${isUpdating ? 'animate-spin' : ''}`}
                />
                {isUpdating ? 'Updating...' : 'Update Now'}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Notifications */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900 flex items-center gap-2">
          <Bell className="w-5 h-5" />
          Push Notifications
        </h3>

        <div className="bg-gray-50 rounded-lg p-4 space-y-3">
          {!notificationsSupported ? (
            <p className="text-sm text-gray-600">
              Push notifications are not supported in your browser
            </p>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium text-gray-900">Status</p>
                  <p className="text-sm text-gray-600">
                    {notificationPermission === 'granted'
                      ? 'Enabled'
                      : notificationPermission === 'denied'
                      ? 'Blocked'
                      : 'Not enabled'}
                  </p>
                </div>
                <div
                  className={`px-3 py-1 rounded-full text-sm font-medium ${
                    notificationPermission === 'granted'
                      ? 'bg-green-100 text-green-700'
                      : notificationPermission === 'denied'
                      ? 'bg-red-100 text-red-700'
                      : 'bg-gray-100 text-gray-700'
                  }`}
                >
                  {notificationPermission === 'granted'
                    ? 'Enabled'
                    : notificationPermission === 'denied'
                    ? 'Blocked'
                    : 'Disabled'}
                </div>
              </div>

              <div className="flex gap-2">
                {notificationPermission !== 'granted' ? (
                  <button
                    onClick={handleEnableNotifications}
                    className="flex-1 flex items-center justify-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    <Bell className="w-4 h-4" />
                    Enable Notifications
                  </button>
                ) : (
                  <>
                    <button
                      onClick={handleTestNotification}
                      disabled={isTesting}
                      className="flex-1 flex items-center justify-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
                    >
                      <Bell className="w-4 h-4" />
                      {isTesting ? 'Sending...' : 'Send Test'}
                    </button>
                    <button
                      onClick={handleDisableNotifications}
                      className="flex-1 flex items-center justify-center gap-2 bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition-colors"
                    >
                      <BellOff className="w-4 h-4" />
                      Disable
                    </button>
                  </>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* Connection Status */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900 flex items-center gap-2">
          {isOnline ? <Wifi className="w-5 h-5" /> : <WifiOff className="w-5 h-5" />}
          Connection Status
        </h3>

        <div className="bg-gray-50 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-gray-900">Network</p>
              <p className="text-sm text-gray-600">
                {isOnline ? 'Connected' : 'Offline'}
              </p>
            </div>
            <div
              className={`px-3 py-1 rounded-full text-sm font-medium ${
                isOnline
                  ? 'bg-green-100 text-green-700'
                  : 'bg-amber-100 text-amber-700'
              }`}
            >
              {isOnline ? 'Online' : 'Offline'}
            </div>
          </div>
        </div>
      </div>

      {/* Features */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900">PWA Features</h3>

        <div className="bg-gray-50 rounded-lg p-4 space-y-2">
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-gray-700">Background Sync</span>
            <div
              className={`flex items-center gap-2 ${
                backgroundSyncSupported ? 'text-green-600' : 'text-gray-400'
              }`}
            >
              {backgroundSyncSupported ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
              <span className="text-sm font-medium">
                {backgroundSyncSupported ? 'Supported' : 'Not Supported'}
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-gray-700">Periodic Background Sync</span>
            <div
              className={`flex items-center gap-2 ${
                periodicSyncSupported ? 'text-green-600' : 'text-gray-400'
              }`}
            >
              {periodicSyncSupported ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
              <span className="text-sm font-medium">
                {periodicSyncSupported ? 'Supported' : 'Not Supported'}
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-gray-700">App Badge</span>
            <div
              className={`flex items-center gap-2 ${
                badgeSupported ? 'text-green-600' : 'text-gray-400'
              }`}
            >
              {badgeSupported ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
              <span className="text-sm font-medium">
                {badgeSupported ? 'Supported' : 'Not Supported'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Storage */}
      <div className="space-y-3">
        <h3 className="font-semibold text-gray-900 flex items-center gap-2">
          <Database className="w-5 h-5" />
          Storage
        </h3>

        <div className="bg-gray-50 rounded-lg p-4 space-y-3">
          <div>
            <div className="flex justify-between text-sm mb-2">
              <span className="text-gray-600">Used</span>
              <span className="font-medium text-gray-900">
                {formatBytes(storageUsage.usage)} of {formatBytes(storageUsage.quota)} (
                {storagePercent}%)
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full transition-all"
                style={{ width: `${storagePercent}%` }}
              />
            </div>
          </div>

          <button
            onClick={handleClearCache}
            disabled={isClearing}
            className="w-full flex items-center justify-center gap-2 bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Trash2 className="w-4 h-4" />
            {isClearing ? 'Clearing...' : 'Clear All Cache & Data'}
          </button>

          <p className="text-xs text-gray-500">
            This will clear all cached data and offline storage. You may need to refresh
            the page.
          </p>
        </div>
      </div>
    </div>
  );
}
