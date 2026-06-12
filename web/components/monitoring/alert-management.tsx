'use client'

import React from 'react'
import { formatDistanceToNow } from 'date-fns'
import {
  Bell,
  BellOff,
  AlertTriangle,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Mail,
  MessageSquare,
  Webhook,
  Clock,
  Plus,
  Edit2,
  Trash2,
  Play,
  Pause,
  Settings,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface Alert {
  id: string
  name: string
  description: string
  enabled: boolean
  severity: 'info' | 'warning' | 'critical'
  condition: {
    metric: string
    operator: '>' | '<' | '=' | '>=' | '<=' | '!='
    threshold: number
    duration?: number // seconds
  }
  channels: ('email' | 'slack' | 'webhook')[]
  last_triggered?: Date
  trigger_count: number
  created_at: Date
  updated_at: Date
}

export interface AlertTrigger {
  id: string
  alert_id: string
  alert_name: string
  severity: 'info' | 'warning' | 'critical'
  triggered_at: Date
  resolved_at?: Date
  current_value: number
  threshold: number
  message: string
  metadata?: Record<string, any>
}

interface AlertManagementProps {
  alerts: Alert[]
  triggers: AlertTrigger[]
  onCreateAlert?: () => void
  onEditAlert?: (alertId: string) => void
  onDeleteAlert?: (alertId: string) => void
  onToggleAlert?: (alertId: string, enabled: boolean) => void
  onTestAlert?: (alertId: string) => void
  onAcknowledge?: (triggerId: string) => void
  className?: string
}

const severityConfig = {
  info: {
    icon: AlertCircle,
    color: 'text-blue-600',
    bg: 'bg-blue-100',
    border: 'border-blue-200',
    label: 'Info',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-yellow-600',
    bg: 'bg-yellow-100',
    border: 'border-yellow-200',
    label: 'Warning',
  },
  critical: {
    icon: XCircle,
    color: 'text-red-600',
    bg: 'bg-red-100',
    border: 'border-red-200',
    label: 'Critical',
  },
}

const channelConfig = {
  email: {
    icon: Mail,
    label: 'Email',
    color: 'text-blue-600',
  },
  slack: {
    icon: MessageSquare,
    label: 'Slack',
    color: 'text-purple-600',
  },
  webhook: {
    icon: Webhook,
    label: 'Webhook',
    color: 'text-green-600',
  },
}

export function AlertManagement({
  alerts,
  triggers,
  onCreateAlert,
  onEditAlert,
  onDeleteAlert,
  onToggleAlert,
  onTestAlert,
  onAcknowledge,
  className,
}: AlertManagementProps) {
  const [activeTab, setActiveTab] = React.useState<'alerts' | 'triggers'>('alerts')
  const [severityFilter, setSeverityFilter] = React.useState<string>('all')

  const filteredAlerts = React.useMemo(() => {
    if (severityFilter === 'all') return alerts
    return alerts.filter((a) => a.severity === severityFilter)
  }, [alerts, severityFilter])

  const activeTriggers = React.useMemo(() => {
    return triggers.filter((t) => !t.resolved_at)
  }, [triggers])

  const alertStats = React.useMemo(() => {
    const enabled = alerts.filter((a) => a.enabled).length
    const bysSeverity = {
      info: alerts.filter((a) => a.severity === 'info').length,
      warning: alerts.filter((a) => a.severity === 'warning').length,
      critical: alerts.filter((a) => a.severity === 'critical').length,
    }
    return { total: alerts.length, enabled, bySeverity: bysSeverity }
  }, [alerts])

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Alerts</span>
            <Bell className="w-5 h-5 text-blue-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{alertStats.total}</div>
          <div className="text-xs text-gray-500 mt-1">{alertStats.enabled} enabled</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Active Triggers</span>
            <AlertTriangle className="w-5 h-5 text-yellow-600" />
          </div>
          <div className="text-2xl font-bold text-yellow-600">{activeTriggers.length}</div>
          <div className="text-xs text-gray-500 mt-1">Requires attention</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Critical Alerts</span>
            <XCircle className="w-5 h-5 text-red-600" />
          </div>
          <div className="text-2xl font-bold text-red-600">{alertStats.bySeverity.critical}</div>
          <div className="text-xs text-gray-500 mt-1">Configured</div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-600">Total Triggers</span>
            <Clock className="w-5 h-5 text-purple-600" />
          </div>
          <div className="text-2xl font-bold text-gray-900">{triggers.length}</div>
          <div className="text-xs text-gray-500 mt-1">Last 24 hours</div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="border-b border-gray-200">
          <div className="flex items-center justify-between px-6 py-4">
            <div className="flex space-x-4">
              <button
                onClick={() => setActiveTab('alerts')}
                className={clsx(
                  'px-4 py-2 text-sm font-medium rounded-md transition-colors',
                  activeTab === 'alerts'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                )}
              >
                Alert Rules ({alerts.length})
              </button>
              <button
                onClick={() => setActiveTab('triggers')}
                className={clsx(
                  'px-4 py-2 text-sm font-medium rounded-md transition-colors relative',
                  activeTab === 'triggers'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                )}
              >
                Triggered Alerts ({triggers.length})
                {activeTriggers.length > 0 && (
                  <span className="absolute -top-1 -right-1 flex h-5 w-5">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-5 w-5 bg-red-500 text-white text-xs items-center justify-center">
                      {activeTriggers.length}
                    </span>
                  </span>
                )}
              </button>
            </div>

            {activeTab === 'alerts' && onCreateAlert && (
              <button
                onClick={onCreateAlert}
                className="px-4 py-2 text-sm text-white bg-blue-600 rounded-md hover:bg-blue-700 flex items-center gap-2"
              >
                <Plus className="w-4 h-4" />
                Create Alert
              </button>
            )}
          </div>

          {/* Filters for Alerts Tab */}
          {activeTab === 'alerts' && (
            <div className="px-6 pb-4">
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium text-gray-700">Severity:</label>
                <div className="flex gap-2">
                  <button
                    onClick={() => setSeverityFilter('all')}
                    className={clsx(
                      'px-3 py-1 text-xs font-medium rounded-md',
                      severityFilter === 'all'
                        ? 'bg-blue-100 text-blue-700'
                        : 'text-gray-600 hover:bg-gray-100'
                    )}
                  >
                    All
                  </button>
                  {Object.entries(severityConfig).map(([severity, config]) => (
                    <button
                      key={severity}
                      onClick={() => setSeverityFilter(severity)}
                      className={clsx(
                        'px-3 py-1 text-xs font-medium rounded-md flex items-center gap-1',
                        severityFilter === severity ? config.bg + ' ' + config.color : 'text-gray-600 hover:bg-gray-100'
                      )}
                    >
                      <config.icon className="w-3 h-3" />
                      {config.label}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Content */}
        <div className="divide-y divide-gray-100">
          {activeTab === 'alerts' ? (
            filteredAlerts.length === 0 ? (
              <div className="px-6 py-12 text-center">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
                  <Bell className="w-8 h-8 text-gray-400" />
                </div>
                <p className="text-gray-500">No alerts configured</p>
                {onCreateAlert && (
                  <button
                    onClick={onCreateAlert}
                    className="mt-4 px-4 py-2 text-sm text-blue-600 hover:text-blue-700 font-medium"
                  >
                    Create your first alert
                  </button>
                )}
              </div>
            ) : (
              filteredAlerts.map((alert) => (
                <AlertRuleItem
                  key={alert.id}
                  alert={alert}
                  onEdit={onEditAlert}
                  onDelete={onDeleteAlert}
                  onToggle={onToggleAlert}
                  onTest={onTestAlert}
                />
              ))
            )
          ) : triggers.length === 0 ? (
            <div className="px-6 py-12 text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
                <CheckCircle2 className="w-8 h-8 text-green-500" />
              </div>
              <p className="text-gray-500">No alerts triggered</p>
            </div>
          ) : (
            triggers.map((trigger) => (
              <AlertTriggerItem key={trigger.id} trigger={trigger} onAcknowledge={onAcknowledge} />
            ))
          )}
        </div>
      </div>
    </div>
  )
}

interface AlertRuleItemProps {
  alert: Alert
  onEdit?: (alertId: string) => void
  onDelete?: (alertId: string) => void
  onToggle?: (alertId: string, enabled: boolean) => void
  onTest?: (alertId: string) => void
}

function AlertRuleItem({ alert, onEdit, onDelete, onToggle, onTest }: AlertRuleItemProps) {
  const severityInfo = severityConfig[alert.severity]
  const SeverityIcon = severityInfo.icon

  return (
    <div className="px-6 py-4 hover:bg-gray-50 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-2">
            <h4 className="text-sm font-semibold text-gray-900">{alert.name}</h4>
            <span
              className={clsx(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border',
                severityInfo.color,
                severityInfo.bg,
                severityInfo.border
              )}
            >
              <SeverityIcon className="w-3 h-3" />
              {severityInfo.label}
            </span>
            {!alert.enabled && (
              <span className="px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-full flex items-center gap-1">
                <BellOff className="w-3 h-3" />
                Disabled
              </span>
            )}
          </div>

          <p className="text-sm text-gray-600 mb-2">{alert.description}</p>

          <div className="flex flex-wrap gap-x-4 gap-y-2 text-xs text-gray-500">
            <span className="font-mono">
              {alert.condition.metric} {alert.condition.operator} {alert.condition.threshold}
              {alert.condition.duration && ` for ${alert.condition.duration}s`}
            </span>
            <span className="flex items-center gap-1">
              Channels:
              {alert.channels.map((channel) => {
                const ChannelIcon = channelConfig[channel].icon
                return (
                  <span key={channel} className={clsx('flex items-center gap-1', channelConfig[channel].color)}>
                    <ChannelIcon className="w-3 h-3" />
                  </span>
                )
              })}
            </span>
            {alert.last_triggered && (
              <span>Last triggered {formatDistanceToNow(alert.last_triggered, { addSuffix: true })}</span>
            )}
            <span>Triggered {alert.trigger_count} times</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {onTest && (
            <button
              onClick={() => onTest(alert.id)}
              className="p-2 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
              title="Test Alert"
            >
              <Play className="w-4 h-4" />
            </button>
          )}
          {onToggle && (
            <button
              onClick={() => onToggle(alert.id, !alert.enabled)}
              className={clsx(
                'p-2 rounded-md transition-colors',
                alert.enabled
                  ? 'text-green-600 hover:text-yellow-600 hover:bg-yellow-50'
                  : 'text-gray-400 hover:text-green-600 hover:bg-green-50'
              )}
              title={alert.enabled ? 'Disable Alert' : 'Enable Alert'}
            >
              {alert.enabled ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            </button>
          )}
          {onEdit && (
            <button
              onClick={() => onEdit(alert.id)}
              className="p-2 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
              title="Edit Alert"
            >
              <Edit2 className="w-4 h-4" />
            </button>
          )}
          {onDelete && (
            <button
              onClick={() => onDelete(alert.id)}
              className="p-2 text-gray-600 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
              title="Delete Alert"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

interface AlertTriggerItemProps {
  trigger: AlertTrigger
  onAcknowledge?: (triggerId: string) => void
}

function AlertTriggerItem({ trigger, onAcknowledge }: AlertTriggerItemProps) {
  const severityInfo = severityConfig[trigger.severity]
  const SeverityIcon = severityInfo.icon
  const isActive = !trigger.resolved_at

  return (
    <div
      className={clsx(
        'px-6 py-4 transition-colors',
        isActive ? 'bg-yellow-50 hover:bg-yellow-100' : 'hover:bg-gray-50'
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-2">
            <h4 className="text-sm font-semibold text-gray-900">{trigger.alert_name}</h4>
            <span
              className={clsx(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border',
                severityInfo.color,
                severityInfo.bg,
                severityInfo.border
              )}
            >
              <SeverityIcon className="w-3 h-3" />
              {severityInfo.label}
            </span>
            {isActive && (
              <span className="px-2 py-0.5 text-xs font-medium bg-red-100 text-red-700 rounded-full animate-pulse">
                Active
              </span>
            )}
            {trigger.resolved_at && (
              <span className="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-700 rounded-full">
                Resolved
              </span>
            )}
          </div>

          <p className="text-sm text-gray-900 mb-2">{trigger.message}</p>

          <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
            <span>
              Current: <span className="font-semibold text-gray-900">{trigger.current_value.toFixed(2)}</span>
            </span>
            <span>
              Threshold: <span className="font-semibold text-gray-900">{trigger.threshold.toFixed(2)}</span>
            </span>
            <span>Triggered {formatDistanceToNow(trigger.triggered_at, { addSuffix: true })}</span>
            {trigger.resolved_at && (
              <span>Resolved {formatDistanceToNow(trigger.resolved_at, { addSuffix: true })}</span>
            )}
          </div>

          {trigger.metadata && (
            <div className="mt-2 p-2 bg-gray-100 rounded text-xs">
              <pre className="text-gray-700">{JSON.stringify(trigger.metadata, null, 2)}</pre>
            </div>
          )}
        </div>

        {isActive && onAcknowledge && (
          <button
            onClick={() => onAcknowledge(trigger.id)}
            className="px-3 py-1.5 text-xs text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-md border border-blue-200 transition-colors"
          >
            Acknowledge
          </button>
        )}
      </div>
    </div>
  )
}

// Sample data generators
export function generateSampleAlerts(count: number = 10): Alert[] {
  const severities: Alert['severity'][] = ['info', 'warning', 'critical']
  const metrics = [
    'backup_failure_rate',
    'api_response_time',
    'disk_usage',
    'cpu_usage',
    'memory_usage',
    'database_connections',
  ]
  const channels: ('email' | 'slack' | 'webhook')[][] = [
    ['email'],
    ['slack'],
    ['email', 'slack'],
    ['email', 'slack', 'webhook'],
  ]

  return Array.from({ length: count }, (_, i) => ({
    id: `alert-${i + 1}`,
    name: `Alert ${i + 1}: ${metrics[i % metrics.length]}`,
    description: `Monitor ${metrics[i % metrics.length]} and alert when threshold is exceeded`,
    enabled: Math.random() > 0.3,
    severity: severities[Math.floor(Math.random() * severities.length)],
    condition: {
      metric: metrics[i % metrics.length],
      operator: Math.random() > 0.5 ? '>' : '<',
      threshold: Math.floor(Math.random() * 100),
      duration: Math.random() > 0.5 ? Math.floor(Math.random() * 300) + 60 : undefined,
    },
    channels: channels[Math.floor(Math.random() * channels.length)],
    last_triggered: Math.random() > 0.5 ? new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000) : undefined,
    trigger_count: Math.floor(Math.random() * 50),
    created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000),
    updated_at: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000),
  }))
}

export function generateSampleTriggers(alerts: Alert[], count: number = 15): AlertTrigger[] {
  const severities: AlertTrigger['severity'][] = ['info', 'warning', 'critical']

  return Array.from({ length: count }, (_, i) => {
    const alert = alerts[Math.floor(Math.random() * alerts.length)]
    const triggeredAt = new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000)
    const isResolved = Math.random() > 0.3

    return {
      id: `trigger-${i + 1}`,
      alert_id: alert.id,
      alert_name: alert.name,
      severity: severities[Math.floor(Math.random() * severities.length)],
      triggered_at: triggeredAt,
      resolved_at: isResolved ? new Date(triggeredAt.getTime() + Math.random() * 3600000) : undefined,
      current_value: alert.condition.threshold + (Math.random() - 0.5) * 20,
      threshold: alert.condition.threshold,
      message: `${alert.condition.metric} exceeded threshold`,
      metadata: {
        database_id: `db-${Math.floor(Math.random() * 5) + 1}`,
        value: Math.floor(Math.random() * 1000),
      },
    }
  }).sort((a, b) => b.triggered_at.getTime() - a.triggered_at.getTime())
}
