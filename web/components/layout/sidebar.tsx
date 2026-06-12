'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  Database,
  HardDrive,
  Calendar,
  Settings,
  LayoutDashboard,
  GitBranch,
  Shield,
  BarChart3,
  Cloud,
  TrendingUp,
  Archive,
  Eye,
  Sparkles,
} from 'lucide-react'
import { useState } from 'react'

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard, badge: null },
  { name: 'Backups', href: '/backups', icon: Archive, badge: '12' },
  { name: 'Databases', href: '/databases', icon: Database, badge: null },
  { name: 'Schedules', href: '/schedules', icon: Calendar, badge: '8' },
  { name: 'Restore', href: '/restore', icon: GitBranch, badge: null },
  { name: 'Security', href: '/security', icon: Shield, badge: null },
  { name: 'Monitoring', href: '/monitoring', icon: BarChart3, badge: null },
  { name: 'Storage', href: '/storage-providers', icon: Cloud, badge: null },
  { name: 'Visualizations', href: '/visualizations', icon: TrendingUp, badge: null },
  { name: 'Settings', href: '/settings', icon: Settings, badge: null },
]

export function Sidebar() {
  const pathname = usePathname()
  const [isCollapsed, setIsCollapsed] = useState(false)

  return (
    <div className={`flex flex-col bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 border-r border-gray-800 transition-all duration-300 ${isCollapsed ? 'w-20' : 'w-72'}`}>
      {/* Logo Section */}
      <div className="h-20 flex items-center justify-between px-6 border-b border-gray-800 bg-gradient-to-r from-orange-600/10 to-transparent">
        {!isCollapsed && (
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-orange-500 to-orange-600 flex items-center justify-center shadow-lg shadow-orange-500/30">
              <HardDrive className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-white">DB Backup</h1>
              <p className="text-xs text-gray-400">Enterprise</p>
            </div>
          </div>
        )}
        {isCollapsed && (
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-orange-500 to-orange-600 flex items-center justify-center shadow-lg shadow-orange-500/30 mx-auto">
            <HardDrive className="w-6 h-6 text-white" />
          </div>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-6 space-y-1 overflow-y-auto custom-scrollbar">
        {navigation.map((item) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.name}
              href={item.href}
              className={`group flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-200 relative ${
                isActive
                  ? 'bg-gradient-to-r from-orange-500/20 to-orange-600/10 text-white shadow-lg shadow-orange-500/20'
                  : 'text-gray-400 hover:bg-gray-800/50 hover:text-white'
              }`}
              title={isCollapsed ? item.name : undefined}
            >
              {/* Active indicator */}
              {isActive && (
                <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-gradient-to-b from-orange-500 to-orange-600 rounded-r-full" />
              )}

              {/* Icon with gradient background */}
              <div className={`flex items-center justify-center w-9 h-9 rounded-lg transition-all ${
                isActive
                  ? 'bg-gradient-to-br from-orange-500/30 to-orange-600/20'
                  : 'bg-gray-800/50 group-hover:bg-gray-700/50'
              }`}>
                <item.icon className={`w-5 h-5 ${isActive ? 'text-orange-400' : 'text-gray-400 group-hover:text-white'}`} />
              </div>

              {!isCollapsed && (
                <>
                  <span className="flex-1">{item.name}</span>
                  {item.badge && (
                    <span className="px-2 py-0.5 text-xs font-semibold bg-orange-500/20 text-orange-400 rounded-full">
                      {item.badge}
                    </span>
                  )}
                </>
              )}
            </Link>
          )
        })}
      </nav>

      {/* Status Card */}
      {!isCollapsed && (
        <div className="mx-3 mb-6 p-4 rounded-xl bg-gradient-to-br from-blue-500/10 to-purple-500/10 border border-blue-500/20">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500/30 to-purple-500/30 flex items-center justify-center">
              <Sparkles className="w-5 h-5 text-blue-400" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-semibold text-white">System Health</h3>
              <p className="text-xs text-gray-400 mt-0.5">All systems operational</p>
              <div className="flex items-center gap-2 mt-2">
                <div className="flex-1 h-1.5 bg-gray-800 rounded-full overflow-hidden">
                  <div className="h-full w-[92%] bg-gradient-to-r from-green-500 to-emerald-500 rounded-full" />
                </div>
                <span className="text-xs font-medium text-green-400">92%</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="p-4 border-t border-gray-800">
        {!isCollapsed ? (
          <div className="flex items-center justify-between text-xs">
            <div className="text-gray-400">
              <p className="font-medium">v2.0.0</p>
              <p className="text-gray-500 mt-0.5">Enterprise Edition</p>
            </div>
            <button
              onClick={() => setIsCollapsed(true)}
              className="p-2 rounded-lg hover:bg-gray-800 text-gray-400 hover:text-white transition-colors"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
              </svg>
            </button>
          </div>
        ) : (
          <button
            onClick={() => setIsCollapsed(false)}
            className="w-full p-2 rounded-lg hover:bg-gray-800 text-gray-400 hover:text-white transition-colors flex justify-center"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 5l7 7-7 7M5 5l7 7-7 7" />
            </svg>
          </button>
        )}
      </div>
    </div>
  )
}
