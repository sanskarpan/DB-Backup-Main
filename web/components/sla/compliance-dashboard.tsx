'use client'

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import {
  CheckCircle2,
  XCircle,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  Clock,
  Database,
  FileText,
  Shield,
  Activity,
  Target,
  Download
} from 'lucide-react'
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface SLADefinition {
  id: string
  database_id: string
  database_name: string
  level: string
  success_rate_target: number
  max_backup_duration: string
  rto_target: string
  rpo_target: string
  backup_frequency: string
  compliance_standards: string[]
}

interface SLAMetrics {
  database_id: string
  database_name: string
  total_backups: number
  successful_backups: number
  failed_backups: number
  success_rate: number
  average_backup_duration: string
  last_backup_time: string
  consecutive_failures: number
  total_data_backed_up: number
}

interface RTOMetrics {
  database_id: string
  target_rto: string
  actual_rto: string
  is_compliant: boolean
  restore_success_rate: number
  average_restore_time: string
  total_restore_tests: number
}

interface RPOMetrics {
  database_id: string
  target_rpo: string
  actual_rpo: string
  is_compliant: boolean
  data_loss_risk: number
  backup_frequency: string
}

interface SLAViolation {
  id: string
  database_id: string
  database_name: string
  violation_type: string
  severity: string
  detected_at: string
  resolved_at?: string
  is_resolved: boolean
  description: string
}

interface ExecutiveSummary {
  overall_compliance_rate: number
  total_databases: number
  compliant_databases: number
  non_compliant_databases: number
  average_success_rate: number
  total_backups: number
  successful_backups: number
  failed_backups: number
  total_violations: number
  critical_violations: number
  unresolved_violations: number
  top_issues: string[]
  recommendations: string[]
}

interface TrendPoint {
  date: string
  success_rate: number
  compliance_rate: number
  violation_count: number
  avg_backup_time: number
}

