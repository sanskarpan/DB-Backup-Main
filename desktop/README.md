# DB Backup Desktop

Cross-platform desktop application for DB Backup built with Tauri and React.

## Features

- Native desktop application for Windows, macOS, and Linux
- Built with Tauri (Rust backend + React frontend)
- Lightweight and fast (~10MB installer)
- Offline-first with local data caching
- System tray integration
- Keyboard shortcuts and command palette
- Multi-language support (i18n)
- Export backups to PDF, Excel, JSON
- Advanced features:
  - Auto-update support
  - Native file system access
  - System notifications
  - Background sync
  - Custom themes

## Tech Stack

- **Framework**: Tauri 1.5 (Rust + WebView)
- **Frontend**: React 18 + TypeScript
- **UI**: Radix UI + Tailwind CSS
- **State**: Zustand
- **Build**: Vite 5
- **Testing**: Vitest

## Shared Packages

This project uses shared packages from `@db-backup/*`:

- `@db-backup/types` - TypeScript type definitions
- `@db-backup/api-client` - API client library
- `@db-backup/utils` - Utility functions

## Development

### Prerequisites

- Node.js 18+
- npm 9+
- Rust 1.70+ (for Tauri)
- Platform-specific requirements:
  - **macOS**: Xcode Command Line Tools
  - **Linux**: `build-essential`, `libwebkit2gtk-4.0-dev`, `libssl-dev`, `libgtk-3-dev`, `libayatana-appindicator3-dev`, `librsvg2-dev`
  - **Windows**: Microsoft Visual Studio C++ Build Tools

### Installation

```bash
npm install
```

### Run Development Server

```bash
# Start Vite dev server + Tauri dev window
npm run tauri:dev

# Or just the frontend (no Tauri window)
npm run dev
```

### Build for Production

```bash
# Build frontend and Tauri app
npm run tauri:build
```

This will create platform-specific installers in `src-tauri/target/release/bundle/`:
- **macOS**: `.dmg` and `.app`
- **Linux**: `.deb`, `.AppImage`
- **Windows**: `.msi` and `.exe`

### Testing

```bash
# Run tests
npm test

# Run tests with UI
npm run test:ui

# Run tests with coverage
npm run test:coverage
```

## Project Structure

```
db-backup-desktop/
├── src/                    # React frontend source
│   ├── App.tsx            # Main app component
│   ├── EnhancedApp.tsx    # Enhanced features wrapper
│   ├── main.tsx           # Entry point
│   ├── i18n/              # Internationalization
│   └── __tests__/         # Frontend tests
├── src-tauri/             # Tauri (Rust) backend
│   ├── src/
│   │   └── main.rs        # Rust main entry
│   ├── Cargo.toml         # Rust dependencies
│   ├── tauri.conf.json    # Tauri configuration
│   └── build.rs           # Build script
├── platform/              # Platform-specific configs
├── dist/                  # Build output (frontend)
├── index.html             # HTML template
├── vite.config.ts         # Vite configuration
├── tailwind.config.js     # Tailwind CSS config
└── package.json
```

## Desktop-Specific Features

### System Tray
The app runs in the system tray with quick access to:
- Start/stop backups
- View backup status
- Open main window
- Quit application

### Keyboard Shortcuts
- `Cmd/Ctrl + K` - Open command palette
- `Cmd/Ctrl + N` - New backup
- `Cmd/Ctrl + R` - Refresh data
- `Cmd/Ctrl + ,` - Open settings
- `Cmd/Ctrl + Q` - Quit application

### Command Palette
Press `Cmd/Ctrl + K` to open the command palette for quick navigation and actions.

### Export Features
Export backups and reports to:
- **PDF** - Formatted backup reports with jsPDF
- **Excel** - Spreadsheet exports with XLSX
- **JSON** - Raw data exports

### Internationalization
Supported languages:
- English (en)
- Spanish (es)
- French (fr)
- German (de)
- Japanese (ja)
- Chinese (zh)

Add new languages in `src/i18n/config.ts`.

## Platform-Specific Build Instructions

### macOS

```bash
# Install Xcode Command Line Tools
xcode-select --install

# Build
npm run tauri:build

# Output: src-tauri/target/release/bundle/macos/
```

### Linux

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt update
sudo apt install libwebkit2gtk-4.0-dev \
    build-essential \
    curl \
    wget \
    file \
    libssl-dev \
    libgtk-3-dev \
    libayatana-appindicator3-dev \
    librsvg2-dev

# Build
npm run tauri:build

# Output: src-tauri/target/release/bundle/
```

### Windows

```bash
# Install Microsoft Visual Studio C++ Build Tools
# https://visualstudio.microsoft.com/visual-cpp-build-tools/

# Build
npm run tauri:build

# Output: src-tauri/target/release/bundle/msi/
```

## Configuration

### Tauri Configuration

Edit `src-tauri/tauri.conf.json` to configure:
- App name and version
- Window size and behavior
- System tray settings
- Update server
- Build targets

### Environment Variables

Create a `.env.local` file:

```env
VITE_API_URL=http://localhost:8080/api
VITE_ENABLE_AUTO_UPDATE=true
```

## Distribution

### Code Signing (macOS)

```bash
# Set signing identity
export APPLE_SIGNING_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"

# Build with signing
npm run tauri:build
```

### Notarization (macOS)

Required for distribution outside the Mac App Store. Configure in `tauri.conf.json`:

```json
{
  "tauri": {
    "bundle": {
      "macOS": {
        "signingIdentity": "Developer ID Application: Your Name (TEAM_ID)",
        "entitlements": "path/to/entitlements.plist"
      }
    }
  }
}
```

## Auto-Update

Tauri supports auto-updates. Configure update server in `tauri.conf.json`:

```json
{
  "tauri": {
    "updater": {
      "active": true,
      "endpoints": [
        "https://releases.myapp.com/{{target}}/{{current_version}}"
      ],
      "dialog": true,
      "pubkey": "YOUR_PUBLIC_KEY"
    }
  }
}
```

## License

MIT
