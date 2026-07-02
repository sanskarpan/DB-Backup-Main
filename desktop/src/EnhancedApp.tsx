import { useState, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/tauri'
import { sendNotification } from '@tauri-apps/api/notification'
import { appWindow } from '@tauri-apps/api/window'
import { useTranslation } from 'react-i18next'
import { Command } from 'cmdk'
import { jsPDF } from 'jspdf'
import autoTable from 'jspdf-autotable'
import * as XLSX from 'xlsx'
import {
  Database,
  Settings,
  Activity,
  Minimize,
  X,
  Search,
  Sun,
  Moon,
  Download,
  Eye,
  Power
} from 'lucide-react'

import './i18n/config'

// Types
interface Backup {
  id: string
  database_id: string
  database_name: string
  status: string
  created_at: string
  size?: number
  duration?: number
}

interface BackupPreview {
  id: string
  database_name: string
  tables: TableInfo[]
  total_size: number
  schema_sql: string
}

interface TableInfo {
  name: string
  row_count: number
  size_bytes: number
}

interface SearchResult {
  id: string
  title: string
  description: string
  category: string
  score: number
}

interface Config {
  api_url: string
  api_key: string
  auto_backup_enabled: boolean
  backup_interval_hours: number
  notifications_enabled: boolean
  theme: string
  language: string
  auto_launch: boolean
  keyboard_shortcuts: KeyboardShortcuts
}

interface KeyboardShortcuts {
  quick_search: string
  new_backup: string
  refresh: string
  toggle_theme: string
  open_settings: string
}

function EnhancedApp() {
  const { t, i18n } = useTranslation()
  const [activeTab, setActiveTab] = useState('backups')
  const [backups, setBackups] = useState<Backup[]>([])
  const [config, setConfig] = useState<Config | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [theme, setTheme] = useState<string>('light')

  // Modal states
  const [showQuickSearch, setShowQuickSearch] = useState(false)
  const [showBackupPreview, setShowBackupPreview] = useState(false)
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null)
  const [previewData, setPreviewData] = useState<BackupPreview | null>(null)

  useEffect(() => {
    loadBackups()
    loadConfig()
    loadTheme()
    setupKeyboardShortcuts()
  }, [])

  const loadBackups = async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await invoke<Backup[]>('get_backups', { limit: 50 })
      setBackups(result)
    } catch (err) {
      setError(err as string)
    } finally {
      setLoading(false)
    }
  }

  const loadConfig = async () => {
    try {
      const result = await invoke<Config>('get_config')
      setConfig(result)
      if (result.language !== i18n.language) {
        i18n.changeLanguage(result.language)
      }
    } catch (err) {
      console.error('Failed to load config:', err)
    }
  }

  const loadTheme = async () => {
    try {
      const currentTheme = await invoke<string>('get_theme')
      setTheme(currentTheme)
      applyTheme(currentTheme)
    } catch (err) {
      console.error('Failed to load theme:', err)
    }
  }

  const applyTheme = (themeName: string) => {
    if (themeName === 'dark') {
      document.documentElement.classList.add('dark')
    } else if (themeName === 'light') {
      document.documentElement.classList.remove('dark')
    } else {
      // Auto mode - detect system preference
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      if (isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    }
  }

  const setupKeyboardShortcuts = () => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K for quick search
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setShowQuickSearch(true)
      }
      // Cmd/Ctrl + N for new backup
      if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault()
        handleCreateBackup()
      }
      // Cmd/Ctrl + R for refresh
      if ((e.metaKey || e.ctrlKey) && e.key === 'r') {
        e.preventDefault()
        loadBackups()
      }
      // Cmd/Ctrl + Shift + D for toggle theme
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'D') {
        e.preventDefault()
        handleToggleTheme()
      }
      // Escape to close modals
      if (e.key === 'Escape') {
        setShowQuickSearch(false)
        setShowBackupPreview(false)
        setShowExportDialog(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }

  const handleCreateBackup = async () => {
    setLoading(true)
    setError(null)
    try {
      // Tauri converts JS camelCase arg keys to the Rust snake_case params,
      // so this must be `databaseId` to bind to the `database_id` parameter.
      await invoke('create_backup', {
        databaseId: 'default-db',
        options: {}
      })
      await sendNotification({
        title: t('success'),
        body: t('newBackup') + ' started'
      })
      loadBackups()
    } catch (err) {
      setError(err as string)
    } finally {
      setLoading(false)
    }
  }

  const handleToggleTheme = async () => {
    try {
      const newTheme = await invoke<string>('toggle_theme')
      setTheme(newTheme)
      applyTheme(newTheme)
    } catch (err) {
      console.error('Failed to toggle theme:', err)
    }
  }

  const handleChangeLanguage = async (lang: string) => {
    try {
      await invoke('set_language', { language: lang })
      i18n.changeLanguage(lang)
      if (config) {
        setConfig({ ...config, language: lang })
      }
    } catch (err) {
      console.error('Failed to change language:', err)
    }
  }

  const handlePreviewBackup = (backup: Backup) => {
    // The backend does not expose a per-backup "preview" endpoint, so build the
    // preview from the backup metadata we already loaded. No fabricated data:
    // table-level detail is unavailable, so `tables` is left empty.
    const preview: BackupPreview = {
      id: backup.id,
      database_name: backup.database_name,
      tables: [],
      total_size: backup.size ?? 0,
      schema_sql: ''
    }
    setPreviewData(preview)
    setSelectedBackup(backup)
    setShowBackupPreview(true)
  }

  const handleExport = async (format: string) => {
    try {
      // Export using the data already fetched from the backend (no server-side
      // export endpoint exists). Files are generated locally.
      const filename = exportBackupsToFile(format, backups)
      await sendNotification({
        title: t('exportSuccess'),
        body: `Exported to ${filename}`
      })
      setShowExportDialog(false)
    } catch (err) {
      setError(err as string)
    }
  }

  const handleSaveConfig = async () => {
    if (!config) return
    setLoading(true)
    try {
      await invoke('update_config', { config })
      await sendNotification({
        title: t('success'),
        body: t('settings') + ' saved'
      })
    } catch (err) {
      setError(err as string)
    } finally {
      setLoading(false)
    }
  }

  const handleMinimize = async () => {
    await appWindow.minimize()
  }

  const handleClose = async () => {
    await appWindow.hide()
  }

  return (
    <div className="flex flex-col h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
      {/* Custom Title Bar */}
      <div
        data-tauri-drag-region
        className="flex items-center justify-between h-8 bg-gray-800 dark:bg-gray-950 text-white px-4"
      >
        <div className="flex items-center gap-2 text-sm font-medium">
          <Database className="w-4 h-4" />
          {t('appName')}
        </div>
        <div className="flex gap-2">
          {/* Theme Toggle */}
          <button
            onClick={handleToggleTheme}
            className="hover:bg-gray-700 p-1 rounded"
            title="Toggle Theme (Cmd+Shift+D)"
          >
            {theme === 'dark' ? <Moon className="w-4 h-4" /> : <Sun className="w-4 h-4" />}
          </button>

          {/* Quick Search */}
          <button
            onClick={() => setShowQuickSearch(true)}
            className="hover:bg-gray-700 p-1 rounded"
            title="Quick Search (Cmd+K)"
          >
            <Search className="w-4 h-4" />
          </button>

          {/* Language Selector */}
          <select
            value={i18n.language}
            onChange={(e) => handleChangeLanguage(e.target.value)}
            className="bg-gray-700 text-white text-xs px-2 py-1 rounded border-none outline-none"
          >
            <option value="en">EN</option>
            <option value="es">ES</option>
            <option value="fr">FR</option>
            <option value="de">DE</option>
          </select>

          <button onClick={handleMinimize} className="hover:bg-gray-700 p-1 rounded">
            <Minimize className="w-4 h-4" />
          </button>
          <button onClick={handleClose} className="hover:bg-red-600 p-1 rounded">
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <TabButton
          active={activeTab === 'backups'}
          onClick={() => setActiveTab('backups')}
          icon={<Database className="w-5 h-5" />}
          label={t('backups')}
        />
        <TabButton
          active={activeTab === 'activity'}
          onClick={() => setActiveTab('activity')}
          icon={<Activity className="w-5 h-5" />}
          label={t('activity')}
        />
        <TabButton
          active={activeTab === 'settings'}
          onClick={() => setActiveTab('settings')}
          icon={<Settings className="w-5 h-5" />}
          label={t('settings')}
        />
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-auto p-6">
        {error && (
          <div className="mb-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-800 dark:text-red-200 px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        {activeTab === 'backups' && (
          <BackupsTab
            backups={backups}
            loading={loading}
            onCreateBackup={handleCreateBackup}
            onRefresh={loadBackups}
            onPreview={handlePreviewBackup}
            onExport={() => setShowExportDialog(true)}
          />
        )}

        {activeTab === 'activity' && <ActivityTab backups={backups} />}

        {activeTab === 'settings' && (
          <SettingsTab
            config={config}
            onConfigChange={setConfig}
            onSave={handleSaveConfig}
            loading={loading}
          />
        )}
      </div>

      {/* Quick Search Modal */}
      {showQuickSearch && (
        <QuickSearchDialog
          onClose={() => setShowQuickSearch(false)}
          onSelect={(result: SearchResult) => {
            console.log('Selected:', result)
            setShowQuickSearch(false)
          }}
        />
      )}

      {/* Backup Preview Modal */}
      {showBackupPreview && previewData && (
        <BackupPreviewModal
          preview={previewData}
          onClose={() => setShowBackupPreview(false)}
          onRestore={() => console.log('Restore:', selectedBackup)}
        />
      )}

      {/* Export Dialog */}
      {showExportDialog && (
        <ExportDialog
          onClose={() => setShowExportDialog(false)}
          onExport={handleExport}
        />
      )}
    </div>
  )
}

