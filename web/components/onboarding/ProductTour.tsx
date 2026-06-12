'use client'

import React, { useState, useEffect, useCallback, useRef } from 'react'
import { X, ChevronLeft, ChevronRight, Play, SkipForward } from 'lucide-react'
import { tourManager, Tour, TourStep } from '@/lib/onboarding-manager'

interface ProductTourProps {
  tour: Tour
  autoStart?: boolean
  onComplete?: () => void
  onSkip?: () => void
}

export function ProductTour({ tour, autoStart = false, onComplete, onSkip }: ProductTourProps) {
  const [isActive, setIsActive] = useState(false)
  const [currentStep, setCurrentStep] = useState<TourStep | null>(null)
  const [stepIndex, setStepIndex] = useState(0)
  const [totalSteps, setTotalSteps] = useState(0)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const tooltipRef = useRef<HTMLDivElement>(null)

  // Initialize tour
  useEffect(() => {
    tourManager.registerTour(tour)

    if (autoStart && !tourManager.isTourCompleted(tour.id)) {
      startTour()
    }

    // Event listeners
    const handleStepShown = ({ step, index }: any) => {
      setCurrentStep(step)
      setStepIndex(index)
      setTotalSteps(tour.steps.length)
      updatePosition(step.target, step.placement || 'bottom')
    }

    const handleTourCompleted = () => {
      setIsActive(false)
      onComplete?.()
    }

    const handleTourSkipped = () => {
      setIsActive(false)
      onSkip?.()
    }

    tourManager.on('step-shown', handleStepShown)
    tourManager.on('tour-completed', handleTourCompleted)
    tourManager.on('tour-skipped', handleTourSkipped)

    return () => {
      tourManager.off('step-shown', handleStepShown)
      tourManager.off('tour-completed', handleTourCompleted)
      tourManager.off('tour-skipped', handleTourSkipped)
    }
  }, [tour, autoStart, onComplete, onSkip])

  // Update position when step changes
  useEffect(() => {
    if (currentStep) {
      updatePosition(currentStep.target, currentStep.placement || 'bottom')
    }
  }, [currentStep])

  const startTour = useCallback(async () => {
    const started = await tourManager.startTour(tour.id)
    if (started) {
      setIsActive(true)
    }
  }, [tour.id])

  const updatePosition = useCallback((selector: string, placement: string) => {
    if (typeof document === 'undefined') return

    const targetElement = document.querySelector(selector) as HTMLElement
    if (!targetElement) {
      console.warn(`Tour target not found: ${selector}`)
      return
    }

    // Scroll to element
    targetElement.scrollIntoView({ behavior: 'smooth', block: 'center' })

    // Highlight element
    const overlay = document.querySelector('.tour-overlay')
    if (overlay) {
      const rect = targetElement.getBoundingClientRect()
      const padding = 8

      // Add highlight class
      targetElement.classList.add('tour-highlight')

      // Calculate tooltip position
      setTimeout(() => {
        if (!tooltipRef.current) return

        const tooltipRect = tooltipRef.current.getBoundingClientRect()
        let top = 0
        let left = 0

        switch (placement) {
          case 'top':
            top = rect.top - tooltipRect.height - padding
            left = rect.left + (rect.width - tooltipRect.width) / 2
            break
          case 'bottom':
            top = rect.bottom + padding
            left = rect.left + (rect.width - tooltipRect.width) / 2
            break
          case 'left':
            top = rect.top + (rect.height - tooltipRect.height) / 2
            left = rect.left - tooltipRect.width - padding
            break
          case 'right':
            top = rect.top + (rect.height - tooltipRect.height) / 2
            left = rect.right + padding
            break
          case 'center':
            top = window.innerHeight / 2 - tooltipRect.height / 2
            left = window.innerWidth / 2 - tooltipRect.width / 2
            break
        }

        // Keep tooltip in viewport
        top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - padding))
        left = Math.max(padding, Math.min(left, window.innerWidth - tooltipRect.width - padding))

        setPosition({ top, left })
      }, 100)
    }
  }, [])

  const handleNext = useCallback(async () => {
    // Clear current highlight
    if (currentStep) {
      const targetElement = document.querySelector(currentStep.target)
      targetElement?.classList.remove('tour-highlight')
    }

    await tourManager.nextStep()
  }, [currentStep])

  const handlePrevious = useCallback(async () => {
    // Clear current highlight
    if (currentStep) {
      const targetElement = document.querySelector(currentStep.target)
      targetElement?.classList.remove('tour-highlight')
    }

    await tourManager.previousStep()
  }, [currentStep])

  const handleSkip = useCallback(() => {
    // Clear highlight
    if (currentStep) {
      const targetElement = document.querySelector(currentStep.target)
      targetElement?.classList.remove('tour-highlight')
    }

    tourManager.skipTour()
    onSkip?.()
  }, [currentStep, onSkip])

  const handleClose = useCallback(() => {
    handleSkip()
  }, [handleSkip])

  if (!isActive || !currentStep) return null

  return (
    <>
      {/* Overlay */}
      <div className="tour-overlay fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />

      {/* Tour Tooltip */}
      <div
        ref={tooltipRef}
        className="fixed z-50 w-full max-w-md rounded-lg bg-white p-6 shadow-2xl dark:bg-gray-800"
        style={{
          top: `${position.top}px`,
          left: `${position.left}px`,
          transition: 'top 0.3s ease, left 0.3s ease'
        }}
        role="dialog"
        aria-labelledby="tour-step-title"
        aria-describedby="tour-step-content"
      >
        {/* Header */}
        <div className="mb-4 flex items-start justify-between">
          <div className="flex-1">
            <div className="mb-1 flex items-center gap-2">
              <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
                Step {stepIndex + 1} of {totalSteps}
              </span>
              {currentStep.required && (
                <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-200">
                  Required
                </span>
              )}
            </div>
            <h3 id="tour-step-title" className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {currentStep.title}
            </h3>
          </div>
          <button
            onClick={handleClose}
            className="ml-4 rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
            aria-label="Close tour"
          >
            <X className="h-5 w-5 text-gray-500" />
          </button>
        </div>

        {/* Content */}
        <div id="tour-step-content" className="mb-6 space-y-4">
          {/* Video */}
          {currentStep.videoUrl && (
            <div className="aspect-video overflow-hidden rounded-lg">
              <video
                src={currentStep.videoUrl}
                controls
                className="h-full w-full"
              />
            </div>
          )}

          {/* Image */}
          {currentStep.imageUrl && (
            <div className="overflow-hidden rounded-lg">
              <img
                src={currentStep.imageUrl}
                alt={currentStep.title}
                className="h-full w-full object-cover"
              />
            </div>
          )}

          {/* Description */}
          <p className="text-sm leading-relaxed text-gray-700 dark:text-gray-300">
            {currentStep.content}
          </p>

          {/* Action Button */}
          {currentStep.action && (
            <button
              onClick={currentStep.action.onClick}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              <Play className="h-4 w-4" />
              {currentStep.action.label}
            </button>
          )}
        </div>

        {/* Progress Bar */}
        <div className="mb-4">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div
              className="h-full bg-blue-600 transition-all duration-300"
              style={{ width: `${((stepIndex + 1) / totalSteps) * 100}%` }}
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between">
          <div className="flex gap-2">
            {stepIndex > 0 && (
              <button
                onClick={handlePrevious}
                className="flex items-center gap-1 rounded-lg border-2 border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                <ChevronLeft className="h-4 w-4" />
                Previous
              </button>
            )}
          </div>

          <div className="flex gap-2">
            {stepIndex < totalSteps - 1 && (
              <button
                onClick={handleSkip}
                className="flex items-center gap-1 px-4 py-2 text-sm font-medium text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              >
                <SkipForward className="h-4 w-4" />
                Skip Tour
              </button>
            )}

            <button
              onClick={handleNext}
              className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              {stepIndex < totalSteps - 1 ? (
                <>
                  Next
                  <ChevronRight className="h-4 w-4" />
                </>
              ) : (
                'Complete'
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Styles */}
      <style jsx global>{`
        .tour-highlight {
          position: relative;
          z-index: 45 !important;
          box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.5), 0 0 0 9999px rgba(0, 0, 0, 0.5);
          border-radius: 8px;
        }

        .tour-overlay {
          pointer-events: none;
        }
      `}</style>
    </>
  )
}

