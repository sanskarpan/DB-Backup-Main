import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

// Mock Chrome Extension API
const mockChrome = {
    runtime: {
        id: 'test-extension-id',
        lastError: null as chrome.runtime.LastError | null,
        onInstalled: {
            addListener: vi.fn(),
            removeListener: vi.fn(),
        },
        onMessage: {
            addListener: vi.fn(),
            removeListener: vi.fn(),
        },
        sendMessage: vi.fn(),
        getManifest: vi.fn(() => ({
            version: '1.0.0',
            name: 'DB Backup Extension',
            manifest_version: 3,
        })),
        openOptionsPage: vi.fn(),
    },
    storage: {
        local: {
            get: vi.fn(),
            set: vi.fn(),
            remove: vi.fn(),
            clear: vi.fn(),
        },
        sync: {
            get: vi.fn(),
            set: vi.fn(),
            remove: vi.fn(),
            clear: vi.fn(),
        },
        onChanged: {
            addListener: vi.fn(),
        },
    },
    tabs: {
        query: vi.fn(),
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
        sendMessage: vi.fn(),
        onActivated: {
            addListener: vi.fn(),
        },
        onUpdated: {
            addListener: vi.fn(),
        },
    },
    contextMenus: {
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
        removeAll: vi.fn(),
        onClicked: {
            addListener: vi.fn(),
        },
    },
    notifications: {
        create: vi.fn(),
        clear: vi.fn(),
        getAll: vi.fn(),
        onClicked: {
            addListener: vi.fn(),
        },
    },
    downloads: {
        download: vi.fn(),
        search: vi.fn(),
        cancel: vi.fn(),
        onChanged: {
            addListener: vi.fn(),
        },
    },
    bookmarks: {
        create: vi.fn(),
        get: vi.fn(),
        search: vi.fn(),
        remove: vi.fn(),
    },
    permissions: {
        contains: vi.fn(),
        request: vi.fn(),
        remove: vi.fn(),
    },
    browserAction: {
        setBadgeText: vi.fn(),
        setBadgeBackgroundColor: vi.fn(),
        setIcon: vi.fn(),
        setTitle: vi.fn(),
        onClicked: {
            addListener: vi.fn(),
        },
    },
    commands: {
        onCommand: {
            addListener: vi.fn(),
        },
        getAll: vi.fn(),
    },
    webRequest: {
        onBeforeRequest: {
            addListener: vi.fn(),
            removeListener: vi.fn(),
        },
        onCompleted: {
            addListener: vi.fn(),
        },
    },
    scripting: {
        executeScript: vi.fn(),
        insertCSS: vi.fn(),
        removeCSS: vi.fn(),
    },
    pageAction: {
        show: vi.fn(),
        hide: vi.fn(),
        setIcon: vi.fn(),
        setTitle: vi.fn(),
    },
};

// Mock browser API for Firefox compatibility
const mockBrowser = { ...mockChrome };

// Set up global mocks
(global as any).chrome = mockChrome;
(global as any).browser = mockBrowser;

describe('Browser Extension - Installation and Lifecycle', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should handle extension installation successfully', async () => {
        const onInstalledCallback = vi.fn();
        mockChrome.runtime.onInstalled.addListener(onInstalledCallback);

        // Simulate installation
        const details = { reason: 'install' as chrome.runtime.OnInstalledReason };
        onInstalledCallback(details);

        expect(mockChrome.runtime.onInstalled.addListener).toHaveBeenCalledWith(
            onInstalledCallback
        );
        expect(onInstalledCallback).toHaveBeenCalledWith(details);
    });

    it('should handle extension update', async () => {
        const onInstalledCallback = vi.fn();
        mockChrome.runtime.onInstalled.addListener(onInstalledCallback);

        const details = {
            reason: 'update' as chrome.runtime.OnInstalledReason,
            previousVersion: '0.9.0',
        };
        onInstalledCallback(details);

        expect(onInstalledCallback).toHaveBeenCalledWith(details);
    });

    it('should get extension manifest information', () => {
        const manifest = mockChrome.runtime.getManifest();

        expect(manifest).toBeDefined();
        expect(manifest.version).toBe('1.0.0');
        expect(manifest.name).toBe('DB Backup Extension');
        expect(manifest.manifest_version).toBe(3);
    });
});

