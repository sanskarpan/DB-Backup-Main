/**
 * Advanced Features Service
 * NFC, QR/Barcode scanning, AR, iOS Shortcuts, CarPlay/Android Auto
 */

import {Platform, NativeModules, Linking} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';

// ============================================================================
// NFC Service
// ============================================================================

export interface NFCTag {
  id: string;
  name: string;
  action: 'backup' | 'restore' | 'sync' | 'custom';
  data?: any;
}

export interface NFCReadResult {
  id: string;
  type: string;
  data: any;
  timestamp: number;
}

class NFCService {
  private tags: Map<string, NFCTag> = new Map();
  private isSupported: boolean = false;

  async initialize(): Promise<void> {
    // Check NFC support
    this.isSupported = await this.checkNFCSupport();

    if (this.isSupported) {
      await this.loadTags();
    }
  }

  private async checkNFCSupport(): Promise<boolean> {
    if (Platform.OS === 'ios') {
      // iOS 13+ supports NFC
      const {NFCModule} = NativeModules;
      return NFCModule?.isSupported?.() || false;
    } else {
      // Android NFC support
      const {NFCModule} = NativeModules;
      return NFCModule?.isSupported?.() || false;
    }
  }

  async registerTag(tag: NFCTag): Promise<void> {
    this.tags.set(tag.id, tag);
    await this.saveTags();
  }

  async readTag(): Promise<NFCReadResult | null> {
    if (!this.isSupported) {
      throw new Error('NFC not supported on this device');
    }

    try {
      const {NFCModule} = NativeModules;
      if (NFCModule) {
        const result = await NFCModule.readTag();
        return result;
      }
    } catch (error) {
      console.error('NFC read error:', error);
    }

    return null;
  }

  async writeTag(tagId: string, data: any): Promise<void> {
    if (!this.isSupported) {
      throw new Error('NFC not supported on this device');
    }

    const {NFCModule} = NativeModules;
    if (NFCModule) {
      await NFCModule.writeTag(tagId, data);
    }
  }

  getRegisteredTags(): NFCTag[] {
    return Array.from(this.tags.values());
  }

  isNFCSupported(): boolean {
    return this.isSupported;
  }

  private async saveTags(): Promise<void> {
    const tagsArray = Array.from(this.tags.values());
    await AsyncStorage.setItem('nfc_tags', JSON.stringify(tagsArray));
  }

  private async loadTags(): Promise<void> {
    const saved = await AsyncStorage.getItem('nfc_tags');
    if (saved) {
      const tagsArray: NFCTag[] = JSON.parse(saved);
      this.tags = new Map(tagsArray.map(t => [t.id, t]));
    }
  }
}

// ============================================================================
// QR/Barcode Scanner Service
// ============================================================================

export interface ScanResult {
  type: 'qr' | 'barcode';
  format: string;
  data: string;
  timestamp: number;
}

export interface QRCodeData {
  action: string;
  params?: any;
}

class ScannerService {
  async scanQRCode(): Promise<ScanResult | null> {
    // Would integrate with react-native-camera or react-native-qrcode-scanner
    // For now, return null as placeholder
    return null;
  }

  parseQRCode(data: string): QRCodeData | null {
    try {
      // Support backup URLs: dbbackup://backup?database=xxx
      if (data.startsWith('dbbackup://')) {
        const url = new URL(data);
        return {
          action: url.pathname.replace('/', ''),
          params: Object.fromEntries(url.searchParams),
        };
      }

      // Support JSON format
      return JSON.parse(data);
    } catch (error) {
      console.error('Failed to parse QR code:', error);
      return null;
    }
  }

  generateQRCodeData(action: string, params?: any): string {
    const queryString = params
      ? '?' + new URLSearchParams(params).toString()
      : '';
    return `dbbackup://${action}${queryString}`;
  }

  async scanBarcode(): Promise<ScanResult | null> {
    // Would integrate with barcode scanner library
    return null;
  }
}

// ============================================================================
// AR Visualization Service
// ============================================================================

export interface ARObject {
  id: string;
  type: 'datacenter' | 'server' | 'database' | 'backup';
  position: {x: number; y: number; z: number};
  rotation: {x: number; y: number; z: number};
  scale: {x: number; y: number; z: number};
  data: any;
}

export interface ARScene {
  objects: ARObject[];
  environment: 'datacenter' | 'office' | 'home';
}

class ARService {
  private isSupported: boolean = false;

  async initialize(): Promise<void> {
    this.isSupported = await this.checkARSupport();
  }

