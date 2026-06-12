'use client'

import React from 'react'
import { formatDistanceToNow } from 'date-fns'
import {
  Server,
  Database,
  HardDrive,
  Wifi,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Activity,
  Zap,
  Shield,
  Cloud,
  RefreshCw,
  TrendingUp,
  TrendingDown,
  Minus,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface ServiceStatus {
  name: string
  status: 'operational' | 'degraded' | 'down' | 'maintenance'
  uptime: number // percentage
  last_check: Date
  response_time?: number // ms
  error_rate?: number // percentage
  version?: string
  dependencies?: string[]
}

export interface SystemHealth {
  overall_status: 'healthy' | 'degraded' | 'critical'
  services: ServiceStatus[]
  infrastructure: {
    cpu_usage: number
    memory_usage: number
    disk_usage: number
    network_status: 'good' | 'fair' | 'poor'
  }
  backup_systems: {
    last_successful_backup: Date
    backup_count_24h: number
    success_rate: number
  }
  security: {
    ssl_certificate_valid: boolean
    ssl_expires_at?: Date
    last_security_scan?: Date
    vulnerabilities_count: number
  }
  incidents: IncidentReport[]
  uptime_stats: {
    current_uptime: number // seconds
    uptime_7d: number // percentage
    uptime_30d: number // percentage
    uptime_90d: number // percentage
  }
}

export interface IncidentReport {
  id: string
  title: string
  status: 'investigating' | 'identified' | 'monitoring' | 'resolved'
  severity: 'minor' | 'major' | 'critical'
  started_at: Date
  resolved_at?: Date
  affected_services: string[]
  description: string
}

interface SystemStatusProps {
  health: SystemHealth
  onRefresh?: () => void
  refreshInterval?: number
  className?: string
}

const statusConfig = {
  operational: {
    icon: CheckCircle2,
    color: 'text-green-600',
    bg: 'bg-green-100',
    border: 'border-green-200',
    label: 'Operational',
  },
  degraded: {
    icon: AlertTriangle,
    color: 'text-yellow-600',
    bg: 'bg-yellow-100',
    border: 'border-yellow-200',
    label: 'Degraded',
  },
  down: {
    icon: XCircle,
    color: 'text-red-600',
    bg: 'bg-red-100',
    border: 'border-red-200',
    label: 'Down',
  },
  maintenance: {
    icon: Clock,
    color: 'text-blue-600',
    bg: 'bg-blue-100',
    border: 'border-blue-200',
    label: 'Maintenance',
  },
}

const incidentStatusConfig = {
  investigating: { color: 'text-yellow-600', bg: 'bg-yellow-100', label: 'Investigating' },
  identified: { color: 'text-orange-600', bg: 'bg-orange-100', label: 'Identified' },
  monitoring: { color: 'text-blue-600', bg: 'bg-blue-100', label: 'Monitoring' },
  resolved: { color: 'text-green-600', bg: 'bg-green-100', label: 'Resolved' },
}

const severityConfig = {
  minor: { color: 'text-yellow-600', bg: 'bg-yellow-50', label: 'Minor' },
  major: { color: 'text-orange-600', bg: 'bg-orange-50', label: 'Major' },
  critical: { color: 'text-red-600', bg: 'bg-red-50', label: 'Critical' },
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

export function SystemStatus({ health, onRefresh, refreshInterval, className }: SystemStatusProps) {
  // Auto-refresh
  React.useEffect(() => {
    if (refreshInterval && onRefresh) {
      const interval = setInterval(onRefresh, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [refreshInterval, onRefresh])

  const overallStatusInfo = {
    healthy: { icon: CheckCircle2, color: 'text-green-600', bg: 'bg-green-100', label: 'All Systems Operational' },
    degraded: { icon: AlertTriangle, color: 'text-yellow-600', bg: 'bg-yellow-100', label: 'Degraded Performance' },
    critical: { icon: XCircle, color: 'text-red-600', bg: 'bg-red-100', label: 'System Issues Detected' },
  }

  const statusInfo = overallStatusInfo[health.overall_status]
  const StatusIcon = statusInfo.icon

  const activeIncidents = health.incidents.filter((i) => i.status !== 'resolved')

  return (
    <div className={clsx('space-y-6', className)}>
      {/* Overall Status Header */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-4">
            <div className={clsx('p-3 rounded-full', statusInfo.bg)}>
              <StatusIcon className={clsx('w-8 h-8', statusInfo.color)} />
            </div>
            <div>
              <h2 className="text-2xl font-bold text-gray-900">{statusInfo.label}</h2>
              <p className="text-sm text-gray-600 mt-1">
                Last updated {formatDistanceToNow(health.services[0]?.last_check || new Date(), { addSuffix: true })}
              </p>
            </div>
          </div>
          {onRefresh && (
            <button
              onClick={onRefresh}
              className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
          )}
        </div>

        {/* Quick Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{health.uptime_stats.uptime_30d.toFixed(2)}%</div>
            <div className="text-xs text-gray-600">30-Day Uptime</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{formatUptime(health.uptime_stats.current_uptime)}</div>
            <div className="text-xs text-gray-600">Current Uptime</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600">{health.backup_systems.success_rate.toFixed(1)}%</div>
            <div className="text-xs text-gray-600">Backup Success</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900">{activeIncidents.length}</div>
            <div className="text-xs text-gray-600">Active Incidents</div>
          </div>
        </div>
      </div>

      {/* Active Incidents */}
      {activeIncidents.length > 0 && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-yellow-600" />
              Active Incidents ({activeIncidents.length})
            </h3>
          </div>
          <div className="divide-y divide-gray-100">
            {activeIncidents.map((incident) => (
              <IncidentItem key={incident.id} incident={incident} />
            ))}
          </div>
        </div>
      )}

      {/* Services Status */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
            <Server className="w-5 h-5 text-blue-600" />
            Services Status
          </h3>
        </div>
        <div className="divide-y divide-gray-100">
          {health.services.map((service) => (
            <ServiceStatusItem key={service.name} service={service} />
          ))}
        </div>
      </div>

      {/* Infrastructure & Security */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Infrastructure */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Activity className="w-5 h-5 text-blue-600" />
            Infrastructure
          </h3>
          <div className="space-y-4">
            <MetricBar
              label="CPU Usage"
              value={health.infrastructure.cpu_usage}
              max={100}
              unit="%"
              threshold={80}
            />
            <MetricBar
              label="Memory Usage"
              value={health.infrastructure.memory_usage}
              max={100}
              unit="%"
              threshold={80}
            />
            <MetricBar
              label="Disk Usage"
              value={health.infrastructure.disk_usage}
              max={100}
              unit="%"
              threshold={90}
            />
            <div className="flex items-center justify-between pt-2 border-t border-gray-200">
              <span className="text-sm text-gray-600 flex items-center gap-2">
                <Wifi className="w-4 h-4" />
                Network Status
              </span>
              <span
                className={clsx(
                  'text-sm font-semibold',
                  health.infrastructure.network_status === 'good'
                    ? 'text-green-600'
                    : health.infrastructure.network_status === 'fair'
                    ? 'text-yellow-600'
                    : 'text-red-600'
                )}
              >
                {health.infrastructure.network_status.toUpperCase()}
              </span>
            </div>
          </div>
        </div>

        {/* Security */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Shield className="w-5 h-5 text-blue-600" />
            Security Status
          </h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">SSL Certificate</span>
              <span
                className={clsx(
                  'text-sm font-semibold flex items-center gap-1',
                  health.security.ssl_certificate_valid ? 'text-green-600' : 'text-red-600'
                )}
              >
                {health.security.ssl_certificate_valid ? (
                  <>
                    <CheckCircle2 className="w-4 h-4" />
                    Valid
                  </>
                ) : (
                  <>
                    <XCircle className="w-4 h-4" />
                    Invalid
                  </>
                )}
              </span>
            </div>
            {health.security.ssl_expires_at && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">SSL Expires</span>
                <span className="text-sm text-gray-900">
                  {formatDistanceToNow(health.security.ssl_expires_at, { addSuffix: true })}
                </span>
              </div>
            )}
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Vulnerabilities</span>
              <span
                className={clsx(
                  'text-sm font-semibold',
                  health.security.vulnerabilities_count === 0
                    ? 'text-green-600'
                    : health.security.vulnerabilities_count < 5
                    ? 'text-yellow-600'
                    : 'text-red-600'
                )}
              >
                {health.security.vulnerabilities_count}
              </span>
            </div>
            {health.security.last_security_scan && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">Last Security Scan</span>
                <span className="text-sm text-gray-900">
                  {formatDistanceToNow(health.security.last_security_scan, { addSuffix: true })}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Backup Systems */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <Database className="w-5 h-5 text-blue-600" />
          Backup Systems
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div>
            <div className="text-sm text-gray-600 mb-1">Last Successful Backup</div>
            <div className="text-lg font-semibold text-gray-900">
              {formatDistanceToNow(health.backup_systems.last_successful_backup, { addSuffix: true })}
            </div>
          </div>
          <div>
            <div className="text-sm text-gray-600 mb-1">Backups (24h)</div>
            <div className="text-lg font-semibold text-gray-900">{health.backup_systems.backup_count_24h}</div>
          </div>
          <div>
            <div className="text-sm text-gray-600 mb-1">Success Rate</div>
            <div className="text-lg font-semibold text-green-600">{health.backup_systems.success_rate.toFixed(1)}%</div>
          </div>
        </div>
      </div>

      {/* Uptime History */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <Clock className="w-5 h-5 text-blue-600" />
          Uptime History
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <UptimeCard label="7 Days" uptime={health.uptime_stats.uptime_7d} />
          <UptimeCard label="30 Days" uptime={health.uptime_stats.uptime_30d} />
          <UptimeCard label="90 Days" uptime={health.uptime_stats.uptime_90d} />
        </div>
      </div>
    </div>
  )
}

interface ServiceStatusItemProps {
  service: ServiceStatus
}

function ServiceStatusItem({ service }: ServiceStatusItemProps) {
  const config = statusConfig[service.status]
  const Icon = config.icon

  return (
    <div className="px-6 py-4 hover:bg-gray-50 transition-colors">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4 flex-1">
          <span
            className={clsx(
              'inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium border',
              config.color,
              config.bg,
              config.border
            )}
          >
            <Icon className="w-4 h-4" />
            {config.label}
          </span>
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <h4 className="text-sm font-semibold text-gray-900">{service.name}</h4>
              {service.version && <span className="text-xs text-gray-500">v{service.version}</span>}
            </div>
            {service.dependencies && service.dependencies.length > 0 && (
              <div className="text-xs text-gray-500 mt-1">
                Dependencies: {service.dependencies.join(', ')}
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-6 text-sm">
          {service.response_time !== undefined && (
            <div className="text-right">
              <div className="text-gray-600">Response Time</div>
              <div className="font-semibold text-gray-900">{service.response_time}ms</div>
            </div>
          )}
          {service.error_rate !== undefined && (
            <div className="text-right">
              <div className="text-gray-600">Error Rate</div>
              <div
                className={clsx(
                  'font-semibold',
                  service.error_rate < 1 ? 'text-green-600' : service.error_rate < 5 ? 'text-yellow-600' : 'text-red-600'
                )}
              >
                {service.error_rate.toFixed(2)}%
              </div>
            </div>
          )}
          <div className="text-right">
            <div className="text-gray-600">Uptime</div>
            <div className="font-semibold text-green-600">{service.uptime.toFixed(2)}%</div>
          </div>
        </div>
      </div>
    </div>
  )
}

interface IncidentItemProps {
  incident: IncidentReport
}

function IncidentItem({ incident }: IncidentItemProps) {
  const statusInfo = incidentStatusConfig[incident.status]
  const severityInfo = severityConfig[incident.severity]

  return (
    <div className="px-6 py-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <h4 className="text-sm font-semibold text-gray-900">{incident.title}</h4>
            <span className={clsx('px-2 py-0.5 text-xs font-medium rounded-full', statusInfo.color, statusInfo.bg)}>
              {statusInfo.label}
            </span>
            <span className={clsx('px-2 py-0.5 text-xs font-medium rounded-full', severityInfo.color, severityInfo.bg)}>
              {severityInfo.label}
            </span>
          </div>
          <p className="text-sm text-gray-600 mb-2">{incident.description}</p>
          <div className="flex items-center gap-4 text-xs text-gray-500">
            <span>Started {formatDistanceToNow(incident.started_at, { addSuffix: true })}</span>
            <span>Affected: {incident.affected_services.join(', ')}</span>
          </div>
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
  threshold: number
}

function MetricBar({ label, value, max, unit, threshold }: MetricBarProps) {
  const percentage = (value / max) * 100
  const isWarning = value > threshold

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-600">{label}</span>
        <span className={clsx('text-sm font-semibold', isWarning ? 'text-yellow-600' : 'text-gray-900')}>
          {value.toFixed(1)}
          {unit}
        </span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={clsx('h-2 rounded-full transition-all', isWarning ? 'bg-yellow-500' : 'bg-green-500')}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  )
}

interface UptimeCardProps {
  label: string
  uptime: number
}

function UptimeCard({ label, uptime }: UptimeCardProps) {
  return (
    <div className="text-center p-4 bg-gray-50 rounded-lg">
      <div className="text-sm text-gray-600 mb-2">{label}</div>
      <div
        className={clsx(
          'text-3xl font-bold',
          uptime >= 99.9 ? 'text-green-600' : uptime >= 99 ? 'text-yellow-600' : 'text-red-600'
        )}
      >
        {uptime.toFixed(3)}%
      </div>
      <div className="mt-2 flex items-center justify-center gap-1 text-xs text-gray-500">
        {uptime >= 99.9 ? (
          <>
            <TrendingUp className="w-3 h-3 text-green-600" />
            Excellent
          </>
        ) : uptime >= 99 ? (
          <>
            <Minus className="w-3 h-3 text-yellow-600" />
            Good
          </>
        ) : (
          <>
            <TrendingDown className="w-3 h-3 text-red-600" />
            Needs Attention
          </>
        )}
      </div>
    </div>
  )
}

// Sample data generator
export function generateSampleSystemHealth(): SystemHealth {
  const services: ServiceStatus[] = [
    {
      name: 'API Server',
      status: 'operational',
      uptime: 99.98,
      last_check: new Date(),
      response_time: 45,
      error_rate: 0.12,
      version: '2.1.0',
      dependencies: ['Database', 'Cache'],
    },
    {
      name: 'Database',
      status: 'operational',
      uptime: 99.95,
      last_check: new Date(),
      response_time: 12,
      error_rate: 0.05,
      version: '14.2',
    },
    {
      name: 'Backup Engine',
      status: 'operational',
      uptime: 99.92,
      last_check: new Date(),
      version: '1.5.3',
    },
    {
      name: 'Storage Manager',
      status: 'degraded',
      uptime: 98.5,
      last_check: new Date(),
      response_time: 250,
      error_rate: 3.2,
      version: '1.2.1',
    },
  ]

  return {
    overall_status: 'healthy',
    services,
    infrastructure: {
      cpu_usage: 45.2,
      memory_usage: 62.8,
      disk_usage: 73.5,
      network_status: 'good',
    },
    backup_systems: {
      last_successful_backup: new Date(Date.now() - 3600000),
      backup_count_24h: 24,
      success_rate: 99.5,
    },
    security: {
      ssl_certificate_valid: true,
      ssl_expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
      last_security_scan: new Date(Date.now() - 24 * 60 * 60 * 1000),
      vulnerabilities_count: 0,
    },
    incidents: [],
    uptime_stats: {
      current_uptime: 864000, // 10 days
      uptime_7d: 99.95,
      uptime_30d: 99.92,
      uptime_90d: 99.88,
    },
  }
}
