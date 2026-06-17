/**
 * Background Service Worker for DB Backup Manager Chrome Extension
 * Manifest V3 compatible
 */

// BackupAPI and Utils are loaded as globals via the manifest "scripts" array
// Initialize API client
let api = new BackupAPI();

// Extension state
const state = {
  isConnected: false,
  lastSyncTime: null,
  backupCount: 0,
  failedBackupCount: 0,
  alerts: [],
};

// ============================================================================
// Installation and Updates
// ============================================================================

chrome.runtime.onInstalled.addListener(async (details) => {
  console.log('Extension installed/updated:', details.reason);

  if (details.reason === 'install') {
    // First time installation
    await initializeExtension();

    // Open options page
    chrome.tabs.create({ url: 'options/options.html' });

    // Show welcome notification
    Utils.createNotification('welcome', {
      title: 'Welcome to DB Backup Manager!',
      message: 'Configure your API settings to get started.',
      iconUrl: '../icons/icon-128.png',
    });
  } else if (details.reason === 'update') {
    // Extension updated
    const manifest = chrome.runtime.getManifest();
    Utils.createNotification('updated', {
      title: 'Extension Updated',
      message: `DB Backup Manager updated to version ${manifest.version}`,
    });
  }

  // Set up context menus
  setupContextMenus();

  // Set up alarms for background sync
  setupAlarms();
});

/**
 * Initialize extension with default settings
 */
async function initializeExtension() {
  const defaults = {
    apiUrl: 'http://localhost:8080',
    apiKey: '',
    syncInterval: 5, // minutes
    notificationsEnabled: true,
    badgeEnabled: true,
    contextMenuEnabled: true,
    autoBackup: false,
    theme: 'light',
  };

  await Utils.setStorage('settings', defaults);
  await Utils.setStorage('lastSync', null);
  await Utils.setStorage('statistics', {
    totalBackups: 0,
    successfulBackups: 0,
    failedBackups: 0,
    totalSize: 0,
  });
}

// ============================================================================
// Context Menus
// ============================================================================

function setupContextMenus() {
  // Remove existing menus
  chrome.contextMenus.removeAll(() => {
    // Create backup menu
    chrome.contextMenus.create({
      id: 'quick-backup',
      title: 'Quick Backup',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'view-backups',
      title: 'View Backups',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'view-monitoring',
      title: 'View Monitoring',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'separator-1',
      type: 'separator',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'sync-now',
      title: 'Sync Now',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'open-dashboard',
      title: 'Open Dashboard',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'separator-2',
      type: 'separator',
      contexts: ['action'],
    });

    chrome.contextMenus.create({
      id: 'settings',
      title: 'Settings',
      contexts: ['action'],
    });
  });
}

// Handle context menu clicks
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  const settings = await Utils.getStorage('settings');

  switch (info.menuItemId) {
    case 'quick-backup':
      handleQuickBackup();
      break;

    case 'view-backups':
      chrome.tabs.create({ url: `${settings.apiUrl.replace('/api', '')}/backups` });
      break;

    case 'view-monitoring':
      chrome.tabs.create({ url: `${settings.apiUrl.replace('/api', '')}/monitoring` });
      break;

    case 'sync-now':
      await syncBackupData();
      break;

    case 'open-dashboard':
      chrome.tabs.create({ url: settings.apiUrl.replace('/api', '') });
      break;

    case 'settings':
      chrome.tabs.create({ url: 'options/options.html' });
      break;
  }
});

// ============================================================================
// Alarms for Background Sync
// ============================================================================

function setupAlarms() {
  // Clear existing alarms
  chrome.alarms.clearAll();

  // Set up sync alarm
  chrome.alarms.create('sync-backups', {
    periodInMinutes: 5, // Sync every 5 minutes
  });

  // Set up monitoring alarm
  chrome.alarms.create('check-monitoring', {
    periodInMinutes: 1, // Check monitoring every minute
  });
}

