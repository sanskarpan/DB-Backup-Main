import { useState, useEffect, useCallback, useRef } from 'react'
import { getNotificationsWsUrl } from '@/lib/notifications-ws'

interface Notification {
  id: string
  type: string
  priority: 'low' | 'normal' | 'high' | 'urgent'
  title: string
  message: string
  category: string
  status: string
  created_at: string
  read_at?: string
  [key: string]: any
}

interface UseNotificationsOptions {
  autoConnect?: boolean
  onNewNotification?: (notification: Notification) => void
  onUpdate?: (notification: Notification) => void
  onDelete?: (id: string) => void
}

export function useNotifications(options: UseNotificationsOptions = {}) {
  const {
    autoConnect = true,
    onNewNotification,
    onUpdate,
    onDelete
  } = options

  const [notifications, setNotifications] = useState<Notification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>()

  // Fetch notifications from API
  const fetchNotifications = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/notifications')

      if (!response.ok) {
        throw new Error('Failed to fetch notifications')
      }

      const data = await response.json()
      setNotifications(data.notifications || [])
      setError(null)
    } catch (err) {
      setError(err as Error)
      console.error('Error fetching notifications:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    const wsUrl = getNotificationsWsUrl()
    if (!wsUrl) {
      return
    }

    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('WebSocket connected')
      setIsConnected(true)
      setError(null)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)

        if (data.type === 'notification') {
          const notification = data.notification
          setNotifications(prev => [notification, ...prev])
          onNewNotification?.(notification)
        } else if (data.type === 'update') {
          const notification = data.notification
          setNotifications(prev =>
            prev.map(n => (n.id === notification.id ? notification : n))
          )
          onUpdate?.(notification)
        } else if (data.type === 'delete') {
          const id = data.id
          setNotifications(prev => prev.filter(n => n.id !== id))
          onDelete?.(id)
        }
      } catch (err) {
        console.error('Error parsing WebSocket message:', err)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      setError(new Error('WebSocket connection error'))
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      setIsConnected(false)

      // Attempt to reconnect after 5 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('Attempting to reconnect...')
        connect()
      }, 5000)
    }

    wsRef.current = ws
  }, [onNewNotification, onUpdate, onDelete])

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setIsConnected(false)
  }, [])

  // Mark notification as read
  const markAsRead = useCallback(async (id: string) => {
    try {
      const response = await fetch(`/api/notifications/${id}/read`, {
        method: 'POST'
      })

      if (!response.ok) {
        throw new Error('Failed to mark notification as read')
      }

      setNotifications(prev =>
        prev.map(n =>
          n.id === id
            ? { ...n, status: 'read', read_at: new Date().toISOString() }
            : n
        )
      )
    } catch (err) {
      console.error('Error marking notification as read:', err)
      throw err
    }
  }, [])

  // Mark all notifications as read
  const markAllAsRead = useCallback(async () => {
    try {
      const response = await fetch('/api/notifications/mark-all-read', {
        method: 'POST'
      })

      if (!response.ok) {
        throw new Error('Failed to mark all notifications as read')
      }

      setNotifications(prev =>
        prev.map(n => ({
          ...n,
          status: 'read',
          read_at: new Date().toISOString()
        }))
      )
    } catch (err) {
      console.error('Error marking all notifications as read:', err)
      throw err
    }
  }, [])

  // Dismiss notification
  const dismiss = useCallback(async (id: string) => {
    try {
      const response = await fetch(`/api/notifications/${id}/dismiss`, {
        method: 'POST'
      })

      if (!response.ok) {
        throw new Error('Failed to dismiss notification')
      }

      setNotifications(prev =>
        prev.map(n =>
          n.id === id ? { ...n, status: 'dismissed' } : n
        )
      )
    } catch (err) {
      console.error('Error dismissing notification:', err)
      throw err
    }
  }, [])

  // Snooze notification
  const snooze = useCallback(async (id: string, duration: number) => {
    try {
      const response = await fetch(`/api/notifications/${id}/snooze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ duration })
      })

      if (!response.ok) {
        throw new Error('Failed to snooze notification')
      }

      const snoozedUntil = new Date(Date.now() + duration * 60 * 1000).toISOString()

      setNotifications(prev =>
        prev.map(n =>
          n.id === id
            ? { ...n, status: 'snoozed', snoozed_until: snoozedUntil }
            : n
        )
      )
    } catch (err) {
      console.error('Error snoozing notification:', err)
      throw err
    }
  }, [])

  // Perform action on notification
  const performAction = useCallback(async (id: string, actionId: string) => {
    try {
      const response = await fetch(`/api/notifications/${id}/action`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action_id: actionId })
      })

      if (!response.ok) {
        throw new Error('Failed to perform action')
      }

      const data = await response.json()
      return data
    } catch (err) {
      console.error('Error performing action:', err)
      throw err
    }
  }, [])

  // Update unread count
  useEffect(() => {
    const count = notifications.filter(
      n => n.status !== 'read' && n.status !== 'dismissed'
    ).length
    setUnreadCount(count)
  }, [notifications])

  // Initial fetch and connection
  useEffect(() => {
    fetchNotifications()

    if (autoConnect) {
      connect()
    }

    return () => {
      disconnect()
    }
  }, [autoConnect, connect, disconnect, fetchNotifications])

  return {
    notifications,
    unreadCount,
    isConnected,
    isLoading,
    error,

    // Actions
    fetchNotifications,
    markAsRead,
    markAllAsRead,
    dismiss,
    snooze,
    performAction,

    // Connection
    connect,
    disconnect
  }
}

// Hook for notification preferences
export function useNotificationPreferences() {
  const [preferences, setPreferences] = useState<any>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/notifications/preferences')

      if (!response.ok) {
        throw new Error('Failed to fetch preferences')
      }

      const data = await response.json()
      setPreferences(data)
      setError(null)
    } catch (err) {
      setError(err as Error)
      console.error('Error fetching preferences:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  const updatePreferences = useCallback(async (newPreferences: any) => {
    try {
      const response = await fetch('/api/notifications/preferences', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newPreferences)
      })

      if (!response.ok) {
        throw new Error('Failed to update preferences')
      }

      const data = await response.json()
      setPreferences(data)
      return data
    } catch (err) {
      console.error('Error updating preferences:', err)
      throw err
    }
  }, [])

  useEffect(() => {
    fetchPreferences()
  }, [fetchPreferences])

  return {
    preferences,
    isLoading,
    error,
    updatePreferences,
    refresh: fetchPreferences
  }
}

// Hook for notification analytics
export function useNotificationAnalytics() {
  const [stats, setStats] = useState<any>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchStats = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/notifications/stats')

      if (!response.ok) {
        throw new Error('Failed to fetch stats')
      }

      const data = await response.json()
      setStats(data)
      setError(null)
    } catch (err) {
      setError(err as Error)
      console.error('Error fetching stats:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  const getEngagement = useCallback(async (userId: string) => {
    try {
      const response = await fetch(`/api/notifications/analytics/engagement/${userId}`)

      if (!response.ok) {
        throw new Error('Failed to fetch engagement')
      }

      return await response.json()
    } catch (err) {
      console.error('Error fetching engagement:', err)
      throw err
    }
  }, [])

  useEffect(() => {
    fetchStats()
  }, [fetchStats])

  return {
    stats,
    isLoading,
    error,
    getEngagement,
    refresh: fetchStats
  }
}
