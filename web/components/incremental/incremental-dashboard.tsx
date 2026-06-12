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
  TrendingDown,
  Database,
  Activity,
  Link2,
  Layers,
  Zap,
  Clock
} from 'lucide-react'
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface BackupManifest {
  id: string
  type: string // "full", "incremental", "synthetic"
  database_id: string
  database_name: string
  file_size: number
  compressed_size: number
  block_size: number
  total_blocks: number
  changed_blocks: number
  parent_backup_id: string
  base_backup_id: string
  created_at: string
  duration: string
  status: string
}

interface BackupChain {
  database_id: string
  database_name: string
  base_backup: BackupManifest
  incrementals: BackupManifest[]
  chain_length: number
  total_size: number
  compressed_size: number
  savings_ratio: number
  health_status: string
}

interface ChangeStatistics {
  database_id: string
  total_snapshots: number
  total_blocks: number
  changed_blocks: number
  change_ratio: number
  average_block_size: number
}

interface SyntheticRecommendation {
  database_id: string
  database_name: string
  recommended: boolean
  reason: string
  priority: string
  incremental_count: number
  base_backup_age: number
  estimated_size: number
}

interface ChainHealth {
  chain_id: string
  database_name: string
  status: string
  chain_length: number
  chain_age: number
  issues: string[]
  recommendations: string[]
}

