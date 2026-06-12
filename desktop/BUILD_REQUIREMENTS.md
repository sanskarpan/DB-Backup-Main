# Desktop App - Build Requirements

## Prerequisites

To build and run the DB Backup Desktop application, you'll need the following tools installed:

### Required Tools

#### 1. **Rust & Cargo** (v1.70+)
The backend is built with Tauri, which requires Rust.

```bash
# Install Rust via rustup
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Verify installation
cargo --version
rustc --version
```

#### 2. **Node.js & npm** (v18+)
The frontend is built with React and Vite.

```bash
# Check if installed
node --version
npm --version

# Install via nvm (recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
nvm use 18
```

#### 3. **Platform-Specific Dependencies**

**macOS:**
```bash
# Install Xcode Command Line Tools
xcode-select --install
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install -y \
  libwebkit2gtk-4.0-dev \
  build-essential \
  curl \
  wget \
  libssl-dev \
  libgtk-3-dev \
  libayatana-appindicator3-dev \
  librsvg2-dev
```

**Windows:**
- Install [Microsoft Visual Studio C++ Build Tools](https://visualstudio.microsoft.com/visual-cpp-build-tools/)
- Install [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)

---

## Installation Steps

### 1. Clone the Repository
```bash
git clone <your-repo-url>
cd db-backup/desktop
```

### 2. Install Frontend Dependencies
```bash
npm install
```

This will install all the required packages including:
- React & React DOM
- Tauri API bindings
- UI libraries (Radix UI, Lucide React)
- i18n support (react-i18next)
- Testing libraries (Vitest, Testing Library)
- And more...

### 3. Install Rust Dependencies
```bash
cd src-tauri
cargo build
cd ..
```

This will download and compile all Rust dependencies defined in `Cargo.toml`:
- Tauri framework
- Serde (serialization)
- Tokio (async runtime)
- Reqwest (HTTP client)
- Auto-launch support
- PDF generation
- And more...

---

## Development

### Run Development Server
```bash
# Start Vite dev server + Tauri dev mode
npm run tauri:dev
```

This will:
1. Start Vite on `http://localhost:1420`
2. Compile the Rust backend
3. Launch the desktop application in development mode
4. Enable hot-reload for frontend changes

### Run Frontend Only
```bash
npm run dev
```

This starts only the Vite development server (useful for frontend-only development).

---

## Building

### Development Build
```bash
npm run build
```

This compiles TypeScript and bundles the frontend assets.

### Production Build
```bash
npm run tauri:build
```

This will:
1. Build the optimized frontend bundle
2. Compile the Rust backend in release mode
3. Create platform-specific installers in `src-tauri/target/release/bundle/`

**Output Locations:**
- **macOS:** `.app` bundle and `.dmg` installer
- **Windows:** `.exe` installer and `.msi` package
- **Linux:** `.deb`, `.AppImage`, and `.rpm` packages

---

## Testing

### Run All Tests
```bash
npm test
```

### Run Tests in Watch Mode
```bash
npm test
```

### Run Tests with UI
```bash
npm run test:ui
```

### Run Tests with Coverage
```bash
npm run test:coverage
```

**Current Test Coverage:**
- ✅ 10/10 tests passing
- ✅ Component rendering
- ✅ Navigation between tabs
- ✅ Theme toggling
- ✅ API integration
- ✅ Keyboard shortcuts

---

## Project Structure

```
desktop/
├── src/                         # Frontend source code
│   ├── __tests__/               # Test files
│   │   └── EnhancedApp.test.tsx # Main app tests
│   ├── i18n/                    # Internationalization
│   │   └── config.ts            # i18n configuration
│   ├── test/                    # Test setup
│   │   └── setup.ts             # Vitest configuration
│   ├── App.tsx                  # Original app (legacy)
│   ├── EnhancedApp.tsx          # Enhanced app with all features
│   ├── index.css                # Global styles
│   └── main.tsx                 # Entry point
├── src-tauri/                   # Backend Rust code
│   ├── src/
│   │   ├── main.rs              # Enhanced backend (650+ lines)
│   │   └── main.rs.backup       # Original backup
│   ├── Cargo.toml               # Rust dependencies
│   └── tauri.conf.json          # Tauri configuration
├── index.html                   # HTML entry point
├── package.json                 # Node dependencies
├── tsconfig.json                # TypeScript configuration
├── vite.config.ts               # Vite configuration
├── vitest.config.ts             # Vitest configuration
├── tailwind.config.js           # Tailwind CSS configuration
├── postcss.config.js            # PostCSS configuration
├── DESKTOP_ENHANCEMENTS.md      # Enhancement documentation
└── BUILD_REQUIREMENTS.md        # This file
```

---

## Environment Variables

Create a `.env` file in the `desktop/` directory (optional):

```env
VITE_API_URL=http://localhost:3000
VITE_API_KEY=your-api-key-here
```

---

## Troubleshooting

### Issue: `cargo: command not found`
**Solution:** Install Rust using rustup (see Prerequisites above)

### Issue: `tauri: command not found`
**Solution:**
```bash
npm install -g @tauri-apps/cli
# Or use npx
npx tauri dev
```

### Issue: Build fails on Linux with WebKit errors
**Solution:** Install WebKit dependencies:
```bash
sudo apt install libwebkit2gtk-4.0-dev
```

### Issue: Windows build fails
**Solution:** Ensure Visual Studio C++ Build Tools are installed

### Issue: Tests fail with "window is not defined"
**Solution:** The test setup mocks the Tauri API. Make sure `src/test/setup.ts` is being loaded.

### Issue: Dark mode not working
**Solution:** Check that `darkMode: ['class']` is set in `tailwind.config.js`

---

## Additional Resources

- [Tauri Documentation](https://tauri.app/)
- [Vite Documentation](https://vitejs.dev/)
- [React Documentation](https://react.dev/)
- [Vitest Documentation](https://vitest.dev/)
- [Tailwind CSS Documentation](https://tailwindcss.com/)

---

## CI/CD

For automated builds in CI/CD pipelines, install all dependencies:

```bash
# Install Node dependencies
npm install

# Build frontend
npm run build

# Build desktop app (requires Rust)
npm run tauri:build
```

**GitHub Actions Example:**
```yaml
- uses: actions/checkout@v3
- uses: actions/setup-node@v3
  with:
    node-version: '18'
- uses: actions-rs/toolchain@v1
  with:
    toolchain: stable
- run: npm install
- run: npm run tauri:build
```

---

## License

See the main project LICENSE file.
