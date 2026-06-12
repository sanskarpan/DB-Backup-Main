'use client'

import { useState } from 'react'
import {
  Shield,
  AlertTriangle,
  Lock,
  Eye,
  AlertCircle,
  CheckCircle,
  XCircle,
  TrendingUp,
  Activity,
  FileKey2,
  Key,
  Scan,
  ShieldCheck,
  Ban,
  Zap,
  Cloud,
  Server,
} from 'lucide-react'

export default function SecurityPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'ransomware' | 'encryption' | 'alerts'>('overview')

  // Mock data
  const securityStats = {
    protectedBackups: 156,
    threatsBlocked: 0,
    encryptionRate: 100,
    lastScan: '2 minutes ago',
  }

  const encryptionMethods = [
    { name: 'AES-256-GCM', backups: 120, percentage: 77 },
    { name: 'ChaCha20-Poly1305', backups: 36, percentage: 23 },
  ]

  const immutableStorageProviders = [
    { name: 'AWS S3 Object Lock', status: 'active', backups: 85, icon: Cloud },
    { name: 'Azure Immutable Blobs', status: 'active', backups: 42, icon: Cloud },
    { name: 'GCS Retention Policies', status: 'active', backups: 29, icon: Cloud },
  ]

  const recentScans = [
    { timestamp: '2 min ago', files: 1247, threats: 0, status: 'clean' },
    { timestamp: '1 hour ago', files: 1189, threats: 0, status: 'clean' },
    { timestamp: '6 hours ago', files: 1145, threats: 0, status: 'clean' },
  ]

  const tabs = [
    { id: 'overview', name: 'Overview', icon: Activity },
    { id: 'ransomware', name: 'Ransomware Detection', icon: AlertTriangle },
    { id: 'encryption', name: 'Encryption', icon: Lock },
    { id: 'alerts', name: 'Threat Alerts', icon: Eye },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Hero Section */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-red-500 via-red-600 to-orange-600 p-8 text-white shadow-2xl">
          <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-20"></div>
          <div className="relative z-10">
            <div className="flex items-center gap-3 mb-2">
              <Shield className="w-10 h-10" />
              <h1 className="text-4xl font-bold">Security & Protection</h1>
            </div>
            <p className="text-red-100 text-lg max-w-3xl">
              Enterprise-grade security with multi-cloud immutable storage, real-time ransomware detection, and end-to-end encryption
            </p>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-lg">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <CheckCircle className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{securityStats.protectedBackups}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Protected Backups</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-red-500 to-red-600 text-white shadow-lg">
                <Ban className="w-6 h-6" />
              </div>
              <TrendingUp className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{securityStats.threatsBlocked}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Threats Blocked</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-lg">
                <Key className="w-6 h-6" />
              </div>
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950 px-2 py-1 rounded-full">
                {securityStats.encryptionRate}%
              </span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{securityStats.encryptionRate}%</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Encryption Rate</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-lg">
                <Scan className="w-6 h-6" />
              </div>
              <span className="flex h-3 w-3">
                <span className="animate-ping absolute inline-flex h-3 w-3 rounded-full bg-purple-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-purple-500"></span>
              </span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{securityStats.lastScan}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Last Scan</div>
          </div>
        </div>

        {/* Tabs */}
        <div className="modern-card modern-card-dark overflow-hidden">
          <div className="border-b border-gray-200 dark:border-gray-700">
            <nav className="flex gap-2 p-4">
              {tabs.map((tab) => {
                const Icon = tab.icon
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id as any)}
                    className={`flex items-center gap-2 px-4 py-3 rounded-xl font-medium text-sm transition-all ${
                      activeTab === tab.id
                        ? 'bg-gradient-to-r from-orange-500 to-orange-600 text-white shadow-lg shadow-orange-500/30'
                        : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    {tab.name}
                  </button>
                )
              })}
            </nav>
          </div>

          <div className="p-6">
            {/* Overview Tab */}
            {activeTab === 'overview' && (
              <div className="space-y-6">
                {/* Protection Status */}
                <div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                    <Shield className="w-5 h-5 text-orange-500" />
                    Protection Status
                  </h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {immutableStorageProviders.map((provider) => {
                      const Icon = provider.icon
                      return (
                        <div key={provider.name} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                          <div className="flex items-center justify-between mb-3">
                            <div className="flex items-center gap-3">
                              <div className="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 text-white">
                                <Icon className="w-5 h-5" />
                              </div>
                              <div>
                                <h4 className="font-semibold text-gray-900 dark:text-white">{provider.name}</h4>
                                <p className="text-xs text-gray-600 dark:text-gray-400">{provider.backups} backups</p>
                              </div>
                            </div>
                            <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800">
                              <CheckCircle className="w-3 h-3" />
                              Active
                            </span>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>

                {/* Recent Scans */}
                <div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                    <Scan className="w-5 h-5 text-orange-500" />
                    Recent Scans
                  </h3>
                  <div className="space-y-3">
                    {recentScans.map((scan, i) => (
                      <div key={i} className="flex items-center justify-between bg-emerald-50 dark:bg-emerald-950 border border-emerald-200 dark:border-emerald-800 rounded-xl p-4">
                        <div className="flex items-center gap-3">
                          <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                          <div>
                            <p className="font-medium text-gray-900 dark:text-white">Scan completed - {scan.status}</p>
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                              {scan.files} files scanned, {scan.threats} threats detected
                            </p>
                          </div>
                        </div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">{scan.timestamp}</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Feature Highlights */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950 dark:to-blue-900 rounded-xl p-6 border border-blue-200 dark:border-blue-800">
                    <Lock className="w-8 h-8 text-blue-600 dark:text-blue-400 mb-3" />
                    <h4 className="font-bold text-gray-900 dark:text-white mb-2">Immutable Storage</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      WORM protection prevents malicious deletion across AWS, Azure, and GCP
                    </p>
                  </div>
                  <div className="bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950 dark:to-purple-900 rounded-xl p-6 border border-purple-200 dark:border-purple-800">
                    <AlertTriangle className="w-8 h-8 text-purple-600 dark:text-purple-400 mb-3" />
                    <h4 className="font-bold text-gray-900 dark:text-white mb-2">Ransomware Detection</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Real-time entropy analysis and behavioral detection identify threats early
                    </p>
                  </div>
                  <div className="bg-gradient-to-br from-emerald-50 to-emerald-100 dark:from-emerald-950 dark:to-emerald-900 rounded-xl p-6 border border-emerald-200 dark:border-emerald-800">
                    <Shield className="w-8 h-8 text-emerald-600 dark:text-emerald-400 mb-3" />
                    <h4 className="font-bold text-gray-900 dark:text-white mb-2">Multi-Layer Protection</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Defense-in-depth with legal holds, retention policies, and auto-isolation
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Ransomware Tab */}
            {activeTab === 'ransomware' && (
              <div className="space-y-6">
                <div className="bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950 dark:to-purple-900 rounded-xl p-6 border border-purple-200 dark:border-purple-800">
                  <div className="flex items-start gap-4">
                    <div className="p-3 rounded-xl bg-purple-600 text-white">
                      <AlertTriangle className="w-6 h-6" />
                    </div>
                    <div className="flex-1">
                      <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">Ransomware Detection Active</h3>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">
                        All backups are continuously monitored for ransomware signatures and suspicious patterns
                      </p>
                      <div className="grid grid-cols-3 gap-4">
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">98.7%</div>
                          <div className="text-xs text-gray-600 dark:text-gray-400">Detection Rate</div>
                        </div>
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">&lt;5s</div>
                          <div className="text-xs text-gray-600 dark:text-gray-400">Scan Time</div>
                        </div>
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">0</div>
                          <div className="text-xs text-gray-600 dark:text-gray-400">Active Threats</div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="font-bold text-gray-900 dark:text-white mb-4">Detection Methods</h4>
                  <div className="space-y-3">
                    {[
                      { name: 'Entropy Analysis', description: 'Detects unusual file randomness patterns', status: 'active' },
                      { name: 'Signature Matching', description: 'Identifies known ransomware families', status: 'active' },
                      { name: 'Behavioral Detection', description: 'Monitors for suspicious file modifications', status: 'active' },
                      { name: 'ML-Based Analysis', description: 'Uses AI to detect emerging threats', status: 'active' },
                    ].map((method) => (
                      <div key={method.name} className="flex items-center justify-between bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div>
                          <h5 className="font-semibold text-gray-900 dark:text-white">{method.name}</h5>
                          <p className="text-sm text-gray-600 dark:text-gray-400">{method.description}</p>
                        </div>
                        <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800">
                          <Zap className="w-3 h-3" />
                          Active
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Encryption Tab */}
            {activeTab === 'encryption' && (
              <div className="space-y-6">
                <div className="bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950 dark:to-blue-900 rounded-xl p-6 border border-blue-200 dark:border-blue-800">
                  <div className="flex items-center gap-3 mb-4">
                    <Key className="w-8 h-8 text-blue-600 dark:text-blue-400" />
                    <div>
                      <h3 className="text-xl font-bold text-gray-900 dark:text-white">End-to-End Encryption</h3>
                      <p className="text-sm text-gray-600 dark:text-gray-400">All backups are encrypted at rest and in transit</p>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="font-bold text-gray-900 dark:text-white mb-4">Encryption Methods</h4>
                  <div className="space-y-4">
                    {encryptionMethods.map((method) => (
                      <div key={method.name} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div className="flex justify-between mb-2">
                          <div>
                            <h5 className="font-semibold text-gray-900 dark:text-white">{method.name}</h5>
                            <p className="text-sm text-gray-600 dark:text-gray-400">{method.backups} backups</p>
                          </div>
                          <span className="text-lg font-bold text-blue-600 dark:text-blue-400">{method.percentage}%</span>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                          <div
                            className="bg-gradient-to-r from-blue-500 to-blue-600 h-3 rounded-full transition-all"
                            style={{ width: `${method.percentage}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-1">256-bit</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Key Length</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-1">100%</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Coverage</div>
                  </div>
                </div>
              </div>
            )}

            {/* Alerts Tab */}
            {activeTab === 'alerts' && (
              <div className="space-y-6">
                <div className="bg-emerald-50 dark:bg-emerald-950 border border-emerald-200 dark:border-emerald-800 rounded-xl p-6">
                  <div className="flex items-center gap-3 mb-2">
                    <CheckCircle className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                    <h3 className="text-lg font-bold text-gray-900 dark:text-white">All Systems Secure</h3>
                  </div>
                  <p className="text-gray-600 dark:text-gray-400">No security threats detected in the last 30 days</p>
                </div>

                <div>
                  <h4 className="font-bold text-gray-900 dark:text-white mb-4">Alert Configuration</h4>
                  <div className="space-y-3">
                    {[
                      { name: 'Ransomware Detection', enabled: true, severity: 'critical' },
                      { name: 'Failed Encryption', enabled: true, severity: 'high' },
                      { name: 'Unauthorized Access', enabled: true, severity: 'high' },
                      { name: 'Storage Quota Warning', enabled: true, severity: 'medium' },
                    ].map((alert) => (
                      <div key={alert.name} className="flex items-center justify-between bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div className="flex items-center gap-3">
                          <div className={`w-3 h-3 rounded-full ${alert.enabled ? 'bg-emerald-500' : 'bg-gray-300'}`} />
                          <div>
                            <h5 className="font-semibold text-gray-900 dark:text-white">{alert.name}</h5>
                            <p className="text-xs text-gray-600 dark:text-gray-400">Severity: {alert.severity}</p>
                          </div>
                        </div>
                        <span className={`px-2.5 py-1 rounded-full text-xs font-semibold ${
                          alert.enabled
                            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400'
                            : 'bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-800 dark:text-gray-400'
                        }`}>
                          {alert.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
