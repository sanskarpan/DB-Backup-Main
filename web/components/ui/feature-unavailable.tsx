'use client'

import { PackageX } from 'lucide-react'
import type { ReactNode } from 'react'

interface FeatureUnavailableProps {
  title: string
  description?: ReactNode
  icon?: ReactNode
}

/**
 * Honest empty-state for features whose backend is not part of this
 * deployment. Instead of endlessly spinning or surfacing fetch errors for
 * routes that do not exist, we clearly state the feature is unavailable.
 */
export function FeatureUnavailable({ title, description, icon }: FeatureUnavailableProps) {
  return (
    <div className="min-h-[60vh] flex items-center justify-center p-8">
      <div className="max-w-md w-full text-center bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-2xl p-10 shadow-sm">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 mb-6">
          {icon ?? <PackageX className="w-8 h-8" />}
        </div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">{title}</h1>
        <p className="text-gray-600 dark:text-gray-400">
          {description ??
            'This feature is not available in this deployment. The backend service that powers it is not configured.'}
        </p>
      </div>
    </div>
  )
}
