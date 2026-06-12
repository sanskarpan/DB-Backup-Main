/**
 * Onboarding & Help System - Core Utilities
 * Comprehensive onboarding management system with tour, help, and feedback features
 */

// ==================== Types & Interfaces ====================

export interface TourStep {
  id: string
  target: string // CSS selector
  title: string
  content: string
  placement?: 'top' | 'bottom' | 'left' | 'right' | 'center'
  action?: {
    label: string
    onClick: () => void
  }
  beforeShow?: () => void | Promise<void>
  afterShow?: () => void | Promise<void>
  videoUrl?: string
  imageUrl?: string
  allowSkip?: boolean
  required?: boolean
}

export interface Tour {
  id: string
  name: string
  description: string
  steps: TourStep[]
  onComplete?: () => void
  onSkip?: () => void
  startCondition?: () => boolean
  priority?: number
}

export interface TooltipConfig {
  id: string
  target: string
  content: string
  title?: string
  trigger?: 'hover' | 'click' | 'focus' | 'manual'
  placement?: 'top' | 'bottom' | 'left' | 'right'
  delay?: number
  persistent?: boolean
  dismissible?: boolean
  showOnce?: boolean
  category?: string
}

export interface HelpArticle {
  id: string
  title: string
  content: string
  category: string
  tags: string[]
  videoUrl?: string
  relatedArticles?: string[]
  difficulty?: 'beginner' | 'intermediate' | 'advanced'
  estimatedReadTime?: number
  lastUpdated: Date
  helpful?: number
  notHelpful?: number
}

export interface WalkthroughStep {
  id: string
  title: string
  description: string
  element?: string // CSS selector to highlight
  position?: { x: number; y: number }
  action?: 'click' | 'input' | 'navigate' | 'wait'
  validation?: () => boolean
  hint?: string
  screenshot?: string
}

export interface Walkthrough {
  id: string
  title: string
  description: string
  steps: WalkthroughStep[]
  category: string
  difficulty: 'easy' | 'medium' | 'hard'
  estimatedTime: number
  prerequisites?: string[]
  onComplete?: () => void
}

export interface ChecklistItem {
  id: string
  title: string
  description: string
  completed: boolean
  required: boolean
  action?: () => void
  helpUrl?: string
  estimatedTime?: number
  category?: string
}

export interface OnboardingChecklist {
  id: string
  title: string
  description: string
  items: ChecklistItem[]
  progress: number
  completedAt?: Date
}

export interface FeedbackData {
  id?: string
  type: 'bug' | 'feature' | 'improvement' | 'question'
  rating?: number
  title: string
  description: string
  screenshot?: string
  userEmail?: string
  metadata?: Record<string, any>
  createdAt?: Date
  status?: 'pending' | 'reviewed' | 'resolved'
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
  suggestions?: string[]
  helpfulLinks?: Array<{ title: string; url: string }>
}

export interface WhatsNewItem {
  id: string
  version: string
  title: string
  description: string
  type: 'feature' | 'improvement' | 'fix' | 'breaking'
  releaseDate: Date
  imageUrl?: string
  videoUrl?: string
  learnMoreUrl?: string
  highlight?: boolean
}

export interface UserProgress {
  userId?: string
  completedTours: string[]
  dismissedTooltips: string[]
  completedWalkthroughs: string[]
  checklistProgress: Record<string, number>
  lastSeen: Record<string, Date>
  preferences: {
    showTooltips: boolean
    showTours: boolean
    helpAnimations: boolean
  }
}

// ==================== Tour Manager ====================

export class TourManager {
  private tours: Map<string, Tour> = new Map()
  private activeTour: Tour | null = null
  private currentStepIndex: number = 0
  private listeners: Map<string, Set<Function>> = new Map()
  private completedTours: Set<string> = new Set()

  constructor() {
    this.loadProgress()
  }

  /**
   * Register a new tour
   */
  registerTour(tour: Tour): void {
    this.tours.set(tour.id, tour)
  }

  /**
   * Start a tour by ID
   */
  async startTour(tourId: string): Promise<boolean> {
    const tour = this.tours.get(tourId)
    if (!tour) {
      console.warn(`Tour ${tourId} not found`)
      return false
    }

    // Check start condition
    if (tour.startCondition && !tour.startCondition()) {
      return false
    }

    this.activeTour = tour
    this.currentStepIndex = 0
    this.emit('tour-started', { tour })

    await this.showStep(0)
    return true
  }

