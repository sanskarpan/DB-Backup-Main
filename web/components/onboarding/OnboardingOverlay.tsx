'use client';

import { useEffect, useState } from 'react';
import { getOnboardingEngine, OnboardingStep, OnboardingTour } from '@/lib/onboarding/onboarding-engine';

interface OnboardingOverlayProps {
  autoStart?: boolean;
}

export default function OnboardingOverlay({ autoStart = false }: OnboardingOverlayProps) {
  const [isActive, setIsActive] = useState(false);
  const [currentTour, setCurrentTour] = useState<OnboardingTour | null>(null);
  const [currentStep, setCurrentStep] = useState<OnboardingStep | null>(null);
  const [progress, setProgress] = useState(0);
  const [targetPosition, setTargetPosition] = useState<{ top: number; left: number; width: number; height: number } | null>(null);

  useEffect(() => {
    const engine = getOnboardingEngine();

    // Subscribe to state changes
    const unsubscribe = engine.subscribe((state) => {
      setIsActive(state.isActive);
      const { tour, step, progress: prog } = engine.getCurrentState();
      setCurrentTour(tour);
      setCurrentStep(step);
      setProgress(prog);

      // Calculate target element position
      if (step?.targetElement) {
        updateTargetPosition(step.targetElement);
      } else {
        setTargetPosition(null);
      }
    });

    // Auto-start getting started tour if user is new
    if (autoStart && !engine.hasTourCompleted('getting-started')) {
      setTimeout(() => engine.startTour('getting-started'), 1000);
    }

    return unsubscribe;
  }, [autoStart]);

  const updateTargetPosition = (selector: string) => {
    const element = document.querySelector(selector);
    if (element) {
      const rect = element.getBoundingClientRect();
      setTargetPosition({
        top: rect.top + window.scrollY,
        left: rect.left + window.scrollX,
        width: rect.width,
        height: rect.height,
      });

      // Scroll element into view
      element.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  };

  const handleNext = () => {
    getOnboardingEngine().nextStep();
  };

  const handlePrevious = () => {
    getOnboardingEngine().previousStep();
  };

  const handleSkip = () => {
    getOnboardingEngine().skipTour();
  };

  const handleClose = () => {
    getOnboardingEngine().skipTour();
  };

  if (!isActive || !currentStep || !currentTour) {
    return null;
  }

  const tooltipPosition = getTooltipPosition(currentStep.position, targetPosition);

  return (
    <>
      {/* Dark overlay */}
      <div
        className="fixed inset-0 bg-black/50 z-[9998] transition-opacity"
        style={{ pointerEvents: 'none' }}
      />

      {/* Spotlight effect on target element */}
      {targetPosition && (
        <div
          className="fixed z-[9999] pointer-events-none"
          style={{
            top: targetPosition.top - 4,
            left: targetPosition.left - 4,
            width: targetPosition.width + 8,
            height: targetPosition.height + 8,
            boxShadow: '0 0 0 99999px rgba(0, 0, 0, 0.5)',
            borderRadius: '8px',
            border: '2px solid #3b82f6',
          }}
        />
      )}

      {/* Tooltip */}
      <div
        className="fixed z-[10000] bg-white dark:bg-gray-800 rounded-lg shadow-2xl p-6 max-w-md"
        style={{
          top: tooltipPosition.top,
          left: tooltipPosition.left,
          transform: tooltipPosition.transform,
        }}
      >
        {/* Header */}
        <div className="flex justify-between items-start mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              {currentStep.title}
            </h3>
            {currentStep.showProgress && (
              <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Step {currentTour.steps.indexOf(currentStep) + 1} of {currentTour.steps.length}
              </div>
            )}
          </div>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            aria-label="Close"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Description */}
        <p className="text-gray-700 dark:text-gray-300 mb-4">
          {currentStep.description}
        </p>

        {/* Progress bar */}
        {currentStep.showProgress && (
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 mb-4">
            <div
              className="bg-blue-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-between items-center">
          <div>
            {currentStep.canSkip !== false && (
              <button
                onClick={handleSkip}
                className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              >
                Skip Tour
              </button>
            )}
          </div>
          <div className="flex gap-2">
            {currentTour.steps.indexOf(currentStep) > 0 && (
              <button
                onClick={handlePrevious}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
              >
                Previous
              </button>
            )}
            <button
              onClick={handleNext}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
            >
              {currentTour.steps.indexOf(currentStep) === currentTour.steps.length - 1
                ? 'Finish'
                : 'Next'}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

function getTooltipPosition(
  position: OnboardingStep['position'],
  targetPosition: { top: number; left: number; width: number; height: number } | null
): { top: number; left: number; transform: string } {
  if (!targetPosition || position === 'center') {
    return {
      top: window.innerHeight / 2,
      left: window.innerWidth / 2,
      transform: 'translate(-50%, -50%)',
    };
  }

  const offset = 20;

  switch (position) {
    case 'top':
      return {
        top: targetPosition.top - offset,
        left: targetPosition.left + targetPosition.width / 2,
        transform: 'translate(-50%, -100%)',
      };
    case 'bottom':
      return {
        top: targetPosition.top + targetPosition.height + offset,
        left: targetPosition.left + targetPosition.width / 2,
        transform: 'translate(-50%, 0)',
      };
    case 'left':
      return {
        top: targetPosition.top + targetPosition.height / 2,
        left: targetPosition.left - offset,
        transform: 'translate(-100%, -50%)',
      };
    case 'right':
      return {
        top: targetPosition.top + targetPosition.height / 2,
        left: targetPosition.left + targetPosition.width + offset,
        transform: 'translate(0, -50%)',
      };
    default:
      return {
        top: window.innerHeight / 2,
        left: window.innerWidth / 2,
        transform: 'translate(-50%, -50%)',
      };
  }
}
