/**
 * Enhanced Accessibility Utilities
 * Comprehensive toolkit for WCAG 2.1 AAA compliance
 */

// =============================================================================
// KEYBOARD NAVIGATION
// =============================================================================

export class KeyboardNavigationManager {
  private focusableElements: HTMLElement[] = []
  private currentIndex = -1
  private rootElement: HTMLElement

  constructor(rootElement: HTMLElement = document.body) {
    this.rootElement = rootElement
    this.updateFocusableElements()
  }

  /**
   * Update list of focusable elements
   */
  updateFocusableElements(): void {
    const selector = [
      'a[href]',
      'button:not([disabled])',
      'textarea:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]',
    ].join(',')

    this.focusableElements = Array.from(
      this.rootElement.querySelectorAll<HTMLElement>(selector)
    ).filter((el) => {
      const style = window.getComputedStyle(el)
      return style.display !== 'none' && style.visibility !== 'hidden'
    })
  }

  /**
   * Focus next element
   */
  focusNext(): void {
    this.updateFocusableElements()
    this.currentIndex = (this.currentIndex + 1) % this.focusableElements.length
    this.focusableElements[this.currentIndex]?.focus()
  }

  /**
   * Focus previous element
   */
  focusPrevious(): void {
    this.updateFocusableElements()
    this.currentIndex =
      (this.currentIndex - 1 + this.focusableElements.length) %
      this.focusableElements.length
    this.focusableElements[this.currentIndex]?.focus()
  }

  /**
   * Focus first element
   */
  focusFirst(): void {
    this.updateFocusableElements()
    this.currentIndex = 0
    this.focusableElements[0]?.focus()
  }

  /**
   * Focus last element
   */
  focusLast(): void {
    this.updateFocusableElements()
    this.currentIndex = this.focusableElements.length - 1
    this.focusableElements[this.currentIndex]?.focus()
  }

  /**
   * Trap focus within container
   */
  trapFocus(container: HTMLElement): () => void {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return

      const focusable = Array.from(
        container.querySelectorAll<HTMLElement>([
          'a[href]',
          'button:not([disabled])',
          'input:not([disabled])',
          'select:not([disabled])',
          'textarea:not([disabled])',
          '[tabindex]:not([tabindex="-1"])',
        ].join(','))
      )

      if (focusable.length === 0) return

      const first = focusable[0]
      const last = focusable[focusable.length - 1]

      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault()
          last.focus()
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault()
          first.focus()
        }
      }
    }

    container.addEventListener('keydown', handleKeyDown)
    return () => container.removeEventListener('keydown', handleKeyDown)
  }
}

/**
 * Setup global keyboard shortcuts
 */
export function setupKeyboardShortcuts() {
  const shortcuts = new Map<string, () => void>()

  const handleKeyDown = (e: KeyboardEvent) => {
    const key = [
      e.ctrlKey && 'ctrl',
      e.altKey && 'alt',
      e.shiftKey && 'shift',
      e.metaKey && 'meta',
      e.key.toLowerCase(),
    ]
      .filter(Boolean)
      .join('+')

    const handler = shortcuts.get(key)
    if (handler) {
      e.preventDefault()
      handler()
    }
  }

  document.addEventListener('keydown', handleKeyDown)

  return {
    register: (key: string, handler: () => void) => {
      shortcuts.set(key, handler)
    },
    unregister: (key: string) => {
      shortcuts.delete(key)
    },
    destroy: () => {
      document.removeEventListener('keydown', handleKeyDown)
      shortcuts.clear()
    },
  }
}

// =============================================================================
// FOCUS INDICATORS
// =============================================================================

export class FocusIndicatorManager {
  private styleElement: HTMLStyleElement | null = null

  /**
   * Enable enhanced focus indicators
   */
  enable(options: {
    color?: string
    width?: number
    offset?: number
    style?: 'solid' | 'dashed' | 'dotted'
  } = {}): void {
    const {
      color = '#0066cc',
      width = 3,
      offset = 2,
      style = 'solid',
    } = options

    this.styleElement = document.createElement('style')
    this.styleElement.textContent = `
      *:focus {
        outline: ${width}px ${style} ${color} !important;
        outline-offset: ${offset}px !important;
      }

      *:focus:not(:focus-visible) {
        outline: none !important;
      }

      *:focus-visible {
        outline: ${width}px ${style} ${color} !important;
        outline-offset: ${offset}px !important;
      }

      /* Enhanced focus for interactive elements */
      button:focus-visible,
      a:focus-visible,
      input:focus-visible,
      select:focus-visible,
      textarea:focus-visible {
        outline: ${width}px ${style} ${color} !important;
        outline-offset: ${offset}px !important;
        box-shadow: 0 0 0 ${width + offset}px rgba(0, 102, 204, 0.1) !important;
      }
    `
    document.head.appendChild(this.styleElement)
  }

  /**
   * Disable enhanced focus indicators
   */
  disable(): void {
    if (this.styleElement) {
      document.head.removeChild(this.styleElement)
      this.styleElement = null
    }
  }

  /**
   * Show focus ring programmatically
   */
  showFocusRing(element: HTMLElement): void {
    element.classList.add('force-focus-visible')
  }

