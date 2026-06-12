'use client'

import React from 'react'
import {
  Activity,
  Database,
  HardDrive,
  Clock,
  TrendingUp,
  TrendingDown,
  CheckCircle2,
  XCircle,
  Zap,
  Server,
  AlertTriangle,
  BarChart3,
  PieChart,
  LineChart,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface MetricValue {
  value: number
  timestamp: Date
  label?: string
}

export interface MetricSeries {
  name: string
  data: MetricValue[]
  color?: string
  unit?: string
}

export interface SystemMetrics {
  // Backup metrics
  backups_total: number
  backups_success: number
  backups_failed: number
  backups_in_progress: number
  backup_success_rate: number
  avg_backup_duration: number // ms
  avg_backup_size: number // bytes
  total_backup_size: number // bytes

  // Database metrics
  databases_total: number
  databases_healthy: number
  databases_warning: number
  databases_critical: number

  // Storage metrics
  storage_used: number // bytes
  storage_available: number // bytes
  storage_total: number // bytes

  // Performance metrics
  api_requests_per_second: number
  api_avg_response_time: number // ms
  api_error_rate: number // percentage

  // System metrics
  cpu_usage: number // percentage
  memory_usage: number // percentage
  disk_io_rate: number // bytes/s

  // Time series data
  backup_rate_series?: MetricSeries
  api_latency_series?: MetricSeries
  storage_usage_series?: MetricSeries
  error_rate_series?: MetricSeries

  timestamp: Date
}

interface MetricsDashboardProps {
  metrics: SystemMetrics
  refreshInterval?: number
  onRefresh?: () => void
  className?: string
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = seconds / 60
  return `${minutes.toFixed(1)}m`
}

const formatRate = (value: number, decimals: number = 1): string => {
  return value.toFixed(decimals)
}

const getHealthColor = (percentage: number, reverse: boolean = false) => {
  if (reverse) {
    // For things like errors where lower is better
    if (percentage < 5) return 'text-green-600'
    if (percentage < 15) return 'text-yellow-600'
    return 'text-red-600'
  } else {
    // For things like success rate where higher is better
    if (percentage >= 95) return 'text-green-600'
    if (percentage >= 80) return 'text-yellow-600'
    return 'text-red-600'
  }
}

const getTrendIcon = (current: number, previous: number) => {
  if (current > previous) {
    return <TrendingUp className="w-4 h-4 text-green-600" />
  } else if (current < previous) {
    return <TrendingDown className="w-4 h-4 text-red-600" />
  }
  return null
}

export function MetricsDashboard({
  metrics,
  refreshInterval,
  onRefresh,
  className,
}: MetricsDashboardProps) {
  // Auto-refresh
  React.useEffect(() => {
    if (refreshInterval && onRefresh) {
      const interval = setInterval(onRefresh, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [refreshInterval, onRefresh])

  const storagePercentage = (metrics.storage_used / metrics.storage_total) * 100

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Top Level KPIs */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Backup Success Rate"
          value={`${metrics.backup_success_rate.toFixed(1)}%`}
          icon={<CheckCircle2 className="w-6 h-6" />}
          color="blue"
          subtitle={`${metrics.backups_success} / ${metrics.backups_total} successful`}
          trend={metrics.backup_success_rate >= 95 ? 'up' : 'down'}
        />

        <MetricCard
          title="Active Databases"
          value={metrics.databases_total.toString()}
          icon={<Database className="w-6 h-6" />}
          color="green"
          subtitle={`${metrics.databases_healthy} healthy`}
          trend="stable"
        />

        <MetricCard
          title="Storage Used"
          value={formatBytes(metrics.storage_used)}
          icon={<HardDrive className="w-6 h-6" />}
          color="purple"
          subtitle={`${storagePercentage.toFixed(1)}% of ${formatBytes(metrics.storage_total)}`}
          trend={storagePercentage > 80 ? 'up' : 'stable'}
        />

        <MetricCard
          title="API Response Time"
          value={formatDuration(metrics.api_avg_response_time)}
          icon={<Zap className="w-6 h-6" />}
          color="orange"
          subtitle={`${metrics.api_requests_per_second.toFixed(1)} req/s`}
          trend={metrics.api_avg_response_time < 200 ? 'down' : 'up'}
        />
      </div>

      {/* Backup Overview */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-blue-600" />
            Backup Status Overview
          </h3>
          <div className="space-y-4">
            <StatusBar
              label="Successful"
              value={metrics.backups_success}
              total={metrics.backups_total}
              color="green"
              icon={<CheckCircle2 className="w-4 h-4" />}
            />
            <StatusBar
              label="Failed"
              value={metrics.backups_failed}
              total={metrics.backups_total}
              color="red"
              icon={<XCircle className="w-4 h-4" />}
            />
            <StatusBar
              label="In Progress"
              value={metrics.backups_in_progress}
              total={metrics.backups_total}
              color="blue"
              icon={<Activity className="w-4 h-4" />}
            />
            <div className="pt-4 border-t border-gray-200">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <div className="text-gray-600">Avg Duration</div>
                  <div className="text-lg font-semibold text-gray-900">
                    {formatDuration(metrics.avg_backup_duration)}
                  </div>
                </div>
                <div>
                  <div className="text-gray-600">Avg Size</div>
                  <div className="text-lg font-semibold text-gray-900">
                    {formatBytes(metrics.avg_backup_size)}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Server className="w-5 h-5 text-blue-600" />
            Database Health
          </h3>
          <div className="space-y-4">
            <StatusBar
              label="Healthy"
              value={metrics.databases_healthy}
              total={metrics.databases_total}
              color="green"
              icon={<CheckCircle2 className="w-4 h-4" />}
            />
            <StatusBar
              label="Warning"
              value={metrics.databases_warning}
              total={metrics.databases_total}
              color="yellow"
              icon={<AlertTriangle className="w-4 h-4" />}
            />
            <StatusBar
              label="Critical"
              value={metrics.databases_critical}
              total={metrics.databases_total}
              color="red"
              icon={<XCircle className="w-4 h-4" />}
            />
            <div className="pt-4 border-t border-gray-200">
              <div className="relative pt-1">
                <div className="flex mb-2 items-center justify-between">
                  <div className="text-sm font-semibold text-gray-700">Overall Health Score</div>
                  <div className="text-sm font-semibold text-gray-700">
                    {(
                      ((metrics.databases_healthy * 100 +
                        metrics.databases_warning * 60 +
                        metrics.databases_critical * 20) /
                        metrics.databases_total) *
                      1
                    ).toFixed(0)}
                    %
                  </div>
                </div>
                <div className="overflow-hidden h-2 text-xs flex rounded bg-gray-200">
                  <div
                    style={{
                      width: `${(metrics.databases_healthy / metrics.databases_total) * 100}%`,
                    }}
                    className="bg-green-500"
                  />
                  <div
                    style={{
                      width: `${(metrics.databases_warning / metrics.databases_total) * 100}%`,
                    }}
                    className="bg-yellow-500"
                  />
                  <div
                    style={{
                      width: `${(metrics.databases_critical / metrics.databases_total) * 100}%`,
                    }}
                    className="bg-red-500"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* System Performance */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Activity className="w-5 h-5 text-blue-600" />
            API Performance
          </h3>
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Requests/sec</span>
                <span className="text-lg font-semibold text-gray-900">
                  {formatRate(metrics.api_requests_per_second)}
                </span>
              </div>
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Avg Response Time</span>
                <span
                  className={clsx(
                    'text-lg font-semibold',
                    metrics.api_avg_response_time < 200 ? 'text-green-600' : 'text-yellow-600'
                  )}
                >
                  {formatDuration(metrics.api_avg_response_time)}
                </span>
              </div>
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Error Rate</span>
                <span className={clsx('text-lg font-semibold', getHealthColor(metrics.api_error_rate, true))}>
                  {metrics.api_error_rate.toFixed(2)}%
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Server className="w-5 h-5 text-blue-600" />
            System Resources
          </h3>
          <div className="space-y-4">
            <GaugeMetric
              label="CPU Usage"
              value={metrics.cpu_usage}
              max={100}
              unit="%"
              color={metrics.cpu_usage > 80 ? 'red' : metrics.cpu_usage > 60 ? 'yellow' : 'green'}
            />
            <GaugeMetric
              label="Memory Usage"
              value={metrics.memory_usage}
              max={100}
              unit="%"
              color={metrics.memory_usage > 80 ? 'red' : metrics.memory_usage > 60 ? 'yellow' : 'green'}
            />
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Disk I/O</span>
                <span className="text-lg font-semibold text-gray-900">
                  {formatBytes(metrics.disk_io_rate)}/s
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <HardDrive className="w-5 h-5 text-blue-600" />
            Storage Overview
          </h3>
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Used</span>
                <span className="text-lg font-semibold text-gray-900">
                  {formatBytes(metrics.storage_used)}
                </span>
              </div>
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Available</span>
                <span className="text-lg font-semibold text-gray-900">
                  {formatBytes(metrics.storage_available)}
                </span>
              </div>
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600">Total Backups</span>
                <span className="text-lg font-semibold text-gray-900">
                  {formatBytes(metrics.total_backup_size)}
                </span>
              </div>
            </div>
            <div className="pt-2">
              <GaugeMetric
                label="Storage"
                value={storagePercentage}
                max={100}
                unit="%"
                color={storagePercentage > 90 ? 'red' : storagePercentage > 75 ? 'yellow' : 'green'}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

interface MetricCardProps {
  title: string
  value: string
  icon: React.ReactNode
  color: 'blue' | 'green' | 'purple' | 'orange' | 'red'
  subtitle?: string
  trend?: 'up' | 'down' | 'stable'
}

function MetricCard({ title, value, icon, color, subtitle, trend }: MetricCardProps) {
  const colorClasses = {
    blue: 'text-blue-600 bg-blue-100',
    green: 'text-green-600 bg-green-100',
    purple: 'text-purple-600 bg-purple-100',
    orange: 'text-orange-600 bg-orange-100',
    red: 'text-red-600 bg-red-100',
  }

  const trendIcons = {
    up: <TrendingUp className="w-4 h-4 text-green-600" />,
    down: <TrendingDown className="w-4 h-4 text-red-600" />,
    stable: <Activity className="w-4 h-4 text-gray-400" />,
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-gray-600">{title}</span>
        <div className={clsx('p-2 rounded-lg', colorClasses[color])}>{icon}</div>
      </div>
      <div className="flex items-baseline justify-between">
        <div className="text-2xl font-bold text-gray-900">{value}</div>
        {trend && trendIcons[trend]}
      </div>
      {subtitle && <div className="text-xs text-gray-500 mt-1">{subtitle}</div>}
    </div>
  )
}

interface StatusBarProps {
  label: string
  value: number
  total: number
  color: 'green' | 'yellow' | 'red' | 'blue'
  icon: React.ReactNode
}

function StatusBar({ label, value, total, color, icon }: StatusBarProps) {
  const percentage = total > 0 ? (value / total) * 100 : 0

  const colorClasses = {
    green: { bg: 'bg-green-600', text: 'text-green-600' },
    yellow: { bg: 'bg-yellow-600', text: 'text-yellow-600' },
    red: { bg: 'bg-red-600', text: 'text-red-600' },
    blue: { bg: 'bg-blue-600', text: 'text-blue-600' },
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-600 flex items-center gap-2">
          <span className={colorClasses[color].text}>{icon}</span>
          {label}
        </span>
        <span className="text-sm font-semibold text-gray-900">
          {value} ({percentage.toFixed(1)}%)
        </span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={clsx('h-2 rounded-full transition-all', colorClasses[color].bg)}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  )
}

interface GaugeMetricProps {
  label: string
  value: number
  max: number
  unit: string
  color: 'green' | 'yellow' | 'red'
}

function GaugeMetric({ label, value, max, unit, color }: GaugeMetricProps) {
  const percentage = (value / max) * 100

  const colorClasses = {
    green: 'bg-green-600',
    yellow: 'bg-yellow-600',
    red: 'bg-red-600',
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-600">{label}</span>
        <span className="text-sm font-semibold text-gray-900">
          {value.toFixed(1)}
          {unit}
        </span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={clsx('h-2 rounded-full transition-all', colorClasses[color])}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  )
}

// Sample data generator for testing
export function generateSampleMetrics(): SystemMetrics {
  const backupsTotal = Math.floor(Math.random() * 1000) + 500
  const backupsSuccess = Math.floor(backupsTotal * (0.9 + Math.random() * 0.09))
  const backupsFailed = Math.floor(backupsTotal * Math.random() * 0.05)
  const backupsInProgress = backupsTotal - backupsSuccess - backupsFailed

  const databasesTotal = Math.floor(Math.random() * 20) + 5
  const databasesHealthy = Math.floor(databasesTotal * (0.7 + Math.random() * 0.2))
  const databasesWarning = Math.floor((databasesTotal - databasesHealthy) * Math.random())
  const databasesCritical = databasesTotal - databasesHealthy - databasesWarning

  const storageTotal = 1000000000000 // 1TB
  const storageUsed = Math.floor(storageTotal * (0.4 + Math.random() * 0.4))
  const storageAvailable = storageTotal - storageUsed

  return {
    backups_total: backupsTotal,
    backups_success: backupsSuccess,
    backups_failed: backupsFailed,
    backups_in_progress: backupsInProgress,
    backup_success_rate: (backupsSuccess / backupsTotal) * 100,
    avg_backup_duration: Math.random() * 600000 + 60000, // 1-10 minutes
    avg_backup_size: Math.random() * 5000000000 + 1000000000, // 1-5GB
    total_backup_size: storageUsed,
    databases_total: databasesTotal,
    databases_healthy: databasesHealthy,
    databases_warning: databasesWarning,
    databases_critical: databasesCritical,
    storage_used: storageUsed,
    storage_available: storageAvailable,
    storage_total: storageTotal,
    api_requests_per_second: Math.random() * 100 + 10,
    api_avg_response_time: Math.random() * 200 + 50, // 50-250ms
    api_error_rate: Math.random() * 5, // 0-5%
    cpu_usage: Math.random() * 80 + 10, // 10-90%
    memory_usage: Math.random() * 70 + 20, // 20-90%
    disk_io_rate: Math.random() * 100000000, // 0-100MB/s
    timestamp: new Date(),
  }
}