export default function IncrementalDashboard() {
  const [data, setData] = useState<{
    summary: {
      total_backups: number
      full_backups: number
      incremental_backups: number
      synthetic_backups: number
      total_size: number
      compressed_size: number
      storage_savings: number
      average_chain_length: number
    }
    chains: BackupChain[]
    change_stats: ChangeStatistics[]
    synthetic_recommendations: SyntheticRecommendation[]
    chain_health: ChainHealth[]
    trends: {
      date: string
      storage_used: number
      change_ratio: number
      incremental_count: number
    }[]
  } | null>(null)

  const [loading, setLoading] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)

  useEffect(() => {
    fetchData()

    if (autoRefresh) {
      const interval = setInterval(fetchData, 30000)
      return () => clearInterval(interval)
    }
  }, [autoRefresh])

  const fetchData = async () => {
    try {
      // Mock data for development
      const mockData = {
        summary: {
          total_backups: 45,
          full_backups: 5,
          incremental_backups: 35,
          synthetic_backups: 5,
          total_size: 524288000000, // 500 GB
          compressed_size: 157286400000, // 150 GB
          storage_savings: 70.0,
          average_chain_length: 7
        },
        chains: [
          {
            database_id: 'db-prod-001',
            database_name: 'Production PostgreSQL',
            base_backup: {
              id: 'full-001',
              type: 'full',
              database_id: 'db-prod-001',
              database_name: 'Production PostgreSQL',
              file_size: 104857600000,
              compressed_size: 31457280000,
              block_size: 4096,
              total_blocks: 25600000,
              changed_blocks: 0,
              parent_backup_id: '',
              base_backup_id: 'full-001',
              created_at: '2025-12-24T10:00:00Z',
              duration: '2h15m',
              status: 'completed'
            },
            incrementals: [
              {
                id: 'incr-001',
                type: 'incremental',
                database_id: 'db-prod-001',
                database_name: 'Production PostgreSQL',
                file_size: 104857600000,
                compressed_size: 1048576000,
                block_size: 4096,
                total_blocks: 25600000,
                changed_blocks: 256000,
                parent_backup_id: 'full-001',
                base_backup_id: 'full-001',
                created_at: '2025-12-25T10:00:00Z',
                duration: '15m',
                status: 'completed'
              },
              {
                id: 'incr-002',
                type: 'incremental',
                database_id: 'db-prod-001',
                database_name: 'Production PostgreSQL',
                file_size: 104857600000,
                compressed_size: 838860800,
                block_size: 4096,
                total_blocks: 25600000,
                changed_blocks: 204800,
                parent_backup_id: 'full-001',
                base_backup_id: 'full-001',
                created_at: '2025-12-26T10:00:00Z',
                duration: '12m',
                status: 'completed'
              }
            ],
            chain_length: 3,
            total_size: 314572800000,
            compressed_size: 33344716800,
            savings_ratio: 89.4,
            health_status: 'healthy'
          },
          {
            database_id: 'db-analytics-001',
            database_name: 'Analytics MongoDB',
            base_backup: {
              id: 'synth-001',
              type: 'synthetic',
              database_id: 'db-analytics-001',
              database_name: 'Analytics MongoDB',
              file_size: 209715200000,
              compressed_size: 62914560000,
              block_size: 8192,
              total_blocks: 25600000,
              changed_blocks: 0,
              parent_backup_id: '',
              base_backup_id: 'synth-001',
              created_at: '2025-12-30T10:00:00Z',
              duration: '1h45m',
              status: 'completed'
            },
            incrementals: [
              {
                id: 'incr-101',
                type: 'incremental',
                database_id: 'db-analytics-001',
                database_name: 'Analytics MongoDB',
                file_size: 209715200000,
                compressed_size: 2097152000,
                block_size: 8192,
                total_blocks: 25600000,
                changed_blocks: 512000,
                parent_backup_id: 'synth-001',
                base_backup_id: 'synth-001',
                created_at: '2025-12-31T10:00:00Z',
                duration: '18m',
                status: 'completed'
              }
            ],
            chain_length: 2,
            total_size: 419430400000,
            compressed_size: 65011712000,
            savings_ratio: 84.5,
            health_status: 'healthy'
          }
        ],
        change_stats: [
          {
            database_id: 'db-prod-001',
            total_snapshots: 10,
            total_blocks: 256000000,
            changed_blocks: 2560000,
            change_ratio: 1.0,
            average_block_size: 4096
          },
          {
            database_id: 'db-analytics-001',
            total_snapshots: 8,
            total_blocks: 204800000,
            changed_blocks: 5120000,
            change_ratio: 2.5,
            average_block_size: 8192
          }
        ],
        synthetic_recommendations: [
          {
            database_id: 'db-reporting-001',
            database_name: 'Reporting MySQL',
            recommended: true,
            reason: 'Long incremental chain (12+)',
            priority: 'high',
            incremental_count: 12,
            base_backup_age: 14 * 24, // hours
            estimated_size: 52428800000
          }
        ],
        chain_health: [
          {
            chain_id: 'db-prod-001',
            database_name: 'Production PostgreSQL',
            status: 'healthy',
            chain_length: 3,
            chain_age: 7 * 24, // hours
            issues: [],
            recommendations: []
          },
          {
            chain_id: 'db-analytics-001',
            database_name: 'Analytics MongoDB',
            status: 'healthy',
            chain_length: 2,
            chain_age: 1 * 24,
            issues: [],
            recommendations: []
          },
          {
            chain_id: 'db-reporting-001',
            database_name: 'Reporting MySQL',
            status: 'warning',
            chain_length: 12,
            chain_age: 14 * 24,
            issues: ['Chain has 12 incrementals (max recommended: 10)', 'Base backup is 14 days old'],
            recommendations: ['Consider creating a synthetic full backup']
          }
        ],
        trends: [
          { date: '2025-12-24', storage_used: 450, change_ratio: 1.2, incremental_count: 28 },
          { date: '2025-12-25', storage_used: 455, change_ratio: 1.1, incremental_count: 29 },
          { date: '2025-12-26', storage_used: 458, change_ratio: 1.0, incremental_count: 30 },
          { date: '2025-12-27', storage_used: 462, change_ratio: 1.3, incremental_count: 31 },
          { date: '2025-12-28', storage_used: 465, change_ratio: 0.9, incremental_count: 32 },
          { date: '2025-12-29', storage_used: 468, change_ratio: 1.1, incremental_count: 33 },
          { date: '2025-12-30', storage_used: 470, change_ratio: 1.4, incremental_count: 34 },
          { date: '2025-12-31', storage_used: 473, change_ratio: 1.0, incremental_count: 35 }
        ]
      }

      setData(mockData)
      setLoading(false)
    } catch (error) {
      console.error('Error fetching incremental backup data:', error)
      setLoading(false)
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'healthy': return 'default'
      case 'warning': return 'default'
      case 'critical': return 'destructive'
      default: return 'secondary'
    }
  }

  const getBackupTypeColor = (type: string) => {
    switch (type.toLowerCase()) {
      case 'full': return 'default'
      case 'incremental': return 'secondary'
      case 'synthetic': return 'outline'
      default: return 'secondary'
    }
  }

  const createSyntheticBackup = async (databaseId: string) => {
    console.log(`Creating synthetic backup for ${databaseId}...`)
    // In production: await fetch('/api/incremental/synthetic', { method: 'POST', body: { database_id } })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Activity className="h-8 w-8 animate-spin mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">Loading incremental backup data...</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>Failed to load incremental backup data.</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Incremental Forever Backups</h1>
          <p className="text-muted-foreground">Block-level incremental backups with synthetic full generation</p>
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

      {/* Warnings for recommendations */}
      {data.synthetic_recommendations.filter(r => r.recommended).length > 0 && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Synthetic Backup Recommendations</AlertTitle>
          <AlertDescription>
            {data.synthetic_recommendations.filter(r => r.recommended).length} database(s) have long incremental chains and should consider synthetic full backups.
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="chains">Backup Chains</TabsTrigger>
          <TabsTrigger value="changes">Block Changes</TabsTrigger>
          <TabsTrigger value="synthetic">Synthetic Backups</TabsTrigger>
          <TabsTrigger value="health">Chain Health</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          {/* Key Metrics */}
          <div className="grid gap-4 md:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Storage Savings</CardTitle>
                <TrendingDown className="h-4 w-4 text-green-600" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">
                  {data.summary.storage_savings.toFixed(1)}%
                </div>
                <Progress value={data.summary.storage_savings} className="mt-2 bg-green-100" />
                <p className="text-xs text-muted-foreground mt-2">
                  {formatBytes(data.summary.compressed_size)} of {formatBytes(data.summary.total_size)}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
                <Database className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {data.summary.total_backups}
                </div>
                <div className="flex gap-2 mt-2">
                  <Badge variant="default" className="text-xs">{data.summary.full_backups} Full</Badge>
                  <Badge variant="secondary" className="text-xs">{data.summary.incremental_backups} Incr</Badge>
                  <Badge variant="outline" className="text-xs">{data.summary.synthetic_backups} Synth</Badge>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Avg Chain Length</CardTitle>
                <Link2 className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {data.summary.average_chain_length}
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  backups per chain
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Compressed Size</CardTitle>
                <Layers className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {formatBytes(data.summary.compressed_size)}
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  from {formatBytes(data.summary.total_size)} total
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Trends */}
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Storage Growth Trend</CardTitle>
                <CardDescription>Compressed storage usage over time</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={data.trends}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip />
                    <Line
                      type="monotone"
                      dataKey="storage_used"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      name="Storage (GB)"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Change Ratio Trend</CardTitle>
                <CardDescription>Block change percentage over time</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={data.trends}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip />
                    <Line
                      type="monotone"
                      dataKey="change_ratio"
                      stroke="#10b981"
                      strokeWidth={2}
                      name="Change Ratio (%)"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Backup Chains Tab */}
        <TabsContent value="chains" className="space-y-4">
          {data.chains.map((chain) => (
            <Card key={chain.database_id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>{chain.database_name}</CardTitle>
                    <CardDescription>Chain Length: {chain.chain_length} backups</CardDescription>
                  </div>
                  <div className="flex gap-2">
                    <Badge variant={getStatusColor(chain.health_status)}>
                      {chain.health_status.toUpperCase()}
                    </Badge>
                    <Badge variant="outline">
                      {chain.savings_ratio.toFixed(1)}% savings
                    </Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {/* Base Backup */}
                <div className="space-y-3">
                  <div className="flex items-center gap-3 p-3 border rounded-lg bg-blue-50 dark:bg-blue-950">
                    <Database className="h-5 w-5 text-blue-600" />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{chain.base_backup.type === 'synthetic' ? 'Synthetic' : 'Full'} Backup</span>
                        <Badge variant={getBackupTypeColor(chain.base_backup.type)} className="text-xs">
                          {chain.base_backup.type.toUpperCase()}
                        </Badge>
                      </div>
                      <div className="text-sm text-muted-foreground mt-1">
                        {formatBytes(chain.base_backup.compressed_size)} • {chain.base_backup.total_blocks.toLocaleString()} blocks • {new Date(chain.base_backup.created_at).toLocaleString()}
                      </div>
                    </div>
                  </div>

                  {/* Incrementals */}
                  {chain.incrementals.map((incr, idx) => (
                    <div key={incr.id} className="flex items-center gap-3 ml-8">
                      <div className="flex flex-col items-center">
                        <div className="h-4 w-0.5 bg-gray-300"></div>
                        <Zap className="h-4 w-4 text-gray-400" />
                        {idx < chain.incrementals.length - 1 && <div className="h-4 w-0.5 bg-gray-300"></div>}
                      </div>
                      <div className="flex-1 p-3 border rounded-lg">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-sm">Incremental #{idx + 1}</span>
                          <Badge variant="secondary" className="text-xs">
                            {incr.changed_blocks.toLocaleString()} changed blocks
                          </Badge>
                        </div>
                        <div className="text-xs text-muted-foreground mt-1">
                          {formatBytes(incr.compressed_size)} • {new Date(incr.created_at).toLocaleString()} • {incr.duration}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Summary */}
                <div className="mt-4 pt-4 border-t grid grid-cols-3 gap-4 text-sm">
                  <div>
                    <div className="text-muted-foreground">Total Size</div>
                    <div className="font-medium">{formatBytes(chain.total_size)}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Compressed</div>
                    <div className="font-medium">{formatBytes(chain.compressed_size)}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Savings</div>
                    <div className="font-medium text-green-600">{chain.savings_ratio.toFixed(1)}%</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Block Changes Tab */}
        <TabsContent value="changes" className="space-y-4">
          <div className="grid gap-4">
            {data.change_stats.map((stat) => (
              <Card key={stat.database_id}>
                <CardHeader>
                  <CardTitle>Change Tracking Statistics</CardTitle>
                  <CardDescription>Database: {stat.database_id}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-4">
                    <div>
                      <div className="text-sm text-muted-foreground">Total Snapshots</div>
                      <div className="text-2xl font-bold">{stat.total_snapshots}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Total Blocks</div>
                      <div className="text-2xl font-bold">{stat.total_blocks.toLocaleString()}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Changed Blocks</div>
                      <div className="text-2xl font-bold">{stat.changed_blocks.toLocaleString()}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Change Ratio</div>
                      <div className="text-2xl font-bold">{stat.change_ratio.toFixed(2)}%</div>
                    </div>
                  </div>
                  <div className="mt-4">
                    <div className="flex justify-between text-sm mb-2">
                      <span>Changed Blocks</span>
                      <span>{stat.change_ratio.toFixed(2)}%</span>
                    </div>
                    <Progress value={stat.change_ratio} />
                  </div>
                  <div className="mt-4 text-sm text-muted-foreground">
                    Block Size: {stat.average_block_size} bytes ({(stat.average_block_size / 1024).toFixed(0)} KB)
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Synthetic Backups Tab */}
        <TabsContent value="synthetic" className="space-y-4">
          {data.synthetic_recommendations.length > 0 ? (
            <div className="grid gap-4">
              {data.synthetic_recommendations.map((rec) => (
                <Card key={rec.database_id} className={rec.recommended ? 'border-orange-200' : ''}>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <div>
                        <CardTitle>{rec.database_name}</CardTitle>
                        <CardDescription>{rec.reason}</CardDescription>
                      </div>
                      <div className="flex gap-2">
                        {rec.recommended && (
                          <Badge variant="default">
                            {rec.priority.toUpperCase()} PRIORITY
                          </Badge>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="grid gap-4 md:grid-cols-3 mb-4">
                      <div>
                        <div className="text-sm text-muted-foreground">Incremental Count</div>
                        <div className="text-lg font-semibold">{rec.incremental_count}</div>
                      </div>
                      <div>
                        <div className="text-sm text-muted-foreground">Base Backup Age</div>
                        <div className="text-lg font-semibold">{(rec.base_backup_age / 24).toFixed(0)} days</div>
                      </div>
                      <div>
                        <div className="text-sm text-muted-foreground">Estimated Size</div>
                        <div className="text-lg font-semibold">{formatBytes(rec.estimated_size)}</div>
                      </div>
                    </div>
                    {rec.recommended && (
                      <Button onClick={() => createSyntheticBackup(rec.database_id)}>
                        <Layers className="h-4 w-4 mr-2" />
                        Create Synthetic Full Backup
                      </Button>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-8 text-muted-foreground">
                  <CheckCircle2 className="h-12 w-12 mx-auto mb-2 text-green-500" />
                  <p>No synthetic backup recommendations at this time</p>
                  <p className="text-sm mt-1">All backup chains are within optimal parameters</p>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Chain Health Tab */}
        <TabsContent value="health" className="space-y-4">
          <div className="grid gap-4">
            {data.chain_health.map((health) => (
              <Card key={health.chain_id} className={health.status !== 'healthy' ? 'border-orange-200' : ''}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      {health.status === 'healthy' ? (
                        <CheckCircle2 className="h-6 w-6 text-green-500" />
                      ) : health.status === 'warning' ? (
                        <AlertTriangle className="h-6 w-6 text-orange-500" />
                      ) : (
                        <XCircle className="h-6 w-6 text-red-500" />
                      )}
                      <div>
                        <CardTitle>{health.database_name}</CardTitle>
                        <CardDescription>Chain ID: {health.chain_id}</CardDescription>
                      </div>
                    </div>
                    <Badge variant={getStatusColor(health.status)}>
                      {health.status.toUpperCase()}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-2 mb-4">
                    <div>
                      <div className="text-sm text-muted-foreground">Chain Length</div>
                      <div className="text-lg font-semibold">{health.chain_length} backups</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Chain Age</div>
                      <div className="text-lg font-semibold">{(health.chain_age / 24).toFixed(0)} days</div>
                    </div>
                  </div>

                  {health.issues.length > 0 && (
                    <div className="mb-4">
                      <div className="text-sm font-semibold mb-2">Issues:</div>
                      <ul className="space-y-1">
                        {health.issues.map((issue, idx) => (
                          <li key={idx} className="flex items-start gap-2 text-sm">
                            <AlertTriangle className="h-4 w-4 text-orange-500 mt-0.5" />
                            <span>{issue}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {health.recommendations.length > 0 && (
                    <div>
                      <div className="text-sm font-semibold mb-2">Recommendations:</div>
                      <ul className="space-y-1">
                        {health.recommendations.map((rec, idx) => (
                          <li key={idx} className="flex items-start gap-2 text-sm">
                            <Zap className="h-4 w-4 text-blue-500 mt-0.5" />
                            <span>{rec}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {health.issues.length === 0 && health.recommendations.length === 0 && (
                    <div className="text-sm text-muted-foreground italic">
                      Backup chain is healthy with no issues or recommendations
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
