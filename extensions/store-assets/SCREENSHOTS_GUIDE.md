# Screenshots Guide

Guide for creating professional screenshots for browser extension store listings.

## Required Screenshots

### 1. Popup Interface (Primary Screenshot)
**What to show:**
- Extension popup with dashboard view
- Statistics cards showing meaningful data
- Recent backups list with 2-3 entries
- Active alerts section
- Clean, professional appearance

**Annotations:**
- "Quick access to all features"
- "Real-time statistics"
- "Recent backup activity"

**Size:** 1280x800 pixels
**Format:** PNG
**Priority:** HIGHEST (this is the first thing users see)

---

### 2. Options/Settings Page
**What to show:**
- Settings page with all sections visible
- API configuration section
- Sync settings with slider
- Notification preferences
- Professional layout

**Annotations:**
- "Configure your API connection"
- "Customize sync interval"
- "Control notifications"

**Size:** 1280x800 pixels
**Format:** PNG
**Priority:** HIGH

---

### 3. Context Menu in Action
**What to show:**
- Right-click context menu displayed
- Show all menu items
- Browser page in background (database tool)
- Clear, readable menu

**Annotations:**
- "Right-click for quick actions"
- "Access features anywhere"

**Size:** 1280x800 pixels
**Format:** PNG
**Priority:** MEDIUM

---

### 4. Database Tool Detection
**What to show:**
- Content script detecting phpMyAdmin or Adminer
- Floating action button visible
- Page indicator at top
- Database tool interface in background

**Annotations:**
- "Auto-detects database tools"
- "One-click backup from any page"

**Size:** 1280x800 pixels
**Format:** PNG
**Priority:** HIGH

---

### 5. Notification Example
**What to show:**
- Browser notification displayed
- Backup completion or alert
- Extension icon visible
- Clean notification design

**Annotations:**
- "Instant notifications"
- "Stay informed of backup status"

**Size:** 1280x800 pixels
**Format:** PNG
**Priority:** MEDIUM

---

## Screenshot Creation Workflow

### Step 1: Prepare Environment

```bash
# 1. Set browser window to exactly 1280x800
# In Chrome DevTools:
# 1. Open DevTools (F12)
# 2. Toggle device toolbar (Ctrl+Shift+M)
# 3. Set dimensions to 1280x800
# 4. Set zoom to 100%

# 2. Clear test data
# 3. Load realistic sample data
# 4. Ensure good lighting/contrast
```

### Step 2: Take Screenshots

**macOS:**
```bash
# Full screen
Cmd + Shift + 3

# Selection
Cmd + Shift + 4

# Window with shadow
Cmd + Shift + 4, then Space, then click window
```

**Windows:**
```bash
# Full screen
PrtScn

# Active window
Alt + PrtScn

# Snipping Tool (Windows 10+)
Windows + Shift + S
```

**Linux:**
```bash
# Varies by distribution
# Usually: PrtScn or Shift + PrtScn
```

### Step 3: Edit Screenshots

Use any image editor (Photoshop, GIMP, Figma, etc.)

**Required edits:**
1. Crop to exactly 1280x800
2. Add annotations (arrows, boxes, text)
3. Sanitize any sensitive data
4. Ensure text is readable
5. Save as PNG (high quality)

### Step 4: Add Annotations

**Tools:**
- **Figma**: Best for professional annotations
- **Canva**: Easy online tool
- **Photoshop**: Professional, but complex
- **GIMP**: Free, powerful
- **Preview (macOS)**: Simple annotations

**Annotation Style:**
```
Font: System font (San Francisco, Segoe UI, Roboto)
Size: 24-32px for main text, 18-24px for descriptions
Color: White text with dark semi-transparent background
       OR Dark text on light background
Arrows: 3-4px stroke, rounded caps
Boxes: 2-3px border, rounded corners (8px radius)
```

### Step 5: Sanitize Data

**Remove or change:**
- Real database names → "production_db", "staging_db"
- Actual URLs → "localhost:8080", "api.example.com"
- Email addresses → "user@example.com"
- API keys → Show dots or placeholder
- User names → "Admin User", "John Doe"
- Sensitive timestamps → Use recent but generic times

**Keep realistic:**
- File sizes should be reasonable (1.2 MB, 5.4 GB)
- Dates should be recent but not real
- Status should show variety (success, warning, etc.)

---

## Screenshot Examples

### Example 1: Popup Interface

```
┌─────────────────────────────────────────────────┐
│  DB Backup Manager                    [Settings]│
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │ ← "Quick access to stats"
│  │   142    │  │    3     │  │  12.4 GB │     │
│  │ Backups  │  │  Failed  │  │   Size   │     │
│  └──────────┘  └──────────┘  └──────────┘     │
│                                                  │
│  Recent Backups:                                │ ← "Recent activity at a glance"
│  ┌────────────────────────────────────────────┐│
│  │ ✓ production_db     12:34 PM   2.1 MB     ││
│  │ ✓ staging_db        11:20 AM   1.8 MB     ││
│  │ ⚠ development_db    10:15 AM   3.4 MB     ││
│  └────────────────────────────────────────────┘│
│                                                  │
│  [Quick Backup] [View All] [Dashboard]         │
└─────────────────────────────────────────────────┘
```

### Example 2: Settings Page

