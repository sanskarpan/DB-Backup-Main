'use client'

import React, { useState } from 'react'
import { Settings, Accessibility, X } from 'lucide-react'
import { HighContrastMode } from './HighContrastMode'
import { FontSizeControl } from './FontSizeControl'
import { ReducedMotion } from './ReducedMotion'
import { ColorBlindMode, ColorBlindFilters } from './ColorBlindMode'
import { SkipLinks } from './SkipLinks'

interface AccessibilitySettingsProps {
  isOpen?: boolean
  onClose?: () => void
}

export function AccessibilitySettings({ isOpen = false, onClose }: AccessibilitySettingsProps) {
  const [activeTab, setActiveTab] = useState<'visual' | 'motion' | 'navigation' | 'audio'>('visual')

  if (!isOpen) return null

  return (
    <>
      {/* Skip Links - Always rendered */}
      <SkipLinks />

      {/* Color Blind Filters - SVG definitions */}
      <ColorBlindFilters />

      {/* Settings Panel */}
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
        <div
          className="relative w-full max-w-4xl max-h-[90vh] overflow-y-auto rounded-lg bg-white shadow-2xl dark:bg-gray-900"
          onClick={(e) => e.stopPropagation()}
          role="dialog"
          aria-labelledby="accessibility-settings-title"
          aria-modal="true"
        >
          {/* Header */}
          <div className="sticky top-0 z-10 flex items-center justify-between border-b border-gray-200 bg-white px-6 py-4 dark:border-gray-700 dark:bg-gray-900">
            <div className="flex items-center gap-3">
              <Accessibility className="h-6 w-6 text-blue-600" />
              <h2 id="accessibility-settings-title" className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Accessibility Settings
              </h2>
            </div>
            <button
              onClick={onClose}
              className="rounded-lg p-2 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
              aria-label="Close accessibility settings"
            >
              <X className="h-6 w-6" />
            </button>
          </div>

          {/* Tabs */}
          <div className="border-b border-gray-200 dark:border-gray-700">
            <nav className="flex gap-2 px-6" aria-label="Accessibility settings tabs">
              <button
                onClick={() => setActiveTab('visual')}
                className={`border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'visual'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400'
                }`}
                aria-selected={activeTab === 'visual'}
                role="tab"
              >
                Visual
              </button>
              <button
                onClick={() => setActiveTab('motion')}
                className={`border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'motion'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400'
                }`}
                aria-selected={activeTab === 'motion'}
                role="tab"
              >
                Motion
              </button>
              <button
                onClick={() => setActiveTab('navigation')}
                className={`border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'navigation'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400'
                }`}
                aria-selected={activeTab === 'navigation'}
                role="tab"
              >
                Navigation
              </button>
              <button
                onClick={() => setActiveTab('audio')}
                className={`border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'audio'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400'
                }`}
                aria-selected={activeTab === 'audio'}
                role="tab"
              >
                Audio & Voice
              </button>
            </nav>
          </div>

          {/* Content */}
          <div className="p-6">
            {activeTab === 'visual' && (
              <div className="space-y-8" role="tabpanel" aria-labelledby="visual-tab">
                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Contrast
                  </h3>
                  <HighContrastMode />
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Font Size
                  </h3>
                  <FontSizeControl />
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Color Blindness
                  </h3>
                  <ColorBlindMode />
                </section>
              </div>
            )}

            {activeTab === 'motion' && (
              <div className="space-y-8" role="tabpanel" aria-labelledby="motion-tab">
                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Animation Preferences
                  </h3>
                  <ReducedMotion />
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Additional Options
                  </h3>
                  <div className="space-y-4">
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Smooth scrolling
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Parallax effects
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Auto-play videos
                      </span>
                      <input type="checkbox" className="rounded" />
                    </label>
                  </div>
                </section>
              </div>
            )}

            {activeTab === 'navigation' && (
              <div className="space-y-8" role="tabpanel" aria-labelledby="navigation-tab">
                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Keyboard Navigation
                  </h3>
                  <div className="space-y-4">
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Enhanced focus indicators
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Skip navigation links
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Keyboard shortcuts
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                  </div>
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Keyboard Shortcuts
                  </h3>
                  <div className="space-y-2 rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-700 dark:text-gray-300">Increase font size</span>
                      <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">
                        Ctrl +
                      </kbd>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-700 dark:text-gray-300">Decrease font size</span>
                      <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">
                        Ctrl -
                      </kbd>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-700 dark:text-gray-300">Reset font size</span>
                      <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">
                        Ctrl 0
                      </kbd>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-700 dark:text-gray-300">Open accessibility</span>
                      <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">
                        Ctrl Alt A
                      </kbd>
                    </div>
                  </div>
                </section>
              </div>
            )}

            {activeTab === 'audio' && (
              <div className="space-y-8" role="tabpanel" aria-labelledby="audio-tab">
                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Voice Control
                  </h3>
                  <div className="space-y-4">
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Enable voice commands
                      </span>
                      <input type="checkbox" className="rounded" />
                    </label>
                    <p className="text-sm text-gray-500">
                      Use voice commands to navigate and interact with the application.
                      Requires microphone access.
                    </p>
                  </div>
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Captions & Audio Description
                  </h3>
                  <div className="space-y-4">
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Show video captions
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Audio descriptions
                      </span>
                      <input type="checkbox" className="rounded" />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Sound effects
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                  </div>
                </section>

                <hr className="border-gray-200 dark:border-gray-700" />

                <section>
                  <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Screen Reader
                  </h3>
                  <div className="space-y-4">
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Enhanced ARIA labels
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                    <label className="flex items-center justify-between">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Live region announcements
                      </span>
                      <input type="checkbox" className="rounded" defaultChecked />
                    </label>
                  </div>
                </section>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="sticky bottom-0 flex items-center justify-between border-t border-gray-200 bg-white px-6 py-4 dark:border-gray-700 dark:bg-gray-900">
            <button
              onClick={() => {
                // Reset all settings
                localStorage.clear()
                window.location.reload()
              }}
              className="rounded-lg border-2 border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
            >
              Reset All
            </button>
            <button
              onClick={onClose}
              className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

// Floating Accessibility Button
export function AccessibilityButton() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="fixed bottom-4 right-4 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-blue-600 text-white shadow-lg transition-all hover:bg-blue-700 hover:shadow-xl focus:outline-none focus:ring-4 focus:ring-blue-300"
        aria-label="Open accessibility settings"
        title="Accessibility Settings (Ctrl+Alt+A)"
      >
        <Accessibility className="h-6 w-6" />
      </button>

      <AccessibilitySettings isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  )
}
