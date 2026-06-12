'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Edit, Play, Pause, RefreshCw, CheckCircle, XCircle,
  AlertCircle, Clock, Package, Server, Database, Calendar
} from 'lucide-react'

// Mock API - replace with actual API calls
const mockApi = {
  getOperatorStatus: async () => ({
    installed: true,
    version: 'v1.0.0',
    namespace: 'db-backup-system',
    replicas: 1,
    ready: 1,
    status: 'running' as const,
  }),

  listBackupPolicies: async () => ([
    {
      id: '1',
      name: 'postgres-daily',
      namespace: 'default',
      databaseType: 'postgres',
      schedule: '0 2 * * *',
      retention: { daily: 7, weekly: 4, monthly: 3 },
      enabled: true,
      lastBackup: '2026-01-08T02:00:00Z',
      status: 'active',
    },
  ]),

  listBackupSchedules: async () => ([
    {
      id: '1',
      name: 'mysql-hourly',
      namespace: 'default',
      databaseType: 'mysql',
      schedule: '0 * * * *',
      enabled: true,
      nextRun: '2026-01-08T15:00:00Z',
      lastRun: '2026-01-08T14:00:00Z',
      status: 'scheduled',
    },
  ]),

  listRestoreJobs: async () => ([
    {
      id: '1',
      name: 'restore-postgres-20260108',
      namespace: 'default',
      backupName: 'postgres-backup-20260108',
      targetDatabase: 'postgres-prod',
      status: 'completed',
      startTime: '2026-01-08T10:00:00Z',
      completionTime: '2026-01-08T10:15:00Z',
    },
  ]),
}

type OperatorStatus = {
  installed: boolean
  version: string
  namespace: string
  replicas: number
  ready: number
  status: 'running' | 'error' | 'pending'
}

type BackupPolicy = {
  id: string
  name: string
  namespace: string
  databaseType: string
  schedule: string
  retention: {
    daily: number
    weekly?: number
    monthly?: number
    yearly?: number
  }
  enabled: boolean
  lastBackup?: string
  status: 'active' | 'paused' | 'error'
}

type BackupSchedule = {
  id: string
  name: string
  namespace: string
  databaseType: string
  schedule: string
  enabled: boolean
  nextRun?: string
  lastRun?: string
  status: 'scheduled' | 'running' | 'error'
}

type RestoreJob = {
  id: string
  name: string
  namespace: string
  backupName: string
  targetDatabase: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  startTime?: string
  completionTime?: string
}

