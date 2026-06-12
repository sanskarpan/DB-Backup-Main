/**
 * Share Target Page
 * Handles shared content from other apps via Web Share Target API
 */

'use client';

import React, { useEffect, useState } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Share2, Upload, FileText, X, Check } from 'lucide-react';
import { usePWA } from '@/components/providers/pwa-provider';

export default function SharePage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { db } = usePWA();

  const [sharedData, setSharedData] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadSharedData = async () => {
      try {
        // Check if data was shared
        const wasShared = searchParams.get('shared');

        if (wasShared === 'true') {
          // Get shared data from cache (stored by service worker)
          const cache = await caches.open('shared-data');
          const response = await cache.match('/shared-data/latest');

          if (response) {
            const data = await response.json();
            setSharedData(data);
          }
        }
      } catch (err) {
        console.error('Failed to load shared data:', err);
        setError('Failed to load shared content');
      } finally {
        setIsLoading(false);
      }
    };

    loadSharedData();
  }, [searchParams]);

  const handleImport = async () => {
    setIsProcessing(true);
    setError(null);

    try {
      // Process the shared data
      // This is a placeholder - implement actual import logic based on your needs

      if (sharedData.files > 0) {
        // Handle file import
        // You would need to implement file handling logic here
        console.log('Importing files:', sharedData);
      } else if (sharedData.text) {
        // Handle text import (e.g., SQL dump)
        console.log('Importing text:', sharedData.text);
      } else if (sharedData.url) {
        // Handle URL import
        console.log('Importing from URL:', sharedData.url);
      }

      // Store in offline DB
      await db.addToQueue({
        id: `share-${Date.now()}`,
        url: '/api/backups/import',
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: sharedData,
        timestamp: Date.now(),
        retries: 0,
      });

      // Clear the shared data cache
      const cache = await caches.open('shared-data');
      await cache.delete('/shared-data/latest');

      // Redirect to backups page
      router.push('/backups?imported=true');
    } catch (err) {
      console.error('Failed to import shared data:', err);
      setError('Failed to import content. Please try again.');
    } finally {
      setIsProcessing(false);
    }
  };

  const handleCancel = () => {
    router.push('/backups');
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-12 w-12 border-4 border-blue-600 border-t-transparent"></div>
          <p className="mt-4 text-gray-600">Loading shared content...</p>
        </div>
      </div>
    );
  }

  if (!sharedData) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-full mb-4">
            <X className="w-8 h-8 text-red-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            No Content to Share
          </h1>
          <p className="text-gray-600 mb-6">
            No shared content was found. Please try sharing again.
          </p>
          <button
            onClick={handleCancel}
            className="w-full bg-gray-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-gray-700 transition-colors"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="max-w-2xl w-full bg-white rounded-lg shadow-lg p-8">
        {/* Header */}
        <div className="flex items-center gap-3 mb-6">
          <div className="flex-shrink-0 w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
            <Share2 className="w-6 h-6 text-blue-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              Import Shared Content
            </h1>
            <p className="text-sm text-gray-600">
              Review and import the shared backup files
            </p>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mb-6 bg-red-50 border border-red-200 rounded-lg p-4 flex items-center gap-3">
            <X className="w-5 h-5 text-red-600 flex-shrink-0" />
            <span className="text-red-700">{error}</span>
          </div>
        )}

        {/* Shared Content Details */}
        <div className="bg-gray-50 rounded-lg p-6 mb-6">
          <h2 className="font-semibold text-gray-900 mb-4">Content Details</h2>

          <div className="space-y-3">
            {sharedData.title && (
              <div>
                <span className="text-sm text-gray-600">Title:</span>
                <p className="text-gray-900 font-medium">{sharedData.title}</p>
              </div>
            )}

            {sharedData.text && (
              <div>
                <span className="text-sm text-gray-600">Description:</span>
                <p className="text-gray-900">{sharedData.text}</p>
              </div>
            )}

            {sharedData.url && (
              <div>
                <span className="text-sm text-gray-600">URL:</span>
                <p className="text-blue-600 break-all">{sharedData.url}</p>
              </div>
            )}

            {sharedData.files > 0 && (
              <div>
                <span className="text-sm text-gray-600">Files:</span>
                <p className="text-gray-900 font-medium">
                  {sharedData.files} file(s) selected
                </p>
              </div>
            )}
          </div>
        </div>

        {/* File Type Info */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
          <div className="flex items-start gap-3">
            <FileText className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
            <div>
              <h3 className="font-semibold text-blue-900 mb-1">
                Supported Formats
              </h3>
              <p className="text-sm text-blue-700">
                SQL dumps (.sql), compressed backups (.gz), and database dump files
                (.dump) are supported for import.
              </p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={handleImport}
            disabled={isProcessing}
            className="flex-1 flex items-center justify-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isProcessing ? (
              <>
                <div className="inline-block animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                Importing...
              </>
            ) : (
              <>
                <Upload className="w-5 h-5" />
                Import Content
              </>
            )}
          </button>

          <button
            onClick={handleCancel}
            disabled={isProcessing}
            className="flex-1 bg-gray-200 text-gray-700 px-6 py-3 rounded-lg font-medium hover:bg-gray-300 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
        </div>

        {/* Help Text */}
        <p className="mt-4 text-xs text-gray-500 text-center">
          The import will be queued and processed in the background. You can continue
          using the app while the import completes.
        </p>
      </div>
    </div>
  );
}
