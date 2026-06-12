'use client';

import React, { useState } from 'react';
import {
  Settings as SettingsIcon,
  User,
  Shield,
  Bell,
  Key,
  Save,
  Eye,
  EyeOff,
  Copy,
  RefreshCw,
  Trash2,
  Mail,
  Lock,
  Globe,
  Palette,
  Clock,
  CheckCircle2,
  AlertCircle,
  Plus,
  X,
  Webhook,
  Smartphone,
} from 'lucide-react';

interface ApiKey {
  id: string;
  name: string;
  key: string;
  createdAt: Date;
  lastUsed?: Date;
  permissions: string[];
}

type TabType = 'general' | 'security' | 'notifications' | 'api';

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<TabType>('general');
  const [showApiKey, setShowApiKey] = useState<{ [key: string]: boolean }>({});
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([
    {
      id: '1',
      name: 'Production API',
      key: 'sk_live_abc123xyz789def456ghi',
      createdAt: new Date('2024-01-15'),
      lastUsed: new Date('2024-01-22'),
      permissions: ['read', 'write', 'delete'],
    },
    {
      id: '2',
      name: 'Development API',
      key: 'sk_dev_xyz789abc123ghi456def',
      createdAt: new Date('2024-01-10'),
      lastUsed: new Date('2024-01-20'),
      permissions: ['read', 'write'],
    },
  ]);

  // General Settings State
  const [generalSettings, setGeneralSettings] = useState({
    companyName: 'Acme Corporation',
    email: 'admin@acme.com',
    timezone: 'UTC-8',
    language: 'en',
    dateFormat: 'MM/DD/YYYY',
  });

  // Security Settings State
  const [securitySettings, setSecuritySettings] = useState({
    twoFactorAuth: true,
    sessionTimeout: '30',
    ipWhitelist: true,
    auditLog: true,
    encryptionAtRest: true,
  });

  // Notification Settings State
  const [notificationSettings, setNotificationSettings] = useState({
    emailNotifications: true,
    backupComplete: true,
    backupFailed: true,
    lowStorage: true,
    securityAlerts: true,
    weeklyReport: false,
    slackIntegration: false,
    webhookUrl: '',
  });

  const [saved, setSaved] = useState(false);
  const [newApiKeyName, setNewApiKeyName] = useState('');
  const [showNewApiKeyForm, setShowNewApiKeyForm] = useState(false);

  const handleSave = () => {
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  const handleGenerateApiKey = () => {
    if (!newApiKeyName.trim()) return;

    const newKey: ApiKey = {
      id: `${Date.now()}`,
      name: newApiKeyName,
      key: `sk_live_${Math.random().toString(36).substring(2, 15)}${Math.random().toString(36).substring(2, 15)}`,
      createdAt: new Date(),
      permissions: ['read', 'write'],
    };

    setApiKeys([...apiKeys, newKey]);
    setNewApiKeyName('');
    setShowNewApiKeyForm(false);
  };

  const handleCopyApiKey = (key: string) => {
    navigator.clipboard.writeText(key);
  };

  const handleDeleteApiKey = (id: string) => {
    setApiKeys(apiKeys.filter((k) => k.id !== id));
  };

  const toggleShowApiKey = (id: string) => {
    setShowApiKey((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const maskApiKey = (key: string) => {
    return `${key.substring(0, 12)}${'•'.repeat(20)}${key.substring(key.length - 4)}`;
  };

  return (
    <div className="min-h-screen bg-[var(--background)]">
      {/* Hero Section */}
      <div className="gradient-hero relative overflow-hidden py-16 px-6">
        <div className="absolute inset-0 opacity-10">
          <div className="absolute top-20 left-10 w-72 h-72 bg-white rounded-full blur-3xl"></div>
          <div className="absolute bottom-20 right-10 w-96 h-96 bg-white rounded-full blur-3xl"></div>
        </div>

        <div className="max-w-7xl mx-auto relative z-10">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-white/20 backdrop-blur-sm rounded-xl">
              <SettingsIcon className="w-8 h-8 text-white" />
            </div>
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">Settings</h1>
              <p className="text-white/90 text-lg">Manage your account preferences and application settings</p>
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-8">
            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">API Keys</p>
                  <p className="text-3xl font-bold text-white">{apiKeys.length}</p>
                </div>
                <Key className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">2FA Status</p>
                  <p className="text-3xl font-bold text-white">
                    {securitySettings.twoFactorAuth ? 'On' : 'Off'}
                  </p>
                </div>
                <Shield className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Notifications</p>
                  <p className="text-3xl font-bold text-white">
                    {Object.values(notificationSettings).filter(Boolean).length}
                  </p>
                </div>
                <Bell className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Session Timeout</p>
                  <p className="text-3xl font-bold text-white">{securitySettings.sessionTimeout}m</p>
                </div>
                <Clock className="w-8 h-8 text-white/50" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Save Success Message */}
        {saved && (
          <div className="modern-card mb-6 border-l-4 border-l-[var(--success)] bg-[var(--success)]/5 animate-fade-in">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="w-5 h-5 text-[var(--success)]" />
              <p className="text-[var(--success)] font-medium">Settings saved successfully!</p>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Sidebar Navigation */}
          <div className="lg:col-span-1">
            <div className="modern-card sticky top-6">
              <nav className="space-y-2">
                <button
                  onClick={() => setActiveTab('general')}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                    activeTab === 'general'
                      ? 'bg-gradient-orange text-white'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--surface)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <User className="w-5 h-5" />
                  <span className="font-medium">General</span>
                </button>

                <button
                  onClick={() => setActiveTab('security')}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                    activeTab === 'security'
                      ? 'bg-gradient-orange text-white'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--surface)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Shield className="w-5 h-5" />
                  <span className="font-medium">Security</span>
                </button>

                <button
                  onClick={() => setActiveTab('notifications')}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                    activeTab === 'notifications'
                      ? 'bg-gradient-orange text-white'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--surface)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Bell className="w-5 h-5" />
                  <span className="font-medium">Notifications</span>
                </button>

                <button
                  onClick={() => setActiveTab('api')}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                    activeTab === 'api'
                      ? 'bg-gradient-orange text-white'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--surface)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Key className="w-5 h-5" />
                  <span className="font-medium">API Keys</span>
                </button>
              </nav>
            </div>
          </div>

          {/* Main Content */}
          <div className="lg:col-span-3">
            {/* General Settings */}
            {activeTab === 'general' && (
              <div className="space-y-6 animate-fade-in">
                <div className="modern-card">
                  <h2 className="text-2xl font-bold text-[var(--text-primary)] mb-6 flex items-center gap-2">
                    <User className="w-6 h-6" />
                    General Settings
                  </h2>

                  <div className="space-y-6">
                    <div>
                      <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                        Company Name
                      </label>
                      <input
                        type="text"
                        value={generalSettings.companyName}
                        onChange={(e) => setGeneralSettings({ ...generalSettings, companyName: e.target.value })}
                        className="input-modern"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                        Email Address
                      </label>
                      <div className="relative">
                        <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-[var(--text-tertiary)]" />
                        <input
                          type="email"
                          value={generalSettings.email}
                          onChange={(e) => setGeneralSettings({ ...generalSettings, email: e.target.value })}
                          className="input-modern pl-10"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div>
                        <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                          <Globe className="w-4 h-4 inline mr-1" />
                          Timezone
                        </label>
                        <select
                          value={generalSettings.timezone}
                          onChange={(e) => setGeneralSettings({ ...generalSettings, timezone: e.target.value })}
                          className="input-modern"
                        >
                          <option value="UTC-8">Pacific Time (UTC-8)</option>
                          <option value="UTC-5">Eastern Time (UTC-5)</option>
                          <option value="UTC+0">UTC</option>
                          <option value="UTC+1">Central European Time (UTC+1)</option>
                          <option value="UTC+8">Singapore Time (UTC+8)</option>
                        </select>
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                          <Palette className="w-4 h-4 inline mr-1" />
                          Language
                        </label>
                        <select
                          value={generalSettings.language}
                          onChange={(e) => setGeneralSettings({ ...generalSettings, language: e.target.value })}
                          className="input-modern"
                        >
                          <option value="en">English</option>
                          <option value="es">Spanish</option>
                          <option value="fr">French</option>
                          <option value="de">German</option>
                          <option value="ja">Japanese</option>
                        </select>
                      </div>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                        Date Format
                      </label>
                      <select
                        value={generalSettings.dateFormat}
                        onChange={(e) => setGeneralSettings({ ...generalSettings, dateFormat: e.target.value })}
                        className="input-modern"
                      >
                        <option value="MM/DD/YYYY">MM/DD/YYYY</option>
                        <option value="DD/MM/YYYY">DD/MM/YYYY</option>
                        <option value="YYYY-MM-DD">YYYY-MM-DD</option>
                      </select>
                    </div>

                    <div className="pt-4 border-t border-[var(--border)]">
                      <button onClick={handleSave} className="btn-primary flex items-center gap-2">
                        <Save className="w-5 h-5" />
                        Save Changes
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Security Settings */}
            {activeTab === 'security' && (
              <div className="space-y-6 animate-fade-in">
                <div className="modern-card">
                  <h2 className="text-2xl font-bold text-[var(--text-primary)] mb-6 flex items-center gap-2">
                    <Shield className="w-6 h-6" />
                    Security Settings
                  </h2>

                  <div className="space-y-6">
                    {/* Two-Factor Authentication */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <Lock className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">Two-Factor Authentication</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Add an extra layer of security to your account
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={securitySettings.twoFactorAuth}
                          onChange={(e) =>
                            setSecuritySettings({ ...securitySettings, twoFactorAuth: e.target.checked })
                          }
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    {/* Session Timeout */}
                    <div className="p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex items-center gap-2 mb-3">
                        <Clock className="w-5 h-5 text-[var(--orange-primary)]" />
                        <h3 className="font-semibold text-[var(--text-primary)]">Session Timeout</h3>
                      </div>
                      <select
                        value={securitySettings.sessionTimeout}
                        onChange={(e) => setSecuritySettings({ ...securitySettings, sessionTimeout: e.target.value })}
                        className="input-modern"
                      >
                        <option value="15">15 minutes</option>
                        <option value="30">30 minutes</option>
                        <option value="60">1 hour</option>
                        <option value="120">2 hours</option>
                        <option value="0">Never</option>
                      </select>
                    </div>

                    {/* IP Whitelist */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <Globe className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">IP Whitelist</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Restrict access to specific IP addresses
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={securitySettings.ipWhitelist}
                          onChange={(e) => setSecuritySettings({ ...securitySettings, ipWhitelist: e.target.checked })}
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    {/* Audit Log */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <AlertCircle className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">Audit Log</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Track all account activities and changes
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={securitySettings.auditLog}
                          onChange={(e) => setSecuritySettings({ ...securitySettings, auditLog: e.target.checked })}
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    {/* Encryption at Rest */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <Lock className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">Encryption at Rest</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Encrypt all stored backups (AES-256)
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={securitySettings.encryptionAtRest}
                          onChange={(e) =>
                            setSecuritySettings({ ...securitySettings, encryptionAtRest: e.target.checked })
                          }
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    <div className="pt-4 border-t border-[var(--border)]">
                      <button onClick={handleSave} className="btn-primary flex items-center gap-2">
                        <Save className="w-5 h-5" />
                        Save Security Settings
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Notification Settings */}
            {activeTab === 'notifications' && (
              <div className="space-y-6 animate-fade-in">
                <div className="modern-card">
                  <h2 className="text-2xl font-bold text-[var(--text-primary)] mb-6 flex items-center gap-2">
                    <Bell className="w-6 h-6" />
                    Notification Settings
                  </h2>

                  <div className="space-y-6">
                    {/* Email Notifications */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <Mail className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">Email Notifications</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Receive notifications via email
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={notificationSettings.emailNotifications}
                          onChange={(e) =>
                            setNotificationSettings({ ...notificationSettings, emailNotifications: e.target.checked })
                          }
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    {notificationSettings.emailNotifications && (
                      <div className="pl-6 space-y-4 animate-fade-in">
                        <div className="flex items-center justify-between">
                          <label className="text-sm text-[var(--text-primary)]">Backup Completed</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationSettings.backupComplete}
                              onChange={(e) =>
                                setNotificationSettings({ ...notificationSettings, backupComplete: e.target.checked })
                              }
                            />
                            <span className="toggle-slider"></span>
                          </label>
                        </div>

                        <div className="flex items-center justify-between">
                          <label className="text-sm text-[var(--text-primary)]">Backup Failed</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationSettings.backupFailed}
                              onChange={(e) =>
                                setNotificationSettings({ ...notificationSettings, backupFailed: e.target.checked })
                              }
                            />
                            <span className="toggle-slider"></span>
                          </label>
                        </div>

                        <div className="flex items-center justify-between">
                          <label className="text-sm text-[var(--text-primary)]">Low Storage Warning</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationSettings.lowStorage}
                              onChange={(e) =>
                                setNotificationSettings({ ...notificationSettings, lowStorage: e.target.checked })
                              }
                            />
                            <span className="toggle-slider"></span>
                          </label>
                        </div>

                        <div className="flex items-center justify-between">
                          <label className="text-sm text-[var(--text-primary)]">Security Alerts</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationSettings.securityAlerts}
                              onChange={(e) =>
                                setNotificationSettings({ ...notificationSettings, securityAlerts: e.target.checked })
                              }
                            />
                            <span className="toggle-slider"></span>
                          </label>
                        </div>

                        <div className="flex items-center justify-between">
                          <label className="text-sm text-[var(--text-primary)]">Weekly Summary Report</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationSettings.weeklyReport}
                              onChange={(e) =>
                                setNotificationSettings({ ...notificationSettings, weeklyReport: e.target.checked })
                              }
                            />
                            <span className="toggle-slider"></span>
                          </label>
                        </div>
                      </div>
                    )}

                    {/* Slack Integration */}
                    <div className="flex items-start justify-between p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <Smartphone className="w-5 h-5 text-[var(--orange-primary)]" />
                          <h3 className="font-semibold text-[var(--text-primary)]">Slack Integration</h3>
                        </div>
                        <p className="text-sm text-[var(--text-secondary)]">
                          Send notifications to Slack channel
                        </p>
                      </div>
                      <label className="toggle">
                        <input
                          type="checkbox"
                          checked={notificationSettings.slackIntegration}
                          onChange={(e) =>
                            setNotificationSettings({ ...notificationSettings, slackIntegration: e.target.checked })
                          }
                        />
                        <span className="toggle-slider"></span>
                      </label>
                    </div>

                    {/* Webhook */}
                    <div className="p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex items-center gap-2 mb-3">
                        <Webhook className="w-5 h-5 text-[var(--orange-primary)]" />
                        <h3 className="font-semibold text-[var(--text-primary)]">Webhook URL</h3>
                      </div>
                      <input
                        type="url"
                        placeholder="https://your-webhook-url.com/notifications"
                        value={notificationSettings.webhookUrl}
                        onChange={(e) =>
                          setNotificationSettings({ ...notificationSettings, webhookUrl: e.target.value })
                        }
                        className="input-modern"
                      />
                      <p className="text-xs text-[var(--text-secondary)] mt-2">
                        Receive real-time notifications via webhook POST requests
                      </p>
                    </div>

                    <div className="pt-4 border-t border-[var(--border)]">
                      <button onClick={handleSave} className="btn-primary flex items-center gap-2">
                        <Save className="w-5 h-5" />
                        Save Notification Settings
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* API Keys */}
            {activeTab === 'api' && (
              <div className="space-y-6 animate-fade-in">
                <div className="modern-card">
                  <div className="flex items-center justify-between mb-6">
                    <h2 className="text-2xl font-bold text-[var(--text-primary)] flex items-center gap-2">
                      <Key className="w-6 h-6" />
                      API Keys
                    </h2>
                    <button
                      onClick={() => setShowNewApiKeyForm(!showNewApiKeyForm)}
                      className="btn-primary flex items-center gap-2"
                    >
                      <Plus className="w-5 h-5" />
                      Generate New Key
                    </button>
                  </div>

                  {/* New API Key Form */}
                  {showNewApiKeyForm && (
                    <div className="p-4 bg-[var(--surface)] rounded-lg border-2 border-[var(--orange-primary)] mb-6 animate-fade-in">
                      <div className="flex gap-3">
                        <input
                          type="text"
                          placeholder="API Key Name (e.g., Production API)"
                          value={newApiKeyName}
                          onChange={(e) => setNewApiKeyName(e.target.value)}
                          className="input-modern flex-1"
                        />
                        <button
                          onClick={handleGenerateApiKey}
                          disabled={!newApiKeyName.trim()}
                          className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Generate
                        </button>
                        <button onClick={() => setShowNewApiKeyForm(false)} className="btn-ghost">
                          <X className="w-5 h-5" />
                        </button>
                      </div>
                    </div>
                  )}

                  {/* API Keys List */}
                  <div className="space-y-4">
                    {apiKeys.length === 0 ? (
                      <div className="empty-state py-12">
                        <Key className="empty-state-icon" />
                        <h3 className="empty-state-title">No API keys</h3>
                        <p className="empty-state-description">
                          Generate your first API key to start using the API
                        </p>
                      </div>
                    ) : (
                      apiKeys.map((apiKey) => (
                        <div
                          key={apiKey.id}
                          className="p-6 bg-[var(--surface)] rounded-lg border border-[var(--border)] hover:border-[var(--orange-light)] transition-all"
                        >
                          <div className="flex items-start justify-between mb-4">
                            <div>
                              <h3 className="font-semibold text-[var(--text-primary)] mb-1">{apiKey.name}</h3>
                              <div className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                                <Clock className="w-4 h-4" />
                                Created {apiKey.createdAt.toLocaleDateString()}
                                {apiKey.lastUsed && (
                                  <>
                                    <span>•</span>
                                    <span>Last used {apiKey.lastUsed.toLocaleDateString()}</span>
                                  </>
                                )}
                              </div>
                            </div>
                            <button
                              onClick={() => handleDeleteApiKey(apiKey.id)}
                              className="text-[var(--error)] hover:bg-[var(--error)]/10 p-2 rounded-lg transition-all"
                            >
                              <Trash2 className="w-5 h-5" />
                            </button>
                          </div>

                          <div className="flex items-center gap-2 mb-3">
                            <code className="flex-1 px-3 py-2 bg-[var(--background)] border border-[var(--border)] rounded-lg font-mono text-sm text-[var(--text-primary)]">
                              {showApiKey[apiKey.id] ? apiKey.key : maskApiKey(apiKey.key)}
                            </code>
                            <button
                              onClick={() => toggleShowApiKey(apiKey.id)}
                              className="btn-ghost p-2"
                            >
                              {showApiKey[apiKey.id] ? (
                                <EyeOff className="w-5 h-5" />
                              ) : (
                                <Eye className="w-5 h-5" />
                              )}
                            </button>
                            <button
                              onClick={() => handleCopyApiKey(apiKey.key)}
                              className="btn-ghost p-2"
                            >
                              <Copy className="w-5 h-5" />
                            </button>
                          </div>

                          <div className="flex gap-2">
                            {apiKey.permissions.map((permission) => (
                              <span key={permission} className="badge badge-orange text-xs">
                                {permission}
                              </span>
                            ))}
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* API Documentation Link */}
                <div className="stats-card">
                  <div className="flex items-start gap-3">
                    <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                      <Key className="w-5 h-5 text-[var(--orange-primary)]" />
                    </div>
                    <div>
                      <h4 className="font-semibold text-[var(--text-primary)] mb-1">API Documentation</h4>
                      <p className="text-sm text-[var(--text-secondary)] mb-3">
                        Learn how to integrate our API into your applications
                      </p>
                      <a href="/docs/api" className="text-[var(--orange-primary)] text-sm font-medium hover:underline">
                        View API Docs →
                      </a>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
