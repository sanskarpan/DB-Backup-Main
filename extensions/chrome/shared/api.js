/**
 * Shared API wrapper for DB Backup Manager extensions
 * Works across Chrome, Firefox, Edge, and Safari
 */

class BackupAPI {
  constructor(baseURL) {
    this.baseURL = baseURL || 'http://localhost:8080';
    this.apiKey = null;
  }

  /**
   * Set API key for authentication
   */
  setApiKey(key) {
    this.apiKey = key;
  }

  /**
   * Set base URL for API
   */
  setBaseURL(url) {
    this.baseURL = url.replace(/\/$/, ''); // Remove trailing slash
  }

  /**
   * Get default headers
   */
  getHeaders() {
    const headers = {
      'Content-Type': 'application/json',
    };

    if (this.apiKey) {
      headers['Authorization'] = `Bearer ${this.apiKey}`;
    }

    return headers;
  }

  /**
   * Unwrap the backend response envelope.
   * Most endpoints return { success, data }, a few return raw payloads.
   * When `key` is provided, dig one level into the data object for that key.
   */
  unwrap(res, key) {
    let data = res;
    if (res && typeof res === 'object' && 'data' in res) {
      data = res.data;
    }
    if (key && data && typeof data === 'object' && key in data) {
      return data[key];
    }
    return data;
  }

  /**
   * Make API request
   */
  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    const config = {
      method: options.method || 'GET',
      headers: this.getHeaders(),
      ...options,
    };

    if (options.body && typeof options.body === 'object') {
      config.body = JSON.stringify(options.body);
    }

    try {
      const response = await fetch(url, config);

      if (!response.ok) {
        const error = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(error.message || `API request failed: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // ============================================================================
  // Authentication
  // ============================================================================

  /**
   * Log in with username/password and obtain a JWT.
   * Backend: POST /api/v1/auth/login -> { token, user }
   * On success the returned token is stored as the API key for this client.
   */
  async login(username, password) {
    const data = await this.request('/api/v1/auth/login', {
      method: 'POST',
      body: { username, password },
    });

    if (data && data.token) {
      this.setApiKey(data.token);
    }

    return data;
  }

  // ============================================================================
  // Backup Operations
  // ============================================================================

  /**
   * List all backups.
   * Backend returns { success, data: { backups: [...], total } };
   * we normalize to a plain array for callers.
   */
  async listBackups(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/api/v1/backups${queryString ? '?' + queryString : ''}`;
    const res = await this.request(endpoint);
    return this.unwrap(res, 'backups') || [];
  }

  /**
   * Get backup details
   */
  async getBackup(id) {
    const res = await this.request(`/api/v1/backups/${id}`);
    return this.unwrap(res);
  }

  /**
   * Create a new backup
   */
  async createBackup(data) {
    const res = await this.request('/api/v1/backups', {
      method: 'POST',
      body: data,
    });
    return this.unwrap(res);
  }

  /**
   * Delete a backup
   */
  async deleteBackup(id) {
    return this.request(`/api/v1/backups/${id}`, {
      method: 'DELETE',
    });
  }

  /**
   * Download a backup
   */
  async downloadBackup(id) {
    const url = `${this.baseURL}/api/v1/backups/${id}/download`;
    return url; // Return URL for download
  }

  // ============================================================================
  // Database Operations
  // ============================================================================

  /**
   * List databases (backend returns a raw array).
   */
  async listDatabases() {
    const res = await this.request('/api/v1/databases');
    return Array.isArray(res) ? res : this.unwrap(res, 'databases') || [];
  }

  /**
   * Get database details
   */
  async getDatabase(id) {
    const res = await this.request(`/api/v1/databases/${id}`);
    return this.unwrap(res);
  }

  /**
   * Test database connection
   */
  async testConnection(id) {
    return this.request(`/api/v1/databases/${id}/test`, {
      method: 'POST',
    });
  }

  // ============================================================================
  // Monitoring Operations
  //
  // The backend does not expose /monitoring/* routes. Overall health/metrics
  // are derived from the real /stats endpoint, and alerts come from
  // /security/alerts. All of these degrade gracefully (never throw) so the
  // popup and background sync do not perpetually error.
  // ============================================================================

  /**
   * Get monitoring status.
   * No dedicated route exists; we surface aggregate stats from /stats instead.
   */
  async getMonitoringStatus() {
    try {
      return await this.request('/api/v1/stats');
    } catch (error) {
      console.warn('getMonitoringStatus unavailable:', error.message);
      return null;
    }
  }

  /**
   * Get metrics.
   * No /monitoring/metrics route exists; fall back to storage stats.
   */
  async getMetrics() {
    try {
      return await this.request('/api/v1/stats/storage');
    } catch (error) {
      console.warn('getMetrics unavailable:', error.message);
      return null;
    }
  }

  /**
   * Get alerts. Backend exposes security threat alerts at /security/alerts,
   * shaped as { success, data: { alerts: [...] } }. Returns an array and
   * never throws so callers can safely .filter() over the result.
   */
  async getAlerts() {
    try {
      const res = await this.request('/api/v1/security/alerts');
      return this.unwrap(res, 'alerts') || [];
    } catch (error) {
      console.warn('getAlerts unavailable:', error.message);
      return [];
    }
  }

  // ============================================================================
  // Schedule Operations
  // ============================================================================

  /**
   * List schedules
   */
  async listSchedules() {
    return this.request('/api/v1/schedules');
  }

  /**
   * Create schedule
   */
  async createSchedule(data) {
    return this.request('/api/v1/schedules', {
      method: 'POST',
      body: data,
    });
  }

  /**
   * Update schedule
   */
  async updateSchedule(id, data) {
    return this.request(`/api/v1/schedules/${id}`, {
      method: 'PUT',
      body: data,
    });
  }

  /**
   * Delete schedule
   */
  async deleteSchedule(id) {
    return this.request(`/api/v1/schedules/${id}`, {
      method: 'DELETE',
    });
  }

  // ============================================================================
  // Compliance Operations
  //
  // The backend does not expose /compliance/* routes for the extension.
  // These return null gracefully rather than throwing on a 404.
  // ============================================================================

  /**
   * Get compliance status (no backend route; returns null).
   */
  async getComplianceStatus() {
    console.warn('getComplianceStatus: no backend route available');
    return null;
  }

  /**
   * Get compliance reports (no backend route; returns empty list).
   */
  async getComplianceReports() {
    console.warn('getComplianceReports: no backend route available');
    return [];
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get dashboard stats.
   * Backend: GET /api/v1/stats -> { total_backups, successful_backups,
   * failed_backups, total_size, databases, schedules }
   */
  async getDashboardStats() {
    const res = await this.request('/api/v1/stats');
    return this.unwrap(res);
  }

  /**
   * Get backup/storage statistics.
   * Backend: GET /api/v1/stats/storage
   */
  async getBackupStats() {
    const res = await this.request('/api/v1/stats/storage');
    return this.unwrap(res);
  }
}

// Export for use in extensions
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BackupAPI;
}

export { BackupAPI };
