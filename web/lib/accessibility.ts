/**
 * Accessibility Utilities for WCAG 2.1 Level AA Compliance
 *
 * This module provides utilities for ensuring WCAG 2.1 Level AA compliance
 * across the application. It includes helpers for:
 * - Color contrast checking
 * - Keyboard navigation
 * - Screen reader support
 * - Focus management
 * - ARIA attributes
 */

// Color Contrast Utilities (WCAG 2.1 - 1.4.3 Contrast Minimum)

/**
 * Calculate relative luminance of a color
 * @param r Red component (0-255)
 * @param g Green component (0-255)
 * @param b Blue component (0-255)
 */
function getRelativeLuminance(r: number, g: number, b: number): number {
  const rsRGB = r / 255
  const gsRGB = g / 255
  const bsRGB = b / 255

  const rLinear = rsRGB <= 0.03928 ? rsRGB / 12.92 : Math.pow((rsRGB + 0.055) / 1.055, 2.4)
  const gLinear = gsRGB <= 0.03928 ? gsRGB / 12.92 : Math.pow((gsRGB + 0.055) / 1.055, 2.4)
  const bLinear = bsRGB <= 0.03928 ? bsRGB / 12.92 : Math.pow((bsRGB + 0.055) / 1.055, 2.4)

  return 0.2126 * rLinear + 0.7152 * gLinear + 0.0722 * bLinear
}

/**
 * Parse hex color to RGB
 */
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : null
}

/**
 * Calculate contrast ratio between two colors
 * @param color1 First color (hex format)
 * @param color2 Second color (hex format)
 * @returns Contrast ratio (1-21)
 */
export function getContrastRatio(color1: string, color2: string): number {
  const rgb1 = hexToRgb(color1)
  const rgb2 = hexToRgb(color2)

  if (!rgb1 || !rgb2) {
    throw new Error('Invalid color format. Use hex format (#RRGGBB)')
  }

  const l1 = getRelativeLuminance(rgb1.r, rgb1.g, rgb1.b)
  const l2 = getRelativeLuminance(rgb2.r, rgb2.g, rgb2.b)

  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)

  return (lighter + 0.05) / (darker + 0.05)
}

/**
 * Check if color combination meets WCAG AA standards
 * @param foreground Foreground color (hex)
 * @param background Background color (hex)
 * @param fontSize Font size in pixels
 * @param isBold Whether text is bold
 */
export function meetsWCAG_AA(
  foreground: string,
  background: string,
  fontSize: number = 16,
  isBold: boolean = false
): boolean {
  const ratio = getContrastRatio(foreground, background)
  const isLargeText = fontSize >= 18 || (fontSize >= 14 && isBold)

  // WCAG AA requires:
  // - 4.5:1 for normal text
  // - 3:1 for large text (18pt+ or 14pt+ bold)
  return isLargeText ? ratio >= 3 : ratio >= 4.5
}

/**
 * Check if color combination meets WCAG AAA standards
 */
export function meetsWCAG_AAA(
  foreground: string,
  background: string,
  fontSize: number = 16,
  isBold: boolean = false
): boolean {
  const ratio = getContrastRatio(foreground, background)
  const isLargeText = fontSize >= 18 || (fontSize >= 14 && isBold)

  // WCAG AAA requires:
  // - 7:1 for normal text
  // - 4.5:1 for large text
  return isLargeText ? ratio >= 4.5 : ratio >= 7
}

// Keyboard Navigation Utilities (WCAG 2.1 - 2.1 Keyboard Accessible)

export const FOCUSABLE_ELEMENTS = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
]

/**
 * Get all focusable elements within a container
 */
export function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll(FOCUSABLE_ELEMENTS.join(',')))
}

/**
 * Trap focus within a container (useful for modals, dropdowns)
 */
export function trapFocus(container: HTMLElement, event: KeyboardEvent): void {
  if (event.key !== 'Tab') return

  const focusableElements = getFocusableElements(container)
  const firstElement = focusableElements[0]
  const lastElement = focusableElements[focusableElements.length - 1]

  if (event.shiftKey) {
    // Shift + Tab
    if (document.activeElement === firstElement) {
      event.preventDefault()
      lastElement?.focus()
    }
  } else {
    // Tab
    if (document.activeElement === lastElement) {
      event.preventDefault()
      firstElement?.focus()
    }
  }
}

/**
 * Create a focus trap for modal dialogs
 */