  private async checkARSupport(): Promise<boolean> {
    if (Platform.OS === 'ios') {
      // ARKit support (iOS 11+)
      const {ARModule} = NativeModules;
      return ARModule?.isARKitSupported?.() || false;
    } else {
      // ARCore support (Android)
      const {ARModule} = NativeModules;
      return ARModule?.isARCoreSupported?.() || false;
    }
  }

  isARSupported(): boolean {
    return this.isSupported;
  }

  generateDatacenterVisualization(datacenterData: any): ARScene {
    const objects: ARObject[] = [];

    // Create datacenter building
    objects.push({
      id: 'datacenter',
      type: 'datacenter',
      position: {x: 0, y: 0, z: 0},
      rotation: {x: 0, y: 0, z: 0},
      scale: {x: 1, y: 1, z: 1},
      data: datacenterData,
    });

    // Add server racks
    if (datacenterData.servers) {
      datacenterData.servers.forEach((server: any, index: number) => {
        objects.push({
          id: `server-${index}`,
          type: 'server',
          position: {x: index * 2, y: 0, z: 0},
          rotation: {x: 0, y: 0, z: 0},
          scale: {x: 0.5, y: 0.5, z: 0.5},
          data: server,
        });
      });
    }

    return {
      objects,
      environment: 'datacenter',
    };
  }

  async startARSession(): Promise<void> {
    if (!this.isSupported) {
      throw new Error('AR not supported on this device');
    }

    const {ARModule} = NativeModules;
    if (ARModule) {
      await ARModule.startSession();
    }
  }

  async stopARSession(): Promise<void> {
    const {ARModule} = NativeModules;
    if (ARModule) {
      await ARModule.stopSession();
    }
  }
}

// ============================================================================
// iOS Shortcuts Service
// ============================================================================

export interface Shortcut {
  id: string;
  name: string;
  description: string;
  action: string;
  params?: any;
  suggestedPhrase?: string;
}

class ShortcutsService {
  private shortcuts: Map<string, Shortcut> = new Map();

  async initialize(): Promise<void> {
    if (Platform.OS === 'ios') {
      await this.loadShortcuts();
      await this.registerDefaultShortcuts();
    }
  }

  async registerShortcut(shortcut: Shortcut): Promise<void> {
    if (Platform.OS !== 'ios') {
      return;
    }

    this.shortcuts.set(shortcut.id, shortcut);

    // Register with iOS Shortcuts app via native module
    const {ShortcutsModule} = NativeModules;
    if (ShortcutsModule) {
      await ShortcutsModule.donateShortcut({
        identifier: shortcut.id,
        title: shortcut.name,
        suggestedInvocationPhrase: shortcut.suggestedPhrase,
      });
    }

    await this.saveShortcuts();
  }

  private async registerDefaultShortcuts(): Promise<void> {
    const defaultShortcuts: Shortcut[] = [
      {
        id: 'backup_now',
        name: 'Backup Now',
        description: 'Start a backup immediately',
        action: 'backup',
        suggestedPhrase: 'Backup my database',
      },
      {
        id: 'check_status',
        name: 'Check Backup Status',
        description: 'Check the status of backups',
        action: 'status',
        suggestedPhrase: 'Check backup status',
      },
      {
        id: 'list_backups',
        name: 'List Recent Backups',
        description: 'Show recent backups',
        action: 'list',
        suggestedPhrase: 'Show my backups',
      },
    ];

    for (const shortcut of defaultShortcuts) {
      if (!this.shortcuts.has(shortcut.id)) {
        await this.registerShortcut(shortcut);
      }
    }
  }

  getShortcuts(): Shortcut[] {
    return Array.from(this.shortcuts.values());
  }

  async handleShortcut(identifier: string): Promise<any> {
    const shortcut = this.shortcuts.get(identifier);
    if (!shortcut) {
      throw new Error(`Shortcut ${identifier} not found`);
    }

    return {
      action: shortcut.action,
      params: shortcut.params,
    };
  }

  private async saveShortcuts(): Promise<void> {
    const shortcutsArray = Array.from(this.shortcuts.values());
    await AsyncStorage.setItem('ios_shortcuts', JSON.stringify(shortcutsArray));
  }

  private async loadShortcuts(): Promise<void> {
    const saved = await AsyncStorage.getItem('ios_shortcuts');
    if (saved) {
      const shortcutsArray: Shortcut[] = JSON.parse(saved);
      this.shortcuts = new Map(shortcutsArray.map(s => [s.id, s]));
    }
  }
}