```
┌─────────────────────────────────────────────────┐
│  Settings                              [Close]  │
│                                                  │
│  API Configuration                              │ ← "Configure your connection"
│  ┌────────────────────────────────────────────┐│
│  │ API URL: [http://localhost:8080        ]  ││
│  │ API Key: [••••••••••••••••••••••]  [👁]   ││
│  │                       [Test Connection]    ││
│  │ ✓ Connection successful!                   ││
│  └────────────────────────────────────────────┘│
│                                                  │
│  Synchronization                                │ ← "Customize sync settings"
│  ┌────────────────────────────────────────────┐│
│  │ Sync Interval: [────────●──] 5 minutes     ││
│  │ ☑ Enable automatic background sync        ││
│  └────────────────────────────────────────────┘│
│                                                  │
│  [Save Settings]  [Reset to Defaults]          │
└─────────────────────────────────────────────────┘
```

---

## Quality Checklist

Before submitting screenshots:

- [ ] Exactly 1280x800 pixels (or store requirement)
- [ ] PNG format, high quality
- [ ] No compression artifacts
- [ ] Readable text (not blurry)
- [ ] Annotations are clear and professional
- [ ] No sensitive data visible
- [ ] Consistent styling across all screenshots
- [ ] Proper lighting/contrast
- [ ] Extension looks good (no bugs visible)
- [ ] Realistic data shown
- [ ] File size < 5 MB (store limit)
- [ ] Preview on mobile (stores show thumbnails)

---

## Advanced Tips

### 1. Use Device Frames

Add browser chrome (address bar, tabs) for context:
- Use Figma browser mockups
- Screenshot actual browser window
- Use online frame generators

### 2. Show Multiple States

Create variations:
- Light mode vs dark mode
- Success vs error states
- Empty vs populated data
- Different database types

### 3. Add Motion

For video walkthroughs:
- Record screen at 1080p or higher
- Use OBS Studio (free)
- Add voiceover explaining features
- Keep under 2 minutes
- Upload to YouTube

### 4. A/B Test

If possible:
- Create 2-3 variations
- Test which converts better
- Stores allow screenshot updates
- Monitor install rates

### 5. Localization

For international markets:
- Create screenshots in different languages
- Adjust text annotations
- Keep UI in English (extension UI)
- Translate annotations only

---

## Tools & Resources

### Screenshot Tools
- **Shottr (macOS)**: Free, powerful screenshot tool
- **ShareX (Windows)**: Free, open source
- **Flameshot (Linux)**: Cross-platform, feature-rich
- **Greenshot (All)**: Simple, effective

### Editing Tools
- **Figma**: Free, collaborative, web-based
- **Canva**: Easy templates, online
- **GIMP**: Free Photoshop alternative
- **Photoshop**: Professional standard
- **Sketch (macOS)**: Popular among designers

### Annotation Tools
- **Skitch**: Simple annotations (macOS, iOS)
- **Monosnap**: Screenshots + annotations
- **CloudApp**: Screenshot + share
- **Annotate.app**: Purpose-built annotations

### Browser Extensions
- **Awesome Screenshot**: Capture + annotate
- **Nimbus Screenshot**: Full page captures
- **Lightshot**: Quick screenshots

---

## File Naming Convention

Use descriptive names:

```
db-backup-manager-chrome-01-popup.png
db-backup-manager-chrome-02-settings.png
db-backup-manager-chrome-03-context-menu.png
db-backup-manager-chrome-04-detection.png
db-backup-manager-chrome-05-notification.png
```

Benefits:
- Easy to identify
- Organized by browser
- Numbered for order
- Descriptive content

---

## Automated Screenshot Generation

For consistent screenshots across updates:

```javascript
// Puppeteer script example
const puppeteer = require('puppeteer');

async function captureScreenshots() {
  const browser = await puppeteer.launch({
    args: ['--load-extension=./chrome'],
    headless: false
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 800 });

  // Navigate and capture
  await page.goto('chrome-extension://[ID]/popup/popup.html');
  await page.screenshot({
    path: 'screenshot-01-popup.png',
    fullPage: false
  });

  await browser.close();
}
```

---

## Common Mistakes to Avoid

1. **Wrong size**: Always check store requirements
2. **Low quality**: Use PNG, not compressed JPEG
3. **Too much text**: Keep annotations minimal
4. **Fake data**: Use realistic but safe data
5. **Inconsistent style**: Maintain visual consistency
6. **Cluttered**: Show one feature per screenshot
7. **Dark backgrounds**: May not show well in stores
8. **Small text**: Ensure readability on mobile
9. **Browser chrome**: Include or exclude consistently
10. **Outdated**: Update screenshots with each release

---

## Store-Specific Requirements

### Chrome Web Store
- Size: 1280x800 or 640x400
- Format: PNG or JPEG
- Max file size: 5 MB
- Quantity: 1-5 screenshots
- Order matters (first is most important)

### Firefox Add-ons
- Size: Flexible (recommend 1280x800)
- Format: PNG, JPEG, or GIF
- Max file size: 10 MB
- Quantity: 1 or more
- Can show animated GIFs

### Edge Add-ons
- Size: 1280x800 or 640x400
- Format: PNG preferred
- Max file size: 5 MB
- Quantity: 1-10 screenshots
- Similar to Chrome

### Safari App Store
- Size: 1280x800 minimum
- Format: PNG or JPEG
- Max file size: 8 MB
- Quantity: 1-10 screenshots
- Must show macOS app (not just extension)

---

## Getting Help

- Review competitor extensions for inspiration
- Check store guidelines before creating
- Test on different devices/screens
- Get feedback from colleagues
- Update regularly with new features
