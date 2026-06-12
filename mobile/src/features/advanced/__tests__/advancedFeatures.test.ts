/**
 * Advanced Mobile Features Tests
 */

import {
  networkPolicyService,
  batteryService,
  locationService,
  backgroundService,
  dataSaverService,
} from '../index';

describe('Network Policy Service', () => {
  it('should initialize without errors', async () => {
    await expect(networkPolicyService.initialize()).resolves.not.toThrow();
  });

  it('should return network stats', () => {
    const stats = networkPolicyService.getNetworkStats();
    expect(stats).toHaveProperty('currentType');
    expect(stats).toHaveProperty('isConnected');
  });

  it('should update network policy', async () => {
    await networkPolicyService.updatePolicy({
      allowCellular: true,
      cellularSizeLimit: 100 * 1024 * 1024,
    });

    const policy = networkPolicyService.getPolicy();
    expect(policy.allowCellular).toBe(true);
    expect(policy.cellularSizeLimit).toBe(100 * 1024 * 1024);
  });
});

describe('Battery Service', () => {
  it('should initialize without errors', async () => {
    await expect(batteryService.initialize()).resolves.not.toThrow();
  });

  it('should return battery status', () => {
    const status = batteryService.getStatus();
    expect(status).toHaveProperty('level');
    expect(status).toHaveProperty('isCharging');
    expect(status).toHaveProperty('lowPowerMode');
  });

  it('should calculate power budget', () => {
    const budget = batteryService.getPowerBudget();
    expect(budget).toHaveProperty('maxDuration');
    expect(budget).toHaveProperty('powerLevel');
    expect(budget).toHaveProperty('canPerformBackup');
  });

  it('should estimate battery impact', () => {
    const impact = batteryService.estimateImpact(60000, 1);
    expect(impact).toHaveProperty('estimatedDrain');
    expect(impact).toHaveProperty('projectedLevel');
    expect(impact).toHaveProperty('isSafe');
  });
});

describe('Location Service', () => {
  it.skip('should initialize without errors', async () => {
    await expect(locationService.initialize()).resolves.not.toThrow();
  }, 10000);

  it('should add and remove geofences', async () => {
    const geofence = {
      id: 'test-fence',
      name: 'Test Geofence',
      latitude: 37.7749,
      longitude: -122.4194,
      radius: 100,
      action: 'backup' as const,
      triggerType: 'enter' as const,
      enabled: true,
    };

    await locationService.addGeofence(geofence);

    let geofences = locationService.getGeofences();
    expect(geofences).toHaveLength(1);
    expect(geofences[0].id).toBe('test-fence');

    await locationService.removeGeofence('test-fence');

    geofences = locationService.getGeofences();
    expect(geofences).toHaveLength(0);
  });

  it('should return location stats', () => {
    const stats = locationService.getStats();
    expect(stats).toHaveProperty('totalGeofences');
    expect(stats).toHaveProperty('enabledGeofences');
    expect(stats).toHaveProperty('totalTriggers');
  });
});

describe('Background Service', () => {
  it('should initialize without errors', async () => {
    await expect(backgroundService.initialize()).resolves.not.toThrow();
  });

  it('should schedule tasks', async () => {
    const task = {
      id: 'test-task',
      type: 'backup' as const,
      priority: 'medium' as const,
      estimatedDuration: 60000,
    };

    await backgroundService.scheduleTask(task);

    const pending = backgroundService.getPendingTasks();
    expect(pending.some(t => t.id === 'test-task')).toBe(true);
  });

  it('should return task statistics', () => {
    const stats = backgroundService.getStats();
    expect(stats).toHaveProperty('pendingTasks');
    expect(stats).toHaveProperty('executedTasks');
    expect(stats).toHaveProperty('successRate');
    expect(stats).toHaveProperty('averageDuration');
  });
});

describe('Data Saver Service', () => {
  it('should initialize without errors', async () => {
    await expect(dataSaverService.initialize()).resolves.not.toThrow();
  });

  it('should enable and disable data saver', async () => {
    await dataSaverService.enable();
    expect(dataSaverService.isEnabled()).toBe(true);

    await dataSaverService.disable();
    expect(dataSaverService.isEnabled()).toBe(false);
  });

  it('should defer large backups when enabled', async () => {
    await dataSaverService.enable();

    const config = dataSaverService.getConfig();
    const largeSize = config.maxBackupSize + 1;

    expect(dataSaverService.shouldDeferBackup(largeSize)).toBe(true);
    expect(dataSaverService.shouldDeferBackup(1024)).toBe(false);
  });

  it('should return recommended compression level', () => {
    const compression = dataSaverService.getRecommendedCompression();
    expect(typeof compression).toBe('number');
    expect(compression).toBeGreaterThanOrEqual(0);
    expect(compression).toBeLessThanOrEqual(9);
  });
});

describe('Integration Tests', () => {
  it('should work together for backup decision', async () => {
    const backupSize = 50 * 1024 * 1024; // 50 MB

    const networkDecision = await networkPolicyService.canPerformBackup(
      backupSize,
    );
    const powerBudget = batteryService.getPowerBudget();
    const shouldDefer = dataSaverService.shouldDeferBackup(backupSize);

    expect(networkDecision).toHaveProperty('allowed');
    expect(networkDecision).toHaveProperty('reason');
    expect(powerBudget).toHaveProperty('canPerformBackup');
    expect(typeof shouldDefer).toBe('boolean');
  });
});