// ============================================================================
// CarPlay/Android Auto Service
// ============================================================================

export interface AutoNotification {
  id: string;
  title: string;
  message: string;
  priority: 'low' | 'medium' | 'high';
  category: 'backup' | 'restore' | 'alert';
  actions?: AutoAction[];
}

export interface AutoAction {
  id: string;
  title: string;
  action: string;
}

class AutoService {
  private isConnected: boolean = false;

  async initialize(): Promise<void> {
    await this.checkAutoConnection();
  }

  private async checkAutoConnection(): Promise<void> {
    if (Platform.OS === 'ios') {
      // Check CarPlay connection
      const {CarPlayModule} = NativeModules;
      if (CarPlayModule) {
        this.isConnected = await CarPlayModule.isConnected();
      }
    } else {
      // Check Android Auto connection
      const {AndroidAutoModule} = NativeModules;
      if (AndroidAutoModule) {
        this.isConnected = await AndroidAutoModule.isConnected();
      }
    }
  }

  async sendNotification(notification: AutoNotification): Promise<void> {
    if (!this.isConnected) {
      console.warn('CarPlay/Android Auto not connected');
      return;
    }

    if (Platform.OS === 'ios') {
      const {CarPlayModule} = NativeModules;
      if (CarPlayModule) {
        await CarPlayModule.sendNotification(notification);
      }
    } else {
      const {AndroidAutoModule} = NativeModules;
      if (AndroidAutoModule) {
        await AndroidAutoModule.sendNotification(notification);
      }
    }
  }

  isAutoConnected(): boolean {
    return this.isConnected;
  }

  async updateConnectionStatus(): Promise<boolean> {
    await this.checkAutoConnection();
    return this.isConnected;
  }

  async sendBackupCompleteNotification(backupName: string): Promise<void> {
    await this.sendNotification({
      id: `backup-complete-${Date.now()}`,
      title: 'Backup Complete',
      message: `${backupName} backup finished successfully`,
      priority: 'medium',
      category: 'backup',
    });
  }

  async sendBackupFailedNotification(
    backupName: string,
    error: string,
  ): Promise<void> {
    await this.sendNotification({
      id: `backup-failed-${Date.now()}`,
      title: 'Backup Failed',
      message: `${backupName} backup failed: ${error}`,
      priority: 'high',
      category: 'alert',
      actions: [
        {
          id: 'retry',
          title: 'Retry',
          action: 'backup_retry',
        },
        {
          id: 'view',
          title: 'View Details',
          action: 'backup_details',
        },
      ],
    });
  }
}

// ============================================================================
// Data Saver Service
// ============================================================================

export interface DataSaverConfig {
  enabled: boolean;
  compressionLevel: number; // 0-9
  reducedQuality: boolean;
  deferLargeBackups: boolean;
  maxBackupSize: number; // bytes
}

class DataSaverService {
  private config: DataSaverConfig = {
    enabled: false,
    compressionLevel: 9,
    reducedQuality: true,
    deferLargeBackups: true,
    maxBackupSize: 50 * 1024 * 1024, // 50 MB
  };

  async initialize(): Promise<void> {
    await this.loadConfig();
  }

  async enable(): Promise<void> {
    this.config.enabled = true;
    await this.saveConfig();
  }

  async disable(): Promise<void> {
    this.config.enabled = false;
    await this.saveConfig();
  }

  isEnabled(): boolean {
    return this.config.enabled;
  }

  async updateConfig(config: Partial<DataSaverConfig>): Promise<void> {
    this.config = {...this.config, ...config};
    await this.saveConfig();
  }

  getConfig(): DataSaverConfig {
    return {...this.config};
  }

  shouldDeferBackup(backupSize: number): boolean {
    if (!this.config.enabled) {
      return false;
    }

    return (
      this.config.deferLargeBackups && backupSize > this.config.maxBackupSize
    );
  }

  getRecommendedCompression(): number {
    return this.config.enabled ? this.config.compressionLevel : 6;
  }

  private async saveConfig(): Promise<void> {
    await AsyncStorage.setItem('data_saver_config', JSON.stringify(this.config));
  }

  private async loadConfig(): Promise<void> {
    const saved = await AsyncStorage.getItem('data_saver_config');
    if (saved) {
      this.config = JSON.parse(saved);
    }
  }
}

// Export singleton instances
export const nfcService = new NFCService();
export const scannerService = new ScannerService();
export const arService = new ARService();
export const shortcutsService = new ShortcutsService();
export const autoService = new AutoService();
export const dataSaverService = new DataSaverService();
