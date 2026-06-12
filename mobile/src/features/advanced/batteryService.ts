/**
 * Battery Service
 * Manages battery-aware operations, low power mode, and adaptive battery (Android)
 */

import {Platform, NativeModules, NativeEventEmitter} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import DeviceInfo from 'react-native-device-info';

export interface BatteryStatus {
  level: number; // 0-100
  isCharging: boolean;
  isPlugged: boolean;
  lowPowerMode: boolean;
  temperature?: number;
  health?: string;
}

export interface PowerBudget {
  maxDuration: number; // milliseconds
  maxOperations: number;
  powerLevel: 'high' | 'medium' | 'low' | 'critical';
  canPerformBackup: boolean;
  reason: string;
}

export interface BatteryConfig {
  lowPowerThreshold: number; // percentage
  enableLowPowerMode: boolean;
  adaptiveBattery: boolean; // Android only
  deferWhenUnplugged: boolean;
  minBatteryForBackup: number;
}

class BatteryService {
  private status: BatteryStatus = {
    level: 100,
    isCharging: false,
    isPlugged: false,
    lowPowerMode: false,
  };

  private config: BatteryConfig = {
    lowPowerThreshold: 20,
    enableLowPowerMode: true,
    adaptiveBattery: Platform.OS === 'android',
    deferWhenUnplugged: false,
    minBatteryForBackup: 15,
  };

  private listeners: ((status: BatteryStatus) => void)[] = [];
  private batteryEmitter: NativeEventEmitter | null = null;

  /**
   * Initialize battery monitoring
   */
  async initialize(): Promise<void> {
    await this.loadConfig();
    await this.updateBatteryStatus();

    // Set up battery monitoring based on platform
    if (Platform.OS === 'ios') {
      this.setupIOSBatteryMonitoring();
    } else {
      this.setupAndroidBatteryMonitoring();
    }

    // Poll battery status periodically
    setInterval(() => {
      this.updateBatteryStatus();
    }, 30000); // Every 30 seconds
  }

  /**
   * Setup iOS battery monitoring
   */
  private setupIOSBatteryMonitoring(): void {
    // iOS battery monitoring would use native module
    // For now, we'll use DeviceInfo polling
  }

  /**
   * Setup Android battery monitoring
   */
  private setupAndroidBatteryMonitoring(): void {
    // Android battery monitoring would use native module
    // Can access BatteryManager via native bridge
    if (NativeModules.BatteryModule) {
      this.batteryEmitter = new NativeEventEmitter(
        NativeModules.BatteryModule,
      );
      this.batteryEmitter.addListener('BatteryChanged', (status: any) => {
        this.updateStatusFromNative(status);
      });
    }
  }

  /**
   * Update battery status
   */
  private async updateBatteryStatus(): Promise<void> {
    try {
      const level = await DeviceInfo.getBatteryLevel();
      const isCharging = await DeviceInfo.isCharging();
      const powerState = await DeviceInfo.getPowerState();

      this.status = {
        level: Math.round(level * 100),
        isCharging,
        isPlugged: isCharging,
        lowPowerMode: powerState.lowPowerMode || false,
      };

      // Notify listeners
      this.notifyListeners();
    } catch (error) {
      console.error('Failed to update battery status:', error);
    }
  }

  /**
   * Update status from native module
   */
  private updateStatusFromNative(nativeStatus: any): void {
    this.status = {
      ...this.status,
      ...nativeStatus,
    };
    this.notifyListeners();
  }

  /**
   * Subscribe to battery status changes
   */
  subscribe(callback: (status: BatteryStatus) => void): () => void {
    this.listeners.push(callback);

    // Send current status immediately
    callback(this.status);

    // Return unsubscribe function
    return () => {
      this.listeners = this.listeners.filter(l => l !== callback);
    };
  }

  /**
   * Notify all listeners
   */
  private notifyListeners(): void {
    this.listeners.forEach(listener => listener(this.status));
  }

  /**
   * Check if backup can be performed based on battery
   */
  canPerformBackup(): PowerBudget {
    // Always allow if charging
    if (this.status.isCharging && this.status.isPlugged) {
      return {
        maxDuration: 30 * 60 * 1000, // 30 minutes
        maxOperations: 100,
        powerLevel: 'high',
        canPerformBackup: true,
        reason: 'Device is charging',
      };
    }

    // Check minimum battery level
    if (this.status.level < this.config.minBatteryForBackup) {
      return {
        maxDuration: 0,
        maxOperations: 0,
        powerLevel: 'critical',
        canPerformBackup: false,
        reason: `Battery level (${this.status.level}%) below minimum (${this.config.minBatteryForBackup}%)`,
      };
    }

    // Respect low power mode
    if (
      this.config.enableLowPowerMode &&
      this.status.lowPowerMode
    ) {
      return {
        maxDuration: 2 * 60 * 1000, // 2 minutes
        maxOperations: 5,
        powerLevel: 'low',
        canPerformBackup: false,
        reason: 'Device in low power mode',
      };
    }

    // Battery level based decisions
    if (this.status.level > 50) {
      return {
        maxDuration: 10 * 60 * 1000, // 10 minutes
        maxOperations: 50,
        powerLevel: 'medium',
        canPerformBackup: true,
        reason: 'Battery level sufficient',
      };
    } else if (this.status.level > 20) {
      return {
        maxDuration: 5 * 60 * 1000, // 5 minutes
        maxOperations: 20,
        powerLevel: 'low',
        canPerformBackup: true,
        reason: 'Battery level moderate - limit operations',
      };
    } else {
      return {
        maxDuration: 2 * 60 * 1000, // 2 minutes
        maxOperations: 5,
        powerLevel: 'critical',
        canPerformBackup: false,
        reason: 'Battery level too low',
      };
    }
  }