  /**
   * Show a specific step
   */
  async showStep(stepIndex: number): Promise<void> {
    if (!this.activeTour || stepIndex >= this.activeTour.steps.length) {
      return
    }

    const step = this.activeTour.steps[stepIndex]
    this.currentStepIndex = stepIndex

    // Run beforeShow hook
    if (step.beforeShow) {
      await step.beforeShow()
    }

    // Emit event
    this.emit('step-shown', { tour: this.activeTour, step, index: stepIndex })

    // Run afterShow hook
    if (step.afterShow) {
      await step.afterShow()
    }
  }

  /**
   * Go to next step
   */
  async nextStep(): Promise<void> {
    if (!this.activeTour) return

    const nextIndex = this.currentStepIndex + 1
    if (nextIndex >= this.activeTour.steps.length) {
      await this.completeTour()
    } else {
      await this.showStep(nextIndex)
    }
  }

  /**
   * Go to previous step
   */
  async previousStep(): Promise<void> {
    if (!this.activeTour || this.currentStepIndex === 0) return
    await this.showStep(this.currentStepIndex - 1)
  }

  /**
   * Skip current tour
   */
  skipTour(): void {
    if (!this.activeTour) return

    const tour = this.activeTour
    this.activeTour = null
    this.currentStepIndex = 0

    if (tour.onSkip) {
      tour.onSkip()
    }

    this.emit('tour-skipped', { tour })
  }

  /**
   * Complete current tour
   */
  async completeTour(): Promise<void> {
    if (!this.activeTour) return

    const tour = this.activeTour
    this.completedTours.add(tour.id)
    this.saveProgress()

    if (tour.onComplete) {
      tour.onComplete()
    }

    this.emit('tour-completed', { tour })

    this.activeTour = null
    this.currentStepIndex = 0
  }

  /**
   * Get current step
   */
  getCurrentStep(): TourStep | null {
    if (!this.activeTour) return null
    return this.activeTour.steps[this.currentStepIndex]
  }

  /**
   * Get active tour
   */
  getActiveTour(): Tour | null {
    return this.activeTour
  }

  /**
   * Get current step index
   */
  getCurrentStepIndex(): number {
    return this.currentStepIndex
  }

  /**
   * Check if tour is completed
   */
  isTourCompleted(tourId: string): boolean {
    return this.completedTours.has(tourId)
  }

  /**
   * Reset tour progress
   */
  resetTour(tourId: string): void {
    this.completedTours.delete(tourId)
    this.saveProgress()
  }

  /**
   * Event listener management
   */
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback)
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data))
  }

  /**
   * Persistence
   */
  private loadProgress(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('onboarding-tours')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        this.completedTours = new Set(data.completed || [])
      } catch (e) {
        console.error('Failed to load tour progress:', e)
      }
    }
  }

  private saveProgress(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('onboarding-tours', JSON.stringify({
      completed: Array.from(this.completedTours),
      lastUpdated: new Date().toISOString()
    }))
  }
}

// ==================== Tooltip Manager ====================

export class TooltipManager {
  private tooltips: Map<string, TooltipConfig> = new Map()
  private dismissedTooltips: Set<string> = new Set()
  private activeTooltip: string | null = null

  constructor() {
    this.loadDismissedTooltips()
  }

  /**
   * Register tooltip
   */
  register(tooltip: TooltipConfig): void {
    this.tooltips.set(tooltip.id, tooltip)
  }

  /**
   * Unregister tooltip
   */
  unregister(tooltipId: string): void {
    this.tooltips.delete(tooltipId)
  }

  /**
   * Show tooltip
   */
  show(tooltipId: string): boolean {
    const tooltip = this.tooltips.get(tooltipId)
    if (!tooltip) return false

    // Check if dismissed and showOnce
    if (tooltip.showOnce && this.dismissedTooltips.has(tooltipId)) {
      return false
    }

    this.activeTooltip = tooltipId
    return true
  }

  /**
   * Hide tooltip
   */
  hide(tooltipId: string): void {
    if (this.activeTooltip === tooltipId) {
      this.activeTooltip = null
    }
  }

  /**
   * Dismiss tooltip permanently
   */
  dismiss(tooltipId: string): void {
    this.dismissedTooltips.add(tooltipId)
    this.hide(tooltipId)
    this.saveDismissedTooltips()
  }

