'use client'

import React from 'react'
import {
  Database,
  Activity,
  HardDrive,
  Clock,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  CheckCircle2,
  Zap,
  Users,
  FileText,
  Server,
  BarChart3,
  Gauge,
  Cpu,
  MemoryStick,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface DatabaseMetrics {
  database_id: string
  database_name: string
  database_type: 'postgres' | 'mysql' | 'mongodb' | 'redis' | 'mariadb'
  status: 'healthy' | 'warning' | 'critical' | 'offline'

  // Performance metrics
  queries_per_second: number
  avg_query_time: number // milliseconds
  slow_queries_count: number
  connections_active: number
  connections_max: number

  // Storage metrics
  size_total: number // bytes
  size_used: number // bytes
  size_available: number // bytes
  tables_count: number
  rows_count: number
  indexes_count: number

  // Resource metrics
  cpu_usage: number // percentage
  memory_usage: number // percentage
  disk_io_read: number // bytes/sec
  disk_io_write: number // bytes/sec

  // Backup metrics
  last_backup_at?: Date
  last_backup_size?: number
  backup_count_24h: number
  backup_count_7d: number

  // Additional metadata
  uptime: number // seconds
  version: string
  replication_status?: 'master' | 'replica' | 'standalone'
  replication_lag?: number // milliseconds

  timestamp: Date
}

interface DatabaseMetricsProps {
  metrics: DatabaseMetrics[]
  refreshInterval?: number // milliseconds
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

const formatBytesPerSec = (bytesPerSec: number): string => {
  return formatBytes(bytesPerSec) + '/s'
}

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    return `${days}d ${hours}h`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
  }
}

const getStatusColor = (status: DatabaseMetrics['status']) => {
  switch (status) {
    case 'healthy':
      return 'text-green-600 bg-green-100 border-green-200'
    case 'warning':
      return 'text-yellow-600 bg-yellow-100 border-yellow-200'
    case 'critical':
      return 'text-red-600 bg-red-100 border-red-200'
    case 'offline':
      return 'text-gray-600 bg-gray-100 border-gray-200'
  }
}

const getStatusIcon = (status: DatabaseMetrics['status']) => {
  switch (status) {
    case 'healthy':
      return <CheckCircle2 className="w-5 h-5" />
    case 'warning':
      return <AlertTriangle className="w-5 h-5" />
    case 'critical':
      return <AlertTriangle className="w-5 h-5" />
    case 'offline':
      return <Server className="w-5 h-5" />
  }
}

const getPerformanceColor = (percentage: number, reverse: boolean = false) => {
  if (reverse) {
    // For things like usage where lower is better
    if (percentage < 60) return 'text-green-600'
    if (percentage < 80) return 'text-yellow-600'
    return 'text-red-600'
  } else {
    // For things like availability where higher is better
    if (percentage > 80) return 'text-green-600'
    if (percentage > 60) return 'text-yellow-600'
    return 'text-red-600'
  }
}

const getDatabaseTypeIcon = (type: DatabaseMetrics['database_type']) => {
  return <Database className="w-5 h-5" />
}

