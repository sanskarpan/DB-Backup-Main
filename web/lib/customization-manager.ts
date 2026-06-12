/**
 * Customization & Personalization - Core Managers
 * Comprehensive system for themes, layouts, widgets, and personalization
 */

// ==================== Types & Interfaces ====================

export interface Theme {
  id: string
  name: string
  description?: string
  colors: ColorScheme
  fonts?: FontScheme
  spacing?: SpacingScheme
  borderRadius?: string
  shadows?: ShadowScheme
  author?: string
  createdAt?: Date
  updatedAt?: Date
  isCustom?: boolean
  preview?: string
}

export interface ColorScheme {
  primary: string
  secondary: string
  accent: string
  background: string
  surface: string
  text: string
  textSecondary: string
  border: string
  error: string
  warning: string
  success: string
  info: string
  [key: string]: string
}

export interface FontScheme {
  family: string
  headingFamily?: string
  size: 'xs' | 'sm' | 'base' | 'lg' | 'xl'
  weight: 'light' | 'normal' | 'medium' | 'semibold' | 'bold'
  lineHeight?: 'tight' | 'normal' | 'relaxed'
}

export interface SpacingScheme {
  unit: number // base unit in pixels
  scale: number // multiplier
}

export interface ShadowScheme {
  sm: string
  md: string
  lg: string
  xl: string
}

export interface Widget {
  id: string
  type: string
  title: string
  component: string
  props?: Record<string, any>
  minWidth?: number
  minHeight?: number
  defaultWidth?: number
  defaultHeight?: number
  resizable?: boolean
  removable?: boolean
  configurable?: boolean
}

export interface LayoutWidget extends Widget {
  x: number
  y: number
  width: number
  height: number
  locked?: boolean
}

export interface DashboardLayout {
  id: string
  name: string
  description?: string
  widgets: LayoutWidget[]
  columns: number
  rowHeight: number
  isDefault?: boolean
  createdAt?: Date
  updatedAt?: Date
}

export interface FavoriteAction {
  id: string
  label: string
  icon?: string
  action: string
  params?: Record<string, any>
  order: number
  category?: string
  addedAt: Date
}

export interface RecentItem {
  id: string
  type: 'backup' | 'restore' | 'schedule' | 'database' | 'report'
  title: string
  subtitle?: string
  url: string
  accessedAt: Date
  metadata?: Record<string, any>
}

export interface UserPreferences {
  theme: string
  layout: string
  density: 'compact' | 'comfortable' | 'spacious'
  fontSize: FontScheme['size']
  fontFamily?: string
  colorScheme?: Partial<ColorScheme>
  sidebarCollapsed?: boolean
  showWelcome?: boolean
  language?: string
  timezone?: string
  dateFormat?: string
  timeFormat?: '12h' | '24h'
}

export interface ProfilePreset {
  id: string
  name: string
  description: string
  icon?: string
  theme: string
  layout: string
  density: 'compact' | 'comfortable' | 'spacious'
  widgets: string[]
  favorites: string[]
  permissions?: string[]
  isDefault?: boolean
}

export interface Recommendation {
  id: string
  type: 'action' | 'feature' | 'content' | 'workflow'
  title: string
  description: string
  reason: string
  confidence: number // 0-1
  category?: string
  actionUrl?: string
  dismissible?: boolean
  priority: 'low' | 'medium' | 'high'
}

// ==================== Theme Manager ====================

export class ThemeManager {
  private themes: Map<string, Theme> = new Map()
  private activeTheme: string = 'default'
  private listeners: Set<(theme: Theme) => void> = new Set()

  constructor() {
    this.initializeDefaultThemes()
    this.loadActiveTheme()
  }