  /**
   * Hide focus ring
   */
  hideFocusRing(element: HTMLElement): void {
    element.classList.remove('force-focus-visible')
  }
}

// =============================================================================
// ARIA LIVE REGIONS
// =============================================================================

export class ARIALiveRegionManager {
  private regions: Map<string, HTMLElement> = new Map()

  /**
   * Create a live region
   */
  createRegion(
    id: string,
    options: {
      politeness?: 'polite' | 'assertive' | 'off'
      atomic?: boolean
      relevant?: string
    } = {}
  ): HTMLElement {
    const { politeness = 'polite', atomic = true, relevant = 'additions text' } = options

    const region = document.createElement('div')
    region.id = id
    region.setAttribute('role', 'status')
    region.setAttribute('aria-live', politeness)
    region.setAttribute('aria-atomic', atomic.toString())
    region.setAttribute('aria-relevant', relevant)
    region.className = 'sr-only'
    region.style.cssText = `
      position: absolute;
      left: -10000px;
      width: 1px;
      height: 1px;
      overflow: hidden;
    `

    document.body.appendChild(region)
    this.regions.set(id, region)

    return region
  }

  /**
   * Announce message to screen readers
   */
  announce(
    message: string,
    options: {
      regionId?: string
      politeness?: 'polite' | 'assertive'
      clearAfter?: number
    } = {}
  ): void {
    const { regionId = 'default-live-region', politeness = 'polite', clearAfter = 3000 } = options

    let region = this.regions.get(regionId)
    if (!region) {
      region = this.createRegion(regionId, { politeness })
    }

    region.textContent = message

    if (clearAfter > 0) {
      setTimeout(() => {
        region!.textContent = ''
      }, clearAfter)
    }
  }

  /**
   * Clear all announcements
   */
  clear(regionId?: string): void {
    if (regionId) {
      const region = this.regions.get(regionId)
      if (region) region.textContent = ''
    } else {
      this.regions.forEach((region) => {
        region.textContent = ''
      })
    }
  }

  /**
   * Remove a live region
   */
  removeRegion(id: string): void {
    const region = this.regions.get(id)
    if (region) {
      document.body.removeChild(region)
      this.regions.delete(id)
    }
  }

  /**
   * Remove all live regions
   */
  removeAll(): void {
    this.regions.forEach((region) => {
      document.body.removeChild(region)
    })
    this.regions.clear()
  }
}

// Global instance
export const ariaLive = new ARIALiveRegionManager()

// =============================================================================
// VOICE CONTROL
// =============================================================================

export class VoiceControlManager {
  private recognition: any = null
  private isListening = false
  private commands: Map<string, () => void> = new Map()

  constructor() {
    // Check for Speech Recognition API
    const SpeechRecognition =
      (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition

    if (SpeechRecognition) {
      this.recognition = new SpeechRecognition()
      this.recognition.continuous = true
      this.recognition.interimResults = false
      this.recognition.lang = 'en-US'

      this.recognition.onresult = (event: any) => {
        const transcript = event.results[event.results.length - 1][0].transcript
          .toLowerCase()
          .trim()

        this.handleCommand(transcript)
      }

      this.recognition.onerror = (event: any) => {
        console.error('Speech recognition error:', event.error)
        this.isListening = false
      }

      this.recognition.onend = () => {
        if (this.isListening) {
          // Restart if we're still supposed to be listening
          this.recognition.start()
        }
      }
    }
  }

  /**
   * Start listening for voice commands
   */
  start(): void {
    if (!this.recognition) {
      console.error('Speech recognition not supported')
      return
    }

    if (!this.isListening) {
      this.isListening = true
      this.recognition.start()
      ariaLive.announce('Voice control started', { politeness: 'polite' })
    }
  }

  /**
   * Stop listening
   */
  stop(): void {
    if (this.recognition && this.isListening) {
      this.isListening = false
      this.recognition.stop()
      ariaLive.announce('Voice control stopped', { politeness: 'polite' })
    }
  }

  /**
   * Register a voice command
   */
  registerCommand(command: string, handler: () => void): void {
    this.commands.set(command.toLowerCase(), handler)
  }

  /**
   * Unregister a command
   */
  unregisterCommand(command: string): void {
    this.commands.delete(command.toLowerCase())
  }

  /**
   * Handle recognized command
   */
  private handleCommand(transcript: string): void {
    // Check for exact match
    const handler = this.commands.get(transcript)
    if (handler) {
      handler()
      ariaLive.announce(`Command executed: ${transcript}`, { politeness: 'polite' })
      return
    }

    // Check for partial matches
    for (const [command, handler] of this.commands.entries()) {
      if (transcript.includes(command)) {
        handler()
        ariaLive.announce(`Command executed: ${command}`, { politeness: 'polite' })
        return
      }
    }

    ariaLive.announce(`Command not recognized: ${transcript}`, { politeness: 'polite' })
  }

  /**
   * Check if voice control is supported
   */
  isSupported(): boolean {
    return !!this.recognition
  }

  /**
   * Get listening status
   */
  getListeningStatus(): boolean {
    return this.isListening
  }
}

// =============================================================================
// VIDEO CAPTIONS
// =============================================================================

export interface Caption {
  startTime: number
  endTime: number
  text: string
}

export class VideoCaptionManager {
  private video: HTMLVideoElement
  private captions: Caption[] = []
  private container: HTMLElement | null = null
  private currentCaptionIndex = -1

