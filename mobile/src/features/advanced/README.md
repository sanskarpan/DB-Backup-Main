# Advanced Mobile Features

This module provides advanced mobile-specific features for the DB Backup mobile app.

## Features

1. **Network Policy Management** - Cellular vs WiFi backup control
2. **Battery Management** - Low power mode and adaptive battery support
3. **Location Services** - Geofencing for location-based backups
4. **Background Optimization** - Efficient background task management
5. **NFC Support** - Quick actions via NFC tags
6. **QR/Barcode Scanning** - Scan codes for quick configuration
7. **AR Visualization** - Augmented reality datacenter visualization
8. **iOS Shortcuts** - Siri shortcuts integration
9. **CarPlay/Android Auto** - Vehicle integration
10. **Data Saver Mode** - Reduced data usage

## Installation

Dependencies are already added to `package.json`. Run:

```bash
npm install
# or
yarn install
```

### iOS Setup

Add to `Info.plist`:

```xml
<key>NSLocationWhenInUseUsageDescription</key>
<string>We need your location for geofencing features</string>
<key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
<string>We need your location for location-based backup triggers</string>
<key>NFCReaderUsageDescription</key>
<string>We use NFC for quick backup actions</string>
```

### Android Setup

Add to `AndroidManifest.xml`:

```xml
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
<uses-permission android:name="android.permission.NFC" />
<uses-permission android:name="android.permission.RECEIVE_BOOT_COMPLETED" />
```

## Usage

### Initialization

Initialize all services in your `App.tsx`:

```typescript
import {initializeAdvancedFeatures} from './src/features/advanced';

// In your App component
useEffect(() => {
  initializeAdvancedFeatures().catch(console.error);
}, []);
```

### Network Policy

```typescript
import {useNetworkPolicy} from './src/features/advanced';

function BackupScreen() {
  const {networkStatus, canBackup, updatePolicy} = useNetworkPolicy();

  const handleBackup = async () => {
    const backupSize = 50 * 1024 * 1024; // 50 MB
    const decision = await canBackup(backupSize);

    if (decision.allowed) {
      // Proceed with backup
      performBackup();
    } else {
      Alert.alert('Cannot Backup', decision.reason);
    }
  };

  const enableCellular = async () => {
    await updatePolicy({
      allowCellular: true,
      cellularSizeLimit: 100 * 1024 * 1024, // 100 MB
    });
  };

  return (
    <View>
      <Text>Network: {networkStatus?.type}</Text>
      <Text>Connected: {networkStatus?.isConnected ? 'Yes' : 'No'}</Text>
      <Button title="Backup Now" onPress={handleBackup} />
      <Button title="Enable Cellular" onPress={enableCellular} />
    </View>
  );
}
```

### Battery Management

```typescript
import {useBattery} from './src/features/advanced';

function SettingsScreen() {
  const {batteryStatus, powerBudget, enableAdaptiveBattery} = useBattery();

  return (
    <View>
      <Text>Battery: {batteryStatus?.level}%</Text>
      <Text>Charging: {batteryStatus?.isCharging ? 'Yes' : 'No'}</Text>
      <Text>Can Backup: {powerBudget?.canPerformBackup ? 'Yes' : 'No'}</Text>
      <Text>Reason: {powerBudget?.reason}</Text>
      {Platform.OS === 'android' && (
        <Button
          title="Enable Adaptive Battery"
          onPress={enableAdaptiveBattery}
        />
      )}
    </View>
  );
}
```

### Location & Geofencing

```typescript
import {useLocation} from './src/features/advanced';

function GeofencingScreen() {
  const {
    currentLocation,
    geofences,
    triggers,
    addGeofence,
    createWorkGeofence,
  } = useLocation();

  const setupWorkGeofence = async () => {
    if (currentLocation) {
      await createWorkGeofence(currentLocation, 100); // 100 meter radius
      Alert.alert('Success', 'Work geofence created');
    }
  };

  useEffect(() => {
    // Listen for geofence triggers
    if (triggers.length > 0) {
      const latest = triggers[triggers.length - 1];
      if (latest.triggerType === 'enter' && latest.name === 'Work Location') {
        // Trigger automatic backup when entering work
        startAutomaticBackup();
      }
    }
  }, [triggers]);

  return (
    <View>
      <Text>
        Location: {currentLocation?.latitude}, {currentLocation?.longitude}
      </Text>
      <Text>Geofences: {geofences.length}</Text>
      <Button title="Setup Work Geofence" onPress={setupWorkGeofence} />
    </View>
  );
}
```

### Background Tasks

