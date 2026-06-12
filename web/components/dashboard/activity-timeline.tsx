'use client'

import React from 'react'
import { formatDistanceToNow } from 'date-fns'
import {
  Database,
  CheckCircle2,
  XCircle,
  Clock,
  AlertTriangle,
  RefreshCw,
  Upload,
  Download,
  Calendar,
  Server,
  User,
  Settings,
} from 'lucide-react'
import { clsx } from 'clsx'

export interface ActivityEvent {
  id: string
  type: 'backup' | 'restore' | 'schedule' | 'error' | 'warning' | 'config' | 'user'
  status: 'success' | 'failed' | 'in_progress' | 'scheduled'
  title: string
  description: string
  timestamp: Date
  metadata?: {
    database?: string
    size?: string
    duration?: string
    user?: string
    [key: string]: any
  }
}

interface ActivityTimelineProps {
  events: ActivityEvent[]
  maxItems?: number
  showFilters?: boolean
  className?: string
}

const eventTypeConfig = {
  backup: {
    icon: Database,
    color: 'text-blue-600',
    bgColor: 'bg-blue-100',
    borderColor: 'border-blue-200',
  },
  restore: {
    icon: RefreshCw,
    color: 'text-purple-600',
    bgColor: 'bg-purple-100',
    borderColor: 'border-purple-200',
  },
  schedule: {
    icon: Calendar,
    color: 'text-indigo-600',
    bgColor: 'bg-indigo-100',
    borderColor: 'border-indigo-200',
  },
  error: {
    icon: XCircle,
    color: 'text-red-600',
    bgColor: 'bg-red-100',
    borderColor: 'border-red-200',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-yellow-600',
    bgColor: 'bg-yellow-100',
    borderColor: 'border-yellow-200',
  },
  config: {
    icon: Settings,
    color: 'text-gray-600',
    bgColor: 'bg-gray-100',
    borderColor: 'border-gray-200',
  },
  user: {
    icon: User,
    color: 'text-green-600',
    bgColor: 'bg-green-100',
    borderColor: 'border-green-200',
  },
}

const statusConfig = {
  success: {
    icon: CheckCircle2,
    color: 'text-green-600',
    label: 'Completed',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-600',
    label: 'Failed',
  },
  in_progress: {
    icon: Clock,
    color: 'text-blue-600',
    label: 'In Progress',
  },
  scheduled: {
    icon: Calendar,
    color: 'text-gray-600',
    label: 'Scheduled',
  },
}

export function ActivityTimeline({
  events,
  maxItems = 10,
  showFilters = true,
  className,
}: ActivityTimelineProps) {
  const [filter, setFilter] = React.useState<string>('all')
  const [statusFilter, setStatusFilter] = React.useState<string>('all')

  const filteredEvents = React.useMemo(() => {
    let filtered = events

    if (filter !== 'all') {
      filtered = filtered.filter((event) => event.type === filter)
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter((event) => event.status === statusFilter)
    }

    return filtered.slice(0, maxItems)
  }, [events, filter, statusFilter, maxItems])

  return (
    <div className={clsx('bg-white rounded-lg shadow-sm border border-gray-200', className)}>
      <div className="px-6 py-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">Activity Timeline</h3>
          <span className="text-sm text-gray-500">{events.length} events</span>
        </div>

        {showFilters && (
          <div className="mt-4 flex flex-wrap gap-2">
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-700">Type:</label>
              <select
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Types</option>
                <option value="backup">Backups</option>
                <option value="restore">Restores</option>
                <option value="schedule">Schedules</option>
                <option value="error">Errors</option>
                <option value="warning">Warnings</option>
                <option value="config">Config</option>
                <option value="user">User</option>
              </select>
            </div>

            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-700">Status:</label>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="text-sm border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Statuses</option>
                <option value="success">Success</option>
                <option value="failed">Failed</option>
                <option value="in_progress">In Progress</option>
                <option value="scheduled">Scheduled</option>
              </select>
            </div>
          </div>
        )}
      </div>

      <div className="divide-y divide-gray-100">
        {filteredEvents.length === 0 ? (
          <div className="px-6 py-12 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
              <Clock className="w-8 h-8 text-gray-400" />
            </div>
            <p className="text-gray-500">No activities to display</p>
          </div>
        ) : (
          filteredEvents.map((event, index) => (
            <ActivityTimelineItem
              key={event.id}
              event={event}
              isLast={index === filteredEvents.length - 1}
            />
          ))
        )}
      </div>

      {filteredEvents.length < events.length && (
        <div className="px-6 py-4 border-t border-gray-200 text-center">
          <button className="text-sm text-blue-600 hover:text-blue-700 font-medium">
            View All Activities ({events.length - filteredEvents.length} more)
          </button>
        </div>
      )}
    </div>
  )
}

