'use client'

import React, { useState, useEffect } from 'react'
import { Check, Circle, Clock, HelpCircle, Play, ExternalLink, ChevronRight } from 'lucide-react'
import { checklistManager, OnboardingChecklist as ChecklistType, ChecklistItem } from '@/lib/onboarding-manager'

interface OnboardingChecklistProps {
  checklist: ChecklistType
  onComplete?: () => void
}

export function OnboardingChecklist({ checklist, onComplete }: OnboardingChecklistProps) {
  const [items, setItems] = useState<ChecklistItem[]>(checklist.items)
  const [progress, setProgress] = useState(checklist.progress)

  useEffect(() => {
    checklistManager.register(checklist)

    const handleItemCompleted = ({ checklist: updatedChecklist }: any) => {
      setItems([...updatedChecklist.items])
      setProgress(updatedChecklist.progress)
    }

    const handleChecklistCompleted = () => {
      onComplete?.()
    }

    checklistManager.on('item-completed', handleItemCompleted)
    checklistManager.on('item-uncompleted', handleItemCompleted)
    checklistManager.on('checklist-completed', handleChecklistCompleted)

    return () => {
      checklistManager.off('item-completed', handleItemCompleted)
      checklistManager.off('item-uncompleted', handleItemCompleted)
      checklistManager.off('checklist-completed', handleChecklistCompleted)
    }
  }, [checklist, onComplete])

  const handleToggle = (itemId: string, completed: boolean) => {
    if (completed) {
      checklistManager.uncompleteItem(checklist.id, itemId)
    } else {
      checklistManager.completeItem(checklist.id, itemId)
    }
  }

  const handleAction = (item: ChecklistItem) => {
    if (item.action) {
      item.action()
    }
  }

  const completedCount = items.filter(i => i.completed).length
  const requiredCount = items.filter(i => i.required).length
  const requiredCompleted = items.filter(i => i.required && i.completed).length

  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      {/* Header */}
      <div className="border-b border-gray-200 p-6 dark:border-gray-700">
        <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100">
          {checklist.title}
        </h2>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {checklist.description}
        </p>

        {/* Progress */}
        <div className="mt-4">
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-medium text-gray-700 dark:text-gray-300">
              Progress: {completedCount} of {items.length} completed
            </span>
            <span className="font-semibold text-blue-600 dark:text-blue-400">
              {progress}%
            </span>
          </div>

          {/* Progress bar */}
          <div className="h-3 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div
              className="h-full bg-gradient-to-r from-blue-500 to-blue-600 transition-all duration-500"
              style={{ width: `${progress}%` }}
            />
          </div>

          {/* Required items */}
          {requiredCount > 0 && (
            <p className="mt-2 text-xs text-gray-600 dark:text-gray-400">
              {requiredCompleted} of {requiredCount} required items completed
            </p>
          )}
        </div>
      </div>

      {/* Checklist Items */}
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        {items.map((item, index) => (
          <ChecklistItemCard
            key={item.id}
            item={item}
            index={index}
            onToggle={() => handleToggle(item.id, item.completed)}
            onAction={() => handleAction(item)}
          />
        ))}
      </div>

      {/* Footer */}
      {progress === 100 && (
        <div className="border-t border-gray-200 bg-green-50 p-6 dark:border-gray-700 dark:bg-green-900/20">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-green-600">
              <Check className="h-6 w-6 text-white" />
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-green-900 dark:text-green-100">
                Setup Complete!
              </h3>
              <p className="text-sm text-green-800 dark:text-green-200">
                Great job! You've completed all onboarding steps.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Checklist Item Card
function ChecklistItemCard({
  item,
  index,
  onToggle,
  onAction
}: {
  item: ChecklistItem
  index: number
  onToggle: () => void
  onAction: () => void
}) {
  return (
    <div className={`group transition-colors ${item.completed ? 'bg-gray-50 dark:bg-gray-800/50' : 'bg-white hover:bg-gray-50 dark:bg-gray-800 dark:hover:bg-gray-750'}`}>
      <div className="flex items-start gap-4 p-4">
        {/* Checkbox */}
        <button
          onClick={onToggle}
          className={`flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full border-2 transition-all ${
            item.completed
              ? 'border-green-600 bg-green-600 text-white'
              : 'border-gray-300 bg-white hover:border-green-600 dark:border-gray-600 dark:bg-gray-700'
          }`}
          aria-label={item.completed ? 'Mark as incomplete' : 'Mark as complete'}
        >
          {item.completed ? (
            <Check className="h-4 w-4" />
          ) : (
            <Circle className="h-3 w-3 opacity-0 group-hover:opacity-50" />
          )}
        </button>

        {/* Content */}
        <div className="flex-1">
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <h3 className={`font-medium ${item.completed ? 'text-gray-500 line-through dark:text-gray-500' : 'text-gray-900 dark:text-gray-100'}`}>
                {item.title}
                {item.required && (
                  <span className="ml-2 rounded bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-200">
                    Required
                  </span>
                )}
              </h3>
              <p className={`mt-1 text-sm ${item.completed ? 'text-gray-400 dark:text-gray-600' : 'text-gray-600 dark:text-gray-400'}`}>
                {item.description}
              </p>

              {/* Estimated time */}
              {item.estimatedTime && !item.completed && (
                <div className="mt-2 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-500">
                  <Clock className="h-3 w-3" />
                  ~{item.estimatedTime} min
                </div>
              )}
            </div>

            {/* Actions */}
            <div className="flex flex-shrink-0 gap-2">
              {item.helpUrl && (
                <a
                  href={item.helpUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                  aria-label="Get help"
                  title="Get help"
                >
                  <HelpCircle className="h-4 w-4" />
                </a>
              )}

              {item.action && !item.completed && (
                <button
                  onClick={onAction}
                  className="flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-blue-700"
                >
                  <Play className="h-3 w-3" />
                  Start
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Compact Checklist (for sidebar/widget)
export function ChecklistCompact({ checklistId }: { checklistId: string }) {
  const [checklist, setChecklist] = useState<ChecklistType | null>(null)

  useEffect(() => {
    const cl = checklistManager.get(checklistId)
    if (cl) setChecklist(cl)

    const handleUpdate = () => {
      const cl = checklistManager.get(checklistId)
      if (cl) setChecklist(cl)
    }

    checklistManager.on('item-completed', handleUpdate)
    checklistManager.on('item-uncompleted', handleUpdate)

    return () => {
      checklistManager.off('item-completed', handleUpdate)
      checklistManager.off('item-uncompleted', handleUpdate)
    }
  }, [checklistId])

  if (!checklist) return null

  const completedCount = checklist.items.filter(i => i.completed).length

  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="font-semibold text-gray-900 dark:text-gray-100">
          {checklist.title}
        </h3>
        <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
          {checklist.progress}%
        </span>
      </div>

      <div className="mb-3 h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
        <div
          className="h-full bg-blue-600 transition-all"
          style={{ width: `${checklist.progress}%` }}
        />
      </div>

      <p className="text-xs text-gray-600 dark:text-gray-400">
        {completedCount} of {checklist.items.length} completed
      </p>

      {checklist.progress < 100 && (
        <a
          href="#onboarding"
          className="mt-3 flex items-center justify-between rounded-lg bg-blue-50 px-3 py-2 text-sm font-medium text-blue-600 transition-colors hover:bg-blue-100 dark:bg-blue-900/20 dark:text-blue-400 dark:hover:bg-blue-900/30"
        >
          Continue setup
          <ChevronRight className="h-4 w-4" />
        </a>
      )}
    </div>
  )
}

// Example checklist
export const exampleChecklist: ChecklistType = {
  id: 'getting-started',
  title: 'Getting Started',
  description: 'Complete these steps to set up your backup system',
  progress: 0,
  items: [
    {
      id: 'connect-database',
      title: 'Connect Your First Database',
      description: 'Add a database connection to start backing up your data',
      completed: false,
      required: true,
      estimatedTime: 5,
      category: 'Setup',
      helpUrl: '/docs/connect-database'
    },
    {
      id: 'create-backup',
      title: 'Create Your First Backup',
      description: 'Run a manual backup to ensure everything is working correctly',
      completed: false,
      required: true,
      estimatedTime: 3,
      category: 'Setup'
    },
    {
      id: 'schedule-backup',
      title: 'Schedule Automatic Backups',
      description: 'Set up recurring backups so you never miss a backup window',
      completed: false,
      required: true,
      estimatedTime: 5,
      category: 'Configuration'
    },
    {
      id: 'configure-storage',
      title: 'Configure Cloud Storage',
      description: 'Set up S3, GCS, or Azure storage for off-site backups',
      completed: false,
      required: false,
      estimatedTime: 10,
      category: 'Configuration'
    },
    {
      id: 'enable-encryption',
      title: 'Enable Encryption',
      description: 'Secure your backups with AES-256 encryption',
      completed: false,
      required: false,
      estimatedTime: 3,
      category: 'Security'
    },
    {
      id: 'setup-alerts',
      title: 'Set Up Alerts',
      description: 'Configure email or Slack notifications for backup events',
      completed: false,
      required: false,
      estimatedTime: 5,
      category: 'Monitoring'
    },
    {
      id: 'test-restore',
      title: 'Test a Restore',
      description: 'Verify your backups work by performing a test restore',
      completed: false,
      required: true,
      estimatedTime: 10,
      category: 'Verification'
    },
    {
      id: 'invite-team',
      title: 'Invite Team Members',
      description: 'Add your team members and configure their permissions',
      completed: false,
      required: false,
      estimatedTime: 5,
      category: 'Team'
    }
  ]
}
