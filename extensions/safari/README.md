# DB Backup Manager - Safari Web Extension

This is the Safari version of the DB Backup Manager browser extension.

## Important Notes

Safari Web Extensions require:
- **macOS 11 (Big Sur) or later** for development
- **Xcode 12 or later** installed from the Mac App Store
- **Apple Developer account** for distribution (free for development)
- **Safari 14 or later** for running the extension

## Architecture

Safari Web Extensions use a hybrid approach:
1. **Web Extension** - The browser extension code (HTML/CSS/JavaScript)
2. **Native App Wrapper** - A macOS app that contains the extension (required by Safari)
3. **Swift Code** - Optional native code for additional functionality

## Setup Process

### Step 1: Install Xcode

```bash
# Install Xcode from Mac App Store or
xcode-select --install
```

### Step 2: Create Extension Using Xcode

You have two options:

#### Option A: Convert Chrome Extension (Recommended)

```bash
# Run the conversion script
./convert-to-safari.sh
```

This script uses Apple's `xcrun safari-web-extension-converter` tool to automatically convert the Chrome extension.

#### Option B: Manual Setup

1. Open Xcode
2. File > New > Project
3. Choose "Safari Extension App" under macOS
4. Name: "DB Backup Manager"
5. Organization Identifier: "com.yourcompany.dbbackup"
6. Replace the generated extension files with ours

### Step 3: Build Extension Files

```bash
# Copy extension files from Chrome
./build.sh
```

### Step 4: Configure Xcode Project

1. Open the `.xcodeproj` file in Xcode
2. Select your extension target
3. In "Signing & Capabilities":
   - Enable "Automatically manage signing"
   - Select your development team
4. Update the bundle identifier if needed

### Step 5: Run and Test

1. In Xcode, select the scheme (DB Backup Manager)
2. Click Run (⌘R) or Product > Run
3. Safari will launch with the extension
4. Enable the extension in Safari Preferences > Extensions

## Build Script

The `build.sh` script copies files from the Chrome extension:

```bash
./build.sh
```

This copies:
- Shared JavaScript libraries
- Background scripts
- Popup HTML/CSS/JS
- Options HTML/CSS/JS
- Content scripts
- Icons

## Packaging for Distribution

### For Testing (Development)

1. Build the project in Xcode
2. The app will be in `DerivedData` folder
3. Distribute the `.app` bundle to testers

### For App Store Distribution

1. Archive the project in Xcode
2. Validate the archive
3. Submit to App Store Connect
4. Wait for review (typically 1-2 days)

```bash
# Package using Xcode command line
./package.sh
```

## Safari-Specific Considerations

### API Differences

Safari supports most WebExtensions APIs, but with some differences:

| Feature | Chrome | Safari |
|---------|--------|--------|
| Namespace | `chrome.*` | `browser.*` |
| Promises | Callbacks (with promises in V3) | Native promises |
| Background | Service Worker (V3) | Non-persistent background page |
| Storage | Unlimited | Limited to 10MB |
| Alarms | Full support | Minimum 1 minute interval |

### Known Limitations

1. **Storage Limits**: Safari limits storage to 10MB
2. **Manifest V3**: Safari supports Manifest V2 and V3 (V2 recommended for now)
3. **Native Messaging**: Different from Chrome, uses XPC
4. **Push Notifications**: Must use native app, not web push

### Handling Differences

Our shared `utils.js` handles most differences automatically:

```javascript
const browserAPI = Utils.getBrowserAPI(); // Returns 'browser' on Safari
```

## File Structure

```
safari/
├── manifest.json              # Safari-compatible manifest (V2)
├── build.sh                   # Build script (copies from Chrome)
├── convert-to-safari.sh       # Convert Chrome extension to Safari
├── package.sh                 # Package for distribution
├── README.md                  # This file
│
├── DBBackupManager/           # Created by Xcode
│   ├── DBBackupManager.xcodeproj
│   ├── DBBackupManager/       # macOS app wrapper
│   │   ├── AppDelegate.swift
│   │   ├── ViewController.swift
│   │   └── Info.plist
│   │
│   └── Extension/             # Web extension files go here
│       ├── manifest.json
│       ├── background/
│       ├── popup/
│       ├── options/
│       ├── content/
│       ├── shared/
│       └── icons/
│
└── Supporting Files/
    └── Resources/
```

