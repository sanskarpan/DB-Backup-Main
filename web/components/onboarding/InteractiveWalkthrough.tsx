'use client'

import React, { useState, useEffect, useCallback } from 'react'
import { ChevronRight, ChevronLeft, X, Check, AlertCircle, Lightbulb } from 'lucide-react'
import { walkthroughManager, Walkthrough, WalkthroughStep } from '@/lib/onboarding-manager'

interface InteractiveWalkthroughProps {
  walkthrough: Walkthrough
  autoStart?: boolean
  onComplete?: () => void
  onExit?: () => void
}

export function InteractiveWalkthrough({
  walkthrough,
  autoStart = false,
  onComplete,
  onExit
}: InteractiveWalkthroughProps) {
  const [isActive, setIsActive] = useState(false)
  const [currentStep, setCurrentStep] = useState<WalkthroughStep | null>(null)
  const [stepIndex, setStepIndex] = useState(0)
  const [canProceed, setCanProceed] = useState(true)
  const [showValidationError, setShowValidationError] = useState(false)

  useEffect(() => {
    walkthroughManager.register(walkthrough)

    if (autoStart && !walkthroughManager.isCompleted(walkthrough.id)) {
      start()
    }

    // Event listeners
    const handleStepShown = ({ step, index }: any) => {
      setCurrentStep(step)
      setStepIndex(index)
      setShowValidationError(false)

      // Check if step can be completed
      if (step.validation) {
        const valid = step.validation()
        setCanProceed(valid)
      } else {
        setCanProceed(true)
      }
    }

    const handleCompleted = () => {
      setIsActive(false)
      onComplete?.()
    }

    const handleExited = () => {
      setIsActive(false)
      onExit?.()
    }

    const handleValidationFailed = () => {
      setShowValidationError(true)
      setCanProceed(false)
    }

    walkthroughManager.on('step-shown', handleStepShown)
    walkthroughManager.on('walkthrough-completed', handleCompleted)
    walkthroughManager.on('walkthrough-exited', handleExited)
    walkthroughManager.on('validation-failed', handleValidationFailed)

    return () => {
      walkthroughManager.off('step-shown', handleStepShown)
      walkthroughManager.off('walkthrough-completed', handleCompleted)
      walkthroughManager.off('walkthrough-exited', handleExited)
      walkthroughManager.off('validation-failed', handleValidationFailed)
    }
  }, [walkthrough, autoStart, onComplete, onExit])

  const start = useCallback(async () => {
    const started = await walkthroughManager.start(walkthrough.id)
    if (started) {
      setIsActive(true)
    }
  }, [walkthrough.id])

  const handleNext = useCallback(async () => {
    await walkthroughManager.next()
  }, [])

  const handlePrevious = useCallback(async () => {
    await walkthroughManager.previous()
  }, [])

  const handleExit = useCallback(() => {
    walkthroughManager.exit()
  }, [])

  if (!isActive || !currentStep) return null

  const progress = ((stepIndex + 1) / walkthrough.steps.length) * 100

  return (
    <>
      {/* Overlay */}
      <div className="fixed inset-0 z-40 bg-black/50" />

      {/* Walkthrough Panel */}
      <div className="fixed bottom-0 left-0 right-0 z-50 border-t-2 border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-800 sm:bottom-4 sm:left-4 sm:right-auto sm:w-96 sm:rounded-lg sm:border-2">
        {/* Header */}
        <div className="border-b border-gray-200 p-4 dark:border-gray-700">
          <div className="mb-2 flex items-start justify-between">
            <div className="flex-1">
              <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                {walkthrough.title}
              </h3>
              <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                Step {stepIndex + 1} of {walkthrough.steps.length}
              </p>
            </div>
            <button
              onClick={handleExit}
              className="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
              aria-label="Exit walkthrough"
            >
              <X className="h-5 w-5 text-gray-500" />
            </button>
          </div>

          {/* Progress bar */}
          <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div
              className="h-full bg-blue-600 transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {/* Content */}
        <div className="max-h-96 overflow-y-auto p-4">
          <h4 className="mb-2 font-semibold text-gray-900 dark:text-gray-100">
            {currentStep.title}
          </h4>

          <p className="mb-4 text-sm text-gray-700 dark:text-gray-300">
            {currentStep.description}
          </p>

          {/* Screenshot */}
          {currentStep.screenshot && (
            <div className="mb-4 overflow-hidden rounded-lg border-2 border-gray-200 dark:border-gray-700">
              <img
                src={currentStep.screenshot}
                alt={currentStep.title}
                className="w-full"
              />
            </div>
          )}

          {/* Hint */}
          {currentStep.hint && (
            <div className="mb-4 flex items-start gap-2 rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
              <Lightbulb className="h-5 w-5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
              <div>
                <p className="text-xs font-semibold text-blue-900 dark:text-blue-100">
                  Hint
                </p>
                <p className="mt-1 text-xs text-blue-800 dark:text-blue-200">
                  {currentStep.hint}
                </p>
              </div>
            </div>
          )}

          {/* Validation error */}
          {showValidationError && (
            <div className="mb-4 flex items-start gap-2 rounded-lg bg-red-50 p-3 dark:bg-red-900/20">
              <AlertCircle className="h-5 w-5 flex-shrink-0 text-red-600 dark:text-red-400" />
              <div>
                <p className="text-xs font-semibold text-red-900 dark:text-red-100">
                  Action Required
                </p>
                <p className="mt-1 text-xs text-red-800 dark:text-red-200">
                  Please complete this step before proceeding
                </p>
              </div>
            </div>
          )}

          {/* Action indicator */}
          {currentStep.action && (
            <div className="rounded-lg border-2 border-dashed border-blue-300 bg-blue-50 p-3 dark:border-blue-700 dark:bg-blue-900/20">
              <p className="text-xs font-semibold text-blue-900 dark:text-blue-100">
                Action: {currentStep.action}
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 p-4 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <button
              onClick={handlePrevious}
              disabled={stepIndex === 0}
              className="flex items-center gap-1 rounded-lg border-2 border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </button>

            <button
              onClick={handleNext}
              disabled={!canProceed}
              className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {stepIndex === walkthrough.steps.length - 1 ? (
                <>
                  <Check className="h-4 w-4" />
                  Complete
                </>
              ) : (
                <>
                  Next
                  <ChevronRight className="h-4 w-4" />
                </>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Highlight styles */}
      <style jsx global>{`
        .walkthrough-highlight {
          position: relative;
          z-index: 45 !important;
          outline: 3px solid rgba(59, 130, 246, 0.5);
          outline-offset: 4px;
          box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.5);
          border-radius: 8px;
        }
      `}</style>
    </>
  )
}

// Hook for easier usage
export function useWalkthrough(walkthroughId: string) {
  const [isCompleted, setIsCompleted] = useState(false)
  const [isActive, setIsActive] = useState(false)

  useEffect(() => {
    setIsCompleted(walkthroughManager.isCompleted(walkthroughId))
    setIsActive(walkthroughManager.getActive()?.id === walkthroughId)
  }, [walkthroughId])

  const start = useCallback(async () => {
    return await walkthroughManager.start(walkthroughId)
  }, [walkthroughId])

  const exit = useCallback(() => {
    walkthroughManager.exit()
  }, [])

  return {
    isCompleted,
    isActive,
    start,
    exit
  }
}

// Walkthrough Library Component
export function WalkthroughLibrary({ walkthroughs }: { walkthroughs: Walkthrough[] }) {
  const [selectedWalkthrough, setSelectedWalkthrough] = useState<Walkthrough | null>(null)

  const difficultyColors = {
    easy: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    hard: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Interactive Walkthroughs
        </h2>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Follow step-by-step guides to master key features
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {walkthroughs.map(walkthrough => {
          const isCompleted = walkthroughManager.isCompleted(walkthrough.id)
          return (
            <div
              key={walkthrough.id}
              className="overflow-hidden rounded-lg border-2 border-gray-200 bg-white transition-all hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
            >
              <div className="p-4">
                <div className="mb-3 flex items-start justify-between">
                  <div className="flex-1">
                    <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                      {walkthrough.title}
                    </h3>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                      {walkthrough.description}
                    </p>
                  </div>
                  {isCompleted && (
                    <div className="ml-2 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-green-100 dark:bg-green-900">
                      <Check className="h-4 w-4 text-green-600 dark:text-green-400" />
                    </div>
                  )}
                </div>

                <div className="mb-4 flex flex-wrap items-center gap-2">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${difficultyColors[walkthrough.difficulty]}`}>
                    {walkthrough.difficulty}
                  </span>
                  <span className="text-xs text-gray-600 dark:text-gray-400">
                    {walkthrough.steps.length} steps
                  </span>
                  <span className="text-xs text-gray-600 dark:text-gray-400">
                    ~{walkthrough.estimatedTime} min
                  </span>
                </div>

                <button
                  onClick={() => {
                    walkthroughManager.start(walkthrough.id)
                    setSelectedWalkthrough(walkthrough)
                  }}
                  className="w-full rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
                >
                  {isCompleted ? 'Restart' : 'Start Walkthrough'}
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {selectedWalkthrough && (
        <InteractiveWalkthrough
          walkthrough={selectedWalkthrough}
          onComplete={() => setSelectedWalkthrough(null)}
          onExit={() => setSelectedWalkthrough(null)}
        />
      )}
    </div>
  )
}

// Example walkthroughs
export const exampleWalkthroughs: Walkthrough[] = [
  {
    id: 'first-backup',
    title: 'Create Your First Backup',
    description: 'Learn how to create and configure your first database backup',
    category: 'Getting Started',
    difficulty: 'easy',
    estimatedTime: 5,
    steps: [
      {
        id: 'step-1',
        title: 'Navigate to Backups',
        description: 'Click on the "Backups" link in the navigation menu',
        element: '[data-walkthrough="nav-backups"]',
        action: 'click',
        hint: 'Look for the menu item with a database icon'
      },
      {
        id: 'step-2',
        title: 'Click New Backup',
        description: 'Click the "New Backup" button to start creating a backup',
        element: '[data-walkthrough="new-backup"]',
        action: 'click'
      },
      {
        id: 'step-3',
        title: 'Select Database',
        description: 'Choose which database you want to backup from the dropdown',
        element: '[data-walkthrough="select-database"]',
        action: 'input',
        hint: 'You can search for your database by typing'
      },
      {
        id: 'step-4',
        title: 'Configure Settings',
        description: 'Review and adjust backup settings as needed',
        element: '[data-walkthrough="backup-settings"]'
      },
      {
        id: 'step-5',
        title: 'Create Backup',
        description: 'Click "Create Backup" to start the backup process',
        element: '[data-walkthrough="create-backup-btn"]',
        action: 'click'
      }
    ]
  },
  {
    id: 'schedule-backup',
    title: 'Schedule Automatic Backups',
    description: 'Set up a recurring backup schedule',
    category: 'Configuration',
    difficulty: 'medium',
    estimatedTime: 8,
    steps: [
      {
        id: 'step-1',
        title: 'Open Schedules',
        description: 'Navigate to the Schedules section',
        element: '[data-walkthrough="nav-schedules"]',
        action: 'click'
      },
      {
        id: 'step-2',
        title: 'Create Schedule',
        description: 'Click "New Schedule" to create a backup schedule',
        element: '[data-walkthrough="new-schedule"]',
        action: 'click'
      },
      {
        id: 'step-3',
        title: 'Set Frequency',
        description: 'Choose how often backups should run (daily, weekly, etc.)',
        element: '[data-walkthrough="schedule-frequency"]',
        action: 'input'
      },
      {
        id: 'step-4',
        title: 'Configure Retention',
        description: 'Set how long to keep backups',
        element: '[data-walkthrough="retention-settings"]',
        action: 'input'
      }
    ]
  }
]
