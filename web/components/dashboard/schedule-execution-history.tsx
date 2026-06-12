'use client'

import React from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import {
  Calendar,
  CheckCircle2,
  XCircle,
  Clock,
  AlertTriangle,
  PlayCircle,
  Database,
  TrendingUp,
  TrendingDown,
  Activity,
  BarChart3,
  Filter,
  Download,
  RefreshCw,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface ScheduleExecution {
  id: string
  schedule_id: string
  schedule_name: string
  database_id: string
  database_name: string
  status: 'success' | 'failed' | 'running' | 'skipped' | 'cancelled'
  started_at: Date
  completed_at?: Date
  duration?: number // milliseconds
  backup_id?: string
  backup_size?: number // bytes
  error_message?: string
  triggered_by: 'schedule' | 'manual' | 'retry'
  retry_count?: number
  metadata?: {
    tables_backed_up?: number
    rows_processed?: number
    compression_ratio?: number
    [key: string]: any
  }
}

interface ScheduleExecutionHistoryProps {
  executions: ScheduleExecution[]
  scheduleId?: string // Optional filter by schedule
  onRetry?: (executionId: string) => void
  onViewBackup?: (backupId: string) => void
  onExport?: () => void
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
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  } else {
    return `${seconds}s`
  }
}

const getStatusConfig = (status: ScheduleExecution['status']) => {
  switch (status) {
    case 'success':
      return {
        icon: CheckCircle2,
        color: 'text-green-600',
        bg: 'bg-green-100',
        border: 'border-green-200',
        label: 'Success',
      }
    case 'failed':
      return {
        icon: XCircle,
        color: 'text-red-600',
        bg: 'bg-red-100',
        border: 'border-red-200',
        label: 'Failed',
      }
    case 'running':
      return {
        icon: Activity,
        color: 'text-blue-600',
        bg: 'bg-blue-100',
        border: 'border-blue-200',
        label: 'Running',
      }
    case 'skipped':
      return {
        icon: AlertTriangle,
        color: 'text-yellow-600',
        bg: 'bg-yellow-100',
        border: 'border-yellow-200',
        label: 'Skipped',
      }
    case 'cancelled':
      return {
        icon: XCircle,
        color: 'text-gray-600',
        bg: 'bg-gray-100',
        border: 'border-gray-200',
        label: 'Cancelled',
      }
  }
}

const getTriggerConfig = (trigger: ScheduleExecution['triggered_by']) => {
  switch (trigger) {
    case 'schedule':
      return { icon: Calendar, color: 'text-blue-600', label: 'Scheduled' }
    case 'manual':
      return { icon: PlayCircle, color: 'text-purple-600', label: 'Manual' }
    case 'retry':
      return { icon: RefreshCw, color: 'text-orange-600', label: 'Retry' }
  }
}

