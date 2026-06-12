'use client'

import { useState } from 'react'
import {
  Save,
  Database,
  HardDrive,
  Bell,
  Shield,
  Globe,
  Mail,
  Key,
  Server,
  Cloud,
  Smartphone,
} from 'lucide-react'
import { PWASettings } from '@/components/pwa/pwa-settings'

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState('storage')
  const [saveStatus, setSaveStatus] = useState<{ success: boolean; message: string } | null>(null)

  const tabs = [
    { id: 'storage', name: 'Storage', icon: HardDrive },
    { id: 'notifications', name: 'Notifications', icon: Bell },
    { id: 'security', name: 'Security', icon: Shield },
    { id: 'api', name: 'API', icon: Server },
    { id: 'pwa', name: 'Progressive Web App', icon: Smartphone },
  ]

  const handleSave = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setSaveStatus({ success: true, message: 'Settings saved successfully!' })
    setTimeout(() => setSaveStatus(null), 3000)
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Settings</h1>
        <p className="text-gray-500 mt-1">Configure your backup utility preferences</p>
      </div>

      <div className="flex space-x-6">
        {/* Sidebar */}
        <div className="w-64 bg-white rounded-lg border border-gray-200 p-4">
          <nav className="space-y-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`w-full flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
                  activeTab === tab.id
                    ? 'bg-primary text-white'
                    : 'text-gray-700 hover:bg-gray-100'
                }`}
              >
                <tab.icon className="w-5 h-5 mr-3" />
                {tab.name}
              </button>
            ))}
          </nav>
        </div>

        {/* Content */}
        <div className="flex-1 bg-white rounded-lg border border-gray-200 p-6">
          {saveStatus && (
            <div
              className={`mb-6 p-4 rounded-lg ${
                saveStatus.success
                  ? 'bg-green-50 text-green-800 border border-green-200'
                  : 'bg-red-50 text-red-800 border border-red-200'
              }`}
            >
              {saveStatus.message}
            </div>
          )}

          {/* Storage Settings */}
          {activeTab === 'storage' && (
            <form onSubmit={handleSave} className="space-y-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">Storage Configuration</h2>
                <p className="text-sm text-gray-600 mb-6">
                  Configure where and how your backups are stored
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Storage Provider
                </label>
                <select className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent">
                  <option value="local">Local Filesystem</option>
                  <option value="s3">Amazon S3</option>
                  <option value="gcs">Google Cloud Storage</option>
                  <option value="azure">Azure Blob Storage</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Storage Path / Bucket
                </label>
                <input
                  type="text"
                  defaultValue="/var/backups/db"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  placeholder="/var/backups/db"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Default Retention Period (days)
                </label>
                <input
                  type="number"
                  defaultValue="30"
                  min="1"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                />
              </div>

              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Enable compression for backups
                  </span>
                </label>
              </div>

              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Enable encryption for backups
                  </span>
                </label>
              </div>

              <button
                type="submit"
                className="flex items-center px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              >
                <Save className="w-5 h-5 mr-2" />
                Save Changes
              </button>
            </form>
          )}

          {/* Notifications Settings */}
          {activeTab === 'notifications' && (
            <form onSubmit={handleSave} className="space-y-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">
                  Notification Preferences
                </h2>
                <p className="text-sm text-gray-600 mb-6">
                  Choose when and how you want to be notified
                </p>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">Email Notifications</h3>
                <div className="space-y-3">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      defaultChecked
                      className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                    />
                    <span className="ml-2 text-sm text-gray-700">
                      Notify on successful backups
                    </span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      defaultChecked
                      className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                    />
                    <span className="ml-2 text-sm text-gray-700">Notify on backup failures</span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                    />
                    <span className="ml-2 text-sm text-gray-700">Weekly summary reports</span>
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Notification Email
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                  <input
                    type="email"
                    defaultValue="admin@example.com"
                    className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                    placeholder="admin@example.com"
                  />
                </div>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">SMTP Configuration</h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      SMTP Server
                    </label>
                    <input
                      type="text"
                      defaultValue="smtp.gmail.com"
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Port</label>
                      <input
                        type="number"
                        defaultValue="587"
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Encryption
                      </label>
                      <select className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent">
                        <option value="tls">TLS</option>
                        <option value="ssl">SSL</option>
                        <option value="none">None</option>
                      </select>
                    </div>
                  </div>
                </div>
              </div>

              <button
                type="submit"
                className="flex items-center px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              >
                <Save className="w-5 h-5 mr-2" />
                Save Changes
              </button>
            </form>
          )}

          {/* Security Settings */}
          {activeTab === 'security' && (
            <form onSubmit={handleSave} className="space-y-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">Security Settings</h2>
                <p className="text-sm text-gray-600 mb-6">
                  Manage authentication and encryption settings
                </p>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">Authentication</h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      JWT Secret Key
                    </label>
                    <div className="relative">
                      <Key className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                      <input
                        type="password"
                        defaultValue="your-secret-key-here"
                        className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Token Expiration (hours)
                    </label>
                    <input
                      type="number"
                      defaultValue="24"
                      min="1"
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                    />
                  </div>
                </div>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">Encryption</h3>
                <div className="space-y-3">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      defaultChecked
                      className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                    />
                    <span className="ml-2 text-sm text-gray-700">
                      Encrypt backups at rest (AES-256)
                    </span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      defaultChecked
                      className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                    />
                    <span className="ml-2 text-sm text-gray-700">
                      Encrypt data in transit (TLS)
                    </span>
                  </label>
                </div>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">Rate Limiting</h3>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Requests per minute
                  </label>
                  <input
                    type="number"
                    defaultValue="100"
                    min="1"
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  />
                </div>
              </div>

              <button
                type="submit"
                className="flex items-center px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              >
                <Save className="w-5 h-5 mr-2" />
                Save Changes
              </button>
            </form>
          )}

          {/* API Settings */}
          {activeTab === 'api' && (
            <form onSubmit={handleSave} className="space-y-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">API Configuration</h2>
                <p className="text-sm text-gray-600 mb-6">Configure API server settings</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">API Port</label>
                <input
                  type="number"
                  defaultValue="8080"
                  min="1"
                  max="65535"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Allowed Origins (CORS)
                </label>
                <textarea
                  defaultValue="http://localhost:3000&#10;https://app.example.com"
                  rows={3}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent font-mono text-sm"
                  placeholder="One origin per line"
                />
                <p className="mt-1 text-xs text-gray-500">Enter one origin per line</p>
              </div>

              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  />
                  <span className="ml-2 text-sm text-gray-700">Enable API documentation (Swagger)</span>
                </label>
              </div>

              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    defaultChecked
                    className="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  />
                  <span className="ml-2 text-sm text-gray-700">Enable request logging</span>
                </label>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3">API Keys</h3>
                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <code className="text-sm font-mono text-gray-900">
                      api_key_1234567890abcdef
                    </code>
                    <button
                      type="button"
                      className="text-xs text-blue-600 hover:text-blue-800"
                    >
                      Regenerate
                    </button>
                  </div>
                  <p className="text-xs text-gray-500">Created: Dec 28, 2025 • Last used: Today</p>
                </div>
              </div>

              <button
                type="submit"
                className="flex items-center px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              >
                <Save className="w-5 h-5 mr-2" />
                Save Changes
              </button>
            </form>
          )}

          {/* PWA Settings */}
          {activeTab === 'pwa' && <PWASettings />}
        </div>
      </div>
    </div>
  )
}
