/**
 * React Hooks for Advanced Mobile Features
 */

import {useState, useEffect, useCallback} from 'react';
import {networkPolicyService, NetworkStatus, NetworkDecision} from './networkPolicyService';
import {batteryService, BatteryStatus, PowerBudget} from './batteryService';
import {locationService, Location, Geofence, GeofenceTrigger} from './locationService';
import {backgroundService, BackgroundTask} from './backgroundService';
import {dataSaverService, DataSaverConfig} from './advancedFeaturesService';

// ============================================================================
// Network Policy Hooks
// ============================================================================

/**
 * Hook to access network status and policies
 */
export function useNetworkPolicy() {
  const [networkStatus, setNetworkStatus] = useState<NetworkStatus | null>(null);

  useEffect(() => {
    const unsubscribe = networkPolicyService.subscribe(status => {
      setNetworkStatus(status);
    });

    return unsubscribe;
  }, []);

  const canBackup = useCallback(
    async (backupSize: number): Promise<NetworkDecision> => {
      return await networkPolicyService.canPerformBackup(backupSize);
    },
    [],
  );

  const waitForOptimalNetwork = useCallback(
    async (timeout?: number): Promise<boolean> => {
      return await networkPolicyService.waitForOptimalNetwork(timeout);
    },
    [],
  );

  return {
    networkStatus,
    canBackup,
    waitForOptimalNetwork,
    updatePolicy: networkPolicyService.updatePolicy.bind(networkPolicyService),
    getPolicy: networkPolicyService.getPolicy.bind(networkPolicyService),
    getStats: networkPolicyService.getNetworkStats.bind(networkPolicyService),
  };
}

// ============================================================================
// Battery Hooks
// ============================================================================

/**
 * Hook to access battery status and power management
 */
export function useBattery() {
  const [batteryStatus, setBatteryStatus] = useState<BatteryStatus | null>(null);
  const [powerBudget, setPowerBudget] = useState<PowerBudget | null>(null);

  useEffect(() => {
    const unsubscribe = batteryService.subscribe(status => {
      setBatteryStatus(status);
      const budget = batteryService.getPowerBudget();
      setPowerBudget(budget);
    });

    return unsubscribe;
  }, []);

  const estimateImpact = useCallback(
    (durationMs: number, intensity: number) => {
      return batteryService.estimateImpact(durationMs, intensity);
    },
    [],
  );

  return {
    batteryStatus,
    powerBudget,
    estimateImpact,
    updateConfig: batteryService.updateConfig.bind(batteryService),
    getConfig: batteryService.getConfig.bind(batteryService),
    enableAdaptiveBattery: batteryService.enableAdaptiveBattery.bind(
      batteryService,
    ),
    disableAdaptiveBattery: batteryService.disableAdaptiveBattery.bind(
      batteryService,
    ),
  };
}

// ============================================================================
// Location and Geofencing Hooks
// ============================================================================

/**
 * Hook to access location services and geofencing
 */
export function useLocation() {
  const [currentLocation, setCurrentLocation] = useState<Location | null>(null);
  const [geofences, setGeofences] = useState<Geofence[]>([]);
  const [triggers, setTriggers] = useState<GeofenceTrigger[]>([]);

  useEffect(() => {
    // Subscribe to location updates
    const unsubscribeLocation = locationService.subscribeToLocation(location => {
      setCurrentLocation(location);
    });

    // Subscribe to geofence triggers
    const unsubscribeGeofences = locationService.subscribeToGeofences(
      trigger => {
        setTriggers(prev => [...prev, trigger].slice(-20)); // Keep last 20
      },
    );

    // Load geofences
    setGeofences(locationService.getGeofences());

    return () => {
      unsubscribeLocation();
      unsubscribeGeofences();
    };
  }, []);

  const addGeofence = useCallback(async (geofence: Geofence) => {
    await locationService.addGeofence(geofence);
    setGeofences(locationService.getGeofences());
  }, []);

  const removeGeofence = useCallback(async (id: string) => {
    await locationService.removeGeofence(id);
    setGeofences(locationService.getGeofences());
  }, []);

  const getCurrentLocation = useCallback(async () => {
    const location = await locationService.getCurrentLocation();
    setCurrentLocation(location);
    return location;
  }, []);

  return {
    currentLocation,
    geofences,
    triggers,
    addGeofence,
    removeGeofence,
    getCurrentLocation,
    createWorkGeofence: locationService.createWorkGeofence.bind(
      locationService,
    ),
    createHomeGeofence: locationService.createHomeGeofence.bind(
      locationService,
    ),
    getStats: locationService.getStats.bind(locationService),
  };
}

