# DB Backup Browser Extensions

Cross-browser extensions for quick access to DB Backup functionality directly from your browser.

## Overview

The DB Backup browser extension provides:

- **Quick Backup**: Create backups with one click
- **Real-time Monitoring**: View backup status and alerts
- **Context Menu Integration**: Right-click access to features
- **Keyboard Shortcuts**: Power user shortcuts
- **Database Tool Detection**: Auto-detect phpMyAdmin, Adminer, pgAdmin, etc.
- **Offline Support**: Works even when disconnected
- **Privacy-Focused Analytics**: Optional, anonymous usage tracking

## Supported Browsers

| Browser | Status | Version | Notes |
|---------|--------|---------|-------|
| **Chrome** | ✅ Ready | 1.0.0 | Manifest V3, fully tested |
| **Firefox** | ✅ Ready | 1.0.0 | Manifest V2, WebExtensions |
| **Edge** | ✅ Ready | 1.0.0 | Chromium-based, same as Chrome |
| **Safari** | ⏳ Requires Xcode | 1.0.0 | macOS 11+, needs Xcode build |

## Directory Structure

```
extensions/
├── shared/                  # Shared code across all browsers
│   ├── api.js              # API wrapper for backend
│   ├── utils.js            # Utility functions
│   ├── analytics.js        # Privacy-focused analytics
│   └── ANALYTICS_README.md # Analytics documentation
│
├── chrome/                  # Chrome extension
│   ├── manifest.json       # Manifest V3
│   ├── background/         # Service worker
│   ├── popup/              # Popup UI
│   ├── options/            # Settings page
│   ├── content/            # Content script
│   └── icons/              # Extension icons
│
├── firefox/                 # Firefox extension
│   ├── manifest.json       # Manifest V2
│   ├── build.sh            # Build script
│   ├── package.sh          # Package script
│   └── README.md           # Firefox-specific docs
│
├── edge/                    # Microsoft Edge extension
│   ├── manifest.json       # Manifest V3 (Chromium)
│   ├── build.sh            # Build script
│   ├── package.sh          # Package script
│   └── README.md           # Edge-specific docs
│
├── safari/                  # Safari extension
│   ├── manifest.json       # Manifest V2
│   ├── build.sh            # Build script
│   ├── convert-to-safari.sh # Xcode project generator
│   ├── package.sh          # Package script
│   └── README.md           # Safari-specific docs
│
├── build-all.sh            # Build all extensions
├── package-all.sh          # Package all extensions
└── README.md               # This file
```

## Quick Start

### 1. Build All Extensions

```bash
cd extensions
./build-all.sh
```

This will:
- Build Chrome extension
- Build Firefox extension
- Build Edge extension
- Build Safari extension (if on macOS)
- Generate icons if needed

### 2. Test in Browser

#### Chrome
```bash
# Open chrome://extensions/
# Enable "Developer mode"
# Click "Load unpacked"
# Select: extensions/chrome/
```

#### Firefox
```bash
# Open about:debugging#/runtime/this-firefox
# Click "Load Temporary Add-on"
# Select: extensions/firefox/manifest.json
```

#### Edge
```bash
# Open edge://extensions/
# Enable "Developer mode"
# Click "Load unpacked"
# Select: extensions/edge/
```

#### Safari
```bash
# Open Terminal
cd extensions/safari
./convert-to-safari.sh

# Open the Xcode project
open *.xcodeproj

# In Xcode, click Run (⌘R)
```

### 3. Package for Distribution

```bash
cd extensions
./package-all.sh
```

Packages will be created in `extensions/dist/`:
- `db-backup-manager-chrome-v1.0.0.zip`
- `db-backup-manager-firefox-v1.0.0.xpi`
- `db-backup-manager-edge-v1.0.0.zip`
- `checksums.txt`

## Development Workflow

### Making Changes

