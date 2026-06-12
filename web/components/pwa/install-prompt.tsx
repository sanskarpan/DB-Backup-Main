/**
 * PWA Install Prompt Component
 * Shows a prompt to install the app when installable
 */

'use client';

import React, { useState, useEffect } from 'react';
import { X, Download, Smartphone } from 'lucide-react';
import { usePWA } from '../providers/pwa-provider';

export function InstallPrompt() {
  const { isInstalled, isInstallable, promptInstall } = usePWA();
  const [dismissed, setDismissed] = useState(false);
  const [isInstalling, setIsInstalling] = useState(false);

  // Check if user previously dismissed the prompt
  useEffect(() => {
    const dismissed = localStorage.getItem('pwa-install-dismissed');
    if (dismissed) {
      const dismissedTime = parseInt(dismissed, 10);
      const daysSinceDismissed = (Date.now() - dismissedTime) / (1000 * 60 * 60 * 24);

      // Show again after 7 days
      if (daysSinceDismissed < 7) {
        setDismissed(true);
      } else {
        localStorage.removeItem('pwa-install-dismissed');
      }
    }
  }, []);

  const handleInstall = async () => {
    setIsInstalling(true);
    const accepted = await promptInstall();

    if (accepted) {
      // Installation accepted
      localStorage.removeItem('pwa-install-dismissed');
    } else {
      // Installation dismissed
      handleDismiss();
    }

    setIsInstalling(false);
  };

  const handleDismiss = () => {
    setDismissed(true);
    localStorage.setItem('pwa-install-dismissed', Date.now().toString());
  };

  // Don't show if already installed or dismissed
  if (isInstalled || !isInstallable || dismissed) {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 max-w-md animate-slide-up">
      <div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white rounded-lg shadow-2xl p-4 border border-blue-500">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0">
            <Smartphone className="w-6 h-6" />
          </div>

          <div className="flex-1">
            <h3 className="font-semibold text-lg mb-1">Install DB Backup Manager</h3>
            <p className="text-sm text-blue-100 mb-3">
              Install our app for quick access, offline support, and real-time notifications
              for your backup operations.
            </p>

            <div className="flex gap-2">
              <button
                onClick={handleInstall}
                disabled={isInstalling}
                className="flex items-center gap-2 bg-white text-blue-700 px-4 py-2 rounded-lg font-medium hover:bg-blue-50 transition-colors disabled:opacity-50"
              >
                <Download className="w-4 h-4" />
                {isInstalling ? 'Installing...' : 'Install'}
              </button>

              <button
                onClick={handleDismiss}
                className="text-blue-100 hover:text-white px-3 py-2 rounded-lg hover:bg-blue-600 transition-colors"
              >
                Not now
              </button>
            </div>
          </div>

          <button
            onClick={handleDismiss}
            className="flex-shrink-0 text-blue-200 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  );
}