  /**
   * Check if tooltip is dismissed
   */
  isDismissed(tooltipId: string): boolean {
    return this.dismissedTooltips.has(tooltipId)
  }

  /**
   * Get tooltip config
   */
  getTooltip(tooltipId: string): TooltipConfig | undefined {
    return this.tooltips.get(tooltipId)
  }

  /**
   * Get tooltips by category
   */
  getByCategory(category: string): TooltipConfig[] {
    return Array.from(this.tooltips.values()).filter(t => t.category === category)
  }

  /**
   * Reset all tooltips
   */
  resetAll(): void {
    this.dismissedTooltips.clear()
    this.saveDismissedTooltips()
  }

  /**
   * Persistence
   */
  private loadDismissedTooltips(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('dismissed-tooltips')
    if (stored) {
      try {
        this.dismissedTooltips = new Set(JSON.parse(stored))
      } catch (e) {
        console.error('Failed to load dismissed tooltips:', e)
      }
    }
  }

  private saveDismissedTooltips(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('dismissed-tooltips', JSON.stringify(
      Array.from(this.dismissedTooltips)
    ))
  }
}

// ==================== Help Search Manager ====================

export class HelpSearchManager {
  private articles: Map<string, HelpArticle> = new Map()
  private searchIndex: Map<string, Set<string>> = new Map() // word -> article IDs

  /**
   * Add article
   */
  addArticle(article: HelpArticle): void {
    this.articles.set(article.id, article)
    this.indexArticle(article)
  }

  /**
   * Search articles
   */
  search(query: string, options?: {
    category?: string
    difficulty?: string
    limit?: number
  }): HelpArticle[] {
    const words = this.tokenize(query.toLowerCase())
    const scores: Map<string, number> = new Map()

    // Calculate relevance scores
    words.forEach(word => {
      const articleIds = this.searchIndex.get(word)
      if (articleIds) {
        articleIds.forEach(id => {
          scores.set(id, (scores.get(id) || 0) + 1)
        })
      }
    })

    // Get articles and filter
    let results = Array.from(scores.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([id]) => this.articles.get(id)!)
      .filter(article => {
        if (options?.category && article.category !== options.category) {
          return false
        }
        if (options?.difficulty && article.difficulty !== options.difficulty) {
          return false
        }
        return true
      })

    // Apply limit
    if (options?.limit) {
      results = results.slice(0, options.limit)
    }

    return results
  }

  /**
   * Get article by ID
   */
  getArticle(id: string): HelpArticle | undefined {
    return this.articles.get(id)
  }

  /**
   * Get articles by category
   */
  getByCategory(category: string): HelpArticle[] {
    return Array.from(this.articles.values()).filter(a => a.category === category)
  }

  /**
   * Get related articles
   */
  getRelated(articleId: string, limit: number = 5): HelpArticle[] {
    const article = this.articles.get(articleId)
    if (!article) return []

    // Use related articles if specified
    if (article.relatedArticles) {
      return article.relatedArticles
        .map(id => this.articles.get(id))
        .filter(a => a !== undefined) as HelpArticle[]
    }

    // Otherwise, find by category and tags
    const related = Array.from(this.articles.values())
      .filter(a => a.id !== articleId)
      .map(a => {
        let score = 0
        if (a.category === article.category) score += 10
        article.tags.forEach(tag => {
          if (a.tags.includes(tag)) score += 5
        })
        return { article: a, score }
      })
      .filter(r => r.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, limit)
      .map(r => r.article)

    return related
  }

  /**
   * Mark article as helpful
   */
  markHelpful(articleId: string, helpful: boolean): void {
    const article = this.articles.get(articleId)
    if (!article) return

    if (helpful) {
      article.helpful = (article.helpful || 0) + 1
    } else {
      article.notHelpful = (article.notHelpful || 0) + 1
    }
  }

  /**
   * Private: Index article for search
   */
  private indexArticle(article: HelpArticle): void {
    const text = `${article.title} ${article.content} ${article.tags.join(' ')}`
    const words = this.tokenize(text.toLowerCase())

    words.forEach(word => {
      if (!this.searchIndex.has(word)) {
        this.searchIndex.set(word, new Set())
      }
      this.searchIndex.get(word)!.add(article.id)
    })
  }

