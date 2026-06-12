'use client'

import { useState } from 'react'
import { Palette, Layout, Sparkles, Settings } from 'lucide-react'

interface CustomizationOption {
  id: string
  name: string
  description: string
  enabled: boolean
}

export function CustomizationCenter() {
  const [theme, setTheme] = useState<'light' | 'dark' | 'auto'>('auto')
  const [customizations, setCustomizations] = useState<CustomizationOption[]>([
    {
      id: 'compact-mode',
      name: 'Compact Mode',
      description: 'Reduce spacing for a more condensed interface',
      enabled: false,
    },
    {
      id: 'animations',
      name: 'Enable Animations',
      description: 'Show smooth transitions and animations throughout the app',
      enabled: true,
    },
    {
      id: 'tooltips',
      name: 'Detailed Tooltips',
      description: 'Show helpful tooltips when hovering over UI elements',
      enabled: true,
    },
  ])

  const toggleCustomization = (id: string) => {
    setCustomizations((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, enabled: !item.enabled } : item
      )
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Customization Center</h2>
        <p className="text-gray-600">Personalize your backup management experience</p>
      </div>

      {/* Theme Selection */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center mb-4">
          <Palette className="w-5 h-5 text-gray-700 mr-2" />
          <h3 className="text-lg font-semibold text-gray-900">Theme</h3>
        </div>
        <div className="flex space-x-4">
          {(['light', 'dark', 'auto'] as const).map((option) => (
            <button
              key={option}
              onClick={() => setTheme(option)}
              className={`px-6 py-3 rounded-lg border-2 transition-colors capitalize ${
                theme === option
                  ? 'border-primary bg-primary/5 text-primary font-medium'
                  : 'border-gray-300 text-gray-700 hover:border-gray-400'
              }`}
            >
              {option}
            </button>
          ))}
        </div>
      </div>

      {/* UI Customizations */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center mb-4">
          <Layout className="w-5 h-5 text-gray-700 mr-2" />
          <h3 className="text-lg font-semibold text-gray-900">Interface Options</h3>
        </div>
        <div className="space-y-4">
          {customizations.map((option) => (
            <div
              key={option.id}
              className="flex items-center justify-between p-4 rounded-lg border border-gray-200 hover:bg-gray-50 transition-colors"
            >
              <div className="flex-1">
                <h4 className="text-sm font-medium text-gray-900">{option.name}</h4>
                <p className="text-sm text-gray-600 mt-1">{option.description}</p>
              </div>
              <button
                onClick={() => toggleCustomization(option.id)}
                className={`ml-4 relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  option.enabled ? 'bg-primary' : 'bg-gray-300'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    option.enabled ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Preview Section */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center mb-4">
          <Sparkles className="w-5 h-5 text-gray-700 mr-2" />
          <h3 className="text-lg font-semibold text-gray-900">Preview</h3>
        </div>
        <div className="bg-gray-50 rounded-lg p-6 border-2 border-dashed border-gray-300">
          <p className="text-gray-600 text-center">
            Your customizations will be reflected throughout the application
          </p>
        </div>
      </div>
    </div>
  )
}

export default CustomizationCenter
