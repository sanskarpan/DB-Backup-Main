"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import {
  Shield,
  Database,
  Link2,
  Play,
  Pause,
  CheckCircle2,
  XCircle,
  Clock,
  GitBranch,
  Lock,
  Unlock,
  FileText,
  Activity
} from 'lucide-react'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

interface ConsistencyGroup {
  id: string
  name: string
  databases: string[]
  level: string
  priority: number
  created_at: string
  last_backup: string
  backup_count: number
  status: 'active' | 'inactive'
}

interface Hook {
  id: string
  type: string
  name: string
  command: string
  enabled: boolean
  last_execution: string
  success_rate: number
  avg_duration: number
}

interface QuiesceOperation {
  database: string
  type: string
  state: string
  started_at: string
  duration_ms: number
  checkpoint_lsn?: string
  binlog_position?: number
  metadata: Record<string, any>
}

interface BackupOrder {
  step: number
  databases: string[]
  can_parallel: boolean
  estimated_duration: number
}

export default function AppConsistencyDashboard() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [selectedGroup, setSelectedGroup] = useState<string>('')

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    // Mock data
    const mockData = {
      stats: {
        total_groups: 8,
        active_groups: 6,
        total_hooks: 24,
        active_hooks: 20,
        avg_consistency_time: 2.5,
        success_rate: 98.7
      },
      consistency_groups: [
        {
          id: 'cg-001',
          name: 'Production E-commerce',
          databases: ['products-db', 'orders-db', 'customers-db'],
          level: 'transactional',
          priority: 1,
          created_at: new Date(Date.now() - 86400000 * 30).toISOString(),
          last_backup: new Date(Date.now() - 3600000).toISOString(),
          backup_count: 156,
          status: 'active' as const
        },
        {
          id: 'cg-002',
          name: 'Analytics Pipeline',
          databases: ['events-db', 'metrics-db'],
          level: 'database',
          priority: 2,
          created_at: new Date(Date.now() - 86400000 * 15).toISOString(),
          last_backup: new Date(Date.now() - 7200000).toISOString(),
          backup_count: 89,
          status: 'active' as const
        }
      ],
      hooks: [
        {
          id: 'hook-001',
          type: 'pre-backup',
          name: 'Database Health Check',
          command: '/scripts/health-check.sh',
          enabled: true,
          last_execution: new Date(Date.now() - 3600000).toISOString(),
          success_rate: 99.5,
          avg_duration: 125
        },
        {
          id: 'hook-002',
          type: 'post-backup',
          name: 'Backup Verification',
          command: '/scripts/verify-backup.sh',
          enabled: true,
          last_execution: new Date(Date.now() - 3600000).toISOString(),
          success_rate: 98.2,
          avg_duration: 340
        },
        {
          id: 'hook-003',
          type: 'pre-quiesce',
          name: 'Flush Application Cache',
          command: '/scripts/flush-cache.sh',
          enabled: true,
          last_execution: new Date(Date.now() - 3600000).toISOString(),
          success_rate: 100.0,
          avg_duration: 45
        }
      ],
      quiesce_operations: [
        {
          database: 'products-db',
          type: 'postgresql',
          state: 'quiesced',
          started_at: new Date(Date.now() - 120000).toISOString(),
          duration_ms: 1250,
          checkpoint_lsn: '0/ABC12345',
          metadata: {
            backup_label: 'backup_20250101',
            database_size_bytes: 15728640000
          }
        },
        {
          database: 'orders-db',
          type: 'mysql',
          state: 'normal',
          started_at: new Date(Date.now() - 300000).toISOString(),
          duration_ms: 2100,
          binlog_position: 1234567,
          metadata: {
            binlog_file: 'mysql-bin.000123',
            table_count: 45
          }
        }
      ],
      backup_order: [
        { step: 1, databases: ['products-db'], can_parallel: false, estimated_duration: 180 },
        { step: 2, databases: ['orders-db', 'customers-db'], can_parallel: true, estimated_duration: 240 },
        { step: 3, databases: ['events-db'], can_parallel: false, estimated_duration: 120 }
      ],
      success_trend: Array.from({ length: 24 }, (_, i) => ({
        hour: `${i}:00`,
        success_rate: 95 + Math.random() * 5,
        avg_duration: 2 + Math.random() * 2
      }))
    }

    setData(mockData)
    setLoading(false)
  }

  const getStateBadgeVariant = (state: string): "default" | "destructive" | "outline" | "secondary" => {
    if (state === 'quiesced' || state === 'active') return 'default'
    if (state === 'failed') return 'destructive'
    if (state === 'quiescing' || state === 'resuming') return 'secondary'
    return 'outline'
  }

  if (loading || !data) {
    return <div className="flex items-center justify-center h-96"><Activity className="h-12 w-12 animate-spin" /></div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Shield className="h-8 w-8 text-primary" />
            Application-Consistent Backups
          </h1>
          <p className="text-muted-foreground mt-1">Consistency groups, hooks, and transaction coordination</p>
        </div>
        <Button onClick={fetchData}><Activity className="h-4 w-4 mr-2" />Refresh</Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Consistency Groups</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.total_groups}</div>
            <p className="text-xs text-green-600">{data.stats.active_groups} active</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Hooks</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.active_hooks}</div>
            <p className="text-xs text-muted-foreground">of {data.stats.total_hooks} total</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Avg Consistency Time</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.avg_consistency_time}s</div>
            <p className="text-xs text-green-600">Within SLA</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.success_rate}%</div>
            <p className="text-xs text-green-600">Highly reliable</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="groups">
        <TabsList>
          <TabsTrigger value="groups">Consistency Groups</TabsTrigger>
          <TabsTrigger value="hooks">Backup Hooks</TabsTrigger>
          <TabsTrigger value="quiesce">Quiesce Operations</TabsTrigger>
          <TabsTrigger value="order">Backup Order</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
        </TabsList>

        {/* Consistency Groups Tab */}
        <TabsContent value="groups" className="space-y-4">
          {data.consistency_groups.map((group: ConsistencyGroup) => (
            <Card key={group.id} className={selectedGroup === group.id ? 'border-2 border-primary' : ''}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{group.name}</CardTitle>
                  <Badge variant={group.status === 'active' ? 'default' : 'outline'}>
                    {group.status}
                  </Badge>
                </div>
                <CardDescription>
                  Priority {group.priority} • {group.level} consistency
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="text-sm font-medium mb-2">Databases</p>
                  <div className="flex flex-wrap gap-2">
                    {group.databases.map((db) => (
                      <Badge key={db} variant="outline" className="text-xs">
                        <Database className="h-3 w-3 mr-1" />
                        {db}
                      </Badge>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Created</p>
                    <p className="text-sm font-medium">{new Date(group.created_at).toLocaleDateString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Last Backup</p>
                    <p className="text-sm font-medium">{new Date(group.last_backup).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Total Backups</p>
                    <p className="text-sm font-medium">{group.backup_count}</p>
                  </div>
                </div>

                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={() => setSelectedGroup(group.id)}>
                    <FileText className="h-4 w-4 mr-2" />
                    View Details
                  </Button>
                  <Button variant="outline" size="sm">
                    <Play className="h-4 w-4 mr-2" />
                    Run Backup
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Hooks Tab */}
        <TabsContent value="hooks" className="space-y-4">
          <div className="grid gap-4">
            {data.hooks.map((hook: Hook) => (
              <Card key={hook.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base">{hook.name}</CardTitle>
                      <CardDescription>{hook.command}</CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={hook.enabled ? 'default' : 'outline'}>
                        {hook.enabled ? <CheckCircle2 className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                        {hook.enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                      <Badge variant="outline">{hook.type}</Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <p className="text-xs text-muted-foreground">Success Rate</p>
                      <p className={`text-sm font-medium ${hook.success_rate >= 95 ? 'text-green-600' : 'text-orange-600'}`}>
                        {hook.success_rate.toFixed(1)}%
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Avg Duration</p>
                      <p className="text-sm font-medium">{hook.avg_duration}ms</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Last Execution</p>
                      <p className="text-sm font-medium">{new Date(hook.last_execution).toLocaleString()}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Quiesce Operations Tab */}
        <TabsContent value="quiesce" className="space-y-4">
          {data.quiesce_operations.map((op: QuiesceOperation, idx: number) => (
            <Card key={idx}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{op.database}</CardTitle>
                  <div className="flex items-center gap-2">
                    <Badge variant={getStateBadgeVariant(op.state)}>
                      {op.state === 'quiesced' && <Lock className="h-3 w-3 mr-1" />}
                      {op.state === 'normal' && <Unlock className="h-3 w-3 mr-1" />}
                      {op.state}
                    </Badge>
                    <Badge variant="outline">{op.type}</Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Started At</p>
                    <p className="text-sm font-medium">{new Date(op.started_at).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Duration</p>
                    <p className="text-sm font-medium">{op.duration_ms}ms</p>
                  </div>
                  {op.checkpoint_lsn && (
                    <div>
                      <p className="text-xs text-muted-foreground">Checkpoint LSN</p>
                      <p className="text-sm font-medium font-mono">{op.checkpoint_lsn}</p>
                    </div>
                  )}
                  {op.binlog_position && (
                    <div>
                      <p className="text-xs text-muted-foreground">Binlog Position</p>
                      <p className="text-sm font-medium">{op.binlog_position}</p>
                    </div>
                  )}
                </div>

                {Object.keys(op.metadata).length > 0 && (
                  <div>
                    <p className="text-sm font-medium mb-2">Metadata</p>
                    <div className="bg-muted p-3 rounded-lg">
                      <pre className="text-xs overflow-x-auto">
                        {JSON.stringify(op.metadata, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Backup Order Tab */}
        <TabsContent value="order">
          <Card>
            <CardHeader>
              <CardTitle>Optimal Backup Order</CardTitle>
              <CardDescription>Calculated based on database dependencies</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-6">
                {data.backup_order.map((step: BackupOrder) => (
                  <div key={step.step} className="relative">
                    <div className="flex items-center gap-4">
                      <div className="flex items-center justify-center w-12 h-12 rounded-full bg-primary text-primary-foreground font-bold">
                        {step.step}
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          {step.can_parallel && (
                            <Badge variant="outline">
                              <GitBranch className="h-3 w-3 mr-1" />
                              Parallel Execution
                            </Badge>
                          )}
                          <Badge variant="outline">
                            <Clock className="h-3 w-3 mr-1" />
                            ~{step.estimated_duration}s
                          </Badge>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {step.databases.map((db) => (
                            <Badge key={db} variant="secondary">
                              <Database className="h-3 w-3 mr-1" />
                              {db}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    </div>
                    {step.step < data.backup_order.length && (
                      <div className="absolute left-6 top-12 bottom-0 w-0.5 bg-border"></div>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Trends Tab */}
        <TabsContent value="trends">
          <div className="grid gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Success Rate Trend (24 Hours)</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={data.success_trend}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="hour" />
                    <YAxis domain={[90, 100]} />
                    <Tooltip />
                    <Legend />
                    <Line type="monotone" dataKey="success_rate" stroke="#10b981" name="Success Rate (%)" />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Average Consistency Duration (24 Hours)</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={data.success_trend}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="hour" />
                    <YAxis />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="avg_duration" fill="#3b82f6" name="Duration (s)" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