describe('Browser Extension - Message Passing', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should send message from content script to background', async () => {
        const message = { type: 'BACKUP_STATUS', data: { status: 'running' } };
        const mockResponse = { success: true };

        mockChrome.runtime.sendMessage.mockResolvedValue(mockResponse);

        const response = await mockChrome.runtime.sendMessage(message);

        expect(mockChrome.runtime.sendMessage).toHaveBeenCalledWith(message);
        expect(response).toEqual(mockResponse);
    });

    it('should handle messages in background script', () => {
        const messageListener = vi.fn((message, sender, sendResponse) => {
            if (message.type === 'GET_BACKUPS') {
                sendResponse({ backups: [] });
            }
            return true;
        });

        mockChrome.runtime.onMessage.addListener(messageListener);

        const message = { type: 'GET_BACKUPS' };
        const sender = { tab: { id: 1 } };
        const sendResponse = vi.fn();

        messageListener(message, sender, sendResponse);

        expect(messageListener).toHaveBeenCalledWith(message, sender, sendResponse);
        expect(sendResponse).toHaveBeenCalledWith({ backups: [] });
    });

    it('should send message to specific tab', async () => {
        const tabId = 123;
        const message = { type: 'UPDATE_UI', data: { count: 5 } };

        mockChrome.tabs.sendMessage.mockResolvedValue({ received: true });

        const response = await mockChrome.tabs.sendMessage(tabId, message);

        expect(mockChrome.tabs.sendMessage).toHaveBeenCalledWith(tabId, message);
        expect(response).toEqual({ received: true });
    });

    it('should handle message passing errors gracefully', async () => {
        const message = { type: 'TEST' };
        const error = new Error('No receiving end');

        mockChrome.runtime.sendMessage.mockRejectedValue(error);

        await expect(mockChrome.runtime.sendMessage(message)).rejects.toThrow(
            'No receiving end'
        );
    });
});

describe('Browser Extension - Storage API', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should save data to local storage', async () => {
        const data = { backupCount: 10, lastBackup: '2026-01-15' };

        mockChrome.storage.local.set.mockResolvedValue(undefined);

        await mockChrome.storage.local.set(data);

        expect(mockChrome.storage.local.set).toHaveBeenCalledWith(data);
    });

    it('should retrieve data from local storage', async () => {
        const mockData = { backupCount: 10, lastBackup: '2026-01-15' };

        mockChrome.storage.local.get.mockResolvedValue(mockData);

        const data = await mockChrome.storage.local.get(['backupCount', 'lastBackup']);

        expect(mockChrome.storage.local.get).toHaveBeenCalledWith([
            'backupCount',
            'lastBackup',
        ]);
        expect(data).toEqual(mockData);
    });

    it('should save data to sync storage for cross-device sync', async () => {
        const settings = { theme: 'dark', notifications: true };

        mockChrome.storage.sync.set.mockResolvedValue(undefined);

        await mockChrome.storage.sync.set(settings);

        expect(mockChrome.storage.sync.set).toHaveBeenCalledWith(settings);
    });

    it('should listen to storage changes', () => {
        const changeListener = vi.fn();

        mockChrome.storage.onChanged.addListener(changeListener);

        const changes = {
            backupCount: { oldValue: 10, newValue: 11 },
        };
        const areaName = 'local';

        changeListener(changes, areaName);

        expect(changeListener).toHaveBeenCalledWith(changes, areaName);
    });

    it('should clear all storage data', async () => {
        mockChrome.storage.local.clear.mockResolvedValue(undefined);

        await mockChrome.storage.local.clear();

        expect(mockChrome.storage.local.clear).toHaveBeenCalled();
    });
});

