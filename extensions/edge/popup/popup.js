/**
 * Popup UI Controller for DB Backup Manager Extension
 */

import { Utils } from '../shared/utils.js';

let settings = null;
let state = null;

// ============================================================================
// Initialization
// ============================================================================

document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await loadState();

  if (!settings || !settings.apiKey) {
    showNotConfigured();
    return;
  }

  setupEventListeners();
  await loadDashboardData();
  updateLastSyncTime();

  // Auto-refresh every 30 seconds
  setInterval(async () => {
    await loadState();
    await loadDashboardData();
    updateLastSyncTime();
  }, 30000);
});

// ============================================================================
// Settings and State
// ============================================================================

async function loadSettings() {
  try {
    settings = await Utils.getStorage('settings');
  } catch (error) {
    console.error('Failed to load settings:', error);
  }
}

async function loadState() {
  try {
    const response = await Utils.sendMessage({ action: 'getState' });
    if (response.success) {
      state = response.data;
      updateConnectionStatus();
    }
  } catch (error) {
    console.error('Failed to load state:', error);
  }
}

// ============================================================================
// UI Updates
// ============================================================================

function updateConnectionStatus() {
  const statusDot = document.getElementById('statusDot');
  const statusText = document.getElementById('statusText');

  if (!statusDot || !statusText) return;

  if (state && state.isConnected) {
    statusDot.classList.remove('disconnected', 'connecting');
    statusText.textContent = 'Connected';
  } else {
    statusDot.classList.add('disconnected');
    statusText.textContent = 'Disconnected';
  }
}

function updateLastSyncTime() {
  const lastSyncText = document.getElementById('lastSyncText');

  if (!lastSyncText) return;

  if (state && state.lastSyncTime) {
    lastSyncText.textContent = Utils.formatDate(state.lastSyncTime);
  } else {
    lastSyncText.textContent = 'Never';
  }
}

// ============================================================================
// Dashboard Data
// ============================================================================

async function loadDashboardData() {
  try {
    // Load statistics
    const statsData = await Utils.getStorage('statistics');

    if (statsData) {
      document.getElementById('totalBackups').textContent = statsData.totalBackups || 0;
      document.getElementById('failedBackups').textContent = statsData.failedBackups || 0;
      document.getElementById('totalSize').textContent = Utils.formatBytes(statsData.totalSize || 0);
    }

    // Load recent backups
    await loadRecentBackups();

    // Load alerts
    if (state && state.alerts && state.alerts.length > 0) {
      displayAlerts(state.alerts);
    }
  } catch (error) {
    console.error('Failed to load dashboard data:', error);
  }
}

async function loadRecentBackups() {
  const backupsList = document.getElementById('backupsList');

  try {
    const response = await Utils.sendMessage({
      action: 'getBackups',
      params: { limit: 5, sort: 'created_at:desc' },
    });

    if (!response.success) {
      throw new Error(response.error);
    }

    const backups = response.data;

    if (!backups || backups.length === 0) {
      backupsList.innerHTML = '<div class="empty-state">No backups yet</div>';
      return;
    }

    backupsList.innerHTML = backups
      .map(
        (backup) => `
      <div class="backup-item" data-id="${backup.id}">
        <div class="backup-header">
          <span class="backup-name">${backup.database_name || 'Unknown'}</span>
          <span class="backup-status ${backup.status}">
            ${Utils.getStatusIcon(backup.status)} ${backup.status}
          </span>
        </div>
        <div class="backup-info">
          <span>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
            ${Utils.formatDate(backup.created_at)}
          </span>
          <span>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
            </svg>
            ${Utils.formatBytes(backup.size || 0)}
          </span>
        </div>
      </div>
    `
      )
      .join('');

    // Add click handlers to backup items
    document.querySelectorAll('.backup-item').forEach((item) => {
      item.addEventListener('click', () => {
        const backupId = item.dataset.id;
        openBackupDetails(backupId);
      });
    });
  } catch (error) {
    console.error('Failed to load backups:', error);
    backupsList.innerHTML = '<div class="empty-state">Failed to load backups</div>';
  }
}

