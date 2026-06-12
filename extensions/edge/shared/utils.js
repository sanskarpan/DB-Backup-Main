/**
 * Shared utilities for DB Backup Manager extensions
 * Cross-browser compatible
 */

const Utils = {
  /**
   * Get browser API (works across Chrome, Firefox, Edge, Safari)
   */
  getBrowserAPI() {
    if (typeof browser !== 'undefined') {
      return browser; // Firefox, Safari
    } else if (typeof chrome !== 'undefined') {
      return chrome; // Chrome, Edge
    }
    throw new Error('Browser API not available');
  },

  /**
   * Promisify Chrome API calls for consistency
   */
  promisify(fn, ...args) {
    return new Promise((resolve, reject) => {
      fn(...args, (result) => {
        if (chrome.runtime.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve(result);
        }
      });
    });
  },

  /**
   * Get storage value
   */
  async getStorage(key) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.storage.local.get.length === 1) {
      // Promise-based (Firefox, modern browsers)
      const result = await browserAPI.storage.local.get(key);
      return key ? result[key] : result;
    } else {
      // Callback-based (Chrome, Edge)
      return this.promisify(browserAPI.storage.local.get.bind(browserAPI.storage.local), key);
    }
  },

  /**
   * Set storage value
   */
  async setStorage(key, value) {
    const browserAPI = this.getBrowserAPI();
    const data = typeof key === 'object' ? key : { [key]: value };

    if (browserAPI.storage.local.set.length === 1) {
      return await browserAPI.storage.local.set(data);
    } else {
      return this.promisify(browserAPI.storage.local.set.bind(browserAPI.storage.local), data);
    }
  },

  /**
   * Remove storage value
   */
  async removeStorage(key) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.storage.local.remove.length === 1) {
      return await browserAPI.storage.local.remove(key);
    } else {
      return this.promisify(browserAPI.storage.local.remove.bind(browserAPI.storage.local), key);
    }
  },

  /**
   * Send message to background script
   */
  async sendMessage(message) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.runtime.sendMessage.length === 1) {
      return await browserAPI.runtime.sendMessage(message);
    } else {
      return this.promisify(browserAPI.runtime.sendMessage.bind(browserAPI.runtime), message);
    }
  },

  /**
   * Create notification
   */
  async createNotification(id, options) {
    const browserAPI = this.getBrowserAPI();

    const notificationOptions = {
      type: options.type || 'basic',
      iconUrl: options.iconUrl || '../icons/icon-128.png',
      title: options.title,
      message: options.message,
      priority: options.priority || 1,
    };

    if (browserAPI.notifications.create.length === 2) {
      return await browserAPI.notifications.create(id, notificationOptions);
    } else {
      return this.promisify(
        browserAPI.notifications.create.bind(browserAPI.notifications),
        id,
        notificationOptions
      );
    }
  },

  /**
   * Update badge text
   */
  setBadgeText(text) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.action) {
      // Manifest V3 (Chrome, Edge)
      browserAPI.action.setBadgeText({ text: String(text) });
    } else if (browserAPI.browserAction) {
      // Manifest V2 (Firefox, older browsers)
      browserAPI.browserAction.setBadgeText({ text: String(text) });
    }
  },

  /**
   * Update badge background color
   */
  setBadgeBackgroundColor(color) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.action) {
      browserAPI.action.setBadgeBackgroundColor({ color });
    } else if (browserAPI.browserAction) {
      browserAPI.browserAction.setBadgeBackgroundColor({ color });
    }
  },

  /**
   * Open new tab
   */
  async openTab(url) {
    const browserAPI = this.getBrowserAPI();

    if (browserAPI.tabs.create.length === 1) {
      return await browserAPI.tabs.create({ url });
    } else {
      return this.promisify(browserAPI.tabs.create.bind(browserAPI.tabs), { url });
    }
  },

  /**
   * Format bytes to human readable
   */
  formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  },

  /**
   * Format date
   */
  formatDate(date) {
    const d = new Date(date);
    const now = new Date();
    const diff = now - d;

    // Less than 1 minute
    if (diff < 60000) {
      return 'Just now';
    }

    // Less than 1 hour
    if (diff < 3600000) {
      const minutes = Math.floor(diff / 60000);
      return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    }

    // Less than 24 hours
    if (diff < 86400000) {
      const hours = Math.floor(diff / 3600000);
      return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    }

    // Less than 7 days
    if (diff < 604800000) {
      const days = Math.floor(diff / 86400000);
      return `${days} day${days > 1 ? 's' : ''} ago`;
    }

    // Full date
    return d.toLocaleDateString();
  },

  /**
   * Format duration
   */
  formatDuration(ms) {
    const seconds = Math.floor(ms / 1000);

    if (seconds < 60) {
      return `${seconds}s`;
    }

    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) {
      return `${minutes}m ${seconds % 60}s`;
    }

    const hours = Math.floor(minutes / 60);
    return `${hours}h ${minutes % 60}m`;
  },

  /**
   * Get status color
   */
  getStatusColor(status) {
    const colors = {
      success: '#10b981',
      completed: '#10b981',
      running: '#3b82f6',
      pending: '#f59e0b',
      failed: '#ef4444',
      error: '#ef4444',
      warning: '#f59e0b',
    };

    return colors[status.toLowerCase()] || '#6b7280';
  },

  /**
   * Get status icon
   */
  getStatusIcon(status) {
    const icons = {
      success: '✓',
      completed: '✓',
      running: '⟳',
      pending: '○',
      failed: '✗',
      error: '✗',
      warning: '⚠',
    };

    return icons[status.toLowerCase()] || '○';
  },

  /**
   * Debounce function
   */
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  },

  /**
   * Throttle function
   */
  throttle(func, limit) {
    let inThrottle;
    return function(...args) {
      if (!inThrottle) {
        func.apply(this, args);
        inThrottle = true;
        setTimeout(() => (inThrottle = false), limit);
      }
    };
  },

  /**
   * Validate URL
   */
  isValidURL(string) {
    try {
      new URL(string);
      return true;
    } catch {
      return false;
    }
  },

  /**
   * Generate unique ID
   */
  generateId() {
    return Date.now().toString(36) + Math.random().toString(36).substring(2);
  },

  /**
   * Copy to clipboard
   */
  async copyToClipboard(text) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch (error) {
      console.error('Failed to copy:', error);
      return false;
    }
  },

  /**
   * Log with timestamp
   */
  log(message, ...args) {
    const timestamp = new Date().toISOString();
    console.log(`[${timestamp}]`, message, ...args);
  },

  /**
   * Error handler
   */
  handleError(error, context = '') {
    console.error(`Error in ${context}:`, error);

    this.createNotification('error', {
      type: 'basic',
      title: 'DB Backup Manager Error',
      message: error.message || 'An error occurred',
      priority: 2,
    });
  },
};

// Export for use in extensions
if (typeof module !== 'undefined' && module.exports) {
  module.exports = Utils;
}
