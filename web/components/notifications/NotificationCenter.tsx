'use client'

import React, { useState, useEffect, useCallback, useRef } from 'react'
import { Bell, X, Check, Clock, Filter, Settings, ChevronDown, Trash2, Archive } from 'lucide-react'
import { getNotificationsWsUrl } from '@/lib/notifications-ws'

// Types
interface Notification {
  id: string
  type: string
  priority: 'low' | 'normal' | 'high' | 'urgent'
  title: string
  message: string
  category: string
  tags: string[]
  metadata: Record<string, any>
  channels: string[]
  status: 'pending' | 'queued' | 'sending' | 'delivered' | 'failed' | 'read' | 'dismissed' | 'snoozed'
  actions: NotificationAction[]
  created_at: string
  scheduled_for?: string
  delivered_at?: string
  read_at?: string
  expires_at?: string
  snoozed_until?: string
  group_id?: string
  thread_id?: string
  user_id: string
  image_url?: string
  video_url?: string
  chart_data?: Record<string, any>
  relevance_score: number
  engagement_score: number
}

interface NotificationAction {
  id: string
  label: string
  type: string
  action: string
  url?: string
  metadata?: Record<string, any>
  style: 'primary' | 'secondary' | 'danger'
}

interface NotificationGroup {
  id: string
  type: string
  category: string
  title: string
  summary: string
  count: number
  priority: 'low' | 'normal' | 'high' | 'urgent'
  notifications: Notification[]
  created_at: string
  updated_at: string
}

interface NotificationPreferences {
  enabled_channels: string[]
  disabled_types: string[]
  minimum_priority: string
  quiet_hours: any[]
  do_not_disturb: boolean
  grouping_enabled: boolean
  batching_enabled: boolean
  batch_interval: number
  digest_enabled: boolean
  digest_frequency: string
}

