/**
 * Privacy-Focused Analytics for DB Backup Manager Extension
 *
 * This module tracks basic usage metrics without collecting personal information.
 * All data is anonymized and stored locally. Users can opt-out in settings.
 *
 * Tracked Events:
 * - Extension installed/updated
 * - Features used (backup, restore, monitoring)
 * - Errors encountered
 * - Performance metrics
 *
 * NOT Tracked:
 * - Personal information
 * - URLs visited
 * - Database names or content
 * - API keys or credentials
 */

const Analytics = {
  // Configuration
  config: {
    enabled: true,
    endpoint: null, // Optional remote endpoint
    localOnly: true, // Store only locally by default
    anonymousId: null, // Generated on first use
    sessionId: null, // Generated per session
  },

  // Initialize analytics
  async init() {
    try {
      // Load settings
      const settings = await this.getStorage('analytics_settings');
      if (settings) {
        this.config = { ...this.config, ...settings };
      }

      // Check if analytics is enabled
      const appSettings = await this.getStorage('settings');
      if (appSettings && appSettings.analyticsEnabled === false) {
        this.config.enabled = false;
        return;
      }

      // Generate anonymous ID if not exists
      if (!this.config.anonymousId) {
        this.config.anonymousId = this.generateId();
        await this.saveConfig();
      }

      // Generate session ID
      this.config.sessionId = this.generateId();

      // Track extension start
      await this.track('extension_started', {
        version: this.getVersion(),
        browser: this.getBrowser(),
      });

      console.log('[Analytics] Initialized');
    } catch (error) {
      console.error('[Analytics] Initialization failed:', error);
    }
  },

  // Track an event
  async track(eventName, properties = {}) {
    if (!this.config.enabled) {
      return;
    }

    try {
      const event = {
        id: this.generateId(),
        name: eventName,
        properties: this.sanitizeProperties(properties),
        anonymousId: this.config.anonymousId,
        sessionId: this.config.sessionId,
        timestamp: new Date().toISOString(),
        version: this.getVersion(),
        browser: this.getBrowser(),
        platform: this.getPlatform(),
      };

      // Store locally
      await this.storeEvent(event);

      // Send to remote endpoint if configured
      if (!this.config.localOnly && this.config.endpoint) {
        await this.sendEvent(event);
      }

      console.log('[Analytics] Tracked:', eventName, properties);
    } catch (error) {
      console.error('[Analytics] Failed to track event:', error);
    }
  },

  // Track page view
  async trackPageView(page) {
    await this.track('page_viewed', { page });
  },

  // Track feature usage
  async trackFeature(feature, action, properties = {}) {
    await this.track('feature_used', {
      feature,
      action,
      ...properties,
    });
  },

  // Track error
  async trackError(error, context = {}) {
    await this.track('error_occurred', {
      error: error.message || String(error),
      stack: error.stack,
      context,
    });
  },

  // Track performance metric
  async trackPerformance(metric, value, unit = 'ms') {
    await this.track('performance_metric', {
      metric,
      value,
      unit,
    });
  },

  // Sanitize properties to remove sensitive data
  sanitizeProperties(properties) {
    const sanitized = {};

    for (const [key, value] of Object.entries(properties)) {
      // Skip sensitive keys
      if (this.isSensitiveKey(key)) {
        continue;
      }

      // Sanitize sensitive values
      if (typeof value === 'string') {
        sanitized[key] = this.sanitizeValue(value);
      } else if (typeof value === 'object' && value !== null) {
        sanitized[key] = this.sanitizeProperties(value);
      } else {
        sanitized[key] = value;
      }
    }

    return sanitized;
  },

  // Check if key is sensitive
  isSensitiveKey(key) {
    const sensitiveKeys = [
      'password',
      'apiKey',
      'token',
      'secret',
      'credential',
      'auth',
      'privateKey',
      'apiUrl', // URLs might contain sensitive info
      'email',
      'username',
      'database', // Database names might be sensitive
    ];

    return sensitiveKeys.some(sensitive =>
      key.toLowerCase().includes(sensitive.toLowerCase())
    );
  },

  // Sanitize value
  sanitizeValue(value) {
    // Redact URLs (keep domain only)
    if (value.startsWith('http://') || value.startsWith('https://')) {
      try {
        const url = new URL(value);
        return `${url.protocol}//${url.hostname}`;
      } catch (e) {
        return '[URL]';
      }
    }

    // Redact email addresses
    if (value.includes('@') && value.includes('.')) {
      return '[EMAIL]';
    }

    // Redact anything that looks like a token or key
    if (value.length > 20 && /^[A-Za-z0-9+/=_-]+$/.test(value)) {
      return '[TOKEN]';
    }

    return value;
  },

  // Store event locally
  async storeEvent(event) {
    try {
      const events = (await this.getStorage('analytics_events')) || [];

      // Add event
      events.push(event);

      // Keep only last 1000 events
      if (events.length > 1000) {
        events.splice(0, events.length - 1000);
      }

      await this.setStorage('analytics_events', events);
    } catch (error) {
      console.error('[Analytics] Failed to store event:', error);
    }
  },

  // Send event to remote endpoint
  async sendEvent(event) {
    if (!this.config.endpoint) {
      return;
    }

    try {
      await fetch(this.config.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(event),
      });
    } catch (error) {
      console.error('[Analytics] Failed to send event to remote:', error);
    }
  },

  // Get all stored events
  async getEvents(filter = {}) {
    const events = (await this.getStorage('analytics_events')) || [];

    if (Object.keys(filter).length === 0) {
      return events;
    }

    return events.filter(event => {
      for (const [key, value] of Object.entries(filter)) {
        if (event[key] !== value && event.properties[key] !== value) {
          return false;
        }
      }
      return true;
    });
  },

  // Get analytics summary
  async getSummary(days = 7) {
    const events = await this.getEvents();
    const since = new Date();
    since.setDate(since.getDate() - days);

    const recentEvents = events.filter(
      e => new Date(e.timestamp) > since
    );

    const summary = {
      totalEvents: recentEvents.length,
      uniqueSessions: new Set(recentEvents.map(e => e.sessionId)).size,
      eventCounts: {},
      features: {},
      errors: recentEvents.filter(e => e.name === 'error_occurred').length,
      lastActive: recentEvents.length > 0
        ? new Date(recentEvents[recentEvents.length - 1].timestamp)
        : null,
    };

    // Count event types
    recentEvents.forEach(event => {
      summary.eventCounts[event.name] =
        (summary.eventCounts[event.name] || 0) + 1;

      if (event.name === 'feature_used' && event.properties.feature) {
        summary.features[event.properties.feature] =
          (summary.features[event.properties.feature] || 0) + 1;
      }
    });

    return summary;
  },

  // Clear all analytics data
  async clear() {
    await this.setStorage('analytics_events', []);
    console.log('[Analytics] Data cleared');
  },

  // Export analytics data
  async export() {
    const events = await this.getEvents();
    const summary = await this.getSummary(30);

    return {
      events,
      summary,
      config: {
        ...this.config,
        anonymousId: '[REDACTED]', // Don't export ID
      },
      exportedAt: new Date().toISOString(),
    };
  },

  // Enable/disable analytics
  async setEnabled(enabled) {
    this.config.enabled = enabled;
    await this.saveConfig();

    if (enabled) {
      await this.track('analytics_enabled');
    } else {
      await this.track('analytics_disabled');
    }
  },

  // Save configuration
  async saveConfig() {
    await this.setStorage('analytics_settings', this.config);
  },

  // Utility: Generate ID
  generateId() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
      const r = (Math.random() * 16) | 0;
      const v = c === 'x' ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  },

  // Utility: Get extension version
  getVersion() {
    const browserAPI = this.getBrowserAPI();
    return browserAPI.runtime.getManifest().version;
  },

  // Utility: Get browser name
  getBrowser() {
    const userAgent = navigator.userAgent.toLowerCase();

    if (userAgent.includes('edg/')) return 'edge';
    if (userAgent.includes('chrome')) return 'chrome';
    if (userAgent.includes('firefox')) return 'firefox';
    if (userAgent.includes('safari')) return 'safari';

    return 'unknown';
  },

  // Utility: Get platform
  getPlatform() {
    const platform = navigator.platform.toLowerCase();

    if (platform.includes('mac')) return 'macos';
    if (platform.includes('win')) return 'windows';
    if (platform.includes('linux')) return 'linux';

    return 'unknown';
  },

  // Utility: Get browser API
  getBrowserAPI() {
    return typeof chrome !== 'undefined' ? chrome : browser;
  },

  // Utility: Get storage
  async getStorage(key) {
    const browserAPI = this.getBrowserAPI();

    try {
      if (browserAPI.storage.local.get.length === 0) {
        const data = await browserAPI.storage.local.get(key);
        return data[key];
      } else {
        return await new Promise((resolve) => {
          browserAPI.storage.local.get(key, (data) => {
            resolve(data[key]);
          });
        });
      }
    } catch (error) {
      console.error('[Analytics] Storage get failed:', error);
      return null;
    }
  },

  // Utility: Set storage
  async setStorage(key, value) {
    const browserAPI = this.getBrowserAPI();

    try {
      if (browserAPI.storage.local.set.length === 1) {
        await browserAPI.storage.local.set({ [key]: value });
      } else {
        await new Promise((resolve) => {
          browserAPI.storage.local.set({ [key]: value }, resolve);
        });
      }
    } catch (error) {
      console.error('[Analytics] Storage set failed:', error);
    }
  },
};

