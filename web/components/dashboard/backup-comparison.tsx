'use client'

import React from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import {
  Database,
  CheckCircle2,
  XCircle,
  Clock,
  HardDrive,
  Calendar,
  Trash2,
  Download,
  BarChart3,
  ArrowUpDown,
  FileText,
  Server,
  AlertCircle,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface BackupData {
  id: string
  database_id: string
  database_name: string
  status: 'completed' | 'failed' | 'in_progress'
  size: number // bytes
  compressed_size?: number // bytes
  created_at: Date
  duration: number // milliseconds
  type: 'full' | 'incremental' | 'differential'
  retention_days: number
  storage_location: string
  checksum?: string
  metadata?: {
    tables_count?: number
    rows_count?: number
    indexes_count?: number
    compression_ratio?: number
    encryption?: boolean
    [key: string]: any
  }
}

interface BackupComparisonProps {
  availableBackups: BackupData[]
  maxComparisons?: number
  className?: string
  onExport?: (backups: BackupData[]) => void
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

const getStatusColor = (status: BackupData['status']) => {
  switch (status) {
    case 'completed':
      return 'text-green-600 bg-green-100'
    case 'failed':
      return 'text-red-600 bg-red-100'
    case 'in_progress':
      return 'text-blue-600 bg-blue-100'
  }
}

const getTypeColor = (type: BackupData['type']) => {
  switch (type) {
    case 'full':
      return 'text-purple-600 bg-purple-100'
    case 'incremental':
      return 'text-blue-600 bg-blue-100'
    case 'differential':
      return 'text-indigo-600 bg-indigo-100'
  }
}

export function BackupComparison({
  availableBackups,
  maxComparisons = 3,
  className,
  onExport,
}: BackupComparisonProps) {
  const [selectedBackups, setSelectedBackups] = React.useState<Set<string>>(new Set())
  const [sortBy, setSortBy] = React.useState<'date' | 'size' | 'duration'>('date')
  const [sortOrder, setSortOrder] = React.useState<'asc' | 'desc'>('desc')

  const toggleBackupSelection = (backupId: string) => {
    const newSelection = new Set(selectedBackups)
    if (newSelection.has(backupId)) {
      newSelection.delete(backupId)
    } else {
      if (newSelection.size >= maxComparisons) {
        // Remove the oldest selection
        const firstItem = newSelection.values().next().value
        if (firstItem !== undefined) {
          newSelection.delete(firstItem)
        }
      }
      newSelection.add(backupId)
    }
    setSelectedBackups(newSelection)
  }

  const selectedBackupData = React.useMemo(() => {
    return availableBackups.filter((b) => selectedBackups.has(b.id))
  }, [availableBackups, selectedBackups])

  const sortedAvailableBackups = React.useMemo(() => {
    const sorted = [...availableBackups]
    sorted.sort((a, b) => {
      let comparison = 0
      switch (sortBy) {
        case 'date':
          comparison = a.created_at.getTime() - b.created_at.getTime()
          break
        case 'size':
          comparison = a.size - b.size
          break
        case 'duration':
          comparison = a.duration - b.duration
          break
      }
      return sortOrder === 'asc' ? comparison : -comparison
    })
    return sorted
  }, [availableBackups, sortBy, sortOrder])

  const handleSort = (field: 'date' | 'size' | 'duration') => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(field)
      setSortOrder('desc')
    }
  }

  const handleExport = () => {
    if (onExport && selectedBackupData.length > 0) {
      onExport(selectedBackupData)
    }
  }

  const comparisonStats = React.useMemo(() => {
    if (selectedBackupData.length === 0) return null

    const totalSize = selectedBackupData.reduce((sum, b) => sum + b.size, 0)
    const avgSize = totalSize / selectedBackupData.length
    const totalDuration = selectedBackupData.reduce((sum, b) => sum + b.duration, 0)
    const avgDuration = totalDuration / selectedBackupData.length
    const successRate =
      (selectedBackupData.filter((b) => b.status === 'completed').length /
        selectedBackupData.length) *
      100

    return {
      totalSize,
      avgSize,
      totalDuration,
      avgDuration,
      successRate,
      count: selectedBackupData.length,
    }
  }, [selectedBackupData])

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Header */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">Backup Comparison</h2>
            <p className="text-sm text-gray-600 mt-1">
              Select up to {maxComparisons} backups to compare side-by-side
            </p>
          </div>
          {selectedBackupData.length > 0 && (
            <div className="flex items-center gap-2">
              <button
                onClick={() => setSelectedBackups(new Set())}
                className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
              >
                <Trash2 className="w-4 h-4" />
                Clear Selection
              </button>
              {onExport && (
                <button
                  onClick={handleExport}
                  className="px-4 py-2 text-sm text-white bg-blue-600 rounded-md hover:bg-blue-700 flex items-center gap-2"
                >
                  <Download className="w-4 h-4" />
                  Export Comparison
                </button>
              )}
            </div>
          )}
        </div>

        {/* Comparison Stats */}
        {comparisonStats && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4 p-4 bg-gray-50 rounded-lg">
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{comparisonStats.count}</div>
              <div className="text-xs text-gray-600">Selected</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {formatBytes(comparisonStats.totalSize)}
              </div>
              <div className="text-xs text-gray-600">Total Size</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {formatBytes(comparisonStats.avgSize)}
              </div>
              <div className="text-xs text-gray-600">Avg Size</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {formatDuration(comparisonStats.avgDuration)}
              </div>
              <div className="text-xs text-gray-600">Avg Duration</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {comparisonStats.successRate.toFixed(0)}%
              </div>
              <div className="text-xs text-gray-600">Success Rate</div>
            </div>
          </div>
        )}
      </div>

      {/* Backup Selection List */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">Available Backups</h3>
            <div className="flex items-center gap-4">
              <span className="text-sm text-gray-600">
                {selectedBackups.size} of {maxComparisons} selected
              </span>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-600">Sort by:</span>
                <button
                  onClick={() => handleSort('date')}
                  className={clsx(
                    'px-3 py-1 text-sm rounded-md flex items-center gap-1',
                    sortBy === 'date'
                      ? 'bg-blue-100 text-blue-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  )}
                >
                  <Calendar className="w-4 h-4" />
                  Date
                  {sortBy === 'date' && <ArrowUpDown className="w-3 h-3" />}
                </button>
                <button
                  onClick={() => handleSort('size')}
                  className={clsx(
                    'px-3 py-1 text-sm rounded-md flex items-center gap-1',
                    sortBy === 'size'
                      ? 'bg-blue-100 text-blue-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  )}
                >
                  <HardDrive className="w-4 h-4" />
                  Size
                  {sortBy === 'size' && <ArrowUpDown className="w-3 h-3" />}
                </button>
                <button
                  onClick={() => handleSort('duration')}
                  className={clsx(
                    'px-3 py-1 text-sm rounded-md flex items-center gap-1',
                    sortBy === 'duration'
                      ? 'bg-blue-100 text-blue-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  )}
                >
                  <Clock className="w-4 h-4" />
                  Duration
                  {sortBy === 'duration' && <ArrowUpDown className="w-3 h-3" />}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="divide-y divide-gray-100 max-h-96 overflow-y-auto">
          {sortedAvailableBackups.map((backup) => (
            <label
              key={backup.id}
              className={clsx(
                'flex items-center gap-4 px-6 py-4 hover:bg-gray-50 cursor-pointer transition-colors',
                selectedBackups.has(backup.id) && 'bg-blue-50'
              )}
            >
              <input
                type="checkbox"
                checked={selectedBackups.has(backup.id)}
                onChange={() => toggleBackupSelection(backup.id)}
                className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Database className="w-4 h-4 text-gray-500" />
                  <span className="font-medium text-gray-900">{backup.database_name}</span>
                  <span
                    className={clsx(
                      'px-2 py-0.5 text-xs font-medium rounded-full',
                      getStatusColor(backup.status)
                    )}
                  >
                    {backup.status}
                  </span>
                  <span
                    className={clsx(
                      'px-2 py-0.5 text-xs font-medium rounded-full',
                      getTypeColor(backup.type)
                    )}
                  >
                    {backup.type}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-xs text-gray-500">
                  <span className="flex items-center gap-1">
                    <HardDrive className="w-3 h-3" />
                    {formatBytes(backup.size)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {formatDuration(backup.duration)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Calendar className="w-3 h-3" />
                    {format(backup.created_at, 'MMM d, yyyy HH:mm')}
                  </span>
                </div>
              </div>
            </label>
          ))}
        </div>
      </div>

      {/* Comparison Table */}
      {selectedBackupData.length > 0 && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <BarChart3 className="w-5 h-5" />
              Detailed Comparison
            </h3>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Attribute
                  </th>
                  {selectedBackupData.map((backup) => (
                    <th
                      key={backup.id}
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                    >
                      {backup.database_name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {/* Backup ID */}
                <ComparisonRow
                  label="Backup ID"
                  icon={<FileText className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.id.substring(0, 8) + '...',
                    display: <span className="font-mono text-xs">{b.id.substring(0, 8)}...</span>,
                  }))}
                />

                {/* Status */}
                <ComparisonRow
                  label="Status"
                  icon={<CheckCircle2 className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.status,
                    display: (
                      <span
                        className={clsx(
                          'px-2 py-1 text-xs font-medium rounded-full inline-flex items-center gap-1',
                          getStatusColor(b.status)
                        )}
                      >
                        {b.status === 'completed' && <CheckCircle2 className="w-3 h-3" />}
                        {b.status === 'failed' && <XCircle className="w-3 h-3" />}
                        {b.status === 'in_progress' && <Clock className="w-3 h-3" />}
                        {b.status}
                      </span>
                    ),
                  }))}
                />

                {/* Type */}
                <ComparisonRow
                  label="Type"
                  icon={<Database className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.type,
                    display: (
                      <span className={clsx('px-2 py-1 text-xs font-medium rounded-full', getTypeColor(b.type))}>
                        {b.type}
                      </span>
                    ),
                  }))}
                />

                {/* Size */}
                <ComparisonRow
                  label="Size"
                  icon={<HardDrive className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.size,
                    display: <span className="font-semibold">{formatBytes(b.size)}</span>,
                  }))}
                  highlightBest={(values) => Math.min(...(values.map((v) => v.value as number)))}
                />

                {/* Compressed Size */}
                {selectedBackupData.some((b) => b.compressed_size) && (
                  <ComparisonRow
                    label="Compressed Size"
                    icon={<HardDrive className="w-4 h-4" />}
                    values={selectedBackupData.map((b) => ({
                      value: b.compressed_size || 0,
                      display: b.compressed_size ? (
                        <span className="font-semibold">{formatBytes(b.compressed_size)}</span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      ),
                    }))}
                    highlightBest={(values) =>
                      Math.min(...values.map((v) => (v.value as number) || Infinity))
                    }
                  />
                )}

                {/* Duration */}
                <ComparisonRow
                  label="Duration"
                  icon={<Clock className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.duration,
                    display: <span className="font-semibold">{formatDuration(b.duration)}</span>,
                  }))}
                  highlightBest={(values) => Math.min(...(values.map((v) => v.value as number)))}
                />

                {/* Created At */}
                <ComparisonRow
                  label="Created"
                  icon={<Calendar className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.created_at.getTime(),
                    display: (
                      <div>
                        <div className="font-medium">{format(b.created_at, 'MMM d, yyyy')}</div>
                        <div className="text-xs text-gray-500">
                          {formatDistanceToNow(b.created_at, { addSuffix: true })}
                        </div>
                      </div>
                    ),
                  }))}
                />

                {/* Storage Location */}
                <ComparisonRow
                  label="Storage"
                  icon={<Server className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.storage_location,
                    display: <span className="text-xs font-mono">{b.storage_location}</span>,
                  }))}
                />

                {/* Retention Days */}
                <ComparisonRow
                  label="Retention"
                  icon={<Calendar className="w-4 h-4" />}
                  values={selectedBackupData.map((b) => ({
                    value: b.retention_days,
                    display: <span>{b.retention_days} days</span>,
                  }))}
                />

                {/* Metadata */}
                {selectedBackupData.some((b) => b.metadata?.tables_count) && (
                  <ComparisonRow
                    label="Tables"
                    icon={<Database className="w-4 h-4" />}
                    values={selectedBackupData.map((b) => ({
                      value: b.metadata?.tables_count || 0,
                      display: b.metadata?.tables_count ? (
                        <span>{b.metadata.tables_count.toLocaleString()}</span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      ),
                    }))}
                  />
                )}

                {selectedBackupData.some((b) => b.metadata?.rows_count) && (
                  <ComparisonRow
                    label="Rows"
                    icon={<Database className="w-4 h-4" />}
                    values={selectedBackupData.map((b) => ({
                      value: b.metadata?.rows_count || 0,
                      display: b.metadata?.rows_count ? (
                        <span>{b.metadata.rows_count.toLocaleString()}</span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      ),
                    }))}
                  />
                )}

                {selectedBackupData.some((b) => b.metadata?.compression_ratio) && (
                  <ComparisonRow
                    label="Compression"
                    icon={<BarChart3 className="w-4 h-4" />}
                    values={selectedBackupData.map((b) => ({
                      value: b.metadata?.compression_ratio || 0,
                      display: b.metadata?.compression_ratio ? (
                        <span>{(b.metadata.compression_ratio * 100).toFixed(1)}%</span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      ),
                    }))}
                    highlightBest={(values) => Math.max(...(values.map((v) => v.value as number)))}
                  />
                )}

                {selectedBackupData.some((b) => b.metadata?.encryption !== undefined) && (
                  <ComparisonRow
                    label="Encryption"
                    icon={<AlertCircle className="w-4 h-4" />}
                    values={selectedBackupData.map((b) => ({
                      value: b.metadata?.encryption ? 1 : 0,
                      display: b.metadata?.encryption !== undefined ? (
                        <span
                          className={clsx(
                            'px-2 py-1 text-xs font-medium rounded-full',
                            b.metadata.encryption
                              ? 'bg-green-100 text-green-700'
                              : 'bg-gray-100 text-gray-700'
                          )}
                        >
                          {b.metadata.encryption ? 'Enabled' : 'Disabled'}
                        </span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      ),
                    }))}
                  />
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {selectedBackupData.length === 0 && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
            <BarChart3 className="w-8 h-8 text-gray-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No Backups Selected</h3>
          <p className="text-gray-600">
            Select backups from the list above to see a detailed comparison
          </p>
        </div>
      )}
    </div>
  )
}