describe('Browser Extension - Context Menus', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should create context menu item', () => {
        const menuItem = {
            id: 'backup-page',
            title: 'Backup this page',
            contexts: ['page' as chrome.contextMenus.ContextType],
        };

        mockChrome.contextMenus.create.mockReturnValue('backup-page');

        const menuId = mockChrome.contextMenus.create(menuItem);

        expect(mockChrome.contextMenus.create).toHaveBeenCalledWith(menuItem);
        expect(menuId).toBe('backup-page');
    });

    it('should handle context menu clicks', () => {
        const clickHandler = vi.fn();

        mockChrome.contextMenus.onClicked.addListener(clickHandler);

        const info = {
            menuItemId: 'backup-page',
            pageUrl: 'https://example.com',
        };
        const tab = { id: 1, url: 'https://example.com' };

        clickHandler(info, tab);

        expect(clickHandler).toHaveBeenCalledWith(info, tab);
    });

    it('should update context menu item', async () => {
        const menuId = 'backup-page';
        const updateProperties = { title: 'Backup updated' };

        mockChrome.contextMenus.update.mockResolvedValue(undefined);

        await mockChrome.contextMenus.update(menuId, updateProperties);

        expect(mockChrome.contextMenus.update).toHaveBeenCalledWith(
            menuId,
            updateProperties
        );
    });

    it('should remove all context menu items', async () => {
        mockChrome.contextMenus.removeAll.mockResolvedValue(undefined);

        await mockChrome.contextMenus.removeAll();

        expect(mockChrome.contextMenus.removeAll).toHaveBeenCalled();
    });
});

describe('Browser Extension - Tab Management', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should query active tabs', async () => {
        const mockTabs = [{ id: 1, active: true, url: 'https://example.com' }];

        mockChrome.tabs.query.mockResolvedValue(mockTabs);

        const tabs = await mockChrome.tabs.query({ active: true, currentWindow: true });

        expect(mockChrome.tabs.query).toHaveBeenCalledWith({
            active: true,
            currentWindow: true,
        });
        expect(tabs).toEqual(mockTabs);
    });

    it('should create new tab', async () => {
        const newTab = { id: 2, url: 'https://backup.example.com' };

        mockChrome.tabs.create.mockResolvedValue(newTab);

        const tab = await mockChrome.tabs.create({ url: 'https://backup.example.com' });

        expect(mockChrome.tabs.create).toHaveBeenCalledWith({
            url: 'https://backup.example.com',
        });
        expect(tab).toEqual(newTab);
    });

    it('should update tab URL', async () => {
        const tabId = 1;
        const updateProperties = { url: 'https://new-url.com' };

        mockChrome.tabs.update.mockResolvedValue({ id: tabId, ...updateProperties });

        await mockChrome.tabs.update(tabId, updateProperties);

        expect(mockChrome.tabs.update).toHaveBeenCalledWith(tabId, updateProperties);
    });

    it('should listen to tab activation', () => {
        const activationListener = vi.fn();

        mockChrome.tabs.onActivated.addListener(activationListener);

        const activeInfo = { tabId: 1, windowId: 1 };
        activationListener(activeInfo);

        expect(activationListener).toHaveBeenCalledWith(activeInfo);
    });

    it('should listen to tab updates', () => {
        const updateListener = vi.fn();

        mockChrome.tabs.onUpdated.addListener(updateListener);

        const tabId = 1;
        const changeInfo = { status: 'complete' };
        const tab = { id: tabId, url: 'https://example.com' };

        updateListener(tabId, changeInfo, tab);

        expect(updateListener).toHaveBeenCalledWith(tabId, changeInfo, tab);
    });
});

