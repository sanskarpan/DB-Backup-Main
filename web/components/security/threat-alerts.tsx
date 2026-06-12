'use client'

import { useState } from 'react'
import {
  AlertTriangle,
  AlertCircle,
  Info,
  XCircle,
  CheckCircle,
  Clock,
  Shield,
  Eye,
  Ban,
  FileWarning,
  Zap,
  Filter,
} from 'lucide-react'

interface ThreatAlert {
  id: string
  timestamp: string
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO'
  type: 'RANSOMWARE_DETECTED' | 'SUSPICIOUS_ACTIVITY' | 'POLICY_VIOLATION' | 'ANOMALY' | 'INFO'
  title: string
  description: string
  affectedResources: string[]
  status: 'active' | 'investigating' | 'resolved' | 'dismissed'
  detectionMethod: string
  recommendedAction: string
}

export function ThreatAlerts() {
  const [filterSeverity, setFilterSeverity] = useState<string>('all')
  const [filterStatus, setFilterStatus] = useState<string>('all')
  const [selectedAlert, setSelectedAlert] = useState<ThreatAlert | null>(null)

  // Mock data - in production, fetch from API
  const alerts: ThreatAlert[] = [
    {
      id: '1',
      timestamp: '2025-12-30T10:15:00Z',
      severity: 'INFO',
      type: 'INFO',
      title: 'Backup Scan Completed Successfully',
      description: 'Regular security scan completed with no threats detected',
      affectedResources: ['All backup files'],
      status: 'resolved',
      detectionMethod: 'Scheduled Scan',
      recommendedAction: 'No action required',
    },
    {
      id: '2',
      timestamp: '2025-12-30T09:45:00Z',
      severity: 'LOW',
      type: 'ANOMALY',
      title: 'Unusual Backup Timing Detected',
      description: 'Backup created outside of normal schedule window',
      affectedResources: ['/backups/prod/manual-backup-2025-12-30.sql'],
      status: 'resolved',
      detectionMethod: 'Behavioral Analysis',
      recommendedAction: 'Verify backup was intentionally created by authorized user',
    },
    {
      id: '3',
      timestamp: '2025-12-30T08:30:00Z',
      severity: 'MEDIUM',
      type: 'POLICY_VIOLATION',
      title: 'Retention Policy Change Attempted',
      description: 'Attempt to reduce retention period from 90 to 30 days blocked',
      affectedResources: ['Azure Blob Storage - Production'],
      status: 'resolved',
      detectionMethod: 'Policy Enforcement',
      recommendedAction: 'Retention policy is locked and cannot be reduced',
    },
  ]

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return <XCircle className="w-5 h-5 text-red-600" />
      case 'HIGH':
        return <AlertTriangle className="w-5 h-5 text-orange-600" />
      case 'MEDIUM':
        return <AlertCircle className="w-5 h-5 text-yellow-600" />
      case 'LOW':
        return <Info className="w-5 h-5 text-blue-600" />
      default:
        return <CheckCircle className="w-5 h-5 text-green-600" />
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return 'bg-red-50 border-red-200 text-red-900'
      case 'HIGH':
        return 'bg-orange-50 border-orange-200 text-orange-900'
      case 'MEDIUM':
        return 'bg-yellow-50 border-yellow-200 text-yellow-900'
      case 'LOW':
        return 'bg-blue-50 border-blue-200 text-blue-900'
      default:
        return 'bg-green-50 border-green-200 text-green-900'
    }
  }

  const getSeverityBadgeColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return 'bg-red-100 text-red-700'
      case 'HIGH':
        return 'bg-orange-100 text-orange-700'
      case 'MEDIUM':
        return 'bg-yellow-100 text-yellow-700'
      case 'LOW':
        return 'bg-blue-100 text-blue-700'
      default:
        return 'bg-green-100 text-green-700'
    }
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'RANSOMWARE_DETECTED':
        return <FileWarning className="w-4 h-4" />
      case 'SUSPICIOUS_ACTIVITY':
        return <Eye className="w-4 h-4" />
      case 'POLICY_VIOLATION':
        return <Ban className="w-4 h-4" />
      case 'ANOMALY':
        return <Zap className="w-4 h-4" />
      default:
        return <Info className="w-4 h-4" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-red-600 bg-red-100'
      case 'investigating':
        return 'text-yellow-600 bg-yellow-100'
      case 'resolved':
        return 'text-green-600 bg-green-100'
      case 'dismissed':
        return 'text-gray-600 bg-gray-100'
      default:
        return 'text-gray-600 bg-gray-100'
    }
  }

  const filteredAlerts = alerts.filter((alert) => {
    const severityMatch = filterSeverity === 'all' || alert.severity === filterSeverity
    const statusMatch = filterStatus === 'all' || alert.status === filterStatus
    return severityMatch && statusMatch
  })

  const severityCounts = {
    CRITICAL: alerts.filter((a) => a.severity === 'CRITICAL').length,
    HIGH: alerts.filter((a) => a.severity === 'HIGH').length,
    MEDIUM: alerts.filter((a) => a.severity === 'MEDIUM').length,
    LOW: alerts.filter((a) => a.severity === 'LOW').length,
    INFO: alerts.filter((a) => a.severity === 'INFO').length,
  }

  const handleResolve = (alertId: string) => {
    // In production, call API to update alert status
    console.log('Resolving alert:', alertId)
    setSelectedAlert(null)
  }

  const handleDismiss = (alertId: string) => {
    // In production, call API to dismiss alert
    console.log('Dismissing alert:', alertId)
    setSelectedAlert(null)
  }

  return (
    <div className="space-y-6">
      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <div className="bg-white rounded-lg shadow p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Critical</p>
              <p className="text-2xl font-bold text-red-600">{severityCounts.CRITICAL}</p>
            </div>
            <XCircle className="w-8 h-8 text-red-600" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">High</p>
              <p className="text-2xl font-bold text-orange-600">{severityCounts.HIGH}</p>
            </div>
            <AlertTriangle className="w-8 h-8 text-orange-600" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Medium</p>
              <p className="text-2xl font-bold text-yellow-600">{severityCounts.MEDIUM}</p>
            </div>
            <AlertCircle className="w-8 h-8 text-yellow-600" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Low</p>
              <p className="text-2xl font-bold text-blue-600">{severityCounts.LOW}</p>
            </div>
            <Info className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Info</p>
              <p className="text-2xl font-bold text-green-600">{severityCounts.INFO}</p>
            </div>
            <CheckCircle className="w-8 h-8 text-green-600" />
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-4">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Filter className="w-5 h-5 text-gray-600" />
            <span className="text-sm font-medium text-gray-700">Filter:</span>
          </div>
          <div>
            <label className="text-sm text-gray-600 mr-2">Severity:</label>
            <select
              value={filterSeverity}
              onChange={(e) => setFilterSeverity(e.target.value)}
              className="px-3 py-1 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">All</option>
              <option value="CRITICAL">Critical</option>
              <option value="HIGH">High</option>
              <option value="MEDIUM">Medium</option>
              <option value="LOW">Low</option>
              <option value="INFO">Info</option>
            </select>
          </div>
          <div>
            <label className="text-sm text-gray-600 mr-2">Status:</label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="px-3 py-1 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">All</option>
              <option value="active">Active</option>
              <option value="investigating">Investigating</option>
              <option value="resolved">Resolved</option>
              <option value="dismissed">Dismissed</option>
            </select>
          </div>
        </div>
      </div>

      {/* Alert List */}
      <div className="space-y-4">
        {filteredAlerts.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-8 text-center">
            <Shield className="w-12 h-12 text-green-600 mx-auto mb-3" />
            <h3 className="text-lg font-semibold text-gray-900 mb-2">No Alerts Found</h3>
            <p className="text-gray-600">
              {filterSeverity !== 'all' || filterStatus !== 'all'
                ? 'No alerts match your current filters'
                : 'All systems are operating normally'}
            </p>
          </div>
        ) : (
          filteredAlerts.map((alert) => (
            <div
              key={alert.id}
              className={`border rounded-lg p-4 cursor-pointer hover:shadow-md transition-shadow ${getSeverityColor(
                alert.severity
              )}`}
              onClick={() => setSelectedAlert(alert)}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3 flex-1">
                  {getSeverityIcon(alert.severity)}
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h4 className="font-semibold text-gray-900">{alert.title}</h4>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getSeverityBadgeColor(alert.severity)}`}>
                        {alert.severity}
                      </span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(alert.status)}`}>
                        {alert.status}
                      </span>
                    </div>
                    <p className="text-sm text-gray-700 mb-2">{alert.description}</p>
                    <div className="flex items-center gap-4 text-xs text-gray-600">
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(alert.timestamp).toLocaleString()}
                      </div>
                      <div className="flex items-center gap-1">
                        {getTypeIcon(alert.type)}
                        {alert.type.replace('_', ' ')}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Alert Details Modal */}
      {selectedAlert && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  {getSeverityIcon(selectedAlert.severity)}
                  <div>
                    <h3 className="text-xl font-bold text-gray-900">{selectedAlert.title}</h3>
                    <div className="flex items-center gap-2 mt-2">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getSeverityBadgeColor(selectedAlert.severity)}`}>
                        {selectedAlert.severity}
                      </span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(selectedAlert.status)}`}>
                        {selectedAlert.status}
                      </span>
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => setSelectedAlert(null)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <XCircle className="w-6 h-6" />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-6">
              {/* Description */}
              <div>
                <h4 className="text-sm font-semibold text-gray-900 mb-2">Description</h4>
                <p className="text-sm text-gray-700">{selectedAlert.description}</p>
              </div>

              {/* Details */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="text-sm font-semibold text-gray-900 mb-2">Detection Time</h4>
                  <p className="text-sm text-gray-700">{new Date(selectedAlert.timestamp).toLocaleString()}</p>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-gray-900 mb-2">Detection Method</h4>
                  <p className="text-sm text-gray-700">{selectedAlert.detectionMethod}</p>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-gray-900 mb-2">Alert Type</h4>
                  <p className="text-sm text-gray-700">{selectedAlert.type.replace('_', ' ')}</p>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-gray-900 mb-2">Status</h4>
                  <p className="text-sm text-gray-700 capitalize">{selectedAlert.status}</p>
                </div>
              </div>

              {/* Affected Resources */}
              <div>
                <h4 className="text-sm font-semibold text-gray-900 mb-2">Affected Resources</h4>
                <ul className="list-disc list-inside space-y-1">
                  {selectedAlert.affectedResources.map((resource, index) => (
                    <li key={index} className="text-sm text-gray-700">
                      {resource}
                    </li>
                  ))}
                </ul>
              </div>

              {/* Recommended Action */}
              <div className="p-4 bg-blue-50 rounded-lg">
                <h4 className="text-sm font-semibold text-blue-900 mb-2 flex items-center gap-2">
                  <Info className="w-4 h-4" />
                  Recommended Action
                </h4>
                <p className="text-sm text-blue-800">{selectedAlert.recommendedAction}</p>
              </div>
            </div>

            {/* Actions */}
            <div className="p-6 border-t border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setSelectedAlert(null)}
                className="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
              >
                Close
              </button>
              {selectedAlert.status === 'active' && (
                <>
                  <button
                    onClick={() => handleDismiss(selectedAlert.id)}
                    className="px-4 py-2 text-gray-700 bg-gray-200 rounded-lg hover:bg-gray-300 transition-colors"
                  >
                    Dismiss
                  </button>
                  <button
                    onClick={() => handleResolve(selectedAlert.id)}
                    className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                  >
                    Mark as Resolved
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