interface ActivityTimelineItemProps {
  event: ActivityEvent
  isLast: boolean
}

function ActivityTimelineItem({ event, isLast }: ActivityTimelineItemProps) {
  const typeConfig = eventTypeConfig[event.type]
  const statusInfo = statusConfig[event.status]
  const TypeIcon = typeConfig.icon
  const StatusIcon = statusInfo.icon

  return (
    <div className="relative px-6 py-4 hover:bg-gray-50 transition-colors">
      {/* Timeline line */}
      {!isLast && (
        <div className="absolute left-[52px] top-16 bottom-0 w-0.5 bg-gray-200" />
      )}

      <div className="flex gap-4">
        {/* Icon */}
        <div
          className={clsx(
            'flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center border-2',
            typeConfig.bgColor,
            typeConfig.borderColor
          )}
        >
          <TypeIcon className={clsx('w-5 h-5', typeConfig.color)} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h4 className="text-sm font-semibold text-gray-900">{event.title}</h4>
                <span
                  className={clsx(
                    'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                    event.status === 'success' && 'bg-green-100 text-green-700',
                    event.status === 'failed' && 'bg-red-100 text-red-700',
                    event.status === 'in_progress' && 'bg-blue-100 text-blue-700',
                    event.status === 'scheduled' && 'bg-gray-100 text-gray-700'
                  )}
                >
                  <StatusIcon className="w-3 h-3" />
                  {statusInfo.label}
                </span>
              </div>
              <p className="mt-1 text-sm text-gray-600">{event.description}</p>

              {/* Metadata */}
              {event.metadata && (
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
                  {event.metadata.database && (
                    <span className="flex items-center gap-1">
                      <Server className="w-3 h-3" />
                      {event.metadata.database}
                    </span>
                  )}
                  {event.metadata.size && (
                    <span className="flex items-center gap-1">
                      <Database className="w-3 h-3" />
                      {event.metadata.size}
                    </span>
                  )}
                  {event.metadata.duration && (
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {event.metadata.duration}
                    </span>
                  )}
                  {event.metadata.user && (
                    <span className="flex items-center gap-1">
                      <User className="w-3 h-3" />
                      {event.metadata.user}
                    </span>
                  )}
                </div>
              )}
            </div>

            {/* Timestamp */}
            <time className="text-xs text-gray-500 whitespace-nowrap">
              {formatDistanceToNow(event.timestamp, { addSuffix: true })}
            </time>
          </div>
        </div>
      </div>
    </div>
  )
}

// Sample data generator for testing
export function generateSampleActivities(count: number = 10): ActivityEvent[] {
  const types: ActivityEvent['type'][] = ['backup', 'restore', 'schedule', 'error', 'warning', 'config', 'user']
  const statuses: ActivityEvent['status'][] = ['success', 'failed', 'in_progress', 'scheduled']

  return Array.from({ length: count }, (_, i) => {
    const type = types[Math.floor(Math.random() * types.length)]
    const status = statuses[Math.floor(Math.random() * statuses.length)]

    return {
      id: `activity-${i}`,
      type,
      status,
      title: `${type.charAt(0).toUpperCase() + type.slice(1)} Operation ${i + 1}`,
      description: `Sample ${type} operation description with details about what happened.`,
      timestamp: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000), // Last 7 days
      metadata: {
        database: `database-${Math.floor(Math.random() * 5) + 1}`,
        size: `${Math.floor(Math.random() * 1000)}MB`,
        duration: `${Math.floor(Math.random() * 60)}m ${Math.floor(Math.random() * 60)}s`,
        user: `user${Math.floor(Math.random() * 10) + 1}@example.com`,
      },
    }
  }).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
}
