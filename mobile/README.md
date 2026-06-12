# DB Backup Mobile

Cross-platform mobile application for DB Backup built with React Native.

## Features

- Native iOS and Android applications
- Offline-first architecture with local SQLite caching
- Background sync and backup scheduling
- Push notifications for backup status
- Biometric authentication
- Dark mode support
- Multi-language support
- Network-aware operations (WiFi-only mode)
- Battery optimization
- Location-based backup triggers

## Tech Stack

- **Framework**: React Native 0.73
- **Navigation**: React Navigation (Stack + Bottom Tabs)
- **State Management**: Redux Toolkit
- **Local Storage**: AsyncStorage + SQLite
- **Networking**: Axios
- **Icons**: React Native Vector Icons
- **Testing**: Jest + React Native Testing Library

## Shared Packages

This project uses shared packages from `@db-backup/*`:

- `@db-backup/types` - TypeScript type definitions
- `@db-backup/api-client` - API client library
- `@db-backup/utils` - Utility functions

## Development

### Prerequisites

**General:**
- Node.js 18+
- npm 9+
- React Native CLI

**iOS:**
- macOS with Xcode 14+
- CocoaPods
- iOS Simulator or physical device

**Android:**
- Android Studio
- Java JDK 17
- Android SDK (API level 33+)
- Android emulator or physical device

### Installation

```bash
npm install

# iOS only - install CocoaPods dependencies
cd ios && pod install && cd ..
```

### Environment Setup

Create a `.env` file:

```env
API_URL=http://localhost:8080/api
ENABLE_PUSH_NOTIFICATIONS=true
ENABLE_BACKGROUND_SYNC=true
```

### Run on iOS

```bash
# Start Metro bundler
npm start

# Run on iOS simulator (in a new terminal)
npm run ios

# Run on specific simulator
npm run ios -- --simulator="iPhone 15 Pro"

# Run on physical device
npm run ios -- --device
```

### Run on Android

```bash
# Start Metro bundler
npm start

# Run on Android emulator/device (in a new terminal)
npm run android

# Run on specific device
npm run android -- --deviceId=<device-id>
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage
```

### Linting

```bash
npm run lint
```

## Project Structure

```
db-backup-mobile/
├── android/                # Android native code
├── ios/                    # iOS native code
├── src/
│   ├── screens/           # Screen components
│   │   └── HomeScreen.tsx
│   ├── services/          # Services
│   │   └── offline.ts     # Offline sync service
│   ├── store/             # Redux store
│   │   ├── index.ts
│   │   └── backupsSlice.ts
│   ├── features/          # Feature modules
│   │   └── advanced/      # Advanced features
│   │       ├── backgroundService.ts
│   │       ├── batteryService.ts
│   │       ├── locationService.ts
│   │       └── networkPolicyService.ts
│   └── __tests__/         # Test files
├── App.tsx                # Root component
├── index.js               # App entry point
├── app.json               # App configuration
├── package.json
├── metro.config.js        # Metro bundler config
├── babel.config.js        # Babel config
└── tsconfig.json          # TypeScript config
```

## Mobile-Specific Features

### Offline Support
- Local SQLite database for caching backups
- Automatic sync when network is available
- Queue failed requests for retry
- Offline indicator

### Background Sync
- Periodic background fetch (iOS)
- Background services (Android)
- Sync backups when app is in background
- Configurable sync intervals

### Push Notifications
- Real-time backup status updates
- Failed backup alerts
- Schedule reminders
- Silent notifications for background updates

### Network Policies
- WiFi-only mode for large backups
- Cellular data usage limits
- Automatic pause on low signal
- Network type detection

### Battery Optimization
- Reduce sync frequency on low battery
- Skip non-critical operations
- Background activity optimization
- Power-saving mode detection

### Location-Based Features
- Trigger backups based on geofence
- Location-aware backup scheduling
- Office/home location detection

### Biometric Authentication
- Face ID / Touch ID (iOS)
- Fingerprint / Face Unlock (Android)
- Secure credential storage
- Fallback to PIN/password

## Platform-Specific Setup

### iOS Setup

1. **Install CocoaPods dependencies:**
   ```bash
   cd ios && pod install
   ```

2. **Configure signing:**
   - Open `ios/DBBackup.xcworkspace` in Xcode
   - Select your development team
   - Update bundle identifier if needed

3. **Permissions (Info.plist):**
   - Location: `NSLocationWhenInUseUsageDescription`
   - Notifications: Configured automatically
   - Biometrics: `NSFaceIDUsageDescription`

### Android Setup

1. **Configure signing:**
   - Generate keystore: `keytool -genkey -v -keystore release.keystore -alias db-backup -keyalg RSA -keysize 2048 -validity 10000`
   - Update `android/gradle.properties` with keystore info

2. **Permissions (AndroidManifest.xml):**
   - Already configured in the manifest
   - Internet, location, notifications, biometric

3. **Enable Hermes (default in RN 0.73):**
   - Already enabled for better performance

## Building for Production

### iOS

```bash
# Build archive
npm run ios -- --configuration Release

# Or use Xcode:
# 1. Open ios/DBBackup.xcworkspace
# 2. Product > Archive
# 3. Distribute App > App Store Connect
```

### Android

```bash
# Build release APK
cd android && ./gradlew assembleRelease

# Build release AAB (for Play Store)
cd android && ./gradlew bundleRelease

# Output:
# APK: android/app/build/outputs/apk/release/app-release.apk
# AAB: android/app/build/outputs/bundle/release/app-release.aab
```

## Distribution

### iOS - TestFlight

1. Archive the app in Xcode
2. Upload to App Store Connect
3. Add to TestFlight
4. Invite internal/external testers

### Android - Google Play

1. Build release AAB
2. Upload to Google Play Console
3. Create a release (Internal/Alpha/Beta/Production)
4. Roll out to testers/users

## Debugging

### React Native Debugger

```bash
# Install standalone debugger
brew install --cask react-native-debugger

# Or use built-in debugger
# Shake device/emulator > Debug
```

### Reactotron

```bash
npm install --save-dev reactotron-react-native
```

### Flipper

Pre-installed with React Native 0.73. Launch Flipper and connect to running app.

## Troubleshooting

### iOS Build Issues

```bash
# Clean build
cd ios && rm -rf Pods Podfile.lock && pod install

# Clean Xcode cache
rm -rf ~/Library/Developer/Xcode/DerivedData
```

### Android Build Issues

```bash
# Clean Gradle cache
cd android && ./gradlew clean

# Clear build folder
rm -rf android/app/build
```

### Metro Bundler Issues

```bash
# Clear Metro cache
npm start -- --reset-cache

# Or
rm -rf $TMPDIR/metro-*
```

## Performance Optimization

- Use Hermes engine (enabled by default)
- Enable RAM bundles for faster startup
- Use React.memo for expensive components
- Implement FlatList virtualization
- Optimize images (WebP, compression)
- Use native driver for animations

## Security

- Sensitive data stored in Keychain (iOS) / Keystore (Android)
- API tokens encrypted
- SSL pinning for API calls
- Biometric authentication
- Automatic logout on inactivity
- Secure storage for credentials

## License

MIT
