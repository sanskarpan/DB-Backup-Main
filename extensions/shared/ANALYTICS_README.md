# Extension Analytics

Privacy-focused analytics for the DB Backup Manager browser extension.

## Philosophy

Our analytics implementation prioritizes user privacy:

✅ **What We Track:**
- Feature usage (which features are used)
- Extension lifecycle events (install, update)
- Errors and crashes (to improve stability)
- Performance metrics (to optimize speed)
- Anonymous usage patterns

❌ **What We DON'T Track:**
- Personal information
- URLs visited
- Database names or content
- API keys or credentials
- Email addresses
- Any personally identifiable information (PII)

## Features

- **Privacy-First**: All data is anonymized
- **Local Storage**: Data stored locally by default
- **Opt-Out**: Users can disable analytics in settings
- **Transparent**: Full source code available for review
- **GDPR Compliant**: No personal data collection

## Usage

### Basic Tracking

```javascript
// Track a simple event
await Analytics.track('button_clicked', {
  button: 'save',
  location: 'settings'
});

// Track feature usage
await Analytics.trackFeature('backup', 'quick_backup');

// Track page view
await Analytics.trackPageView('settings');

// Track error
try {
  // Some operation
} catch (error) {
  await Analytics.trackError(error, { context: 'saving_settings' });
}

// Track performance
const startTime = Date.now();
// ... operation ...
const duration = Date.now() - startTime;
await Analytics.trackPerformance('settings_load', duration);
```

### Convenience Functions

```javascript
// Extension lifecycle
await AnalyticsEvents.installed();
await AnalyticsEvents.updated('1.0.1');

// User actions
await AnalyticsEvents.quickBackup();
await AnalyticsEvents.scheduledBackup();
await AnalyticsEvents.viewBackups();
await AnalyticsEvents.openDashboard();

// Settings
await AnalyticsEvents.settingsSaved();
await AnalyticsEvents.connectionTested(true);
await AnalyticsEvents.dataCleared();

// Notifications
await AnalyticsEvents.notificationShown('backup_complete');
await AnalyticsEvents.notificationClicked('backup_complete');

// Context menu
await AnalyticsEvents.contextMenuUsed('quick_backup');

// Content script
await AnalyticsEvents.toolDetected('phpMyAdmin');
await AnalyticsEvents.floatingButtonClicked();

// Errors
await AnalyticsEvents.apiError('/api/backups', 404);
await AnalyticsEvents.storageError('get');
```

## Integration

### In Background Script

```javascript
// Load analytics
importScripts('shared/analytics.js');

// Track events
chrome.runtime.onInstalled.addListener(async (details) => {
  if (details.reason === 'install') {
    await AnalyticsEvents.installed();
  } else if (details.reason === 'update') {
    await AnalyticsEvents.updated(details.previousVersion);
  }
});

// Track feature usage
async function handleQuickBackup() {
  await AnalyticsEvents.quickBackup();
  // ... backup logic ...
}
```

### In Popup

```javascript
// Load analytics (already loaded via shared scripts)

// Track page view
await Analytics.trackPageView('popup');

// Track button click
document.getElementById('quickBackupBtn').addEventListener('click', async () => {
  await AnalyticsEvents.quickBackup();
  // ... backup logic ...
});
```

### In Options Page

```javascript
// Track page view
await Analytics.trackPageView('options');

// Track settings saved
document.getElementById('saveBtn').addEventListener('click', async () => {
  // ... save logic ...
  await AnalyticsEvents.settingsSaved();
});
```

### In Content Script

```javascript
// Track tool detection
if (detectedTool) {
  await AnalyticsEvents.toolDetected(detectedTool.name);
}

// Track floating button click
floatingButton.addEventListener('click', async () => {
  await AnalyticsEvents.floatingButtonClicked();
});
```

## Configuration

### Enable/Disable Analytics

```javascript
// Disable analytics
await Analytics.setEnabled(false);

// Enable analytics
await Analytics.setEnabled(true);
```

### Remote Endpoint (Optional)

```javascript
// Configure remote endpoint for centralized analytics
Analytics.config.endpoint = 'https://analytics.example.com/api/track';
Analytics.config.localOnly = false;
await Analytics.saveConfig();
```

## Viewing Analytics Data

### Get Summary

```javascript
// Get summary for last 7 days
const summary = await Analytics.getSummary(7);

console.log('Total events:', summary.totalEvents);
console.log('Unique sessions:', summary.uniqueSessions);
console.log('Feature usage:', summary.features);
console.log('Errors:', summary.errors);
```

