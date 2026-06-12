/**
 * Accessibility Tests - WCAG 2.1 Level AA Compliance
 *
 * These tests ensure our application meets WCAG 2.1 Level AA standards
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  getContrastRatio,
  meetsWCAG_AA,
  meetsWCAG_AAA,
  getFocusableElements,
  trapFocus,
  announceToScreenReader,
  validateFormAccessibility,
  hasProperAltText,
  validateHeadingHierarchy,
  prefersReducedMotion,
  meetsMinimumTouchTargetSize,
  auditAccessibility,
} from '../lib/accessibility'

describe('Color Contrast (WCAG 2.1 - 1.4.3)', () => {
  it('should calculate contrast ratio correctly', () => {
    // White on black should have maximum contrast
    const whiteOnBlack = getContrastRatio('#FFFFFF', '#000000')
    expect(whiteOnBlack).toBeCloseTo(21, 1)

    // Same colors should have minimum contrast
    const blackOnBlack = getContrastRatio('#000000', '#000000')
    expect(blackOnBlack).toBe(1)
  })

  it('should validate WCAG AA compliance for normal text', () => {
    // These combinations should pass WCAG AA (4.5:1 minimum)
    expect(meetsWCAG_AA('#000000', '#FFFFFF')).toBe(true) // 21:1
    expect(meetsWCAG_AA('#595959', '#FFFFFF')).toBe(true) // 7.00:1
    expect(meetsWCAG_AA('#767676', '#FFFFFF')).toBe(true) // 4.54:1 - passes AA

    // Test with color that actually fails AA (below 4.5:1)
    expect(meetsWCAG_AA('#959595', '#FFFFFF')).toBe(false) // 2.99:1 - fails AA
  })

  it('should validate WCAG AA compliance for large text', () => {
    // Large text has lower requirements (3:1)
    expect(meetsWCAG_AA('#767676', '#FFFFFF', 18)).toBe(true) // 4.47:1
    expect(meetsWCAG_AA('#949494', '#FFFFFF', 18)).toBe(true) // 3.04:1
    expect(meetsWCAG_AA('#949494', '#FFFFFF', 18, true)).toBe(true) // 14pt bold
  })

  it('should validate WCAG AAA compliance', () => {
    // AAA requires 7:1 for normal text
    expect(meetsWCAG_AAA('#000000', '#FFFFFF')).toBe(true) // 21:1
    expect(meetsWCAG_AAA('#595959', '#FFFFFF')).toBe(true) // 7.00:1 - passes AAA

    // Test with colors below 7:1 ratio - these should fail AAA
    expect(meetsWCAG_AAA('#767676', '#FFFFFF')).toBe(false) // 4.54:1 - fails AAA (needs 7:1)
  })
})

describe('Keyboard Navigation (WCAG 2.1 - 2.1)', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  it('should find all focusable elements', () => {
    container.innerHTML = `
      <button>Button 1</button>
      <a href="#">Link</a>
      <input type="text" />
      <button disabled>Disabled</button>
      <div tabindex="0">Focusable div</div>
      <div tabindex="-1">Not focusable</div>
    `

    const focusable = getFocusableElements(container)
    expect(focusable).toHaveLength(4) // button, a, input, div with tabindex="0"
  })

  it('should trap focus within container', () => {
    container.innerHTML = `
      <button id="first">First</button>
      <button id="middle">Middle</button>
      <button id="last">Last</button>
    `

    const firstButton = container.querySelector('#first') as HTMLElement
    const lastButton = container.querySelector('#last') as HTMLElement

    // Mock Tab key event on last button
    lastButton.focus()
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
    Object.defineProperty(tabEvent, 'preventDefault', { value: vi.fn() })

    trapFocus(container, tabEvent)

    // Should have prevented default and focused first element
    expect((tabEvent as any).preventDefault).toHaveBeenCalled()
  })

  it('should handle Shift+Tab for reverse navigation', () => {
    container.innerHTML = `
      <button id="first">First</button>
      <button id="last">Last</button>
    `

    const firstButton = container.querySelector('#first') as HTMLElement
    firstButton.focus()

    const shiftTabEvent = new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
    })
    Object.defineProperty(shiftTabEvent, 'preventDefault', { value: vi.fn() })

    trapFocus(container, shiftTabEvent)
    expect((shiftTabEvent as any).preventDefault).toHaveBeenCalled()
  })
})

describe('Screen Reader Support (WCAG 2.1 - 4.1.2)', () => {
  afterEach(() => {
    // Clean up any announcement elements
    const announcements = document.querySelectorAll('[role="status"]')
    announcements.forEach((el) => el.remove())
  })

  it('should create screen reader announcement', (done) => {
    announceToScreenReader('Test announcement', 'polite')

    // Check that announcement element was created
    const announcement = document.querySelector('[role="status"]')
    expect(announcement).toBeTruthy()
    expect(announcement?.getAttribute('aria-live')).toBe('polite')
    expect(announcement?.textContent).toBe('Test announcement')

    // Should be removed after timeout
    setTimeout(() => {
      const announcement = document.querySelector('[role="status"]')
      expect(announcement).toBeFalsy()
      done()
    }, 1100)
  })

  it('should create assertive announcements', () => {
    announceToScreenReader('Important message', 'assertive')

    const announcement = document.querySelector('[aria-live="assertive"]')
    expect(announcement).toBeTruthy()
    expect(announcement?.textContent).toBe('Important message')
  })
})

describe('Form Accessibility (WCAG 2.1 - 3.3)', () => {
  let form: HTMLFormElement

  beforeEach(() => {
    form = document.createElement('form')
    document.body.appendChild(form)
  })

  afterEach(() => {
    document.body.removeChild(form)
  })

  it('should detect inputs without labels', () => {
    form.innerHTML = `
      <input type="text" id="unlabeled" />
      <input type="text" id="labeled" />
      <label for="labeled">Label</label>
    `

    const issues = validateFormAccessibility(form)
    expect(issues).toHaveLength(1)
    expect(issues[0].issue).toContain('missing label')
  })

  it('should accept aria-label as valid labeling', () => {
    form.innerHTML = `
      <input type="text" aria-label="Username" />
    `

    const issues = validateFormAccessibility(form)
    const labelIssues = issues.filter((i) => i.issue.includes('label'))
    expect(labelIssues).toHaveLength(0)
  })

  it('should warn about missing aria-required on required fields', () => {
    form.innerHTML = `
      <label for="required-field">Required</label>
      <input type="text" id="required-field" required />
    `

    const issues = validateFormAccessibility(form)
    const requiredIssues = issues.filter((i) => i.issue.includes('aria-required'))
    expect(requiredIssues).toHaveLength(1)
    expect(requiredIssues[0].severity).toBe('warning')
  })

  it('should detect unassociated error messages', () => {
    form.innerHTML = `
      <input type="text" id="field" />
      <div role="alert">Error message</div>
    `

    const issues = validateFormAccessibility(form)
    const errorIssues = issues.filter((i) => i.issue.includes('Error message'))
    expect(errorIssues).toHaveLength(1)
  })
})

describe('Image Accessibility (WCAG 2.1 - 1.1.1)', () => {
  it('should validate informative images have alt text', () => {
    const img = document.createElement('img')
    img.src = 'test.jpg'
    img.alt = 'Description of image'

    expect(hasProperAltText(img)).toBe(true)
  })

  it('should require empty alt for decorative images', () => {
    const img = document.createElement('img')
    img.src = 'decorative.jpg'
    img.setAttribute('role', 'presentation')
    img.alt = ''

    expect(hasProperAltText(img)).toBe(true)
  })

  it('should fail images without alt attribute', () => {
    const img = document.createElement('img')
    img.src = 'test.jpg'

    expect(hasProperAltText(img)).toBe(false)
  })

  it('should fail decorative images with non-empty alt', () => {
    const img = document.createElement('img')
    img.src = 'decorative.jpg'
    img.setAttribute('role', 'presentation')
    img.alt = 'Should be empty'

    expect(hasProperAltText(img)).toBe(false)
  })
})

describe('Heading Hierarchy (WCAG 2.1 - 1.3.1)', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  it('should validate correct heading hierarchy', () => {
    container.innerHTML = `
      <h1>Main Title</h1>
      <h2>Section</h2>
      <h3>Subsection</h3>
      <h2>Another Section</h2>
    `

    const issues = validateHeadingHierarchy(container)
    expect(issues).toHaveLength(0)
  })

  it('should detect skipped heading levels', () => {
    container.innerHTML = `
      <h1>Main Title</h1>
      <h3>Skipped h2</h3>
    `

    const issues = validateHeadingHierarchy(container)
    expect(issues).toHaveLength(1)
    expect(issues[0]).toContain('skipped')
  })

  it('should require h1 as first heading', () => {
    container.innerHTML = `
      <h2>Wrong first heading</h2>
    `

    const issues = validateHeadingHierarchy(container)
    // Should report both: wrong first heading AND skipped level (h0 to h2)
    expect(issues).toHaveLength(2)
    expect(issues[0]).toContain('First heading should be h1')
    expect(issues[1]).toContain('skipped')
  })
})

describe('Motion and Animation (WCAG 2.1 - 2.3)', () => {
  it('should detect reduced motion preference', () => {
    // Mock matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query) => ({
        matches: query === '(prefers-reduced-motion: reduce)',
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })

    expect(prefersReducedMotion()).toBe(true)
  })
})

describe('Touch Target Size (WCAG 2.1 - 2.5.5)', () => {
  it('should validate minimum touch target size', () => {
    const button = document.createElement('button')
    button.style.width = '44px'
    button.style.height = '44px'
    document.body.appendChild(button)

    // Mock getBoundingClientRect
    button.getBoundingClientRect = vi.fn().mockReturnValue({
      width: 44,
      height: 44,
      top: 0,
      left: 0,
      right: 44,
      bottom: 44,
      x: 0,
      y: 0,
      toJSON: () => {},
    })

    expect(meetsMinimumTouchTargetSize(button)).toBe(true)

    document.body.removeChild(button)
  })

  it('should fail elements below minimum size', () => {
    const button = document.createElement('button')
    document.body.appendChild(button)

    button.getBoundingClientRect = vi.fn().mockReturnValue({
      width: 30,
      height: 30,
      top: 0,
      left: 0,
      right: 30,
      bottom: 30,
      x: 0,
      y: 0,
      toJSON: () => {},
    })

    expect(meetsMinimumTouchTargetSize(button)).toBe(false)

    document.body.removeChild(button)
  })
})

describe('Comprehensive Accessibility Audit', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  it('should pass for accessible content', () => {
    container.innerHTML = `
      <h1>Title</h1>
      <img src="test.jpg" alt="Description" />
      <form>
        <label for="input1">Label</label>
        <input type="text" id="input1" />
      </form>
    `

    const result = auditAccessibility(container)
    expect(result.passed).toBe(true)
    expect(result.errors).toHaveLength(0)
  })

  it('should detect multiple accessibility issues', () => {
    container.innerHTML = `
      <h2>Wrong first heading</h2>
      <img src="test.jpg" />
      <form>
        <input type="text" />
      </form>
      <div role="invalid-role">Content</div>
    `

    const result = auditAccessibility(container)
    expect(result.passed).toBe(false)
    expect(result.errors.length).toBeGreaterThan(0)
  })

  it('should categorize errors and warnings correctly', () => {
    container.innerHTML = `
      <h1>Title</h1>
      <form>
        <label for="field">Field</label>
        <input type="text" id="field" required />
      </form>
    `

    const result = auditAccessibility(container)

    // Should have warning about missing aria-required
    expect(result.warnings.length).toBeGreaterThan(0)
  })
})

// Integration test examples
describe('Component Accessibility Integration Tests', () => {
  it('should demonstrate testing a button component', () => {
    const button = document.createElement('button')
    button.textContent = 'Click me'
    button.setAttribute('aria-label', 'Close dialog')
    button.style.width = '48px'
    button.style.height = '48px'

    document.body.appendChild(button)

    // Should be focusable
    const focusable = getFocusableElements(document.body)
    expect(focusable).toContain(button)

    // Should have adequate size
    button.getBoundingClientRect = vi.fn().mockReturnValue({
      width: 48,
      height: 48,
      top: 0,
      left: 0,
      right: 48,
      bottom: 48,
      x: 0,
      y: 0,
      toJSON: () => {},
    })
    expect(meetsMinimumTouchTargetSize(button)).toBe(true)

    // Should have accessible name
    expect(button.getAttribute('aria-label')).toBeTruthy()

    document.body.removeChild(button)
  })
})