// Main Component
export default function NotificationCenter() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [groups, setGroups] = useState<NotificationGroup[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [unreadCount, setUnreadCount] = useState(0)
  const [filter, setFilter] = useState<'all' | 'unread' | 'read'>('all')
  const [categoryFilter, setCategoryFilter] = useState<string>('all')
  const [priorityFilter, setPriorityFilter] = useState<string>('all')
  const [showSettings, setShowSettings] = useState(false)
  const [preferences, setPreferences] = useState<NotificationPreferences | null>(null)
  const [ws, setWs] = useState<WebSocket | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  // WebSocket connection for real-time updates
  useEffect(() => {
    const connectWebSocket = () => {
      const wsUrl = getNotificationsWsUrl()
      if (!wsUrl) {
        return
      }

      const socket = new WebSocket(wsUrl)

      socket.onopen = () => {
        console.log('WebSocket connected')
      }

      socket.onmessage = (event) => {
        const data = JSON.parse(event.data)

        if (data.type === 'notification') {
          handleNewNotification(data.notification)
        } else if (data.type === 'update') {
          handleNotificationUpdate(data.notification)
        } else if (data.type === 'delete') {
          handleNotificationDelete(data.id)
        }
      }

      socket.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      socket.onclose = () => {
        console.log('WebSocket disconnected, reconnecting...')
        setTimeout(connectWebSocket, 5000)
      }

      wsRef.current = socket
      setWs(socket)
    }

    connectWebSocket()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  // Fetch notifications on mount
  useEffect(() => {
    fetchNotifications()
    fetchPreferences()
  }, [])

  // Update unread count
  useEffect(() => {
    const count = notifications.filter(n => n.status !== 'read' && n.status !== 'dismissed').length
    setUnreadCount(count)
  }, [notifications])

  // Fetch notifications from API
  const fetchNotifications = async () => {
    try {
      const response = await fetch('/api/notifications')
      const data = await response.json()
      setNotifications(data.notifications || [])
      setGroups(data.groups || [])
    } catch (error) {
      console.error('Error fetching notifications:', error)
    }
  }

  // Fetch user preferences
  const fetchPreferences = async () => {
    try {
      const response = await fetch('/api/notifications/preferences')
      const data = await response.json()
      setPreferences(data)
    } catch (error) {
      console.error('Error fetching preferences:', error)
    }
  }

  // Handle new notification
  const handleNewNotification = (notification: Notification) => {
    setNotifications(prev => [notification, ...prev])

    // Show toast for high/urgent priority
    if (notification.priority === 'high' || notification.priority === 'urgent') {
      showToast(notification)
    }

    // Play sound if enabled
    if (preferences && !preferences.do_not_disturb) {
      playNotificationSound()
    }
  }

  // Handle notification update
  const handleNotificationUpdate = (notification: Notification) => {
    setNotifications(prev =>
      prev.map(n => (n.id === notification.id ? notification : n))
    )
  }

  // Handle notification delete
  const handleNotificationDelete = (id: string) => {
    setNotifications(prev => prev.filter(n => n.id !== id))
  }

  // Mark notification as read
  const markAsRead = async (id: string) => {
    try {
      await fetch(`/api/notifications/${id}/read`, { method: 'POST' })

      setNotifications(prev =>
        prev.map(n =>
          n.id === id
            ? { ...n, status: 'read' as const, read_at: new Date().toISOString() }
            : n
        )
      )
    } catch (error) {
      console.error('Error marking as read:', error)
    }
  }

  // Mark all as read
  const markAllAsRead = async () => {
    try {
      await fetch('/api/notifications/mark-all-read', { method: 'POST' })

      setNotifications(prev =>
        prev.map(n => ({
          ...n,
          status: 'read' as const,
          read_at: new Date().toISOString()
        }))
      )
    } catch (error) {
      console.error('Error marking all as read:', error)
    }
  }

  // Dismiss notification
  const dismissNotification = async (id: string) => {
    try {
      await fetch(`/api/notifications/${id}/dismiss`, { method: 'POST' })

      setNotifications(prev =>
        prev.map(n =>
          n.id === id ? { ...n, status: 'dismissed' as const } : n
        )
      )
    } catch (error) {
      console.error('Error dismissing notification:', error)
    }
  }

  // Snooze notification
  const snoozeNotification = async (id: string, duration: number) => {
    try {
      await fetch(`/api/notifications/${id}/snooze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ duration })
      })

      const snoozedUntil = new Date(Date.now() + duration * 60 * 1000).toISOString()

      setNotifications(prev =>
        prev.map(n =>
          n.id === id
            ? { ...n, status: 'snoozed' as const, snoozed_until: snoozedUntil }
            : n
        )
      )
    } catch (error) {
      console.error('Error snoozing notification:', error)
    }
  }

  // Handle action click
  const handleAction = async (notification: Notification, action: NotificationAction) => {
    try {
      await fetch(`/api/notifications/${notification.id}/action`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action_id: action.id })
      })

      if (action.url) {
        window.location.href = action.url
      }
    } catch (error) {
      console.error('Error handling action:', error)
    }
  }

  // Show toast notification
  const showToast = (notification: Notification) => {
    // Would integrate with a toast library like react-hot-toast
    console.log('Toast:', notification.title)
  }

  // Play notification sound
  const playNotificationSound = () => {
    const audio = new Audio('/sounds/notification.mp3')
    audio.play().catch(console.error)
  }

  // Filter notifications
  const filteredNotifications = notifications.filter(n => {
    if (filter === 'unread' && (n.status === 'read' || n.status === 'dismissed')) return false
    if (filter === 'read' && n.status !== 'read') return false
    if (categoryFilter !== 'all' && n.category !== categoryFilter) return false
    if (priorityFilter !== 'all' && n.priority !== priorityFilter) return false
    return true
  })

  // Get priority color
  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent':
        return 'bg-red-100 text-red-800 border-red-300'
      case 'high':
        return 'bg-orange-100 text-orange-800 border-orange-300'
      case 'normal':
        return 'bg-blue-100 text-blue-800 border-blue-300'
      case 'low':
        return 'bg-gray-100 text-gray-800 border-gray-300'
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300'
    }
  }

  // Format time ago
  const timeAgo = (dateString: string) => {
    const date = new Date(dateString)
    const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000)

    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`
    return date.toLocaleDateString()
  }

  return (
    <>
      {/* Bell Icon with Badge */}
      <div className="relative">
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="relative p-2 rounded-full hover:bg-gray-100 transition-colors"
          aria-label="Notifications"
        >
          <Bell className="h-6 w-6 text-gray-600" />
          {unreadCount > 0 && (
            <span className="absolute top-0 right-0 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs font-bold text-white">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </button>
      </div>

      {/* Notification Panel */}
      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-start justify-end pt-16 pr-4">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black bg-opacity-25"
            onClick={() => setIsOpen(false)}
          />

          {/* Panel */}
          <div className="relative z-50 w-full max-w-md rounded-lg bg-white shadow-2xl">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-gray-200 p-4">
              <div className="flex items-center gap-2">
                <Bell className="h-5 w-5 text-gray-600" />
                <h2 className="text-lg font-semibold text-gray-900">Notifications</h2>
                {unreadCount > 0 && (
                  <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800">
                    {unreadCount} new
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setShowSettings(!showSettings)}
                  className="rounded p-1 hover:bg-gray-100"
                  aria-label="Settings"
                >
                  <Settings className="h-5 w-5 text-gray-600" />
                </button>
                <button
                  onClick={() => setIsOpen(false)}
                  className="rounded p-1 hover:bg-gray-100"
                  aria-label="Close"
                >
                  <X className="h-5 w-5 text-gray-600" />
                </button>
              </div>
            </div>

            {/* Filters */}
            <div className="border-b border-gray-200 p-4">
              <div className="flex gap-2">
                <button
                  onClick={() => setFilter('all')}
                  className={`rounded px-3 py-1 text-sm font-medium transition-colors ${
                    filter === 'all'
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  All
                </button>
                <button
                  onClick={() => setFilter('unread')}
                  className={`rounded px-3 py-1 text-sm font-medium transition-colors ${
                    filter === 'unread'
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  Unread
                </button>
                <button
                  onClick={() => setFilter('read')}
                  className={`rounded px-3 py-1 text-sm font-medium transition-colors ${
                    filter === 'read'
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  Read
                </button>
                {unreadCount > 0 && (
                  <button
                    onClick={markAllAsRead}
                    className="ml-auto rounded px-3 py-1 text-sm font-medium text-blue-600 hover:bg-blue-50"
                  >
                    Mark all read
                  </button>
                )}
              </div>
            </div>

            {/* Notification List */}
            <div className="max-h-[calc(100vh-16rem)] overflow-y-auto">
              {filteredNotifications.length === 0 ? (
                <div className="p-8 text-center">
                  <Bell className="mx-auto h-12 w-12 text-gray-300" />
                  <p className="mt-2 text-sm text-gray-500">No notifications</p>
                </div>
              ) : (
                <div className="divide-y divide-gray-100">
                  {filteredNotifications.map(notification => (
                    <NotificationItem
                      key={notification.id}
                      notification={notification}
                      onMarkAsRead={markAsRead}
                      onDismiss={dismissNotification}
                      onSnooze={snoozeNotification}
                      onAction={handleAction}
                      getPriorityColor={getPriorityColor}
                      timeAgo={timeAgo}
                    />
                  ))}
                </div>
              )}
            </div>

            {/* Settings Panel */}
            {showSettings && preferences && (
              <NotificationSettings
                preferences={preferences}
                onClose={() => setShowSettings(false)}
                onSave={(newPrefs) => {
                  setPreferences(newPrefs)
                  setShowSettings(false)
                }}
              />
            )}
          </div>
        </div>
      )}
    </>
  )
}

// Notification Item Component
interface NotificationItemProps {
  notification: Notification
  onMarkAsRead: (id: string) => void
  onDismiss: (id: string) => void
  onSnooze: (id: string, duration: number) => void
  onAction: (notification: Notification, action: NotificationAction) => void
  getPriorityColor: (priority: string) => string
  timeAgo: (date: string) => string
}

function NotificationItem({
  notification,
  onMarkAsRead,
  onDismiss,
  onSnooze,
  onAction,
  getPriorityColor,
  timeAgo
}: NotificationItemProps) {
  const [showActions, setShowActions] = useState(false)
  const isUnread = notification.status !== 'read' && notification.status !== 'dismissed'

  return (
    <div
      className={`group relative p-4 transition-colors hover:bg-gray-50 ${
        isUnread ? 'bg-blue-50' : ''
      }`}
    >
      {/* Priority Badge */}
      <div className="mb-2 flex items-center gap-2">
        <span
          className={`rounded-full border px-2 py-0.5 text-xs font-medium ${getPriorityColor(
            notification.priority
          )}`}
        >
          {notification.priority}
        </span>
        <span className="text-xs text-gray-500">{timeAgo(notification.created_at)}</span>
        {isUnread && (
          <span className="ml-auto h-2 w-2 rounded-full bg-blue-500" />
        )}
      </div>

      {/* Title and Message */}
      <h3 className="mb-1 font-medium text-gray-900">{notification.title}</h3>
      <p className="mb-2 text-sm text-gray-600">{notification.message}</p>

      {/* Image/Chart */}
      {notification.image_url && (
        <img
          src={notification.image_url}
          alt=""
          className="mb-2 h-32 w-full rounded object-cover"
        />
      )}

      {/* Actions */}
      {notification.actions && notification.actions.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {notification.actions.map(action => (
            <button
              key={action.id}
              onClick={() => onAction(notification, action)}
              className={`rounded px-3 py-1 text-sm font-medium transition-colors ${
                action.style === 'primary'
                  ? 'bg-blue-600 text-white hover:bg-blue-700'
                  : action.style === 'danger'
                  ? 'bg-red-600 text-white hover:bg-red-700'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {action.label}
            </button>
          ))}
        </div>
      )}

      {/* Quick Actions */}
      <div className="mt-3 flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
        {isUnread && (
          <button
            onClick={() => onMarkAsRead(notification.id)}
            className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-200"
          >
            <Check className="h-3 w-3" />
            Mark read
          </button>
        )}
        <button
          onClick={() => setShowActions(!showActions)}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-200"
        >
          <Clock className="h-3 w-3" />
          Snooze
        </button>
        <button
          onClick={() => onDismiss(notification.id)}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-200"
        >
          <Trash2 className="h-3 w-3" />
          Dismiss
        </button>
      </div>

      {/* Snooze Options */}
      {showActions && (
        <div className="mt-2 flex gap-2">
          <button
            onClick={() => onSnooze(notification.id, 15)}
            className="rounded bg-gray-100 px-2 py-1 text-xs hover:bg-gray-200"
          >
            15m
          </button>
          <button
            onClick={() => onSnooze(notification.id, 60)}
            className="rounded bg-gray-100 px-2 py-1 text-xs hover:bg-gray-200"
          >
            1h
          </button>
          <button
            onClick={() => onSnooze(notification.id, 240)}
            className="rounded bg-gray-100 px-2 py-1 text-xs hover:bg-gray-200"
          >
            4h
          </button>
          <button
            onClick={() => onSnooze(notification.id, 1440)}
            className="rounded bg-gray-100 px-2 py-1 text-xs hover:bg-gray-200"
          >
            1d
          </button>
        </div>
      )}
    </div>
  )
}

// Settings Component
interface NotificationSettingsProps {
  preferences: NotificationPreferences
  onClose: () => void
  onSave: (preferences: NotificationPreferences) => void
}

function NotificationSettings({ preferences, onClose, onSave }: NotificationSettingsProps) {
  const [localPrefs, setLocalPrefs] = useState(preferences)

  const handleSave = async () => {
    try {
      await fetch('/api/notifications/preferences', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(localPrefs)
      })
      onSave(localPrefs)
    } catch (error) {
      console.error('Error saving preferences:', error)
    }
  }

  return (
    <div className="border-t border-gray-200 bg-gray-50 p-4">
      <h3 className="mb-4 font-semibold text-gray-900">Notification Settings</h3>

      <div className="space-y-4">
        {/* Do Not Disturb */}
        <label className="flex items-center justify-between">
          <span className="text-sm text-gray-700">Do Not Disturb</span>
          <input
            type="checkbox"
            checked={localPrefs.do_not_disturb}
            onChange={(e) =>
              setLocalPrefs({ ...localPrefs, do_not_disturb: e.target.checked })
            }
            className="rounded"
          />
        </label>

        {/* Grouping */}
        <label className="flex items-center justify-between">
          <span className="text-sm text-gray-700">Group similar notifications</span>
          <input
            type="checkbox"
            checked={localPrefs.grouping_enabled}
            onChange={(e) =>
              setLocalPrefs({ ...localPrefs, grouping_enabled: e.target.checked })
            }
            className="rounded"
          />
        </label>

        {/* Batching */}
        <label className="flex items-center justify-between">
          <span className="text-sm text-gray-700">Batch notifications</span>
          <input
            type="checkbox"
            checked={localPrefs.batching_enabled}
            onChange={(e) =>
              setLocalPrefs({ ...localPrefs, batching_enabled: e.target.checked })
            }
            className="rounded"
          />
        </label>

        {/* Digest */}
        <label className="flex items-center justify-between">
          <span className="text-sm text-gray-700">Daily digest</span>
          <input
            type="checkbox"
            checked={localPrefs.digest_enabled}
            onChange={(e) =>
              setLocalPrefs({ ...localPrefs, digest_enabled: e.target.checked })
            }
            className="rounded"
          />
        </label>
      </div>

      <div className="mt-4 flex gap-2">
        <button
          onClick={handleSave}
          className="flex-1 rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Save
        </button>
        <button
          onClick={onClose}
          className="flex-1 rounded bg-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-300"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