  private initializeDefaultThemes(): void {
    // Default Light Theme
    this.registerTheme({
      id: 'default',
      name: 'Default Light',
      description: 'Clean and modern light theme',
      colors: {
        primary: '#3b82f6',
        secondary: '#8b5cf6',
        accent: '#06b6d4',
        background: '#ffffff',
        surface: '#f9fafb',
        text: '#111827',
        textSecondary: '#6b7280',
        border: '#e5e7eb',
        error: '#ef4444',
        warning: '#f59e0b',
        success: '#10b981',
        info: '#3b82f6'
      },
      fonts: {
        family: 'Inter, system-ui, sans-serif',
        headingFamily: 'Inter, system-ui, sans-serif',
        size: 'base',
        weight: 'normal',
        lineHeight: 'normal'
      },
      borderRadius: '0.5rem',
      shadows: {
        sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
        md: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
        lg: '0 10px 15px -3px rgb(0 0 0 / 0.1)',
        xl: '0 20px 25px -5px rgb(0 0 0 / 0.1)'
      }
    })

    // Dark Theme
    this.registerTheme({
      id: 'dark',
      name: 'Dark',
      description: 'Easy on the eyes dark theme',
      colors: {
        primary: '#60a5fa',
        secondary: '#a78bfa',
        accent: '#22d3ee',
        background: '#111827',
        surface: '#1f2937',
        text: '#f9fafb',
        textSecondary: '#9ca3af',
        border: '#374151',
        error: '#f87171',
        warning: '#fbbf24',
        success: '#34d399',
        info: '#60a5fa'
      },
      fonts: {
        family: 'Inter, system-ui, sans-serif',
        size: 'base',
        weight: 'normal'
      },
      borderRadius: '0.5rem'
    })

    // High Contrast
    this.registerTheme({
      id: 'high-contrast',
      name: 'High Contrast',
      description: 'Maximum contrast for accessibility',
      colors: {
        primary: '#000000',
        secondary: '#1a1a1a',
        accent: '#0066cc',
        background: '#ffffff',
        surface: '#f5f5f5',
        text: '#000000',
        textSecondary: '#333333',
        border: '#000000',
        error: '#cc0000',
        warning: '#ff6600',
        success: '#008800',
        info: '#0066cc'
      },
      fonts: {
        family: 'Arial, sans-serif',
        size: 'lg',
        weight: 'semibold'
      },
      borderRadius: '0.25rem'
    })
  }

  registerTheme(theme: Theme): void {
    this.themes.set(theme.id, theme)
  }

  getTheme(themeId: string): Theme | undefined {
    return this.themes.get(themeId)
  }

  getAllThemes(): Theme[] {
    return Array.from(this.themes.values())
  }

  setActiveTheme(themeId: string): boolean {
    const theme = this.themes.get(themeId)
    if (!theme) return false

    this.activeTheme = themeId
    this.applyTheme(theme)
    this.saveActiveTheme()
    this.notifyListeners(theme)
    return true
  }

  getActiveTheme(): Theme {
    return this.themes.get(this.activeTheme) || this.themes.get('default')!
  }

  private applyTheme(theme: Theme): void {
    if (typeof document === 'undefined') return

    const root = document.documentElement

    // Apply colors
    Object.entries(theme.colors).forEach(([key, value]) => {
      root.style.setProperty(`--color-${key}`, value)
    })

    // Apply fonts
    if (theme.fonts) {
      root.style.setProperty('--font-family', theme.fonts.family)
      if (theme.fonts.headingFamily) {
        root.style.setProperty('--font-family-heading', theme.fonts.headingFamily)
      }
    }

    // Apply border radius
    if (theme.borderRadius) {
      root.style.setProperty('--border-radius', theme.borderRadius)
    }
  }

  subscribe(listener: (theme: Theme) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(theme: Theme): void {
    this.listeners.forEach(listener => listener(theme))
  }

  private loadActiveTheme(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('active-theme')
    if (stored) {
      this.setActiveTheme(stored)
    }
  }

  private saveActiveTheme(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('active-theme', this.activeTheme)
  }

  createCustomTheme(base: Theme, overrides: Partial<Theme>): Theme {
    return {
      ...base,
      ...overrides,
      id: overrides.id || `custom-${Date.now()}`,
      isCustom: true,
      createdAt: new Date()
    }
  }

  exportTheme(themeId: string): string {
    const theme = this.themes.get(themeId)
    if (!theme) throw new Error('Theme not found')
    return JSON.stringify(theme, null, 2)
  }

  importTheme(themeJson: string): Theme {
    const theme = JSON.parse(themeJson) as Theme
    this.registerTheme(theme)
    return theme
  }
}

// ==================== Layout Manager ====================

export class LayoutManager {
  private layouts: Map<string, DashboardLayout> = new Map()
  private activeLayout: string = 'default'
  private listeners: Set<(layout: DashboardLayout) => void> = new Set()

