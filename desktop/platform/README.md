# Desktop Platform Integration

Comprehensive platform-specific integrations for Windows, macOS, and Linux desktop environments.

## Overview

This module provides native desktop integration features for the DB Backup application, including:

- **Windows**: Action Center, Live Tiles, Jump Lists, Thumbnail Toolbar
- **macOS**: Control Center, Touch Bar, Menu Bar, Quick Look, Finder Extensions
- **Linux**: GNOME Shell Extension, KDE Plasma Widget, System Tray, D-Bus Notifications
- **Cross-Platform**: Notifications, Dialogs, Shell Integration (file associations, context menus, protocols)

## Architecture

```
desktop/platform/
├── windows/          # Windows-specific integrations
│   └── index.ts      # Action Center, Live Tiles, Jump Lists, Thumbnail Toolbar
├── macos/            # macOS-specific integrations
│   └── index.ts      # Control Center, Touch Bar, Menu Bar, Quick Look, Finder
├── linux/            # Linux-specific integrations
│   └── index.ts      # GNOME, KDE, System Tray, D-Bus
├── cross/            # Cross-platform features
│   ├── notifications.ts    # Unified notification system
│   ├── dialogs.ts          # File/message dialogs
│   └── shellIntegration.ts # File associations, context menus, protocols
├── hooks.ts          # React hooks for easy integration
├── index.ts          # Main entry point and unified API
└── __tests__/        # Comprehensive test suite
```

## Installation

The platform integration is already included in the desktop app. Dependencies are managed through `package.json`:

```json
{
  "dependencies": {
    "@tauri-apps/api": "^1.5.0"
  }
}
```

## Quick Start

### Basic Initialization

```typescript
import { initializeDesktopIntegration } from './platform';

// Initialize with defaults
await initializeDesktopIntegration();

// Or with custom config
await initializeDesktopIntegration({
  enableNotifications: true,
  enableShellIntegration: true,
  enablePlatformFeatures: true,
});
```

### Using React Hooks

```typescript
import { useCompleteDesktopIntegration } from './platform/hooks';

function App() {
  const desktop = useCompleteDesktopIntegration(true);

  const handleBackup = async () => {
    await desktop.backup.notifyBackupStarted('production-db');

    // Simulate progress
    await desktop.backup.notifyBackupProgress('production-db', 50);

    // Complete
    await desktop.backup.notifyBackupComplete('production-db', '100 MB', '5s');
  };

  return (
    <button onClick={handleBackup}>
      Backup Database
    </button>
  );
}
```

## Features

### 1. Cross-Platform Notifications

Unified notification API that works across all platforms:

```typescript
import { useNotifications } from './platform/hooks';

function MyComponent() {
  const notifications = useNotifications();

  const notify = async () => {
    await notifications.send({
      title: 'Backup Complete',
      body: 'Database backed up successfully',
      urgency: 'normal',
      actions: [
        { id: 'view', title: 'View Details' },
        { id: 'dismiss', title: 'Dismiss' },
      ],
    });
  };

  // Listen for notification events
  useEffect(() => {
    const unsubscribe = notifications.onNotificationEvent((event) => {
      console.log('Notification event:', event);
      if (event.actionId === 'view') {
        // Handle view action
      }
    });

    return unsubscribe;
  }, []);

  return <button onClick={notify}>Send Notification</button>;
}
```

### 2. Cross-Platform Dialogs

File and message dialogs that adapt to each platform:

```typescript
import { useDialogs } from './platform/hooks';

function BackupSelector() {
  const dialogs = useDialogs();

  const selectBackup = async () => {
    const file = await dialogs.selectBackupFile();
    if (file) {
      console.log('Selected backup:', file);
    }
  };

  const saveBackup = async () => {
    const path = await dialogs.selectBackupSaveLocation('my-backup');
    if (path) {
      console.log('Save to:', path);
    }
  };

  const confirmDelete = async () => {
    const confirmed = await dialogs.confirmBackupDeletion('backup-001');
    if (confirmed) {
      // Delete backup
    }
  };

  return (
    <div>
      <button onClick={selectBackup}>Select Backup</button>
      <button onClick={saveBackup}>Save Backup</button>
      <button onClick={confirmDelete}>Delete Backup</button>
    </div>
  );
}
```

### 3. Shell Integration

File associations, context menus, and protocol handlers:

```typescript
import { useShellIntegration } from './platform/hooks';

function ShellSetup() {
  const shell = useShellIntegration();

  useEffect(() => {
    // Register context menu
    shell.registerContextMenu([
      {
        id: 'backup_file',
        label: 'Backup with DB Backup',
        icon: 'backup',
        files: true,
        extensions: ['db', 'sqlite'],
      },
    ]);

    // Listen for context menu actions
    shell.onContextMenuAction((data) => {
      console.log('Context menu clicked:', data);
      if (data.itemId === 'backup_file') {
        // Backup the selected files
        console.log('Files to backup:', data.filePaths);
      }
    });

    // Register protocol handler (dbbackup://)
    shell.registerProtocol({
      protocol: 'dbbackup',
      description: 'DB Backup Protocol',
    });

    // Listen for protocol invocations
    shell.onProtocolInvoked((data) => {
      console.log('Protocol invoked:', data);
      // Handle: dbbackup://backup?database=prod
    });
  }, []);

  return <div>Shell integration active</div>;
}
```

