import { Backup } from '@/lib/api'
import { CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react'

interface RecentBackupsProps {
  backups: Backup[]
}

export function RecentBackups({ backups }: RecentBackupsProps) {
  const getStatusIcon = (status: Backup['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="w-5 h-5 text-green-500" />
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-500" />
      case 'in_progress':
        return <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
      case 'pending':
        return <Clock className="w-5 h-5 text-gray-500" />
    }
  }

  const getStatusText = (status: Backup['status']) => {
    return status.charAt(0).toUpperCase() + status.slice(1).replace('_', ' ')
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
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

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Backups</h3>
      <div className="space-y-4">
        {backups.length === 0 ? (
          <p className="text-gray-500 text-sm text-center py-8">No backups yet</p>
        ) : (
          backups.map((backup) => (
            <div
              key={backup.id}
              className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <div className="flex items-center space-x-4 flex-1">
                {getStatusIcon(backup.status)}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {backup.database_name || backup.database_id}
                  </p>
                  <p className="text-xs text-gray-500 mt-1">{backup.filename}</p>
                </div>
              </div>
              <div className="flex items-center space-x-6 text-sm">
                <div className="text-right">
                  <p className="text-gray-900 font-medium">{formatSize(backup.size)}</p>
                  <p className="text-gray-500 text-xs mt-1">{formatDate(backup.created_at)}</p>
                </div>
                <span
                  className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                    backup.status === 'completed'
                      ? 'bg-green-100 text-green-800'
                      : backup.status === 'failed'
                      ? 'bg-red-100 text-red-800'
                      : backup.status === 'in_progress'
                      ? 'bg-blue-100 text-blue-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}
                >
                  {getStatusText(backup.status)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