  constructor() {
    this.initializeDefaultLayouts()
    this.loadActiveLayout()
  }

  private initializeDefaultLayouts(): void {
    this.registerLayout({
      id: 'default',
      name: 'Default Layout',
      description: 'Standard dashboard layout',
      columns: 12,
      rowHeight: 60,
      widgets: [
        { id: 'stats', type: 'stats', title: 'Statistics', component: 'StatsWidget', x: 0, y: 0, width: 12, height: 2 },
        { id: 'recent', type: 'recent', title: 'Recent Backups', component: 'RecentWidget', x: 0, y: 2, width: 8, height: 4 },
        { id: 'activity', type: 'activity', title: 'Activity', component: 'ActivityWidget', x: 8, y: 2, width: 4, height: 4 }
      ],
      isDefault: true
    })
  }

  registerLayout(layout: DashboardLayout): void {
    this.layouts.set(layout.id, layout)
  }

  getLayout(layoutId: string): DashboardLayout | undefined {
    return this.layouts.get(layoutId)
  }

  getAllLayouts(): DashboardLayout[] {
    return Array.from(this.layouts.values())
  }

  setActiveLayout(layoutId: string): boolean {
    const layout = this.layouts.get(layoutId)
    if (!layout) return false

    this.activeLayout = layoutId
    this.saveActiveLayout()
    this.notifyListeners(layout)
    return true
  }

  getActiveLayout(): DashboardLayout {
    return this.layouts.get(this.activeLayout) || this.layouts.get('default')!
  }

  updateWidget(layoutId: string, widgetId: string, updates: Partial<LayoutWidget>): void {
    const layout = this.layouts.get(layoutId)
    if (!layout) return

    const widgetIndex = layout.widgets.findIndex(w => w.id === widgetId)
    if (widgetIndex === -1) return

    layout.widgets[widgetIndex] = { ...layout.widgets[widgetIndex], ...updates }
    layout.updatedAt = new Date()
    this.saveLayout(layout)
    this.notifyListeners(layout)
  }

  addWidget(layoutId: string, widget: LayoutWidget): void {
    const layout = this.layouts.get(layoutId)
    if (!layout) return

    layout.widgets.push(widget)
    layout.updatedAt = new Date()
    this.saveLayout(layout)
    this.notifyListeners(layout)
  }

  removeWidget(layoutId: string, widgetId: string): void {
    const layout = this.layouts.get(layoutId)
    if (!layout) return

    layout.widgets = layout.widgets.filter(w => w.id !== widgetId)
    layout.updatedAt = new Date()
    this.saveLayout(layout)
    this.notifyListeners(layout)
  }

  subscribe(listener: (layout: DashboardLayout) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(layout: DashboardLayout): void {
    this.listeners.forEach(listener => listener(layout))
  }

  private saveLayout(layout: DashboardLayout): void {
    if (typeof window === 'undefined') return

    const layouts = this.getAllLayouts()
    localStorage.setItem('dashboard-layouts', JSON.stringify(layouts))
  }

  private loadActiveLayout(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('active-layout')
    if (stored) {
      this.activeLayout = stored
    }
  }

  private saveActiveLayout(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('active-layout', this.activeLayout)
  }

  exportLayout(layoutId: string): string {
    const layout = this.layouts.get(layoutId)
    if (!layout) throw new Error('Layout not found')
    return JSON.stringify(layout, null, 2)
  }

  importLayout(layoutJson: string): DashboardLayout {
    const layout = JSON.parse(layoutJson) as DashboardLayout
    this.registerLayout(layout)
    return layout
  }
}

// ==================== Favorites Manager ====================

export class FavoritesManager {
  private favorites: FavoriteAction[] = []
  private listeners: Set<() => void> = new Set()

