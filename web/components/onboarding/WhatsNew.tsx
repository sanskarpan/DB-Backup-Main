'use client'

import React, { useState } from 'react'
import { Sparkles, X, ExternalLink, Play, Check } from 'lucide-react'
import { WhatsNewItem } from '@/lib/onboarding-manager'

interface WhatsNewProps {
  items?: WhatsNewItem[]
  currentVersion?: string
}

export function WhatsNew({ items = defaultWhatsNewItems, currentVersion = '2.1.0' }: WhatsNewProps) {
  const [selectedItem, setSelectedItem] = useState<WhatsNewItem | null>(null)

  const typeColors = {
    feature: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    improvement: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    fix: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    breaking: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  }

  const typeIcons = {
    feature: '✨',
    improvement: '⚡',
    fix: '🔧',
    breaking: '⚠️'
  }

  // Group by version
  const groupedItems = items.reduce((acc, item) => {
    if (!acc[item.version]) {
      acc[item.version] = []
    }
    acc[item.version].push(item)
    return acc
  }, {} as Record<string, WhatsNewItem[]>)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
          <Sparkles className="h-6 w-6 text-blue-600" />
          What's New
        </h2>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Discover the latest features, improvements, and fixes
        </p>
      </div>

      {/* Current version banner */}
      <div className="rounded-lg border-2 border-blue-200 bg-gradient-to-r from-blue-50 to-purple-50 p-6 dark:border-blue-800 dark:from-blue-900/20 dark:to-purple-900/20">
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-600 text-white">
            <Sparkles className="h-6 w-6" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              You're on version {currentVersion}!
            </h3>
            <p className="mt-1 text-sm text-gray-700 dark:text-gray-300">
              Check out the latest features and improvements below
            </p>
          </div>
        </div>
      </div>

      {/* Changelog by version */}
      <div className="space-y-8">
        {Object.entries(groupedItems)
          .sort(([a], [b]) => b.localeCompare(a)) // Sort versions descending
          .map(([version, versionItems]) => (
            <div key={version} className="space-y-4">
              {/* Version header */}
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600 text-sm font-bold text-white">
                  v{version}
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                    Version {version}
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Released {versionItems[0].releaseDate.toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'long',
                      day: 'numeric'
                    })}
                  </p>
                </div>
                {version === currentVersion && (
                  <span className="rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200">
                    Current
                  </span>
                )}
              </div>

              {/* Items */}
              <div className="space-y-3 pl-13">
                {versionItems.map(item => (
                  <WhatsNewCard
                    key={item.id}
                    item={item}
                    typeColors={typeColors}
                    typeIcons={typeIcons}
                    onClick={() => setSelectedItem(item)}
                  />
                ))}
              </div>
            </div>
          ))}
      </div>

      {/* Detail Modal */}
      {selectedItem && (
        <WhatsNewModal
          item={selectedItem}
          onClose={() => setSelectedItem(null)}
          typeColors={typeColors}
        />
      )}
    </div>
  )
}