export function ScheduleExecutionHistory({
  executions,
  scheduleId,
  onRetry,
  onViewBackup,
  onExport,
  className,
}: ScheduleExecutionHistoryProps) {
  const [statusFilter, setStatusFilter] = React.useState<string>('all')
  const [scheduleFilter, setScheduleFilter] = React.useState<string>(scheduleId || 'all')
  const [triggerFilter, setTriggerFilter] = React.useState<string>('all')
  const [sortBy, setSortBy] = React.useState<'date' | 'duration' | 'size'>('date')
  const [sortOrder, setSortOrder] = React.useState<'asc' | 'desc'>('desc')

  const filteredExecutions = React.useMemo(() => {
    let filtered = executions

    if (statusFilter !== 'all') {
      filtered = filtered.filter((e) => e.status === statusFilter)
    }

    if (scheduleFilter !== 'all') {
      filtered = filtered.filter((e) => e.schedule_id === scheduleFilter)
    }

    if (triggerFilter !== 'all') {
      filtered = filtered.filter((e) => e.triggered_by === triggerFilter)
    }

    // Sort
    const sorted = [...filtered]
    sorted.sort((a, b) => {
      let comparison = 0
      switch (sortBy) {
        case 'date':
          comparison = a.started_at.getTime() - b.started_at.getTime()
          break
        case 'duration':
          comparison = (a.duration || 0) - (b.duration || 0)
          break
        case 'size':
          comparison = (a.backup_size || 0) - (b.backup_size || 0)
          break
      }
      return sortOrder === 'asc' ? comparison : -comparison
    })

    return sorted
  }, [executions, statusFilter, scheduleFilter, triggerFilter, sortBy, sortOrder])

  const stats = React.useMemo(() => {
    const total = executions.length
    const success = executions.filter((e) => e.status === 'success').length
    const failed = executions.filter((e) => e.status === 'failed').length
    const running = executions.filter((e) => e.status === 'running').length
    const successRate = total > 0 ? (success / total) * 100 : 0
    const avgDuration =
      executions.filter((e) => e.duration).reduce((sum, e) => sum + (e.duration || 0), 0) /
      (executions.filter((e) => e.duration).length || 1)
    const totalSize = executions.reduce((sum, e) => sum + (e.backup_size || 0), 0)

    return {
      total,
      success,
      failed,
      running,
      successRate,
      avgDuration,
      totalSize,
    }
  }, [executions])

  const uniqueSchedules = React.useMemo(() => {
    const schedules = new Map<string, string>()
    executions.forEach((e) => {
      schedules.set(e.schedule_id, e.schedule_name)
    })
    return Array.from(schedules.entries())
  }, [executions])

  const handleSort = (field: 'date' | 'duration' | 'size') => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(field)
      setSortOrder('desc')
    }
  }

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Executions</span>
            <Activity className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{stats.total}</div>
          <div className="text-xs text-gray-500 mt-1">All time</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Success Rate</span>
            <TrendingUp className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-2xl font-bold text-green-600">{stats.successRate.toFixed(1)}%</div>
          <div className="text-xs text-gray-500 mt-1">
            {stats.success} successful, {stats.failed} failed
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Avg Duration</span>
            <Clock className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{formatDuration(stats.avgDuration)}</div>
          <div className="text-xs text-gray-500 mt-1">Per execution</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Data</span>
            <Database className="w-5 h-5 text-purple-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{formatBytes(stats.totalSize)}</div>
          <div className="text-xs text-gray-500 mt-1">Backed up</div>
        </div>
      </div>

      {/* Filters and Actions */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
            <Filter className="w-5 h-5" />
            Execution History
          </h3>
          <div className="flex items-center gap-2">
            {onExport && (
              <button
                onClick={onExport}
                className="px-3 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
              >
                <Download className="w-4 h-4" />
                Export
              </button>
            )}
          </div>
        </div>

        <div className="flex flex-wrap gap-4">
          {/* Status Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Status:</label>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All</option>
              <option value="success">Success</option>
              <option value="failed">Failed</option>
              <option value="running">Running</option>
              <option value="skipped">Skipped</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </div>

          {/* Schedule Filter */}
          {!scheduleId && uniqueSchedules.length > 1 && (
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-700">Schedule:</label>
              <select
                value={scheduleFilter}
                onChange={(e) => setScheduleFilter(e.target.value)}
                className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Schedules</option>
                {uniqueSchedules.map(([id, name]) => (
                  <option key={id} value={id}>
                    {name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Trigger Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Trigger:</label>
            <select
              value={triggerFilter}
              onChange={(e) => setTriggerFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All</option>
              <option value="schedule">Scheduled</option>
              <option value="manual">Manual</option>
              <option value="retry">Retry</option>
            </select>
          </div>

          {/* Sort Options */}
          <div className="flex items-center gap-2 ml-auto">
            <span className="text-sm text-gray-600">Sort by:</span>
            <button
              onClick={() => handleSort('date')}
              className={clsx(
                'px-3 py-1 text-sm rounded-md',
                sortBy === 'date' ? 'bg-blue-100 text-blue-700' : 'text-gray-700 hover:bg-gray-100'
              )}
            >
              Date {sortBy === 'date' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              onClick={() => handleSort('duration')}
              className={clsx(
                'px-3 py-1 text-sm rounded-md',
                sortBy === 'duration' ? 'bg-blue-100 text-blue-700' : 'text-gray-700 hover:bg-gray-100'
              )}
            >
              Duration {sortBy === 'duration' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              onClick={() => handleSort('size')}
              className={clsx(
                'px-3 py-1 text-sm rounded-md',
                sortBy === 'size' ? 'bg-blue-100 text-blue-700' : 'text-gray-700 hover:bg-gray-100'
              )}
            >
              Size {sortBy === 'size' && (sortOrder === 'asc' ? '↑' : '↓')}
            </button>
          </div>
        </div>
      </div>

      {/* Execution List */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="divide-y divide-gray-100">
          {filteredExecutions.length === 0 ? (
            <div className="px-6 py-12 text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
                <Calendar className="w-8 h-8 text-gray-400" />
              </div>
              <p className="text-gray-500">No execution history found</p>
            </div>
          ) : (
            filteredExecutions.map((execution) => (
              <ExecutionHistoryItem
                key={execution.id}
                execution={execution}
                onRetry={onRetry}
                onViewBackup={onViewBackup}
              />
            ))
          )}
        </div>
      </div>
    </div>
  )
}

interface ExecutionHistoryItemProps {
  execution: ScheduleExecution
  onRetry?: (executionId: string) => void
  onViewBackup?: (backupId: string) => void
}

function ExecutionHistoryItem({ execution, onRetry, onViewBackup }: ExecutionHistoryItemProps) {
  const statusConfig = getStatusConfig(execution.status)
  const triggerConfig = getTriggerConfig(execution.triggered_by)
  const StatusIcon = statusConfig.icon
  const TriggerIcon = triggerConfig.icon

  return (
    <div className="px-6 py-4 hover:bg-gray-50 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-center gap-2 mb-2">
            <h4 className="text-sm font-semibold text-gray-900">{execution.schedule_name}</h4>
            <span
              className={clsx(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border',
                statusConfig.color,
                statusConfig.bg,
                statusConfig.border
              )}
            >
              <StatusIcon className="w-3 h-3" />
              {statusConfig.label}
            </span>
            <span
              className={clsx(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100',
                triggerConfig.color
              )}
            >
              <TriggerIcon className="w-3 h-3" />
              {triggerConfig.label}
            </span>
            {execution.retry_count && execution.retry_count > 0 && (
              <span className="px-2 py-0.5 text-xs font-medium bg-orange-100 text-orange-700 rounded-full">
                Retry #{execution.retry_count}
              </span>
            )}
          </div>

          {/* Database Info */}
          <div className="flex items-center gap-1 text-sm text-gray-600 mb-2">
            <Database className="w-4 h-4" />
            <span>{execution.database_name}</span>
          </div>

          {/* Metadata */}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <Calendar className="w-3 h-3" />
              Started {formatDistanceToNow(execution.started_at, { addSuffix: true })}
            </span>
            {execution.duration && (
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                Duration: {formatDuration(execution.duration)}
              </span>
            )}
            {execution.backup_size && (
              <span className="flex items-center gap-1">
                <Database className="w-3 h-3" />
                Size: {formatBytes(execution.backup_size)}
              </span>
            )}
            {execution.metadata?.tables_backed_up && (
              <span className="flex items-center gap-1">
                <BarChart3 className="w-3 h-3" />
                {execution.metadata.tables_backed_up} tables
              </span>
            )}
            {execution.metadata?.compression_ratio && (
              <span className="flex items-center gap-1">
                <TrendingDown className="w-3 h-3" />
                {(execution.metadata.compression_ratio * 100).toFixed(0)}% compression
              </span>
            )}
          </div>

          {/* Error Message */}
          {execution.error_message && (
            <div className="mt-2 p-2 bg-red-50 border border-red-200 rounded text-xs text-red-700">
              <span className="font-medium">Error:</span> {execution.error_message}
            </div>
          )}

          {/* Completed At */}
          {execution.completed_at && (
            <div className="mt-2 text-xs text-gray-500">
              Completed at {format(execution.completed_at, 'MMM d, yyyy HH:mm:ss')}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {execution.backup_id && onViewBackup && (
            <button
              onClick={() => onViewBackup(execution.backup_id!)}
              className="px-3 py-1.5 text-xs text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-md border border-blue-200 transition-colors"
            >
              View Backup
            </button>
          )}
          {execution.status === 'failed' && onRetry && (
            <button
              onClick={() => onRetry(execution.id)}
              className="px-3 py-1.5 text-xs text-orange-700 bg-orange-50 hover:bg-orange-100 rounded-md border border-orange-200 transition-colors flex items-center gap-1"
            >
              <RefreshCw className="w-3 h-3" />
              Retry
            </button>
          )}
          <time className="text-xs text-gray-500 whitespace-nowrap">
            {format(execution.started_at, 'MMM d, HH:mm')}
          </time>
        </div>
      </div>
    </div>
  )
}

// Sample data generator for testing
export function generateSampleExecutions(count: number = 20): ScheduleExecution[] {
  const schedules = [
    { id: 'schedule-1', name: 'Daily Production Backup' },
    { id: 'schedule-2', name: 'Hourly Incremental' },
    { id: 'schedule-3', name: 'Weekly Full Backup' },
  ]
  const databases = [
    { id: 'db-1', name: 'postgres_prod' },
    { id: 'db-2', name: 'mysql_app' },
    { id: 'db-3', name: 'mongodb_users' },
  ]
  const statuses: ScheduleExecution['status'][] = ['success', 'failed', 'running', 'skipped', 'cancelled']
  const triggers: ScheduleExecution['triggered_by'][] = ['schedule', 'manual', 'retry']

  return Array.from({ length: count }, (_, i) => {
    const schedule = schedules[Math.floor(Math.random() * schedules.length)]
    const database = databases[Math.floor(Math.random() * databases.length)]
    const status = statuses[Math.floor(Math.random() * statuses.length)]
    const trigger = triggers[Math.floor(Math.random() * triggers.length)]
    const startedAt = new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000)
    const duration = status !== 'running' ? Math.floor(Math.random() * 3600000) + 10000 : undefined
    const completedAt = duration ? new Date(startedAt.getTime() + duration) : undefined

    return {
      id: `execution-${i + 1}`,
      schedule_id: schedule.id,
      schedule_name: schedule.name,
      database_id: database.id,
      database_name: database.name,
      status,
      started_at: startedAt,
      completed_at: completedAt,
      duration,
      backup_id: status === 'success' ? `backup-${i + 1}` : undefined,
      backup_size: status === 'success' ? Math.floor(Math.random() * 10000000000) + 1000000 : undefined,
      error_message:
        status === 'failed'
          ? [
              'Connection timeout to database',
              'Insufficient disk space',
              'Authentication failed',
              'Table lock timeout',
            ][Math.floor(Math.random() * 4)]
          : undefined,
      triggered_by: trigger,
      retry_count: trigger === 'retry' ? Math.floor(Math.random() * 3) + 1 : undefined,
      metadata: {
        tables_backed_up: Math.floor(Math.random() * 100) + 10,
        rows_processed: Math.floor(Math.random() * 10000000) + 1000,
        compression_ratio: 0.3 + Math.random() * 0.4,
      },
    }
  }).sort((a, b) => b.started_at.getTime() - a.started_at.getTime())
}
