'use client'

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import {
  tourManager,
  tooltipManager,
  walkthroughManager,
  checklistManager,
  Tour,
  TooltipConfig,
  Walkthrough,
  OnboardingChecklist
} from '@/lib/onboarding-manager'

interface OnboardingContextValue {
  // Tours
  activeTour: Tour | null
  startTour: (tourId: string) => Promise<boolean>
  skipTour: () => void
  completedTours: string[]

  // Tooltips
  dismissedTooltips: string[]
  dismissTooltip: (tooltipId: string) => void
  resetTooltips: () => void

  // Walkthroughs
  activeWalkthrough: Walkthrough | null
  startWalkthrough: (walkthroughId: string) => Promise<boolean>
  exitWalkthrough: () => void
  completedWalkthroughs: string[]

  // Checklists
  checklists: Map<string, OnboardingChecklist>
  completeChecklistItem: (checklistId: string, itemId: string) => void
  getChecklistProgress: (checklistId: string) => number

  // Help system state
  helpOpen: boolean
  setHelpOpen: (open: boolean) => void
  chatbotOpen: boolean
  setChatbotOpen: (open: boolean) => void
  feedbackOpen: boolean
  setFeedbackOpen: (open: boolean) => void

  // Global preferences
  showOnboarding: boolean
  setShowOnboarding: (show: boolean) => void
  hasCompletedInitialSetup: boolean
  markInitialSetupComplete: () => void
}

const OnboardingContext = createContext<OnboardingContextValue | undefined>(undefined)

export function OnboardingProvider({ children }: { children: React.ReactNode }) {
  // Tours
  const [activeTour, setActiveTour] = useState<Tour | null>(null)
  const [completedTours, setCompletedTours] = useState<string[]>([])

  // Tooltips
  const [dismissedTooltips, setDismissedTooltips] = useState<string[]>([])

  // Walkthroughs
  const [activeWalkthrough, setActiveWalkthrough] = useState<Walkthrough | null>(null)
  const [completedWalkthroughs, setCompletedWalkthroughs] = useState<string[]>([])

  // Checklists
  const [checklists, setChecklists] = useState<Map<string, OnboardingChecklist>>(new Map())

  // Help system
  const [helpOpen, setHelpOpen] = useState(false)
  const [chatbotOpen, setChatbotOpen] = useState(false)
  const [feedbackOpen, setFeedbackOpen] = useState(false)

  // Preferences
  const [showOnboarding, setShowOnboarding] = useState(true)
  const [hasCompletedInitialSetup, setHasCompletedInitialSetup] = useState(false)

  // Initialize from localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return

    // Load preferences
    const storedPrefs = localStorage.getItem('onboarding-preferences')
    if (storedPrefs) {
      try {
        const prefs = JSON.parse(storedPrefs)
        setShowOnboarding(prefs.showOnboarding ?? true)
        setHasCompletedInitialSetup(prefs.hasCompletedInitialSetup ?? false)
      } catch (e) {
        console.error('Failed to load onboarding preferences:', e)
      }
    }

    // Load completed tours
    const storedTours = localStorage.getItem('onboarding-tours')
    if (storedTours) {
      try {
        const data = JSON.parse(storedTours)
        setCompletedTours(data.completed || [])
      } catch (e) {
        console.error('Failed to load tour progress:', e)
      }
    }

    // Load dismissed tooltips
    const storedTooltips = localStorage.getItem('dismissed-tooltips')
    if (storedTooltips) {
      try {
        setDismissedTooltips(JSON.parse(storedTooltips))
      } catch (e) {
        console.error('Failed to load dismissed tooltips:', e)
      }
    }

    // Load completed walkthroughs
    const storedWalkthroughs = localStorage.getItem('completed-walkthroughs')
    if (storedWalkthroughs) {
      try {
        setCompletedWalkthroughs(JSON.parse(storedWalkthroughs))
      } catch (e) {
        console.error('Failed to load walkthrough progress:', e)
      }
    }
  }, [])

  // Save preferences
  useEffect(() => {
    if (typeof window === 'undefined') return

    localStorage.setItem('onboarding-preferences', JSON.stringify({
      showOnboarding,
      hasCompletedInitialSetup
    }))
  }, [showOnboarding, hasCompletedInitialSetup])

  // Tour event listeners
  useEffect(() => {
    const handleTourStarted = ({ tour }: any) => {
      setActiveTour(tour)
    }

    const handleTourCompleted = ({ tour }: any) => {
      setActiveTour(null)
      setCompletedTours(prev => [...prev, tour.id])
    }

    const handleTourSkipped = () => {
      setActiveTour(null)
    }

    tourManager.on('tour-started', handleTourStarted)
    tourManager.on('tour-completed', handleTourCompleted)
    tourManager.on('tour-skipped', handleTourSkipped)

    return () => {
      tourManager.off('tour-started', handleTourStarted)
      tourManager.off('tour-completed', handleTourCompleted)
      tourManager.off('tour-skipped', handleTourSkipped)
    }
  }, [])

  // Walkthrough event listeners
  useEffect(() => {
    const handleWalkthroughStarted = ({ walkthrough }: any) => {
      setActiveWalkthrough(walkthrough)
    }

    const handleWalkthroughCompleted = ({ walkthrough }: any) => {
      setActiveWalkthrough(null)
      setCompletedWalkthroughs(prev => [...prev, walkthrough.id])
    }

    const handleWalkthroughExited = () => {
      setActiveWalkthrough(null)
    }

    walkthroughManager.on('walkthrough-started', handleWalkthroughStarted)
    walkthroughManager.on('walkthrough-completed', handleWalkthroughCompleted)
    walkthroughManager.on('walkthrough-exited', handleWalkthroughExited)

    return () => {
      walkthroughManager.off('walkthrough-started', handleWalkthroughStarted)
      walkthroughManager.off('walkthrough-completed', handleWalkthroughCompleted)
      walkthroughManager.off('walkthrough-exited', handleWalkthroughExited)
    }
  }, [])

  // Tour functions
  const startTour = useCallback(async (tourId: string) => {
    return await tourManager.startTour(tourId)
  }, [])

  const skipTour = useCallback(() => {
    tourManager.skipTour()
  }, [])

  // Tooltip functions
  const dismissTooltip = useCallback((tooltipId: string) => {
    tooltipManager.dismiss(tooltipId)
    setDismissedTooltips(prev => [...prev, tooltipId])
  }, [])

  const resetTooltips = useCallback(() => {
    tooltipManager.resetAll()
    setDismissedTooltips([])
  }, [])

  // Walkthrough functions
  const startWalkthrough = useCallback(async (walkthroughId: string) => {
    return await walkthroughManager.start(walkthroughId)
  }, [])

  const exitWalkthrough = useCallback(() => {
    walkthroughManager.exit()
  }, [])

  // Checklist functions
  const completeChecklistItem = useCallback((checklistId: string, itemId: string) => {
    checklistManager.completeItem(checklistId, itemId)
    // Trigger re-render by creating new Map
    setChecklists(new Map(checklists))
  }, [checklists])

  const getChecklistProgress = useCallback((checklistId: string): number => {
    const checklist = checklistManager.get(checklistId)
    return checklist?.progress || 0
  }, [])

  // Preferences
  const markInitialSetupComplete = useCallback(() => {
    setHasCompletedInitialSetup(true)
  }, [])

  const value: OnboardingContextValue = {
    // Tours
    activeTour,
    startTour,
    skipTour,
    completedTours,

    // Tooltips
    dismissedTooltips,
    dismissTooltip,
    resetTooltips,

    // Walkthroughs
    activeWalkthrough,
    startWalkthrough,
    exitWalkthrough,
    completedWalkthroughs,

    // Checklists
    checklists,
    completeChecklistItem,
    getChecklistProgress,

    // Help system
    helpOpen,
    setHelpOpen,
    chatbotOpen,
    setChatbotOpen,
    feedbackOpen,
    setFeedbackOpen,

    // Preferences
    showOnboarding,
    setShowOnboarding,
    hasCompletedInitialSetup,
    markInitialSetupComplete
  }

  return (
    <OnboardingContext.Provider value={value}>
      {children}
    </OnboardingContext.Provider>
  )
}

