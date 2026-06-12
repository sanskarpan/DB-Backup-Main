/**
 * Content Script for DB Backup Manager Extension
 * Injects functionality into web pages to detect database-related pages
 * and provide quick backup capabilities
 */

// ============================================================================
// Constants
// ============================================================================

const DB_TOOLS = {
  phpmyadmin: {
    name: 'phpMyAdmin',
    selectors: ['#pma_navigation', '.navigation', '#topmenu'],
    color: '#f89406',
  },
  adminer: {
    name: 'Adminer',
    selectors: ['#menu', 'p.logout', '.version'],
    color: '#2196f3',
  },
  pgadmin: {
    name: 'pgAdmin',
    selectors: ['#tree', '.pg-panel-header', '#navbar'],
    color: '#336791',
  },
  mongoexpress: {
    name: 'Mongo Express',
    selectors: ['#navbar', '.navbar-brand', '#collection-view'],
    color: '#4caf50',
  },
  mysqlworkbench: {
    name: 'MySQL Workbench',
    selectors: ['.workbench', '#wb-main'],
    color: '#00758f',
  },
  redis: {
    name: 'Redis Commander',
    selectors: ['.redis-commander', '#keyTree'],
    color: '#d82c20',
  },
};

// State
let detectedTool = null;
let floatingButton = null;
let isInjected = false;
let settings = null;

// ============================================================================
// Initialization
// ============================================================================

(function init() {
  // Load settings first
  loadSettings().then(() => {
    // Detect database tools on the page
    detectDatabaseTool();

    // If a tool is detected, inject the UI
    if (detectedTool) {
      injectFloatingButton();
      addPageIndicators();
    }

    // Listen for dynamic content changes
    observePageChanges();

    // Listen for messages from background script
    setupMessageListener();
  });
})();

// ============================================================================
// Settings
// ============================================================================

async function loadSettings() {
  try {
    const browserAPI = getBrowserAPI();

    if (browserAPI.storage.local.get.length === 0) {
      const data = await browserAPI.storage.local.get('settings');
      settings = data.settings || getDefaultSettings();
    } else {
      const data = await new Promise((resolve) => {
        browserAPI.storage.local.get('settings', resolve);
      });
      settings = data.settings || getDefaultSettings();
    }
  } catch (error) {
    console.error('[DB Backup] Failed to load settings:', error);
    settings = getDefaultSettings();
  }
}

function getDefaultSettings() {
  return {
    apiUrl: 'http://localhost:8080',
    apiKey: '',
    autoDetect: true,
    showFloatingButton: true,
    contextMenuEnabled: true,
  };
}

// ============================================================================
// Database Tool Detection
// ============================================================================

function detectDatabaseTool() {
  // Check URL patterns first
  const url = window.location.href.toLowerCase();
  const hostname = window.location.hostname.toLowerCase();

  if (url.includes('phpmyadmin') || hostname.includes('phpmyadmin')) {
    detectedTool = DB_TOOLS.phpmyadmin;
    return;
  }

  if (url.includes('adminer') || hostname.includes('adminer')) {
    detectedTool = DB_TOOLS.adminer;
    return;
  }

  if (url.includes('pgadmin') || hostname.includes('pgadmin')) {
    detectedTool = DB_TOOLS.pgadmin;
    return;
  }

  if (url.includes('mongo-express') || hostname.includes('mongoexpress')) {
    detectedTool = DB_TOOLS.mongoexpress;
    return;
  }

  if (url.includes('redis-commander') || hostname.includes('redis')) {
    detectedTool = DB_TOOLS.redis;
    return;
  }

  // Check DOM selectors
  for (const [key, tool] of Object.entries(DB_TOOLS)) {
    for (const selector of tool.selectors) {
      if (document.querySelector(selector)) {
        detectedTool = tool;
        console.log(`[DB Backup] Detected ${tool.name}`);
        return;
      }
    }
  }
}

// ============================================================================
// Floating Action Button
// ============================================================================

