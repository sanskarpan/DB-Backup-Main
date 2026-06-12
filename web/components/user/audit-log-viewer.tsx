'use client'

import React from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import {
  User,
  Shield,
  Key,
  Database,
  Settings,
  LogIn,
  LogOut,
  Edit,
  Trash2,
  Plus,
  Search,
  Filter,
  Download,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  Globe,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface AuditLogEntry {
  id: string
  timestamp: Date
  user_id: string
  user_email: string
  user_name?: string
  action: string
  resource_type: 'user' | 'database' | 'backup' | 'schedule' | 'settings' | 'auth' | 'api_key'
  resource_id?: string
  resource_name?: string
  action_type: 'create' | 'read' | 'update' | 'delete' | 'login' | 'logout' | 'failed_login' | 'permission_change'
  status: 'success' | 'failed' | 'denied'
  ip_address: string
  user_agent?: string
  changes?: {
    field: string
    old_value: any
    new_value: any
  }[]
  metadata?: Record<string, any>
  risk_level?: 'low' | 'medium' | 'high' | 'critical'
}

interface AuditLogViewerProps {
  logs: AuditLogEntry[]
  onRefresh?: () => void
  onExport?: (logs: AuditLogEntry[]) => void
  autoRefresh?: boolean
  refreshInterval?: number
  className?: string
}

const actionTypeConfig = {
  create: {
    icon: Plus,
    color: 'text-green-600',
    bg: 'bg-green-100',
    label: 'Created',
  },
  read: {
    icon: Search,
    color: 'text-blue-600',
    bg: 'bg-blue-100',
    label: 'Viewed',
  },
  update: {
    icon: Edit,
    color: 'text-yellow-600',
    bg: 'bg-yellow-100',
    label: 'Updated',
  },
  delete: {
    icon: Trash2,
    color: 'text-red-600',
    bg: 'bg-red-100',
    label: 'Deleted',
  },
  login: {
    icon: LogIn,
    color: 'text-green-600',
    bg: 'bg-green-100',
    label: 'Login',
  },
  logout: {
    icon: LogOut,
    color: 'text-gray-600',
    bg: 'bg-gray-100',
    label: 'Logout',
  },
  failed_login: {
    icon: XCircle,
    color: 'text-red-600',
    bg: 'bg-red-100',
    label: 'Failed Login',
  },
  permission_change: {
    icon: Shield,
    color: 'text-purple-600',
    bg: 'bg-purple-100',
    label: 'Permission Change',
  },
}

const resourceTypeConfig = {
  user: { icon: User, color: 'text-blue-600', label: 'User' },
  database: { icon: Database, color: 'text-green-600', label: 'Database' },
  backup: { icon: Database, color: 'text-purple-600', label: 'Backup' },
  schedule: { icon: Clock, color: 'text-orange-600', label: 'Schedule' },
  settings: { icon: Settings, color: 'text-gray-600', label: 'Settings' },
  auth: { icon: Shield, color: 'text-red-600', label: 'Authentication' },
  api_key: { icon: Key, color: 'text-yellow-600', label: 'API Key' },
}

const statusConfig = {
  success: {
    icon: CheckCircle2,
    color: 'text-green-600',
    bg: 'bg-green-100',
    label: 'Success',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-600',
    bg: 'bg-red-100',
    label: 'Failed',
  },
  denied: {
    icon: AlertTriangle,
    color: 'text-yellow-600',
    bg: 'bg-yellow-100',
    label: 'Denied',
  },
}

const riskLevelConfig = {
  low: { color: 'text-gray-600', bg: 'bg-gray-100', label: 'Low Risk' },
  medium: { color: 'text-yellow-600', bg: 'bg-yellow-100', label: 'Medium Risk' },
  high: { color: 'text-orange-600', bg: 'bg-orange-100', label: 'High Risk' },
  critical: { color: 'text-red-600', bg: 'bg-red-100', label: 'Critical Risk' },
}

export function AuditLogViewer({
  logs,
  onRefresh,
  onExport,
  autoRefresh = false,
  refreshInterval = 30000,
  className,
}: AuditLogViewerProps) {
  const [actionFilter, setActionFilter] = React.useState<string>('all')
  const [resourceFilter, setResourceFilter] = React.useState<string>('all')
  const [statusFilter, setStatusFilter] = React.useState<string>('all')
  const [userFilter, setUserFilter] = React.useState<string>('all')
  const [riskFilter, setRiskFilter] = React.useState<string>('all')
  const [searchQuery, setSearchQuery] = React.useState('')
  const [expandedLogs, setExpandedLogs] = React.useState<Set<string>>(new Set())
  const [dateRange, setDateRange] = React.useState<'today' | 'week' | 'month' | 'all'>('all')

  // Auto-refresh
  React.useEffect(() => {
    if (autoRefresh && onRefresh && refreshInterval) {
      const interval = setInterval(onRefresh, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [autoRefresh, onRefresh, refreshInterval])

  const uniqueUsers = React.useMemo(() => {
    const users = new Map<string, string>()
    logs.forEach((log) => {
      users.set(log.user_id, log.user_email)
    })
    return Array.from(users.entries())
  }, [logs])

  const filteredLogs = React.useMemo(() => {
    let filtered = logs

    // Date range filter
    const now = new Date()
    if (dateRange !== 'all') {
      const cutoff = new Date()
      switch (dateRange) {
        case 'today':
          cutoff.setHours(0, 0, 0, 0)
          break
        case 'week':
          cutoff.setDate(now.getDate() - 7)
          break
        case 'month':
          cutoff.setMonth(now.getMonth() - 1)
          break
      }
      filtered = filtered.filter((log) => log.timestamp >= cutoff)
    }

    // Action filter
    if (actionFilter !== 'all') {
      filtered = filtered.filter((log) => log.action_type === actionFilter)
    }

    // Resource filter
    if (resourceFilter !== 'all') {
      filtered = filtered.filter((log) => log.resource_type === resourceFilter)
    }

    // Status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter((log) => log.status === statusFilter)
    }

    // User filter
    if (userFilter !== 'all') {
      filtered = filtered.filter((log) => log.user_id === userFilter)
    }

    // Risk filter
    if (riskFilter !== 'all') {
      filtered = filtered.filter((log) => log.risk_level === riskFilter)
    }

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (log) =>
          log.action.toLowerCase().includes(query) ||
          log.user_email.toLowerCase().includes(query) ||
          log.ip_address.toLowerCase().includes(query) ||
          (log.resource_name && log.resource_name.toLowerCase().includes(query))
      )
    }

    return filtered
  }, [logs, actionFilter, resourceFilter, statusFilter, userFilter, riskFilter, searchQuery, dateRange])

  const stats = React.useMemo(() => {
    return {
      total: logs.length,
      success: logs.filter((l) => l.status === 'success').length,
      failed: logs.filter((l) => l.status === 'failed').length,
      denied: logs.filter((l) => l.status === 'denied').length,
      highRisk: logs.filter((l) => l.risk_level === 'high' || l.risk_level === 'critical').length,
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

  const handleExport = () => {
    if (onExport) {
      onExport(filteredLogs)
    }
  }

  const clearFilters = () => {
    setActionFilter('all')
    setResourceFilter('all')
    setStatusFilter('all')
    setUserFilter('all')
    setRiskFilter('all')
    setSearchQuery('')
    setDateRange('all')
  }

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Statistics */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Events</span>
            <Clock className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{stats.total}</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Successful</span>
            <CheckCircle2 className="w-5 h-5 text-green-600" />
          </div>
          <div className="text-2xl font-bold text-green-600">{stats.success}</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Failed</span>
            <XCircle className="w-5 h-5 text-red-600" />
          </div>
          <div className="text-2xl font-bold text-red-600">{stats.failed}</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Denied</span>
            <AlertTriangle className="w-5 h-5 text-yellow-600" />
          </div>
          <div className="text-2xl font-bold text-yellow-600">{stats.denied}</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">High Risk</span>
            <AlertTriangle className="w-5 h-5 text-red-600" />
          </div>
          <div className="text-2xl font-bold text-red-600">{stats.highRisk}</div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
            <Filter className="w-5 h-5" />
            Audit Logs
          </h3>
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

        {/* Search */}
        <div className="mb-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search logs by action, user, IP address..."
              className="w-full pl-10 pr-4 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Filter Controls */}
        <div className="flex flex-wrap gap-4">
          {/* Date Range */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Period:</label>
            <select
              value={dateRange}
              onChange={(e) => setDateRange(e.target.value as any)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Time</option>
              <option value="today">Today</option>
              <option value="week">Last 7 Days</option>
              <option value="month">Last 30 Days</option>
            </select>
          </div>

          {/* Action Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Action:</label>
            <select
              value={actionFilter}
              onChange={(e) => setActionFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Actions</option>
              {Object.entries(actionTypeConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>

          {/* Resource Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Resource:</label>
            <select
              value={resourceFilter}
              onChange={(e) => setResourceFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Resources</option>
              {Object.entries(resourceTypeConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>

          {/* Status Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Status:</label>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Status</option>
              {Object.entries(statusConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>

          {/* User Filter */}
          {uniqueUsers.length > 1 && (
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-700">User:</label>
              <select
                value={userFilter}
                onChange={(e) => setUserFilter(e.target.value)}
                className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Users</option>
                {uniqueUsers.map(([id, email]) => (
                  <option key={id} value={id}>
                    {email}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Risk Filter */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700">Risk:</label>
            <select
              value={riskFilter}
              onChange={(e) => setRiskFilter(e.target.value)}
              className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">All Risk Levels</option>
              {Object.entries(riskLevelConfig).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>

          {/* Clear Filters */}
          {(actionFilter !== 'all' ||
            resourceFilter !== 'all' ||
            statusFilter !== 'all' ||
            userFilter !== 'all' ||
            riskFilter !== 'all' ||
            searchQuery ||
            dateRange !== 'all') && (
            <button onClick={clearFilters} className="px-3 py-1.5 text-sm text-blue-600 hover:text-blue-700 font-medium">
              Clear All Filters
            </button>
          )}
        </div>

        <div className="mt-4 text-sm text-gray-600">
          Showing {filteredLogs.length} of {logs.length} entries
        </div>
      </div>

      {/* Log Entries */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 divide-y divide-gray-100">
        {filteredLogs.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
              <Shield className="w-8 h-8 text-gray-400" />
            </div>
            <p className="text-gray-500">No audit logs found</p>
          </div>
        ) : (
          filteredLogs.map((log) => (
            <AuditLogEntry
              key={log.id}
              log={log}
              isExpanded={expandedLogs.has(log.id)}
              onToggleExpand={() => toggleLogExpansion(log.id)}
            />
          ))
        )}
      </div>
    </div>
  )
}

interface AuditLogEntryProps {
  log: AuditLogEntry
  isExpanded: boolean
  onToggleExpand: () => void
}

function AuditLogEntry({ log, isExpanded, onToggleExpand }: AuditLogEntryProps) {
  const actionConfig = actionTypeConfig[log.action_type]
  const resourceConfig = resourceTypeConfig[log.resource_type]
  const statusInfo = statusConfig[log.status]
  const ActionIcon = actionConfig.icon
  const ResourceIcon = resourceConfig.icon
  const StatusIcon = statusInfo.icon

  const hasDetails = log.changes || log.metadata || log.user_agent

  return (
    <div className="px-6 py-4 hover:bg-gray-50 transition-colors">
      <div className="flex items-start gap-4">
        {/* Expand/Collapse */}
        {hasDetails && (
          <button
            onClick={onToggleExpand}
            className="flex-shrink-0 mt-1 text-gray-400 hover:text-gray-600 transition-colors"
          >
            {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </button>
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-start justify-between gap-4 mb-2">
            <div className="flex items-center gap-2 flex-wrap">
              <span
                className={clsx(
                  'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                  actionConfig.color,
                  actionConfig.bg
                )}
              >
                <ActionIcon className="w-3 h-3" />
                {actionConfig.label}
              </span>
              <span
                className={clsx(
                  'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                  resourceConfig.color,
                  'bg-gray-100'
                )}
              >
                <ResourceIcon className="w-3 h-3" />
                {resourceConfig.label}
              </span>
              <span
                className={clsx(
                  'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                  statusInfo.color,
                  statusInfo.bg
                )}
              >
                <StatusIcon className="w-3 h-3" />
                {statusInfo.label}
              </span>
              {log.risk_level && (
                <span
                  className={clsx(
                    'px-2 py-0.5 text-xs font-medium rounded-full',
                    riskLevelConfig[log.risk_level].color,
                    riskLevelConfig[log.risk_level].bg
                  )}
                >
                  {riskLevelConfig[log.risk_level].label}
                </span>
              )}
            </div>
            <time className="text-xs text-gray-500 whitespace-nowrap">
              {formatDistanceToNow(log.timestamp, { addSuffix: true })}
            </time>
          </div>

          {/* Main Info */}
          <div className="text-sm text-gray-900 mb-2">
            <span className="font-medium">{log.user_email}</span>
            {log.user_name && <span className="text-gray-600"> ({log.user_name})</span>} {log.action}
            {log.resource_name && (
              <>
                {' '}
                <span className="font-medium">{log.resource_name}</span>
              </>
            )}
          </div>

          {/* Metadata Summary */}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <Globe className="w-3 h-3" />
              {log.ip_address}
            </span>
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {format(log.timestamp, 'MMM d, yyyy HH:mm:ss')}
            </span>
            {log.resource_id && <span>Resource ID: {log.resource_id}</span>}
          </div>

          {/* Expanded Details */}
          {isExpanded && (
            <div className="mt-4 space-y-3">
              {/* Changes */}
              {log.changes && log.changes.length > 0 && (
                <div className="bg-gray-50 rounded-lg p-3">
                  <div className="text-xs font-semibold text-gray-700 mb-2">CHANGES</div>
                  <div className="space-y-2">
                    {log.changes.map((change, idx) => (
                      <div key={idx} className="text-xs">
                        <div className="font-medium text-gray-900">{change.field}</div>
                        <div className="flex items-center gap-2 mt-1">
                          <span className="text-red-600">
                            {change.old_value !== null ? String(change.old_value) : 'null'}
                          </span>
                          <span className="text-gray-400">→</span>
                          <span className="text-green-600">
                            {change.new_value !== null ? String(change.new_value) : 'null'}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* User Agent */}
              {log.user_agent && (
                <div className="bg-blue-50 rounded-lg p-3">
                  <div className="text-xs font-semibold text-blue-700 mb-1">USER AGENT</div>
                  <div className="text-xs text-blue-900 font-mono">{log.user_agent}</div>
                </div>
              )}

              {/* Metadata */}
              {log.metadata && Object.keys(log.metadata).length > 0 && (
                <div className="bg-gray-900 text-gray-100 rounded-lg p-3 overflow-x-auto">
                  <div className="text-xs font-semibold text-gray-400 mb-2">METADATA</div>
                  <pre className="text-xs">{JSON.stringify(log.metadata, null, 2)}</pre>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// Sample data generator
export function generateSampleAuditLogs(count: number = 50): AuditLogEntry[] {
  const users = [
    { id: 'user-1', email: 'admin@example.com', name: 'Admin User' },
    { id: 'user-2', email: 'john@example.com', name: 'John Doe' },
    { id: 'user-3', email: 'jane@example.com', name: 'Jane Smith' },
  ]

  const actions = [
    { type: 'create', resource: 'database', text: 'created database', risk: 'low' },
    { type: 'update', resource: 'database', text: 'updated database configuration', risk: 'medium' },
    { type: 'delete', resource: 'backup', text: 'deleted backup', risk: 'high' },
    { type: 'login', resource: 'auth', text: 'logged in', risk: 'low' },
    { type: 'failed_login', resource: 'auth', text: 'failed login attempt', risk: 'medium' },
    { type: 'permission_change', resource: 'user', text: 'changed user permissions', risk: 'high' },
    { type: 'create', resource: 'api_key', text: 'created API key', risk: 'medium' },
    { type: 'update', resource: 'settings', text: 'updated system settings', risk: 'medium' },
  ]

  const statuses: AuditLogEntry['status'][] = ['success', 'failed', 'denied']
  const riskLevels: AuditLogEntry['risk_level'][] = ['low', 'medium', 'high', 'critical']

  return Array.from({ length: count }, (_, i) => {
    const user = users[Math.floor(Math.random() * users.length)]
    const action = actions[Math.floor(Math.random() * actions.length)]
    const status = statuses[Math.floor(Math.random() * statuses.length)]

    return {
      id: `audit-${i + 1}`,
      timestamp: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000),
      user_id: user.id,
      user_email: user.email,
      user_name: user.name,
      action: action.text,
      resource_type: action.resource as any,
      resource_id: `resource-${Math.floor(Math.random() * 100)}`,
      resource_name: `${action.resource}-${Math.floor(Math.random() * 100)}`,
      action_type: action.type as any,
      status,
      ip_address: `${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}.${Math.floor(
        Math.random() * 255
      )}.${Math.floor(Math.random() * 255)}`,
      user_agent: Math.random() > 0.5 ? 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' : undefined,
      changes:
        action.type === 'update' && Math.random() > 0.5
          ? [
              {
                field: 'name',
                old_value: 'old-value',
                new_value: 'new-value',
              },
              {
                field: 'enabled',
                old_value: false,
                new_value: true,
              },
            ]
          : undefined,
      metadata:
        Math.random() > 0.6
          ? {
              session_id: `session-${Math.random().toString(36).substring(7)}`,
              request_id: `req-${Math.random().toString(36).substring(7)}`,
            }
          : undefined,
      risk_level: riskLevels[Math.floor(Math.random() * riskLevels.length)],
    }
  }).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
}