// ============================================================================
// Background Tasks Hook
// ============================================================================

/**
 * Hook to manage background tasks
 */
export function useBackgroundTasks() {
  const [pendingTasks, setPendingTasks] = useState<BackgroundTask[]>([]);
  const [stats, setStats] = useState(backgroundService.getStats());

  useEffect(() => {
    const interval = setInterval(() => {
      setPendingTasks(backgroundService.getPendingTasks());
      setStats(backgroundService.getStats());
    }, 5000); // Update every 5 seconds

    return () => clearInterval(interval);
  }, []);

  const scheduleTask = useCallback(async (task: BackgroundTask) => {
    await backgroundService.scheduleTask(task);
    setPendingTasks(backgroundService.getPendingTasks());
  }, []);

  const registerHandler = useCallback(
    (type: string, handler: (task: BackgroundTask) => Promise<void>) => {
      backgroundService.registerTaskHandler(type, handler);
    },
    [],
  );

  return {
    pendingTasks,
    stats,
    scheduleTask,
    registerHandler,
    updateConfig: backgroundService.updateConfig.bind(backgroundService),
    getConfig: backgroundService.getConfig.bind(backgroundService),
    getHistory: backgroundService.getTaskHistory.bind(backgroundService),
  };
}

// ============================================================================
// Data Saver Hook
// ============================================================================

/**
 * Hook to manage data saver mode
 */
export function useDataSaver() {
  const [config, setConfig] = useState<DataSaverConfig>(
    dataSaverService.getConfig(),
  );
  const [enabled, setEnabled] = useState(dataSaverService.isEnabled());

  const updateConfig = useCallback(async (newConfig: Partial<DataSaverConfig>) => {
    await dataSaverService.updateConfig(newConfig);
    setConfig(dataSaverService.getConfig());
  }, []);

  const enable = useCallback(async () => {
    await dataSaverService.enable();
    setEnabled(true);
    setConfig(dataSaverService.getConfig());
  }, []);

  const disable = useCallback(async () => {
    await dataSaverService.disable();
    setEnabled(false);
    setConfig(dataSaverService.getConfig());
  }, []);

  const shouldDeferBackup = useCallback(
    (backupSize: number): boolean => {
      return dataSaverService.shouldDeferBackup(backupSize);
    },
    [config],
  );

  return {
    config,
    enabled,
    updateConfig,
    enable,
    disable,
    shouldDeferBackup,
    getRecommendedCompression: dataSaverService.getRecommendedCompression.bind(
      dataSaverService,
    ),
  };
}

// ============================================================================
// Combined Backup Decision Hook
// ============================================================================

/**
 * Hook that combines all factors to determine if backup can proceed
 */
export function useBackupDecision() {
  const network = useNetworkPolicy();
  const battery = useBattery();
  const dataSaver = useDataSaver();

  const canPerformBackup = useCallback(
    async (backupSize: number) => {
      const decisions = {
        network: await network.canBackup(backupSize),
        battery: battery.powerBudget,
        dataSaver: dataSaver.shouldDeferBackup(backupSize),
      };

      const canProceed =
        decisions.network.allowed &&
        decisions.battery?.canPerformBackup &&
        !decisions.dataSaver;

      return {
        canProceed,
        decisions,
        reason: !canProceed
          ? getBlockingReason(decisions)
          : 'All conditions met',
      };
    },
    [network, battery.powerBudget, dataSaver],
  );

  return {
    canPerformBackup,
    networkStatus: network.networkStatus,
    batteryStatus: battery.batteryStatus,
    powerBudget: battery.powerBudget,
    dataSaverEnabled: dataSaver.enabled,
  };
}

function getBlockingReason(decisions: any): string {
  if (!decisions.network.allowed) {
    return decisions.network.reason;
  }
  if (!decisions.battery?.canPerformBackup) {
    return decisions.battery.reason;
  }
  if (decisions.dataSaver) {
    return 'Data saver mode: backup size exceeds limit';
  }
  return 'Unknown reason';
}
