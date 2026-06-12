'use client'

import React, { useState, useRef, useEffect } from 'react'
import { HelpCircle, X, ExternalLink, BookOpen } from 'lucide-react'

interface ContextualHelpProps {
  topic: string
  title?: string
  content: string | React.ReactNode
  learnMoreUrl?: string
  placement?: 'top' | 'bottom' | 'left' | 'right'
  size?: 'sm' | 'md' | 'lg'
  inline?: boolean
}

export function ContextualHelp({
  topic,
  title,
  content,
  learnMoreUrl,
  placement = 'top',
  size = 'md',
  inline = false
}: ContextualHelpProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const triggerRef = useRef<HTMLButtonElement>(null)
  const popoverRef = useRef<HTMLDivElement>(null)

  const iconSizes = {
    sm: 'h-3.5 w-3.5',
    md: 'h-4 w-4',
    lg: 'h-5 w-5'
  }

  useEffect(() => {
    if (isOpen) {
      updatePosition()
      window.addEventListener('resize', updatePosition)
      window.addEventListener('scroll', updatePosition, true)

      // Close on outside click
      const handleClickOutside = (e: MouseEvent) => {
        if (
          popoverRef.current &&
          !popoverRef.current.contains(e.target as Node) &&
          !triggerRef.current?.contains(e.target as Node)
        ) {
          setIsOpen(false)
        }
      }
      document.addEventListener('mousedown', handleClickOutside)

      return () => {
        window.removeEventListener('resize', updatePosition)
        window.removeEventListener('scroll', updatePosition, true)
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }
  }, [isOpen])

  const updatePosition = () => {
    if (!triggerRef.current || !popoverRef.current) return

    const triggerRect = triggerRef.current.getBoundingClientRect()
    const popoverRect = popoverRef.current.getBoundingClientRect()
    const offset = 12

    let top = 0
    let left = 0

    switch (placement) {
      case 'top':
        top = triggerRect.top - popoverRect.height - offset
        left = triggerRect.left + (triggerRect.width - popoverRect.width) / 2
        break
      case 'bottom':
        top = triggerRect.bottom + offset
        left = triggerRect.left + (triggerRect.width - popoverRect.width) / 2
        break
      case 'left':
        top = triggerRect.top + (triggerRect.height - popoverRect.height) / 2
        left = triggerRect.left - popoverRect.width - offset
        break
      case 'right':
        top = triggerRect.top + (triggerRect.height - popoverRect.height) / 2
        left = triggerRect.right + offset
        break
    }

    // Keep in viewport
    const padding = 12
    top = Math.max(padding, Math.min(top, window.innerHeight - popoverRect.height - padding))
    left = Math.max(padding, Math.min(left, window.innerWidth - popoverRect.width - padding))

    setPosition({ top, left })
  }

  return (
    <>
      <button
        ref={triggerRef}
        onClick={() => setIsOpen(!isOpen)}
        className={`${
          inline ? 'inline-flex' : 'flex'
        } items-center justify-center rounded-full text-blue-600 transition-all hover:bg-blue-50 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 dark:text-blue-400 dark:hover:bg-blue-900/20 dark:hover:text-blue-300 ${
          size === 'sm' ? 'p-0.5' : size === 'lg' ? 'p-1.5' : 'p-1'
        }`}
        aria-label={`Help about ${topic}`}
        aria-expanded={isOpen}
        aria-haspopup="dialog"
      >
        <HelpCircle className={iconSizes[size]} />
      </button>

      {isOpen && (
        <div
          ref={popoverRef}
          className="fixed z-50 w-full max-w-sm animate-fade-in rounded-lg border-2 border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
          style={{
            top: `${position.top}px`,
            left: `${position.left}px`
          }}
          role="dialog"
          aria-labelledby={`help-title-${topic}`}
          aria-modal="true"
        >
          {/* Header */}
          <div className="flex items-start justify-between border-b border-gray-200 p-4 dark:border-gray-700">
            <div className="flex items-center gap-2">
              <BookOpen className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              <h3
                id={`help-title-${topic}`}
                className="font-semibold text-gray-900 dark:text-gray-100"
              >
                {title || 'Help'}
              </h3>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
              aria-label="Close help"
            >
              <X className="h-4 w-4 text-gray-500" />
            </button>
          </div>

          {/* Content */}
          <div className="p-4">
            {typeof content === 'string' ? (
              <p className="text-sm leading-relaxed text-gray-700 dark:text-gray-300">
                {content}
              </p>
            ) : (
              content
            )}
          </div>

          {/* Footer */}
          {learnMoreUrl && (
            <div className="border-t border-gray-200 p-4 dark:border-gray-700">
              <a
                href={learnMoreUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-sm font-medium text-blue-600 transition-colors hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
              >
                <ExternalLink className="h-4 w-4" />
                Learn more
              </a>
            </div>
          )}

          {/* Arrow */}
          <div className={`help-arrow ${placement}`} />

          <style jsx>{`
            .help-arrow {
              position: absolute;
              width: 0;
              height: 0;
              border-style: solid;
            }

            .help-arrow.top {
              bottom: -10px;
              left: 50%;
              transform: translateX(-50%);
              border-width: 10px 10px 0 10px;
              border-color: #e5e7eb transparent transparent transparent;
            }

            .help-arrow.bottom {
              top: -10px;
              left: 50%;
              transform: translateX(-50%);
              border-width: 0 10px 10px 10px;
              border-color: transparent transparent #e5e7eb transparent;
            }

            .help-arrow.left {
              right: -10px;
              top: 50%;
              transform: translateY(-50%);
              border-width: 10px 0 10px 10px;
              border-color: transparent transparent transparent #e5e7eb;
            }

            .help-arrow.right {
              left: -10px;
              top: 50%;
              transform: translateY(-50%);
              border-width: 10px 10px 10px 0;
              border-color: transparent #e5e7eb transparent transparent;
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
              animation: fade-in 0.15s ease-out;
            }
          `}</style>
        </div>
      )}
    </>
  )
}

// Help Card Component - for standalone help sections
export function HelpCard({
  title,
  children,
  icon,
  variant = 'info'
}: {
  title: string
  children: React.ReactNode
  icon?: React.ReactNode
  variant?: 'info' | 'tip' | 'warning' | 'success'
}) {
  const variants = {
    info: {
      container: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20',
      icon: 'text-blue-600 dark:text-blue-400',
      title: 'text-blue-900 dark:text-blue-100'
    },
    tip: {
      container: 'border-purple-200 bg-purple-50 dark:border-purple-800 dark:bg-purple-900/20',
      icon: 'text-purple-600 dark:text-purple-400',
      title: 'text-purple-900 dark:text-purple-100'
    },
    warning: {
      container: 'border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900/20',
      icon: 'text-yellow-600 dark:text-yellow-400',
      title: 'text-yellow-900 dark:text-yellow-100'
    },
    success: {
      container: 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20',
      icon: 'text-green-600 dark:text-green-400',
      title: 'text-green-900 dark:text-green-100'
    }
  }

  const style = variants[variant]

  return (
    <div className={`rounded-lg border-2 p-4 ${style.container}`}>
      <div className="flex items-start gap-3">
        {icon && <div className={`flex-shrink-0 ${style.icon}`}>{icon}</div>}
        <div className="flex-1">
          <h4 className={`mb-2 font-semibold ${style.title}`}>{title}</h4>
          <div className="text-sm text-gray-700 dark:text-gray-300">{children}</div>
        </div>
      </div>
    </div>
  )
}

// Help Section Component - for grouping multiple help items
export function HelpSection({
  title,
  children,
  collapsible = false,
  defaultOpen = true
}: {
  title: string
  children: React.ReactNode
  collapsible?: boolean
  defaultOpen?: boolean
}) {
  const [isOpen, setIsOpen] = useState(defaultOpen)

  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      <button
        onClick={() => collapsible && setIsOpen(!isOpen)}
        className={`flex w-full items-center justify-between p-4 text-left ${
          collapsible ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-750' : 'cursor-default'
        }`}
        disabled={!collapsible}
      >
        <h3 className="font-semibold text-gray-900 dark:text-gray-100">{title}</h3>
        {collapsible && (
          <svg
            className={`h-5 w-5 transform text-gray-500 transition-transform ${
              isOpen ? 'rotate-180' : ''
            }`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        )}
      </button>
      {isOpen && <div className="border-t border-gray-200 p-4 dark:border-gray-700">{children}</div>}
    </div>
  )
}

// Inline Help Text - for subtle inline hints
export function InlineHelp({ children }: { children: React.ReactNode }) {
  return (
    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {children}
    </p>
  )
}

// Example usage component
export function HelpExamples() {
  return (
    <div className="space-y-6 p-6">
      <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
        Contextual Help Examples
      </h2>

      {/* Inline help icon */}
      <div>
        <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
          Backup Name
          <ContextualHelp
            topic="backup-name"
            title="Backup Name"
            content="Give your backup a descriptive name to help you identify it later. Good names include dates, database names, or project identifiers."
            size="sm"
            inline
          />
        </label>
        <input
          type="text"
          className="mt-1 w-full rounded-lg border-2 border-gray-200 px-4 py-2 dark:border-gray-700 dark:bg-gray-800"
          placeholder="e.g., prod-db-2024-01"
        />
      </div>

      {/* Help cards */}
      <div className="space-y-4">
        <HelpCard title="Getting Started" icon={<BookOpen className="h-5 w-5" />} variant="info">
          <p className="mb-2">Welcome to the backup system! Here's how to get started:</p>
          <ol className="list-inside list-decimal space-y-1">
            <li>Create your first backup</li>
            <li>Schedule automatic backups</li>
            <li>Monitor backup status</li>
          </ol>
        </HelpCard>

        <HelpCard
          title="Pro Tip"
          icon={<HelpCircle className="h-5 w-5" />}
          variant="tip"
        >
          Enable compression to reduce storage space and speed up transfers. Most databases compress well!
        </HelpCard>
      </div>

      {/* Help sections */}
      <div className="space-y-4">
        <HelpSection title="Frequently Asked Questions" collapsible defaultOpen={false}>
          <div className="space-y-4">
            <div>
              <h4 className="font-medium text-gray-900 dark:text-gray-100">
                How often should I backup?
              </h4>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                It depends on how often your data changes. For production databases, daily or even hourly backups are recommended.
              </p>
            </div>
            <div>
              <h4 className="font-medium text-gray-900 dark:text-gray-100">
                Where are backups stored?
              </h4>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                Backups can be stored locally, in cloud storage (S3, GCS, Azure), or both for extra redundancy.
              </p>
            </div>
          </div>
        </HelpSection>
      </div>
    </div>
  )
}
