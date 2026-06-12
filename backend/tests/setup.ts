import { beforeAll, afterAll, afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';

// Mock environment variables
process.env.NODE_ENV = 'test';
process.env.TEST_DATABASE_URL = 'postgres://test:test@localhost:5432/test_db';

// Global test setup
beforeAll(() => {
    // Setup code that runs before all tests
    console.log('Starting test suite...');

    // Mock console methods to reduce noise in tests
    global.console = {
        ...console,
        log: vi.fn(),
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        // Keep error for debugging
        error: console.error,
    };

    // Mock window.matchMedia for responsive design tests
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
    });

    // Mock IntersectionObserver for lazy loading tests
    global.IntersectionObserver = class IntersectionObserver {
        constructor() {}
        disconnect() {}
        observe() {}
        takeRecords() {
            return [];
        }
        unobserve() {}
    } as any;

    // Mock ResizeObserver for responsive component tests
    global.ResizeObserver = class ResizeObserver {
        constructor() {}
        disconnect() {}
        observe() {}
        unobserve() {}
    } as any;

    // Mock localStorage
    const localStorageMock = {
        getItem: vi.fn(),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn(),
        length: 0,
        key: vi.fn(),
    };
    global.localStorage = localStorageMock as Storage;

    // Mock sessionStorage
    const sessionStorageMock = {
        getItem: vi.fn(),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn(),
        length: 0,
        key: vi.fn(),
    };
    global.sessionStorage = sessionStorageMock as Storage;

    // Mock navigator.clipboard
    Object.assign(navigator, {
        clipboard: {
            writeText: vi.fn().mockResolvedValue(undefined),
            readText: vi.fn().mockResolvedValue(''),
        },
    });

    // Mock navigator.geolocation
    Object.assign(navigator, {
        geolocation: {
            getCurrentPosition: vi.fn(),
            watchPosition: vi.fn(),
            clearWatch: vi.fn(),
        },
    });

    // Mock navigator.serviceWorker
    Object.defineProperty(navigator, 'serviceWorker', {
        value: {
            register: vi.fn().mockResolvedValue({
                installing: null,
                waiting: null,
                active: { state: 'activated' },
                addEventListener: vi.fn(),
                removeEventListener: vi.fn(),
            }),
            ready: Promise.resolve({
                installing: null,
                waiting: null,
                active: { state: 'activated' },
            }),
            controller: null,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        },
        writable: true,
    });

    // Mock Notification API
    (global as any).Notification = class Notification {
        static permission = 'default';
        static requestPermission = vi.fn().mockResolvedValue('granted');

        constructor(public title: string, public options?: NotificationOptions) {}

        close() {}
        addEventListener() {}
        removeEventListener() {}
    };

    // Mock fetch for network tests
    global.fetch = vi.fn();

    // Mock performance API
    global.performance = {
        ...performance,
        now: vi.fn(() => Date.now()),
        mark: vi.fn(),
        measure: vi.fn(),
        clearMarks: vi.fn(),
        clearMeasures: vi.fn(),
        getEntriesByName: vi.fn(() => []),
        getEntriesByType: vi.fn(() => []),
    } as any;
});

// Cleanup after each test
afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    localStorage.clear();
    sessionStorage.clear();
});

// Global teardown
afterAll(() => {
    console.log('Test suite completed.');
    vi.restoreAllMocks();
});

// Custom matchers
expect.extend({
    toBeWithinRange(received: number, floor: number, ceiling: number) {
        const pass = received >= floor && received <= ceiling;
        if (pass) {
            return {
                message: () =>
                    `expected ${received} not to be within range ${floor} - ${ceiling}`,
                pass: true,
            };
        } else {
            return {
                message: () =>
                    `expected ${received} to be within range ${floor} - ${ceiling}`,
                pass: false,
            };
        }
    },
});
