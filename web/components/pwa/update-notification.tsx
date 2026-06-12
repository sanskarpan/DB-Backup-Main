/**
 * Update Notification Component
 * Shows when a new version of the app is available
 */

'use client';

import React, { useState } from 'react';
import { RefreshCw, X } from 'lucide-react';
import { usePWA } from '../providers/pwa-provider';

export function UpdateNotification() {
  const { isUpdateAvailable, updateApp } = usePWA();
  const [dismissed, setDismissed] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const handleUpdate = async () => {
    setIsUpdating(true);
    updateApp();
  };

  const handleDismiss = () => {
    setDismissed(true);
  };

  if (!isUpdateAvailable || dismissed) {
    return null;
  }

  return (
    <div className="fixed top-4 left-1/2 -translate-x-1/2 z-50 max-w-md w-full mx-4 animate-slide-down">
      <div className="bg-gradient-to-r from-purple-600 to-purple-700 text-white rounded-lg shadow-2xl p-4 border border-purple-500">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0">
            <RefreshCw className="w-6 h-6" />
          </div>

          <div className="flex-1">
            <h3 className="font-semibold text-lg mb-1">Update Available</h3>
            <p className="text-sm text-purple-100 mb-3">
              A new version of DB Backup Manager is available with improvements and bug
              fixes.
            </p>

            <div className="flex gap-2">
              <button
                onClick={handleUpdate}
                disabled={isUpdating}
                className="flex items-center gap-2 bg-white text-purple-700 px-4 py-2 rounded-lg font-medium hover:bg-purple-50 transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${isUpdating ? 'animate-spin' : ''}`} />
                {isUpdating ? 'Updating...' : 'Update Now'}
              </button>

              <button
                onClick={handleDismiss}
                className="text-purple-100 hover:text-white px-3 py-2 rounded-lg hover:bg-purple-600 transition-colors"
              >
                Later
              </button>
            </div>
          </div>

          <button
            onClick={handleDismiss}
            className="flex-shrink-0 text-purple-200 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  );
}