// What's New Card
function WhatsNewCard({
  item,
  typeColors,
  typeIcons,
  onClick
}: {
  item: WhatsNewItem
  typeColors: Record<string, string>
  typeIcons: Record<string, string>
  onClick: () => void
}) {
  return (
    <div
      className={`group cursor-pointer overflow-hidden rounded-lg border-2 transition-all hover:shadow-md ${
        item.highlight
          ? 'border-blue-300 bg-blue-50/50 dark:border-blue-700 dark:bg-blue-900/10'
          : 'border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800'
      }`}
      onClick={onClick}
    >
      <div className="p-4">
        <div className="flex items-start gap-3">
          <span className="text-2xl">{typeIcons[item.type]}</span>
          <div className="flex-1">
            <div className="mb-2 flex items-start justify-between">
              <div className="flex-1">
                <h4 className="font-semibold text-gray-900 dark:text-gray-100">
                  {item.title}
                </h4>
                <p className="mt-1 text-sm text-gray-700 dark:text-gray-300">
                  {item.description}
                </p>
              </div>
              <span className={`ml-2 flex-shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${typeColors[item.type]}`}>
                {item.type}
              </span>
            </div>

            {(item.imageUrl || item.videoUrl || item.learnMoreUrl) && (
              <div className="mt-3 flex items-center gap-2 text-xs text-blue-600 dark:text-blue-400">
                {item.videoUrl && (
                  <span className="flex items-center gap-1">
                    <Play className="h-3 w-3" />
                    Watch demo
                  </span>
                )}
                {item.learnMoreUrl && (
                  <span className="flex items-center gap-1">
                    <ExternalLink className="h-3 w-3" />
                    Learn more
                  </span>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// What's New Modal
function WhatsNewModal({
  item,
  onClose,
  typeColors
}: {
  item: WhatsNewItem
  onClose: () => void
  typeColors: Record<string, string>
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="relative w-full max-w-2xl overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-gray-800"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute right-4 top-4 z-10 flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 transition-colors hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600"
          aria-label="Close"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Media */}
        {(item.imageUrl || item.videoUrl) && (
          <div className="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-900">
            {item.videoUrl ? (
              <video src={item.videoUrl} controls className="h-full w-full" />
            ) : item.imageUrl ? (
              <img src={item.imageUrl} alt={item.title} className="h-full w-full object-cover" />
            ) : null}
          </div>
        )}

        {/* Content */}
        <div className="p-6">
          <div className="mb-4 flex items-start gap-3">
            <span className={`flex-shrink-0 rounded-full px-3 py-1 text-xs font-medium ${typeColors[item.type]}`}>
              {item.type}
            </span>
            <span className="text-xs text-gray-600 dark:text-gray-400">
              Version {item.version} • {item.releaseDate.toLocaleDateString()}
            </span>
          </div>

          <h2 className="mb-3 text-2xl font-bold text-gray-900 dark:text-gray-100">
            {item.title}
          </h2>

          <p className="mb-6 text-gray-700 dark:text-gray-300">
            {item.description}
          </p>

          {item.learnMoreUrl && (
            <a
              href={item.learnMoreUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              <ExternalLink className="h-4 w-4" />
              Learn More
            </a>
          )}
        </div>
      </div>
    </div>
  )
}

// Default What's New items
const defaultWhatsNewItems: WhatsNewItem[] = [
  {
    id: '1',
    version: '2.1.0',
    title: 'Intelligent Onboarding System',
    description: 'New comprehensive onboarding with interactive tours, tooltips, and walkthroughs to help you get started quickly.',
    type: 'feature',
    releaseDate: new Date('2024-01-15'),
    highlight: true
  },
  {
    id: '2',
    version: '2.1.0',
    title: 'Help Center & Chatbot',
    description: 'Access instant help with our new intelligent help search and AI chatbot for common questions.',
    type: 'feature',
    releaseDate: new Date('2024-01-15'),
    highlight: true
  },
  {
    id: '3',
    version: '2.1.0',
    title: 'Video Tutorials',
    description: 'Learn at your own pace with our library of video tutorials covering all features.',
    type: 'feature',
    releaseDate: new Date('2024-01-15')
  },
  {
    id: '4',
    version: '2.0.5',
    title: 'Improved Backup Performance',
    description: 'Backups are now up to 40% faster with our optimized compression algorithm.',
    type: 'improvement',
    releaseDate: new Date('2024-01-10')
  },
  {
    id: '5',
    version: '2.0.5',
    title: 'Fixed Restore Issues',
    description: 'Resolved issues with point-in-time recovery for large databases.',
    type: 'fix',
    releaseDate: new Date('2024-01-10')
  },
  {
    id: '6',
    version: '2.0.0',
    title: 'New API Version',
    description: 'Major API update with breaking changes. Please review the migration guide.',
    type: 'breaking',
    releaseDate: new Date('2024-01-01'),
    highlight: true,
    learnMoreUrl: '/docs/migration-guide'
  }
]

// Compact What's New Component (for sidebar/dropdown)
export function WhatsNewCompact({ maxItems = 3 }: { maxItems?: number }) {
  const [isExpanded, setIsExpanded] = useState(false)
  const items = defaultWhatsNewItems.slice(0, maxItems)

  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="flex items-center gap-2 font-semibold text-gray-900 dark:text-gray-100">
          <Sparkles className="h-4 w-4 text-blue-600" />
          What's New
        </h3>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="text-xs font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400"
        >
          {isExpanded ? 'Show less' : 'View all'}
        </button>
      </div>

      <div className="space-y-2">
        {items.map(item => (
          <div key={item.id} className="flex items-start gap-2 text-sm">
            <Check className="mt-0.5 h-4 w-4 flex-shrink-0 text-green-600" />
            <div className="flex-1">
              <p className="font-medium text-gray-900 dark:text-gray-100">
                {item.title}
              </p>
              {isExpanded && (
                <p className="mt-0.5 text-xs text-gray-600 dark:text-gray-400">
                  {item.description}
                </p>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