### 4. Platform-Specific Features

#### Windows

```typescript
import { windowsIntegration } from './platform';

// Update Live Tile
await windowsIntegration.liveTiles.updateWithStats({
  totalBackups: 42,
  lastBackup: '2 hours ago',
  nextScheduled: 'Tomorrow at 3 AM',
});

// Update Jump List
await windowsIntegration.jumpList.setDefaultTasks();
await windowsIntegration.jumpList.addRecentBackup('backup-001', 'backup-001-id');

// Set Thumbnail Toolbar buttons
await windowsIntegration.thumbnailToolbar.setBackupControls();
```

#### macOS

```typescript
import { macOSIntegration } from './platform';

// Update Control Center widget
await macOSIntegration.controlCenter.updateWithStatus({
  lastBackup: '2 hours ago',
  nextScheduled: 'Tomorrow at 3 AM',
  totalBackups: 42,
  status: 'idle',
});

// Set Touch Bar buttons (if available)
if (await macOSIntegration.hasTouchBar()) {
  await macOSIntegration.touchBar.setBackupControls();
}

// Update Menu Bar
await macOSIntegration.menuBar.setDefaultMenu();

// Set Finder badge
await macOSIntegration.finderExtension.setBadge('/path/to/backup', 'uploaded');
```

#### Linux

```typescript
import { linuxIntegration } from './platform';

// Initialize based on desktop environment
await linuxIntegration.initialize();

const de = linuxIntegration.getDesktopEnvironment();
console.log('Desktop environment:', de); // 'gnome', 'kde', or 'other'

// For GNOME
if (de === 'gnome') {
  await linuxIntegration.gnome.setDefaultMenu();
}

// For KDE
if (de === 'kde') {
  await linuxIntegration.kde.updateWithStatus({
    lastBackup: '2 hours ago',
    nextScheduled: 'Tomorrow at 3 AM',
    totalBackups: 42,
  });
}

// System tray (works on all DEs)
await linuxIntegration.systemTray.setDefaultTray();
```

### 5. Backup Operations Hook

Combines notifications and dialogs for complete backup workflows:

```typescript
import { useBackupOperations } from './platform/hooks';

function BackupManager() {
  const backup = useBackupOperations();

  const performBackup = async () => {
    const dbName = 'production-db';

    try {
      // Start backup
      await backup.notifyBackupStarted(dbName);

      // Simulate progress
      for (let i = 0; i <= 100; i += 10) {
        await backup.notifyBackupProgress(dbName, i);
        await new Promise(resolve => setTimeout(resolve, 500));
      }

      // Complete
      await backup.notifyBackupComplete(dbName, '150 MB', '5s');
    } catch (error) {
      await backup.notifyBackupFailed(dbName, error.message);
    }
  };

  const performRestore = async () => {
    const backupFile = await backup.selectBackupFile();
    if (!backupFile) return;

    const confirmed = await backup.confirmDatabaseRestore(
      'production-db',
      backupFile,
    );

    if (confirmed) {
      try {
        // Restore logic here
        await backup.notifyRestoreComplete('production-db', backupFile);
      } catch (error) {
        await backup.notifyRestoreFailed('production-db', error.message);
      }
    }
  };

  return (
    <div>
      <button onClick={performBackup}>Backup Database</button>
      <button onClick={performRestore}>Restore Database</button>
    </div>
  );
}
```

## API Reference

### Platform Detection

```typescript
import { detectPlatform, usePlatform } from './platform';

// Function
const platform = detectPlatform(); // 'windows' | 'macos' | 'linux' | 'unknown'

// Hook
const { platform, isWindows, isMacOS, isLinux, capabilities } = usePlatform();
```

### Desktop Integration Manager

```typescript
import { desktopIntegration } from './platform';

// Initialize
await desktopIntegration.initialize(config);

// Check initialization
const isReady = desktopIntegration.isInitialized();

// Get platform
const platform = desktopIntegration.getPlatform();

// Update status
await desktopIntegration.updateStatus({
  state: 'backing_up',
  message: 'Backing up database',
  progress: 50,
});

// Send notification
await desktopIntegration.sendNotification('Title', 'Message', {
  urgency: 'normal',
  actions: [{ id: 'view', title: 'View' }],
});

// Cleanup
await desktopIntegration.cleanup();
```

### Notifications

```typescript
// Send notification
const result = await notifications.send({
  title: 'Backup Complete',
  body: 'Database backed up',
  urgency: 'normal',
  timeout: 5000,
  actions: [{ id: 'view', title: 'View' }],
});

// Convenience methods
await notifications.notifyBackupComplete('db-name', '100MB', '5s');
await notifications.notifyBackupFailed('db-name', 'error message');
await notifications.notifyBackupInProgress('db-name', 50);
await notifications.notifyRestoreComplete('db-name', 'backup-id');
await notifications.notifyRestoreFailed('db-name', 'error message');

// Close notification
await notifications.close(result.id);
```