export function createFocusTrap(container: HTMLElement): () => void {
  const handleKeyDown = (e: KeyboardEvent) => trapFocus(container, e)

  container.addEventListener('keydown', handleKeyDown)

  // Focus first focusable element
  const firstFocusable = getFocusableElements(container)[0]
  firstFocusable?.focus()

  // Return cleanup function
  return () => {
    container.removeEventListener('keydown', handleKeyDown)
  }
}

// Screen Reader Utilities (WCAG 2.1 - 4.1.2 Name, Role, Value)

/**
 * Announce message to screen readers
 */
export function announceToScreenReader(message: string, priority: 'polite' | 'assertive' = 'polite'): void {
  const announcement = document.createElement('div')
  announcement.setAttribute('role', 'status')
  announcement.setAttribute('aria-live', priority)
  announcement.setAttribute('aria-atomic', 'true')
  announcement.className = 'sr-only'
  announcement.textContent = message

  document.body.appendChild(announcement)

  // Remove after announcement
  setTimeout(() => {
    document.body.removeChild(announcement)
  }, 1000)
}

/**
 * Generate unique ID for ARIA relationships
 */
let idCounter = 0
export function generateAriaId(prefix: string = 'aria'): string {
  return `${prefix}-${++idCounter}-${Date.now()}`
}

// Focus Management Utilities

/**
 * Restore focus to previously focused element
 */
export function createFocusRestore(): () => void {
  const previouslyFocused = document.activeElement as HTMLElement

  return () => {
    if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
      previouslyFocused.focus()
    }
  }
}

/**
 * Check if element is visible
 */
export function isElementVisible(element: HTMLElement): boolean {
  return !!(
    element.offsetWidth ||
    element.offsetHeight ||
    element.getClientRects().length
  )
}

/**
 * Get next/previous focusable element
 */
export function getNextFocusableElement(
  current: HTMLElement,
  direction: 'next' | 'previous' = 'next'
): HTMLElement | null {
  const focusable = getFocusableElements(document.body)
  const currentIndex = focusable.indexOf(current)

  if (currentIndex === -1) return null

  const nextIndex = direction === 'next' ? currentIndex + 1 : currentIndex - 1
  return focusable[nextIndex] || null
}

// Form Accessibility Utilities

/**
 * Validate form accessibility
 */
export interface FormAccessibilityIssue {
  element: HTMLElement
  issue: string
  severity: 'error' | 'warning'
}

export function validateFormAccessibility(form: HTMLFormElement): FormAccessibilityIssue[] {
  const issues: FormAccessibilityIssue[] = []

  // Check all inputs have labels
  const inputs = form.querySelectorAll('input, select, textarea')
  inputs.forEach((input) => {
    const hasLabel = input.hasAttribute('aria-label') ||
                    input.hasAttribute('aria-labelledby') ||
                    form.querySelector(`label[for="${input.id}"]`)

    if (!hasLabel) {
      issues.push({
        element: input as HTMLElement,
        issue: 'Form control missing label',
        severity: 'error',
      })
    }
  })

  // Check required fields have aria-required
  const requiredInputs = form.querySelectorAll('[required]')
  requiredInputs.forEach((input) => {
    if (!input.hasAttribute('aria-required')) {
      issues.push({
        element: input as HTMLElement,
        issue: 'Required field missing aria-required attribute',
        severity: 'warning',
      })
    }
  })

  // Check error messages are associated
  const errorMessages = form.querySelectorAll('[role="alert"], .error-message')
  errorMessages.forEach((error) => {
    const associatedInput = form.querySelector(`[aria-describedby="${error.id}"]`)
    if (!associatedInput && !error.id) {
      issues.push({
        element: error as HTMLElement,
        issue: 'Error message not associated with input',
        severity: 'error',
      })
    }
  })

  return issues
}

// Image Accessibility

/**
 * Check if image has proper alt text
 */
export function hasProperAltText(img: HTMLImageElement): boolean {
  // Decorative images should have empty alt
  if (img.hasAttribute('role') && img.getAttribute('role') === 'presentation') {
    return img.alt === ''
  }

  // Informative images need descriptive alt
  return img.hasAttribute('alt') && img.alt.trim().length > 0
}

// Heading Structure Utilities (WCAG 2.1 - 1.3.1 Info and Relationships)

/**
 * Validate heading hierarchy
 */
