'use client'

import React, { useEffect, useState } from 'react'
import { Palette } from 'lucide-react'

export type ColorBlindType = 'none' | 'protanopia' | 'deuteranopia' | 'tritanopia' | 'achromatopsia'

interface ColorBlindModeProps {
  onChange?: (type: ColorBlindType) => void
  defaultType?: ColorBlindType
}

const colorBlindInfo = {
  none: {
    name: 'Normal Vision',
    description: 'No color blindness',
    prevalence: 'Majority',
  },
  protanopia: {
    name: 'Protanopia',
    description: 'Red-blind (no red cones)',
    prevalence: '~1% of males',
  },
  deuteranopia: {
    name: 'Deuteranopia',
    description: 'Green-blind (no green cones)',
    prevalence: '~1% of males',
  },
  tritanopia: {
    name: 'Tritanopia',
    description: 'Blue-blind (no blue cones)',
    prevalence: '~0.001%',
  },
  achromatopsia: {
    name: 'Achromatopsia',
    description: 'Complete color blindness',
    prevalence: '~0.003%',
  },
}

export function ColorBlindMode({ onChange, defaultType = 'none' }: ColorBlindModeProps) {
  const [colorBlindType, setColorBlindType] = useState<ColorBlindType>(defaultType)

  // Load saved preference
  useEffect(() => {
    const saved = localStorage.getItem('color-blind-type')
    if (saved && Object.keys(colorBlindInfo).includes(saved)) {
      setColorBlindType(saved as ColorBlindType)
    }
  }, [])

  // Apply color blind mode
  useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-color-blind-type', colorBlindType)

    // Remove all color blind classes
    Object.keys(colorBlindInfo).forEach((type) => {
      root.classList.remove(`color-blind-${type}`)
    })

    // Add current class
    if (colorBlindType !== 'none') {
      root.classList.add(`color-blind-${colorBlindType}`)
    }
  }, [colorBlindType])

  const handleTypeChange = (type: ColorBlindType) => {
    setColorBlindType(type)
    localStorage.setItem('color-blind-type', type)
    onChange?.(type)

    // Announce to screen readers
    const announcement = document.createElement('div')
    announcement.setAttribute('role', 'status')
    announcement.setAttribute('aria-live', 'polite')
    announcement.className = 'sr-only'
    announcement.textContent = `Color blind mode changed to ${colorBlindInfo[type].name}`
    document.body.appendChild(announcement)
    setTimeout(() => document.body.removeChild(announcement), 1000)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <label className="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          <Palette className="h-4 w-4" />
          Color Blind Mode
        </label>
      </div>

      <div className="grid gap-3">
        {(Object.keys(colorBlindInfo) as ColorBlindType[]).map((type) => (
          <button
            key={type}
            onClick={() => handleTypeChange(type)}
            className={`flex items-start gap-3 rounded-lg border-2 p-4 text-left transition-all ${
              colorBlindType === type
                ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
            }`}
            aria-pressed={colorBlindType === type}
          >
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="font-medium">{colorBlindInfo[type].name}</span>
                {colorBlindType === type && (
                  <span className="rounded bg-blue-600 px-2 py-0.5 text-xs font-medium text-white">
                    Active
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {colorBlindInfo[type].description}
              </p>
              <p className="mt-1 text-xs text-gray-500">{colorBlindInfo[type].prevalence}</p>
            </div>
          </button>
        ))}
      </div>

      {/* Color Preview */}
      <div className="rounded-lg border-2 border-gray-200 p-4 dark:border-gray-700">
        <p className="mb-3 text-sm font-medium">Color Preview:</p>
        <div className="grid grid-cols-6 gap-2">
          <div className="h-12 rounded bg-red-600" title="Red"></div>
          <div className="h-12 rounded bg-green-600" title="Green"></div>
          <div className="h-12 rounded bg-blue-600" title="Blue"></div>
          <div className="h-12 rounded bg-yellow-500" title="Yellow"></div>
          <div className="h-12 rounded bg-purple-600" title="Purple"></div>
          <div className="h-12 rounded bg-orange-500" title="Orange"></div>
        </div>
        <p className="mt-3 text-xs text-gray-500">
          Colors are adjusted for {colorBlindInfo[colorBlindType].name.toLowerCase()}
        </p>
      </div>

      {/* Information */}
      {colorBlindType !== 'none' && (
        <div className="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
          <p className="text-sm text-blue-900 dark:text-blue-100">
            <strong>Note:</strong> This mode adjusts colors to be more distinguishable for {colorBlindInfo[colorBlindType].name.toLowerCase()}.
            Critical information should never rely on color alone.
          </p>
        </div>
      )}
    </div>
  )
}

// Hook for using color blind mode
export function useColorBlindMode() {
  const [type, setType] = useState<ColorBlindType>('none')

  useEffect(() => {
    const saved = localStorage.getItem('color-blind-type')
    if (saved && Object.keys(colorBlindInfo).includes(saved)) {
      setType(saved as ColorBlindType)
    }
  }, [])

  const setColorBlindType = (newType: ColorBlindType) => {
    setType(newType)
    localStorage.setItem('color-blind-type', newType)
    document.documentElement.setAttribute('data-color-blind-type', newType)

    // Remove all color blind classes
    Object.keys(colorBlindInfo).forEach((t) => {
      document.documentElement.classList.remove(`color-blind-${t}`)
    })

    // Add current class
    if (newType !== 'none') {
      document.documentElement.classList.add(`color-blind-${newType}`)
    }
  }

  return {
    type,
    setColorBlindType,
    isColorBlind: type !== 'none',
  }
}

// CSS for color blind modes using SVG filters
export const colorBlindStyles = `
  /* Color Blind Mode Filters */
  .color-blind-protanopia {
    filter: url(#protanopia-filter);
  }

  .color-blind-deuteranopia {
    filter: url(#deuteranopia-filter);
  }

  .color-blind-tritanopia {
    filter: url(#tritanopia-filter);
  }

  .color-blind-achromatopsia {
    filter: grayscale(100%);
  }
`

// SVG Filter Definitions
export const ColorBlindFilters = () => (
  <svg style={{ position: 'absolute', width: 0, height: 0 }} aria-hidden="true">
    <defs>
      {/* Protanopia (Red-blind) */}
      <filter id="protanopia-filter">
        <feColorMatrix
          type="matrix"
          values="0.567, 0.433, 0,     0, 0
                  0.558, 0.442, 0,     0, 0
                  0,     0.242, 0.758, 0, 0
                  0,     0,     0,     1, 0"
        />
      </filter>

      {/* Deuteranopia (Green-blind) */}
      <filter id="deuteranopia-filter">
        <feColorMatrix
          type="matrix"
          values="0.625, 0.375, 0,   0, 0
                  0.7,   0.3,   0,   0, 0
                  0,     0.3,   0.7, 0, 0
                  0,     0,     0,   1, 0"
        />
      </filter>

      {/* Tritanopia (Blue-blind) */}
      <filter id="tritanopia-filter">
        <feColorMatrix
          type="matrix"
          values="0.95, 0.05,  0,     0, 0
                  0,    0.433, 0.567, 0, 0
                  0,    0.475, 0.525, 0, 0
                  0,    0,     0,     1, 0"
        />
      </filter>
    </defs>
  </svg>
)
