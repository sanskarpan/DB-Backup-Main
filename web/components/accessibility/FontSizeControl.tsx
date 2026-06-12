'use client'

import React, { useEffect, useState } from 'react'
import { Type, Plus, Minus, RotateCcw } from 'lucide-react'

export type FontSize = 'xs' | 'sm' | 'base' | 'lg' | 'xl' | '2xl' | '3xl'

interface FontSizeControlProps {
  onChange?: (size: FontSize) => void
  defaultSize?: FontSize
  minSize?: FontSize
  maxSize?: FontSize
}

const fontSizeMap: Record<FontSize, number> = {
  xs: 12,
  sm: 14,
  base: 16,
  lg: 18,
  xl: 20,
  '2xl': 24,
  '3xl': 30,
}

const fontSizeOrder: FontSize[] = ['xs', 'sm', 'base', 'lg', 'xl', '2xl', '3xl']

export function FontSizeControl({
  onChange,
  defaultSize = 'base',
  minSize = 'xs',
  maxSize = '3xl',
}: FontSizeControlProps) {
  const [fontSize, setFontSize] = useState<FontSize>(defaultSize)
  const [customSize, setCustomSize] = useState<number>(16)
  const [useCustom, setUseCustom] = useState(false)

  const minIndex = fontSizeOrder.indexOf(minSize)
  const maxIndex = fontSizeOrder.indexOf(maxSize)

  // Load saved preference
  useEffect(() => {
    const saved = localStorage.getItem('font-size')
    const savedCustom = localStorage.getItem('font-size-custom')
    const savedUseCustom = localStorage.getItem('use-custom-font-size')

    if (savedUseCustom === 'true' && savedCustom) {
      setUseCustom(true)
      setCustomSize(parseInt(savedCustom, 10))
    } else if (saved && fontSizeOrder.includes(saved as FontSize)) {
      setFontSize(saved as FontSize)
    }
  }, [])

  // Apply font size
  useEffect(() => {
    const root = document.documentElement

    if (useCustom) {
      root.style.fontSize = `${customSize}px`
      root.setAttribute('data-font-size', 'custom')
    } else {
      root.style.fontSize = `${fontSizeMap[fontSize]}px`
      root.setAttribute('data-font-size', fontSize)
    }
  }, [fontSize, customSize, useCustom])

  const handleSizeChange = (size: FontSize) => {
    setFontSize(size)
    setUseCustom(false)
    localStorage.setItem('font-size', size)
    localStorage.setItem('use-custom-font-size', 'false')
    onChange?.(size)

    // Announce to screen readers
    announceChange(`Font size changed to ${fontSizeMap[size]} pixels`)
  }

  const handleCustomSizeChange = (size: number) => {
    const clamped = Math.max(8, Math.min(48, size))
    setCustomSize(clamped)
    setUseCustom(true)
    localStorage.setItem('font-size-custom', clamped.toString())
    localStorage.setItem('use-custom-font-size', 'true')

    announceChange(`Font size changed to ${clamped} pixels`)
  }

  const increase = () => {
    if (useCustom) {
      handleCustomSizeChange(customSize + 2)
    } else {
      const currentIndex = fontSizeOrder.indexOf(fontSize)
      if (currentIndex < maxIndex) {
        handleSizeChange(fontSizeOrder[currentIndex + 1])
      }
    }
  }

  const decrease = () => {
    if (useCustom) {
      handleCustomSizeChange(customSize - 2)
    } else {
      const currentIndex = fontSizeOrder.indexOf(fontSize)
      if (currentIndex > minIndex) {
        handleSizeChange(fontSizeOrder[currentIndex - 1])
      }
    }
  }

  const reset = () => {
    handleSizeChange(defaultSize)
  }

  const announceChange = (message: string) => {
    const announcement = document.createElement('div')
    announcement.setAttribute('role', 'status')
    announcement.setAttribute('aria-live', 'polite')
    announcement.className = 'sr-only'
    announcement.textContent = message
    document.body.appendChild(announcement)
    setTimeout(() => document.body.removeChild(announcement), 1000)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <label className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          <Type className="h-4 w-4" />
          Font Size
        </label>
        <span className="text-xs text-gray-500">
          {useCustom ? `${customSize}px` : `${fontSizeMap[fontSize]}px`}
        </span>
      </div>

      {/* Quick Controls */}
      <div className="flex items-center gap-2">
        <button
          onClick={decrease}
          className="flex h-10 w-10 items-center justify-center rounded-lg border-2 border-gray-200 transition-colors hover:bg-gray-100 disabled:opacity-50 dark:border-gray-700 dark:hover:bg-gray-800"
          disabled={!useCustom && fontSizeOrder.indexOf(fontSize) <= minIndex}
          aria-label="Decrease font size"
        >
          <Minus className="h-5 w-5" />
        </button>

        <button
          onClick={reset}
          className="flex h-10 flex-1 items-center justify-center gap-2 rounded-lg border-2 border-gray-200 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
          aria-label="Reset font size to default"
        >
          <RotateCcw className="h-4 w-4" />
          <span className="text-sm font-medium">Reset</span>
        </button>

        <button
          onClick={increase}
          className="flex h-10 w-10 items-center justify-center rounded-lg border-2 border-gray-200 transition-colors hover:bg-gray-100 disabled:opacity-50 dark:border-gray-700 dark:hover:bg-gray-800"
          disabled={!useCustom && fontSizeOrder.indexOf(fontSize) >= maxIndex}
          aria-label="Increase font size"
        >
          <Plus className="h-5 w-5" />
        </button>
      </div>

      {/* Preset Sizes */}
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
        {fontSizeOrder.slice(minIndex, maxIndex + 1).map((size) => (
          <button
            key={size}
            onClick={() => handleSizeChange(size)}
            className={`flex flex-col items-center gap-1 rounded-lg border-2 p-3 transition-all ${
              !useCustom && fontSize === size
                ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
            }`}
            aria-pressed={!useCustom && fontSize === size}
          >
            <Type className="h-4 w-4" style={{ fontSize: `${fontSizeMap[size] / 2}px` }} />
            <span className="text-xs font-medium">{fontSizeMap[size]}px</span>
          </button>
        ))}
      </div>

      {/* Custom Size Slider */}
      <div className="space-y-2">
        <label htmlFor="custom-font-size" className="text-sm text-gray-700 dark:text-gray-300">
          Custom Size
        </label>
        <input
          id="custom-font-size"
          type="range"
          min="8"
          max="48"
          step="2"
          value={customSize}
          onChange={(e) => handleCustomSizeChange(parseInt(e.target.value, 10))}
          className="w-full"
          aria-label="Custom font size slider"
          aria-valuemin={8}
          aria-valuemax={48}
          aria-valuenow={customSize}
          aria-valuetext={`${customSize} pixels`}
        />
        <div className="flex justify-between text-xs text-gray-500">
          <span>8px</span>
          <span className="font-medium">{customSize}px</span>
          <span>48px</span>
        </div>
      </div>

      {/* Preview */}
      <div className="rounded-lg border-2 border-gray-200 p-4 dark:border-gray-700">
        <p className="mb-3 text-sm font-medium">Preview:</p>
        <div className="space-y-2">
          <p className="text-sm">Small text (0.875em)</p>
          <p>Regular text (1em)</p>
          <p className="text-lg font-medium">Large text (1.125em)</p>
          <h3 className="text-xl font-bold">Heading text (1.25em)</h3>
        </div>
      </div>
    </div>
  )
}