  /**
   * Private: Tokenize text
   */
  private tokenize(text: string): string[] {
    return text
      .replace(/[^\w\s]/g, ' ')
      .split(/\s+/)
      .filter(word => word.length > 2) // Ignore very short words
  }
}

// ==================== Walkthrough Manager ====================

export class WalkthroughManager {
  private walkthroughs: Map<string, Walkthrough> = new Map()
  private activeWalkthrough: Walkthrough | null = null
  private currentStepIndex: number = 0
  private completedWalkthroughs: Set<string> = new Set()
  private listeners: Map<string, Set<Function>> = new Map()

  constructor() {
    this.loadProgress()
  }

  /**
   * Register walkthrough
   */
  register(walkthrough: Walkthrough): void {
    this.walkthroughs.set(walkthrough.id, walkthrough)
  }

  /**
   * Start walkthrough
   */
  async start(walkthroughId: string): Promise<boolean> {
    const walkthrough = this.walkthroughs.get(walkthroughId)
    if (!walkthrough) return false

    this.activeWalkthrough = walkthrough
    this.currentStepIndex = 0
    this.emit('walkthrough-started', { walkthrough })

    await this.showStep(0)
    return true
  }

  /**
   * Show step
   */
  async showStep(stepIndex: number): Promise<void> {
    if (!this.activeWalkthrough || stepIndex >= this.activeWalkthrough.steps.length) {
      return
    }

    const step = this.activeWalkthrough.steps[stepIndex]
    this.currentStepIndex = stepIndex

    // Highlight element if specified
    if (step.element) {
      this.highlightElement(step.element)
    }

    this.emit('step-shown', {
      walkthrough: this.activeWalkthrough,
      step,
      index: stepIndex
    })
  }

  /**
   * Next step
   */
  async next(): Promise<void> {
    if (!this.activeWalkthrough) return

    const currentStep = this.activeWalkthrough.steps[this.currentStepIndex]

    // Validate current step
    if (currentStep.validation && !currentStep.validation()) {
      this.emit('validation-failed', { step: currentStep })
      return
    }

    const nextIndex = this.currentStepIndex + 1
    if (nextIndex >= this.activeWalkthrough.steps.length) {
      await this.complete()
    } else {
      await this.showStep(nextIndex)
    }
  }

  /**
   * Previous step
   */
  async previous(): Promise<void> {
    if (!this.activeWalkthrough || this.currentStepIndex === 0) return
    await this.showStep(this.currentStepIndex - 1)
  }

  /**
   * Complete walkthrough
   */
  async complete(): Promise<void> {
    if (!this.activeWalkthrough) return

    const walkthrough = this.activeWalkthrough
    this.completedWalkthroughs.add(walkthrough.id)
    this.saveProgress()

    if (walkthrough.onComplete) {
      walkthrough.onComplete()
    }

    this.emit('walkthrough-completed', { walkthrough })

    this.activeWalkthrough = null
    this.currentStepIndex = 0
    this.clearHighlight()
  }

  /**
   * Exit walkthrough
   */
  exit(): void {
    if (!this.activeWalkthrough) return

    const walkthrough = this.activeWalkthrough
    this.activeWalkthrough = null
    this.currentStepIndex = 0
    this.clearHighlight()

    this.emit('walkthrough-exited', { walkthrough })
  }

  /**
   * Get current walkthrough
   */
  getActive(): Walkthrough | null {
    return this.activeWalkthrough
  }

  /**
   * Get current step
   */
  getCurrentStep(): WalkthroughStep | null {
    if (!this.activeWalkthrough) return null
    return this.activeWalkthrough.steps[this.currentStepIndex]
  }

  /**
   * Check if completed
   */
  isCompleted(walkthroughId: string): boolean {
    return this.completedWalkthroughs.has(walkthroughId)
  }

  /**
   * Event listeners
   */
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback)
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data))
  }

  /**
   * Private: Highlight element
   */
  private highlightElement(selector: string): void {
    if (typeof document === 'undefined') return

    const element = document.querySelector(selector)
    if (element) {
      element.classList.add('walkthrough-highlight')
      element.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }

  /**
   * Private: Clear highlight
   */
  private clearHighlight(): void {
    if (typeof document === 'undefined') return

    const highlighted = document.querySelectorAll('.walkthrough-highlight')
    highlighted.forEach(el => el.classList.remove('walkthrough-highlight'))
  }

  /**
   * Persistence
   */
  private loadProgress(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('completed-walkthroughs')
    if (stored) {
      try {
        this.completedWalkthroughs = new Set(JSON.parse(stored))
      } catch (e) {
        console.error('Failed to load walkthrough progress:', e)
      }
    }
  }

  private saveProgress(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('completed-walkthroughs', JSON.stringify(
      Array.from(this.completedWalkthroughs)
    ))
  }
}

