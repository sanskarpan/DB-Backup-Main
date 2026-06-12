'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, Schedule } from '@/lib/api'
import { useState } from 'react'
import {
  Plus,
  Trash2,
  Edit,
  Power,
  PowerOff,
  Calendar,
  Clock,
  Loader2,
  History,
  TrendingUp,
  CheckCircle,
  AlertCircle,
} from 'lucide-react'

export default function SchedulesPage() {
  const [showModal, setShowModal] = useState(false)
  const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null)

  const queryClient = useQueryClient()

  const { data: schedules, isLoading } = useQuery({
    queryKey: ['schedules'],
    queryFn: () => api.listSchedules(),
  })

  const { data: databases } = useQuery({
    queryKey: ['databases'],
    queryFn: () => api.listDatabases(),
  })

  const createMutation = useMutation({
    mutationFn: (data: Partial<Schedule>) => api.createSchedule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
      setShowModal(false)
      setEditingSchedule(null)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Schedule> }) =>
      api.updateSchedule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
      setShowModal(false)
      setEditingSchedule(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })

  const toggleEnabledMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.updateSchedule(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data = {
      database_id: formData.get('database_id') as string,
      cron_expression: formData.get('cron_expression') as string,
      retention_days: parseInt(formData.get('retention_days') as string),
      enabled: true,
    }

    if (editingSchedule) {
      updateMutation.mutate({ id: editingSchedule.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const openEditModal = (schedule: Schedule) => {
    setEditingSchedule(schedule)
    setShowModal(true)
  }

  const openCreateModal = () => {
    setEditingSchedule(null)
    setShowModal(true)
  }

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never'
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const cronExamples = [
    { value: '0 0 * * *', label: 'Daily at midnight', icon: '🌙' },
    { value: '0 */6 * * *', label: 'Every 6 hours', icon: '⏰' },
    { value: '0 0 * * 0', label: 'Weekly (Sunday)', icon: '📅' },
    { value: '0 0 1 * *', label: 'Monthly', icon: '📆' },
  ]

  const stats = {
    total: schedules?.length || 0,
    active: schedules?.filter(s => s.enabled).length || 0,
    inactive: schedules?.filter(s => !s.enabled).length || 0,
    avgRetention: schedules?.length ? Math.round(schedules.reduce((sum, s) => sum + s.retention_days, 0) / schedules.length) : 0,
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Hero Section */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-purple-500 via-purple-600 to-pink-600 p-8 text-white shadow-2xl">
          <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-20"></div>
          <div className="relative z-10">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <Calendar className="w-10 h-10" />
                  <h1 className="text-4xl font-bold">Backup Schedules</h1>
                </div>
                <p className="text-purple-100 text-lg max-w-2xl">
                  Automate your database backups with flexible scheduling and retention policies
                </p>
              </div>
              <button
                onClick={openCreateModal}
                className="btn-primary bg-white text-purple-600 hover:bg-purple-50 shadow-xl"
              >
                <Plus className="w-5 h-5 mr-2" />
                Create Schedule
              </button>
            </div>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-lg">
                <Calendar className="w-6 h-6" />
              </div>
              <TrendingUp className="w-5 h-5 text-purple-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.total}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Total Schedules</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-lg">
                <Power className="w-6 h-6" />
              </div>
              <span className="flex h-3 w-3">
                <span className="animate-ping absolute inline-flex h-3 w-3 rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
              </span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.active}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Active Schedules</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-gray-400 to-gray-500 text-white shadow-lg">
                <PowerOff className="w-6 h-6" />
              </div>
              <AlertCircle className="w-5 h-5 text-gray-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.inactive}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Inactive Schedules</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-lg">
                <History className="w-6 h-6" />
              </div>
              <Clock className="w-5 h-5 text-blue-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.avgRetention}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Avg Retention (days)</div>
          </div>
        </div>

        {/* Schedules Table */}
        <div className="modern-card modern-card-dark overflow-hidden">
          <div className="overflow-x-auto custom-scrollbar">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
                <tr>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Database
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Schedule
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Retention
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Last Run
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Next Run
                  </th>
                  <th className="px-6 py-4 text-right text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={i} className="animate-pulse">
                      <td className="px-6 py-4">
                        <div className="h-6 w-24 bg-gray-200 dark:bg-gray-700 rounded-full"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="h-6 w-28 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="h-4 w-28 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex justify-end gap-2">
                          <div className="h-8 w-8 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                          <div className="h-8 w-8 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                          <div className="h-8 w-8 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : schedules && schedules.length > 0 ? (
                  schedules.map((schedule) => (
                    <tr key={schedule.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold border ${
                            schedule.enabled
                              ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950 dark:text-emerald-400 dark:border-emerald-800'
                              : 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700'
                          }`}
                        >
                          {schedule.enabled ? (
                            <>
                              <Power className="w-3 h-3" />
                              Active
                            </>
                          ) : (
                            <>
                              <PowerOff className="w-3 h-3" />
                              Disabled
                            </>
                          )}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {schedule.database_name || schedule.database_id}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          <Calendar className="w-4 h-4 text-purple-500" />
                          <code className="bg-purple-50 dark:bg-purple-950 text-purple-700 dark:text-purple-300 px-3 py-1 rounded-lg text-xs font-mono border border-purple-200 dark:border-purple-800">
                            {schedule.cron_expression}
                          </code>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          {schedule.retention_days} days
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                          <Clock className="w-4 h-4" />
                          {formatDate(schedule.last_run)}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                          <Clock className="w-4 h-4" />
                          {formatDate(schedule.next_run)}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() =>
                              toggleEnabledMutation.mutate({
                                id: schedule.id,
                                enabled: !schedule.enabled,
                              })
                            }
                            className={`p-2 rounded-lg transition-all hover:scale-110 ${
                              schedule.enabled
                                ? 'text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-950'
                                : 'text-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-950'
                            }`}
                            title={schedule.enabled ? 'Disable' : 'Enable'}
                          >
                            {schedule.enabled ? (
                              <PowerOff className="w-4 h-4" />
                            ) : (
                              <Power className="w-4 h-4" />
                            )}
                          </button>
                          <button
                            onClick={() => openEditModal(schedule)}
                            className="p-2 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-950 rounded-lg transition-all hover:scale-110"
                            title="Edit"
                          >
                            <Edit className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => {
                              if (confirm('Are you sure you want to delete this schedule?')) {
                                deleteMutation.mutate(schedule.id)
                              }
                            }}
                            className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-950 rounded-lg transition-all hover:scale-110"
                            title="Delete"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={7} className="px-6 py-16">
                      <div className="flex flex-col items-center justify-center text-center">
                        <div className="w-20 h-20 mb-6 rounded-full bg-gradient-to-br from-purple-100 to-pink-100 dark:from-purple-950 dark:to-pink-950 flex items-center justify-center">
                          <Calendar className="w-10 h-10 text-purple-600 dark:text-purple-400" />
                        </div>
                        <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-3">No schedules configured</h3>
                        <p className="text-gray-600 dark:text-gray-400 mb-8 max-w-md">
                          Create your first automated backup schedule to ensure your data is protected regularly.
                        </p>
                        <button
                          onClick={openCreateModal}
                          className="btn-primary"
                        >
                          <Plus className="w-5 h-5 mr-2" />
                          Create Your First Schedule
                        </button>
                      </div>
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-900 rounded-2xl p-8 w-full max-w-2xl shadow-2xl border border-gray-200 dark:border-gray-800 max-h-[90vh] overflow-y-auto custom-scrollbar">
            <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
              {editingSchedule ? 'Edit Schedule' : 'Create New Schedule'}
            </h3>
            <form onSubmit={handleSubmit}>
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Database
                  </label>
                  <select
                    name="database_id"
                    required
                    defaultValue={editingSchedule?.database_id}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent transition-all"
                  >
                    <option value="">Select database...</option>
                    {databases?.map((db) => (
                      <option key={db.id} value={db.id}>
                        {db.name} ({db.type})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Cron Expression
                  </label>
                  <input
                    type="text"
                    name="cron_expression"
                    required
                    defaultValue={editingSchedule?.cron_expression}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono transition-all"
                    placeholder="0 0 * * *"
                  />
                  <div className="mt-4 grid grid-cols-2 gap-3">
                    <p className="col-span-2 text-xs font-semibold text-gray-600 dark:text-gray-400 mb-2">Quick Templates:</p>
                    {cronExamples.map((example) => (
                      <button
                        key={example.value}
                        type="button"
                        onClick={(e) => {
                          const input = e.currentTarget.form?.elements.namedItem(
                            'cron_expression'
                          ) as HTMLInputElement
                          if (input) input.value = example.value
                        }}
                        className="flex items-center gap-2 p-3 text-left text-sm bg-gradient-to-r from-purple-50 to-pink-50 dark:from-purple-950 dark:to-pink-950 text-purple-700 dark:text-purple-300 rounded-lg hover:from-purple-100 hover:to-pink-100 dark:hover:from-purple-900 dark:hover:to-pink-900 transition-all border border-purple-200 dark:border-purple-800"
                      >
                        <span className="text-lg">{example.icon}</span>
                        <div>
                          <div className="font-mono text-xs text-purple-900 dark:text-purple-200">{example.value}</div>
                          <div className="text-xs text-purple-600 dark:text-purple-400">{example.label}</div>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Retention Period (days)
                  </label>
                  <input
                    type="number"
                    name="retention_days"
                    required
                    min="1"
                    defaultValue={editingSchedule?.retention_days || 30}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent transition-all"
                    placeholder="30"
                  />
                  <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 flex items-center gap-2">
                    <History className="w-4 h-4" />
                    Backups older than this will be automatically deleted to save storage space
                  </p>
                </div>
              </div>

              <div className="flex justify-end gap-3 mt-8">
                <button
                  type="button"
                  onClick={() => {
                    setShowModal(false)
                    setEditingSchedule(null)
                  }}
                  className="px-6 py-3 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 rounded-xl hover:bg-gray-200 dark:hover:bg-gray-700 transition-all font-medium"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {createMutation.isPending || updateMutation.isPending ? (
                    <>
                      <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    <>
                      {editingSchedule ? (
                        <>
                          <Edit className="w-5 h-5 mr-2" />
                          Update Schedule
                        </>
                      ) : (
                        <>
                          <Plus className="w-5 h-5 mr-2" />
                          Create Schedule
                        </>
                      )}
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
