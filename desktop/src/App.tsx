import { useState, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/tauri'
import { sendNotification } from '@tauri-apps/api/notification'
import { Database, Settings, Activity, Bell, Minimize, X } from 'lucide-react'
import { appWindow } from '@tauri-apps/api/window'

interface Backup {
  id: string
  database_id: string
  database_name: string
  status: string
  created_at: string
  size?: number
  duration?: number
}

interface Config {
  api_url: string
  api_key: string
  auto_backup_enabled: boolean
  backup_interval_hours: number
  notifications_enabled: boolean
}

function App() {
  const [activeTab, setActiveTab] = useState('backups')
  const [backups, setBackups] = useState<Backup[]>([])
  const [config, setConfig] = useState<Config | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadBackups()
    loadConfig()
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
    } catch (err) {
      console.error('Failed to load config:', err)
    }
  }

  const handleCreateBackup = async () => {
    setLoading(true)
    setError(null)
    try {
      await invoke('create_backup', {
        databaseId: 'default-db',
        options: {}
      })
      await sendNotification({
        title: 'Backup Started',
        body: 'Database backup has been initiated'
      })
      loadBackups()
    } catch (err) {
      setError(err as string)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveConfig = async () => {
    if (!config) return
    setLoading(true)
    try {
      await invoke('update_config', { config })
      await sendNotification({
        title: 'Settings Saved',
        body: 'Configuration has been updated successfully'
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
    <div className="flex flex-col h-screen bg-gray-50">
      {/* Custom Title Bar */}
      <div data-tauri-drag-region className="flex items-center justify-between h-8 bg-gray-800 text-white px-4">
        <div className="flex items-center gap-2 text-sm font-medium">
          <Database className="w-4 h-4" />
          DB Backup Desktop
        </div>
        <div className="flex gap-2">
          <button onClick={handleMinimize} className="hover:bg-gray-700 p-1 rounded">
            <Minimize className="w-4 h-4" />
          </button>
          <button onClick={handleClose} className="hover:bg-red-600 p-1 rounded">
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-gray-200 bg-white">
        <TabButton
          active={activeTab === 'backups'}
          onClick={() => setActiveTab('backups')}
          icon={<Database className="w-5 h-5" />}
          label="Backups"
        />
        <TabButton
          active={activeTab === 'activity'}
          onClick={() => setActiveTab('activity')}
          icon={<Activity className="w-5 h-5" />}
          label="Activity"
        />
        <TabButton
          active={activeTab === 'settings'}
          onClick={() => setActiveTab('settings')}
          icon={<Settings className="w-5 h-5" />}
          label="Settings"
        />
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-auto p-6">
        {error && (
          <div className="mb-4 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
            {error}
          </div>
        )}

        {activeTab === 'backups' && (
          <BackupsTab
            backups={backups}
            loading={loading}
            onCreateBackup={handleCreateBackup}
            onRefresh={loadBackups}
          />
        )}

        {activeTab === 'activity' && <ActivityTab />}

        {activeTab === 'settings' && (
          <SettingsTab
            config={config}
            onConfigChange={setConfig}
            onSave={handleSaveConfig}
            loading={loading}
          />
        )}
      </div>
    </div>
  )
}

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
          ? 'border-b-2 border-blue-600 text-blue-600'
          : 'text-gray-600 hover:text-gray-900'
      }`}
    >
      {icon}
      {label}
    </button>
  )
}

interface BackupsTabProps {
  backups: Backup[]
  loading: boolean
  onCreateBackup: () => void
  onRefresh: () => void
}

function BackupsTab({ backups, loading, onCreateBackup, onRefresh }: BackupsTabProps) {
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Backups</h1>
        <div className="flex gap-3">
          <button
            onClick={onRefresh}
            disabled={loading}
            className="px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
          >
            Refresh
          </button>
          <button
            onClick={onCreateBackup}
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            New Backup
          </button>
        </div>
      </div>

      {loading && backups.length === 0 ? (
        <div className="text-center py-12 text-gray-500">Loading backups...</div>
      ) : backups.length === 0 ? (
        <div className="text-center py-12">
          <Database className="w-16 h-16 text-gray-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">No backups yet</h3>
          <p className="text-gray-500 mb-4">Create your first backup to get started</p>
          <button
            onClick={onCreateBackup}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Create Backup
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Database
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Created
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Size
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Duration
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {backups.map((backup) => (
                <tr key={backup.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {backup.database_name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={backup.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {new Date(backup.created_at).toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {backup.size ? formatBytes(backup.size) : '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {backup.duration ? formatDuration(backup.duration) : '-'}
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

function ActivityTab() {
  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Activity Log</h1>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="space-y-4">
          <ActivityItem
            icon={<Database className="w-5 h-5" />}
            title="Backup Completed"
            description="PostgreSQL production backup completed successfully"
            time="2 minutes ago"
            status="success"
          />
          <ActivityItem
            icon={<Bell className="w-5 h-5" />}
            title="Notification Sent"
            description="Backup completion notification delivered"
            time="2 minutes ago"
            status="info"
          />
          <ActivityItem
            icon={<Settings className="w-5 h-5" />}
            title="Settings Updated"
            description="API configuration has been updated"
            time="15 minutes ago"
            status="info"
          />
        </div>
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
  const colors = {
    success: 'bg-green-100 text-green-600',
    error: 'bg-red-100 text-red-600',
    info: 'bg-blue-100 text-blue-600'
  }

  return (
    <div className="flex items-start gap-4 p-4 border border-gray-200 rounded-lg">
      <div className={`p-2 rounded-lg ${colors[status]}`}>
        {icon}
      </div>
      <div className="flex-1">
        <h3 className="font-medium text-gray-900">{title}</h3>
        <p className="text-sm text-gray-600">{description}</p>
        <p className="text-xs text-gray-500 mt-1">{time}</p>
      </div>
    </div>
  )
}

interface SettingsTabProps {
  config: Config | null
  onConfigChange: (config: Config) => void
  onSave: () => void
  loading: boolean
}

function SettingsTab({ config, onConfigChange, onSave, loading }: SettingsTabProps) {
  if (!config) return <div>Loading...</div>

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Settings</h1>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              API URL
            </label>
            <input
              type="text"
              value={config.api_url}
              onChange={(e) => onConfigChange({ ...config, api_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              API Key
            </label>
            <input
              type="password"
              value={config.api_key}
              onChange={(e) => onConfigChange({ ...config, api_key: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
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
            <label htmlFor="notifications" className="text-sm font-medium text-gray-700">
              Enable desktop notifications
            </label>
          </div>

          <div className="pt-4 border-t border-gray-200">
            <button
              onClick={onSave}
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              Save Settings
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    completed: 'bg-green-100 text-green-800',
    running: 'bg-blue-100 text-blue-800',
    failed: 'bg-red-100 text-red-800',
    pending: 'bg-yellow-100 text-yellow-800',
  }

  return (
    <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[status] || 'bg-gray-100 text-gray-800'}`}>
      {status}
    </span>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${minutes}m ${secs}s`
}

export default App
