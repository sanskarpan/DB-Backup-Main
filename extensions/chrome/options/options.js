/**
 * Options Page Controller for DB Backup Manager Extension
 */

// ============================================================================
// Initialize
// ============================================================================

document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await loadStatistics();
  setupEventListeners();
});

// ============================================================================
// Load Settings
// ============================================================================

async function loadSettings() {
  try {
    const settings = await Utils.getStorage('settings');

    if (settings) {
      // API Configuration
      document.getElementById('apiUrl').value = settings.apiUrl || 'http://localhost:8080';
      document.getElementById('apiKey').value = settings.apiKey || '';

      // Sync Settings
      document.getElementById('syncInterval').value = settings.syncInterval || 5;
      updateSyncIntervalValue(settings.syncInterval || 5);
      document.getElementById('autoSync').checked = settings.autoSync !== false;

      // Notifications
      document.getElementById('notificationsEnabled').checked = settings.notificationsEnabled !== false;
      document.getElementById('notifySuccess').checked = settings.notifySuccess !== false;
      document.getElementById('notifyFailure').checked = settings.notifyFailure !== false;
      document.getElementById('notifyAlerts').checked = settings.notifyAlerts !== false;

      // Display
      document.getElementById('badgeEnabled').checked = settings.badgeEnabled !== false;
      document.getElementById('contextMenuEnabled').checked = settings.contextMenuEnabled !== false;
      document.getElementById('theme').value = settings.theme || 'light';

      // Advanced
      document.getElementById('autoBackup').checked = settings.autoBackup || false;
      document.getElementById('debugMode').checked = settings.debugMode || false;
    }
  } catch (error) {
    console.error('Failed to load settings:', error);
    showError('Failed to load settings');
  }
}

// ============================================================================
// Load Statistics
// ============================================================================

async function loadStatistics() {
  try {
    const stats = await Utils.getStorage('statistics');

    if (stats) {
      document.getElementById('statTotalBackups').textContent = stats.totalBackups || 0;
      document.getElementById('statSuccessful').textContent = stats.successfulBackups || 0;
      document.getElementById('statFailed').textContent = stats.failedBackups || 0;
      document.getElementById('statTotalSize').textContent = Utils.formatBytes(stats.totalSize || 0);
    }
  } catch (error) {
    console.error('Failed to load statistics:', error);
  }
}

// ============================================================================
// Event Listeners
// ============================================================================

function setupEventListeners() {
  // Form submission
  document.getElementById('settingsForm').addEventListener('submit', handleSaveSettings);

  // API Key toggle
  document.getElementById('toggleApiKey').addEventListener('click', toggleApiKeyVisibility);

  // Test connection
  document.getElementById('testConnection').addEventListener('click', testConnection);

  // Sync interval slider
  document.getElementById('syncInterval').addEventListener('input', (e) => {
    updateSyncIntervalValue(e.target.value);
  });

  // Reset button
  document.getElementById('resetBtn').addEventListener('click', resetToDefaults);

  // Clear data button
  document.getElementById('clearDataBtn').addEventListener('click', clearAllData);
}

// ============================================================================
// Save Settings
// ============================================================================

async function handleSaveSettings(e) {
  e.preventDefault();

  const saveBtn = document.getElementById('saveBtn');
  const originalHTML = saveBtn.innerHTML;

  saveBtn.disabled = true;
  saveBtn.innerHTML = '<svg class="spinning" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg> Saving...';

  try {
    const settings = {
      // API Configuration
      apiUrl: document.getElementById('apiUrl').value,
      apiKey: document.getElementById('apiKey').value,

      // Sync Settings
      syncInterval: parseInt(document.getElementById('syncInterval').value),
      autoSync: document.getElementById('autoSync').checked,

      // Notifications
      notificationsEnabled: document.getElementById('notificationsEnabled').checked,
      notifySuccess: document.getElementById('notifySuccess').checked,
      notifyFailure: document.getElementById('notifyFailure').checked,
      notifyAlerts: document.getElementById('notifyAlerts').checked,

      // Display
      badgeEnabled: document.getElementById('badgeEnabled').checked,
      contextMenuEnabled: document.getElementById('contextMenuEnabled').checked,
      theme: document.getElementById('theme').value,

      // Advanced
      autoBackup: document.getElementById('autoBackup').checked,
      debugMode: document.getElementById('debugMode').checked,
    };

    // Validate API URL
    if (!Utils.isValidURL(settings.apiUrl)) {
      throw new Error('Invalid API URL');
    }

    // Validate API Key
    if (!settings.apiKey || settings.apiKey.trim() === '') {
      throw new Error('API Key is required');
    }

    // Save to storage
    await Utils.setStorage('settings', settings);

    // Notify background script
    await Utils.sendMessage({
      action: 'updateSettings',
      data: settings,
    });

    showSuccess('Settings saved successfully!');

    // Reload statistics after save
    setTimeout(() => loadStatistics(), 1000);
  } catch (error) {
    console.error('Failed to save settings:', error);
    showError(error.message || 'Failed to save settings');
  } finally {
    saveBtn.disabled = false;
    saveBtn.innerHTML = originalHTML;
  }
}