  constructor(video: HTMLVideoElement) {
    this.video = video
  }

  /**
   * Load captions from VTT file
   */
  async loadVTT(url: string): Promise<void> {
    const response = await fetch(url)
    const text = await response.text()
    this.captions = this.parseVTT(text)
  }

  /**
   * Load captions from array
   */
  loadCaptions(captions: Caption[]): void {
    this.captions = captions
  }

  /**
   * Enable captions
   */
  enable(container?: HTMLElement): void {
    if (!container) {
      // Create default container
      this.container = document.createElement('div')
      this.container.className = 'video-captions'
      this.container.setAttribute('aria-live', 'polite')
      this.video.parentElement?.appendChild(this.container)
    } else {
      this.container = container
    }

    this.video.addEventListener('timeupdate', this.handleTimeUpdate)
  }

  /**
   * Disable captions
   */
  disable(): void {
    this.video.removeEventListener('timeupdate', this.handleTimeUpdate)
    if (this.container) {
      this.container.textContent = ''
    }
  }

  /**
   * Handle time update
   */
  private handleTimeUpdate = (): void => {
    if (!this.container) return

    const currentTime = this.video.currentTime

    const captionIndex = this.captions.findIndex(
      (caption) => currentTime >= caption.startTime && currentTime <= caption.endTime
    )

    if (captionIndex !== this.currentCaptionIndex) {
      this.currentCaptionIndex = captionIndex

      if (captionIndex >= 0) {
        this.container.textContent = this.captions[captionIndex].text
      } else {
        this.container.textContent = ''
      }
    }
  }

  /**
   * Parse VTT file
   */
  private parseVTT(vtt: string): Caption[] {
    const captions: Caption[] = []
    const lines = vtt.split('\n')

    let i = 0
    while (i < lines.length) {
      const line = lines[i].trim()

      // Look for timestamp line
      if (line.includes('-->')) {
        const [start, end] = line.split('-->').map((t) => this.parseTimestamp(t.trim()))

        // Get caption text (next lines until empty line)
        const textLines: string[] = []
        i++
        while (i < lines.length && lines[i].trim() !== '') {
          textLines.push(lines[i].trim())
          i++
        }

        captions.push({
          startTime: start,
          endTime: end,
          text: textLines.join(' '),
        })
      }

      i++
    }

    return captions
  }

  /**
   * Parse VTT timestamp
   */
  private parseTimestamp(timestamp: string): number {
    const parts = timestamp.split(':')
    let seconds = 0

    if (parts.length === 3) {
      // HH:MM:SS.mmm
      seconds = parseInt(parts[0]) * 3600 + parseInt(parts[1]) * 60 + parseFloat(parts[2])
    } else if (parts.length === 2) {
      // MM:SS.mmm
      seconds = parseInt(parts[0]) * 60 + parseFloat(parts[1])
    }

    return seconds
  }
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

/**
 * Add ARIA labels to elements
 */
export function enhanceARIA(element: HTMLElement, options: {
  label?: string
  describedBy?: string
  role?: string
  expanded?: boolean
  selected?: boolean
  checked?: boolean
  invalid?: boolean
  required?: boolean
}): void {
  if (options.label) element.setAttribute('aria-label', options.label)
  if (options.describedBy) element.setAttribute('aria-describedby', options.describedBy)
  if (options.role) element.setAttribute('role', options.role)
  if (options.expanded !== undefined) element.setAttribute('aria-expanded', String(options.expanded))
  if (options.selected !== undefined) element.setAttribute('aria-selected', String(options.selected))
  if (options.checked !== undefined) element.setAttribute('aria-checked', String(options.checked))
  if (options.invalid !== undefined) element.setAttribute('aria-invalid', String(options.invalid))
  if (options.required !== undefined) element.setAttribute('aria-required', String(options.required))
}

/**
 * Create accessible button
 */
export function createAccessibleButton(options: {
  text: string
  onClick: () => void
  ariaLabel?: string
  disabled?: boolean
}): HTMLButtonElement {
  const button = document.createElement('button')
  button.textContent = options.text
  button.onclick = options.onClick
  if (options.ariaLabel) button.setAttribute('aria-label', options.ariaLabel)
  if (options.disabled) button.disabled = true

  return button
}

/**
 * Setup accessibility testing mode
 */
export function enableAccessibilityTesting(): void {
  // Add visual focus indicators
  const style = document.createElement('style')
  style.textContent = `
    *:focus {
      outline: 3px dashed red !important;
      outline-offset: 2px !important;
    }

    [aria-hidden="true"] {
      border: 2px dashed orange !important;
    }

    [role] {
      border: 1px dotted blue !important;
    }
  `
  document.head.appendChild(style)

  console.log('Accessibility testing mode enabled')
}
