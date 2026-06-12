/**
 * Advanced Mobile Features - Main Export
 *
 * This module provides advanced mobile-specific features including:
 * - Network policy management (cellular vs WiFi)
 * - Battery-aware operations
 * - Location-based triggers (geofencing)
 * - Background task optimization
 * - NFC tag support
 * - QR/Barcode scanning
 * - AR visualization
 * - iOS Shortcuts integration
 * - CarPlay/Android Auto support
 * - Data saver mode
 */

// Services
export {
  networkPolicyService,
  type NetworkPolicy,
  type NetworkStatus,
  type NetworkDecision,
} from './networkPolicyService';

export {
  batteryService,
  type BatteryStatus,
  type PowerBudget,
  type BatteryConfig,
} from './batteryService';

export {
  locationService,
  type Location,
  type Geofence,
  type GeofenceTrigger,
} from './locationService';

export {
  backgroundService,
  type BackgroundTask,
  type BackgroundTaskResult,
  type BackgroundConfig,
} from './backgroundService';

export {
  nfcService,
  scannerService,
  arService,
  shortcutsService,
  autoService,
  dataSaverService,
  type NFCTag,
  type NFCReadResult,
  type ScanResult,
  type QRCodeData,
  type ARObject,
  type ARScene,
  type Shortcut,
  type AutoNotification,
  type AutoAction,
  type DataSaverConfig,
} from './advancedFeaturesService';

// Hooks
export {
  useNetworkPolicy,
  useBattery,
  useLocation,
  useBackgroundTasks,
  useDataSaver,
  useBackupDecision,
} from './hooks';

// Initialize all services
import {networkPolicyService} from './networkPolicyService';
import {batteryService} from './batteryService';
import {locationService} from './locationService';
import {backgroundService} from './backgroundService';
import {
  nfcService,
  arService,
  shortcutsService,
  autoService,
  dataSaverService,
} from './advancedFeaturesService';

/**
 * Initialize all advanced features
 * Call this once during app startup
 */
export async function initializeAdvancedFeatures(): Promise<void> {
  try {
    await Promise.all([
      networkPolicyService.initialize(),
      batteryService.initialize(),
      locationService.initialize(),
      backgroundService.initialize(),
      nfcService.initialize(),
      arService.initialize(),
      shortcutsService.initialize(),
      autoService.initialize(),
      dataSaverService.initialize(),
    ]);

    console.log('[AdvancedFeatures] All services initialized successfully');
  } catch (error) {
    console.error('[AdvancedFeatures] Initialization error:', error);
    throw error;
  }
}

/**
 * Cleanup all services
 * Call this during app shutdown
 */
export function disposeAdvancedFeatures(): void {
  locationService.dispose();
  console.log('[AdvancedFeatures] All services disposed');
}
