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

    // Mock localStorage with actual storage
    const localStorageStore: Record<string, string> = {};
    const localStorageMock = {
        getItem: vi.fn((key: string) => localStorageStore[key] || null),
        setItem: vi.fn((key: string, value: string) => { localStorageStore[key] = value; }),
        removeItem: vi.fn((key: string) => { delete localStorageStore[key]; }),
        clear: vi.fn(() => { for (const key in localStorageStore) delete localStorageStore[key]; }),
        get length() { return Object.keys(localStorageStore).length; },
        key: vi.fn((index: number) => Object.keys(localStorageStore)[index] || null),
    };
    global.localStorage = localStorageMock as Storage;

    // Mock sessionStorage with actual storage
    const sessionStorageStore: Record<string, string> = {};
    const sessionStorageMock = {
        getItem: vi.fn((key: string) => sessionStorageStore[key] || null),
        setItem: vi.fn((key: string, value: string) => { sessionStorageStore[key] = value; }),
        removeItem: vi.fn((key: string) => { delete sessionStorageStore[key]; }),
        clear: vi.fn(() => { for (const key in sessionStorageStore) delete sessionStorageStore[key]; }),
        get length() { return Object.keys(sessionStorageStore).length; },
        key: vi.fn((index: number) => Object.keys(sessionStorageStore)[index] || null),
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
                update: vi.fn().mockResolvedValue(undefined),
            }),
            ready: Promise.resolve({
                installing: null,
                waiting: null,
                active: { state: 'activated' },
                update: vi.fn().mockResolvedValue(undefined),
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

    // Mock fetch for network tests - let individual tests configure it
    // Provide a factory function for creating proper Response objects
    (global as any).createMockResponse = (overrides: Partial<Response> = {}) => ({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        redirected: false,
        type: 'basic' as ResponseType,
        url: '',
        clone: vi.fn(),
        body: null,
        bodyUsed: false,
        arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(0)),
        blob: vi.fn().mockResolvedValue(new Blob()),
        formData: vi.fn().mockResolvedValue(new FormData()),
        json: vi.fn().mockResolvedValue({}),
        text: vi.fn().mockResolvedValue(''),
        ...overrides,
    });

    // Set a minimal default implementation for fetch
    global.fetch = vi.fn((url: string, options?: RequestInit) =>
        Promise.resolve((global as any).createMockResponse())
    ) as any;

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

    // Mock HTMLCanvasElement.prototype.getContext for Canvas tests
    HTMLCanvasElement.prototype.getContext = vi.fn((contextType: string) => {
        if (contextType === '2d') {
            return {
                fillStyle: '',
                strokeStyle: '',
                lineWidth: 1,
                fillRect: vi.fn(),
                clearRect: vi.fn(),
                strokeRect: vi.fn(),
                beginPath: vi.fn(),
                closePath: vi.fn(),
                moveTo: vi.fn(),
                lineTo: vi.fn(),
                arc: vi.fn(),
                stroke: vi.fn(),
                fill: vi.fn(),
                save: vi.fn(),
                restore: vi.fn(),
                scale: vi.fn(),
                rotate: vi.fn(),
                translate: vi.fn(),
                transform: vi.fn(),
                setTransform: vi.fn(),
                drawImage: vi.fn(),
                createImageData: vi.fn(),
                getImageData: vi.fn(),
                putImageData: vi.fn(),
                canvas: {} as HTMLCanvasElement,
            } as any;
        }
        return null;
    }) as any;

    // Mock URL.createObjectURL and URL.revokeObjectURL
    global.URL.createObjectURL = vi.fn((blob: Blob) => `blob:${Math.random().toString(36)}`);
    global.URL.revokeObjectURL = vi.fn();
});

// Cleanup after each test
afterEach(() => {
    cleanup();
    // Clear call history but preserve mock implementations
    vi.clearAllTimers();
    // Clear storage (these call the mocked clear functions)
    if (typeof localStorage !== 'undefined') {
        localStorage.clear();
    }
    if (typeof sessionStorage !== 'undefined') {
        sessionStorage.clear();
    }
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