  constructor() {
    this.loadFavorites()
  }

  add(action: Omit<FavoriteAction, 'id' | 'addedAt' | 'order'>): FavoriteAction {
    const favorite: FavoriteAction = {
      ...action,
      id: `fav-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      order: this.favorites.length,
      addedAt: new Date()
    }

    this.favorites.push(favorite)
    this.saveFavorites()
    this.notifyListeners()
    return favorite
  }

  remove(id: string): void {
    this.favorites = this.favorites.filter(f => f.id !== id)
    this.reorderFavorites()
    this.saveFavorites()
    this.notifyListeners()
  }

  reorder(id: string, newOrder: number): void {
    const favorite = this.favorites.find(f => f.id === id)
    if (!favorite) return

    const oldOrder = favorite.order
    this.favorites.forEach(f => {
      if (f.order > oldOrder && f.order <= newOrder) {
        f.order--
      } else if (f.order < oldOrder && f.order >= newOrder) {
        f.order++
      }
    })

    favorite.order = newOrder
    this.saveFavorites()
    this.notifyListeners()
  }

  getAll(): FavoriteAction[] {
    return [...this.favorites].sort((a, b) => a.order - b.order)
  }

  getByCategory(category: string): FavoriteAction[] {
    return this.favorites.filter(f => f.category === category).sort((a, b) => a.order - b.order)
  }

  isFavorite(action: string): boolean {
    return this.favorites.some(f => f.action === action)
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => listener())
  }

  private reorderFavorites(): void {
    this.favorites.forEach((f, index) => {
      f.order = index
    })
  }

  private loadFavorites(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('favorite-actions')
    if (stored) {
      try {
        this.favorites = JSON.parse(stored).map((f: any) => ({
          ...f,
          addedAt: new Date(f.addedAt)
        }))
      } catch (e) {
        console.error('Failed to load favorites:', e)
      }
    }
  }

  private saveFavorites(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('favorite-actions', JSON.stringify(this.favorites))
  }
}

// ==================== Recent Items Manager ====================

export class RecentItemsManager {
  private items: RecentItem[] = []
  private maxItems: number = 20
  private listeners: Set<() => void> = new Set()

  constructor(maxItems: number = 20) {
    this.maxItems = maxItems
    this.loadItems()
  }

  add(item: Omit<RecentItem, 'id' | 'accessedAt'>): void {
    // Remove existing item if present
    this.items = this.items.filter(i => !(i.type === item.type && i.url === item.url))

    // Add new item at the beginning
    const recentItem: RecentItem = {
      ...item,
      id: `recent-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      accessedAt: new Date()
    }

    this.items.unshift(recentItem)

    // Trim to max items
    if (this.items.length > this.maxItems) {
      this.items = this.items.slice(0, this.maxItems)
    }

    this.saveItems()
    this.notifyListeners()
  }

  getAll(): RecentItem[] {
    return [...this.items]
  }

  getByType(type: RecentItem['type']): RecentItem[] {
    return this.items.filter(i => i.type === type)
  }

  clear(): void {
    this.items = []
    this.saveItems()
    this.notifyListeners()
  }

  clearOlderThan(days: number): void {
    const cutoff = new Date()
    cutoff.setDate(cutoff.getDate() - days)

    this.items = this.items.filter(i => i.accessedAt > cutoff)
    this.saveItems()
    this.notifyListeners()
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => listener())
  }

  private loadItems(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('recent-items')
    if (stored) {
      try {
        this.items = JSON.parse(stored).map((i: any) => ({
          ...i,
          accessedAt: new Date(i.accessedAt)
        }))
      } catch (e) {
        console.error('Failed to load recent items:', e)
      }
    }
  }

  private saveItems(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('recent-items', JSON.stringify(this.items))
  }
}

// ==================== Recommendations Manager ====================

export class RecommendationsManager {
  private recommendations: Recommendation[] = []
  private dismissed: Set<string> = new Set()
  private listeners: Set<() => void> = new Set()

