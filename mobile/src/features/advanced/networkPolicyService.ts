/**
 * Network Policy Service
 * Manages cellular vs WiFi backup policies and network-aware operations
 */

import NetInfo, {
  NetInfoState,
  NetInfoStateType,
} from '@react-native-community/netinfo';
import AsyncStorage from '@react-native-async-storage/async-storage';

export interface NetworkPolicy {
  allowCellular: boolean;
  cellularSizeLimit: number; // bytes
  wifiOnly: boolean;
  allow5G: boolean;
  allowLTE: boolean;
  allow3G: boolean;
  allowRoaming: boolean;
  minSignalStrength: number;
}

export interface NetworkStatus {
  type: NetInfoStateType;
  isConnected: boolean;
  isMetered: boolean | null;
  isInternetReachable: boolean | null;
  details: any;
}

export interface NetworkDecision {
  allowed: boolean;
  reason: string;
  shouldThrottle: boolean;
  throttleRate?: number; // bytes per second
  networkType: string;
}

class NetworkPolicyService {
  private policy: NetworkPolicy;
  private currentStatus: NetworkStatus | null = null;
  private listeners: ((status: NetworkStatus) => void)[] = [];
  private unsubscribe: (() => void) | null = null;

  constructor() {
    this.policy = {
      allowCellular: false,
      cellularSizeLimit: 100 * 1024 * 1024, // 100 MB
      wifiOnly: true,
      allow5G: true,
      allowLTE: true,
      allow3G: false,
      allowRoaming: false,
      minSignalStrength: 20,
    };
  }

  /**
   * Initialize network monitoring
   */
  async initialize(): Promise<void> {
    await this.loadPolicy();

    // Subscribe to network state changes
    this.unsubscribe = NetInfo.addEventListener(state => {
      this.updateNetworkStatus(state);
    });

    // Get initial state
    const state = await NetInfo.fetch();
    this.updateNetworkStatus(state);
  }