// ============================================================================
// API Key Visibility Toggle
// ============================================================================

function toggleApiKeyVisibility() {
  const apiKeyInput = document.getElementById('apiKey');
  const toggleBtn = document.getElementById('toggleApiKey');

  if (apiKeyInput.type === 'password') {
    apiKeyInput.type = 'text';
    toggleBtn.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line></svg>';
  } else {
    apiKeyInput.type = 'password';
    toggleBtn.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
  }
}

// ============================================================================
// Test Connection
// ============================================================================

async function testConnection() {
  const testBtn = document.getElementById('testConnection');
  const statusSpan = document.getElementById('connectionStatus');
  const originalHTML = testBtn.innerHTML;

  testBtn.disabled = true;
  testBtn.innerHTML = '<svg class="spinning" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg> Testing...';
  statusSpan.textContent = '';
  statusSpan.className = 'connection-status';

  try {
    const apiUrl = document.getElementById('apiUrl').value;
    const apiKey = document.getElementById('apiKey').value;

    if (!apiUrl || !apiKey) {
      throw new Error('Please enter both API URL and API Key');
    }

    // Test connection by fetching stats
    const response = await fetch(`${apiUrl}/api/v1/stats/dashboard`, {
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Connection failed: ${response.status} ${response.statusText}`);
    }

    await response.json();

    statusSpan.textContent = '✓ Connection successful!';
    statusSpan.className = 'connection-status success';
  } catch (error) {
    console.error('Connection test failed:', error);
    statusSpan.textContent = `✗ ${error.message}`;
    statusSpan.className = 'connection-status error';
  } finally {
    testBtn.disabled = false;
    testBtn.innerHTML = originalHTML;
  }
}

// ============================================================================
// Reset to Defaults
// ============================================================================

async function resetToDefaults() {
  if (!confirm('Are you sure you want to reset all settings to defaults? This cannot be undone.')) {
    return;
  }

  try {
    const defaults = {
      apiUrl: 'http://localhost:8080',
      apiKey: '',
      syncInterval: 5,
      autoSync: true,
      notificationsEnabled: true,
      notifySuccess: true,
      notifyFailure: true,
      notifyAlerts: true,
      badgeEnabled: true,
      contextMenuEnabled: true,
      theme: 'light',
      autoBackup: false,
      debugMode: false,
    };

    await Utils.setStorage('settings', defaults);

    // Reload the page to show default values
    window.location.reload();
  } catch (error) {
    console.error('Failed to reset settings:', error);
    showError('Failed to reset settings');
  }
}

// ============================================================================
// Clear All Data
// ============================================================================

async function clearAllData() {
  if (!confirm('Are you sure you want to clear ALL extension data? This will remove all settings, statistics, and cached data. This cannot be undone.')) {
    return;
  }

  if (!confirm('This is your last chance! Are you ABSOLUTELY sure you want to clear all data?')) {
    return;
  }

  try {
    // Clear all storage
    const browserAPI = Utils.getBrowserAPI();

    if (browserAPI.storage.local.clear.length === 0) {
      await browserAPI.storage.local.clear();
    } else {
      await Utils.promisify(browserAPI.storage.local.clear.bind(browserAPI.storage.local));
    }

    showSuccess('All data cleared successfully! Reloading...');

    // Reload after 2 seconds
    setTimeout(() => window.location.reload(), 2000);
  } catch (error) {
    console.error('Failed to clear data:', error);
    showError('Failed to clear data');
  }
}

// ============================================================================
// UI Helpers
// ============================================================================

function updateSyncIntervalValue(value) {
  const valueSpan = document.getElementById('syncIntervalValue');
  valueSpan.textContent = `${value} minute${value !== 1 ? 's' : ''}`;
}

function showSuccess(message) {
  const successDiv = document.getElementById('successMessage');
  const errorDiv = document.getElementById('errorMessage');

  errorDiv.style.display = 'none';
  successDiv.querySelector('span').textContent = message;
  successDiv.style.display = 'flex';

  // Hide after 5 seconds
  setTimeout(() => {
    successDiv.style.display = 'none';
  }, 5000);

  // Scroll to top
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function showError(message) {
  const successDiv = document.getElementById('successMessage');
  const errorDiv = document.getElementById('errorMessage');

  successDiv.style.display = 'none';
  document.getElementById('errorText').textContent = message;
  errorDiv.style.display = 'flex';

  // Hide after 10 seconds
  setTimeout(() => {
    errorDiv.style.display = 'none';
  }, 10000);

  // Scroll to top
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

// ============================================================================
// Error Handling
// ============================================================================

window.addEventListener('error', (event) => {
  console.error('Options page error:', event.error);
});

window.addEventListener('unhandledrejection', (event) => {
  console.error('Unhandled promise rejection:', event.reason);
});