  constructor() {
    this.loadDismissed()
    this.generateRecommendations()
  }

  private generateRecommendations(): void {
    // AI-based recommendations would be generated here
    // For now, using rule-based recommendations

    this.recommendations = [
      {
        id: 'rec-1',
        type: 'action',
        title: 'Create Your First Backup',
        description: 'Get started by creating a backup of your database',
        reason: 'You haven\'t created any backups yet',
        confidence: 0.9,
        category: 'getting-started',
        actionUrl: '/backups/new',
        dismissible: true,
        priority: 'high'
      },
      {
        id: 'rec-2',
        type: 'feature',
        title: 'Enable Compression',
        description: 'Reduce backup size by up to 80%',
        reason: 'Based on your database size',
        confidence: 0.85,
        category: 'optimization',
        dismissible: true,
        priority: 'medium'
      }
    ]
  }

  getRecommendations(): Recommendation[] {
    return this.recommendations.filter(r => !this.dismissed.has(r.id))
  }

  getByPriority(priority: Recommendation['priority']): Recommendation[] {
    return this.getRecommendations().filter(r => r.priority === priority)
  }

  dismiss(id: string): void {
    this.dismissed.add(id)
    this.saveDismissed()
    this.notifyListeners()
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => listener())
  }

  private loadDismissed(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('dismissed-recommendations')
    if (stored) {
      try {
        this.dismissed = new Set(JSON.parse(stored))
      } catch (e) {
        console.error('Failed to load dismissed recommendations:', e)
      }
    }
  }

  private saveDismissed(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('dismissed-recommendations', JSON.stringify(
      Array.from(this.dismissed)
    ))
  }
}

// ==================== Preferences Manager ====================

export class PreferencesManager {
  private preferences: UserPreferences
  private listeners: Set<(preferences: UserPreferences) => void> = new Set()

  constructor() {
    this.preferences = this.getDefaultPreferences()
    this.loadPreferences()
  }

  private getDefaultPreferences(): UserPreferences {
    return {
      theme: 'default',
      layout: 'default',
      density: 'comfortable',
      fontSize: 'base',
      sidebarCollapsed: false,
      showWelcome: true,
      dateFormat: 'MM/DD/YYYY',
      timeFormat: '12h'
    }
  }

  getPreferences(): UserPreferences {
    return { ...this.preferences }
  }

  updatePreferences(updates: Partial<UserPreferences>): void {
    this.preferences = { ...this.preferences, ...updates }
    this.savePreferences()
    this.notifyListeners(this.preferences)
  }

  resetToDefaults(): void {
    this.preferences = this.getDefaultPreferences()
    this.savePreferences()
    this.notifyListeners(this.preferences)
  }

  subscribe(listener: (preferences: UserPreferences) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(preferences: UserPreferences): void {
    this.listeners.forEach(listener => listener(preferences))
  }

  private loadPreferences(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('user-preferences')
    if (stored) {
      try {
        this.preferences = { ...this.preferences, ...JSON.parse(stored) }
      } catch (e) {
        console.error('Failed to load preferences:', e)
      }
    }
  }

  private savePreferences(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('user-preferences', JSON.stringify(this.preferences))
  }

  exportPreferences(): string {
    return JSON.stringify(this.preferences, null, 2)
  }

  importPreferences(preferencesJson: string): void {
    try {
      const imported = JSON.parse(preferencesJson) as UserPreferences
      this.preferences = { ...this.getDefaultPreferences(), ...imported }
      this.savePreferences()
      this.notifyListeners(this.preferences)
    } catch (e) {
      throw new Error('Invalid preferences format')
    }
  }
}

// ==================== Profile Presets Manager ====================

export class ProfilePresetsManager {
  private presets: Map<string, ProfilePreset> = new Map()
  private activePreset: string = 'operator'

  constructor() {
    this.initializeDefaultPresets()
    this.loadActivePreset()
  }

