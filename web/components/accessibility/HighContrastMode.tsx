'use client'

import React, { useEffect, useState } from 'react'
import { Monitor, Sun, Moon } from 'lucide-react'

export type ContrastMode = 'auto' | 'normal' | 'high' | 'ultra'

interface HighContrastModeProps {
  onChange?: (mode: ContrastMode) => void
  defaultMode?: ContrastMode
}

export function HighContrastMode({ onChange, defaultMode = 'auto' }: HighContrastModeProps) {
  const [contrastMode, setContrastMode] = useState<ContrastMode>(defaultMode)
  const [systemPreference, setSystemPreference] = useState<'light' | 'dark' | 'high-contrast'>('light')

  // Detect system contrast preference
  useEffect(() => {
    // Check for high contrast preference
    const highContrastQuery = window.matchMedia('(prefers-contrast: more)')
    const darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const updateSystemPreference = () => {
      if (highContrastQuery.matches) {
        setSystemPreference('high-contrast')
      } else if (darkModeQuery.matches) {
        setSystemPreference('dark')
      } else {
        setSystemPreference('light')
      }
    }

    updateSystemPreference()

    highContrastQuery.addEventListener('change', updateSystemPreference)
    darkModeQuery.addEventListener('change', updateSystemPreference)

    return () => {
      highContrastQuery.removeEventListener('change', updateSystemPreference)
      darkModeQuery.removeEventListener('change', updateSystemPreference)
    }
  }, [])

  // Apply contrast mode
  useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-contrast-mode', contrastMode)

    // Load mode from localStorage
    const saved = localStorage.getItem('contrast-mode')
    if (saved && (saved === 'auto' || saved === 'normal' || saved === 'high' || saved === 'ultra')) {
      setContrastMode(saved as ContrastMode)
    }

    // Apply auto mode
    if (contrastMode === 'auto') {
      if (systemPreference === 'high-contrast') {
        root.classList.add('high-contrast')
        root.classList.remove('ultra-contrast')
      } else {
        root.classList.remove('high-contrast', 'ultra-contrast')
      }
    } else if (contrastMode === 'high') {
      root.classList.add('high-contrast')
      root.classList.remove('ultra-contrast')
    } else if (contrastMode === 'ultra') {
      root.classList.add('high-contrast', 'ultra-contrast')
    } else {
      root.classList.remove('high-contrast', 'ultra-contrast')
    }
  }, [contrastMode, systemPreference])

  const handleModeChange = (mode: ContrastMode) => {
    setContrastMode(mode)
    localStorage.setItem('contrast-mode', mode)
    onChange?.(mode)

    // Announce change to screen readers
    const announcement = document.createElement('div')
    announcement.setAttribute('role', 'status')
    announcement.setAttribute('aria-live', 'polite')
    announcement.className = 'sr-only'
    announcement.textContent = `Contrast mode changed to ${mode}`
    document.body.appendChild(announcement)
    setTimeout(() => document.body.removeChild(announcement), 1000)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-gray-900 dark:text-gray-100">
          Contrast Mode
        </label>
        <span className="text-xs text-gray-500">
          {contrastMode === 'auto' && `(System: ${systemPreference})`}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <button
          onClick={() => handleModeChange('auto')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            contrastMode === 'auto'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={contrastMode === 'auto'}
        >
          <Monitor className="h-6 w-6" />
          <span className="text-sm font-medium">Auto</span>
          <span className="text-xs text-gray-500">System</span>
        </button>

        <button
          onClick={() => handleModeChange('normal')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            contrastMode === 'normal'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={contrastMode === 'normal'}
        >
          <Sun className="h-6 w-6" />
          <span className="text-sm font-medium">Normal</span>
          <span className="text-xs text-gray-500">Default</span>
        </button>

        <button
          onClick={() => handleModeChange('high')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            contrastMode === 'high'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={contrastMode === 'high'}
        >
          <div className="flex h-6 w-6 items-center justify-center rounded bg-black text-white">
            <span className="text-xs font-bold">A</span>
          </div>
          <span className="text-sm font-medium">High</span>
          <span className="text-xs text-gray-500">7:1 ratio</span>
        </button>

        <button
          onClick={() => handleModeChange('ultra')}
          className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-all ${
            contrastMode === 'ultra'
              ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
          }`}
          aria-pressed={contrastMode === 'ultra'}
        >
          <div className="flex h-6 w-6 items-center justify-center rounded border-2 border-black bg-white text-black">
            <span className="text-xs font-bold">A</span>
          </div>
          <span className="text-sm font-medium">Ultra</span>
          <span className="text-xs text-gray-500">21:1 ratio</span>
        </button>
      </div>

      {/* Preview */}
      <div className="rounded-lg border-2 border-gray-200 p-4 dark:border-gray-700">
        <p className="mb-2 text-sm font-medium">Preview:</p>
        <div className="space-y-2">
          <div className="rounded bg-white p-3 text-gray-900 dark:bg-gray-900 dark:text-gray-100">
            Normal text on background
          </div>
          <div className="rounded bg-blue-600 p-3 text-white">
            Primary button text
          </div>
          <div className="rounded bg-gray-200 p-3 text-gray-700 dark:bg-gray-700 dark:text-gray-200">
            Secondary button text
          </div>
        </div>
      </div>
    </div>
  )
}

// Hook for using contrast mode
export function useContrastMode() {
  const [mode, setMode] = useState<ContrastMode>('auto')
  const [systemPreference, setSystemPreference] = useState<'light' | 'dark' | 'high-contrast'>('light')

  useEffect(() => {
    const saved = localStorage.getItem('contrast-mode')
    if (saved && (saved === 'auto' || saved === 'normal' || saved === 'high' || saved === 'ultra')) {
      setMode(saved as ContrastMode)
    }

    const highContrastQuery = window.matchMedia('(prefers-contrast: more)')
    const darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const updateSystemPreference = () => {
      if (highContrastQuery.matches) {
        setSystemPreference('high-contrast')
      } else if (darkModeQuery.matches) {
        setSystemPreference('dark')
      } else {
        setSystemPreference('light')
      }
    }

    updateSystemPreference()

    highContrastQuery.addEventListener('change', updateSystemPreference)
    darkModeQuery.addEventListener('change', updateSystemPreference)

    return () => {
      highContrastQuery.removeEventListener('change', updateSystemPreference)
      darkModeQuery.removeEventListener('change', updateSystemPreference)
    }
  }, [])

  const setContrastMode = (newMode: ContrastMode) => {
    setMode(newMode)
    localStorage.setItem('contrast-mode', newMode)
    document.documentElement.setAttribute('data-contrast-mode', newMode)

    if (newMode === 'auto') {
      if (systemPreference === 'high-contrast') {
        document.documentElement.classList.add('high-contrast')
      } else {
        document.documentElement.classList.remove('high-contrast', 'ultra-contrast')
      }
    } else if (newMode === 'high') {
      document.documentElement.classList.add('high-contrast')
      document.documentElement.classList.remove('ultra-contrast')
    } else if (newMode === 'ultra') {
      document.documentElement.classList.add('high-contrast', 'ultra-contrast')
    } else {
      document.documentElement.classList.remove('high-contrast', 'ultra-contrast')
    }
  }

  return {
    mode,
    systemPreference,
    setContrastMode,
    isHighContrast: mode === 'high' || mode === 'ultra' || (mode === 'auto' && systemPreference === 'high-contrast'),
  }
}

// CSS-in-JS styles for high contrast mode
export const highContrastStyles = `
  /* High Contrast Mode Styles */
  .high-contrast {
    --contrast-bg: #ffffff;
    --contrast-fg: #000000;
    --contrast-border: #000000;
    --contrast-focus: #0066cc;
    --contrast-link: #0000ee;
    --contrast-visited: #551a8b;
    --contrast-error: #cc0000;
    --contrast-success: #006600;
    --contrast-warning: #cc6600;
  }

  .high-contrast.dark {
    --contrast-bg: #000000;
    --contrast-fg: #ffffff;
    --contrast-border: #ffffff;
    --contrast-focus: #66ccff;
    --contrast-link: #66ccff;
    --contrast-visited: #cc99ff;
    --contrast-error: #ff6666;
    --contrast-success: #66ff66;
    --contrast-warning: #ffcc66;
  }

  .ultra-contrast {
    filter: contrast(1.5);
  }

  /* Override default colors in high contrast */
  .high-contrast * {
    color: var(--contrast-fg) !important;
    background-color: var(--contrast-bg) !important;
    border-color: var(--contrast-border) !important;
  }

  .high-contrast button,
  .high-contrast input,
  .high-contrast select,
  .high-contrast textarea {
    border-width: 2px !important;
  }

  .high-contrast a {
    color: var(--contrast-link) !important;
    text-decoration: underline !important;
  }

  .high-contrast a:visited {
    color: var(--contrast-visited) !important;
  }

  .high-contrast a:focus,
  .high-contrast button:focus,
  .high-contrast input:focus,
  .high-contrast select:focus,
  .high-contrast textarea:focus {
    outline: 3px solid var(--contrast-focus) !important;
    outline-offset: 2px !important;
  }

  /* Error states */
  .high-contrast .error,
  .high-contrast [aria-invalid="true"] {
    color: var(--contrast-error) !important;
    border-color: var(--contrast-error) !important;
  }

  /* Success states */
  .high-contrast .success {
    color: var(--contrast-success) !important;
  }

  /* Warning states */
  .high-contrast .warning {
    color: var(--contrast-warning) !important;
  }
`
