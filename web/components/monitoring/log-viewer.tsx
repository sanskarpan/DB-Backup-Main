'use client'

import React from 'react'
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Info,
  XCircle,
  Search,
  Filter,
  Download,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Copy,
  Terminal,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface LogEntry {
  id: string
  timestamp: Date
  level: 'debug' | 'info' | 'warn' | 'error' | 'fatal'
  source: string
  message: string
  metadata?: Record<string, any>
  stackTrace?: string
}

interface LogViewerProps {
  logs: LogEntry[]
  onRefresh?: () => void
  onExport?: (logs: LogEntry[]) => void
  autoScroll?: boolean
  maxLogs?: number
  className?: string
}

const levelConfig = {
  debug: {
    icon: Terminal,
    color: 'text-gray-600',
    bg: 'bg-gray-100',
    border: 'border-gray-300',
    label: 'DEBUG',
  },
  info: {
    icon: Info,
    color: 'text-blue-600',
    bg: 'bg-blue-100',
    border: 'border-blue-300',
    label: 'INFO',
  },
  warn: {
    icon: AlertTriangle,
    color: 'text-yellow-600',
    bg: 'bg-yellow-100',
    border: 'border-yellow-300',
    label: 'WARN',
  },
  error: {
    icon: XCircle,
    color: 'text-red-600',
    bg: 'bg-red-100',
    border: 'border-red-300',
    label: 'ERROR',
  },
  fatal: {
    icon: AlertCircle,
    color: 'text-red-800',
    bg: 'bg-red-200',
    border: 'border-red-400',
    label: 'FATAL',
  },
}

