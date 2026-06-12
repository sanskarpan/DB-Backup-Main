'use client'

import React, { useEffect, useState } from 'react'
import { Zap, ZapOff } from 'lucide-react'

export type MotionPreference = 'auto' | 'full' | 'reduced' | 'none'

interface ReducedMotionProps {
  onChange?: (preference: MotionPreference) => void
  defaultPreference?: MotionPreference
}

export function ReducedMotion({ onChange, defaultPreference = 'auto' }: ReducedMotionProps) {
  const [motionPreference, setMotionPreference] = useState<MotionPreference>(defaultPreference)
  const [systemPreference, setSystemPreference] = useState<'reduce' | 'no-preference'>('no-preference')

  // Detect system motion preference
  useEffect(() => {
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')

    const updateSystemPreference = () => {
      setSystemPreference(motionQuery.matches ? 'reduce' : 'no-preference')
    }

    updateSystemPreference()
    motionQuery.addEventListener('change', updateSystemPreference)

    return () => {
      motionQuery.removeEventListener('change', updateSystemPreference)
    }
  }, [])

  // Load saved preference
  useEffect(() => {
    const saved = localStorage.getItem('motion-preference')
    if (saved && (saved === 'auto' || saved === 'full' || saved === 'reduced' || saved === 'none')) {
      setMotionPreference(saved as MotionPreference)
    }
  }, [])

  // Apply motion preference
  useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-motion-preference', motionPreference)

    if (motionPreference === 'auto') {
      if (systemPreference === 'reduce') {
        root.classList.add('reduce-motion')
        root.classList.remove('no-motion')
      } else {
        root.classList.remove('reduce-motion', 'no-motion')
      }
    } else if (motionPreference === 'reduced') {
      root.classList.add('reduce-motion')
      root.classList.remove('no-motion')
    } else if (motionPreference === 'none') {
      root.classList.add('reduce-motion', 'no-motion')
    } else {
      root.classList.remove('reduce-motion', 'no-motion')
    }
  }, [motionPreference, systemPreference])

  const handlePreferenceChange = (preference: MotionPreference) => {
    setMotionPreference(preference)
    localStorage.setItem('motion-preference', preference)
    onChange?.(preference)

    // Announce to screen readers
    const announcement = document.createElement('div')
    announcement.setAttribute('role', 'status')
    announcement.setAttribute('aria-live', 'polite')
    announcement.className = 'sr-only'
    announcement.textContent = `Motion preference changed to ${preference}`
    document.body.appendChild(announcement)
    setTimeout(() => document.body.removeChild(announcement), 1000)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-gray-900 dark:text-gray-100">
          Motion Preference
        </label>
        <span className="text-xs text-gray-500">
          {motionPreference === 'auto' && `(System: ${systemPreference})`}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <button
          onClick={() => handlePreferenceChange('auto')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            motionPreference === 'auto'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={motionPreference === 'auto'}
        >
          <Zap className="h-6 w-6" />
          <span className="text-sm font-medium">Auto</span>
          <span className="text-xs text-gray-500">System</span>
        </button>

        <button
          onClick={() => handlePreferenceChange('full')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            motionPreference === 'full'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={motionPreference === 'full'}
        >
          <Zap className="h-6 w-6 animate-pulse" />
          <span className="text-sm font-medium">Full</span>
          <span className="text-xs text-gray-500">Animations</span>
        </button>

        <button
          onClick={() => handlePreferenceChange('reduced')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            motionPreference === 'reduced'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={motionPreference === 'reduced'}
        >
          <ZapOff className="h-6 w-6" />
          <span className="text-sm font-medium">Reduced</span>
          <span className="text-xs text-gray-500">Minimal</span>
        </button>

        <button
          onClick={() => handlePreferenceChange('none')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            motionPreference === 'none'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={motionPreference === 'none'}
        >
          <ZapOff className="h-6 w-6 opacity-50" />
          <span className="text-sm font-medium">None</span>
          <span className="text-xs text-gray-500">No motion</span>
        </button>
      </div>

      {/* Preview */}
      <div className="rounded-lg border-2 border-gray-200 p-4 dark:border-gray-700">
        <p className="mb-3 text-sm font-medium">Preview:</p>
        <div className="space-y-3">
          <div className="h-8 w-24 animate-pulse rounded bg-blue-600"></div>
          <div className="h-8 w-32 animate-bounce rounded bg-green-600"></div>
          <div className="h-8 w-40 animate-spin rounded bg-purple-600"></div>
        </div>
        <p className="mt-3 text-xs text-gray-500">
          Animations will be {motionPreference === 'none' ? 'completely disabled' : motionPreference === 'reduced' ? 'minimized' : 'fully enabled'}
        </p>
      </div>
    </div>
  )
}

// Hook for using reduced motion
export function useReducedMotion() {
  const [preference, setPreference] = useState<MotionPreference>('auto')
  const [systemPreference, setSystemPreference] = useState<'reduce' | 'no-preference'>('no-preference')

  useEffect(() => {
    const saved = localStorage.getItem('motion-preference')
    if (saved && (saved === 'auto' || saved === 'full' || saved === 'reduced' || saved === 'none')) {
      setPreference(saved as MotionPreference)
    }

    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')

    const updateSystemPreference = () => {
      setSystemPreference(motionQuery.matches ? 'reduce' : 'no-preference')
    }

    updateSystemPreference()
    motionQuery.addEventListener('change', updateSystemPreference)

    return () => {
      motionQuery.removeEventListener('change', updateSystemPreference)
    }
  }, [])

  const setMotionPreference = (newPreference: MotionPreference) => {
    setPreference(newPreference)
    localStorage.setItem('motion-preference', newPreference)
    document.documentElement.setAttribute('data-motion-preference', newPreference)

    if (newPreference === 'auto') {
      if (systemPreference === 'reduce') {
        document.documentElement.classList.add('reduce-motion')
      } else {
        document.documentElement.classList.remove('reduce-motion', 'no-motion')
      }
    } else if (newPreference === 'reduced') {
      document.documentElement.classList.add('reduce-motion')
      document.documentElement.classList.remove('no-motion')
    } else if (newPreference === 'none') {
      document.documentElement.classList.add('reduce-motion', 'no-motion')
    } else {
      document.documentElement.classList.remove('reduce-motion', 'no-motion')
    }
  }

  const shouldReduceMotion =
    preference === 'reduced' ||
    preference === 'none' ||
    (preference === 'auto' && systemPreference === 'reduce')

  return {
    preference,
    systemPreference,
    setMotionPreference,
    shouldReduceMotion,
    isMotionDisabled: preference === 'none',
  }
}

// CSS for reduced motion
export const reducedMotionStyles = `
  /* Reduced Motion Styles */
  .reduce-motion *,
  .reduce-motion *::before,
  .reduce-motion *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }

  .no-motion *,
  .no-motion *::before,
  .no-motion *::after {
    animation: none !important;
    transition: none !important;
    scroll-behavior: auto !important;
  }

  /* Respect system preference */
  @media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 0.01ms !important;
      scroll-behavior: auto !important;
    }
  }
`
