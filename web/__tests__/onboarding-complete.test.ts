/**
 * Comprehensive Onboarding & Help System Tests
 * Tests for all WCAG 2.1 compliant onboarding features
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  TourManager,
  TooltipManager,
  HelpSearchManager,
  WalkthroughManager,
  ChecklistManager,
  FeedbackManager,
  ChatbotManager,
  Tour,
  TourStep,
  TooltipConfig,
  HelpArticle,
  Walkthrough,
  WalkthroughStep,
  OnboardingChecklist,
  ChecklistItem,
  FeedbackData
} from '../lib/onboarding-manager'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => { store[key] = value },
    removeItem: (key: string) => { delete store[key] },
    clear: () => { store = {} }
  }
})()

Object.defineProperty(global, 'localStorage', { value: localStorageMock })

describe('Tour Manager', () => {
  let tourManager: TourManager
  let sampleTour: Tour

  beforeEach(() => {
    localStorageMock.clear()
    tourManager = new TourManager()

    const steps: TourStep[] = [
      {
        id: 'step-1',
        target: '#element-1',
        title: 'Step 1',
        content: 'This is step 1',
        placement: 'bottom'
      },
      {
        id: 'step-2',
        target: '#element-2',
        title: 'Step 2',
        content: 'This is step 2',
        placement: 'top'
      }
    ]

    sampleTour = {
      id: 'test-tour',
      name: 'Test Tour',
      description: 'A test tour',
      steps
    }
  })

  it('should register a tour', () => {
    tourManager.registerTour(sampleTour)
    expect(tourManager['tours'].has('test-tour')).toBe(true)
  })

  it('should start a tour', async () => {
    tourManager.registerTour(sampleTour)
    const started = await tourManager.startTour('test-tour')
    expect(started).toBe(true)
    expect(tourManager.getActiveTour()).toBe(sampleTour)
  })

  it('should track current step index', async () => {
    tourManager.registerTour(sampleTour)
    await tourManager.startTour('test-tour')
    expect(tourManager.getCurrentStepIndex()).toBe(0)

    await tourManager.nextStep()
    expect(tourManager.getCurrentStepIndex()).toBe(1)
  })

  it('should navigate to previous step', async () => {
    tourManager.registerTour(sampleTour)
    await tourManager.startTour('test-tour')
    await tourManager.nextStep()
    await tourManager.previousStep()
    expect(tourManager.getCurrentStepIndex()).toBe(0)
  })

  it('should complete tour', async () => {
    tourManager.registerTour(sampleTour)
    await tourManager.startTour('test-tour')
    await tourManager.completeTour()

    expect(tourManager.isTourCompleted('test-tour')).toBe(true)
    expect(tourManager.getActiveTour()).toBeNull()
  })

  it('should skip tour', () => {
    tourManager.registerTour(sampleTour)
    tourManager.startTour('test-tour')
    tourManager.skipTour()

    expect(tourManager.getActiveTour()).toBeNull()
    expect(tourManager.isTourCompleted('test-tour')).toBe(false)
  })

  it('should persist completed tours', async () => {
    tourManager.registerTour(sampleTour)
    await tourManager.startTour('test-tour')
    await tourManager.completeTour()

    const stored = localStorage.getItem('onboarding-tours')
    expect(stored).toBeTruthy()
    const data = JSON.parse(stored!)
    expect(data.completed).toContain('test-tour')
  })

  it('should emit tour events', async () => {
    const mockCallback = vi.fn()
    tourManager.on('tour-started', mockCallback)

    tourManager.registerTour(sampleTour)
    await tourManager.startTour('test-tour')

    expect(mockCallback).toHaveBeenCalledWith({ tour: sampleTour })
  })
})

describe('Tooltip Manager', () => {
  let tooltipManager: TooltipManager
  let sampleTooltip: TooltipConfig

  beforeEach(() => {
    localStorageMock.clear()
    tooltipManager = new TooltipManager()

    sampleTooltip = {
      id: 'test-tooltip',
      target: '#element',
      content: 'This is a tooltip',
      title: 'Test Tooltip',
      trigger: 'hover',
      showOnce: true
    }
  })

  it('should register tooltip', () => {
    tooltipManager.register(sampleTooltip)
    expect(tooltipManager.getTooltip('test-tooltip')).toBe(sampleTooltip)
  })

  it('should show tooltip', () => {
    tooltipManager.register(sampleTooltip)
    const shown = tooltipManager.show('test-tooltip')
    expect(shown).toBe(true)
  })

  it('should not show dismissed tooltip with showOnce', () => {
    tooltipManager.register(sampleTooltip)
    tooltipManager.dismiss('test-tooltip')
    const shown = tooltipManager.show('test-tooltip')
    expect(shown).toBe(false)
  })

  it('should persist dismissed tooltips', () => {
    tooltipManager.register(sampleTooltip)
    tooltipManager.dismiss('test-tooltip')

    const stored = localStorage.getItem('dismissed-tooltips')
    expect(stored).toBeTruthy()
    const data = JSON.parse(stored!)
    expect(data).toContain('test-tooltip')
  })

  it('should reset all tooltips', () => {
    tooltipManager.register(sampleTooltip)
    tooltipManager.dismiss('test-tooltip')
    tooltipManager.resetAll()

    expect(tooltipManager.isDismissed('test-tooltip')).toBe(false)
  })

  it('should get tooltips by category', () => {
    const tooltip1: TooltipConfig = { ...sampleTooltip, id: 'tip-1', category: 'setup' }
    const tooltip2: TooltipConfig = { ...sampleTooltip, id: 'tip-2', category: 'setup' }
    const tooltip3: TooltipConfig = { ...sampleTooltip, id: 'tip-3', category: 'advanced' }

    tooltipManager.register(tooltip1)
    tooltipManager.register(tooltip2)
    tooltipManager.register(tooltip3)

    const setupTooltips = tooltipManager.getByCategory('setup')
    expect(setupTooltips.length).toBe(2)
  })
})

describe('Help Search Manager', () => {
  let searchManager: HelpSearchManager
  let sampleArticles: HelpArticle[]

  beforeEach(() => {
    searchManager = new HelpSearchManager()

    sampleArticles = [
      {
        id: '1',
        title: 'How to create a backup',
        content: 'This guide explains how to create database backups.',
        category: 'Getting Started',
        tags: ['backup', 'tutorial'],
        difficulty: 'beginner',
        lastUpdated: new Date()
      },
      {
        id: '2',
        title: 'Advanced restore strategies',
        content: 'Learn about point-in-time recovery and advanced restore techniques.',
        category: 'Advanced',
        tags: ['restore', 'pitr'],
        difficulty: 'advanced',
        lastUpdated: new Date()
      }
    ]

    sampleArticles.forEach(article => searchManager.addArticle(article))
  })

  it('should search articles by keywords', () => {
    const results = searchManager.search('backup')
    expect(results.length).toBeGreaterThan(0)
    expect(results[0].title).toContain('backup')
  })

  it('should filter by category', () => {
    const results = searchManager.search('backup', { category: 'Getting Started' })
    expect(results.length).toBe(1)
    expect(results[0].category).toBe('Getting Started')
  })

  it('should filter by difficulty', () => {
    const results = searchManager.search('', { difficulty: 'advanced' })
    expect(results.length).toBe(1)
    expect(results[0].difficulty).toBe('advanced')
  })

  it('should limit results', () => {
    const results = searchManager.search('', { limit: 1 })
    expect(results.length).toBe(1)
  })

  it('should get article by ID', () => {
    const article = searchManager.getArticle('1')
    expect(article?.id).toBe('1')
  })

  it('should get articles by category', () => {
    const articles = searchManager.getByCategory('Advanced')
    expect(articles.length).toBe(1)
    expect(articles[0].category).toBe('Advanced')
  })

  it('should mark article as helpful', () => {
    const article = searchManager.getArticle('1')!
    const initialHelpful = article.helpful || 0

    searchManager.markHelpful('1', true)
    expect(article.helpful).toBe(initialHelpful + 1)
  })
})

describe('Walkthrough Manager', () => {
  let walkthroughManager: WalkthroughManager
  let sampleWalkthrough: Walkthrough

  beforeEach(() => {
    localStorageMock.clear()
    walkthroughManager = new WalkthroughManager()

    const steps: WalkthroughStep[] = [
      {
        id: 'step-1',
        title: 'Navigate to dashboard',
        description: 'Click on the dashboard link',
        element: '#dashboard-link',
        action: 'click'
      },
      {
        id: 'step-2',
        title: 'Create backup',
        description: 'Click the create backup button',
        element: '#create-backup',
        action: 'click'
      }
    ]

    sampleWalkthrough = {
      id: 'test-walkthrough',
      title: 'Test Walkthrough',
      description: 'A test walkthrough',
      category: 'Getting Started',
      difficulty: 'easy',
      estimatedTime: 5,
      steps
    }
  })

  it('should register walkthrough', () => {
    walkthroughManager.register(sampleWalkthrough)
    expect(walkthroughManager['walkthroughs'].has('test-walkthrough')).toBe(true)
  })

  it('should start walkthrough', async () => {
    walkthroughManager.register(sampleWalkthrough)
    const started = await walkthroughManager.start('test-walkthrough')
    expect(started).toBe(true)
    expect(walkthroughManager.getActive()).toBe(sampleWalkthrough)
  })

  it('should navigate through steps', async () => {
    walkthroughManager.register(sampleWalkthrough)
    await walkthroughManager.start('test-walkthrough')

    const firstStep = walkthroughManager.getCurrentStep()
    expect(firstStep?.id).toBe('step-1')

    await walkthroughManager.next()
    const secondStep = walkthroughManager.getCurrentStep()
    expect(secondStep?.id).toBe('step-2')
  })

  it('should complete walkthrough', async () => {
    walkthroughManager.register(sampleWalkthrough)
    await walkthroughManager.start('test-walkthrough')
    await walkthroughManager.complete()

    expect(walkthroughManager.isCompleted('test-walkthrough')).toBe(true)
    expect(walkthroughManager.getActive()).toBeNull()
  })

  it('should exit walkthrough', async () => {
    walkthroughManager.register(sampleWalkthrough)
    await walkthroughManager.start('test-walkthrough')
    walkthroughManager.exit()

    expect(walkthroughManager.getActive()).toBeNull()
    expect(walkthroughManager.isCompleted('test-walkthrough')).toBe(false)
  })
})

describe('Checklist Manager', () => {
  let checklistManager: ChecklistManager
  let sampleChecklist: OnboardingChecklist

  beforeEach(() => {
    localStorageMock.clear()
    checklistManager = new ChecklistManager()

    const items: ChecklistItem[] = [
      {
        id: 'item-1',
        title: 'Connect database',
        description: 'Connect your first database',
        completed: false,
        required: true
      },
      {
        id: 'item-2',
        title: 'Create backup',
        description: 'Create your first backup',
        completed: false,
        required: true
      }
    ]

    sampleChecklist = {
      id: 'test-checklist',
      title: 'Test Checklist',
      description: 'A test checklist',
      items,
      progress: 0
    }
  })

  it('should register checklist', () => {
    checklistManager.register(sampleChecklist)
    expect(checklistManager.get('test-checklist')).toBe(sampleChecklist)
  })

  it('should complete checklist item', () => {
    checklistManager.register(sampleChecklist)
    checklistManager.completeItem('test-checklist', 'item-1')

    const checklist = checklistManager.get('test-checklist')!
    const item = checklist.items.find(i => i.id === 'item-1')!
    expect(item.completed).toBe(true)
  })

  it('should update progress', () => {
    checklistManager.register(sampleChecklist)
    checklistManager.completeItem('test-checklist', 'item-1')

    const checklist = checklistManager.get('test-checklist')!
    expect(checklist.progress).toBe(50)
  })

  it('should mark checklist as complete when all items done', () => {
    checklistManager.register(sampleChecklist)
    checklistManager.completeItem('test-checklist', 'item-1')
    checklistManager.completeItem('test-checklist', 'item-2')

    const checklist = checklistManager.get('test-checklist')!
    expect(checklist.progress).toBe(100)
    expect(checklist.completedAt).toBeTruthy()
  })

  it('should persist checklist progress', () => {
    checklistManager.register(sampleChecklist)
    checklistManager.completeItem('test-checklist', 'item-1')

    const stored = localStorage.getItem('checklist-progress')
    expect(stored).toBeTruthy()
    const data = JSON.parse(stored!)
    expect(data['test-checklist']['item-1']).toBe(true)
  })

  it('should reset checklist', () => {
    checklistManager.register(sampleChecklist)
    checklistManager.completeItem('test-checklist', 'item-1')
    checklistManager.reset('test-checklist')

    const checklist = checklistManager.get('test-checklist')!
    expect(checklist.progress).toBe(0)
    expect(checklist.items.every(i => !i.completed)).toBe(true)
  })
})

describe('Feedback Manager', () => {
  let feedbackManager: FeedbackManager

  beforeEach(() => {
    feedbackManager = new FeedbackManager()
  })

  it('should submit feedback', async () => {
    const feedback: FeedbackData = {
      type: 'bug',
      title: 'Test Bug',
      description: 'This is a test bug report',
      rating: 3
    }

    const id = await feedbackManager.submit(feedback)
    expect(id).toBeTruthy()
  })

  it('should get feedback by ID', async () => {
    const feedback: FeedbackData = {
      type: 'feature',
      title: 'Test Feature',
      description: 'This is a test feature request'
    }

    const id = await feedbackManager.submit(feedback)
    const retrieved = feedbackManager.get(id)

    expect(retrieved?.id).toBe(id)
    expect(retrieved?.title).toBe('Test Feature')
  })

  it('should get all feedback', async () => {
    await feedbackManager.submit({
      type: 'bug',
      title: 'Bug 1',
      description: 'Description 1'
    })

    await feedbackManager.submit({
      type: 'feature',
      title: 'Feature 1',
      description: 'Description 2'
    })

    const all = feedbackManager.getAll()
    expect(all.length).toBe(2)
  })
})

describe('Chatbot Manager', () => {
  let chatbotManager: ChatbotManager

  beforeEach(() => {
    chatbotManager = new ChatbotManager()
  })

  it('should send message', async () => {
    const response = await chatbotManager.sendMessage('How do I create a backup?')
    expect(response.role).toBe('assistant')
    expect(response.content).toBeTruthy()
  })

  it('should track message history', async () => {
    await chatbotManager.sendMessage('Test message 1')
    await chatbotManager.sendMessage('Test message 2')

    const history = chatbotManager.getHistory()
    expect(history.length).toBeGreaterThanOrEqual(4) // 2 user + 2 assistant
  })

  it('should clear history', async () => {
    await chatbotManager.sendMessage('Test message')
    chatbotManager.clearHistory()

    const history = chatbotManager.getHistory()
    expect(history.length).toBe(0)
  })

  it('should add knowledge', () => {
    chatbotManager.addKnowledge('test question', 'test answer')
    expect(chatbotManager['knowledgeBase'].has('test question')).toBe(true)
  })
})

describe('Integration Tests', () => {
  it('should handle complete onboarding flow', async () => {
    const tourManager = new TourManager()
    const checklistManager = new ChecklistManager()

    // Register tour
    const tour: Tour = {
      id: 'onboarding-tour',
      name: 'Onboarding Tour',
      description: 'Get started',
      steps: [
        { id: 'step-1', target: '#el-1', title: 'Step 1', content: 'Content 1' }
      ]
    }
    tourManager.registerTour(tour)

    // Register checklist
    const checklist: OnboardingChecklist = {
      id: 'setup-checklist',
      title: 'Setup',
      description: 'Complete setup',
      items: [
        { id: 'item-1', title: 'Item 1', description: 'Desc 1', completed: false, required: true }
      ],
      progress: 0
    }
    checklistManager.register(checklist)

    // Start tour
    await tourManager.startTour('onboarding-tour')
    expect(tourManager.getActiveTour()).toBeTruthy()

    // Complete tour
    await tourManager.completeTour()
    expect(tourManager.isTourCompleted('onboarding-tour')).toBe(true)

    // Complete checklist
    checklistManager.completeItem('setup-checklist', 'item-1')
    expect(checklistManager.get('setup-checklist')?.progress).toBe(100)
  })
})
