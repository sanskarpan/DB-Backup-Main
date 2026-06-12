'use client'

import React from 'react'
import {
  Activity,
  Users,
  Clock,
  TrendingUp,
  TrendingDown,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Zap,
  Timer,
  Database,
  Link,
  Unlink,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface ConnectionPoolMetrics {
  pool_id: string
  database_id: string
  database_name: string
  database_type: 'postgres' | 'mysql' | 'mongodb' | 'redis' | 'mariadb'

  // Connection counts
  connections_total: number
  connections_active: number
  connections_idle: number
  connections_waiting: number
  connections_max: number

  // Pool configuration
  pool_size_min: number
  pool_size_max: number
  connection_timeout: number // milliseconds
  idle_timeout: number // milliseconds
  max_lifetime: number // milliseconds

  // Performance metrics
  wait_duration_avg: number // milliseconds
  wait_duration_max: number // milliseconds
  connection_duration_avg: number // milliseconds
  acquire_count: number // total acquisitions
  acquire_rate: number // per second

  // Health metrics
  errors_total: number
  errors_timeout: number
  errors_rejected: number
  success_rate: number // percentage

  // Utilization
  utilization: number // percentage (active / max)
  saturation: number // percentage (waiting / max)

  // Historical data (last 60 seconds)
  history?: ConnectionPoolHistory[]

  timestamp: Date
}

export interface ConnectionPoolHistory {
  timestamp: Date
  active: number
  idle: number
  waiting: number
}

interface ConnectionPoolMonitorProps {
  metrics: ConnectionPoolMetrics[]
  refreshInterval?: number // milliseconds
  onRefresh?: () => void
  className?: string
}

const formatDuration = (ms: number): string => {
  if (ms < 1000) {
    return `${ms.toFixed(0)}ms`
  } else {
    return `${(ms / 1000).toFixed(2)}s`
  }
}

const getUtilizationColor = (percentage: number) => {
  if (percentage < 60) return 'text-green-600'
  if (percentage < 80) return 'text-yellow-600'
  return 'text-red-600'
}

const getUtilizationBgColor = (percentage: number) => {
  if (percentage < 60) return 'bg-green-600'
  if (percentage < 80) return 'bg-yellow-600'
  return 'bg-red-600'
}

const getHealthStatus = (successRate: number, utilization: number) => {
  if (successRate >= 99 && utilization < 80) {
    return { status: 'healthy', color: 'text-green-600', bg: 'bg-green-100', icon: CheckCircle2 }
  } else if (successRate >= 95 && utilization < 90) {
    return { status: 'warning', color: 'text-yellow-600', bg: 'bg-yellow-100', icon: AlertCircle }
  } else {
    return { status: 'critical', color: 'text-red-600', bg: 'bg-red-100', icon: XCircle }
  }
}

export function ConnectionPoolMonitor({
  metrics,
  refreshInterval,
  onRefresh,
  className,
}: ConnectionPoolMonitorProps) {
  const [selectedPool, setSelectedPool] = React.useState<string | null>(
    metrics.length > 0 ? metrics[0].pool_id : null
  )

  // Auto-refresh
  React.useEffect(() => {
    if (refreshInterval && onRefresh) {
      const interval = setInterval(onRefresh, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [refreshInterval, onRefresh])

  const selectedMetrics = React.useMemo(() => {
    return metrics.find((m) => m.pool_id === selectedPool)
  }, [metrics, selectedPool])

  const overallStats = React.useMemo(() => {
    const totalPools = metrics.length
    const totalConnections = metrics.reduce((sum, m) => sum + m.connections_active, 0)
    const totalMaxConnections = metrics.reduce((sum, m) => sum + m.connections_max, 0)
    const avgUtilization =
      metrics.reduce((sum, m) => sum + m.utilization, 0) / (totalPools || 1)
    const poolsWithIssues = metrics.filter(
      (m) => m.success_rate < 95 || m.utilization > 80 || m.saturation > 10
    ).length

    return {
      totalPools,
      totalConnections,
      totalMaxConnections,
      avgUtilization,
      poolsWithIssues,
    }
  }, [metrics])

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Connection Pools</span>
            <Database className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{overallStats.totalPools}</div>
          <div className="text-xs text-gray-500 mt-1">Active pools</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Active Connections</span>
            <Activity className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{overallStats.totalConnections}</div>
          <div className="text-xs text-gray-500 mt-1">
            of {overallStats.totalMaxConnections} max
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Avg Utilization</span>
            <TrendingUp className="w-5 h-5 text-blue-600" />
          </div>
          <div
            className={clsx('text-2xl font-bold', getUtilizationColor(overallStats.avgUtilization))}
          >
            {overallStats.avgUtilization.toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500 mt-1">Across all pools</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Pools with Issues</span>
            <AlertCircle className="w-5 h-5 text-yellow-600" />
          </div>
          <div className="text-2xl font-bold text-yellow-600">{overallStats.poolsWithIssues}</div>
          <div className="text-xs text-gray-500 mt-1">
            {overallStats.poolsWithIssues > 0 ? 'Needs attention' : 'All healthy'}
          </div>
        </div>
      </div>

      {/* Pool List */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">Connection Pools</h3>
        </div>
        <div className="divide-y divide-gray-100">
          {metrics.map((pool) => {
            const health = getHealthStatus(pool.success_rate, pool.utilization)
            const HealthIcon = health.icon

            return (
              <button
                key={pool.pool_id}
                onClick={() => setSelectedPool(pool.pool_id)}
                className={clsx(
                  'w-full px-6 py-4 text-left hover:bg-gray-50 transition-colors',
                  selectedPool === pool.pool_id && 'bg-blue-50'
                )}
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Database className="w-5 h-5 text-gray-500" />
                    <span className="font-medium text-gray-900">{pool.database_name}</span>
                    <span className="text-xs text-gray-500">({pool.database_type})</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span
                      className={clsx(
                        'px-2 py-1 text-xs font-medium rounded-full inline-flex items-center gap-1',
                        health.color,
                        health.bg
                      )}
                    >
                      <HealthIcon className="w-3 h-3" />
                      {health.status}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-6 text-sm">
                  <div className="flex items-center gap-1 text-green-600">
                    <Link className="w-4 h-4" />
                    <span className="font-semibold">{pool.connections_active}</span>
                    <span className="text-gray-500">active</span>
                  </div>
                  <div className="flex items-center gap-1 text-blue-600">
                    <Activity className="w-4 h-4" />
                    <span className="font-semibold">{pool.connections_idle}</span>
                    <span className="text-gray-500">idle</span>
                  </div>
                  {pool.connections_waiting > 0 && (
                    <div className="flex items-center gap-1 text-yellow-600">
                      <Clock className="w-4 h-4" />
                      <span className="font-semibold">{pool.connections_waiting}</span>
                      <span className="text-gray-500">waiting</span>
                    </div>
                  )}
                  <div className="flex items-center gap-1 text-gray-600">
                    <span className="text-xs">Utilization:</span>
                    <span className={clsx('font-semibold', getUtilizationColor(pool.utilization))}>
                      {pool.utilization.toFixed(0)}%
                    </span>
                  </div>
                </div>

                {/* Progress bar */}
                <div className="mt-3 w-full bg-gray-200 rounded-full h-2">
                  <div className="relative h-2 rounded-full overflow-hidden">
                    <div
                      className="absolute top-0 left-0 h-full bg-green-500"
                      style={{
                        width: `${(pool.connections_active / pool.connections_max) * 100}%`,
                      }}
                    />
                    <div
                      className="absolute top-0 h-full bg-blue-400"
                      style={{
                        left: `${(pool.connections_active / pool.connections_max) * 100}%`,
                        width: `${(pool.connections_idle / pool.connections_max) * 100}%`,
                      }}
                    />
                    {pool.connections_waiting > 0 && (
                      <div
                        className="absolute top-0 h-full bg-yellow-500"
                        style={{
                          left: `${
                            ((pool.connections_active + pool.connections_idle) / pool.connections_max) *
                            100
                          }%`,
                          width: `${(pool.connections_waiting / pool.connections_max) * 100}%`,
                        }}
                      />
                    )}
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Pool Details */}
      {selectedMetrics && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Connection Stats */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Users className="w-5 h-5 text-blue-600" />
              Connection Statistics
            </h4>
            <div className="space-y-4">
              <ConnectionMetric
                label="Active Connections"
                value={selectedMetrics.connections_active}
                max={selectedMetrics.connections_max}
                icon={<Link className="w-4 h-4 text-green-600" />}
                color="green"
              />
              <ConnectionMetric
                label="Idle Connections"
                value={selectedMetrics.connections_idle}
                max={selectedMetrics.connections_max}
                icon={<Activity className="w-4 h-4 text-blue-600" />}
                color="blue"
              />
              <ConnectionMetric
                label="Waiting Connections"
                value={selectedMetrics.connections_waiting}
                max={selectedMetrics.connections_max}
                icon={<Clock className="w-4 h-4 text-yellow-600" />}
                color="yellow"
              />
              <div className="pt-3 border-t border-gray-200">
                <div className="flex items-center justify-between text-sm mb-2">
                  <span className="text-gray-600">Pool Configuration</span>
                </div>
                <div className="grid grid-cols-2 gap-3 text-xs">
                  <div className="bg-gray-50 rounded p-2">
                    <div className="text-gray-600">Min Size</div>
                    <div className="font-semibold text-gray-900">
                      {selectedMetrics.pool_size_min}
                    </div>
                  </div>
                  <div className="bg-gray-50 rounded p-2">
                    <div className="text-gray-600">Max Size</div>
                    <div className="font-semibold text-gray-900">
                      {selectedMetrics.pool_size_max}
                    </div>
                  </div>
                  <div className="bg-gray-50 rounded p-2">
                    <div className="text-gray-600">Conn Timeout</div>
                    <div className="font-semibold text-gray-900">
                      {formatDuration(selectedMetrics.connection_timeout)}
                    </div>
                  </div>
                  <div className="bg-gray-50 rounded p-2">
                    <div className="text-gray-600">Idle Timeout</div>
                    <div className="font-semibold text-gray-900">
                      {formatDuration(selectedMetrics.idle_timeout)}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Performance Metrics */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Zap className="w-5 h-5 text-blue-600" />
              Performance Metrics
            </h4>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Clock className="w-4 h-4" />
                  Avg Wait Time
                </span>
                <span className="text-sm font-semibold text-gray-900">
                  {formatDuration(selectedMetrics.wait_duration_avg)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Timer className="w-4 h-4" />
                  Max Wait Time
                </span>
                <span
                  className={clsx(
                    'text-sm font-semibold',
                    selectedMetrics.wait_duration_max < 100
                      ? 'text-green-600'
                      : selectedMetrics.wait_duration_max < 500
                      ? 'text-yellow-600'
                      : 'text-red-600'
                  )}
                >
                  {formatDuration(selectedMetrics.wait_duration_max)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Activity className="w-4 h-4" />
                  Avg Connection Time
                </span>
                <span className="text-sm font-semibold text-gray-900">
                  {formatDuration(selectedMetrics.connection_duration_avg)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <TrendingUp className="w-4 h-4" />
                  Acquire Rate
                </span>
                <span className="text-sm font-semibold text-gray-900">
                  {selectedMetrics.acquire_rate.toFixed(1)} /s
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Users className="w-4 h-4" />
                  Total Acquisitions
                </span>
                <span className="text-sm font-semibold text-gray-900">
                  {selectedMetrics.acquire_count.toLocaleString()}
                </span>
              </div>
              <div className="pt-3 border-t border-gray-200">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-gray-600">Success Rate</span>
                  <span
                    className={clsx(
                      'text-lg font-bold',
                      selectedMetrics.success_rate >= 99
                        ? 'text-green-600'
                        : selectedMetrics.success_rate >= 95
                        ? 'text-yellow-600'
                        : 'text-red-600'
                    )}
                  >
                    {selectedMetrics.success_rate.toFixed(2)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className={clsx(
                      'h-2 rounded-full',
                      selectedMetrics.success_rate >= 99
                        ? 'bg-green-600'
                        : selectedMetrics.success_rate >= 95
                        ? 'bg-yellow-600'
                        : 'bg-red-600'
                    )}
                    style={{ width: `${selectedMetrics.success_rate}%` }}
                  />
                </div>
              </div>
            </div>
          </div>

          {/* Error Statistics */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <AlertCircle className="w-5 h-5 text-red-600" />
              Error Statistics
            </h4>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <XCircle className="w-4 h-4" />
                  Total Errors
                </span>
                <span className="text-sm font-semibold text-red-600">
                  {selectedMetrics.errors_total}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Clock className="w-4 h-4" />
                  Timeout Errors
                </span>
                <span className="text-sm font-semibold text-yellow-600">
                  {selectedMetrics.errors_timeout}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 flex items-center gap-2">
                  <Unlink className="w-4 h-4" />
                  Rejected Connections
                </span>
                <span className="text-sm font-semibold text-orange-600">
                  {selectedMetrics.errors_rejected}
                </span>
              </div>
              {selectedMetrics.errors_total > 0 && (
                <div className="pt-3 border-t border-gray-200">
                  <div className="text-xs text-gray-600 mb-2">Error Distribution</div>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-200 rounded-full h-2">
                        <div
                          className="bg-yellow-600 h-2 rounded-full"
                          style={{
                            width: `${
                              (selectedMetrics.errors_timeout / selectedMetrics.errors_total) * 100
                            }%`,
                          }}
                        />
                      </div>
                      <span className="text-xs text-gray-600 w-16 text-right">
                        {((selectedMetrics.errors_timeout / selectedMetrics.errors_total) * 100).toFixed(
                          0
                        )}
                        %
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-200 rounded-full h-2">
                        <div
                          className="bg-orange-600 h-2 rounded-full"
                          style={{
                            width: `${
                              (selectedMetrics.errors_rejected / selectedMetrics.errors_total) * 100
                            }%`,
                          }}
                        />
                      </div>
                      <span className="text-xs text-gray-600 w-16 text-right">
                        {((selectedMetrics.errors_rejected / selectedMetrics.errors_total) * 100).toFixed(
                          0
                        )}
                        %
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Utilization & Saturation */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h4 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-blue-600" />
              Utilization & Saturation
            </h4>
            <div className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-gray-600">Pool Utilization</span>
                  <span
                    className={clsx(
                      'text-lg font-bold',
                      getUtilizationColor(selectedMetrics.utilization)
                    )}
                  >
                    {selectedMetrics.utilization.toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-3">
                  <div
                    className={clsx('h-3 rounded-full', getUtilizationBgColor(selectedMetrics.utilization))}
                    style={{ width: `${Math.min(selectedMetrics.utilization, 100)}%` }}
                  />
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {selectedMetrics.connections_active} active of {selectedMetrics.connections_max} max
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-gray-600">Queue Saturation</span>
                  <span
                    className={clsx(
                      'text-lg font-bold',
                      selectedMetrics.saturation < 5
                        ? 'text-green-600'
                        : selectedMetrics.saturation < 15
                        ? 'text-yellow-600'
                        : 'text-red-600'
                    )}
                  >
                    {selectedMetrics.saturation.toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-3">
                  <div
                    className={clsx(
                      'h-3 rounded-full',
                      selectedMetrics.saturation < 5
                        ? 'bg-green-600'
                        : selectedMetrics.saturation < 15
                        ? 'bg-yellow-600'
                        : 'bg-red-600'
                    )}
                    style={{ width: `${Math.min(selectedMetrics.saturation, 100)}%` }}
                  />
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  {selectedMetrics.connections_waiting} connections waiting
                </div>
              </div>

              <div className="pt-3 border-t border-gray-200">
                <div className="grid grid-cols-3 gap-3 text-center">
                  <div className="bg-green-50 rounded p-2">
                    <div className="text-2xl font-bold text-green-600">
                      {selectedMetrics.connections_active}
                    </div>
                    <div className="text-xs text-gray-600">Active</div>
                  </div>
                  <div className="bg-blue-50 rounded p-2">
                    <div className="text-2xl font-bold text-blue-600">
                      {selectedMetrics.connections_idle}
                    </div>
                    <div className="text-xs text-gray-600">Idle</div>
                  </div>
                  <div className="bg-yellow-50 rounded p-2">
                    <div className="text-2xl font-bold text-yellow-600">
                      {selectedMetrics.connections_waiting}
                    </div>
                    <div className="text-xs text-gray-600">Waiting</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

interface ConnectionMetricProps {
  label: string
  value: number
  max: number
  icon: React.ReactNode
  color: 'green' | 'blue' | 'yellow' | 'red'
}

function ConnectionMetric({ label, value, max, icon, color }: ConnectionMetricProps) {
  const percentage = (value / max) * 100

  const colorClasses = {
    green: 'bg-green-600',
    blue: 'bg-blue-600',
    yellow: 'bg-yellow-600',
    red: 'bg-red-600',
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-600 flex items-center gap-2">
          {icon}
          {label}
        </span>
        <span className="text-sm font-semibold text-gray-900">
          {value} / {max}
        </span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={clsx('h-2 rounded-full', colorClasses[color])}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  )
}

// Sample data generator for testing
export function generateSamplePoolMetrics(count: number = 5): ConnectionPoolMetrics[] {
  const types: ConnectionPoolMetrics['database_type'][] = ['postgres', 'mysql', 'mongodb', 'redis', 'mariadb']

  return Array.from({ length: count }, (_, i) => {
    const maxConnections = Math.floor(Math.random() * 100) + 50
    const activeConnections = Math.floor(Math.random() * maxConnections * 0.8)
    const idleConnections = Math.floor(Math.random() * (maxConnections - activeConnections))
    const waitingConnections = Math.random() > 0.7 ? Math.floor(Math.random() * 10) : 0
    const totalAcquires = Math.floor(Math.random() * 100000) + 10000
    const errors = Math.floor(Math.random() * totalAcquires * 0.05)

    return {
      pool_id: `pool-${i + 1}`,
      database_id: `db-${i + 1}`,
      database_name: `${types[i % types.length]}_${['prod', 'staging', 'dev'][i % 3]}`,
      database_type: types[i % types.length],
      connections_total: activeConnections + idleConnections,
      connections_active: activeConnections,
      connections_idle: idleConnections,
      connections_waiting: waitingConnections,
      connections_max: maxConnections,
      pool_size_min: 5,
      pool_size_max: maxConnections,
      connection_timeout: 30000,
      idle_timeout: 300000,
      max_lifetime: 1800000,
      wait_duration_avg: Math.random() * 100,
      wait_duration_max: Math.random() * 500,
      connection_duration_avg: Math.random() * 50,
      acquire_count: totalAcquires,
      acquire_rate: Math.random() * 100,
      errors_total: errors,
      errors_timeout: Math.floor(errors * 0.6),
      errors_rejected: Math.floor(errors * 0.4),
      success_rate: ((totalAcquires - errors) / totalAcquires) * 100,
      utilization: (activeConnections / maxConnections) * 100,
      saturation: (waitingConnections / maxConnections) * 100,
      timestamp: new Date(),
    }
  })
}
