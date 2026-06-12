'use client'

import React, { useState, useEffect, useRef, useCallback } from 'react'
import { X, Lightbulb, Info } from 'lucide-react'
import { tooltipManager, TooltipConfig } from '@/lib/onboarding-manager'

interface FeatureTooltipProps {
  config: TooltipConfig
  children?: React.ReactNode
}

export function FeatureTooltip({ config, children }: FeatureTooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const [isDismissed, setIsDismissed] = useState(false)
  const tooltipRef = useRef<HTMLDivElement>(null)
  const targetRef = useRef<HTMLElement | null>(null)
  const hoverTimeoutRef = useRef<NodeJS.Timeout>()

  useEffect(() => {
    // Register tooltip
    tooltipManager.register(config)

    // Check if dismissed
    setIsDismissed(tooltipManager.isDismissed(config.id))

    // Find target element
    if (config.target) {
      const target = document.querySelector(config.target) as HTMLElement
      if (target) {
        targetRef.current = target
        setupListeners(target)
      }
    }

    return () => {
      tooltipManager.unregister(config.id)
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current)
      }
    }
  }, [config])

  const setupListeners = useCallback((target: HTMLElement) => {
    if (!target) return

    const trigger = config.trigger || 'hover'

    if (trigger === 'hover') {
      target.addEventListener('mouseenter', handleMouseEnter)
      target.addEventListener('mouseleave', handleMouseLeave)
    } else if (trigger === 'click') {
      target.addEventListener('click', handleClick)
    } else if (trigger === 'focus') {
      target.addEventListener('focus', handleFocus)
      target.addEventListener('blur', handleBlur)
    }

    return () => {
      target.removeEventListener('mouseenter', handleMouseEnter)
      target.removeEventListener('mouseleave', handleMouseLeave)
      target.removeEventListener('click', handleClick)
      target.removeEventListener('focus', handleFocus)
      target.removeEventListener('blur', handleBlur)
    }
  }, [config.trigger])

  const handleMouseEnter = useCallback(() => {
    if (isDismissed) return

    const delay = config.delay || 500
    hoverTimeoutRef.current = setTimeout(() => {
      show()
    }, delay)
  }, [config.delay, isDismissed])

  const handleMouseLeave = useCallback(() => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
    }
    if (!config.persistent) {
      hide()
    }
  }, [config.persistent])

  const handleClick = useCallback(() => {
    if (isDismissed) return
    setIsVisible(prev => !prev)
  }, [isDismissed])

  const handleFocus = useCallback(() => {
    if (isDismissed) return
    show()
  }, [isDismissed])

  const handleBlur = useCallback(() => {
    if (!config.persistent) {
      hide()
    }
  }, [config.persistent])

  const show = useCallback(() => {
    if (isDismissed) return

    const success = tooltipManager.show(config.id)
    if (success) {
      setIsVisible(true)
      updatePosition()
    }
  }, [config.id, isDismissed])

  const hide = useCallback(() => {
    setIsVisible(false)
    tooltipManager.hide(config.id)
  }, [config.id])

  const handleDismiss = useCallback(() => {
    tooltipManager.dismiss(config.id)
    setIsDismissed(true)
    setIsVisible(false)
  }, [config.id])

  const updatePosition = useCallback(() => {
    if (!targetRef.current || !tooltipRef.current) return

    const targetRect = targetRef.current.getBoundingClientRect()
    const tooltipRect = tooltipRef.current.getBoundingClientRect()
    const placement = config.placement || 'top'
    const offset = 12

    let top = 0
    let left = 0

    switch (placement) {
      case 'top':
        top = targetRect.top - tooltipRect.height - offset
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2
        break
      case 'bottom':
        top = targetRect.bottom + offset
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2
        break
      case 'left':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2
        left = targetRect.left - tooltipRect.width - offset
        break
      case 'right':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2
        left = targetRect.right + offset
        break
    }

    // Keep in viewport
    const padding = 8
    top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - padding))
    left = Math.max(padding, Math.min(left, window.innerWidth - tooltipRect.width - padding))

    setPosition({ top, left })
  }, [config.placement])

  useEffect(() => {
    if (isVisible) {
      updatePosition()
      window.addEventListener('resize', updatePosition)
      window.addEventListener('scroll', updatePosition)

      return () => {
        window.removeEventListener('resize', updatePosition)
        window.removeEventListener('scroll', updatePosition)
      }
    }
  }, [isVisible, updatePosition])

  if (isDismissed || !isVisible) return null

  return (
    <div
      ref={tooltipRef}
      className="feature-tooltip fixed z-50 w-full max-w-sm animate-fade-in rounded-lg border-2 border-blue-200 bg-white p-4 shadow-xl dark:border-blue-800 dark:bg-gray-800"
      style={{
        top: `${position.top}px`,
        left: `${position.left}px`
      }}
      role="tooltip"
      aria-labelledby={config.title ? `tooltip-title-${config.id}` : undefined}
    >
      {/* Header */}
      {config.title && (
        <div className="mb-2 flex items-start justify-between">
          <div className="flex items-center gap-2">
            <Lightbulb className="h-4 w-4 text-blue-600 dark:text-blue-400" />
            <h4
              id={`tooltip-title-${config.id}`}
              className="font-semibold text-gray-900 dark:text-gray-100"
            >
              {config.title}
            </h4>
          </div>
          {config.dismissible !== false && (
            <button
              onClick={handleDismiss}
              className="rounded p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
              aria-label="Dismiss tooltip"
            >
              <X className="h-4 w-4 text-gray-500" />
            </button>
          )}
        </div>
      )}

      {/* Content */}
      <p className="text-sm text-gray-700 dark:text-gray-300">
        {config.content}
      </p>

      {/* Dismiss button if no close icon */}
      {!config.title && config.dismissible !== false && (
        <button
          onClick={handleDismiss}
          className="mt-3 text-xs font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
        >
          Got it, don't show again
        </button>
      )}

      {/* Arrow */}
      <div
        className={`tooltip-arrow ${config.placement || 'top'}`}
        style={{
          position: 'absolute',
          width: '0',
          height: '0',
          borderStyle: 'solid'
        }}
      />

      <style jsx>{`
        .tooltip-arrow.top {
          bottom: -8px;
          left: 50%;
          transform: translateX(-50%);
          border-width: 8px 8px 0 8px;
          border-color: #fff transparent transparent transparent;
        }

        .tooltip-arrow.bottom {
          top: -8px;
          left: 50%;
          transform: translateX(-50%);
          border-width: 0 8px 8px 8px;
          border-color: transparent transparent #fff transparent;
        }

        .tooltip-arrow.left {
          right: -8px;
          top: 50%;
          transform: translateY(-50%);
          border-width: 8px 0 8px 8px;
          border-color: transparent transparent transparent #fff;
        }

        .tooltip-arrow.right {
          left: -8px;
          top: 50%;
          transform: translateY(-50%);
          border-width: 8px 8px 8px 0;
          border-color: transparent #fff transparent transparent;
        }

        @keyframes fade-in {
          from {
            opacity: 0;
            transform: scale(0.95);
          }
          to {
            opacity: 1;
            transform: scale(1);
          }
        }

        .animate-fade-in {
          animation: fade-in 0.2s ease-out;
        }
      `}</style>
    </div>
  )
}