export function DatabaseMetrics({ metrics, refreshInterval, onRefresh, className }: DatabaseMetricsProps) {
  const [selectedDatabase, setSelectedDatabase] = React.useState<string | null>(
    metrics.length > 0 ? metrics[0].database_id : null
  )

  // Auto-refresh
  React.useEffect(() => {
    if (refreshInterval && onRefresh) {
      const interval = setInterval(onRefresh, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [refreshInterval, onRefresh])

  const selectedMetrics = React.useMemo(() => {
    return metrics.find((m) => m.database_id === selectedDatabase)
  }, [metrics, selectedDatabase])

  const overallHealth = React.useMemo(() => {
    const healthCounts = {
      healthy: metrics.filter((m) => m.status === 'healthy').length,
      warning: metrics.filter((m) => m.status === 'warning').length,
      critical: metrics.filter((m) => m.status === 'critical').length,
      offline: metrics.filter((m) => m.status === 'offline').length,
    }

    const total = metrics.length
    const healthScore = (
      (healthCounts.healthy * 100 +
        healthCounts.warning * 60 +
        healthCounts.critical * 20 +
        healthCounts.offline * 0) /
      total
    )

    return { counts: healthCounts, score: healthScore }
  }, [metrics])

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Databases</span>
            <Database className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{metrics.length}</div>
          <div className="text-xs text-gray-500 mt-1">Monitored instances</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Healthy</span>
            <CheckCircle2 className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-2xl font-bold text-green-600">{overallHealth.counts.healthy}</div>
          <div className="text-xs text-gray-500 mt-1">
            {((overallHealth.counts.healthy / metrics.length) * 100).toFixed(0)}% of total
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Issues</span>
            <AlertTriangle className="w-5 h-5 text-yellow-600" />
          </div>
          <div className="text-2xl font-bold text-yellow-600">
            {overallHealth.counts.warning + overallHealth.counts.critical}
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {overallHealth.counts.warning} warnings, {overallHealth.counts.critical} critical
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Health Score</span>
            <Gauge className="w-5 h-5 text-blue-600" />
          </div>
          <div className={clsx('text-2xl font-bold', getPerformanceColor(overallHealth.score, false))}>
            {overallHealth.score.toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500 mt-1">Overall system health</div>
        </div>
      </div>

      {/* Database List and Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Database List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg shadow-sm border border-gray-200">
            <div className="px-4 py-3 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900">Databases</h3>
            </div>
            <div className="divide-y divide-gray-100 max-h-[600px] overflow-y-auto">
              {metrics.map((db) => (
                <button
                  key={db.database_id}
                  onClick={() => setSelectedDatabase(db.database_id)}
                  className={clsx(
                    'w-full px-4 py-3 text-left hover:bg-gray-50 transition-colors',
                    selectedDatabase === db.database_id && 'bg-blue-50'
                  )}
                >
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      {getDatabaseTypeIcon(db.database_type)}
                      <span className="font-medium text-gray-900">{db.database_name}</span>
                    </div>
                    <span
                      className={clsx(
                        'px-2 py-0.5 text-xs font-medium rounded-full border inline-flex items-center gap-1',
                        getStatusColor(db.status)
                      )}
                    >
                      {getStatusIcon(db.status)}
                      {db.status}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-gray-500">
                    <span className="flex items-center gap-1">
                      <Activity className="w-3 h-3" />
                      {db.queries_per_second.toFixed(0)} qps
                    </span>
                    <span className="flex items-center gap-1">
                      <Cpu className="w-3 h-3" />
                      {db.cpu_usage.toFixed(0)}%
                    </span>
                    <span className="flex items-center gap-1">
                      <HardDrive className="w-3 h-3" />
                      {formatBytes(db.size_used)}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Database Details */}
        <div className="lg:col-span-2">
          {selectedMetrics ? (
            <div className="space-y-6">
              {/* Database Info */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h3 className="text-xl font-semibold text-gray-900">{selectedMetrics.database_name}</h3>
                    <div className="flex items-center gap-3 mt-1 text-sm text-gray-600">
                      <span className="flex items-center gap-1">
                        <Database className="w-4 h-4" />
                        {selectedMetrics.database_type}
                      </span>
                      <span>{selectedMetrics.version}</span>
                      {selectedMetrics.replication_status && (
                        <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded">
                          {selectedMetrics.replication_status}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="text-right">
                    <div
                      className={clsx(
                        'inline-flex items-center gap-2 px-3 py-1.5 rounded-full border font-medium',
                        getStatusColor(selectedMetrics.status)
                      )}
                    >
                      {getStatusIcon(selectedMetrics.status)}
                      {selectedMetrics.status.toUpperCase()}
                    </div>
                    <div className="text-xs text-gray-500 mt-1">
                      Uptime: {formatUptime(selectedMetrics.uptime)}
                    </div>
                  </div>
                </div>

                {/* Quick Stats */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 pt-4 border-t border-gray-200">
                  <div>
                    <div className="text-xs text-gray-600 mb-1">Tables</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {selectedMetrics.tables_count.toLocaleString()}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-600 mb-1">Rows</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {selectedMetrics.rows_count.toLocaleString()}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-600 mb-1">Indexes</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {selectedMetrics.indexes_count.toLocaleString()}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-600 mb-1">Total Size</div>
                    <div className="text-lg font-semibold text-gray-900">
                      {formatBytes(selectedMetrics.size_total)}
                    </div>
                  </div>
                </div>
              </div>

              {/* Performance Metrics */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                  <Activity className="w-5 h-5 text-blue-600" />
                  Performance
                </h4>
                <div className="space-y-4">
                  <MetricBar
                    label="Queries per Second"
                    value={selectedMetrics.queries_per_second}
                    max={1000}
                    unit="qps"
                    icon={<Zap className="w-4 h-4" />}
                  />
                  <MetricBar
                    label="Avg Query Time"
                    value={selectedMetrics.avg_query_time}
                    max={1000}
                    unit="ms"
                    icon={<Clock className="w-4 h-4" />}
                    reverseColor
                  />
                  <MetricBar
                    label="Active Connections"
                    value={selectedMetrics.connections_active}
                    max={selectedMetrics.connections_max}
                    unit={`/ ${selectedMetrics.connections_max}`}
                    icon={<Users className="w-4 h-4" />}
                    reverseColor
                  />
                  <div className="pt-2 border-t border-gray-100">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 flex items-center gap-2">
                        <AlertTriangle className="w-4 h-4 text-yellow-600" />
                        Slow Queries (24h)
                      </span>
                      <span className="font-semibold text-gray-900">
                        {selectedMetrics.slow_queries_count.toLocaleString()}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Resource Usage */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                  <Server className="w-5 h-5 text-blue-600" />
                  Resource Usage
                </h4>
                <div className="space-y-4">
                  <MetricBar
                    label="CPU Usage"
                    value={selectedMetrics.cpu_usage}
                    max={100}
                    unit="%"
                    icon={<Cpu className="w-4 h-4" />}
                    reverseColor
                  />
                  <MetricBar
                    label="Memory Usage"
                    value={selectedMetrics.memory_usage}
                    max={100}
                    unit="%"
                    icon={<MemoryStick className="w-4 h-4" />}
                    reverseColor
                  />
                  <div className="grid grid-cols-2 gap-4 pt-2 border-t border-gray-100">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 flex items-center gap-2">
                        <TrendingUp className="w-4 h-4 text-blue-600" />
                        Disk Read
                      </span>
                      <span className="font-semibold text-gray-900">
                        {formatBytesPerSec(selectedMetrics.disk_io_read)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 flex items-center gap-2">
                        <TrendingDown className="w-4 h-4 text-purple-600" />
                        Disk Write
                      </span>
                      <span className="font-semibold text-gray-900">
                        {formatBytesPerSec(selectedMetrics.disk_io_write)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Storage */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                  <HardDrive className="w-5 h-5 text-blue-600" />
                  Storage
                </h4>
                <div className="space-y-4">
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm text-gray-600">Disk Usage</span>
                      <span className="text-sm font-semibold text-gray-900">
                        {formatBytes(selectedMetrics.size_used)} / {formatBytes(selectedMetrics.size_total)}
                      </span>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <div
                        className={clsx(
                          'h-2 rounded-full transition-all',
                          getPerformanceColor(
                            (selectedMetrics.size_used / selectedMetrics.size_total) * 100,
                            true
                          ) === 'text-green-600'
                            ? 'bg-green-600'
                            : getPerformanceColor(
                                (selectedMetrics.size_used / selectedMetrics.size_total) * 100,
                                true
                              ) === 'text-yellow-600'
                            ? 'bg-yellow-600'
                            : 'bg-red-600'
                        )}
                        style={{
                          width: `${(selectedMetrics.size_used / selectedMetrics.size_total) * 100}%`,
                        }}
                      />
                    </div>
                    <div className="flex items-center justify-between mt-1 text-xs text-gray-500">
                      <span>Available: {formatBytes(selectedMetrics.size_available)}</span>
                      <span>
                        {((selectedMetrics.size_used / selectedMetrics.size_total) * 100).toFixed(1)}% used
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Backup Info */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                  <Database className="w-5 h-5 text-blue-600" />
                  Backup Status
                </h4>
                <div className="space-y-3">
                  {selectedMetrics.last_backup_at && (
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 flex items-center gap-2">
                        <Clock className="w-4 h-4" />
                        Last Backup
                      </span>
                      <span className="font-semibold text-gray-900">
                        {formatUptime(
                          Math.floor((Date.now() - selectedMetrics.last_backup_at.getTime()) / 1000)
                        )}{' '}
                        ago
                      </span>
                    </div>
                  )}
                  {selectedMetrics.last_backup_size && (
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 flex items-center gap-2">
                        <HardDrive className="w-4 h-4" />
                        Last Backup Size
                      </span>
                      <span className="font-semibold text-gray-900">
                        {formatBytes(selectedMetrics.last_backup_size)}
                      </span>
                    </div>
                  )}
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 flex items-center gap-2">
                      <FileText className="w-4 h-4" />
                      Backups (24h)
                    </span>
                    <span className="font-semibold text-gray-900">{selectedMetrics.backup_count_24h}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 flex items-center gap-2">
                      <FileText className="w-4 h-4" />
                      Backups (7d)
                    </span>
                    <span className="font-semibold text-gray-900">{selectedMetrics.backup_count_7d}</span>
                  </div>
                </div>
              </div>

              {/* Replication Info */}
              {selectedMetrics.replication_status && selectedMetrics.replication_status !== 'standalone' && (
                <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                  <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                    <Server className="w-5 h-5 text-blue-600" />
                    Replication
                  </h4>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600">Role</span>
                      <span className="font-semibold text-gray-900 capitalize">
                        {selectedMetrics.replication_status}
                      </span>
                    </div>
                    {selectedMetrics.replication_lag !== undefined && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-gray-600">Replication Lag</span>
                        <span
                          className={clsx(
                            'font-semibold',
                            selectedMetrics.replication_lag < 100
                              ? 'text-green-600'
                              : selectedMetrics.replication_lag < 500
                              ? 'text-yellow-600'
                              : 'text-red-600'
                          )}
                        >
                          {selectedMetrics.replication_lag}ms
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
              <Database className="w-16 h-16 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">No Database Selected</h3>
              <p className="text-gray-600">Select a database from the list to view metrics</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

interface MetricBarProps {
  label: string
  value: number
  max: number
  unit: string
  icon: React.ReactNode
  reverseColor?: boolean
}

function MetricBar({ label, value, max, unit, icon, reverseColor = false }: MetricBarProps) {
  const percentage = (value / max) * 100
  const displayPercentage = Math.min(percentage, 100)

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-600 flex items-center gap-2">
          {icon}
          {label}
        </span>
        <span className="text-sm font-semibold text-gray-900">
          {value.toFixed(reverseColor && value < 1 ? 2 : 0)} {unit}
        </span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={clsx(
            'h-2 rounded-full transition-all',
            getPerformanceColor(displayPercentage, reverseColor) === 'text-green-600'
              ? 'bg-green-600'
              : getPerformanceColor(displayPercentage, reverseColor) === 'text-yellow-600'
              ? 'bg-yellow-600'
              : 'bg-red-600'
          )}
          style={{ width: `${displayPercentage}%` }}
        />
      </div>
    </div>
  )
}

// Sample data generator for testing
export function generateSampleMetrics(count: number = 5): DatabaseMetrics[] {
  const types: DatabaseMetrics['database_type'][] = ['postgres', 'mysql', 'mongodb', 'redis', 'mariadb']
  const statuses: DatabaseMetrics['status'][] = ['healthy', 'warning', 'critical', 'offline']
  const replicationStatuses: DatabaseMetrics['replication_status'][] = ['master', 'replica', 'standalone']

  return Array.from({ length: count }, (_, i) => {
    const type = types[i % types.length]
    const sizeTotal = Math.floor(Math.random() * 1000000000000) + 10000000000 // 10GB - 1TB
    const sizeUsed = Math.floor(sizeTotal * (0.3 + Math.random() * 0.6)) // 30-90% used
    const connectionsMax = Math.floor(Math.random() * 200) + 100
    const status = statuses[Math.floor(Math.random() * statuses.length)]

    return {
      database_id: `db-${i + 1}`,
      database_name: `${type}_${['prod', 'staging', 'dev', 'test'][i % 4]}`,
      database_type: type,
      status,
      queries_per_second: Math.random() * 500,
      avg_query_time: Math.random() * 200,
      slow_queries_count: Math.floor(Math.random() * 100),
      connections_active: Math.floor(Math.random() * connectionsMax * 0.7),
      connections_max: connectionsMax,
      size_total: sizeTotal,
      size_used: sizeUsed,
      size_available: sizeTotal - sizeUsed,
      tables_count: Math.floor(Math.random() * 500) + 50,
      rows_count: Math.floor(Math.random() * 100000000) + 10000,
      indexes_count: Math.floor(Math.random() * 1000) + 100,
      cpu_usage: Math.random() * 100,
      memory_usage: Math.random() * 100,
      disk_io_read: Math.random() * 100000000, // 0-100MB/s
      disk_io_write: Math.random() * 50000000, // 0-50MB/s
      last_backup_at: new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000),
      last_backup_size: Math.floor(sizeUsed * 0.8),
      backup_count_24h: Math.floor(Math.random() * 5),
      backup_count_7d: Math.floor(Math.random() * 30) + 5,
      uptime: Math.floor(Math.random() * 30 * 24 * 60 * 60), // 0-30 days
      version: `${Math.floor(Math.random() * 5) + 10}.${Math.floor(Math.random() * 10)}.${Math.floor(
        Math.random() * 20
      )}`,
      replication_status: replicationStatuses[Math.floor(Math.random() * replicationStatuses.length)],
      replication_lag: Math.random() * 1000,
      timestamp: new Date(),
    }
  })
}