function injectFloatingButton() {
  if (isInjected || !settings.showFloatingButton) {
    return;
  }

  // Create floating button container
  floatingButton = document.createElement('div');
  floatingButton.id = 'db-backup-fab';
  floatingButton.innerHTML = `
    <div class="db-backup-fab-button" title="Quick Backup">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
        <polyline points="7 10 12 15 17 10"></polyline>
        <line x1="12" y1="15" x2="12" y2="3"></line>
      </svg>
    </div>
    <div class="db-backup-fab-menu" style="display: none;">
      <div class="db-backup-fab-menu-item" data-action="quick-backup">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        <span>Quick Backup</span>
      </div>
      <div class="db-backup-fab-menu-item" data-action="scheduled-backup">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polyline points="12 6 12 12 16 14"></polyline>
        </svg>
        <span>Schedule Backup</span>
      </div>
      <div class="db-backup-fab-menu-item" data-action="view-backups">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
          <polyline points="13 2 13 9 20 9"></polyline>
        </svg>
        <span>View Backups</span>
      </div>
      <div class="db-backup-fab-menu-item" data-action="open-dashboard">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="7" height="7"></rect>
          <rect x="14" y="3" width="7" height="7"></rect>
          <rect x="14" y="14" width="7" height="7"></rect>
          <rect x="3" y="14" width="7" height="7"></rect>
        </svg>
        <span>Open Dashboard</span>
      </div>
    </div>
  `;

  // Add styles
  injectStyles();

  // Add to page
  document.body.appendChild(floatingButton);

  // Set color based on detected tool
  if (detectedTool) {
    floatingButton.querySelector('.db-backup-fab-button').style.background = detectedTool.color;
  }

  // Add event listeners
  const fabButton = floatingButton.querySelector('.db-backup-fab-button');
  const fabMenu = floatingButton.querySelector('.db-backup-fab-menu');
  const menuItems = floatingButton.querySelectorAll('.db-backup-fab-menu-item');

  fabButton.addEventListener('click', (e) => {
    e.stopPropagation();
    const isVisible = fabMenu.style.display !== 'none';
    fabMenu.style.display = isVisible ? 'none' : 'block';
    fabButton.classList.toggle('active');
  });

  // Close menu when clicking outside
  document.addEventListener('click', () => {
    if (fabMenu.style.display !== 'none') {
      fabMenu.style.display = 'none';
      fabButton.classList.remove('active');
    }
  });

  // Menu item actions
  menuItems.forEach((item) => {
    item.addEventListener('click', async (e) => {
      e.stopPropagation();
      const action = item.dataset.action;
      await handleAction(action);
      fabMenu.style.display = 'none';
      fabButton.classList.remove('active');
    });
  });

  isInjected = true;
  console.log('[DB Backup] Floating button injected');
}

function injectStyles() {
  if (document.getElementById('db-backup-styles')) {
    return;
  }

  const styles = document.createElement('style');
  styles.id = 'db-backup-styles';
  styles.textContent = `
    #db-backup-fab {
      position: fixed;
      bottom: 24px;
      right: 24px;
      z-index: 999999;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    }

    .db-backup-fab-button {
      width: 56px;
      height: 56px;
      border-radius: 50%;
      background: #3b82f6;
      color: white;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .db-backup-fab-button:hover {
      transform: scale(1.1);
      box-shadow: 0 6px 16px rgba(0, 0, 0, 0.4);
    }

    .db-backup-fab-button.active {
      transform: rotate(45deg);
    }

    .db-backup-fab-menu {
      position: absolute;
      bottom: 70px;
      right: 0;
      background: white;
      border-radius: 12px;
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
      padding: 8px;
      min-width: 200px;
      animation: slideUp 0.2s ease-out;
    }

    @keyframes slideUp {
      from {
        opacity: 0;
        transform: translateY(10px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .db-backup-fab-menu-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      border-radius: 8px;
      cursor: pointer;
      color: #1f2937;
      font-size: 14px;
      font-weight: 500;
      transition: background 0.2s;
    }

    .db-backup-fab-menu-item:hover {
      background: #f3f4f6;
    }

    .db-backup-fab-menu-item svg {
      flex-shrink: 0;
    }

    .db-backup-page-indicator {
      position: fixed;
      top: 16px;
      right: 16px;
      background: rgba(59, 130, 246, 0.95);
      color: white;
      padding: 8px 16px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      z-index: 999998;
      display: flex;
      align-items: center;
      gap: 8px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
      animation: slideIn 0.3s ease-out;
    }

    @keyframes slideIn {
      from {
        opacity: 0;
        transform: translateX(20px);
      }
      to {
        opacity: 1;
        transform: translateX(0);
      }
    }

    .db-backup-notification {
      position: fixed;
      top: 24px;
      left: 50%;
      transform: translateX(-50%);
      background: white;
      color: #1f2937;
      padding: 16px 24px;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 500;
      z-index: 9999999;
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
      animation: slideDown 0.3s ease-out;
      display: flex;
      align-items: center;
      gap: 12px;
    }

    @keyframes slideDown {
      from {
        opacity: 0;
        transform: translate(-50%, -20px);
      }
      to {
        opacity: 1;
        transform: translate(-50%, 0);
      }
    }

    .db-backup-notification.success {
      background: #10b981;
      color: white;
    }

    .db-backup-notification.error {
      background: #ef4444;
      color: white;
    }
  `;

  document.head.appendChild(styles);
}