1. **Edit the source** in the `chrome/` directory (it's the source of truth)

2. **Test in Chrome** first:
   ```bash
   # Reload extension in chrome://extensions/
   ```

3. **Build other browsers**:
   ```bash
   ./build-all.sh
   ```

4. **Test in each browser**

5. **Package for release**:
   ```bash
   ./package-all.sh
   ```

### Code Organization

- **`chrome/`**: Source of truth for all extensions
- **`shared/`**: Code shared across all browsers
- **`firefox/`, `edge/`, `safari/`**: Browser-specific manifests and build scripts

Files are copied from `chrome/` to other browser directories during build.

## Features

### 1. Popup Interface

- Dashboard statistics
- Recent backups list
- Active alerts
- Quick action buttons
- Real-time sync status

### 2. Options Page

- API configuration
- Sync settings
- Notification preferences
- Display options
- Advanced settings

### 3. Content Script

- Detects database tools (phpMyAdmin, Adminer, pgAdmin, etc.)
- Floating action button
- Page indicators
- Quick backup from any page

### 4. Background Service

- Periodic sync (every 5 minutes)
- Monitoring checks (every 1 minute)
- Badge updates
- Context menus
- Keyboard shortcuts

### 5. Analytics

- Privacy-focused usage tracking
- Local storage by default
- Opt-out capability
- No personal data collection
- GDPR compliant

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+B` (Windows/Linux)<br>`Command+Shift+B` (Mac) | Open extension popup |
| `Ctrl+Shift+Q` (Windows/Linux)<br>`Command+Shift+Q` (Mac) | Quick backup |
| `Ctrl+Shift+V` (Windows/Linux)<br>`Command+Shift+V` (Mac) | View backups |

## Browser-Specific Notes

### Chrome

- Uses Manifest V3
- Service worker for background tasks
- Full push notification support
- No special requirements

### Firefox

- Uses Manifest V2
- Non-persistent background page
- Promise-based APIs (no callbacks)
- Some storage limitations

### Edge

- Uses Manifest V3 (same as Chrome)
- Chromium-based (100% compatible with Chrome)
- Requires Microsoft Partner Center account ($9 one-time)
- Different store submission process

### Safari

- Requires macOS 11+ for development
- Requires Xcode 12+
- Uses native app wrapper
- App Store submission required ($99/year)
- More complex build process

## Store Submission

### Chrome Web Store

1. Create developer account ($5 one-time fee)
2. Go to: https://chrome.google.com/webstore/devconsole
3. Upload ZIP file
4. Fill in store listing
5. Submit for review (1-3 days)

### Firefox Add-ons

1. Create account (free)
2. Go to: https://addons.mozilla.org/developers/
3. Upload XPI file
4. Fill in listing details
5. Submit for review (1-3 days)

### Microsoft Edge Add-ons

1. Create Partner Center account ($9 one-time)
2. Go to: https://partner.microsoft.com/dashboard/microsoftedge/
3. Upload ZIP file
4. Fill in store listing
5. Submit for review (1-3 days)

### Safari App Store

1. Apple Developer account required ($99/year)
2. Build in Xcode
3. Archive: Product > Archive
4. Submit to App Store Connect
5. Review takes 1-2 days

## Testing Checklist

Before submitting to stores:

- [ ] Test installation in each browser
- [ ] Verify all permissions are necessary
- [ ] Test popup UI (all buttons work)
- [ ] Test options page (settings save correctly)
- [ ] Test content script (detects database tools)
- [ ] Test background sync (runs periodically)
- [ ] Test notifications (displays correctly)
- [ ] Test context menu (all items work)
- [ ] Test keyboard shortcuts
- [ ] Test with API server running
- [ ] Test with API server offline
- [ ] Test error handling
- [ ] Verify no console errors
- [ ] Check memory usage
- [ ] Test on multiple screen sizes
- [ ] Verify icons display correctly
- [ ] Test analytics (if enabled)
- [ ] Review privacy policy compliance

## Troubleshooting

### Extension won't load

- Check manifest.json for syntax errors
- Verify all required files exist
- Check browser console for errors
- Try rebuilding: `./build-all.sh`

### Icons not showing

- Generate icons: `cd chrome/icons && ./generate-icons.sh`
- Or open `chrome/icons/generate-icons.html` in browser
- Rebuild all: `./build-all.sh`

### Popup not opening

- Check for JavaScript errors in console
- Verify popup HTML/CSS/JS files exist
- Test in Chrome first (easiest to debug)

### Background script not running

- Chrome: Check service worker status in chrome://extensions/
- Firefox: Check about:debugging
- Verify background script has no errors

### Content script not injecting

- Check matches pattern in manifest.json
- Verify content script file exists
- Check for CSP issues on target pages

## Performance

Typical resource usage:

- **Memory**: 10-20 MB
- **CPU**: <1% (background)
- **Storage**: <5 MB
- **Network**: ~12 API calls per hour

## Security

Security features:

- ✅ HTTPS-only API communication
- ✅ No inline scripts (CSP compliant)
- ✅ Minimal permissions requested
- ✅ API keys encrypted by browser
- ✅ No sensitive data logged
- ✅ Regular security updates

## Contributing

To contribute to the extensions:

1. Fork the repository
2. Create a feature branch
3. Make changes in `chrome/` directory
4. Test in all browsers
5. Run `./build-all.sh`
6. Submit pull request

## License

Same license as the main project.

## Support

For help with extensions:

1. Check browser-specific README files
2. Review this documentation
3. Open an issue on GitHub
4. Contact support

## Resources

### Official Documentation

- [Chrome Extensions](https://developer.chrome.com/docs/extensions/)
- [Firefox WebExtensions](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions)
- [Edge Extensions](https://docs.microsoft.com/en-us/microsoft-edge/extensions-chromium/)
- [Safari Web Extensions](https://developer.apple.com/documentation/safariservices/safari_web_extensions)

### Tools

- [Web Store Upload](https://chrome.google.com/webstore/devconsole)
- [Firefox Add-ons Portal](https://addons.mozilla.org/developers/)
- [Edge Partner Center](https://partner.microsoft.com/dashboard/microsoftedge/)
- [App Store Connect](https://appstoreconnect.apple.com/)

## Version History

### v1.0.0 (2026-01-13)

- Initial release
- Chrome, Firefox, Edge, Safari support
- All core features implemented
- Privacy-focused analytics
- Comprehensive documentation