// ==================== Checklist Manager ====================

export class ChecklistManager {
  private checklists: Map<string, OnboardingChecklist> = new Map()
  private listeners: Map<string, Set<Function>> = new Map()

  constructor() {
    this.loadProgress()
  }

  /**
   * Register checklist
   */
  register(checklist: OnboardingChecklist): void {
    this.checklists.set(checklist.id, checklist)
  }

  /**
   * Get checklist
   */
  get(checklistId: string): OnboardingChecklist | undefined {
    return this.checklists.get(checklistId)
  }

  /**
   * Complete checklist item
   */
  completeItem(checklistId: string, itemId: string): void {
    const checklist = this.checklists.get(checklistId)
    if (!checklist) return

    const item = checklist.items.find(i => i.id === itemId)
    if (!item) return

    item.completed = true
    this.updateProgress(checklist)
    this.saveProgress()

    this.emit('item-completed', { checklist, item })

    // Check if all completed
    if (checklist.progress === 100) {
      checklist.completedAt = new Date()
      this.emit('checklist-completed', { checklist })
    }
  }

  /**
   * Uncomplete item
   */
  uncompleteItem(checklistId: string, itemId: string): void {
    const checklist = this.checklists.get(checklistId)
    if (!checklist) return

    const item = checklist.items.find(i => i.id === itemId)
    if (!item) return

    item.completed = false
    checklist.completedAt = undefined
    this.updateProgress(checklist)
    this.saveProgress()

    this.emit('item-uncompleted', { checklist, item })
  }

  /**
   * Reset checklist
   */
  reset(checklistId: string): void {
    const checklist = this.checklists.get(checklistId)
    if (!checklist) return

    checklist.items.forEach(item => item.completed = false)
    checklist.completedAt = undefined
    this.updateProgress(checklist)
    this.saveProgress()

    this.emit('checklist-reset', { checklist })
  }

  /**
   * Event listeners
   */
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback)
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data))
  }

  /**
   * Private: Update progress
   */
  private updateProgress(checklist: OnboardingChecklist): void {
    const completed = checklist.items.filter(i => i.completed).length
    checklist.progress = Math.round((completed / checklist.items.length) * 100)
  }

  /**
   * Persistence
   */
  private loadProgress(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('checklist-progress')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        Object.entries(data).forEach(([id, items]: [string, any]) => {
          const checklist = this.checklists.get(id)
          if (checklist) {
            checklist.items.forEach(item => {
              item.completed = items[item.id] || false
            })
            this.updateProgress(checklist)
          }
        })
      } catch (e) {
        console.error('Failed to load checklist progress:', e)
      }
    }
  }

  private saveProgress(): void {
    if (typeof window === 'undefined') return

    const data: Record<string, Record<string, boolean>> = {}
    this.checklists.forEach((checklist, id) => {
      data[id] = {}
      checklist.items.forEach(item => {
        data[id][item.id] = item.completed
      })
    })

    localStorage.setItem('checklist-progress', JSON.stringify(data))
  }
}

// ==================== Feedback Manager ====================

export class FeedbackManager {
  private feedbackItems: FeedbackData[] = []
  private listeners: Map<string, Set<Function>> = new Map()