// Hook for programmatic control
export function useFeatureTooltip(tooltipId: string) {
  const [isVisible, setIsVisible] = useState(false)
  const [isDismissed, setIsDismissed] = useState(false)

  useEffect(() => {
    setIsDismissed(tooltipManager.isDismissed(tooltipId))
  }, [tooltipId])

  const show = useCallback(() => {
    if (!isDismissed) {
      const success = tooltipManager.show(tooltipId)
      setIsVisible(success)
    }
  }, [tooltipId, isDismissed])

  const hide = useCallback(() => {
    tooltipManager.hide(tooltipId)
    setIsVisible(false)
  }, [tooltipId])

  const dismiss = useCallback(() => {
    tooltipManager.dismiss(tooltipId)
    setIsDismissed(true)
    setIsVisible(false)
  }, [tooltipId])

  return {
    isVisible,
    isDismissed,
    show,
    hide,
    dismiss
  }
}

// Tooltip Manager Component
export function TooltipManagerPanel() {
  const [tooltips, setTooltips] = useState<TooltipConfig[]>([])
  const [groupedTooltips, setGroupedTooltips] = useState<Record<string, TooltipConfig[]>>({})

  useEffect(() => {
    // Get all tooltips - this would need to be exposed by the manager
    // For now, show categories
    setGroupedTooltips({
      'Getting Started': [],
      'Features': [],
      'Advanced': []
    })
  }, [])

  const handleResetAll = () => {
    tooltipManager.resetAll()
    window.location.reload()
  }

  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Feature Tooltips
        </h3>
        <button
          onClick={handleResetAll}
          className="rounded-lg border-2 border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700"
        >
          Reset All
        </button>
      </div>

      <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
        Tooltips help you discover new features as you use the application. You can reset them to see them again.
      </p>

      <div className="space-y-4">
        {Object.entries(groupedTooltips).map(([category, items]) => (
          <div key={category}>
            <h4 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
              {category}
            </h4>
            <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              {items.length === 0 ? (
                <p className="text-sm text-gray-500">No tooltips in this category</p>
              ) : (
                <div className="space-y-2">
                  {items.map(tooltip => (
                    <div
                      key={tooltip.id}
                      className="flex items-center justify-between text-sm"
                    >
                      <span className="text-gray-700 dark:text-gray-300">
                        {tooltip.title || tooltip.id}
                      </span>
                      <span className="text-xs text-gray-500">
                        {tooltipManager.isDismissed(tooltip.id) ? 'Dismissed' : 'Active'}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

// Example tooltips
export const exampleTooltips: TooltipConfig[] = [
  {
    id: 'welcome-dashboard',
    target: '[data-tour="dashboard"]',
    title: 'Welcome to your Dashboard!',
    content: 'This is your central hub for managing backups, monitoring status, and accessing key features.',
    trigger: 'manual',
    placement: 'bottom',
    showOnce: true,
    dismissible: true,
    category: 'Getting Started'
  },
  {
    id: 'quick-backup',
    target: '[data-tour="quick-backup"]',
    title: 'Quick Backup',
    content: 'Click here to quickly create a backup with default settings.',
    trigger: 'hover',
    placement: 'right',
    delay: 1000,
    showOnce: true,
    category: 'Features'
  },
  {
    id: 'advanced-settings',
    target: '[data-tour="advanced-settings"]',
    title: 'Advanced Options',
    content: 'Customize your backup with advanced settings like compression, encryption, and more.',
    trigger: 'hover',
    placement: 'left',
    delay: 1000,
    category: 'Advanced'
  }
]
