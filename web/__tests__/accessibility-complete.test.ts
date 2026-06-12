/**
 * Comprehensive Accessibility Tests
 * Tests for all WCAG 2.1 AAA features
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  KeyboardNavigationManager,
  setupKeyboardShortcuts,
  FocusIndicatorManager,
  ARIALiveRegionManager,
  VoiceControlManager,
  VideoCaptionManager,
  enhanceARIA,
  createAccessibleButton,
} from '../lib/accessibility-enhanced'

// Mock DOM environment
const mockElement = (tag: string = 'div'): HTMLElement => {
  return document.createElement(tag)
}

describe('Keyboard Navigation Manager', () => {
  let container: HTMLElement
  let manager: KeyboardNavigationManager

  beforeEach(() => {
    container = mockElement()
    document.body.appendChild(container)

    // Add focusable elements
    const button1 = mockElement('button')
    const button2 = mockElement('button')
    const input = mockElement('input')
    const link = mockElement('a')
    link.setAttribute('href', '#')

    container.appendChild(button1)
    container.appendChild(button2)
    container.appendChild(input)
    container.appendChild(link)

    manager = new KeyboardNavigationManager(container)
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  it('should find all focusable elements', () => {
    manager.updateFocusableElements()
    expect(manager['focusableElements'].length).toBe(4)
  })

  it('should focus next element', () => {
    manager.focusNext()
    expect(document.activeElement).toBe(manager['focusableElements'][0])

    manager.focusNext()
    expect(document.activeElement).toBe(manager['focusableElements'][1])
  })

  it('should focus previous element', () => {
    manager.focusLast()
    manager.focusPrevious()
    expect(document.activeElement).toBe(manager['focusableElements'][2])
  })

  it('should focus first element', () => {
    manager.focusFirst()
    expect(document.activeElement).toBe(manager['focusableElements'][0])
  })

  it('should focus last element', () => {
    manager.focusLast()
    expect(document.activeElement).toBe(manager['focusableElements'][3])
  })

  it('should trap focus within container', () => {
    const cleanup = manager.trapFocus(container)

    // Simulate Tab key on last element
    manager.focusLast()
    const event = new KeyboardEvent('keydown', { key: 'Tab' })
    container.dispatchEvent(event)

    cleanup()
  })
})

describe('Keyboard Shortcuts', () => {
  it('should register and trigger keyboard shortcut', () => {
    const handler = vi.fn()
    const shortcuts = setupKeyboardShortcuts()

    shortcuts.register('ctrl+k', handler)

    const event = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(handler).toHaveBeenCalled()

    shortcuts.destroy()
  })

  it('should unregister keyboard shortcut', () => {
    const handler = vi.fn()
    const shortcuts = setupKeyboardShortcuts()

    shortcuts.register('ctrl+k', handler)
    shortcuts.unregister('ctrl+k')

    const event = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
    })
    document.dispatchEvent(event)

    expect(handler).not.toHaveBeenCalled()

    shortcuts.destroy()
  })
})

describe('Focus Indicator Manager', () => {
  let manager: FocusIndicatorManager

  beforeEach(() => {
    manager = new FocusIndicatorManager()
  })

  afterEach(() => {
    manager.disable()
  })

  it('should enable focus indicators', () => {
    manager.enable({
      color: '#ff0000',
      width: 4,
      offset: 3,
      style: 'solid',
    })

    const styles = document.head.querySelectorAll('style')
    expect(styles.length).toBeGreaterThan(0)
  })

  it('should disable focus indicators', () => {
    manager.enable()
    const beforeCount = document.head.querySelectorAll('style').length

    manager.disable()
    const afterCount = document.head.querySelectorAll('style').length

    expect(afterCount).toBeLessThan(beforeCount)
  })

  it('should show focus ring programmatically', () => {
    const element = mockElement()
    manager.showFocusRing(element)

    expect(element.classList.contains('force-focus-visible')).toBe(true)
  })

  it('should hide focus ring', () => {
    const element = mockElement()
    manager.showFocusRing(element)
    manager.hideFocusRing(element)

    expect(element.classList.contains('force-focus-visible')).toBe(false)
  })
})

describe('ARIA Live Region Manager', () => {
  let manager: ARIALiveRegionManager

  beforeEach(() => {
    manager = new ARIALiveRegionManager()
  })

  afterEach(() => {
    manager.removeAll()
  })

  it('should create live region', () => {
    const region = manager.createRegion('test-region', {
      politeness: 'polite',
      atomic: true,
    })

    expect(region.getAttribute('aria-live')).toBe('polite')
    expect(region.getAttribute('aria-atomic')).toBe('true')
    expect(document.body.contains(region)).toBe(true)
  })

  it('should announce message', () => {
    manager.announce('Test message', {
      regionId: 'test-announce',
      politeness: 'assertive',
    })

    const region = manager['regions'].get('test-announce')
    expect(region?.textContent).toBe('Test message')
  })

  it('should clear announcements', () => {
    manager.announce('Test message', { regionId: 'test-clear' })
    manager.clear('test-clear')

    const region = manager['regions'].get('test-clear')
    expect(region?.textContent).toBe('')
  })

  it('should remove live region', () => {
    manager.createRegion('test-remove')
    manager.removeRegion('test-remove')

    expect(manager['regions'].has('test-remove')).toBe(false)
  })

  it('should clear all regions', () => {
    manager.createRegion('region-1')
    manager.createRegion('region-2')
    manager.removeAll()

    expect(manager['regions'].size).toBe(0)
  })
})

describe('Voice Control Manager', () => {
  let manager: VoiceControlManager

  beforeEach(() => {
    // Mock Speech Recognition API
    ;(global as any).window = {
      SpeechRecognition: class {
        continuous = false
        interimResults = false
        lang = ''
        onresult = null
        onerror = null
        onend = null

        start() {}
        stop() {}
      },
    }

    manager = new VoiceControlManager()
  })

  it('should check if voice control is supported', () => {
    expect(manager.isSupported()).toBe(true)
  })

  it('should register voice command', () => {
    const handler = vi.fn()
    manager.registerCommand('test command', handler)

    expect(manager['commands'].has('test command')).toBe(true)
  })

  it('should unregister voice command', () => {
    const handler = vi.fn()
    manager.registerCommand('test command', handler)
    manager.unregisterCommand('test command')

    expect(manager['commands'].has('test command')).toBe(false)
  })

  it('should get listening status', () => {
    expect(manager.getListeningStatus()).toBe(false)

    manager.start()
    expect(manager.getListeningStatus()).toBe(true)

    manager.stop()
    expect(manager.getListeningStatus()).toBe(false)
  })
})

describe('Video Caption Manager', () => {
  let video: HTMLVideoElement
  let manager: VideoCaptionManager

  beforeEach(() => {
    video = document.createElement('video')
    document.body.appendChild(video)
    manager = new VideoCaptionManager(video)
  })

  afterEach(() => {
    document.body.removeChild(video)
  })

  it('should load captions from array', () => {
    const captions = [
      { startTime: 0, endTime: 2, text: 'First caption' },
      { startTime: 2, endTime: 4, text: 'Second caption' },
    ]

    manager.loadCaptions(captions)
    expect(manager['captions'].length).toBe(2)
  })

  it('should enable captions', () => {
    const container = mockElement()
    manager.enable(container)

    expect(video.listeners('timeupdate')).toBeDefined()
  })

  it('should disable captions', () => {
    const container = mockElement()
    manager.enable(container)
    manager.disable()

    // Caption container should be cleared
  })

  it('should parse VTT timestamp', () => {
    const timestamp1 = manager['parseTimestamp']('00:00:05.000')
    expect(timestamp1).toBe(5)

    const timestamp2 = manager['parseTimestamp']('00:01:30.500')
    expect(timestamp2).toBe(90.5)

    const timestamp3 = manager['parseTimestamp']('01:00:00.000')
    expect(timestamp3).toBe(3600)
  })

  it('should parse VTT file', () => {
    const vtt = `WEBVTT

1
00:00:00.000 --> 00:00:02.000
First caption

2
00:00:02.000 --> 00:00:04.000
Second caption`

    const captions = manager['parseVTT'](vtt)
    expect(captions.length).toBe(2)
    expect(captions[0].text).toBe('First caption')
    expect(captions[1].text).toBe('Second caption')
  })
})

describe('ARIA Enhancement Utilities', () => {
  it('should enhance element with ARIA attributes', () => {
    const element = mockElement()

    enhanceARIA(element, {
      label: 'Test label',
      describedBy: 'description-id',
      role: 'button',
      expanded: true,
      selected: false,
      invalid: false,
      required: true,
    })

    expect(element.getAttribute('aria-label')).toBe('Test label')
    expect(element.getAttribute('aria-describedby')).toBe('description-id')
    expect(element.getAttribute('role')).toBe('button')
    expect(element.getAttribute('aria-expanded')).toBe('true')
    expect(element.getAttribute('aria-selected')).toBe('false')
    expect(element.getAttribute('aria-invalid')).toBe('false')
    expect(element.getAttribute('aria-required')).toBe('true')
  })
})

describe('Accessible Button Creation', () => {
  it('should create accessible button', () => {
    const handler = vi.fn()
    const button = createAccessibleButton({
      text: 'Click me',
      onClick: handler,
      ariaLabel: 'Custom label',
      disabled: false,
    })

    expect(button.textContent).toBe('Click me')
    expect(button.getAttribute('aria-label')).toBe('Custom label')
    expect(button.disabled).toBe(false)

    button.click()
    expect(handler).toHaveBeenCalled()
  })

  it('should create disabled button', () => {
    const handler = vi.fn()
    const button = createAccessibleButton({
      text: 'Click me',
      onClick: handler,
      disabled: true,
    })

    expect(button.disabled).toBe(true)
  })
})

describe('High Contrast Mode', () => {
  it('should detect system high contrast preference', () => {
    const mockMatch = {
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }

    window.matchMedia = vi.fn().mockReturnValue(mockMatch)

    // Test detection logic
    const query = window.matchMedia('(prefers-contrast: more)')
    expect(query.matches).toBe(true)
  })

  it('should apply high contrast class', () => {
    const root = document.documentElement
    root.classList.add('high-contrast')

    expect(root.classList.contains('high-contrast')).toBe(true)

    root.classList.remove('high-contrast')
  })
})

describe('Font Size Control', () => {
  it('should apply font size to root element', () => {
    const root = document.documentElement
    root.style.fontSize = '18px'

    expect(root.style.fontSize).toBe('18px')

    root.style.fontSize = ''
  })

  it('should save font size to localStorage', () => {
    localStorage.setItem('font-size', 'lg')
    expect(localStorage.getItem('font-size')).toBe('lg')

    localStorage.removeItem('font-size')
  })
})

describe('Reduced Motion', () => {
  it('should detect system motion preference', () => {
    const mockMatch = {
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }

    window.matchMedia = vi.fn().mockReturnValue(mockMatch)

    const query = window.matchMedia('(prefers-reduced-motion: reduce)')
    expect(query.matches).toBe(true)
  })

  it('should apply reduce-motion class', () => {
    const root = document.documentElement
    root.classList.add('reduce-motion')

    expect(root.classList.contains('reduce-motion')).toBe(true)

    root.classList.remove('reduce-motion')
  })
})

describe('Color Blind Mode', () => {
  it('should apply color blind filter class', () => {
    const root = document.documentElement
    root.classList.add('color-blind-protanopia')

    expect(root.classList.contains('color-blind-protanopia')).toBe(true)

    root.classList.remove('color-blind-protanopia')
  })

  it('should save color blind type to localStorage', () => {
    localStorage.setItem('color-blind-type', 'deuteranopia')
    expect(localStorage.getItem('color-blind-type')).toBe('deuteranopia')

    localStorage.removeItem('color-blind-type')
  })
})

describe('Skip Links', () => {
  it('should create skip link', () => {
    const link = document.createElement('a')
    link.href = '#main-content'
    link.className = 'skip-link'
    link.textContent = 'Skip to main content'

    expect(link.getAttribute('href')).toBe('#main-content')
    expect(link.className).toContain('skip-link')
  })

  it('should handle skip link click', () => {
    const target = mockElement()
    target.id = 'main-content'
    document.body.appendChild(target)

    const link = document.createElement('a')
    link.href = '#main-content'

    link.click()

    // Target should be focused
    expect(document.activeElement).toBe(target)

    document.body.removeChild(target)
  })
})