export function LogViewer({
  logs,
  onRefresh,
  onExport,
  autoScroll = false,
  maxLogs = 1000,
  className,
}: LogViewerProps) {
  const [levelFilter, setLevelFilter] = React.useState<string>('all')
  const [sourceFilter, setSourceFilter] = React.useState<string>('all')
  const [searchQuery, setSearchQuery] = React.useState('')
  const [expandedLogs, setExpandedLogs] = React.useState<Set<string>>(new Set())
  const logContainerRef = React.useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new logs arrive
  React.useEffect(() => {
    if (autoScroll && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  const uniqueSources = React.useMemo(() => {
    const sources = new Set(logs.map((log) => log.source))
    return Array.from(sources).sort()
  }, [logs])

  const filteredLogs = React.useMemo(() => {
    let filtered = logs

    // Level filter
    if (levelFilter !== 'all') {
      filtered = filtered.filter((log) => log.level === levelFilter)
    }

    // Source filter
    if (sourceFilter !== 'all') {
      filtered = filtered.filter((log) => log.source === sourceFilter)
    }

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (log) =>
          log.message.toLowerCase().includes(query) ||
          log.source.toLowerCase().includes(query) ||
          (log.metadata && JSON.stringify(log.metadata).toLowerCase().includes(query))
      )
    }

    // Limit to maxLogs
    return filtered.slice(-maxLogs)
  }, [logs, levelFilter, sourceFilter, searchQuery, maxLogs])

  const logCounts = React.useMemo(() => {
    return {
      debug: logs.filter((l) => l.level === 'debug').length,
      info: logs.filter((l) => l.level === 'info').length,
      warn: logs.filter((l) => l.level === 'warn').length,
      error: logs.filter((l) => l.level === 'error').length,
      fatal: logs.filter((l) => l.level === 'fatal').length,
    }
  }, [logs])

  const toggleLogExpansion = (logId: string) => {
    const newExpanded = new Set(expandedLogs)
    if (newExpanded.has(logId)) {
      newExpanded.delete(logId)
    } else {
      newExpanded.add(logId)
    }
    setExpandedLogs(newExpanded)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  const handleExport = () => {
    if (onExport) {
      onExport(filteredLogs)
    }
  }

  const clearFilters = () => {
    setLevelFilter('all')
    setSourceFilter('all')
    setSearchQuery('')
  }

  return (
    <div className={clsx('bg-white rounded-lg shadow-sm border border-gray-200', className)}>
      {/* Header */}
      <div className="px-6 py-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">System Logs</h3>
            <p className="text-sm text-gray-600 mt-1">
              Showing {filteredLogs.length} of {logs.length} logs
            </p>
          </div>
          <div className="flex items-center gap-2">
            {onRefresh && (
              <button
                onClick={onRefresh}
                className="px-3 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
              >
                <RefreshCw className="w-4 h-4" />
                Refresh
              </button>
            )}
            {onExport && (
              <button
                onClick={handleExport}
                className="px-3 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
              >
                <Download className="w-4 h-4" />
                Export
              </button>
            )}
          </div>
        </div>

        {/* Log Level Stats */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-2 mb-4">
          {Object.entries(logCounts).map(([level, count]) => {
            const config = levelConfig[level as keyof typeof levelConfig]
            const Icon = config.icon
            return (
              <div
                key={level}
                className={clsx(
                  'px-3 py-2 rounded-lg border flex items-center justify-between cursor-pointer hover:opacity-80 transition-opacity',
                  levelFilter === level ? config.bg : 'bg-gray-50',
                  config.border
                )}
                onClick={() => setLevelFilter(levelFilter === level ? 'all' : level)}
              >
                <div className="flex items-center gap-2">
                  <Icon className={clsx('w-4 h-4', config.color)} />
                  <span className="text-xs font-medium text-gray-700">{config.label}</span>
                </div>
                <span className="text-sm font-bold text-gray-900">{count}</span>
              </div>
            )
          })}
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-4">
          {/* Search */}
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search logs..."
                className="w-full pl-10 pr-4 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          {/* Source Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Source:</label>
            <select
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Sources</option>
              {uniqueSources.map((source) => (
                <option key={source} value={source}>
                  {source}
                </option>
              ))}
            </select>
          </div>

          {/* Clear Filters */}
          {(levelFilter !== 'all' || sourceFilter !== 'all' || searchQuery) && (
            <button
              onClick={clearFilters}
              className="px-3 py-2 text-sm text-blue-600 hover:text-blue-700 font-medium"
            >
              Clear Filters
            </button>
          )}
        </div>
      </div>

      {/* Log Entries */}
      <div ref={logContainerRef} className="max-h-[600px] overflow-y-auto divide-y divide-gray-100">
        {filteredLogs.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
              <Terminal className="w-8 h-8 text-gray-400" />
            </div>
            <p className="text-gray-500">No logs found matching your criteria</p>
          </div>
        ) : (
          filteredLogs.map((log) => (
            <LogEntryItem
              key={log.id}
              log={log}
              isExpanded={expandedLogs.has(log.id)}
              onToggleExpand={() => toggleLogExpansion(log.id)}
              onCopy={copyToClipboard}
            />
          ))
        )}
      </div>
    </div>
  )
}

interface LogEntryItemProps {
  log: LogEntry
  isExpanded: boolean
  onToggleExpand: () => void
  onCopy: (text: string) => void
}

function LogEntryItem({ log, isExpanded, onToggleExpand, onCopy }: LogEntryItemProps) {
  const config = levelConfig[log.level]
  const Icon = config.icon
  const hasDetails = log.metadata || log.stackTrace

  return (
    <div className="px-6 py-3 hover:bg-gray-50 transition-colors font-mono text-xs">
      <div className="flex items-start gap-3">
        {/* Expand/Collapse Button */}
        {hasDetails && (
          <button
            onClick={onToggleExpand}
            className="flex-shrink-0 mt-1 text-gray-400 hover:text-gray-600 transition-colors"
          >
            {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </button>
        )}

        {/* Log Level Icon */}
        <div className="flex-shrink-0 mt-1">
          <span
            className={clsx(
              'inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium border',
              config.color,
              config.bg,
              config.border
            )}
          >
            <Icon className="w-3 h-3" />
            {config.label}
          </span>
        </div>

        {/* Log Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-baseline gap-3 mb-1">
            <time className="text-gray-500 whitespace-nowrap">
              {log.timestamp.toLocaleString()}
            </time>
            <span className="text-blue-600 font-medium">{log.source}</span>
          </div>

          {/* Message */}
          <div className="text-gray-900 break-words">{log.message}</div>

          {/* Expanded Details */}
          {isExpanded && (
            <div className="mt-3 space-y-3">
              {/* Metadata */}
              {log.metadata && Object.keys(log.metadata).length > 0 && (
                <div className="bg-gray-900 text-gray-100 rounded-lg p-3 overflow-x-auto">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-semibold text-gray-400">METADATA</span>
                    <button
                      onClick={() => onCopy(JSON.stringify(log.metadata, null, 2))}
                      className="text-gray-400 hover:text-gray-200 transition-colors"
                    >
                      <Copy className="w-3 h-3" />
                    </button>
                  </div>
                  <pre className="text-xs">{JSON.stringify(log.metadata, null, 2)}</pre>
                </div>
              )}

              {/* Stack Trace */}
              {log.stackTrace && (
                <div className="bg-red-50 border border-red-200 rounded-lg p-3 overflow-x-auto">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-semibold text-red-700">STACK TRACE</span>
                    <button
                      onClick={() => onCopy(log.stackTrace!)}
                      className="text-red-600 hover:text-red-800 transition-colors"
                    >
                      <Copy className="w-3 h-3" />
                    </button>
                  </div>
                  <pre className="text-xs text-red-900 whitespace-pre-wrap">{log.stackTrace}</pre>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Copy Button */}
        <button
          onClick={() => onCopy(log.message)}
          className="flex-shrink-0 mt-1 text-gray-400 hover:text-gray-600 transition-colors"
        >
          <Copy className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}

// Live Log Stream Component (for WebSocket streaming)
interface LiveLogStreamProps {
  onConnect?: () => void
  onDisconnect?: () => void
  isConnected?: boolean
  className?: string
}

export function LiveLogStream({ onConnect, onDisconnect, isConnected = false, className }: LiveLogStreamProps) {
  const [logs, setLogs] = React.useState<LogEntry[]>([])
  const [isPaused, setIsPaused] = React.useState(false)

  const addLog = React.useCallback(
    (log: LogEntry) => {
      if (!isPaused) {
        setLogs((prev) => [...prev.slice(-999), log])
      }
    },
    [isPaused]
  )

  const clearLogs = () => {
    setLogs([])
  }

  return (
    <div className={className}>
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className={clsx('w-3 h-3 rounded-full', isConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400')} />
          <span className="text-sm font-medium text-gray-700">
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsPaused(!isPaused)}
            className={clsx(
              'px-3 py-1.5 text-sm rounded-md border transition-colors',
              isPaused
                ? 'bg-green-50 text-green-700 border-green-300 hover:bg-green-100'
                : 'bg-gray-50 text-gray-700 border-gray-300 hover:bg-gray-100'
            )}
          >
            {isPaused ? 'Resume' : 'Pause'}
          </button>
          <button
            onClick={clearLogs}
            className="px-3 py-1.5 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Clear
          </button>
          {isConnected ? (
            <button
              onClick={onDisconnect}
              className="px-3 py-1.5 text-sm text-red-700 bg-red-50 border border-red-300 rounded-md hover:bg-red-100"
            >
              Disconnect
            </button>
          ) : (
            <button
              onClick={onConnect}
              className="px-3 py-1.5 text-sm text-green-700 bg-green-50 border border-green-300 rounded-md hover:bg-green-100"
            >
              Connect
            </button>
          )}
        </div>
      </div>

      <LogViewer logs={logs} autoScroll={!isPaused} maxLogs={1000} />
    </div>
  )
}

// Sample data generator for testing
export function generateSampleLogs(count: number = 50): LogEntry[] {
  const levels: LogEntry['level'][] = ['debug', 'info', 'warn', 'error', 'fatal']
  const sources = ['backup-engine', 'restore-engine', 'scheduler', 'api-server', 'database-driver', 'storage-manager']
  const messages = [
    'Starting backup operation',
    'Database connection established',
    'Compression started',
    'Upload to cloud storage initiated',
    'Backup completed successfully',
    'Failed to connect to database',
    'Timeout while waiting for response',
    'Invalid configuration detected',
    'Memory usage threshold exceeded',
    'Disk space running low',
  ]

  return Array.from({ length: count }, (_, i) => {
    const level = levels[Math.floor(Math.random() * levels.length)]
    const hasError = level === 'error' || level === 'fatal'

    return {
      id: `log-${i + 1}`,
      timestamp: new Date(Date.now() - Math.random() * 3600000),
      level,
      source: sources[Math.floor(Math.random() * sources.length)],
      message: messages[Math.floor(Math.random() * messages.length)],
      metadata:
        Math.random() > 0.5
          ? {
              database_id: `db-${Math.floor(Math.random() * 5) + 1}`,
              backup_id: `backup-${Math.floor(Math.random() * 100) + 1}`,
              duration: Math.floor(Math.random() * 60000),
              size: Math.floor(Math.random() * 10000000000),
            }
          : undefined,
      stackTrace: hasError && Math.random() > 0.5
        ? `Error: ${messages[Math.floor(Math.random() * messages.length)]}\n    at BackupEngine.run (engine.go:123)\n    at main (main.go:45)`
        : undefined,
    }
  }).sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
}
