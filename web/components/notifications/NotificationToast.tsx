'use client'

import React, { useEffect, useState } from 'react'
import { X, AlertCircle, CheckCircle, Info, AlertTriangle } from 'lucide-react'

interface ToastNotification {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
}

interface NotificationToastProps {
  notification: ToastNotification
  onClose: (id: string) => void
}

export function NotificationToast({ notification, onClose }: NotificationToastProps) {
  const [isExiting, setIsExiting] = useState(false)

  useEffect(() => {
    const duration = notification.duration || 5000

    const timer = setTimeout(() => {
      handleClose()
    }, duration)

    return () => clearTimeout(timer)
  }, [notification])

  const handleClose = () => {
    setIsExiting(true)
    setTimeout(() => {
      onClose(notification.id)
    }, 300) // Match animation duration
  }

  const getIcon = () => {
    switch (notification.type) {
      case 'success':
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />
      case 'warning':
        return <AlertTriangle className="h-5 w-5 text-yellow-500" />
      case 'info':
      default:
        return <Info className="h-5 w-5 text-blue-500" />
    }
  }

  const getBgColor = () => {
    switch (notification.type) {
      case 'success':
        return 'bg-green-50 border-green-200'
      case 'error':
        return 'bg-red-50 border-red-200'
      case 'warning':
        return 'bg-yellow-50 border-yellow-200'
      case 'info':
      default:
        return 'bg-blue-50 border-blue-200'
    }
  }

  return (
    <div
      className={`pointer-events-auto mb-4 w-full max-w-sm overflow-hidden rounded-lg border shadow-lg transition-all duration-300 ${
        isExiting
          ? 'translate-x-full opacity-0'
          : 'translate-x-0 opacity-100'
      } ${getBgColor()}`}
    >
      <div className="flex items-start p-4">
        <div className="flex-shrink-0">{getIcon()}</div>

        <div className="ml-3 flex-1">
          <p className="text-sm font-medium text-gray-900">{notification.title}</p>
          <p className="mt-1 text-sm text-gray-600">{notification.message}</p>

          {notification.action && (
            <button
              onClick={() => {
                notification.action?.onClick()
                handleClose()
              }}
              className="mt-2 text-sm font-medium text-blue-600 hover:text-blue-500"
            >
              {notification.action.label}
            </button>
          )}
        </div>

        <button
          onClick={handleClose}
          className="ml-4 flex-shrink-0 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Progress bar */}
      <div className="h-1 w-full bg-gray-200">
        <div
          className="h-full bg-gray-400 transition-all"
          style={{
            width: '100%',
            animation: `shrink ${notification.duration || 5000}ms linear forwards`
          }}
        />
      </div>

      <style jsx>{`
        @keyframes shrink {
          from {
            width: 100%;
          }
          to {
            width: 0%;
          }
        }
      `}</style>
    </div>
  )
}

// Toast Container
export function ToastContainer() {
  const [toasts, setToasts] = useState<ToastNotification[]>([])

  useEffect(() => {
    // Listen for custom toast events
    const handleToast = (event: CustomEvent<ToastNotification>) => {
      setToasts(prev => [...prev, event.detail])
    }

    window.addEventListener('show-toast' as any, handleToast)

    return () => {
      window.removeEventListener('show-toast' as any, handleToast)
    }
  }, [])

  const handleClose = (id: string) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }

  return (
    <div className="pointer-events-none fixed bottom-0 right-0 z-50 flex flex-col items-end p-6">
      {toasts.map(toast => (
        <NotificationToast key={toast.id} notification={toast} onClose={handleClose} />
      ))}
    </div>
  )
}

// Helper function to show toast
export function showToast(toast: Omit<ToastNotification, 'id'>) {
  const event = new CustomEvent('show-toast', {
    detail: {
      ...toast,
      id: Math.random().toString(36).substr(2, 9)
    }
  })
  window.dispatchEvent(event)
}
