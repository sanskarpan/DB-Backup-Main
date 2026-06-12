import type { Metadata, Viewport } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { QueryProvider } from '@/components/providers/query-provider'
import { PWAProvider } from '@/components/providers/pwa-provider'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { InstallPrompt } from '@/components/pwa/install-prompt'
import { OfflineIndicator } from '@/components/pwa/offline-indicator'
import { UpdateNotification } from '@/components/pwa/update-notification'
import { NotificationPermission } from '@/components/pwa/notification-permission'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'DB Backup Manager - Enterprise Backup Solution',
  description: 'Enterprise-grade database backup and recovery management system with real-time monitoring, compliance tracking, and disaster recovery capabilities',
  manifest: '/manifest.json',
  applicationName: 'DB Backup Manager',
  appleWebApp: {
    capable: true,
    statusBarStyle: 'default',
    title: 'DB Backup Manager',
  },
  formatDetection: {
    telephone: false,
  },
  icons: {
    icon: '/icons/icon-192x192.png',
    apple: '/icons/icon-192x192.png',
  },
}

export const viewport: Viewport = {
  themeColor: '#3b82f6',
  width: 'device-width',
  initialScale: 1,
  maximumScale: 5,
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <head>
        <link rel="manifest" href="/manifest.json" />
        <link rel="icon" href="/icons/icon-192x192.png" />
        <link rel="apple-touch-icon" href="/icons/icon-192x192.png" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content="DB Backup" />
        <meta name="mobile-web-app-capable" content="yes" />
        <meta name="theme-color" content="#3b82f6" />
      </head>
      <body className={inter.className}>
        <QueryProvider>
          <PWAProvider>
            <div className="flex h-screen overflow-hidden">
              <Sidebar />
              <div className="flex-1 flex flex-col overflow-hidden">
                <Header />
                <main className="flex-1 overflow-y-auto bg-gray-50 p-6">
                  {children}
                </main>
              </div>
            </div>

            {/* PWA Components */}
            <InstallPrompt />
            <OfflineIndicator />
            <UpdateNotification />
            <NotificationPermission />
          </PWAProvider>
        </QueryProvider>
      </body>
    </html>
  )
}