// ============================================================================
// Page Indicators
// ============================================================================

function addPageIndicators() {
  if (!detectedTool) {
    return;
  }

  // Add indicator at top of page
  const indicator = document.createElement('div');
  indicator.className = 'db-backup-page-indicator';
  indicator.innerHTML = `
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
      <circle cx="12" cy="7" r="4"></circle>
    </svg>
    <span>${detectedTool.name} detected</span>
  `;

  indicator.style.background = detectedTool.color;
  document.body.appendChild(indicator);

  // Remove after 5 seconds
  setTimeout(() => {
    indicator.style.opacity = '0';
    setTimeout(() => indicator.remove(), 300);
  }, 5000);
}

// ============================================================================
// Action Handlers
// ============================================================================

async function handleAction(action) {
  console.log(`[DB Backup] Handling action: ${action}`);

  switch (action) {
    case 'quick-backup':
      await handleQuickBackup();
      break;

    case 'scheduled-backup':
      await handleScheduledBackup();
      break;

    case 'view-backups':
      openDashboard('/backups');
      break;

    case 'open-dashboard':
      openDashboard('/');
      break;

    default:
      console.warn(`[DB Backup] Unknown action: ${action}`);
  }
}

async function handleQuickBackup() {
  showNotification('Starting quick backup...', 'info');

  try {
    // Get current page info
    const pageInfo = extractPageInfo();

    // Send message to background script
    const response = await sendMessageToBackground({
      action: 'quickBackup',
      data: {
        source: 'content-script',
        tool: detectedTool?.name,
        url: window.location.href,
        pageInfo,
      },
    });

    if (response.success) {
      showNotification('Backup started successfully!', 'success');
    } else {
      showNotification(response.error || 'Failed to start backup', 'error');
    }
  } catch (error) {
    console.error('[DB Backup] Quick backup failed:', error);
    showNotification('Failed to start backup', 'error');
  }
}

async function handleScheduledBackup() {
  // Open popup to schedule backup
  openDashboard('/backups/schedule');
}

function extractPageInfo() {
  const info = {
    title: document.title,
    url: window.location.href,
    hostname: window.location.hostname,
    pathname: window.location.pathname,
    tool: detectedTool?.name,
  };

  // Try to extract database name from common patterns
  if (detectedTool) {
    // phpMyAdmin
    if (detectedTool.name === 'phpMyAdmin') {
      const match = window.location.href.match(/db=([^&]+)/);
      if (match) {
        info.database = decodeURIComponent(match[1]);
      }
    }

    // Adminer
    if (detectedTool.name === 'Adminer') {
      const match = window.location.href.match(/[?&]db=([^&]+)/);
      if (match) {
        info.database = decodeURIComponent(match[1]);
      }
    }

    // pgAdmin - check page title
    if (detectedTool.name === 'pgAdmin') {
      const match = document.title.match(/- ([^-]+?) -/);
      if (match) {
        info.database = match[1].trim();
      }
    }

    // Mongo Express - check URL
    if (detectedTool.name === 'Mongo Express') {
      const match = window.location.pathname.match(/\/db\/([^/]+)/);
      if (match) {
        info.database = match[1];
      }
    }
  }

  return info;
}