// Component implementations below...
// (Due to length, I'll create these as separate files)

interface TabButtonProps {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}

function TabButton({ active, onClick, icon, label }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-6 py-3 font-medium transition-colors ${
        active
          ? 'border-b-2 border-blue-600 text-blue-600 dark:text-blue-400'
          : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
      }`}
    >
      {icon}
      {label}
    </button>
  )
}

// Placeholder components - will be created separately
function BackupsTab({ backups, loading, onCreateBackup, onRefresh, onPreview, onExport }: any) {
  const { t } = useTranslation()
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('backups')}</h1>
        <div className="flex gap-3">
          <button
            onClick={onExport}
            className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors flex items-center gap-2"
          >
            <Download className="w-4 h-4" />
            {t('export')}
          </button>
          <button
            onClick={onRefresh}
            disabled={loading}
            className="px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            {t('refresh')}
          </button>
          <button
            onClick={onCreateBackup}
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            {t('newBackup')}
          </button>
        </div>
      </div>

      {loading && backups.length === 0 ? (
        <div className="text-center py-12 text-gray-500 dark:text-gray-400">{t('loading')}</div>
      ) : backups.length === 0 ? (
        <div className="text-center py-12">
          <Database className="w-16 h-16 text-gray-400 dark:text-gray-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">No backups yet</h3>
          <p className="text-gray-500 dark:text-gray-400 mb-4">Create your first backup to get started</p>
          <button
            onClick={onCreateBackup}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {t('newBackup')}
          </button>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-900">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Database
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Created
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Size
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {backups.map((backup: Backup) => (
                <tr key={backup.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                    {backup.database_name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={backup.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                    {new Date(backup.created_at).toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                    {backup.size ? formatBytes(backup.size) : '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <button
                      onClick={() => onPreview(backup)}
                      className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 flex items-center gap-1"
                    >
                      <Eye className="w-4 h-4" />
                      {t('preview')}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function ActivityTab({ backups }: { backups: Backup[] }) {
  const { t } = useTranslation()

  // Derive activity from real backup data (most recent first). No fabricated rows.
  const items = [...backups].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  const statusFor = (status: string): ActivityItemProps['status'] => {
    if (status === 'completed') return 'success'
    if (status === 'failed') return 'error'
    return 'info'
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">{t('activity')} Log</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        {items.length === 0 ? (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            No recent activity
          </div>
        ) : (
          <div className="space-y-4">
            {items.map((backup) => (
              <ActivityItem
                key={backup.id}
                icon={<Database className="w-5 h-5" />}
                title={`Backup ${t(backup.status)}`}
                description={`${backup.database_name}${
                  backup.size ? ' • ' + formatBytes(backup.size) : ''
                }`}
                time={new Date(backup.created_at).toLocaleString()}
                status={statusFor(backup.status)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

interface ActivityItemProps {
  icon: React.ReactNode
  title: string
  description: string
  time: string
  status: 'success' | 'error' | 'info'
}

function ActivityItem({ icon, title, description, time, status }: ActivityItemProps) {
  const colors: Record<ActivityItemProps['status'], string> = {
    success: 'bg-green-100 dark:bg-green-900/20 text-green-600 dark:text-green-400',
    error: 'bg-red-100 dark:bg-red-900/20 text-red-600 dark:text-red-400',
    info: 'bg-blue-100 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400'
  }

  return (
    <div className="flex items-start gap-4 p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
      <div className={`p-2 rounded-lg ${colors[status]}`}>
        {icon}
      </div>
      <div className="flex-1">
        <h3 className="font-medium text-gray-900 dark:text-white">{title}</h3>
        <p className="text-sm text-gray-600 dark:text-gray-400">{description}</p>
        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">{time}</p>
      </div>
    </div>
  )
}

function SettingsTab({ config, onConfigChange, onSave, loading }: any) {
  const { t } = useTranslation()
  if (!config) return <div className="dark:text-white">{t('loading')}</div>

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">{t('settings')}</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('apiUrl')}
            </label>
            <input
              type="text"
              value={config.api_url}
              onChange={(e) => onConfigChange({ ...config, api_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('apiKey')}
            </label>
            <input
              type="password"
              value={config.api_key}
              onChange={(e) => onConfigChange({ ...config, api_key: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="notifications"
              checked={config.notifications_enabled}
              onChange={(e) => onConfigChange({ ...config, notifications_enabled: e.target.checked })}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="notifications" className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('notifications')}
            </label>
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="autoLaunch"
              checked={config.auto_launch}
              onChange={async (e) => {
                const enabled = e.target.checked
                try {
                  await invoke('set_auto_launch', { enabled })
                  onConfigChange({ ...config, auto_launch: enabled })
                } catch (err) {
                  console.error('Failed to set auto-launch:', err)
                }
              }}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="autoLaunch" className="text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <Power className="w-4 h-4" />
              {t('autoLaunch')}
            </label>
          </div>

          <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={onSave}
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {t('save')} {t('settings')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Quick Search Dialog
function QuickSearchDialog({ onClose, onSelect }: any) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const search = async () => {
      if (!query.trim()) {
        setResults([])
        return
      }

      setLoading(true)
      try {
        const searchResults = await invoke<SearchResult[]>('quick_search', { query })
        setResults(searchResults)
      } catch (err) {
        console.error('Search failed:', err)
      } finally {
        setLoading(false)
      }
    }

    const debounce = setTimeout(search, 300)
    return () => clearTimeout(debounce)
  }, [query])

  return (
    <div className="fixed inset-0 bg-black/50 flex items-start justify-center pt-20 z-50" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-2xl w-full max-w-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <Command className="rounded-lg border-0">
          <div className="flex items-center border-b border-gray-200 dark:border-gray-700 px-4">
            <Search className="w-5 h-5 text-gray-400 dark:text-gray-500" />
            <input
              className="w-full px-4 py-4 bg-transparent border-0 outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500"
              placeholder="Search backups, databases, settings..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              autoFocus
            />
          </div>
          <div className="max-h-96 overflow-y-auto p-2">
            {loading ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">Searching...</div>
            ) : results.length === 0 && query ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">No results found</div>
            ) : (
              results.map((result) => (
                <button
                  key={result.id}
                  onClick={() => onSelect(result)}
                  className="w-full text-left px-4 py-3 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                >
                  <div className="font-medium text-gray-900 dark:text-white">{result.title}</div>
                  <div className="text-sm text-gray-600 dark:text-gray-400">{result.description}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-500 mt-1">{result.category}</div>
                </button>
              ))
            )}
          </div>
          <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-2 text-xs text-gray-500 dark:text-gray-500">
            Tip: Use Cmd+K to open search • ↑↓ to navigate • Enter to select • Esc to close
          </div>
        </Command>
      </div>
    </div>
  )
}

// Backup Preview Modal
function BackupPreviewModal({ preview, onClose, onRestore }: any) {
  const { t } = useTranslation()
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">{t('preview')}: {preview.database_name}</h2>
            <button onClick={onClose} className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
              <X className="w-6 h-6" />
            </button>
          </div>
        </div>
        <div className="p-6">
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">Tables ({preview.tables.length})</h3>
            <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 space-y-2">
              {preview.tables.map((table: TableInfo) => (
                <div key={table.name} className="flex justify-between items-center">
                  <span className="text-gray-900 dark:text-white font-medium">{table.name}</span>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    {table.row_count.toLocaleString()} rows • {formatBytes(table.size_bytes)}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="mb-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">Total Size</h3>
            <div className="text-3xl font-bold text-blue-600 dark:text-blue-400">
              {formatBytes(preview.total_size)}
            </div>
          </div>

          <div className="flex gap-3">
            <button
              onClick={onRestore}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              {t('restore')} Backup
            </button>
            <button
              onClick={onClose}
              className="px-6 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
            >
              {t('close')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Export Dialog
function ExportDialog({ onClose, onExport }: any) {
  const { t } = useTranslation()
  const [format, setFormat] = useState('pdf')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-2xl w-full max-w-md"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">{t('export')} Backups</h2>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('exportFormat')}
            </label>
            <select
              value={format}
              onChange={(e) => setFormat(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg"
            >
              <option value="pdf">PDF</option>
              <option value="excel">Excel (XLSX)</option>
              <option value="csv">CSV</option>
            </select>
          </div>

          <div className="flex gap-3 pt-4">
            <button
              onClick={() => onExport(format)}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              {t('export')}
            </button>
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
            >
              {t('cancel')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  const colors: Record<string, string> = {
    completed: 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-400',
    running: 'bg-blue-100 dark:bg-blue-900/20 text-blue-800 dark:text-blue-400',
    failed: 'bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-400',
    pending: 'bg-yellow-100 dark:bg-yellow-900/20 text-yellow-800 dark:text-yellow-400',
  }

  return (
    <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[status] || 'bg-gray-100 dark:bg-gray-900/20 text-gray-800 dark:text-gray-400'}`}>
      {t(status)}
    </span>
  )
}