  /**
   * Submit feedback
   */
  async submit(feedback: FeedbackData): Promise<string> {
    const id = `feedback-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

    const feedbackData: FeedbackData = {
      ...feedback,
      id,
      createdAt: new Date(),
      status: 'pending'
    }

    this.feedbackItems.push(feedbackData)
    this.emit('feedback-submitted', { feedback: feedbackData })

    // In production, send to backend
    await this.sendToBackend(feedbackData)

    return id
  }

  /**
   * Get feedback by ID
   */
  get(id: string): FeedbackData | undefined {
    return this.feedbackItems.find(f => f.id === id)
  }

  /**
   * Get all feedback
   */
  getAll(): FeedbackData[] {
    return [...this.feedbackItems]
  }

  /**
   * Event listeners
   */
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback)
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data))
  }

  /**
   * Private: Send to backend
   */
  private async sendToBackend(feedback: FeedbackData): Promise<void> {
    // Mock implementation - replace with actual API call
    console.log('Feedback submitted:', feedback)

    // Example API call:
    // await fetch('/api/feedback', {
    //   method: 'POST',
    //   headers: { 'Content-Type': 'application/json' },
    //   body: JSON.stringify(feedback)
    // })
  }
}

// ==================== Chatbot Manager ====================

export class ChatbotManager {
  private messages: ChatMessage[] = []
  private knowledgeBase: Map<string, string> = new Map()
  private listeners: Map<string, Set<Function>> = new Map()

  constructor() {
    this.initializeKnowledgeBase()
  }

  /**
   * Send message
   */
  async sendMessage(content: string): Promise<ChatMessage> {
    // Add user message
    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}-user`,
      role: 'user',
      content,
      timestamp: new Date()
    }
    this.messages.push(userMessage)
    this.emit('message-sent', { message: userMessage })

    // Generate response
    const response = await this.generateResponse(content)
    this.messages.push(response)
    this.emit('message-received', { message: response })

    return response
  }

  /**
   * Get chat history
   */
  getHistory(): ChatMessage[] {
    return [...this.messages]
  }

  /**
   * Clear history
   */
  clearHistory(): void {
    this.messages = []
    this.emit('history-cleared', {})
  }

  /**
   * Add knowledge
   */
  addKnowledge(question: string, answer: string): void {
    this.knowledgeBase.set(question.toLowerCase(), answer)
  }

  /**
   * Event listeners
   */
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback)
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data))
  }

  /**
   * Private: Generate response
   */
  private async generateResponse(userMessage: string): Promise<ChatMessage> {
    // Simple keyword matching for now
    const lowerMessage = userMessage.toLowerCase()

    let responseText = ''
    let suggestions: string[] = []
    let helpfulLinks: Array<{ title: string; url: string }> = []

    // Check knowledge base
    for (const [question, answer] of this.knowledgeBase.entries()) {
      if (lowerMessage.includes(question)) {
        responseText = answer
        break
      }
    }

    // Default response
    if (!responseText) {
      responseText = "I'm here to help! Could you please provide more details about what you're looking for?"
      suggestions = [
        "How do I create a backup?",
        "How do I restore data?",
        "How do I schedule backups?",
        "What are the system requirements?"
      ]
    }

    return {
      id: `msg-${Date.now()}-assistant`,
      role: 'assistant',
      content: responseText,
      timestamp: new Date(),
      suggestions: suggestions.length > 0 ? suggestions : undefined,
      helpfulLinks: helpfulLinks.length > 0 ? helpfulLinks : undefined
    }
  }

  /**
   * Private: Initialize knowledge base
   */
  private initializeKnowledgeBase(): void {
    this.addKnowledge('backup', 'To create a backup, go to the Backups section and click "Create New Backup". You can configure the backup settings and schedule.')
    this.addKnowledge('restore', 'To restore data, navigate to the Restore section, select the backup you want to restore from, and follow the wizard.')
    this.addKnowledge('schedule', 'You can schedule backups in the Schedule section. Choose your frequency, time, and retention policy.')
    this.addKnowledge('help', 'I can help you with backups, restores, scheduling, and more! What would you like to know?')
    this.addKnowledge('getting started', 'Welcome! To get started, complete the onboarding checklist. It will guide you through setting up your first backup.')
  }
}

// ==================== Global Instance Export ====================

// Create global instances
export const tourManager = new TourManager()
export const tooltipManager = new TooltipManager()
export const helpSearchManager = new HelpSearchManager()
export const walkthroughManager = new WalkthroughManager()
export const checklistManager = new ChecklistManager()
export const feedbackManager = new FeedbackManager()
export const chatbotManager = new ChatbotManager()

// Export utility function to reset all
export function resetAllOnboarding(): void {
  if (typeof window === 'undefined') return

  localStorage.removeItem('onboarding-tours')
  localStorage.removeItem('dismissed-tooltips')
  localStorage.removeItem('completed-walkthroughs')
  localStorage.removeItem('checklist-progress')

  window.location.reload()
}