// ============================================================================
// Notifications
// ============================================================================

function showNotification(message, type = 'info') {
  const notification = document.createElement('div');
  notification.className = `db-backup-notification ${type}`;

  let icon = '';
  switch (type) {
    case 'success':
      icon = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
      break;
    case 'error':
      icon = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>';
      break;
    default:
      icon = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>';
  }

  notification.innerHTML = `${icon}<span>${message}</span>`;
  document.body.appendChild(notification);

  // Remove after 4 seconds
  setTimeout(() => {
    notification.style.opacity = '0';
    setTimeout(() => notification.remove(), 300);
  }, 4000);
}

// ============================================================================
// Utilities
// ============================================================================

function openDashboard(path = '/') {
  const url = `${settings.apiUrl}${path}`;
  window.open(url, '_blank');
}

async function sendMessageToBackground(message) {
  const browserAPI = getBrowserAPI();

  try {
    if (browserAPI.runtime.sendMessage.length === 1) {
      // Chrome - returns promise
      return await browserAPI.runtime.sendMessage(message);
    } else {
      // Firefox - needs callback
      return await new Promise((resolve, reject) => {
        browserAPI.runtime.sendMessage(message, (response) => {
          if (browserAPI.runtime.lastError) {
            reject(browserAPI.runtime.lastError);
          } else {
            resolve(response);
          }
        });
      });
    }
  } catch (error) {
    console.error('[DB Backup] Failed to send message:', error);
    return { success: false, error: error.message };
  }
}

function getBrowserAPI() {
  return typeof chrome !== 'undefined' ? chrome : browser;
}

// ============================================================================
// Page Change Observer
// ============================================================================

function observePageChanges() {
  // Observe DOM changes for single-page applications
  const observer = new MutationObserver((mutations) => {
    // Re-detect tool if not already detected
    if (!detectedTool) {
      detectDatabaseTool();
      if (detectedTool && !isInjected) {
        injectFloatingButton();
        addPageIndicators();
      }
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  // Also listen for URL changes (for SPAs)
  let lastUrl = window.location.href;
  setInterval(() => {
    const currentUrl = window.location.href;
    if (currentUrl !== lastUrl) {
      lastUrl = currentUrl;
      console.log('[DB Backup] URL changed, re-detecting...');

      // Reset detection
      detectedTool = null;
      detectDatabaseTool();

      if (detectedTool && !isInjected) {
        injectFloatingButton();
        addPageIndicators();
      }
    }
  }, 1000);
}

// ============================================================================
// Message Listener
// ============================================================================

function setupMessageListener() {
  const browserAPI = getBrowserAPI();

  browserAPI.runtime.onMessage.addListener((message, sender, sendResponse) => {
    console.log('[DB Backup] Content script received message:', message);

    switch (message.action) {
      case 'getPageInfo':
        sendResponse({
          success: true,
          data: {
            detectedTool: detectedTool?.name,
            pageInfo: extractPageInfo(),
          },
        });
        break;

      case 'showNotification':
        showNotification(message.message, message.type);
        sendResponse({ success: true });
        break;

      case 'updateSettings':
        loadSettings().then(() => {
          sendResponse({ success: true });
        });
        return true; // Keep channel open for async response

      default:
        sendResponse({ success: false, error: 'Unknown action' });
    }

    return false;
  });
}

// ============================================================================
// Cleanup
// ============================================================================

window.addEventListener('beforeunload', () => {
  // Clean up before page unload
  if (floatingButton) {
    floatingButton.remove();
  }
});

console.log('[DB Backup] Content script loaded');
