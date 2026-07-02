'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { ThreatAlert } from '@db-backup/types'
import {
  Shield,
  AlertTriangle,
  Lock,
  Eye,
  CheckCircle,
  XCircle,
  Activity,
  Key,
  Scan,
  ShieldCheck,
  Ban,
  Zap,
  Cloud,
  Server,
  Loader2,
} from 'lucide-react'

function formatLastScan(value: string | null): string {
  if (!value) return 'Never'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Never'
  return date.toLocaleString()
}

export default function SecurityPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'ransomware' | 'encryption' | 'alerts'>('overview')

  const {
    data: stats,
    isLoading: statsLoading,
    isError: statsError,
  } = useQuery({
    queryKey: ['security-stats'],
    queryFn: () => api.getSecurityStats(),
  })

  const {
    data: alertsResponse,
    isLoading: alertsLoading,
    isError: alertsError,
  } = useQuery({
    queryKey: ['threat-alerts'],
    queryFn: () => api.listThreatAlerts(),
  })

  const alerts: ThreatAlert[] = alertsResponse?.alerts ?? []

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
              Ransomware detection and immutable, encrypted storage. Metrics below are
              reported directly by the backend security service.
            </p>
          </div>
        </div>

        {statsError && (
          <div className="rounded-xl border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-red-700 dark:text-red-300">
            Unable to load security statistics from the backend.
          </div>
        )}

        {/* Stats Cards - real data from GET /security/stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-lg">
                <ShieldCheck className="w-6 h-6" />
              </div>
              {stats?.detector_active ? (
                <CheckCircle className="w-5 h-5 text-emerald-500" />
              ) : (
                <XCircle className="w-5 h-5 text-gray-400" />
              )}
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
              {statsLoading ? '—' : stats?.detector_active ? 'Active' : 'Inactive'}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Ransomware Detector</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-red-500 to-red-600 text-white shadow-lg">
                <Ban className="w-6 h-6" />
              </div>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
              {statsLoading ? '—' : (stats?.threats_blocked ?? 0).toLocaleString()}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Threats Blocked</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-amber-500 to-amber-600 text-white shadow-lg">
                <AlertTriangle className="w-6 h-6" />
              </div>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
              {statsLoading ? '—' : (stats?.threats_detected ?? 0).toLocaleString()}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Threats Detected</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-lg">
                <Cloud className="w-6 h-6" />
              </div>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
              {statsLoading ? '—' : (stats?.configured_providers ?? 0).toLocaleString()}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Configured Providers</div>
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
                {/* Detector Status */}
                <div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                    <Shield className="w-5 h-5 text-orange-500" />
                    Protection Status
                  </h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="p-2 rounded-lg bg-gradient-to-br from-purple-500 to-purple-600 text-white">
                            <Scan className="w-5 h-5" />
                          </div>
                          <div>
                            <h4 className="font-semibold text-gray-900 dark:text-white">Ransomware Detector</h4>
                            <p className="text-xs text-gray-600 dark:text-gray-400">
                              Last scan: {stats ? formatLastScan(stats.last_scan_time) : '—'}
                            </p>
                          </div>
                        </div>
                        <span
                          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border ${
                            stats?.detector_active
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800'
                              : 'bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700'
                          }`}
                        >
                          {stats?.detector_active ? <CheckCircle className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                          {stats?.detector_active ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    </div>

                    <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 text-white">
                            <Server className="w-5 h-5" />
                          </div>
                          <div>
                            <h4 className="font-semibold text-gray-900 dark:text-white">Storage Providers</h4>
                            <p className="text-xs text-gray-600 dark:text-gray-400">
                              {(stats?.configured_providers ?? 0).toLocaleString()} configured
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-3">
                    Scans run on demand and no cumulative scan history is retained, so files
                    scanned, threats detected and threats blocked report honest zeros until a
                    scan records activity.
                  </p>
                </div>

                {/* Feature Highlights */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950 dark:to-blue-900 rounded-xl p-6 border border-blue-200 dark:border-blue-800">
                    <Lock className="w-8 h-8 text-blue-600 dark:text-blue-400 mb-3" />
                    <h4 className="font-bold text-gray-900 dark:text-white mb-2">Immutable Storage</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      WORM protection prevents malicious deletion across supported providers
                    </p>
                  </div>
                  <div className="bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950 dark:to-purple-900 rounded-xl p-6 border border-purple-200 dark:border-purple-800">
                    <AlertTriangle className="w-8 h-8 text-purple-600 dark:text-purple-400 mb-3" />
                    <h4 className="font-bold text-gray-900 dark:text-white mb-2">Ransomware Detection</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      Entropy analysis and behavioral detection identify threats during scans
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
                      <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                        Ransomware Detection {stats?.detector_active ? 'Active' : 'Inactive'}
                      </h3>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">
                        Backups are scanned for ransomware signatures and suspicious patterns.
                      </p>
                      <div className="grid grid-cols-3 gap-4">
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">
                            {(stats?.files_scanned ?? 0).toLocaleString()}
                          </div>
                          <div className="text-xs text-gray-600 dark:text-gray-400">Files Scanned</div>
                        </div>
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">
                            {(stats?.threats_detected ?? 0).toLocaleString()}
                          </div>
                          <div className="text-xs text-gray-600 dark:text-gray-400">Threats Detected</div>
                        </div>
                        <div className="bg-white dark:bg-gray-800 rounded-lg p-3">
                          <div className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
                            {alerts.filter((a) => a.status === 'active').length}
                          </div>
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
                      { name: 'Entropy Analysis', description: 'Detects unusual file randomness patterns' },
                      { name: 'Signature Matching', description: 'Identifies known ransomware families' },
                      { name: 'Behavioral Detection', description: 'Monitors for suspicious file modifications' },
                    ].map((method) => (
                      <div key={method.name} className="flex items-center justify-between bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div>
                          <h5 className="font-semibold text-gray-900 dark:text-white">{method.name}</h5>
                          <p className="text-sm text-gray-600 dark:text-gray-400">{method.description}</p>
                        </div>
                        <span
                          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border ${
                            stats?.detector_active
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800'
                              : 'bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700'
                          }`}
                        >
                          <Zap className="w-3 h-3" />
                          {stats?.detector_active ? 'Active' : 'Inactive'}
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
                      <p className="text-sm text-gray-600 dark:text-gray-400">Backups are encrypted at rest and in transit</p>
                    </div>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Supported ciphers include AES-256-GCM and ChaCha20-Poly1305 with 256-bit keys.
                    Per-backup encryption breakdown is not yet exposed by the backend API.
                  </p>
                </div>
              </div>
            )}

            {/* Alerts Tab - real data from GET /security/alerts */}
            {activeTab === 'alerts' && (
              <div className="space-y-6">
                {alertsError && (
                  <div className="rounded-xl border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4 text-red-700 dark:text-red-300">
                    Unable to load threat alerts from the backend.
                  </div>
                )}

                {alertsLoading ? (
                  <div className="flex items-center justify-center gap-2 py-12 text-gray-500 dark:text-gray-400">
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Loading threat alerts…
                  </div>
                ) : alerts.length === 0 ? (
                  <div className="bg-emerald-50 dark:bg-emerald-950 border border-emerald-200 dark:border-emerald-800 rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-2">
                      <CheckCircle className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                      <h3 className="text-lg font-bold text-gray-900 dark:text-white">No Active Threat Alerts</h3>
                    </div>
                    <p className="text-gray-600 dark:text-gray-400">
                      The backend reports no threat alerts.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {alerts.map((alert) => (
                      <div key={alert.id} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div className="flex items-center justify-between mb-1">
                          <h5 className="font-semibold text-gray-900 dark:text-white">{alert.title}</h5>
                          <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-red-50 text-red-700 border border-red-200 dark:bg-red-950 dark:text-red-400 dark:border-red-800">
                            {alert.severity}
                          </span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">{alert.description}</p>
                        <div className="mt-2 flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400">
                          <span>Type: {alert.type}</span>
                          <span>Status: {alert.status}</span>
                          {alert.detection_method && <span>Detected via: {alert.detection_method}</span>}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