// Hook to use onboarding context
export function useOnboarding() {
  const context = useContext(OnboardingContext)
  if (context === undefined) {
    throw new Error('useOnboarding must be used within OnboardingProvider')
  }
  return context
}

// Keyboard shortcuts hook
export function useOnboardingShortcuts() {
  const { setHelpOpen, setChatbotOpen, setFeedbackOpen } = useOnboarding()

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ctrl/Cmd + Shift + H: Open help
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'H') {
        e.preventDefault()
        setHelpOpen(true)
      }

      // Ctrl/Cmd + Shift + C: Open chatbot
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'C') {
        e.preventDefault()
        setChatbotOpen(true)
      }

      // Ctrl/Cmd + Shift + F: Open feedback
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
        e.preventDefault()
        setFeedbackOpen(true)
      }

      // Escape: Close all
      if (e.key === 'Escape') {
        setHelpOpen(false)
        setChatbotOpen(false)
        setFeedbackOpen(false)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [setHelpOpen, setChatbotOpen, setFeedbackOpen])
}

// Hook to track user progress
export function useOnboardingProgress() {
  const { completedTours, completedWalkthroughs, getChecklistProgress } = useOnboarding()

  const calculateOverallProgress = useCallback((
    totalTours: number,
    totalWalkthroughs: number,
    checklistIds: string[]
  ): number => {
    const tourProgress = (completedTours.length / totalTours) * 100
    const walkthroughProgress = (completedWalkthroughs.length / totalWalkthroughs) * 100

    let checklistProgress = 0
    checklistIds.forEach(id => {
      checklistProgress += getChecklistProgress(id)
    })
    checklistProgress = checklistProgress / checklistIds.length

    return Math.round((tourProgress + walkthroughProgress + checklistProgress) / 3)
  }, [completedTours.length, completedWalkthroughs.length, getChecklistProgress])

  return {
    completedTours: completedTours.length,
    completedWalkthroughs: completedWalkthroughs.length,
    calculateOverallProgress
  }
}