  private initializeDefaultPresets(): void {
    this.registerPreset({
      id: 'operator',
      name: 'Backup Operator',
      description: 'Focused on day-to-day backup operations',
      icon: 'database',
      theme: 'default',
      layout: 'default',
      density: 'comfortable',
      widgets: ['stats', 'recent', 'schedule'],
      favorites: ['create-backup', 'view-backups', 'restore'],
      isDefault: true
    })

    this.registerPreset({
      id: 'admin',
      name: 'Administrator',
      description: 'Full system access and monitoring',
      icon: 'shield',
      theme: 'dark',
      layout: 'admin',
      density: 'compact',
      widgets: ['stats', 'recent', 'activity', 'health', 'users'],
      favorites: ['users', 'settings', 'audit-log', 'reports'],
      permissions: ['admin']
    })

    this.registerPreset({
      id: 'viewer',
      name: 'Viewer',
      description: 'Read-only access for monitoring',
      icon: 'eye',
      theme: 'default',
      layout: 'simple',
      density: 'spacious',
      widgets: ['stats', 'recent'],
      favorites: ['view-backups', 'reports'],
      permissions: ['read']
    })
  }

  registerPreset(preset: ProfilePreset): void {
    this.presets.set(preset.id, preset)
  }

  getPreset(presetId: string): ProfilePreset | undefined {
    return this.presets.get(presetId)
  }

  getAllPresets(): ProfilePreset[] {
    return Array.from(this.presets.values())
  }

  setActivePreset(presetId: string): boolean {
    const preset = this.presets.get(presetId)
    if (!preset) return false

    this.activePreset = presetId
    this.applyPreset(preset)
    this.saveActivePreset()
    return true
  }

  getActivePreset(): ProfilePreset {
    return this.presets.get(this.activePreset) || this.presets.get('operator')!
  }

  private applyPreset(preset: ProfilePreset): void {
    // Apply theme
    themeManager.setActiveTheme(preset.theme)

    // Apply layout
    layoutManager.setActiveLayout(preset.layout)

    // Apply other preferences
    preferencesManager.updatePreferences({
      density: preset.density,
      theme: preset.theme,
      layout: preset.layout
    })
  }

  private loadActivePreset(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('active-preset')
    if (stored) {
      this.activePreset = stored
    }
  }

  private saveActivePreset(): void {
    if (typeof window === 'undefined') return

    localStorage.setItem('active-preset', this.activePreset)
  }
}

// ==================== Global Instances ====================

export const themeManager = new ThemeManager()
export const layoutManager = new LayoutManager()
export const favoritesManager = new FavoritesManager()
export const recentItemsManager = new RecentItemsManager()
export const recommendationsManager = new RecommendationsManager()
export const preferencesManager = new PreferencesManager()
export const profilePresetsManager = new ProfilePresetsManager()

// ==================== Utility Functions ====================

export function exportAllCustomizations(): string {
  return JSON.stringify({
    theme: themeManager.getActiveTheme(),
    layout: layoutManager.getActiveLayout(),
    favorites: favoritesManager.getAll(),
    preferences: preferencesManager.getPreferences(),
    preset: profilePresetsManager.getActivePreset().id,
    exportedAt: new Date().toISOString()
  }, null, 2)
}

export function importAllCustomizations(json: string): void {
  try {
    const data = JSON.parse(json)

    if (data.theme) themeManager.registerTheme(data.theme)
    if (data.layout) layoutManager.registerLayout(data.layout)
    if (data.preferences) preferencesManager.importPreferences(JSON.stringify(data.preferences))
    if (data.preset) profilePresetsManager.setActivePreset(data.preset)
  } catch (e) {
    throw new Error('Invalid customization data')
  }
}

export function resetAllCustomizations(): void {
  if (typeof window === 'undefined') return

  localStorage.removeItem('active-theme')
  localStorage.removeItem('active-layout')
  localStorage.removeItem('favorite-actions')
  localStorage.removeItem('recent-items')
  localStorage.removeItem('user-preferences')
  localStorage.removeItem('active-preset')
  localStorage.removeItem('dismissed-recommendations')

  window.location.reload()
}