export default function ComplianceDashboard() {
  const [data, setData] = useState<{
    executive_summary: ExecutiveSummary
    databases: Array<{
      definition: SLADefinition
      metrics: SLAMetrics
      rto_metrics?: RTOMetrics
      rpo_metrics?: RPOMetrics
      is_compliant: boolean
      violation_count: number
    }>
    violations: SLAViolation[]
    trends: TrendPoint[]
  } | null>(null)

  const [loading, setLoading] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [selectedDatabase, setSelectedDatabase] = useState<string>('all')

  useEffect(() => {
    fetchData()

    if (autoRefresh) {
      const interval = setInterval(fetchData, 30000) // Refresh every 30 seconds
      return () => clearInterval(interval)
    }
  }, [autoRefresh])

  const fetchData = async () => {
    try {
      // In production, this would be an API call
      // const response = await fetch('/api/sla/compliance')
      // const data = await response.json()

      // Mock data for development
      const mockData = {
        executive_summary: {
          overall_compliance_rate: 92.5,
          total_databases: 12,
          compliant_databases: 11,
          non_compliant_databases: 1,
          average_success_rate: 99.4,
          total_backups: 1450,
          successful_backups: 1442,
          failed_backups: 8,
          total_violations: 5,
          critical_violations: 1,
          unresolved_violations: 2,
          top_issues: [
            '1 database below target success rate',
            '1 database exceeding RPO target',
            '3 consecutive failures on Analytics DB'
          ],
          recommendations: [
            'Review backup configuration for Analytics DB',
            'Increase backup frequency to reduce RPO',
            'Consider implementing incremental backups for large databases'
          ]
        },
        databases: [
          {
            definition: {
              id: 'sla-001',
              database_id: 'db-prod-001',
              database_name: 'Production PostgreSQL',
              level: 'critical',
              success_rate_target: 99.99,
              max_backup_duration: '2h',
              rto_target: '4h',
              rpo_target: '1h',
              backup_frequency: '30m',
              compliance_standards: ['SOC2', 'HIPAA', 'GDPR']
            },
            metrics: {
              database_id: 'db-prod-001',
              database_name: 'Production PostgreSQL',
              total_backups: 480,
              successful_backups: 480,
              failed_backups: 0,
              success_rate: 100.0,
              average_backup_duration: '1h15m',
              last_backup_time: '2025-12-31T10:30:00Z',
              consecutive_failures: 0,
              total_data_backed_up: 524288000000
            },
            rto_metrics: {
              database_id: 'db-prod-001',
              target_rto: '4h',
              actual_rto: '2h30m',
              is_compliant: true,
              restore_success_rate: 100.0,
              average_restore_time: '2h15m',
              total_restore_tests: 12
            },
            rpo_metrics: {
              database_id: 'db-prod-001',
              target_rpo: '1h',
              actual_rpo: '28m',
              is_compliant: true,
              data_loss_risk: 0,
              backup_frequency: '30m'
            },
            is_compliant: true,
            violation_count: 0
          },
          {
            definition: {
              id: 'sla-002',
              database_id: 'db-analytics-001',
              database_name: 'Analytics MongoDB',
              level: 'high',
              success_rate_target: 99.9,
              max_backup_duration: '3h',
              rto_target: '8h',
              rpo_target: '2h',
              backup_frequency: '1h',
              compliance_standards: ['SOC2', 'GDPR']
            },
            metrics: {
              database_id: 'db-analytics-001',
              database_name: 'Analytics MongoDB',
              total_backups: 240,
              successful_backups: 237,
              failed_backups: 3,
              success_rate: 98.75,
              average_backup_duration: '2h45m',
              last_backup_time: '2025-12-31T10:00:00Z',
              consecutive_failures: 3,
              total_data_backed_up: 1048576000000
            },
            rpo_metrics: {
              database_id: 'db-analytics-001',
              target_rpo: '2h',
              actual_rpo: '3h15m',
              is_compliant: false,
              data_loss_risk: 62.5,
              backup_frequency: '1h'
            },
            is_compliant: false,
            violation_count: 2
          },
          {
            definition: {
              id: 'sla-003',
              database_id: 'db-reporting-001',
              database_name: 'Reporting MySQL',
              level: 'medium',
              success_rate_target: 99.5,
              max_backup_duration: '4h',
              rto_target: '12h',
              rpo_target: '4h',
              backup_frequency: '2h',
              compliance_standards: ['SOC2']
            },
            metrics: {
              database_id: 'db-reporting-001',
              database_name: 'Reporting MySQL',
              total_backups: 120,
              successful_backups: 120,
              failed_backups: 0,
              success_rate: 100.0,
              average_backup_duration: '1h30m',
              last_backup_time: '2025-12-31T09:00:00Z',
              consecutive_failures: 0,
              total_data_backed_up: 209715200000
            },
            rto_metrics: {
              database_id: 'db-reporting-001',
              target_rto: '12h',
              actual_rto: '6h',
              is_compliant: true,
              restore_success_rate: 100.0,
              average_restore_time: '5h30m',
              total_restore_tests: 6
            },
            rpo_metrics: {
              database_id: 'db-reporting-001',
              target_rpo: '4h',
              actual_rpo: '1h45m',
              is_compliant: true,
              data_loss_risk: 0,
              backup_frequency: '2h'
            },
            is_compliant: true,
            violation_count: 0
          }
        ],
        violations: [
          {
            id: 'viol-001',
            database_id: 'db-analytics-001',
            database_name: 'Analytics MongoDB',
            violation_type: 'success_rate',
            severity: 'high',
            detected_at: '2025-12-31T08:30:00Z',
            is_resolved: false,
            description: 'Success rate (98.75%) below target (99.9%)'
          },
          {
            id: 'viol-002',
            database_id: 'db-analytics-001',
            database_name: 'Analytics MongoDB',
            violation_type: 'rpo',
            severity: 'critical',
            detected_at: '2025-12-31T09:00:00Z',
            is_resolved: false,
            description: 'Actual RPO (3h15m) exceeds target (2h)'
          },
          {
            id: 'viol-003',
            database_id: 'db-prod-001',
            database_name: 'Production PostgreSQL',
            violation_type: 'duration',
            severity: 'medium',
            detected_at: '2025-12-30T14:00:00Z',
            resolved_at: '2025-12-30T16:00:00Z',
            is_resolved: true,
            description: 'Backup duration exceeded maximum threshold'
          }
        ],
        trends: [
          { date: '2025-12-24', success_rate: 99.2, compliance_rate: 91.7, violation_count: 3, avg_backup_time: 95 },
          { date: '2025-12-25', success_rate: 99.3, compliance_rate: 91.7, violation_count: 2, avg_backup_time: 92 },
          { date: '2025-12-26', success_rate: 99.5, compliance_rate: 91.7, violation_count: 2, avg_backup_time: 90 },
          { date: '2025-12-27', success_rate: 99.4, compliance_rate: 91.7, violation_count: 3, avg_backup_time: 93 },
          { date: '2025-12-28', success_rate: 99.6, compliance_rate: 91.7, violation_count: 1, avg_backup_time: 88 },
          { date: '2025-12-29', success_rate: 99.5, compliance_rate: 91.7, violation_count: 2, avg_backup_time: 91 },
          { date: '2025-12-30', success_rate: 99.3, compliance_rate: 91.7, violation_count: 4, avg_backup_time: 94 },
          { date: '2025-12-31', success_rate: 99.4, compliance_rate: 92.5, violation_count: 2, avg_backup_time: 90 }
        ]
      }

      setData(mockData)
      setLoading(false)
    } catch (error) {
      console.error('Error fetching SLA data:', error)
      setLoading(false)
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical': return 'destructive'
      case 'high': return 'destructive'
      case 'medium': return 'default'
      case 'low': return 'secondary'
      default: return 'default'
    }
  }

  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'critical': return 'destructive'
      case 'high': return 'default'
      case 'medium': return 'secondary'
      case 'low': return 'outline'
      default: return 'default'
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDuration = (duration: string) => {
    return duration
  }

  const generateReport = async (reportType: string, format: string) => {
    console.log(`Generating ${reportType} report in ${format} format...`)
    // In production: await fetch('/api/sla/reports', { method: 'POST', body: { type, format } })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Activity className="h-8 w-8 animate-spin mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">Loading SLA compliance data...</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>Failed to load SLA compliance data.</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">SLA Compliance Dashboard</h1>
          <p className="text-muted-foreground">Monitor backup SLA compliance and performance metrics</p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
          >
            <Activity className={`h-4 w-4 mr-2 ${autoRefresh ? 'animate-spin' : ''}`} />
            {autoRefresh ? 'Auto-refresh On' : 'Auto-refresh Off'}
          </Button>
        </div>
      </div>

      {/* Critical Alerts */}
      {data.executive_summary.critical_violations > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Critical SLA Violations Detected</AlertTitle>
          <AlertDescription>
            {data.executive_summary.critical_violations} critical violation(s) require immediate attention.
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="executive" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="executive">Executive Summary</TabsTrigger>
          <TabsTrigger value="databases">Database Compliance</TabsTrigger>
          <TabsTrigger value="violations">SLA Violations</TabsTrigger>
          <TabsTrigger value="rto-rpo">RTO/RPO Tracking</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
        </TabsList>

        {/* Executive Summary Tab */}
        <TabsContent value="executive" className="space-y-4">
          {/* Key Metrics */}
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Overall Compliance</CardTitle>
                <Shield className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {data.executive_summary.overall_compliance_rate.toFixed(1)}%
                </div>
                <Progress value={data.executive_summary.overall_compliance_rate} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-2">
                  {data.executive_summary.compliant_databases} of {data.executive_summary.total_databases} databases compliant
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
                <Target className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {data.executive_summary.average_success_rate.toFixed(2)}%
                </div>
                <Progress value={data.executive_summary.average_success_rate} className="mt-2" />
                <p className="text-xs text-muted-foreground mt-2">
                  {data.executive_summary.successful_backups} / {data.executive_summary.total_backups} backups successful
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Active Violations</CardTitle>
                <AlertTriangle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-destructive">
                  {data.executive_summary.unresolved_violations}
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  {data.executive_summary.critical_violations} critical violations
                </p>
                <div className="flex gap-2 mt-2">
                  <Badge variant="destructive" className="text-xs">
                    {data.executive_summary.critical_violations} Critical
                  </Badge>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
                <Database className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {data.executive_summary.total_backups.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  {data.executive_summary.failed_backups} failed backups
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Top Issues & Recommendations */}
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <AlertTriangle className="h-5 w-5 text-destructive" />
                  Top Issues
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {data.executive_summary.top_issues.map((issue, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <XCircle className="h-4 w-4 text-destructive mt-0.5" />
                      <span className="text-sm">{issue}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-500" />
                  Recommendations
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {data.executive_summary.recommendations.map((rec, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <TrendingUp className="h-4 w-4 text-blue-500 mt-0.5" />
                      <span className="text-sm">{rec}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          </div>

          {/* Report Generation */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Generate Compliance Report
              </CardTitle>
              <CardDescription>Create executive reports for stakeholders</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" onClick={() => generateReport('executive', 'pdf')}>
                  <Download className="h-4 w-4 mr-2" />
                  Executive Summary (PDF)
                </Button>
                <Button variant="outline" size="sm" onClick={() => generateReport('detailed', 'pdf')}>
                  <Download className="h-4 w-4 mr-2" />
                  Detailed Report (PDF)
                </Button>
                <Button variant="outline" size="sm" onClick={() => generateReport('executive', 'json')}>
                  <Download className="h-4 w-4 mr-2" />
                  Data Export (JSON)
                </Button>
                <Button variant="outline" size="sm" onClick={() => generateReport('compliance', 'markdown')}>
                  <Download className="h-4 w-4 mr-2" />
                  Compliance Standards (MD)
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Database Compliance Tab */}
        <TabsContent value="databases" className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Showing {data.databases.length} database(s)
            </p>
          </div>

          <div className="grid gap-4">
            {data.databases.map((db) => (
              <Card key={db.definition.id} className={!db.is_compliant ? 'border-destructive' : ''}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      {db.is_compliant ? (
                        <CheckCircle2 className="h-6 w-6 text-green-500" />
                      ) : (
                        <XCircle className="h-6 w-6 text-destructive" />
                      )}
                      <div>
                        <CardTitle>{db.definition.database_name}</CardTitle>
                        <CardDescription>ID: {db.definition.database_id}</CardDescription>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Badge variant={getLevelColor(db.definition.level)}>
                        {db.definition.level.toUpperCase()}
                      </Badge>
                      {db.is_compliant ? (
                        <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                          COMPLIANT
                        </Badge>
                      ) : (
                        <Badge variant="destructive">
                          NON-COMPLIANT
                        </Badge>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-3">
                    {/* Backup Metrics */}
                    <div className="space-y-2">
                      <h4 className="text-sm font-semibold">Backup Performance</h4>
                      <div className="space-y-1 text-sm">
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Success Rate:</span>
                          <span className="font-medium">
                            {db.metrics.success_rate.toFixed(2)}%
                            {db.metrics.success_rate >= db.definition.success_rate_target ? (
                              <CheckCircle2 className="h-3 w-3 text-green-500 inline ml-1" />
                            ) : (
                              <XCircle className="h-3 w-3 text-destructive inline ml-1" />
                            )}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Target:</span>
                          <span className="font-medium">{db.definition.success_rate_target}%</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Total Backups:</span>
                          <span className="font-medium">{db.metrics.total_backups}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Failed:</span>
                          <span className={`font-medium ${db.metrics.failed_backups > 0 ? 'text-destructive' : ''}`}>
                            {db.metrics.failed_backups}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Avg Duration:</span>
                          <span className="font-medium">{formatDuration(db.metrics.average_backup_duration)}</span>
                        </div>
                      </div>
                    </div>

                    {/* RTO/RPO Metrics */}
                    <div className="space-y-2">
                      <h4 className="text-sm font-semibold">RTO/RPO Compliance</h4>
                      <div className="space-y-1 text-sm">
                        {db.rto_metrics && (
                          <>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">RTO Status:</span>
                              <span className="font-medium">
                                {db.rto_metrics.is_compliant ? (
                                  <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 text-xs">
                                    Compliant
                                  </Badge>
                                ) : (
                                  <Badge variant="destructive" className="text-xs">
                                    Non-Compliant
                                  </Badge>
                                )}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Actual RTO:</span>
                              <span className="font-medium">{formatDuration(db.rto_metrics.actual_rto)}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Target RTO:</span>
                              <span className="font-medium">{formatDuration(db.rto_metrics.target_rto)}</span>
                            </div>
                          </>
                        )}
                        {db.rpo_metrics && (
                          <>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">RPO Status:</span>
                              <span className="font-medium">
                                {db.rpo_metrics.is_compliant ? (
                                  <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 text-xs">
                                    Compliant
                                  </Badge>
                                ) : (
                                  <Badge variant="destructive" className="text-xs">
                                    Non-Compliant
                                  </Badge>
                                )}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Actual RPO:</span>
                              <span className="font-medium">{formatDuration(db.rpo_metrics.actual_rpo)}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-muted-foreground">Target RPO:</span>
                              <span className="font-medium">{formatDuration(db.rpo_metrics.target_rpo)}</span>
                            </div>
                          </>
                        )}
                      </div>
                    </div>

                    {/* Compliance Standards */}
                    <div className="space-y-2">
                      <h4 className="text-sm font-semibold">Compliance Standards</h4>
                      <div className="flex flex-wrap gap-1">
                        {db.definition.compliance_standards.map((standard) => (
                          <Badge key={standard} variant="secondary" className="text-xs">
                            {standard}
                          </Badge>
                        ))}
                      </div>
                      <div className="space-y-1 text-sm mt-3">
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Violations:</span>
                          <span className={`font-medium ${db.violation_count > 0 ? 'text-destructive' : ''}`}>
                            {db.violation_count}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Consecutive Failures:</span>
                          <span className={`font-medium ${db.metrics.consecutive_failures > 0 ? 'text-destructive' : ''}`}>
                            {db.metrics.consecutive_failures}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Data Backed Up:</span>
                          <span className="font-medium">{formatBytes(db.metrics.total_data_backed_up)}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* SLA Violations Tab */}
        <TabsContent value="violations" className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {data.violations.filter(v => !v.is_resolved).length} unresolved violation(s)
            </p>
          </div>

          {/* Unresolved Violations */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-destructive" />
                Unresolved Violations
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {data.violations.filter(v => !v.is_resolved).map((violation) => (
                  <div
                    key={violation.id}
                    className="flex items-start justify-between p-3 border rounded-lg"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge variant={getSeverityColor(violation.severity)}>
                          {violation.severity.toUpperCase()}
                        </Badge>
                        <Badge variant="outline">{violation.violation_type}</Badge>
                        <span className="text-sm font-medium">{violation.database_name}</span>
                      </div>
                      <p className="text-sm text-muted-foreground">{violation.description}</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Detected: {new Date(violation.detected_at).toLocaleString()}
                      </p>
                    </div>
                    <Button variant="outline" size="sm">
                      Resolve
                    </Button>
                  </div>
                ))}
                {data.violations.filter(v => !v.is_resolved).length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    <CheckCircle2 className="h-12 w-12 mx-auto mb-2 text-green-500" />
                    <p>No unresolved violations</p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Resolved Violations */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                Resolved Violations
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {data.violations.filter(v => v.is_resolved).map((violation) => (
                  <div
                    key={violation.id}
                    className="flex items-start justify-between p-3 border rounded-lg opacity-60"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge variant="outline">{violation.severity}</Badge>
                        <Badge variant="outline">{violation.violation_type}</Badge>
                        <span className="text-sm font-medium">{violation.database_name}</span>
                      </div>
                      <p className="text-sm text-muted-foreground">{violation.description}</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Resolved: {violation.resolved_at ? new Date(violation.resolved_at).toLocaleString() : 'N/A'}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* RTO/RPO Tracking Tab */}
        <TabsContent value="rto-rpo" className="space-y-4">
          <div className="grid gap-4">
            {data.databases.map((db) => (
              <Card key={db.definition.id}>
                <CardHeader>
                  <CardTitle>{db.definition.database_name}</CardTitle>
                  <CardDescription>Recovery objectives and compliance</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-6 md:grid-cols-2">
                    {/* RTO Metrics */}
                    {db.rto_metrics && (
                      <div className="space-y-3">
                        <div className="flex items-center justify-between">
                          <h4 className="text-sm font-semibold">Recovery Time Objective (RTO)</h4>
                          {db.rto_metrics.is_compliant ? (
                            <CheckCircle2 className="h-5 w-5 text-green-500" />
                          ) : (
                            <XCircle className="h-5 w-5 text-destructive" />
                          )}
                        </div>
                        <div className="space-y-2">
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Target RTO:</span>
                            <span className="font-medium">{formatDuration(db.rto_metrics.target_rto)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Actual RTO:</span>
                            <span className="font-medium">{formatDuration(db.rto_metrics.actual_rto)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Average Restore Time:</span>
                            <span className="font-medium">{formatDuration(db.rto_metrics.average_restore_time)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Restore Success Rate:</span>
                            <span className="font-medium">{db.rto_metrics.restore_success_rate.toFixed(1)}%</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Total Restore Tests:</span>
                            <span className="font-medium">{db.rto_metrics.total_restore_tests}</span>
                          </div>
                        </div>
                      </div>
                    )}

                    {/* RPO Metrics */}
                    {db.rpo_metrics && (
                      <div className="space-y-3">
                        <div className="flex items-center justify-between">
                          <h4 className="text-sm font-semibold">Recovery Point Objective (RPO)</h4>
                          {db.rpo_metrics.is_compliant ? (
                            <CheckCircle2 className="h-5 w-5 text-green-500" />
                          ) : (
                            <XCircle className="h-5 w-5 text-destructive" />
                          )}
                        </div>
                        <div className="space-y-2">
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Target RPO:</span>
                            <span className="font-medium">{formatDuration(db.rpo_metrics.target_rpo)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Actual RPO:</span>
                            <span className="font-medium">{formatDuration(db.rpo_metrics.actual_rpo)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Backup Frequency:</span>
                            <span className="font-medium">{formatDuration(db.rpo_metrics.backup_frequency)}</span>
                          </div>
                          <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">Data Loss Risk:</span>
                            <span className={`font-medium ${db.rpo_metrics.data_loss_risk > 0 ? 'text-destructive' : ''}`}>
                              {db.rpo_metrics.data_loss_risk.toFixed(1)}%
                            </span>
                          </div>
                        </div>
                        {db.rpo_metrics.data_loss_risk > 0 && (
                          <Alert variant="destructive" className="mt-2">
                            <AlertTriangle className="h-4 w-4" />
                            <AlertDescription>
                              RPO target exceeded - increase backup frequency
                            </AlertDescription>
                          </Alert>
                        )}
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Trends Tab */}
        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Success Rate Trend</CardTitle>
              <CardDescription>Backup success rate over time</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={data.trends}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" />
                  <YAxis domain={[98, 100]} />
                  <Tooltip />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="success_rate"
                    stroke="#10b981"
                    strokeWidth={2}
                    name="Success Rate (%)"
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Compliance Rate Trend</CardTitle>
                <CardDescription>Database compliance over time</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={data.trends}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis domain={[85, 100]} />
                    <Tooltip />
                    <Line
                      type="monotone"
                      dataKey="compliance_rate"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      name="Compliance Rate (%)"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Violation Count Trend</CardTitle>
                <CardDescription>SLA violations over time</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <BarChart data={data.trends}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="violation_count" fill="#ef4444" name="Violations" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Average Backup Time Trend</CardTitle>
              <CardDescription>Backup duration performance over time</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={data.trends}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" />
                  <YAxis />
                  <Tooltip />
                  <Line
                    type="monotone"
                    dataKey="avg_backup_time"
                    stroke="#8b5cf6"
                    strokeWidth={2}
                    name="Avg Backup Time (min)"
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
