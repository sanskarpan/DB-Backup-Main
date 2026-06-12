'use client'

import React, { useState } from 'react'
import { BookOpen, Video, Lightbulb, CheckSquare, Search as SearchIcon, MessageSquare, Sparkles, HelpCircle } from 'lucide-react'
import { HelpSearch } from './HelpSearch'
import { VideoTutorials } from './VideoTutorials'
import { TipsAndTricks } from './TipsAndTricks'
import { WhatsNew } from './WhatsNew'
import { OnboardingChecklist, exampleChecklist } from './OnboardingChecklist'
import { TourLauncher } from './ProductTour'
import { WalkthroughLibrary, exampleWalkthroughs } from './InteractiveWalkthrough'
import { HelpChatbot } from './HelpChatbot'
import { FeedbackButton } from './FeedbackSystem'
import { useOnboarding, useOnboardingShortcuts } from './OnboardingContext'

type HelpCenterTab = 'search' | 'videos' | 'tips' | 'whats-new' | 'checklist' | 'tours' | 'walkthroughs'

export function HelpCenter() {
  const [activeTab, setActiveTab] = useState<HelpCenterTab>('search')
  const { chatbotOpen, setChatbotOpen } = useOnboarding()

  // Register keyboard shortcuts
  useOnboardingShortcuts()

  const tabs = [
    {
      id: 'search' as const,
      label: 'Search',
      icon: SearchIcon,
      description: 'Find answers quickly'
    },
    {
      id: 'videos' as const,
      label: 'Videos',
      icon: Video,
      description: 'Watch tutorials'
    },
    {
      id: 'tips' as const,
      label: 'Tips & Tricks',
      icon: Lightbulb,
      description: 'Learn best practices'
    },
    {
      id: 'whats-new' as const,
      label: "What's New",
      icon: Sparkles,
      description: 'Latest features'
    },
    {
      id: 'checklist' as const,
      label: 'Setup',
      icon: CheckSquare,
      description: 'Get started'
    },
    {
      id: 'tours' as const,
      label: 'Tours',
      icon: HelpCircle,
      description: 'Interactive guides'
    },
    {
      id: 'walkthroughs' as const,
      label: 'Walkthroughs',
      icon: BookOpen,
      description: 'Step-by-step'
    }
  ]

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-800">
        <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
                Help Center
              </h1>
              <p className="mt-2 text-gray-600 dark:text-gray-400">
                Everything you need to master the backup system
              </p>
            </div>

            {/* Quick actions */}
            <div className="flex gap-2">
              <button
                onClick={() => setChatbotOpen(!chatbotOpen)}
                className="flex items-center gap-2 rounded-lg border-2 border-gray-200 px-4 py-2 font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                <MessageSquare className="h-5 w-5" />
                Ask Assistant
              </button>
            </div>
          </div>

          {/* Tabs */}
          <div className="mt-6 flex gap-2 overflow-x-auto">
            {tabs.map(tab => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex flex-shrink-0 items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all ${
                    isActive
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  <div className="text-left">
                    <div>{tab.label}</div>
                    {!isActive && (
                      <div className="text-xs opacity-70">{tab.description}</div>
                    )}
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {activeTab === 'search' && <HelpSearch />}
        {activeTab === 'videos' && <VideoTutorials />}
        {activeTab === 'tips' && <TipsAndTricks />}
        {activeTab === 'whats-new' && <WhatsNew />}
        {activeTab === 'checklist' && <OnboardingChecklist checklist={exampleChecklist} />}
        {activeTab === 'tours' && <TourLauncher tours={[]} />}
        {activeTab === 'walkthroughs' && <WalkthroughLibrary walkthroughs={exampleWalkthroughs} />}
      </div>

      {/* Chatbot */}
      <HelpChatbot
        isOpen={chatbotOpen}
        onClose={() => setChatbotOpen(false)}
      />

      {/* Feedback Button */}
      <FeedbackButton />

      {/* Keyboard shortcuts hint */}
      <div className="fixed bottom-4 left-1/2 z-30 -translate-x-1/2 rounded-lg border-2 border-gray-200 bg-white px-4 py-2 text-xs text-gray-600 shadow-lg dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">
        <div className="flex items-center gap-4">
          <span>Keyboard shortcuts:</span>
          <div className="flex gap-2">
            <kbd className="rounded bg-gray-100 px-2 py-0.5 dark:bg-gray-700">Ctrl+Shift+H</kbd>
            <span>Help</span>
          </div>
          <div className="flex gap-2">
            <kbd className="rounded bg-gray-100 px-2 py-0.5 dark:bg-gray-700">Ctrl+Shift+C</kbd>
            <span>Chat</span>
          </div>
          <div className="flex gap-2">
            <kbd className="rounded bg-gray-100 px-2 py-0.5 dark:bg-gray-700">Ctrl+Shift+F</kbd>
            <span>Feedback</span>
          </div>
        </div>
      </div>
    </div>
  )
}

// Quick Help Modal (for global access)
export function QuickHelpModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [searchQuery, setSearchQuery] = useState('')

  if (!isOpen) return null

  const quickLinks = [
    { title: 'Create a backup', url: '/help/create-backup', category: 'Getting Started' },
    { title: 'Schedule backups', url: '/help/schedule', category: 'Configuration' },
    { title: 'Restore data', url: '/help/restore', category: 'Recovery' },
    { title: 'Troubleshoot issues', url: '/help/troubleshooting', category: 'Support' }
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="w-full max-w-2xl rounded-lg bg-white shadow-2xl dark:bg-gray-800"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search */}
        <div className="p-6">
          <div className="relative">
            <SearchIcon className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search for help..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-lg border-2 border-gray-200 py-3 pl-12 pr-4 text-lg focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900"
              autoFocus
            />
          </div>
        </div>

        {/* Quick Links */}
        <div className="border-t border-gray-200 p-6 dark:border-gray-700">
          <h3 className="mb-4 font-semibold text-gray-900 dark:text-gray-100">
            Quick Links
          </h3>
          <div className="grid gap-3 sm:grid-cols-2">
            {quickLinks.map(link => (
              <a
                key={link.url}
                href={link.url}
                className="flex items-start gap-3 rounded-lg border-2 border-gray-200 p-4 transition-all hover:border-blue-500 hover:bg-blue-50 dark:border-gray-700 dark:hover:bg-blue-900/10"
              >
                <BookOpen className="h-5 w-5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
                <div>
                  <p className="font-medium text-gray-900 dark:text-gray-100">
                    {link.title}
                  </p>
                  <p className="text-xs text-gray-600 dark:text-gray-400">
                    {link.category}
                  </p>
                </div>
              </a>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 p-6 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <button
              onClick={onClose}
              className="text-sm font-medium text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
            >
              Close
            </button>
            <a
              href="/help"
              className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              Go to Help Center
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

// Export all components for easy import
export * from './ProductTour'
export * from './FeatureTooltip'
export * from './ContextualHelp'
export * from './VideoTutorials'
export * from './InteractiveWalkthrough'
export * from './TipsAndTricks'
export * from './WhatsNew'
export * from './OnboardingChecklist'
export * from './HelpSearch'
export * from './FeedbackSystem'
export * from './HelpChatbot'
export * from './OnboardingContext'
