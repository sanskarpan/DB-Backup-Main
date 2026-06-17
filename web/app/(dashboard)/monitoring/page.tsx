'use client'

import { useState } from 'react'
import {
  Activity,
  BarChart3,
  Shield,
  DollarSign,
  TrendingUp,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Database,
  HardDrive,
  Zap,
  Cpu,
  MemoryStick,
  Gauge,
  LineChart,
  PieChart,
  Target,
} from 'lucide-react'

export default function MonitoringPage() {
  const [selectedView, setSelectedView] = useState<'overview' | 'security' | 'cost' | 'performance'>('overview')
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h')

  // Mock data
  const stats = {
    totalBackups: 1247,
    successRate: 98.5,
    storageUsed: 2.4 * 1024 * 1024 * 1024 * 1024, // 2.4TB
    storageTotal: 5 * 1024 * 1024 * 1024 * 1024, // 5TB
    activeDatabases: 12,
    failedBackups: 3,
    avgBackupDuration: 124.5,
    costMonthly: 186.42,
  }

  const performanceMetrics = [
    { name: 'Avg Throughput', value: '125 MB/s', icon: Zap, trend: '+12%', color: 'text-blue-600 dark:text-blue-400' },
    { name: 'API Latency (p95)', value: '245ms', icon: Activity, trend: '-8%', color: 'text-emerald-600 dark:text-emerald-400' },
    { name: 'CPU Usage', value: '45%', icon: Cpu, trend: '+3%', color: 'text-amber-600 dark:text-amber-400' },
    { name: 'Memory Usage', value: '62%', icon: MemoryStick, trend: '+5%', color: 'text-purple-600 dark:text-purple-400' },
  ]

  const costBreakdown = [
    { name: 'Storage', amount: 120.5, percentage: 65, color: 'bg-blue-500' },
    { name: 'Transfer', amount: 45.2, percentage: 24, color: 'bg-emerald-500' },
    { name: 'API Calls', amount: 20.7, percentage: 11, color: 'bg-amber-500' },
  ]

  const viewCards = [
    { id: 'overview', name: 'Overview', description: 'Backup status & health', icon: Activity, color: 'from-blue-500 to-blue-600' },
    { id: 'security', name: 'Security', description: 'Encryption & compliance', icon: Shield, color: 'from-emerald-500 to-emerald-600' },
    { id: 'cost', name: 'Cost', description: 'Usage & optimization', icon: DollarSign, color: 'from-amber-500 to-amber-600' },
    { id: 'performance', name: 'Performance', description: 'Speed & throughput', icon: Zap, color: 'from-purple-500 to-purple-600' },
  ]

  const timeRanges = [
    { value: '1h', label: 'Last Hour' },
    { value: '24h', label: 'Last 24h' },
    { value: '7d', label: 'Last 7 Days' },
    { value: '30d', label: 'Last 30 Days' },
  ]

  const formatBytes = (bytes: number) => {
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    if (bytes === 0) return '0 B'
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Hero Section */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-cyan-500 via-cyan-600 to-blue-600 p-8 text-white shadow-2xl">
          <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-20"></div>
          <div className="relative z-10">
            <div className="flex items-center gap-3 mb-2">
              <Activity className="w-10 h-10" />
              <h1 className="text-4xl font-bold">Monitoring & Observability</h1>
            </div>
            <p className="text-cyan-100 text-lg max-w-3xl">
              Real-time insights into your backup infrastructure with comprehensive metrics and analytics
            </p>
          </div>
        </div>

        {/* Time Range Selector */}
        <div className="flex gap-2">
          {timeRanges.map((range) => (
            <button
              key={range.value}
              onClick={() => setTimeRange(range.value as any)}
              className={`px-4 py-2 rounded-xl text-sm font-medium transition-all ${
                timeRange === range.value
                  ? 'bg-gradient-to-r from-cyan-500 to-cyan-600 text-white shadow-lg shadow-cyan-500/30'
                  : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-700 hover:border-cyan-500 dark:hover:border-cyan-500'
              }`}
            >
              {range.label}
            </button>
          ))}
        </div>

        {/* View Selector Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {viewCards.map((view) => {
            const Icon = view.icon
            const isActive = selectedView === view.id
            return (
              <button
                key={view.id}
                onClick={() => setSelectedView(view.id as any)}
                className={`p-6 rounded-2xl border-2 transition-all text-left ${
                  isActive
                    ? 'border-orange-500 dark:border-orange-500 bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-950 dark:to-orange-900 shadow-xl shadow-orange-500/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-orange-300 dark:hover:border-orange-700 hover:shadow-lg'
                }`}
              >
                <div className={`p-3 rounded-xl bg-gradient-to-br ${view.color} text-white inline-block mb-3 shadow-lg`}>
                  <Icon className="w-6 h-6" />
                </div>
                <h3 className={`font-bold text-lg mb-1 ${isActive ? 'text-orange-600 dark:text-orange-400' : 'text-gray-900 dark:text-white'}`}>
                  {view.name}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">{view.description}</p>
              </button>
            )
          })}
        </div>

        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-lg">
                <Database className="w-6 h-6" />
              </div>
              <TrendingUp className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.totalBackups.toLocaleString()}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Total Backups</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-lg">
                <CheckCircle className="w-6 h-6" />
              </div>
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950 px-2 py-1 rounded-full">
                Excellent
              </span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.successRate}%</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Success Rate</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-lg">
                <HardDrive className="w-6 h-6" />
              </div>
              <Gauge className="w-5 h-5 text-purple-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">
              {((stats.storageUsed / stats.storageTotal) * 100).toFixed(1)}%
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              Storage Used ({formatBytes(stats.storageUsed)})
            </div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-amber-500 to-amber-600 text-white shadow-lg">
                <Clock className="w-6 h-6" />
              </div>
              <Target className="w-5 h-5 text-amber-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.avgBackupDuration.toFixed(1)}s</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Avg Duration</div>
          </div>
        </div>

        {/* Main Content Area */}
        <div className="modern-card modern-card-dark p-8">
          {/* Overview */}
          {selectedView === 'overview' && (
            <div className="space-y-8">
              <div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
                  <BarChart3 className="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                  Backup Overview
                </h2>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="text-3xl font-bold text-gray-900 dark:text-white mb-2">{stats.activeDatabases}</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Active Databases</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="text-3xl font-bold text-gray-900 dark:text-white mb-2">{stats.failedBackups}</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Failed Backups (24h)</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="text-3xl font-bold text-gray-900 dark:text-white mb-2">{stats.avgBackupDuration.toFixed(1)}s</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Avg Duration</div>
                  </div>
                </div>

                <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-bold text-gray-900 dark:text-white">Storage Utilization</h3>
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      {formatBytes(stats.storageUsed)} / {formatBytes(stats.storageTotal)}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden">
                    <div
                      className="bg-gradient-to-r from-cyan-500 to-cyan-600 h-4 rounded-full transition-all shadow-lg"
                      style={{ width: `${(stats.storageUsed / stats.storageTotal) * 100}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Security */}
          {selectedView === 'security' && (
            <div className="space-y-8">
              <div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
                  <Shield className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                  Security & Compliance
                </h2>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                  <div className="bg-emerald-50 dark:bg-emerald-950 rounded-xl p-6 border border-emerald-200 dark:border-emerald-800">
                    <div className="flex items-center gap-3 mb-2">
                      <CheckCircle className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                      <div className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">100%</div>
                    </div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Encrypted Backups</div>
                  </div>
                  <div className="bg-emerald-50 dark:bg-emerald-950 rounded-xl p-6 border border-emerald-200 dark:border-emerald-800">
                    <div className="flex items-center gap-3 mb-2">
                      <CheckCircle className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                      <div className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">0</div>
                    </div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Failed Auth (24h)</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="flex items-center gap-3 mb-2">
                      <Activity className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                      <div className="text-3xl font-bold text-gray-900 dark:text-white">1,234</div>
                    </div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Audit Events</div>
                  </div>
                </div>

                <div>
                  <h3 className="font-bold text-gray-900 dark:text-white mb-4">Encryption Coverage</h3>
                  <div className="space-y-4">
                    <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                      <div className="flex justify-between mb-2">
                        <span className="font-medium text-gray-900 dark:text-white">AES-256-GCM</span>
                        <span className="text-sm text-gray-600 dark:text-gray-400">75%</span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div className="bg-gradient-to-r from-emerald-500 to-emerald-600 h-2 rounded-full" style={{ width: '75%' }} />
                      </div>
                    </div>
                    <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                      <div className="flex justify-between mb-2">
                        <span className="font-medium text-gray-900 dark:text-white">ChaCha20-Poly1305</span>
                        <span className="text-sm text-gray-600 dark:text-gray-400">25%</span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div className="bg-gradient-to-r from-emerald-500 to-emerald-600 h-2 rounded-full" style={{ width: '25%' }} />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Cost */}
          {selectedView === 'cost' && (
            <div className="space-y-8">
              <div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
                  <DollarSign className="w-6 h-6 text-amber-600 dark:text-amber-400" />
                  Cost Analytics
                </h2>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                  <div className="bg-amber-50 dark:bg-amber-950 rounded-xl p-6 border border-amber-200 dark:border-amber-800">
                    <div className="text-3xl font-bold text-amber-600 dark:text-amber-400 mb-2">
                      ${stats.costMonthly.toFixed(2)}
                    </div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Monthly Cost</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="text-3xl font-bold text-gray-900 dark:text-white mb-2">$0.023</div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Cost per GB</div>
                  </div>
                  <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                    <div className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
                      ${(stats.costMonthly * 12).toFixed(2)}
                    </div>
                    <div className="text-sm text-gray-600 dark:text-gray-400">Annual Projection</div>
                  </div>
                </div>

                <div>
                  <h3 className="font-bold text-gray-900 dark:text-white mb-4">Cost Breakdown</h3>
                  <div className="space-y-4">
                    {costBreakdown.map((item) => (
                      <div key={item.name} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div className="flex justify-between mb-2">
                          <span className="font-medium text-gray-900 dark:text-white">{item.name}</span>
                          <span className="text-sm text-gray-600 dark:text-gray-400">
                            ${item.amount.toFixed(2)} ({item.percentage}%)
                          </span>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                          <div className={`${item.color} h-2 rounded-full`} style={{ width: `${item.percentage}%` }} />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Performance */}
          {selectedView === 'performance' && (
            <div className="space-y-8">
              <div>
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
                  <Zap className="w-6 h-6 text-purple-600 dark:text-purple-400" />
                  Performance Metrics
                </h2>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                  {performanceMetrics.map((metric) => {
                    const Icon = metric.icon
                    return (
                      <div key={metric.name} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
                        <div className="flex items-center justify-between mb-3">
                          <Icon className={`w-6 h-6 ${metric.color}`} />
                          <span className={`text-xs font-semibold px-2 py-1 rounded-full ${
                            metric.trend.startsWith('+') ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400' : 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400'
                          }`}>
                            {metric.trend}
                          </span>
                        </div>
                        <div className="text-2xl font-bold text-gray-900 dark:text-white mb-1">{metric.value}</div>
                        <div className="text-sm text-gray-600 dark:text-gray-400">{metric.name}</div>
                      </div>
                    )
                  })}
                </div>

                <div>
                  <h3 className="font-bold text-gray-900 dark:text-white mb-4">Resource Utilization</h3>
                  <div className="space-y-4">
                    {[
                      { name: 'CPU', value: 45, color: 'bg-blue-500' },
                      { name: 'Memory', value: 62, color: 'bg-purple-500' },
                      { name: 'Disk I/O', value: 38, color: 'bg-cyan-500' },
                      { name: 'Network', value: 28, color: 'bg-emerald-500' },
                    ].map((resource) => (
                      <div key={resource.name} className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
                        <div className="flex justify-between mb-2">
                          <span className="font-medium text-gray-900 dark:text-white">{resource.name}</span>
                          <span className="text-sm text-gray-600 dark:text-gray-400">{resource.value}%</span>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                          <div
                            className={`${resource.color} h-2 rounded-full transition-all`}
                            style={{ width: `${resource.value}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Grafana Integration */}
        <div className="modern-card modern-card-dark p-8">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
            <LineChart className="w-6 h-6 text-orange-600 dark:text-orange-400" />
            Grafana Dashboards
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {[
              { title: 'Backup Overview', description: 'Comprehensive backup metrics and trends', icon: BarChart3, color: 'text-blue-600 dark:text-blue-400' },
              { title: 'Security & Compliance', description: 'Encryption status and audit logs', icon: Shield, color: 'text-emerald-600 dark:text-emerald-400' },
              { title: 'Cost Analytics', description: 'Storage costs and optimization opportunities', icon: DollarSign, color: 'text-amber-600 dark:text-amber-400' },
              { title: 'Performance Metrics', description: 'Throughput, latency, and resource usage', icon: TrendingUp, color: 'text-purple-600 dark:text-purple-400' },
            ].map((dashboard) => {
              const Icon = dashboard.icon
              return (
                <a
                  key={dashboard.title}
                  href="#"
                  className="block bg-gray-50 dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700 hover:border-orange-500 dark:hover:border-orange-500 hover:shadow-lg transition-all group"
                >
                  <div className="flex items-start gap-4">
                    <Icon className={`w-6 h-6 ${dashboard.color} group-hover:scale-110 transition-transform`} />
                    <div className="flex-1">
                      <h3 className="font-bold text-gray-900 dark:text-white mb-1 group-hover:text-orange-600 dark:group-hover:text-orange-400 transition-colors">
                        {dashboard.title}
                      </h3>
                      <p className="text-sm text-gray-600 dark:text-gray-400">{dashboard.description}</p>
                    </div>
                  </div>
                </a>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}