export function validateHeadingHierarchy(container: HTMLElement = document.body): string[] {
  const issues: string[] = []
  const headings = Array.from(container.querySelectorAll('h1, h2, h3, h4, h5, h6'))

  let previousLevel = 0

  headings.forEach((heading, index) => {
    const level = parseInt(heading.tagName[1])

    if (index === 0 && level !== 1) {
      issues.push(`First heading should be h1, found ${heading.tagName}`)
    }

    if (level > previousLevel + 1) {
      issues.push(`Heading level skipped: ${heading.tagName} after h${previousLevel}`)
    }

    previousLevel = level
  })

  return issues
}

// Motion and Animation (WCAG 2.1 - 2.3.1 Three Flashes or Below Threshold)

/**
 * Check if user prefers reduced motion
 */
export function prefersReducedMotion(): boolean {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

/**
 * Get safe animation duration based on user preference
 */
export function getSafeAnimationDuration(normalDuration: number): number {
  return prefersReducedMotion() ? 0 : normalDuration
}

// Touch Target Size (WCAG 2.1 - 2.5.5 Target Size)

/**
 * Check if element meets minimum touch target size (44x44 CSS pixels)
 */
export function meetsMinimumTouchTargetSize(element: HTMLElement): boolean {
  const rect = element.getBoundingClientRect()
  return rect.width >= 44 && rect.height >= 44
}

// Testing Utilities

/**
 * Comprehensive accessibility audit
 */
export interface AccessibilityAuditResult {
  passed: boolean
  errors: Array<{ message: string; element?: HTMLElement }>
  warnings: Array<{ message: string; element?: HTMLElement }>
}

export function auditAccessibility(container: HTMLElement = document.body): AccessibilityAuditResult {
  const errors: Array<{ message: string; element?: HTMLElement }> = []
  const warnings: Array<{ message: string; element?: HTMLElement }> = []

  // Check heading hierarchy
  const headingIssues = validateHeadingHierarchy(container)
  headingIssues.forEach((issue) => errors.push({ message: issue }))

  // Check images
  const images = container.querySelectorAll('img')
  images.forEach((img) => {
    if (!hasProperAltText(img)) {
      errors.push({
        message: 'Image missing or has improper alt text',
        element: img as HTMLElement,
      })
    }
  })

  // Check forms
  const forms = container.querySelectorAll('form')
  forms.forEach((form) => {
    const formIssues = validateFormAccessibility(form as HTMLFormElement)
    formIssues.forEach((issue) => {
      if (issue.severity === 'error') {
        errors.push({ message: issue.issue, element: issue.element })
      } else {
        warnings.push({ message: issue.issue, element: issue.element })
      }
    })
  })

  // Check interactive elements have sufficient size
  const interactive = container.querySelectorAll('button, a, input, select')
  interactive.forEach((element) => {
    if (!meetsMinimumTouchTargetSize(element as HTMLElement)) {
      warnings.push({
        message: 'Interactive element below minimum touch target size (44x44)',
        element: element as HTMLElement,
      })
    }
  })

  // Check for proper ARIA usage
  const ariaElements = container.querySelectorAll('[role]')
  ariaElements.forEach((element) => {
    const role = element.getAttribute('role')
    if (role && !isValidAriaRole(role)) {
      errors.push({
        message: `Invalid ARIA role: ${role}`,
        element: element as HTMLElement,
      })
    }
  })

  return {
    passed: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Valid ARIA roles (subset of most common roles)
 */
const VALID_ARIA_ROLES = [
  'alert',
  'alertdialog',
  'button',
  'checkbox',
  'dialog',
  'link',
  'log',
  'main',
  'navigation',
  'region',
  'search',
  'status',
  'tab',
  'tabpanel',
  'textbox',
  'timer',
  'tooltip',
  'banner',
  'complementary',
  'contentinfo',
  'form',
  'menu',
  'menubar',
  'menuitem',
  'menuitemcheckbox',
  'menuitemradio',
  'progressbar',
  'radio',
  'radiogroup',
  'slider',
  'spinbutton',
  'switch',
  'tree',
  'treeitem',
]

function isValidAriaRole(role: string): boolean {
  return VALID_ARIA_ROLES.includes(role)
}

// Export utilities for testing
export const accessibility = {
  getContrastRatio,
  meetsWCAG_AA,
  meetsWCAG_AAA,
  getFocusableElements,
  trapFocus,
  createFocusTrap,
  announceToScreenReader,
  generateAriaId,
  createFocusRestore,
  validateFormAccessibility,
  hasProperAltText,
  validateHeadingHierarchy,
  prefersReducedMotion,
  getSafeAnimationDuration,
  meetsMinimumTouchTargetSize,
  auditAccessibility,
}
