/**
 * Offline Indicator Component
 * Shows when the app is offline with sync status
 */

'use client';

import React, { useEffect, useState } from 'react';
import { WifiOff, Wifi, Cloud, CloudOff } from 'lucide-react';
import { usePWA } from '../providers/pwa-provider';

export function OfflineIndicator() {
  const { isOnline, db } = usePWA();
  const [queuedItems, setQueuedItems] = useState(0);
  const [showNotification, setShowNotification] = useState(false);
  const [wasOffline, setWasOffline] = useState(false);

  // Check queued items
  useEffect(() => {
    const checkQueue = async () => {
      const items = await db.getAllQueueItems();
      setQueuedItems(items.length);
    };

    checkQueue();

    // Check every 5 seconds
    const interval = setInterval(checkQueue, 5000);

    return () => clearInterval(interval);
  }, [db]);

  // Show notification when going offline/online
  useEffect(() => {
    if (!isOnline && !wasOffline) {
      setWasOffline(true);
      setShowNotification(true);

      // Hide notification after 5 seconds
      setTimeout(() => setShowNotification(false), 5000);
    } else if (isOnline && wasOffline) {
      setWasOffline(false);
      setShowNotification(true);

      // Hide notification after 5 seconds
      setTimeout(() => setShowNotification(false), 5000);
    }
  }, [isOnline, wasOffline]);

  return (
    <>
      {/* Persistent indicator */}
      {!isOnline && (
        <div className="fixed top-0 left-0 right-0 z-40 bg-amber-500 text-white px-4 py-2 text-center text-sm font-medium">
          <div className="flex items-center justify-center gap-2">
            <WifiOff className="w-4 h-4" />
            <span>You are offline</span>
            {queuedItems > 0 && (
              <span className="ml-2 px-2 py-0.5 bg-amber-600 rounded-full text-xs">
                {queuedItems} queued
              </span>
            )}
          </div>
        </div>
      )}

      {/* Toast notification */}
      {showNotification && (
        <div className="fixed bottom-4 left-4 z-50 animate-slide-up">
          <div
            className={`${
              isOnline
                ? 'bg-green-600 border-green-500'
                : 'bg-amber-600 border-amber-500'
            } text-white rounded-lg shadow-lg p-4 border max-w-sm`}
          >
            <div className="flex items-center gap-3">
              <div className="flex-shrink-0">
                {isOnline ? (
                  <Wifi className="w-6 h-6" />
                ) : (
                  <WifiOff className="w-6 h-6" />
                )}
              </div>

              <div className="flex-1">
                <h4 className="font-semibold">
                  {isOnline ? 'Back Online' : 'You are Offline'}
                </h4>
                <p className="text-sm mt-1 opacity-90">
                  {isOnline
                    ? queuedItems > 0
                      ? `Syncing ${queuedItems} queued items...`
                      : 'All data synced successfully'
                    : 'Changes will be synced when you reconnect'}
                </p>
              </div>

              <div className="flex-shrink-0">
                {isOnline ? (
                  <Cloud className="w-5 h-5" />
                ) : (
                  <CloudOff className="w-5 h-5" />
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