export default function KubernetesPage() {
  const [activeTab, setActiveTab] = useState<'policies' | 'schedules' | 'restores'>('policies')
  const [showPolicyModal, setShowPolicyModal] = useState(false)
  const [showScheduleModal, setShowScheduleModal] = useState(false)
  const [showRestoreModal, setShowRestoreModal] = useState(false)

  const queryClient = useQueryClient()

  // Queries
  const { data: operatorStatus } = useQuery({
    queryKey: ['operator-status'],
    queryFn: mockApi.getOperatorStatus,
    refetchInterval: 10000, // Refresh every 10 seconds
  })

  const { data: backupPolicies } = useQuery({
    queryKey: ['backup-policies'],
    queryFn: mockApi.listBackupPolicies,
  })

  const { data: backupSchedules } = useQuery({
    queryKey: ['backup-schedules'],
    queryFn: mockApi.listBackupSchedules,
  })

  const { data: restoreJobs } = useQuery({
    queryKey: ['restore-jobs'],
    queryFn: mockApi.listRestoreJobs,
  })

  const getStatusBadge = (status: string) => {
    const badges = {
      running: 'bg-green-100 text-green-800',
      active: 'bg-green-100 text-green-800',
      completed: 'bg-blue-100 text-blue-800',
      scheduled: 'bg-blue-100 text-blue-800',
      pending: 'bg-yellow-100 text-yellow-800',
      paused: 'bg-gray-100 text-gray-800',
      error: 'bg-red-100 text-red-800',
      failed: 'bg-red-100 text-red-800',
    }
    return badges[status as keyof typeof badges] || 'bg-gray-100 text-gray-800'
  }

  const getStatusIcon = (status: string) => {
    const icons = {
      running: <CheckCircle className="w-4 h-4" />,
      active: <CheckCircle className="w-4 h-4" />,
      completed: <CheckCircle className="w-4 h-4" />,
      scheduled: <Clock className="w-4 h-4" />,
      pending: <Clock className="w-4 h-4" />,
      paused: <Pause className="w-4 h-4" />,
      error: <XCircle className="w-4 h-4" />,
      failed: <XCircle className="w-4 h-4" />,
    }
    return icons[status as keyof typeof icons] || <AlertCircle className="w-4 h-4" />
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Kubernetes Operator</h1>
          <p className="text-gray-500 mt-1">Manage database backups in Kubernetes</p>
        </div>
        <button
          onClick={() => queryClient.invalidateQueries()}
          className="flex items-center px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
        >
          <RefreshCw className="w-5 h-5 mr-2" />
          Refresh
        </button>
      </div>

      {/* Operator Status Card */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 flex items-center">
            <Server className="w-6 h-6 mr-2 text-blue-600" />
            Operator Status
          </h2>
          {operatorStatus && (
            <span className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${getStatusBadge(operatorStatus.status)}`}>
              {getStatusIcon(operatorStatus.status)}
              {operatorStatus.status.toUpperCase()}
            </span>
          )}
        </div>

        {operatorStatus && operatorStatus.installed ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
            <div>
              <p className="text-sm text-gray-500">Version</p>
              <p className="text-lg font-semibold text-gray-900">{operatorStatus.version}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Namespace</p>
              <p className="text-lg font-semibold text-gray-900">{operatorStatus.namespace}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Replicas</p>
              <p className="text-lg font-semibold text-gray-900">{operatorStatus.ready}/{operatorStatus.replicas}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Health</p>
              <p className="text-lg font-semibold text-green-600">Healthy</p>
            </div>
          </div>
        ) : (
          <div className="text-center py-8">
            <Package className="w-16 h-16 mx-auto text-gray-400 mb-4" />
            <p className="text-gray-600 mb-4">Operator not installed</p>
            <button className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors">
              Install Operator
            </button>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('policies')}
            className={`py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'policies'
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            <Database className="w-4 h-4 inline mr-2" />
            Backup Policies
            {backupPolicies && (
              <span className="ml-2 py-0.5 px-2 rounded-full bg-gray-100 text-gray-900 text-xs">
                {backupPolicies.length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('schedules')}
            className={`py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'schedules'
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            <Calendar className="w-4 h-4 inline mr-2" />
            Backup Schedules
            {backupSchedules && (
              <span className="ml-2 py-0.5 px-2 rounded-full bg-gray-100 text-gray-900 text-xs">
                {backupSchedules.length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('restores')}
            className={`py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'restores'
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            <RefreshCw className="w-4 h-4 inline mr-2" />
            Restore Jobs
            {restoreJobs && (
              <span className="ml-2 py-0.5 px-2 rounded-full bg-gray-100 text-gray-900 text-xs">
                {restoreJobs.length}
              </span>
            )}
          </button>
        </nav>
      </div>

      {/* Backup Policies Tab */}
      {activeTab === 'policies' && (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-semibold text-gray-900">Backup Policies</h2>
            <button
              onClick={() => setShowPolicyModal(true)}
              className="flex items-center px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
            >
              <Plus className="w-5 h-5 mr-2" />
              Create Policy
            </button>
          </div>

          <div className="grid grid-cols-1 gap-4">
            {backupPolicies && backupPolicies.length > 0 ? (
              backupPolicies.map((policy) => (
                <div key={policy.id} className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-blue-100 rounded-lg">
                        <Database className="w-5 h-5 text-blue-600" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">{policy.name}</h3>
                        <p className="text-sm text-gray-500">Namespace: {policy.namespace}</p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium ${getStatusBadge(policy.status)}`}>
                        {getStatusIcon(policy.status)}
                        {policy.status.toUpperCase()}
                      </span>
                      <button className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
                        <Edit className="w-4 h-4" />
                      </button>
                      <button className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <p className="text-gray-500">Database Type</p>
                      <p className="font-medium text-gray-900">{policy.databaseType.toUpperCase()}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Schedule</p>
                      <p className="font-medium text-gray-900">{policy.schedule}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Retention</p>
                      <p className="font-medium text-gray-900">
                        {policy.retention.daily}d / {policy.retention.weekly || 0}w / {policy.retention.monthly || 0}m
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500">Last Backup</p>
                      <p className="font-medium text-gray-900">
                        {policy.lastBackup ? new Date(policy.lastBackup).toLocaleString() : 'Never'}
                      </p>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
                <Database className="w-16 h-16 mx-auto text-gray-400 mb-4" />
                <p className="text-gray-600 mb-4">No backup policies configured</p>
                <button
                  onClick={() => setShowPolicyModal(true)}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                >
                  Create Your First Policy
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Backup Schedules Tab */}
      {activeTab === 'schedules' && (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-semibold text-gray-900">Backup Schedules</h2>
            <button
              onClick={() => setShowScheduleModal(true)}
              className="flex items-center px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
            >
              <Plus className="w-5 h-5 mr-2" />
              Create Schedule
            </button>
          </div>

          <div className="grid grid-cols-1 gap-4">
            {backupSchedules && backupSchedules.length > 0 ? (
              backupSchedules.map((schedule) => (
                <div key={schedule.id} className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-purple-100 rounded-lg">
                        <Calendar className="w-5 h-5 text-purple-600" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">{schedule.name}</h3>
                        <p className="text-sm text-gray-500">Namespace: {schedule.namespace}</p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium ${getStatusBadge(schedule.status)}`}>
                        {getStatusIcon(schedule.status)}
                        {schedule.status.toUpperCase()}
                      </span>
                      <button className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
                        {schedule.enabled ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                      </button>
                      <button className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
                        <Edit className="w-4 h-4" />
                      </button>
                      <button className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <p className="text-gray-500">Database Type</p>
                      <p className="font-medium text-gray-900">{schedule.databaseType.toUpperCase()}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Schedule</p>
                      <p className="font-medium text-gray-900">{schedule.schedule}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Next Run</p>
                      <p className="font-medium text-gray-900">
                        {schedule.nextRun ? new Date(schedule.nextRun).toLocaleString() : 'N/A'}
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500">Last Run</p>
                      <p className="font-medium text-gray-900">
                        {schedule.lastRun ? new Date(schedule.lastRun).toLocaleString() : 'Never'}
                      </p>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
                <Calendar className="w-16 h-16 mx-auto text-gray-400 mb-4" />
                <p className="text-gray-600 mb-4">No backup schedules configured</p>
                <button
                  onClick={() => setShowScheduleModal(true)}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                >
                  Create Your First Schedule
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Restore Jobs Tab */}
      {activeTab === 'restores' && (
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-semibold text-gray-900">Restore Jobs</h2>
            <button
              onClick={() => setShowRestoreModal(true)}
              className="flex items-center px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
            >
              <Plus className="w-5 h-5 mr-2" />
              Create Restore Job
            </button>
          </div>

          <div className="grid grid-cols-1 gap-4">
            {restoreJobs && restoreJobs.length > 0 ? (
              restoreJobs.map((job) => (
                <div key={job.id} className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-green-100 rounded-lg">
                        <RefreshCw className="w-5 h-5 text-green-600" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">{job.name}</h3>
                        <p className="text-sm text-gray-500">Namespace: {job.namespace}</p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium ${getStatusBadge(job.status)}`}>
                        {getStatusIcon(job.status)}
                        {job.status.toUpperCase()}
                      </span>
                      <button className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <p className="text-gray-500">Backup Source</p>
                      <p className="font-medium text-gray-900">{job.backupName}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Target Database</p>
                      <p className="font-medium text-gray-900">{job.targetDatabase}</p>
                    </div>
                    <div>
                      <p className="text-gray-500">Start Time</p>
                      <p className="font-medium text-gray-900">
                        {job.startTime ? new Date(job.startTime).toLocaleString() : 'N/A'}
                      </p>
                    </div>
                    <div>
                      <p className="text-gray-500">Duration</p>
                      <p className="font-medium text-gray-900">
                        {job.startTime && job.completionTime
                          ? `${Math.round((new Date(job.completionTime).getTime() - new Date(job.startTime).getTime()) / 60000)}m`
                          : 'N/A'}
                      </p>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
                <RefreshCw className="w-16 h-16 mx-auto text-gray-400 mb-4" />
                <p className="text-gray-600 mb-4">No restore jobs found</p>
                <button
                  onClick={() => setShowRestoreModal(true)}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                >
                  Create Restore Job
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Modals placeholders */}
      {showPolicyModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-2xl">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Create Backup Policy</h3>
            <p className="text-gray-600 mb-4">Policy creation form will go here</p>
            <div className="flex justify-end space-x-3">
              <button
                onClick={() => setShowPolicyModal(false)}
                className="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
              >
                Cancel
              </button>
              <button className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors">
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