function displayAlerts(alerts) {
  const alertsSection = document.getElementById('alertsSection');
  const alertsList = document.getElementById('alertsList');
  const alertCount = document.getElementById('alertCount');

  const activeAlerts = alerts.filter((a) => !a.acknowledged);

  if (activeAlerts.length === 0) {
    alertsSection.style.display = 'none';
    return;
  }

  alertsSection.style.display = 'block';
  alertCount.textContent = activeAlerts.length;

  alertsList.innerHTML = activeAlerts
    .map(
      (alert) => `
    <div class="alert-item ${alert.severity}">
      <div class="alert-title">${alert.title}</div>
      <div class="alert-message">${alert.message}</div>
    </div>
  `
    )
    .join('');
}

// ============================================================================
// Event Listeners
// ============================================================================

function setupEventListeners() {
  // Quick Backup
  document.getElementById('quickBackupBtn').addEventListener('click', async () => {
    const btn = document.getElementById('quickBackupBtn');
    btn.disabled = true;
    btn.innerHTML = '<svg class="spinning" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg> Starting...';

    try {
      await Utils.sendMessage({ action: 'quickBackup' });
      setTimeout(() => loadRecentBackups(), 2000);
    } catch (error) {
      console.error('Quick backup failed:', error);
    } finally {
      btn.disabled = false;
      btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg> Quick Backup';
    }
  });

  // View Backups
  document.getElementById('viewBackupsBtn').addEventListener('click', () => {
    if (settings && settings.apiUrl) {
      Utils.openTab(`${settings.apiUrl.replace('/api', '')}/backups`);
    }
  });

  // Monitoring
  document.getElementById('monitoringBtn').addEventListener('click', () => {
    if (settings && settings.apiUrl) {
      Utils.openTab(`${settings.apiUrl.replace('/api', '')}/monitoring`);
    }
  });

  // Refresh
  document.getElementById('refreshBtn').addEventListener('click', async () => {
    const btn = document.getElementById('refreshBtn');
    btn.classList.add('spinning');

    try {
      await Utils.sendMessage({ action: 'sync' });
      await loadState();
      await loadDashboardData();
    } catch (error) {
      console.error('Refresh failed:', error);
    } finally {
      btn.classList.remove('spinning');
    }
  });

  // Sync
  document.getElementById('syncBtn').addEventListener('click', async () => {
    const btn = document.getElementById('syncBtn');
    const svg = btn.querySelector('svg');
    if (svg) svg.classList.add('spinning');

    try {
      await Utils.sendMessage({ action: 'sync' });
      await loadState();
      await loadDashboardData();
      updateLastSyncTime();
    } catch (error) {
      console.error('Sync failed:', error);
    } finally {
      if (svg) svg.classList.remove('spinning');
    }
  });

  // Settings
  document.getElementById('settingsBtn').addEventListener('click', () => {
    Utils.openTab('options/options.html');
  });

  // Dashboard
  document.getElementById('dashboardBtn').addEventListener('click', () => {
    if (settings && settings.apiUrl) {
      Utils.openTab(settings.apiUrl.replace('/api', ''));
    }
  });

  // Configure button (if not configured)
  const configureBtn = document.getElementById('configureBtn');
  if (configureBtn) {
    configureBtn.addEventListener('click', () => {
      Utils.openTab('options/options.html');
    });
  }
}

// ============================================================================
// Helper Functions
// ============================================================================

function openBackupDetails(backupId) {
  if (settings && settings.apiUrl) {
    Utils.openTab(`${settings.apiUrl.replace('/api', '')}/backups/${backupId}`);
  }
}

function showNotConfigured() {
  document.getElementById('notConfigured').style.display = 'flex';
}

// ============================================================================
// Error Handling
// ============================================================================

window.addEventListener('error', (event) => {
  console.error('Popup error:', event.error);
});

window.addEventListener('unhandledrejection', (event) => {
  console.error('Unhandled promise rejection:', event.reason);
});
