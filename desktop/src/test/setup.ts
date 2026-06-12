import { vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'

// Cleanup after each test
// Note: afterEach is globally available due to globals: true in vitest.config.ts
if (typeof afterEach !== 'undefined') {
  afterEach(() => {
    cleanup()
  })
}

// Mock Tauri API
(globalThis as any).window = (globalThis as any).window || ({} as any)
;(window as any).__TAURI__ = {
  invoke: vi.fn(),
  notification: {
    sendNotification: vi.fn(),
  },
  window: {
    appWindow: {
      minimize: vi.fn(),
      close: vi.fn(),
      listen: vi.fn(),
    },
  },
}

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})