### Get All Events

```javascript
// Get all events
const events = await Analytics.getEvents();

// Get filtered events
const backupEvents = await Analytics.getEvents({
  name: 'feature_used',
  'properties.feature': 'backup'
});
```

### Export Data

```javascript
// Export all analytics data
const data = await Analytics.export();

// Convert to JSON
const json = JSON.stringify(data, null, 2);

// Download as file
const blob = new Blob([json], { type: 'application/json' });
const url = URL.createObjectURL(blob);
const a = document.createElement('a');
a.href = url;
a.download = 'analytics-export.json';
a.click();
```

### Clear Data

```javascript
// Clear all analytics data
await Analytics.clear();
```

## Privacy Policy Compliance

This analytics implementation is designed to be compliant with privacy regulations:

### GDPR Compliance

- ✅ No personal data collection
- ✅ Anonymous user IDs
- ✅ Opt-out capability
- ✅ Data export functionality
- ✅ Data deletion functionality
- ✅ Transparent data collection

### User Rights

Users can:
1. **Opt-out**: Disable analytics in extension settings
2. **View data**: See what data is collected
3. **Export data**: Download all their analytics data
4. **Delete data**: Clear all analytics data

## Data Structure

### Event Format

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "feature_used",
  "properties": {
    "feature": "backup",
    "action": "quick_backup"
  },
  "anonymousId": "user-12345678",
  "sessionId": "session-87654321",
  "timestamp": "2026-01-13T12:34:56.789Z",
  "version": "1.0.0",
  "browser": "chrome",
  "platform": "macos"
}
```

### Sanitization Example

Before:
```javascript
{
  apiUrl: 'https://api.example.com/backups',
  apiKey: 'sk_test_abc123def456',
  email: 'user@example.com',
  database: 'production_db'
}
```

After sanitization:
```javascript
{
  // Sensitive fields removed or sanitized
}
```

## Best Practices

### 1. Track User Intent, Not Details

```javascript
// Good: Track what the user is trying to do
await Analytics.trackFeature('backup', 'created');

// Bad: Track sensitive details
await Analytics.track('backup', { database: 'production' });
```

### 2. Use Appropriate Event Names

```javascript
// Good: Clear, hierarchical event names
await Analytics.track('feature_used', { feature: 'backup', action: 'create' });

// Bad: Vague event names
await Analytics.track('click', { button: 'btn1' });
```

### 3. Provide Context Without Sensitivity

```javascript
// Good: Generic context
await Analytics.trackError(error, { context: 'api_call_failed' });

// Bad: Sensitive context
await Analytics.trackError(error, { endpoint: 'https://api.example.com/users/123' });
```

### 4. Track Performance for Optimization

```javascript
// Measure and track performance
const start = performance.now();
await someOperation();
const duration = performance.now() - start;
await Analytics.trackPerformance('operation_duration', duration);
```

## Testing

### Local Testing

```javascript
// Enable local-only mode
Analytics.config.localOnly = true;

// Track some events
await Analytics.track('test_event');

// View events
const events = await Analytics.getEvents();
console.log('Tracked events:', events);

// Clear test data
await Analytics.clear();
```

### Development Mode

Add to extension settings:

```javascript
// Disable analytics in development
if (process.env.NODE_ENV === 'development') {
  await Analytics.setEnabled(false);
}
```

## Troubleshooting

### Analytics not working

1. Check if analytics is enabled:
```javascript
console.log('Analytics enabled:', Analytics.config.enabled);
```

2. Check for errors in console:
```javascript
// Enable verbose logging
Analytics.debug = true;
```

3. Verify storage permissions:
```json
// In manifest.json
"permissions": ["storage"]
```

### Events not appearing

1. Check storage:
```javascript
const events = await Analytics.getStorage('analytics_events');
console.log('Stored events:', events);
```

2. Verify initialization:
```javascript
// Manually initialize if needed
await Analytics.init();
```

## Future Enhancements

Potential additions (while maintaining privacy):

- [ ] Session replay (visual, not sensitive data)
- [ ] Funnel analysis (feature adoption flow)
- [ ] Cohort analysis (anonymous user groups)
- [ ] A/B testing support
- [ ] Crash reporting (with stack traces)
- [ ] Heatmaps (UI interaction patterns)

## Support

For questions or issues:
1. Check this README
2. Review the source code (`analytics.js`)
3. Open an issue on GitHub
4. Contact support

## License

Same license as the main project.