// Hook for using font size
export function useFontSize() {
  const [fontSize, setFontSize] = useState<FontSize>('base')
  const [customSize, setCustomSize] = useState<number>(16)
  const [useCustom, setUseCustom] = useState(false)

  useEffect(() => {
    const saved = localStorage.getItem('font-size')
    const savedCustom = localStorage.getItem('font-size-custom')
    const savedUseCustom = localStorage.getItem('use-custom-font-size')

    if (savedUseCustom === 'true' && savedCustom) {
      setUseCustom(true)
      setCustomSize(parseInt(savedCustom, 10))
    } else if (saved && fontSizeOrder.includes(saved as FontSize)) {
      setFontSize(saved as FontSize)
    }
  }, [])

  const setFontSizeSetting = (size: FontSize) => {
    setFontSize(size)
    setUseCustom(false)
    localStorage.setItem('font-size', size)
    localStorage.setItem('use-custom-font-size', 'false')
    document.documentElement.style.fontSize = `${fontSizeMap[size]}px`
    document.documentElement.setAttribute('data-font-size', size)
  }

  const setCustomFontSize = (size: number) => {
    const clamped = Math.max(8, Math.min(48, size))
    setCustomSize(clamped)
    setUseCustom(true)
    localStorage.setItem('font-size-custom', clamped.toString())
    localStorage.setItem('use-custom-font-size', 'true')
    document.documentElement.style.fontSize = `${clamped}px`
    document.documentElement.setAttribute('data-font-size', 'custom')
  }

  const increaseFontSize = () => {
    if (useCustom) {
      setCustomFontSize(customSize + 2)
    } else {
      const currentIndex = fontSizeOrder.indexOf(fontSize)
      if (currentIndex < fontSizeOrder.length - 1) {
        setFontSizeSetting(fontSizeOrder[currentIndex + 1])
      }
    }
  }

  const decreaseFontSize = () => {
    if (useCustom) {
      setCustomFontSize(customSize - 2)
    } else {
      const currentIndex = fontSizeOrder.indexOf(fontSize)
      if (currentIndex > 0) {
        setFontSizeSetting(fontSizeOrder[currentIndex - 1])
      }
    }
  }

  const resetFontSize = () => {
    setFontSizeSetting('base')
  }

  return {
    fontSize,
    customSize,
    useCustom,
    setFontSize: setFontSizeSetting,
    setCustomFontSize,
    increaseFontSize,
    decreaseFontSize,
    resetFontSize,
    actualSize: useCustom ? customSize : fontSizeMap[fontSize],
  }
}

// Keyboard shortcuts for font size
export function setupFontSizeKeyboardShortcuts() {
  const handleKeyDown = (e: KeyboardEvent) => {
    // Ctrl/Cmd + Plus: Increase font size
    if ((e.ctrlKey || e.metaKey) && (e.key === '=' || e.key === '+')) {
      e.preventDefault()
      const event = new CustomEvent('font-size-increase')
      window.dispatchEvent(event)
    }

    // Ctrl/Cmd + Minus: Decrease font size
    if ((e.ctrlKey || e.metaKey) && e.key === '-') {
      e.preventDefault()
      const event = new CustomEvent('font-size-decrease')
      window.dispatchEvent(event)
    }

    // Ctrl/Cmd + 0: Reset font size
    if ((e.ctrlKey || e.metaKey) && e.key === '0') {
      e.preventDefault()
      const event = new CustomEvent('font-size-reset')
      window.dispatchEvent(event)
    }
  }

  document.addEventListener('keydown', handleKeyDown)

  return () => {
    document.removeEventListener('keydown', handleKeyDown)
  }
}