```typescript
import {useBackgroundTasks} from './src/features/advanced';

function App() {
  const {scheduleTask, registerHandler} = useBackgroundTasks();

  useEffect(() => {
    // Register handler for backup tasks
    registerHandler('backup', async task => {
      console.log('Processing background backup:', task.id);
      await performBackup(task.data);
    });
  }, []);

  const scheduleBackup = async () => {
    await scheduleTask({
      id: `backup-${Date.now()}`,
      type: 'backup',
      priority: 'high',
      estimatedDuration: 60000, // 1 minute
      data: {databaseId: 'db-123'},
    });
  };

  return <Button title="Schedule Backup" onPress={scheduleBackup} />;
}
```

### Data Saver

```typescript
import {useDataSaver} from './src/features/advanced';

function DataSaverScreen() {
  const {enabled, enable, disable, shouldDeferBackup} = useDataSaver();

  const handleBackup = async (backupSize: number) => {
    if (shouldDeferBackup(backupSize)) {
      Alert.alert(
        'Data Saver Active',
        'This backup exceeds data saver limit. Backup will be deferred until WiFi.',
      );
      return;
    }

    await performBackup();
  };

  return (
    <View>
      <Text>Data Saver: {enabled ? 'Enabled' : 'Disabled'}</Text>
      <Switch value={enabled} onValueChange={enabled ? disable : enable} />
    </View>
  );
}
```

### Combined Backup Decision

```typescript
import {useBackupDecision} from './src/features/advanced';

function SmartBackupButton() {
  const {
    canPerformBackup,
    networkStatus,
    batteryStatus,
    powerBudget,
  } = useBackupDecision();

  const handleBackup = async () => {
    const backupSize = 100 * 1024 * 1024; // 100 MB
    const decision = await canPerformBackup(backupSize);

    if (decision.canProceed) {
      performBackup();
    } else {
      Alert.alert('Cannot Backup', decision.reason, [
        {text: 'OK'},
        {
          text: 'View Details',
          onPress: () => showDecisionDetails(decision),
        },
      ]);
    }
  };

  return (
    <View>
      <Button
        title="Smart Backup"
        onPress={handleBackup}
        disabled={!networkStatus?.isConnected}
      />
      <Text>Network: {networkStatus?.type}</Text>
      <Text>Battery: {batteryStatus?.level}%</Text>
      <Text>
        Can Backup: {powerBudget?.canPerformBackup ? 'Yes' : 'No'}
      </Text>
    </View>
  );
}
```

### iOS Shortcuts

```typescript
import {shortcutsService} from './src/features/advanced';

// In your app initialization
useEffect(() => {
  if (Platform.OS === 'ios') {
    // Register custom shortcuts
    shortcutsService.registerShortcut({
      id: 'quick_backup',
      name: 'Quick Backup',
      description: 'Start a quick backup',
      action: 'backup',
      params: {quick: true},
      suggestedPhrase: 'Quick backup my database',
    });
  }
}, []);

// Handle shortcut invocation
Linking.addEventListener('url', event => {
  const {url} = event;
  if (url.startsWith('dbbackup://shortcut/')) {
    const shortcutId = url.split('/').pop();
    shortcutsService.handleShortcut(shortcutId).then(action => {
      // Execute the shortcut action
      executeAction(action);
    });
  }
});
```

### CarPlay/Android Auto

```typescript
import {autoService} from './src/features/advanced';

// Send notifications to vehicle display
const notifyBackupComplete = async () => {
  await autoService.sendBackupCompleteNotification('Production DB');
};

const notifyBackupFailed = async (error: string) => {
  await autoService.sendBackupFailedNotification('Production DB', error);
};
```

## Testing

Run tests:

```bash
npm test
# or
yarn test
```

## Architecture

- **Services**: Singleton services managing features
- **Hooks**: React hooks for easy component integration
- **Types**: TypeScript interfaces for type safety
- **Storage**: AsyncStorage for persistence

## Performance

- Services use efficient polling and event listeners
- Background tasks are throttled to respect battery
- Location updates use distance filtering
- Network status is cached and updated on changes

## Platform Differences

### iOS-specific
- CarPlay integration
- iOS Shortcuts/Siri integration
- ARKit for AR features

### Android-specific
- Adaptive Battery support
- Android Auto integration
- ARCore for AR features

## Troubleshooting

### Location not updating
- Check permissions are granted
- Verify location services are enabled
- Check Info.plist (iOS) / AndroidManifest.xml (Android)

### Background tasks not running
- Ensure background fetch is enabled in capabilities
- Check battery optimization settings
- Verify `stopOnTerminate` is set correctly

### NFC not working
- NFC must be supported by device
- Check NFC is enabled in device settings
- Verify app has NFC permissions

## License

Copyright (c) 2026 DB Backup. All rights reserved.
