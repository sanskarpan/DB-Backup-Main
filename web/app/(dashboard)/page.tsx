'use client'

import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { StatsCard } from '@/components/dashboard/stats-card'
import { RecentBackups } from '@/components/dashboard/recent-backups'
import { StorageChart } from '@/components/dashboard/storage-chart'
import { Database, HardDrive, Activity, CheckCircle2, TrendingUp, Shield, Zap } from 'lucide-react'
import { useRouter } from 'next/navigation'

export default function DashboardPage() {
  const router = useRouter()

  const { data: stats } = useQuery({
    queryKey: ['stats'],
    queryFn: () => api.getStats(),
  })

  const { data: backups } = useQuery({
    queryKey: ['backups'],
    queryFn: () => api.listBackups(),
  })

  const totalBackups = backups?.total || 0
  const successfulBackups = backups?.backups?.filter((b: any) => b.status === 'completed').length || 0
  const totalSize = backups?.backups?.reduce((sum: number, b: any) => sum + (b.size || 0), 0) || 0

  const successRate = totalBackups > 0 ? ((successfulBackups / totalBackups) * 100).toFixed(1) : '0'

  return (
    <div className="space-y-8 fade-in">
      {/* Hero Section */}
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-r from-orange-500/10 via-orange-400/5 to-transparent rounded-3xl" />
        <div className="relative p-8 rounded-3xl">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-2">
                Welcome back
              </h1>
              <p className="text-lg text-gray-600 dark:text-gray-400">
                Here's what's happening with your database backups today
              </p>
            </div>
            <div className="hidden lg:flex items-center gap-3">
              <button className="btn-primary flex items-center gap-2" onClick={() => router.push('/backups')}>
                <Zap className="w-5 h-5" />
                New Backup
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <div className="stats-card group">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500/10 to-blue-600/10 group-hover:from-blue-500/20 group-hover:to-blue-600/20 transition-colors">
              <Database className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
            <TrendingUp className="w-4 h-4 text-green-500" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Backups</p>
            <p className="text-3xl font-bold text-gray-900 dark:text-white">{totalBackups}</p>
            <p className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
              <TrendingUp className="w-3 h-3" />
              +12% from last month
            </p>
          </div>
        </div>

        <div className="stats-card group">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 rounded-xl bg-gradient-to-br from-green-500/10 to-green-600/10 group-hover:from-green-500/20 group-hover:to-green-600/20 transition-colors">
              <CheckCircle2 className="w-6 h-6 text-green-600 dark:text-green-400" />
            </div>
            <TrendingUp className="w-4 h-4 text-green-500" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Success Rate</p>
            <p className="text-3xl font-bold text-gray-900 dark:text-white">{successRate}%</p>
            <p className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
              <TrendingUp className="w-3 h-3" />
              +2.5% from last month
            </p>
          </div>
        </div>

        <div className="stats-card group">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 rounded-xl bg-gradient-to-br from-orange-500/10 to-orange-600/10 group-hover:from-orange-500/20 group-hover:to-orange-600/20 transition-colors">
              <HardDrive className="w-6 h-6 text-orange-600 dark:text-orange-400" />
            </div>
            <TrendingUp className="w-4 h-4 text-orange-500" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Storage Used</p>
            <p className="text-3xl font-bold text-gray-900 dark:text-white">{formatBytes(totalSize)}</p>
            <p className="text-xs text-orange-600 dark:text-orange-400 flex items-center gap-1">
              <TrendingUp className="w-3 h-3" />
              +8% from last month
            </p>
          </div>
        </div>

        <div className="stats-card group">
          <div className="flex items-center justify-between mb-4">
            <div className="p-3 rounded-xl bg-gradient-to-br from-purple-500/10 to-purple-600/10 group-hover:from-purple-500/20 group-hover:to-purple-600/20 transition-colors">
              <Activity className="w-6 h-6 text-purple-600 dark:text-purple-400" />
            </div>
            <Shield className="w-4 h-4 text-purple-500" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Active Schedules</p>
            <p className="text-3xl font-bold text-gray-900 dark:text-white">12</p>
            <p className="text-xs text-purple-600 dark:text-purple-400 flex items-center gap-1">
              2 new this week
            </p>
          </div>
        </div>
      </div>

      {/* Charts and Lists */}
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="modern-card modern-card-dark p-6">
          <StorageChart />
        </div>
        <div className="modern-card modern-card-dark p-6">
          <RecentBackups backups={backups?.backups?.slice(0, 5) || []} />
        </div>
      </div>

      {/* Quick Actions */}
      <div className="modern-card modern-card-dark p-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Quick Actions</h2>
        <div className="grid gap-4 md:grid-cols-3">
          <button onClick={() => router.push('/backups')} className="p-4 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 hover:border-orange-500 dark:hover:border-orange-500 transition-colors group">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-orange-500/10 group-hover:bg-orange-500/20 transition-colors">
                <Database className="w-5 h-5 text-orange-600 dark:text-orange-400" />
              </div>
              <div className="text-left">
                <p className="font-semibold text-gray-900 dark:text-white">Create Backup</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">Start a new backup job</p>
              </div>
            </div>
          </button>

          <button onClick={() => router.push('/schedules')} className="p-4 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 hover:border-blue-500 dark:hover:border-blue-500 transition-colors group">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-500/10 group-hover:bg-blue-500/20 transition-colors">
                <Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="text-left">
                <p className="font-semibold text-gray-900 dark:text-white">Schedule Backup</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">Set up automated backups</p>
              </div>
            </div>
          </button>

          <button onClick={() => router.push('/security')} className="p-4 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 hover:border-green-500 dark:hover:border-green-500 transition-colors group">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-green-500/10 group-hover:bg-green-500/20 transition-colors">
                <Shield className="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div className="text-left">
                <p className="font-semibold text-gray-900 dark:text-white">Security Scan</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">Run ransomware detection</p>
              </div>
            </div>
          </button>
        </div>
      </div>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