## Development Workflow

### 1. Make Changes

Edit files in the `chrome/` directory (our source of truth).

### 2. Rebuild Safari Extension

```bash
cd extensions/safari
./build.sh
```

### 3. Test in Xcode

1. Open the Xcode project
2. Click Run (⌘R)
3. Safari launches with extension
4. Test the changes

### 4. Debug

- Use Safari Web Inspector for web extension code
- Use Xcode debugger for native app code
- Check Console.app for extension logs

## Common Issues

### Issue: Extension doesn't appear in Safari

**Solution**:
1. Enable "Develop" menu in Safari Preferences
2. Check "Allow Unsigned Extensions"
3. Restart Safari

### Issue: Storage not working

**Solution**:
- Safari has stricter storage limits (10MB)
- Use `browser.storage.local` not `localStorage`
- Check storage usage in Web Inspector

### Issue: Background script not running

**Solution**:
- Ensure `persistent: false` in manifest
- Safari may suspend background pages more aggressively
- Use alarms API to wake up

### Issue: Can't submit to App Store

**Solution**:
1. Ensure valid bundle identifier
2. Add privacy policy URL
3. Follow App Store Review Guidelines
4. Include required screenshots (macOS app)

## App Store Submission

### Requirements

1. **Apple Developer Account** ($99/year)
2. **App Store Connect** access
3. **Privacy Policy** (if collecting data)
4. **Support URL**
5. **macOS screenshots** (required even for Safari extension)
6. **App icon** (macOS app icon, 512x512 and 1024x1024)

### Submission Steps

1. **Prepare Metadata**
   - App name: "DB Backup Manager"
   - Category: Developer Tools or Productivity
   - Description: Detailed description
   - Keywords: database, backup, management, etc.
   - Screenshots: macOS app screenshots

2. **Archive in Xcode**
   - Product > Archive
   - Wait for archive to complete
   - Open Organizer

3. **Validate Archive**
   - Click "Validate App"
   - Fix any issues
   - Re-archive if needed

4. **Submit**
   - Click "Distribute App"
   - Choose "App Store Connect"
   - Upload to App Store Connect
   - Fill in metadata in App Store Connect
   - Submit for review

5. **Review Process**
   - Typically 1-2 days
   - May ask for demo account
   - May require privacy policy

## Native App Wrapper

The Safari extension requires a macOS app wrapper. Here's a minimal `AppDelegate.swift`:

```swift
import Cocoa
import SafariServices

@NSApplicationMain
class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ aNotification: Notification) {
        // Check extension state
        SFSafariExtensionManager.getStateOfSafariExtension(
            withIdentifier: "com.yourcompany.dbbackup.Extension"
        ) { (state, error) in
            if let state = state {
                print("Extension is \\(state.isEnabled ? "enabled" : "disabled")")
            }
        }
    }
}
```

## Advanced Features

### Native Messaging

For advanced features that require native code:

1. Add XPC service to Xcode project
2. Implement message handling in Swift
3. Use `browser.runtime.sendNativeMessage()` from JavaScript

### Toolbar Icon Updates

Safari requires native code to update toolbar icons:

```swift
SFSafariApplication.dispatchMessage(
    messageName: "updateBadge",
    toExtensionWithIdentifier: extensionIdentifier
) { (error) in
    if let error = error {
        print("Error: \\(error)")
    }
}
```

## Resources

- [Safari Web Extensions Documentation](https://developer.apple.com/documentation/safariservices/safari_web_extensions)
- [Converting a Web Extension](https://developer.apple.com/documentation/safariservices/safari_web_extensions/converting_a_web_extension_for_safari)
- [Safari Web Extensions Guide](https://developer.apple.com/videos/play/wwdc2021/10104/)
- [WebExtensions API Support](https://developer.apple.com/documentation/safariservices/safari_web_extensions/assessing_your_safari_web_extension_s_browser_compatibility)

## Support

For Safari-specific issues:
1. Check Safari Web Extensions documentation
2. Use Safari Developer Forums
3. Contact Apple Developer Support (if enrolled)

## Notes

- Safari Web Extensions are the future of Safari extensions
- Legacy Safari Extensions (.safariextz) are deprecated
- Safari 14+ required for Web Extensions
- macOS 11+ required for development
- The extension will also work on iOS Safari 15+ (with some modifications)