  /**
   * Clean up network monitoring
   */
  dispose(): void {
    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = null;
    }
  }

  /**
   * Update network status from NetInfo state
   */
  private updateNetworkStatus(state: NetInfoState): void {
    this.currentStatus = {
      type: state.type,
      isConnected: state.isConnected ?? false,
      isMetered: state.isInternetReachable,
      isInternetReachable: state.isInternetReachable,
      details: state.details,
    };

    // Notify listeners
    this.listeners.forEach(listener => {
      if (this.currentStatus) {
        listener(this.currentStatus);
      }
    });
  }

  /**
   * Subscribe to network status changes
   */
  subscribe(callback: (status: NetworkStatus) => void): () => void {
    this.listeners.push(callback);

    // Send current status immediately
    if (this.currentStatus) {
      callback(this.currentStatus);
    }

    // Return unsubscribe function
    return () => {
      this.listeners = this.listeners.filter(l => l !== callback);
    };
  }

  /**
   * Check if backup can be performed with current network
   */
  async canPerformBackup(backupSize: number): Promise<NetworkDecision> {
    if (!this.currentStatus) {
      return {
        allowed: false,
        reason: 'Network status not available',
        shouldThrottle: false,
        networkType: 'unknown',
      };
    }

    // No network connection
    if (!this.currentStatus.isConnected) {
      return {
        allowed: false,
        reason: 'No network connection',
        shouldThrottle: false,
        networkType: this.currentStatus.type,
      };
    }

    const networkType = this.currentStatus.type;

    // WiFi or Ethernet - always allowed
    if (networkType === 'wifi' || networkType === 'ethernet') {
      return {
        allowed: true,
        reason: 'Connected to WiFi/Ethernet',
        shouldThrottle: false,
        networkType,
      };
    }

    // WiFi-only mode
    if (this.policy.wifiOnly) {
      return {
        allowed: false,
        reason: 'WiFi-only mode enabled, currently on cellular',
        shouldThrottle: false,
        networkType,
      };
    }

    // Cellular backups not allowed
    if (!this.policy.allowCellular) {
      return {
        allowed: false,
        reason: 'Cellular backups are disabled',
        shouldThrottle: false,
        networkType,
      };
    }

    // Check cellular data type restrictions
    if (networkType === '3g' && !this.policy.allow3G) {
      return {
        allowed: false,
        reason: '3G backups are disabled',
        shouldThrottle: false,
        networkType,
      };
    }

    // Check size limit for cellular
    if (backupSize > this.policy.cellularSizeLimit) {
      return {
        allowed: false,
        reason: `Backup size (${this.formatBytes(backupSize)}) exceeds cellular limit (${this.formatBytes(this.policy.cellularSizeLimit)})`,
        shouldThrottle: false,
        networkType,
      };
    }

    // Check roaming
    if (this.currentStatus.details?.isConnectionExpensive && !this.policy.allowRoaming) {
      return {
        allowed: false,
        reason: 'Device is roaming',
        shouldThrottle: false,
        networkType,
      };
    }

    // Allow with throttling on cellular
    const throttleRate = this.getThrottleRate(networkType);
    return {
      allowed: true,
      reason: 'Cellular backup allowed',
      shouldThrottle: true,
      throttleRate,
      networkType,
    };
  }

  /**
   * Get recommended throttle rate based on network type
   */
  private getThrottleRate(networkType: string): number {
    switch (networkType) {
      case '5g':
        return 10 * 1024 * 1024; // 10 MB/s
      case '4g':
      case 'lte':
        return 5 * 1024 * 1024; // 5 MB/s
      case '3g':
        return 1 * 1024 * 1024; // 1 MB/s
      default:
        return 2 * 1024 * 1024; // 2 MB/s
    }
  }

  /**
   * Wait for optimal network conditions
   */
  async waitForOptimalNetwork(timeoutMs: number = 60000): Promise<boolean> {
    return new Promise(resolve => {
      const timeout = setTimeout(() => {
        unsubscribe();
        resolve(false);
      }, timeoutMs);

      const unsubscribe = this.subscribe(status => {
        if (status.type === 'wifi' || status.type === 'ethernet') {
          clearTimeout(timeout);
          unsubscribe();
          resolve(true);
        }
      });
    });
  }

  /**
   * Get current network status
   */
  getCurrentStatus(): NetworkStatus | null {
    return this.currentStatus;
  }

  /**
   * Update network policy
   */
  async updatePolicy(policy: Partial<NetworkPolicy>): Promise<void> {
    this.policy = {...this.policy, ...policy};
    await this.savePolicy();
  }

  /**
   * Get current policy
   */
  getPolicy(): NetworkPolicy {
    return {...this.policy};
  }

  /**
   * Save policy to storage
   */
  private async savePolicy(): Promise<void> {
    try {
      await AsyncStorage.setItem(
        'network_policy',
        JSON.stringify(this.policy),
      );
    } catch (error) {
      console.error('Failed to save network policy:', error);
    }
  }

  /**
   * Load policy from storage
   */
  private async loadPolicy(): Promise<void> {
    try {
      const saved = await AsyncStorage.getItem('network_policy');
      if (saved) {
        this.policy = JSON.parse(saved);
      }
    } catch (error) {
      console.error('Failed to load network policy:', error);
    }
  }

  /**
   * Format bytes to human-readable string
   */
  private formatBytes(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
    if (bytes < 1024 * 1024 * 1024)
      return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
  }

  /**
   * Get network statistics
   */
  getNetworkStats(): {
    currentType: string;
    isConnected: boolean;
    isMetered: boolean;
    isOptimal: boolean;
  } {
    return {
      currentType: this.currentStatus?.type || 'unknown',
      isConnected: this.currentStatus?.isConnected || false,
      isMetered: this.currentStatus?.isMetered !== false,
      isOptimal:
        this.currentStatus?.type === 'wifi' ||
        this.currentStatus?.type === 'ethernet',
    };
  }
}

export const networkPolicyService = new NetworkPolicyService();