  /**
   * Get current battery status
   */
  getStatus(): BatteryStatus {
    return {...this.status};
  }

  /**
   * Get power budget for operations
   */
  getPowerBudget(): PowerBudget {
    return this.canPerformBackup();
  }

  /**
   * Update battery configuration
   */
  async updateConfig(config: Partial<BatteryConfig>): Promise<void> {
    this.config = {...this.config, ...config};
    await this.saveConfig();
  }

  /**
   * Get current configuration
   */
  getConfig(): BatteryConfig {
    return {...this.config};
  }

  /**
   * Enable adaptive battery (Android)
   */
  async enableAdaptiveBattery(): Promise<void> {
    if (Platform.OS === 'android') {
      this.config.adaptiveBattery = true;
      await this.saveConfig();

      // Call native module to enable adaptive battery
      if (NativeModules.BatteryModule) {
        NativeModules.BatteryModule.enableAdaptiveBattery();
      }
    }
  }

  /**
   * Disable adaptive battery (Android)
   */
  async disableAdaptiveBattery(): Promise<void> {
    if (Platform.OS === 'android') {
      this.config.adaptiveBattery = false;
      await this.saveConfig();

      if (NativeModules.BatteryModule) {
        NativeModules.BatteryModule.disableAdaptiveBattery();
      }
    }
  }

  /**
   * Estimate battery impact of operation
   */
  estimateImpact(durationMs: number, intensity: number): {
    estimatedDrain: number;
    projectedLevel: number;
    isSafe: boolean;
  } {
    // Rough estimation: 1% per 10 minutes of active use
    const baseDrainRate = 0.1; // % per minute
    const estimatedDrain = (durationMs / 60000) * baseDrainRate * intensity;

    const projectedLevel = Math.max(0, this.status.level - estimatedDrain);

    return {
      estimatedDrain,
      projectedLevel,
      isSafe: projectedLevel > this.config.minBatteryForBackup,
    };
  }

  /**
   * Wait until device is charging (with timeout)
   */
  async waitForCharging(timeoutMs: number = 300000): Promise<boolean> {
    if (this.status.isCharging) {
      return true;
    }

    return new Promise(resolve => {
      const timeout = setTimeout(() => {
        unsubscribe();
        resolve(false);
      }, timeoutMs);

      const unsubscribe = this.subscribe(status => {
        if (status.isCharging) {
          clearTimeout(timeout);
          unsubscribe();
          resolve(true);
        }
      });
    });
  }

  /**
   * Get optimal backup time based on battery
   */
  getOptimalBackupTime(): {
    canBackupNow: boolean;
    recommendedTime?: Date;
    reason: string;
  } {
    const budget = this.canPerformBackup();

    if (budget.canPerformBackup) {
      return {
        canBackupNow: true,
        reason: budget.reason,
      };
    }

    // Defer to next hour (user might charge)
    const recommendedTime = new Date();
    recommendedTime.setHours(recommendedTime.getHours() + 1);

    return {
      canBackupNow: false,
      recommendedTime,
      reason: budget.reason,
    };
  }

  /**
   * Save configuration to storage
   */
  private async saveConfig(): Promise<void> {
    try {
      await AsyncStorage.setItem(
        'battery_config',
        JSON.stringify(this.config),
      );
    } catch (error) {
      console.error('Failed to save battery config:', error);
    }
  }

  /**
   * Load configuration from storage
   */
  private async loadConfig(): Promise<void> {
    try {
      const saved = await AsyncStorage.getItem('battery_config');
      if (saved) {
        this.config = JSON.parse(saved);
      }
    } catch (error) {
      console.error('Failed to load battery config:', error);
    }
  }

  /**
   * Get battery statistics
   */
  getStats(): {
    currentLevel: number;
    isCharging: boolean;
    lowPowerMode: boolean;
    canBackup: boolean;
    powerBudget: PowerBudget;
  } {
    const powerBudget = this.getPowerBudget();

    return {
      currentLevel: this.status.level,
      isCharging: this.status.isCharging,
      lowPowerMode: this.status.lowPowerMode,
      canBackup: powerBudget.canPerformBackup,
      powerBudget,
    };
  }
}

export const batteryService = new BatteryService();
