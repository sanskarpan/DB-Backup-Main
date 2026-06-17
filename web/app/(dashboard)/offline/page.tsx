/**
 * Offline Fallback Page
 * Displayed when the user is offline and tries to navigate to an uncached page
 */

'use client';

import React, { useEffect, useState } from 'react';
import { WifiOff, RefreshCw, Home, Database, Activity } from 'lucide-react';
import Link from 'next/link';

export default function OfflinePage() {
  const [isOnline, setIsOnline] = useState(false);

  useEffect(() => {
    setIsOnline(navigator.onLine);

    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  const handleRetry = () => {
    if (isOnline) {
      window.location.reload();
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 flex items-center justify-center p-4">
      <div className="max-w-md w-full">
        <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
          {/* Icon */}
          <div className="mb-6">
            <div className="inline-flex items-center justify-center w-20 h-20 bg-amber-100 rounded-full">
              <WifiOff className="w-10 h-10 text-amber-600" />
            </div>
          </div>

          {/* Title */}
          <h1 className="text-3xl font-bold text-gray-900 mb-3">
            You're Offline
          </h1>

          {/* Description */}
          <p className="text-gray-600 mb-6">
            It looks like you've lost your internet connection. Don't worry, some
            features are still available offline.
          </p>

          {/* Status */}
          <div className="mb-8">
            <div
              className={`inline-flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium ${
                isOnline
                  ? 'bg-green-100 text-green-700'
                  : 'bg-amber-100 text-amber-700'
              }`}
            >
              <div
                className={`w-2 h-2 rounded-full ${
                  isOnline ? 'bg-green-600' : 'bg-amber-600'
                }`}
              />
              {isOnline ? 'Connection Restored' : 'No Connection'}
            </div>
          </div>

          {/* Actions */}
          <div className="space-y-3 mb-8">
            <button
              onClick={handleRetry}
              disabled={!isOnline}
              className="w-full flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <RefreshCw className="w-5 h-5" />
              {isOnline ? 'Retry Loading' : 'Waiting for Connection...'}
            </button>

            <Link
              href="/"
              className="w-full flex items-center justify-center gap-2 bg-gray-100 text-gray-700 px-6 py-3 rounded-lg font-medium hover:bg-gray-200 transition-colors"
            >
              <Home className="w-5 h-5" />
              Go to Home
            </Link>
          </div>

          {/* Available Features */}
          <div className="border-t border-gray-200 pt-6">
            <h2 className="text-sm font-semibold text-gray-900 mb-4 uppercase tracking-wide">
              Available Offline
            </h2>

            <div className="space-y-3 text-left">
              <div className="flex items-center gap-3 text-sm text-gray-600">
                <div className="flex-shrink-0 w-8 h-8 bg-blue-50 rounded-lg flex items-center justify-center">
                  <Database className="w-4 h-4 text-blue-600" />
                </div>
                <span>View recent backup history</span>
              </div>

              <div className="flex items-center gap-3 text-sm text-gray-600">
                <div className="flex-shrink-0 w-8 h-8 bg-blue-50 rounded-lg flex items-center justify-center">
                  <Activity className="w-4 h-4 text-blue-600" />
                </div>
                <span>Check cached monitoring data</span>
              </div>

              <div className="flex items-center gap-3 text-sm text-gray-600">
                <div className="flex-shrink-0 w-8 h-8 bg-blue-50 rounded-lg flex items-center justify-center">
                  <WifiOff className="w-4 h-4 text-blue-600" />
                </div>
                <span>Queue operations for later sync</span>
              </div>
            </div>
          </div>

          {/* Tips */}
          <div className="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4 text-left">
            <h3 className="text-sm font-semibold text-blue-900 mb-2">
              Offline Tips
            </h3>
            <ul className="text-xs text-blue-700 space-y-1">
              <li>• Changes will sync automatically when you're back online</li>
              <li>• Cached data may be up to 5 minutes old</li>
              <li>• Some features require an internet connection</li>
            </ul>
          </div>
        </div>

        {/* Footer */}
        <div className="text-center mt-6">
          <p className="text-sm text-gray-500">
            DB Backup Manager • Progressive Web App
          </p>
        </div>
      </div>
    </div>
  );
}