// Handle alarms
chrome.alarms.onAlarm.addListener(async (alarm) => {
  const settings = await Utils.getStorage('settings');

  if (!settings || !settings.apiKey) {
    return; // Not configured yet
  }

  switch (alarm.name) {
    case 'sync-backups':
      await syncBackupData();
      break;

    case 'check-monitoring':
      await checkMonitoring();
      break;
  }
});

// ============================================================================
// Background Sync Functions
// ============================================================================

/**
 * Sync backup data from API
 */
async function syncBackupData() {
  try {
    const settings = await Utils.getStorage('settings');

    if (!settings || !settings.apiKey) {
      return;
    }

    // Configure API
    api.setBaseURL(settings.apiUrl);
    api.setApiKey(settings.apiKey);

    // Fetch backups
    const backups = await api.listBackups({ limit: 100 });

    // Update state
    state.backupCount = backups.total || backups.length || 0;
    state.failedBackupCount = backups.filter(b => b.status === 'failed').length;
    state.lastSyncTime = new Date();
    state.isConnected = true;

    // Update badge
    if (settings.badgeEnabled) {
      updateBadge();
    }

    // Store sync time
    await Utils.setStorage('lastSync', Date.now());

    // Update statistics
    const stats = await Utils.getStorage('statistics') || {};
    stats.totalBackups = state.backupCount;
    stats.failedBackups = state.failedBackupCount;
    await Utils.setStorage('statistics', stats);

    console.log('Sync completed:', backups.length, 'backups');
  } catch (error) {
    console.error('Sync failed:', error);
    state.isConnected = false;

    // Show error notification (only once per session)
    const lastError = await Utils.getStorage('lastErrorNotification');
    if (!lastError || Date.now() - lastError > 3600000) {
      // Only show if last error was more than 1 hour ago
      Utils.createNotification('sync-error', {
        type: 'basic',
        title: 'Sync Failed',
        message: 'Unable to connect to DB Backup Manager API',
        priority: 1,
      });
      await Utils.setStorage('lastErrorNotification', Date.now());
    }
  }
}

/**
 * Check monitoring status and alerts
 */
async function checkMonitoring() {
  try {
    const settings = await Utils.getStorage('settings');

    if (!settings || !settings.apiKey || !settings.notificationsEnabled) {
      return;
    }

    api.setBaseURL(settings.apiUrl);
    api.setApiKey(settings.apiKey);

    // Fetch alerts
    const alerts = await api.getAlerts();

    // Check for new critical alerts
    const criticalAlerts = alerts.filter(a => a.severity === 'critical' && !a.acknowledged);

    for (const alert of criticalAlerts) {
      // Check if we already notified about this alert
      const notifiedAlerts = await Utils.getStorage('notifiedAlerts') || [];

      if (!notifiedAlerts.includes(alert.id)) {
        // Send notification
        Utils.createNotification(`alert-${alert.id}`, {
          type: 'basic',
          title: 'Critical Alert',
          message: alert.message,
          priority: 2,
        });

        // Mark as notified
        notifiedAlerts.push(alert.id);
        await Utils.setStorage('notifiedAlerts', notifiedAlerts);
      }
    }

    state.alerts = alerts;
  } catch (error) {
    console.error('Monitoring check failed:', error);
  }
}

/**
 * Update extension badge
 */
function updateBadge() {
  if (state.failedBackupCount > 0) {
    Utils.setBadgeText(String(state.failedBackupCount));
    Utils.setBadgeBackgroundColor('#ef4444'); // Red for failures
  } else if (!state.isConnected) {
    Utils.setBadgeText('!');
    Utils.setBadgeBackgroundColor('#f59e0b'); // Orange for disconnected
  } else {
    Utils.setBadgeText('');
  }
}

// ============================================================================
// Quick Backup
// ============================================================================