// Hook for easier usage
export function useProductTour(tour: Tour) {
  const [isCompleted, setIsCompleted] = useState(false)

  useEffect(() => {
    setIsCompleted(tourManager.isTourCompleted(tour.id))
  }, [tour.id])

  const start = useCallback(async () => {
    return await tourManager.startTour(tour.id)
  }, [tour.id])

  const reset = useCallback(() => {
    tourManager.resetTour(tour.id)
    setIsCompleted(false)
  }, [tour.id])

  return {
    isCompleted,
    start,
    reset
  }
}

// Tour Launcher Component
export function TourLauncher({ tours }: { tours: Tour[] }) {
  return (
    <div className="rounded-lg border-2 border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
      <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
        Product Tours
      </h3>
      <div className="space-y-3">
        {tours.map(tour => {
          const isCompleted = tourManager.isTourCompleted(tour.id)
          return (
            <div
              key={tour.id}
              className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700"
            >
              <div className="flex-1">
                <h4 className="font-medium text-gray-900 dark:text-gray-100">
                  {tour.name}
                </h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {tour.description}
                </p>
                <p className="mt-1 text-xs text-gray-500">
                  {tour.steps.length} steps
                </p>
              </div>
              <div className="ml-4 flex gap-2">
                {isCompleted && (
                  <span className="rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200">
                    Completed
                  </span>
                )}
                <button
                  onClick={() => tourManager.startTour(tour.id)}
                  className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
                >
                  {isCompleted ? 'Restart' : 'Start Tour'}
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