### Dialogs

```typescript
// File dialogs
const file = await dialogs.openSingleFile(options);
const files = await dialogs.openMultipleFiles(options);
const dir = await dialogs.openDirectory(options);
const savePath = await dialogs.saveFile(options);

// Message dialogs
await dialogs.showInfo('Message', 'Title');
await dialogs.showWarning('Message', 'Title');
await dialogs.showError('Message', 'Title');

// Confirmation dialogs
const confirmed = await dialogs.showConfirm('Question?', {
  title: 'Confirm',
  okLabel: 'Yes',
  cancelLabel: 'No',
});

// Convenience methods
const backupFile = await dialogs.selectBackupFile();
const savePath = await dialogs.selectBackupSaveLocation('backup-name');
const deleteConfirmed = await dialogs.confirmBackupDeletion('backup-001');
const restoreConfirmed = await dialogs.confirmDatabaseRestore('db', 'backup');
```

### Shell Integration

```typescript
// File associations
await shell.registerFileAssociation({
  extension: 'bak',
  mimeType: 'application/x-backup',
  description: 'Database Backup File',
  icon: 'backup',
});

// Context menu
await shell.registerContextMenuItems([
  {
    id: 'backup',
    label: 'Backup File',
    icon: 'backup',
    files: true,
    extensions: ['db', 'sqlite'],
  },
]);

// Protocol handler
await shell.registerProtocolHandler({
  protocol: 'dbbackup',
  description: 'DB Backup Protocol',
});

// Event listeners
shell.onContextMenuAction((data) => {
  console.log('Menu clicked:', data);
});

shell.onProtocolInvoked((data) => {
  console.log('Protocol invoked:', data);
});
```

## Testing

Run the test suite:

```bash
cd desktop
npm test

# Or with UI
npm run test:ui

# With coverage
npm run test:coverage
```

## Platform Requirements

### Windows
- Windows 10 or later for full feature support
- Windows 8.1 for basic features

### macOS
- macOS 11 (Big Sur) or later for modern features
- macOS 10.15 (Catalina) for basic features
- Touch Bar requires MacBook Pro with Touch Bar

### Linux
- GNOME Shell 3.36+ for GNOME extension
- KDE Plasma 5.20+ for Plasma widget
- System tray support for all desktop environments
- D-Bus for notifications

## Platform Capabilities

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| Notifications | ✅ | ✅ | ✅ |
| File Dialogs | ✅ | ✅ | ✅ |
| Context Menus | ✅ | ✅ | ✅ |
| File Associations | ✅ | ✅ | ✅ |
| Protocol Handlers | ✅ | ✅ | ✅ |
| System Tray | ✅ | ✅ | ✅ |
| Live Tiles | ✅ | ❌ | ❌ |
| Jump Lists | ✅ | ❌ | ❌ |
| Thumbnail Toolbar | ✅ | ❌ | ❌ |
| Touch Bar | ❌ | ✅ | ❌ |
| Menu Bar Extra | ❌ | ✅ | ❌ |
| Quick Look | ❌ | ✅ | ❌ |
| Finder Extension | ❌ | ✅ | ❌ |
| Control Center | ❌ | ✅ | ❌ |
| GNOME Extension | ❌ | ❌ | ✅ |
| KDE Widget | ❌ | ❌ | ✅ |

## Best Practices

1. **Always initialize on app start**: Call `initializeDesktopIntegration()` in your main component
2. **Use hooks in React components**: Hooks provide automatic cleanup and state management
3. **Handle errors gracefully**: All methods return promises that may reject
4. **Check platform capabilities**: Use `getCapabilities()` to detect available features
5. **Clean up on app exit**: Call `cleanup()` when the app is closing
6. **Respect user preferences**: Don't spam notifications, use appropriate urgency levels
7. **Test on all platforms**: Each platform has unique behavior and requirements

## Troubleshooting

### Notifications not appearing

- Check platform permissions
- Verify app is not in Do Not Disturb mode
- Ensure notifications are enabled in desktop integration config

### Context menu not showing

- Verify file associations are registered
- Check that the file extension matches the context menu filter
- Restart File Explorer / Finder after registration

### Platform-specific features not working

- Check platform version requirements
- Verify the feature is supported on the current platform
- Check browser console for error messages

## Examples

See the `desktop/src/examples/` directory for complete working examples:

- `NotificationExample.tsx` - Comprehensive notification usage
- `DialogExample.tsx` - File and message dialog examples
- `ShellIntegrationExample.tsx` - Context menus and protocols
- `BackupWorkflowExample.tsx` - Complete backup/restore workflow
- `PlatformFeaturesExample.tsx` - Platform-specific feature demos

## License

This module is part of the DB Backup application and follows the same license.

## Contributing

When adding new platform features:

1. Add the feature to the appropriate platform file (`windows/`, `macos/`, or `linux/`)
2. Update the main integration manager in `index.ts`
3. Create a React hook in `hooks.ts`
4. Add comprehensive tests in `__tests__/`
5. Update this README with usage examples
6. Test on the target platform

## Support

For issues or questions, please file an issue on the main DB Backup repository.
