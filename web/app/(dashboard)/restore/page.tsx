'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, Backup } from '@/lib/api'
import { useState } from 'react'
import { GitBranch, Search, CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-react'

export default function RestorePage() {
  const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null)
  const [targetDatabaseId, setTargetDatabaseId] = useState<string>('')
  const [searchTerm, setSearchTerm] = useState('')
  const [restoreStatus, setRestoreStatus] = useState<{
    success: boolean
    message: string
  } | null>(null)

  const queryClient = useQueryClient()

  const { data: backupsData, isLoading: backupsLoading } = useQuery({
    queryKey: ['backups'],
    queryFn: () => api.listBackups(),
  })

  const { data: databases, isLoading: databasesLoading } = useQuery({
    queryKey: ['databases'],
    queryFn: () => api.listDatabases(),
  })

  const restoreMutation = useMutation({
    mutationFn: ({ backupId, targetDbId }: { backupId: string; targetDbId?: string }) =>
      api.restoreBackup(backupId, targetDbId),
    onSuccess: () => {
      setRestoreStatus({
        success: true,
        message: 'Database restore completed successfully!',
      })
      queryClient.invalidateQueries({ queryKey: ['backups'] })
      setTimeout(() => {
        setRestoreStatus(null)
        setSelectedBackup(null)
        setTargetDatabaseId('')
      }, 5000)
    },
    onError: (error: any) => {
      setRestoreStatus({
        success: false,
        message: error.response?.data?.error || 'Failed to restore database',
      })
      setTimeout(() => setRestoreStatus(null), 5000)
    },
  })

  const completedBackups = backupsData?.backups.filter((b) => b.status === 'completed') || []

  const filteredBackups = completedBackups.filter(
    (backup) =>
      backup.filename.toLowerCase().includes(searchTerm.toLowerCase()) ||
      backup.database_name?.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const handleRestore = () => {
    if (!selectedBackup) return

    const targetDb = targetDatabaseId || undefined
    restoreMutation.mutate({
      backupId: selectedBackup.id,
      targetDbId: targetDb,
    })
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Restore Database</h1>
        <p className="text-gray-500 mt-1">Restore your database from a backup</p>
      </div>

      {/* Info Alert */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 flex items-start">
        <Info className="w-5 h-5 text-blue-600 mr-3 flex-shrink-0 mt-0.5" />
        <div className="text-sm text-blue-800">
          <p className="font-medium mb-1">Important Notes:</p>
          <ul className="list-disc list-inside space-y-1">
            <li>Restoring will overwrite the existing database data</li>
            <li>Only completed backups can be restored</li>
            <li>You can restore to the original database or a different target database</li>
            <li>Make sure the target database exists and is accessible</li>
          </ul>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Backup Selection */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Select Backup</h2>

          <div className="mb-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="text"
                placeholder="Search backups..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              />
            </div>
          </div>

          <div className="space-y-2 max-h-96 overflow-y-auto">
            {backupsLoading ? (
              <div className="text-center py-8 text-gray-500">Loading backups...</div>
            ) : filteredBackups.length > 0 ? (
              filteredBackups.map((backup) => (
                <button
                  key={backup.id}
                  onClick={() => setSelectedBackup(backup)}
                  className={`w-full text-left p-4 border rounded-lg transition-all ${
                    selectedBackup?.id === backup.id
                      ? 'border-primary bg-primary/5 ring-2 ring-primary'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                  }`}
                >
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 truncate">
                        {backup.database_name || backup.database_id}
                      </p>
                      <p className="text-xs text-gray-500 mt-1 truncate">{backup.filename}</p>
                    </div>
                    {selectedBackup?.id === backup.id && (
                      <CheckCircle2 className="w-5 h-5 text-primary flex-shrink-0 ml-2" />
                    )}
                  </div>
                  <div className="flex items-center justify-between text-xs text-gray-500">
                    <span>{formatSize(backup.size)}</span>
                    <span>{formatDate(backup.created_at)}</span>
                  </div>
                </button>
              ))
            ) : (
              <div className="text-center py-8 text-gray-500">
                {searchTerm ? 'No backups match your search' : 'No completed backups available'}
              </div>
            )}
          </div>
        </div>

        {/* Restore Configuration */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Restore Configuration</h2>

          {selectedBackup ? (
            <div className="space-y-6">
              {/* Selected Backup Info */}
              <div className="bg-gray-50 rounded-lg p-4">
                <h3 className="text-sm font-medium text-gray-700 mb-3">Selected Backup</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Database:</span>
                    <span className="font-medium text-gray-900">
                      {selectedBackup.database_name || selectedBackup.database_id}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Filename:</span>
                    <span className="font-medium text-gray-900 truncate ml-2">
                      {selectedBackup.filename}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Size:</span>
                    <span className="font-medium text-gray-900">
                      {formatSize(selectedBackup.size)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Created:</span>
                    <span className="font-medium text-gray-900">
                      {formatDate(selectedBackup.created_at)}
                    </span>
                  </div>
                </div>
              </div>

              {/* Target Database Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Target Database (Optional)
                </label>
                <select
                  value={targetDatabaseId}
                  onChange={(e) => setTargetDatabaseId(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  disabled={databasesLoading}
                >
                  <option value="">Restore to original database</option>
                  {databases
                    ?.filter((db) => db.id !== selectedBackup.database_id)
                    .map((db) => (
                      <option key={db.id} value={db.id}>
                        {db.name} ({db.type})
                      </option>
                    ))}
                </select>
                <p className="mt-2 text-xs text-gray-500">
                  Leave empty to restore to the original database, or select a different target
                  database
                </p>
              </div>

              {/* Warning */}
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 flex items-start">
                <AlertTriangle className="w-5 h-5 text-yellow-600 mr-3 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-yellow-800">
                  <p className="font-medium">Warning</p>
                  <p className="mt-1">
                    This operation will overwrite all existing data in the target database. Make
                    sure you have a recent backup before proceeding.
                  </p>
                </div>
              </div>

              {/* Restore Status */}
              {restoreStatus && (
                <div
                  className={`rounded-lg p-4 flex items-start ${
                    restoreStatus.success
                      ? 'bg-green-50 border border-green-200'
                      : 'bg-red-50 border border-red-200'
                  }`}
                >
                  {restoreStatus.success ? (
                    <CheckCircle2 className="w-5 h-5 text-green-600 mr-3 flex-shrink-0 mt-0.5" />
                  ) : (
                    <XCircle className="w-5 h-5 text-red-600 mr-3 flex-shrink-0 mt-0.5" />
                  )}
                  <p
                    className={`text-sm ${
                      restoreStatus.success ? 'text-green-800' : 'text-red-800'
                    }`}
                  >
                    {restoreStatus.message}
                  </p>
                </div>
              )}

              {/* Restore Button */}
              <button
                onClick={handleRestore}
                disabled={restoreMutation.isPending}
                className="w-full flex items-center justify-center px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-colors"
              >
                <GitBranch className="w-5 h-5 mr-2" />
                {restoreMutation.isPending ? 'Restoring...' : 'Start Restore'}
              </button>
            </div>
          ) : (
            <div className="text-center py-12 text-gray-500">
              <GitBranch className="w-12 h-12 mx-auto mb-3 text-gray-400" />
              <p>Select a backup to configure restore options</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