async function handleQuickBackup() {
  try {
    const settings = await Utils.getStorage('settings');

    if (!settings || !settings.apiKey) {
      Utils.createNotification('not-configured', {
        title: 'Not Configured',
        message: 'Please configure your API settings first',
      });
      chrome.tabs.create({ url: 'options/options.html' });
      return;
    }

    api.setBaseURL(settings.apiUrl);
    api.setApiKey(settings.apiKey);

    // Show notification
    Utils.createNotification('backup-starting', {
      title: 'Backup Starting',
      message: 'Initiating quick backup...',
    });

    // Get databases
    const databases = await api.listDatabases();

    if (databases.length === 0) {
      Utils.createNotification('no-databases', {
        title: 'No Databases',
        message: 'No databases found to backup',
      });
      return;
    }

    // Create backup for first database (or show selection UI)
    const database = databases[0];

    const backup = await api.createBackup({
      database_id: database.id,
      type: 'full',
      compression: true,
      encryption: true,
    });

    Utils.createNotification('backup-success', {
      title: 'Backup Created',
      message: `Backup for ${database.name} created successfully`,
    });

    // Trigger sync
    await syncBackupData();
  } catch (error) {
    Utils.handleError(error, 'Quick Backup');
  }
}

// ============================================================================
// Message Handling
// ============================================================================

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    try {
      switch (message.action) {
        case 'getState':
          sendResponse({ success: true, data: state });
          break;

        case 'sync':
          await syncBackupData();
          sendResponse({ success: true });
          break;

        case 'quickBackup':
          await handleQuickBackup();
          sendResponse({ success: true });
          break;

        case 'updateSettings':
          await Utils.setStorage('settings', message.data);
          api.setBaseURL(message.data.apiUrl);
          api.setApiKey(message.data.apiKey);
          setupAlarms();
          await syncBackupData();
          sendResponse({ success: true });
          break;

        case 'getBackups':
          const backups = await api.listBackups(message.params || {});
          sendResponse({ success: true, data: backups });
          break;

        case 'getDatabases':
          const databases = await api.listDatabases();
          sendResponse({ success: true, data: databases });
          break;

        case 'getStats':
          const stats = await api.getDashboardStats();
          sendResponse({ success: true, data: stats });
          break;

        default:
          sendResponse({ success: false, error: 'Unknown action' });
      }
    } catch (error) {
      console.error('Message handling error:', error);
      sendResponse({ success: false, error: error.message });
    }
  })();

  return true; // Keep message channel open for async response
});

// ============================================================================
// Keyboard Commands
// ============================================================================

chrome.commands.onCommand.addListener(async (command) => {
  switch (command) {
    case 'quick_backup':
      await handleQuickBackup();
      break;

    case 'view_backups':
      const settings = await Utils.getStorage('settings');
      if (settings && settings.apiUrl) {
        chrome.tabs.create({ url: `${settings.apiUrl.replace('/api', '')}/backups` });
      }
      break;
  }
});

// ============================================================================
// Notification Click Handling
// ============================================================================

chrome.notifications.onClicked.addListener(async (notificationId) => {
  const settings = await Utils.getStorage('settings');

  if (!settings || !settings.apiUrl) {
    return;
  }

  // Open relevant page based on notification
  if (notificationId.startsWith('backup-')) {
    chrome.tabs.create({ url: `${settings.apiUrl.replace('/api', '')}/backups` });
  } else if (notificationId.startsWith('alert-')) {
    chrome.tabs.create({ url: `${settings.apiUrl.replace('/api', '')}/monitoring` });
  }
});

// ============================================================================
// Initialize on startup
// ============================================================================

// Initial sync on extension load
(async () => {
  const settings = await Utils.getStorage('settings');

  if (settings && settings.apiKey) {
    api.setBaseURL(settings.apiUrl);
    api.setApiKey(settings.apiKey);
    await syncBackupData();
    await checkMonitoring();
  }
})();

console.log('DB Backup Manager background service worker loaded');