// Common event tracking functions for convenience
const AnalyticsEvents = {
  // Extension lifecycle
  installed: () => Analytics.track('extension_installed'),
  updated: (version) => Analytics.track('extension_updated', { version }),
  uninstalled: () => Analytics.track('extension_uninstalled'),

  // User actions
  quickBackup: () => Analytics.trackFeature('backup', 'quick_backup'),
  scheduledBackup: () => Analytics.trackFeature('backup', 'scheduled_backup'),
  viewBackups: () => Analytics.trackFeature('backups', 'view_list'),
  openDashboard: () => Analytics.trackFeature('dashboard', 'open'),
  openSettings: () => Analytics.trackFeature('settings', 'open'),

  // Settings
  settingsSaved: () => Analytics.trackFeature('settings', 'saved'),
  connectionTested: (success) =>
    Analytics.trackFeature('settings', 'connection_test', { success }),
  dataCleared: () => Analytics.trackFeature('settings', 'data_cleared'),

  // Notifications
  notificationShown: (type) =>
    Analytics.trackFeature('notifications', 'shown', { type }),
  notificationClicked: (type) =>
    Analytics.trackFeature('notifications', 'clicked', { type }),

  // Context menu
  contextMenuUsed: (action) =>
    Analytics.trackFeature('context_menu', 'used', { action }),

  // Content script
  toolDetected: (tool) =>
    Analytics.trackFeature('content_script', 'tool_detected', { tool }),
  floatingButtonClicked: () =>
    Analytics.trackFeature('content_script', 'floating_button_clicked'),

  // Errors
  apiError: (endpoint, status) =>
    Analytics.trackError(new Error(`API Error: ${status}`), { endpoint }),
  storageError: (operation) =>
    Analytics.trackError(new Error('Storage Error'), { operation }),
};

// Auto-initialize when loaded
if (typeof window !== 'undefined') {
  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      Analytics.init();
    });
  } else {
    Analytics.init();
  }
} else {
  // In background script, initialize immediately
  Analytics.init();
}

// Export for use in other modules (CommonJS)
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { Analytics, AnalyticsEvents };
}

// Export for use as an ES module (consistent with api.js / utils.js)
export { Analytics, AnalyticsEvents };