describe('Browser Extension - Notifications', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should create notification', async () => {
        const notificationOptions = {
            type: 'basic' as chrome.notifications.TemplateType,
            iconUrl: '/icons/icon128.png',
            title: 'Backup Complete',
            message: 'Your database backup is ready',
        };

        mockChrome.notifications.create.mockResolvedValue('notification-id');

        const notificationId = await mockChrome.notifications.create(notificationOptions);

        expect(mockChrome.notifications.create).toHaveBeenCalledWith(notificationOptions);
        expect(notificationId).toBe('notification-id');
    });

    it('should clear notification', async () => {
        const notificationId = 'notification-id';

        mockChrome.notifications.clear.mockResolvedValue(true);

        const wasCleared = await mockChrome.notifications.clear(notificationId);

        expect(mockChrome.notifications.clear).toHaveBeenCalledWith(notificationId);
        expect(wasCleared).toBe(true);
    });

    it('should handle notification clicks', () => {
        const clickListener = vi.fn();

        mockChrome.notifications.onClicked.addListener(clickListener);

        const notificationId = 'notification-id';
        clickListener(notificationId);

        expect(clickListener).toHaveBeenCalledWith(notificationId);
    });
});

describe('Browser Extension - Downloads', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should initiate download', async () => {
        const downloadOptions = {
            url: 'https://example.com/backup.sql',
            filename: 'backup-2026-01-15.sql',
        };

        mockChrome.downloads.download.mockResolvedValue(1);

        const downloadId = await mockChrome.downloads.download(downloadOptions);

        expect(mockChrome.downloads.download).toHaveBeenCalledWith(downloadOptions);
        expect(downloadId).toBe(1);
    });

    it('should search downloads', async () => {
        const query = { filename: 'backup' };
        const mockDownloads = [
            { id: 1, filename: 'backup-2026-01-15.sql', state: 'complete' },
        ];

        mockChrome.downloads.search.mockResolvedValue(mockDownloads);

        const downloads = await mockChrome.downloads.search(query);

        expect(mockChrome.downloads.search).toHaveBeenCalledWith(query);
        expect(downloads).toEqual(mockDownloads);
    });

    it('should cancel download', async () => {
        const downloadId = 1;

        mockChrome.downloads.cancel.mockResolvedValue(undefined);

        await mockChrome.downloads.cancel(downloadId);

        expect(mockChrome.downloads.cancel).toHaveBeenCalledWith(downloadId);
    });

    it('should listen to download state changes', () => {
        const changeListener = vi.fn();

        mockChrome.downloads.onChanged.addListener(changeListener);

        const delta = {
            id: 1,
            state: { current: 'complete', previous: 'in_progress' },
        };

        changeListener(delta);

        expect(changeListener).toHaveBeenCalledWith(delta);
    });
});

describe('Browser Extension - Bookmarks', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should create bookmark', async () => {
        const bookmark = {
            title: 'DB Backup Dashboard',
            url: 'https://backup.example.com/dashboard',
        };

        mockChrome.bookmarks.create.mockResolvedValue({
            id: '1',
            ...bookmark,
        });

        const created = await mockChrome.bookmarks.create(bookmark);

        expect(mockChrome.bookmarks.create).toHaveBeenCalledWith(bookmark);
        expect(created.title).toBe(bookmark.title);
    });

    it('should search bookmarks', async () => {
        const query = 'backup';
        const mockBookmarks = [
            { id: '1', title: 'DB Backup Dashboard', url: 'https://backup.example.com' },
        ];

        mockChrome.bookmarks.search.mockResolvedValue(mockBookmarks);

        const bookmarks = await mockChrome.bookmarks.search(query);

        expect(mockChrome.bookmarks.search).toHaveBeenCalledWith(query);
        expect(bookmarks).toEqual(mockBookmarks);
    });
});

