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
  // Backup Operations
  // ============================================================================

  /**
   * List all backups
   */
  async listBackups(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/api/backups${queryString ? '?' + queryString : ''}`;
    return this.request(endpoint);
  }

  /**
   * Get backup details
   */
  async getBackup(id) {
    return this.request(`/api/backups/${id}`);
  }

  /**
   * Create a new backup
   */
  async createBackup(data) {
    return this.request('/api/backups', {
      method: 'POST',
      body: data,
    });
  }

  /**
   * Delete a backup
   */
  async deleteBackup(id) {
    return this.request(`/api/backups/${id}`, {
      method: 'DELETE',
    });
  }

  /**
   * Download a backup
   */
  async downloadBackup(id) {
    const url = `${this.baseURL}/api/backups/${id}/download`;
    return url; // Return URL for download
  }

  // ============================================================================
  // Database Operations
  // ============================================================================

  /**
   * List databases
   */
  async listDatabases() {
    return this.request('/api/databases');
  }

  /**
   * Get database details
   */
  async getDatabase(id) {
    return this.request(`/api/databases/${id}`);
  }

  /**
   * Test database connection
   */
  async testConnection(id) {
    return this.request(`/api/databases/${id}/test`, {
      method: 'POST',
    });
  }

  // ============================================================================
  // Monitoring Operations
  // ============================================================================

  /**
   * Get monitoring status
   */
  async getMonitoringStatus() {
    return this.request('/api/monitoring/status');
  }

  /**
   * Get metrics
   */
  async getMetrics(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/api/monitoring/metrics${queryString ? '?' + queryString : ''}`;
    return this.request(endpoint);
  }

  /**
   * Get alerts
   */
  async getAlerts() {
    return this.request('/api/monitoring/alerts');
  }

  // ============================================================================
  // Schedule Operations
  // ============================================================================

  /**
   * List schedules
   */
  async listSchedules() {
    return this.request('/api/schedules');
  }

  /**
   * Create schedule
   */
  async createSchedule(data) {
    return this.request('/api/schedules', {
      method: 'POST',
      body: data,
    });
  }

  /**
   * Update schedule
   */
  async updateSchedule(id, data) {
    return this.request(`/api/schedules/${id}`, {
      method: 'PUT',
      body: data,
    });
  }

  /**
   * Delete schedule
   */
  async deleteSchedule(id) {
    return this.request(`/api/schedules/${id}`, {
      method: 'DELETE',
    });
  }

  // ============================================================================
  // Compliance Operations
  // ============================================================================

  /**
   * Get compliance status
   */
  async getComplianceStatus() {
    return this.request('/api/compliance/status');
  }

  /**
   * Get compliance reports
   */
  async getComplianceReports() {
    return this.request('/api/compliance/reports');
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get dashboard stats
   */
  async getDashboardStats() {
    return this.request('/api/stats/dashboard');
  }

  /**
   * Get backup statistics
   */
  async getBackupStats(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = `/api/stats/backups${queryString ? '?' + queryString : ''}`;
    return this.request(endpoint);
  }
}

// Export for use in extensions
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BackupAPI;
}