interface ComparisonRowProps {
  label: string
  icon: React.ReactNode
  values: Array<{ value: any; display: React.ReactNode }>
  highlightBest?: (values: Array<{ value: any }>) => any
}

function ComparisonRow({ label, icon, values, highlightBest }: ComparisonRowProps) {
  const bestValue = highlightBest ? highlightBest(values) : null

  return (
    <tr>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center gap-2 text-sm font-medium text-gray-900">
          {icon}
          {label}
        </div>
      </td>
      {values.map((item, index) => (
        <td
          key={index}
          className={clsx(
            'px-6 py-4 whitespace-nowrap text-sm text-gray-900',
            bestValue !== null && item.value === bestValue && 'bg-green-50 font-semibold'
          )}
        >
          {item.display}
        </td>
      ))}
    </tr>
  )
}

// Sample data generator for testing
export function generateSampleBackups(count: number = 10): BackupData[] {
  const databases = ['postgres_prod', 'mysql_app', 'mongodb_users', 'redis_cache', 'mariadb_legacy']
  const types: BackupData['type'][] = ['full', 'incremental', 'differential']
  const statuses: BackupData['status'][] = ['completed', 'failed', 'in_progress']
  const locations = ['s3://backups/', 'azure://storage/', 'gcs://backup-bucket/', 'local://var/backups/']

  return Array.from({ length: count }, (_, i) => {
    const size = Math.floor(Math.random() * 10000000000) + 1000000 // 1MB - 10GB
    const compressedSize = Math.floor(size * (0.3 + Math.random() * 0.4)) // 30-70% compression
    const duration = Math.floor(Math.random() * 3600000) + 10000 // 10s - 1h
    const type = types[Math.floor(Math.random() * types.length)]
    const status = statuses[Math.floor(Math.random() * statuses.length)]

    return {
      id: `backup-${i + 1}-${Math.random().toString(36).substring(7)}`,
      database_id: `db-${Math.floor(Math.random() * 5) + 1}`,
      database_name: databases[Math.floor(Math.random() * databases.length)],
      status,
      size,
      compressed_size: compressedSize,
      created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000), // Last 30 days
      duration,
      type,
      retention_days: [7, 14, 30, 60, 90][Math.floor(Math.random() * 5)],
      storage_location: locations[Math.floor(Math.random() * locations.length)] + `backup-${i + 1}`,
      checksum: `sha256:${Math.random().toString(36).substring(2, 15)}`,
      metadata: {
        tables_count: Math.floor(Math.random() * 100) + 10,
        rows_count: Math.floor(Math.random() * 10000000) + 1000,
        indexes_count: Math.floor(Math.random() * 200) + 20,
        compression_ratio: compressedSize / size,
        encryption: Math.random() > 0.3,
      },
    }
  }).sort((a, b) => b.created_at.getTime() - a.created_at.getTime())
}