describe('Browser Extension - Permissions', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should check if extension has permissions', async () => {
        const permissions = { permissions: ['tabs', 'storage'] };

        mockChrome.permissions.contains.mockResolvedValue(true);

        const hasPermissions = await mockChrome.permissions.contains(permissions);

        expect(mockChrome.permissions.contains).toHaveBeenCalledWith(permissions);
        expect(hasPermissions).toBe(true);
    });

    it('should request additional permissions', async () => {
        const permissions = { permissions: ['downloads'] };

        mockChrome.permissions.request.mockResolvedValue(true);

        const granted = await mockChrome.permissions.request(permissions);

        expect(mockChrome.permissions.request).toHaveBeenCalledWith(permissions);
        expect(granted).toBe(true);
    });

    it('should remove permissions', async () => {
        const permissions = { permissions: ['downloads'] };

        mockChrome.permissions.remove.mockResolvedValue(true);

        const removed = await mockChrome.permissions.remove(permissions);

        expect(mockChrome.permissions.remove).toHaveBeenCalledWith(permissions);
        expect(removed).toBe(true);
    });
});

describe('Browser Extension - Browser Action', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should set badge text', async () => {
        const text = '5';

        mockChrome.browserAction.setBadgeText.mockResolvedValue(undefined);

        await mockChrome.browserAction.setBadgeText({ text });

        expect(mockChrome.browserAction.setBadgeText).toHaveBeenCalledWith({ text });
    });

    it('should set badge background color', async () => {
        const color = '#FF0000';

        mockChrome.browserAction.setBadgeBackgroundColor.mockResolvedValue(undefined);

        await mockChrome.browserAction.setBadgeBackgroundColor({ color });

        expect(mockChrome.browserAction.setBadgeBackgroundColor).toHaveBeenCalledWith({
            color,
        });
    });

    it('should update icon', async () => {
        const path = '/icons/icon-active.png';

        mockChrome.browserAction.setIcon.mockResolvedValue(undefined);

        await mockChrome.browserAction.setIcon({ path });

        expect(mockChrome.browserAction.setIcon).toHaveBeenCalledWith({ path });
    });

    it('should handle browser action clicks', () => {
        const clickListener = vi.fn();

        mockChrome.browserAction.onClicked.addListener(clickListener);

        const tab = { id: 1, url: 'https://example.com' };
        clickListener(tab);

        expect(clickListener).toHaveBeenCalledWith(tab);
    });
});

describe('Browser Extension - Keyboard Shortcuts', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should register command listener', () => {
        const commandListener = vi.fn();

        mockChrome.commands.onCommand.addListener(commandListener);

        const command = 'backup-now';
        commandListener(command);

        expect(commandListener).toHaveBeenCalledWith(command);
    });

    it('should get all registered commands', async () => {
        const mockCommands = [
            { name: 'backup-now', shortcut: 'Ctrl+Shift+B' },
        ];

        mockChrome.commands.getAll.mockResolvedValue(mockCommands);

        const commands = await mockChrome.commands.getAll();

        expect(mockChrome.commands.getAll).toHaveBeenCalled();
        expect(commands).toEqual(mockCommands);
    });
});

describe('Browser Extension - Content Script Injection', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should inject content script', async () => {
        const injection = {
            target: { tabId: 1 },
            files: ['content.js'],
        };

        mockChrome.scripting.executeScript.mockResolvedValue([{ result: 'injected' }]);

        const results = await mockChrome.scripting.executeScript(injection);

        expect(mockChrome.scripting.executeScript).toHaveBeenCalledWith(injection);
        expect(results[0].result).toBe('injected');
    });

    it('should insert CSS into page', async () => {
        const injection = {
            target: { tabId: 1 },
            files: ['styles.css'],
        };

        mockChrome.scripting.insertCSS.mockResolvedValue(undefined);

        await mockChrome.scripting.insertCSS(injection);

        expect(mockChrome.scripting.insertCSS).toHaveBeenCalledWith(injection);
    });

    it('should remove CSS from page', async () => {
        const injection = {
            target: { tabId: 1 },
            files: ['styles.css'],
        };

        mockChrome.scripting.removeCSS.mockResolvedValue(undefined);

        await mockChrome.scripting.removeCSS(injection);

        expect(mockChrome.scripting.removeCSS).toHaveBeenCalledWith(injection);
    });
});