// Export the already-fetched backups to a local file. Returns the file name.
function exportBackupsToFile(format: string, backups: Backup[]): string {
  const headers = ['Database', 'Status', 'Created', 'Size']
  const rows = backups.map((b) => ({
    Database: b.database_name,
    Status: b.status,
    Created: new Date(b.created_at).toLocaleString(),
    Size: b.size ? formatBytes(b.size) : '-'
  }))
  const stamp = Date.now()

  if (format === 'excel') {
    const worksheet = XLSX.utils.json_to_sheet(rows)
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Backups')
    const filename = `backups-${stamp}.xlsx`
    XLSX.writeFile(workbook, filename)
    return filename
  }

  if (format === 'pdf') {
    const doc = new jsPDF()
    doc.text('Backups', 14, 16)
    autoTable(doc, {
      startY: 20,
      head: [headers],
      body: rows.map((r) => [r.Database, r.Status, r.Created, r.Size])
    })
    const filename = `backups-${stamp}.pdf`
    doc.save(filename)
    return filename
  }

  // Default: CSV
  const escape = (value: string) => `"${value.replace(/"/g, '""')}"`
  const csv = [
    headers.join(','),
    ...rows.map((r) => headers.map((h) => escape(String((r as Record<string, string>)[h] ?? ''))).join(','))
  ].join('\n')
  const filename = `backups-${stamp}.csv`
  downloadBlob(new Blob([csv], { type: 'text/csv;charset=utf-8' }), filename)
  return filename
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(url)
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

export default EnhancedApp