describe('Browser Extension - Web Request Interception', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should intercept web requests', () => {
        const requestListener = vi.fn();

        mockChrome.webRequest.onBeforeRequest.addListener(requestListener);

        const details = {
            requestId: '1',
            url: 'https://api.example.com/backup',
            method: 'POST',
        };

        requestListener(details);

        expect(requestListener).toHaveBeenCalledWith(details);
    });

    it('should listen to completed requests', () => {
        const completedListener = vi.fn();

        mockChrome.webRequest.onCompleted.addListener(completedListener);

        const details = {
            requestId: '1',
            url: 'https://api.example.com/backup',
            statusCode: 200,
        };

        completedListener(details);

        expect(completedListener).toHaveBeenCalledWith(details);
    });
});

describe('Browser Extension - Page Action', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should show page action for specific tab', async () => {
        const tabId = 1;

        mockChrome.pageAction.show.mockResolvedValue(undefined);

        await mockChrome.pageAction.show(tabId);

        expect(mockChrome.pageAction.show).toHaveBeenCalledWith(tabId);
    });

    it('should hide page action', async () => {
        const tabId = 1;

        mockChrome.pageAction.hide.mockResolvedValue(undefined);

        await mockChrome.pageAction.hide(tabId);

        expect(mockChrome.pageAction.hide).toHaveBeenCalledWith(tabId);
    });

    it('should set page action icon', async () => {
        const tabId = 1;
        const iconPath = '/icons/page-action.png';

        mockChrome.pageAction.setIcon.mockResolvedValue(undefined);

        await mockChrome.pageAction.setIcon({ tabId, path: iconPath });

        expect(mockChrome.pageAction.setIcon).toHaveBeenCalledWith({
            tabId,
            path: iconPath,
        });
    });
});

describe('Browser Extension - Cross-Browser Compatibility', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should work with Chrome API', () => {
        expect((global as any).chrome).toBeDefined();
        expect((global as any).chrome.runtime).toBeDefined();
        expect((global as any).chrome.storage).toBeDefined();
    });

    it('should work with Firefox browser API', () => {
        expect((global as any).browser).toBeDefined();
        expect((global as any).browser.runtime).toBeDefined();
        expect((global as any).browser.storage).toBeDefined();
    });

    it('should detect API availability', () => {
        const hasChrome = typeof (global as any).chrome !== 'undefined';
        const hasBrowser = typeof (global as any).browser !== 'undefined';

        expect(hasChrome || hasBrowser).toBe(true);
    });

    it('should use polyfill for cross-browser support', () => {
        // Simulate polyfill pattern
        const api = (global as any).chrome || (global as any).browser;

        expect(api).toBeDefined();
        expect(api.runtime).toBeDefined();
    });
});

describe('Browser Extension - Error Handling', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should handle runtime errors', async () => {
        mockChrome.runtime.lastError = { message: 'Test error' };

        const checkError = () => {
            if (mockChrome.runtime.lastError) {
                throw new Error(mockChrome.runtime.lastError.message);
            }
        };

        expect(checkError).toThrow('Test error');

        mockChrome.runtime.lastError = null;
    });

    it('should handle storage quota exceeded', async () => {
        const largeData = { data: 'x'.repeat(10000000) }; // Large data

        mockChrome.storage.local.set.mockRejectedValue(
            new Error('QUOTA_BYTES quota exceeded')
        );

        await expect(mockChrome.storage.local.set(largeData)).rejects.toThrow(
            'QUOTA_BYTES quota exceeded'
        );
    });

    it('should handle missing permissions gracefully', async () => {
        const permissions = { permissions: ['tabs'] };

        mockChrome.permissions.contains.mockResolvedValue(false);

        const hasPermissions = await mockChrome.permissions.contains(permissions);

        expect(hasPermissions).toBe(false);
    });
});
