"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import {
  Cloud,
  CloudCog,
  ArrowRightLeft,
  Shield,
  TrendingDown,
  Activity,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Zap,
  Database,
  DollarSign,
  Clock,
  HardDrive
} from 'lucide-react'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, PieChart, Pie, Cell } from 'recharts'

const PROVIDER_COLORS = {
  'aws': '#FF9900',
  'azure': '#0089D6',
  'gcp': '#4285F4',
  'local': '#6B7280'
}

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444']

interface ProviderMetrics {
  provider: string
  cost_per_gb: number
  upload_speed_mbps: number
  download_speed_mbps: number
  availability: number
  latency_ms: number
  certifications: string[]
  data_residency: string[]
  active_backups: number
  total_storage_gb: number
  monthly_cost: number
}

interface MigrationJob {
  id: string
  database_name: string
  source_provider: string
  target_providers: string[]
  status: 'PENDING' | 'IN_PROGRESS' | 'COMPLETED' | 'FAILED'
  progress: number
  total_size_gb: number
  transferred_gb: number
  started_at: string
  estimated_completion: string
  error?: string
}

interface FailoverStatus {
  provider: string
  status: 'HEALTHY' | 'DEGRADED' | 'UNHEALTHY'
  last_check: string
  uptime_percentage: number
  consecutive_failures: number
  failover_ready: boolean
  backup_provider?: string
}

interface ProviderComparison {
  provider: string
  cost_score: number
  performance_score: number
  compliance_score: number
  overall_score: number
  monthly_cost: number
  avg_latency: number
  certifications_count: number
}

export default function CloudManagement() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [selectedStrategy, setSelectedStrategy] = useState<string>('balanced')
  const [migrationSource, setMigrationSource] = useState<string>('')
  const [migrationTargets, setMigrationTargets] = useState<string[]>([])

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    // Mock data - in production, this would fetch from the API
    const mockData = {
      providers: [
        {
          provider: 'aws',
          cost_per_gb: 0.023,
          upload_speed_mbps: 125.5,
          download_speed_mbps: 180.2,
          availability: 99.99,
          latency_ms: 45,
          certifications: ['SOC2', 'ISO27001', 'HIPAA'],
          data_residency: ['US', 'EU', 'APAC'],
          active_backups: 245,
          total_storage_gb: 12500,
          monthly_cost: 2875.50
        },
        {
          provider: 'azure',
          cost_per_gb: 0.021,
          upload_speed_mbps: 110.3,
          download_speed_mbps: 165.8,
          availability: 99.95,
          latency_ms: 52,
          certifications: ['SOC2', 'ISO27001', 'FedRAMP'],
          data_residency: ['US', 'EU'],
          active_backups: 189,
          total_storage_gb: 9800,
          monthly_cost: 2058.00
        },
        {
          provider: 'gcp',
          cost_per_gb: 0.020,
          upload_speed_mbps: 135.7,
          download_speed_mbps: 195.3,
          availability: 99.97,
          latency_ms: 38,
          certifications: ['SOC2', 'ISO27001', 'PCI-DSS'],
          data_residency: ['US', 'EU', 'APAC', 'LATAM'],
          active_backups: 156,
          total_storage_gb: 7650,
          monthly_cost: 1530.00
        },
        {
          provider: 'local',
          cost_per_gb: 0.015,
          upload_speed_mbps: 250.0,
          download_speed_mbps: 250.0,
          availability: 99.50,
          latency_ms: 5,
          certifications: ['Internal'],
          data_residency: ['US'],
          active_backups: 89,
          total_storage_gb: 4200,
          monthly_cost: 630.00
        }
      ],
      active_migrations: [
        {
          id: 'mig-001',
          database_name: 'production-db',
          source_provider: 'local',
          target_providers: ['aws', 'azure'],
          status: 'IN_PROGRESS' as const,
          progress: 67,
          total_size_gb: 2500,
          transferred_gb: 1675,
          started_at: new Date(Date.now() - 3600000).toISOString(),
          estimated_completion: new Date(Date.now() + 1800000).toISOString()
        },
        {
          id: 'mig-002',
          database_name: 'analytics-db',
          source_provider: 'aws',
          target_providers: ['gcp'],
          status: 'COMPLETED' as const,
          progress: 100,
          total_size_gb: 1800,
          transferred_gb: 1800,
          started_at: new Date(Date.now() - 7200000).toISOString(),
          estimated_completion: new Date(Date.now() - 1800000).toISOString()
        }
      ],
      failover_status: [
        {
          provider: 'aws',
          status: 'HEALTHY' as const,
          last_check: new Date(Date.now() - 300000).toISOString(),
          uptime_percentage: 99.99,
          consecutive_failures: 0,
          failover_ready: true,
          backup_provider: 'azure'
        },
        {
          provider: 'azure',
          status: 'HEALTHY' as const,
          last_check: new Date(Date.now() - 300000).toISOString(),
          uptime_percentage: 99.95,
          consecutive_failures: 0,
          failover_ready: true,
          backup_provider: 'gcp'
        },
        {
          provider: 'gcp',
          status: 'DEGRADED' as const,
          last_check: new Date(Date.now() - 300000).toISOString(),
          uptime_percentage: 98.50,
          consecutive_failures: 2,
          failover_ready: true,
          backup_provider: 'aws'
        },
        {
          provider: 'local',
          status: 'HEALTHY' as const,
          last_check: new Date(Date.now() - 300000).toISOString(),
          uptime_percentage: 99.50,
          consecutive_failures: 0,
          failover_ready: false
        }
      ],
      provider_comparison: [
        {
          provider: 'aws',
          cost_score: 75,
          performance_score: 85,
          compliance_score: 95,
          overall_score: 85,
          monthly_cost: 2875.50,
          avg_latency: 45,
          certifications_count: 3
        },
        {
          provider: 'azure',
          cost_score: 80,
          performance_score: 80,
          compliance_score: 90,
          overall_score: 83,
          monthly_cost: 2058.00,
          avg_latency: 52,
          certifications_count: 3
        },
        {
          provider: 'gcp',
          cost_score: 85,
          performance_score: 90,
          compliance_score: 88,
          overall_score: 88,
          monthly_cost: 1530.00,
          avg_latency: 38,
          certifications_count: 3
        },
        {
          provider: 'local',
          cost_score: 95,
          performance_score: 95,
          compliance_score: 60,
          overall_score: 83,
          monthly_cost: 630.00,
          avg_latency: 5,
          certifications_count: 1
        }
      ],
      distribution: [
        { name: 'AWS S3', backups: 245, storage: 12500 },
        { name: 'Azure', backups: 189, storage: 9800 },
        { name: 'GCP', backups: 156, storage: 7650 },
        { name: 'Local', backups: 89, storage: 4200 }
      ]
    }

    setData(mockData)
    setLoading(false)
  }

  const getStatusColor = (status: string) => {
    const colors = {
      HEALTHY: 'text-green-600',
      DEGRADED: 'text-yellow-600',
      UNHEALTHY: 'text-red-600',
      COMPLETED: 'text-green-600',
      IN_PROGRESS: 'text-blue-600',
      FAILED: 'text-red-600',
      PENDING: 'text-gray-600'
    }
    return colors[status as keyof typeof colors] || 'text-gray-600'
  }

  const getStatusBadgeVariant = (status: string): "default" | "destructive" | "outline" | "secondary" => {
    if (status === 'HEALTHY' || status === 'COMPLETED') return 'default'
    if (status === 'UNHEALTHY' || status === 'FAILED') return 'destructive'
    if (status === 'DEGRADED' || status === 'IN_PROGRESS') return 'secondary'
    return 'outline'
  }

  const startMigration = () => {
    console.log('Starting migration from', migrationSource, 'to', migrationTargets)
    // In production, this would call the API
  }

  if (loading || !data) {
    return <div className="flex items-center justify-center h-96"><Activity className="h-12 w-12 animate-spin" /></div>
  }

  const totalBackups = data.providers.reduce((sum: number, p: ProviderMetrics) => sum + p.active_backups, 0)
  const totalStorage = data.providers.reduce((sum: number, p: ProviderMetrics) => sum + p.total_storage_gb, 0)
  const totalMonthlyCost = data.providers.reduce((sum: number, p: ProviderMetrics) => sum + p.monthly_cost, 0)
  const avgAvailability = data.providers.reduce((sum: number, p: ProviderMetrics) => sum + p.availability, 0) / data.providers.length

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <CloudCog className="h-8 w-8 text-primary" />
            Multi-Cloud Management
          </h1>
          <p className="text-muted-foreground mt-1">Intelligent provider selection, migration, and failover</p>
        </div>
        <Button onClick={fetchData}><Activity className="h-4 w-4 mr-2" />Refresh</Button>
      </div>

      {/* Stats Overview */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalBackups}</div>
            <p className="text-xs text-muted-foreground">Across {data.providers.length} providers</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Storage</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(totalStorage / 1000).toFixed(1)} TB</div>
            <p className="text-xs text-muted-foreground">{totalStorage.toLocaleString()} GB</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Monthly Cost</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${totalMonthlyCost.toFixed(2)}</div>
            <p className="text-xs text-green-600">Avg ${(totalMonthlyCost / totalBackups).toFixed(2)}/backup</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Avg Availability</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{avgAvailability.toFixed(2)}%</div>
            <p className="text-xs text-muted-foreground">Multi-cloud resilience</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="providers">
        <TabsList>
          <TabsTrigger value="providers">Provider Overview</TabsTrigger>
          <TabsTrigger value="selection">Provider Selection</TabsTrigger>
          <TabsTrigger value="migration">Migration</TabsTrigger>
          <TabsTrigger value="failover">Failover Status</TabsTrigger>
          <TabsTrigger value="comparison">Cost Comparison</TabsTrigger>
        </TabsList>

        {/* Provider Overview Tab */}
        <TabsContent value="providers" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Backup Distribution</CardTitle>
                <CardDescription>Backups across providers</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={data.distribution}
                      dataKey="backups"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      outerRadius={100}
                      label
                    >
                      {data.distribution.map((_: any, index: number) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Storage Distribution</CardTitle>
                <CardDescription>Storage capacity by provider (GB)</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={data.distribution}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="storage" fill="#3b82f6" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4">
            {data.providers.map((provider: ProviderMetrics) => (
              <Card key={provider.provider}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="capitalize">{provider.provider}</CardTitle>
                    <Badge variant="outline">{provider.active_backups} backups</Badge>
                  </div>
                  <CardDescription>${provider.monthly_cost.toFixed(2)}/month</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <p className="text-xs text-muted-foreground">Cost/GB</p>
                      <p className="text-sm font-medium">${provider.cost_per_gb.toFixed(3)}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Upload Speed</p>
                      <p className="text-sm font-medium">{provider.upload_speed_mbps.toFixed(1)} Mbps</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Availability</p>
                      <p className="text-sm font-medium">{provider.availability}%</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Latency</p>
                      <p className="text-sm font-medium">{provider.latency_ms}ms</p>
                    </div>
                  </div>
                  <div className="mt-4">
                    <p className="text-xs text-muted-foreground mb-2">Certifications</p>
                    <div className="flex flex-wrap gap-2">
                      {provider.certifications.map((cert: string) => (
                        <Badge key={cert} variant="outline" className="text-xs">
                          <Shield className="h-3 w-3 mr-1" />
                          {cert}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Provider Selection Tab */}
        <TabsContent value="selection" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Intelligent Provider Selection</CardTitle>
              <CardDescription>Choose optimal providers based on your strategy</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-2 block">Selection Strategy</label>
                <Select value={selectedStrategy} onValueChange={setSelectedStrategy}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="cost">Cost Optimized</SelectItem>
                    <SelectItem value="performance">Performance Optimized</SelectItem>
                    <SelectItem value="compliance">Compliance Focused</SelectItem>
                    <SelectItem value="balanced">Balanced (Recommended)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Alert>
                <Zap className="h-4 w-4" />
                <AlertTitle>
                  {selectedStrategy === 'cost' && 'Cost Optimized Strategy'}
                  {selectedStrategy === 'performance' && 'Performance Optimized Strategy'}
                  {selectedStrategy === 'compliance' && 'Compliance Focused Strategy'}
                  {selectedStrategy === 'balanced' && 'Balanced Strategy'}
                </AlertTitle>
                <AlertDescription>
                  {selectedStrategy === 'cost' && 'Prioritizes lowest cost per GB. Best for non-critical data with flexible SLAs.'}
                  {selectedStrategy === 'performance' && 'Prioritizes upload/download speed and low latency. Best for time-sensitive backups.'}
                  {selectedStrategy === 'compliance' && 'Prioritizes certifications and data residency. Best for regulated industries.'}
                  {selectedStrategy === 'balanced' && 'Balances cost, performance, and compliance. Best for general use cases.'}
                </AlertDescription>
              </Alert>

              <div>
                <h3 className="text-sm font-medium mb-3">Recommended Providers</h3>
                <div className="space-y-2">
                  {data.provider_comparison
                    .sort((a: ProviderComparison, b: ProviderComparison) => {
                      if (selectedStrategy === 'cost') return b.cost_score - a.cost_score
                      if (selectedStrategy === 'performance') return b.performance_score - a.performance_score
                      if (selectedStrategy === 'compliance') return b.compliance_score - a.compliance_score
                      return b.overall_score - a.overall_score
                    })
                    .map((comp: ProviderComparison, idx: number) => (
                      <Card key={comp.provider} className={idx === 0 ? 'border-2 border-primary' : ''}>
                        <CardContent className="pt-4">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                              <div className={`w-2 h-2 rounded-full ${idx === 0 ? 'bg-green-500' : idx === 1 ? 'bg-blue-500' : 'bg-gray-400'}`}></div>
                              <span className="font-medium capitalize">{comp.provider}</span>
                              {idx === 0 && <Badge>Recommended</Badge>}
                            </div>
                            <div className="text-right">
                              <p className="text-sm font-medium">
                                Score: {
                                  selectedStrategy === 'cost' ? comp.cost_score :
                                  selectedStrategy === 'performance' ? comp.performance_score :
                                  selectedStrategy === 'compliance' ? comp.compliance_score :
                                  comp.overall_score
                                }
                              </p>
                              <p className="text-xs text-muted-foreground">${comp.monthly_cost.toFixed(2)}/mo</p>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Migration Tab */}
        <TabsContent value="migration" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Cloud Migration</CardTitle>
              <CardDescription>Migrate backups between cloud providers</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-2 block">Source Provider</label>
                <Select value={migrationSource} onValueChange={setMigrationSource}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select source" />
                  </SelectTrigger>
                  <SelectContent>
                    {data.providers.map((p: ProviderMetrics) => (
                      <SelectItem key={p.provider} value={p.provider} className="capitalize">
                        {p.provider} ({p.active_backups} backups)
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-2 block">Target Provider(s)</label>
                <Select onValueChange={(value) => setMigrationTargets([...migrationTargets, value])}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select target(s)" />
                  </SelectTrigger>
                  <SelectContent>
                    {data.providers
                      .filter((p: ProviderMetrics) => p.provider !== migrationSource)
                      .map((p: ProviderMetrics) => (
                        <SelectItem key={p.provider} value={p.provider} className="capitalize">
                          {p.provider}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                {migrationTargets.length > 0 && (
                  <div className="flex flex-wrap gap-2 mt-2">
                    {migrationTargets.map((target) => (
                      <Badge key={target} variant="outline" className="capitalize">
                        {target}
                        <button
                          onClick={() => setMigrationTargets(migrationTargets.filter(t => t !== target))}
                          className="ml-2 text-muted-foreground hover:text-foreground"
                        >
                          ×
                        </button>
                      </Badge>
                    ))}
                  </div>
                )}
              </div>

              <Button
                onClick={startMigration}
                disabled={!migrationSource || migrationTargets.length === 0}
                className="w-full"
              >
                <ArrowRightLeft className="h-4 w-4 mr-2" />
                Start Migration
              </Button>
            </CardContent>
          </Card>

          <div>
            <h3 className="text-lg font-semibold mb-3">Active Migrations</h3>
            <div className="space-y-4">
              {data.active_migrations.map((migration: MigrationJob) => (
                <Card key={migration.id}>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base">{migration.database_name}</CardTitle>
                      <Badge variant={getStatusBadgeVariant(migration.status)}>
                        {migration.status}
                      </Badge>
                    </div>
                    <CardDescription>
                      <span className="capitalize">{migration.source_provider}</span> → {migration.target_providers.map(t => <span key={t} className="capitalize">{t}</span>).reduce((prev, curr) => <>{prev}, {curr}</>)}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium">Progress</span>
                        <span className="text-sm text-muted-foreground">
                          {migration.transferred_gb.toFixed(1)} / {migration.total_size_gb.toFixed(1)} GB
                        </span>
                      </div>
                      <Progress value={migration.progress} />
                      <p className="text-xs text-muted-foreground mt-1">{migration.progress}% complete</p>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          Started
                        </p>
                        <p className="text-sm font-medium">{new Date(migration.started_at).toLocaleString()}</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {migration.status === 'COMPLETED' ? 'Completed' : 'ETA'}
                        </p>
                        <p className="text-sm font-medium">{new Date(migration.estimated_completion).toLocaleString()}</p>
                      </div>
                    </div>

                    {migration.error && (
                      <Alert variant="destructive">
                        <AlertTriangle className="h-4 w-4" />
                        <AlertDescription>{migration.error}</AlertDescription>
                      </Alert>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        </TabsContent>

        {/* Failover Status Tab */}
        <TabsContent value="failover" className="space-y-4">
          <Alert>
            <Shield className="h-4 w-4" />
            <AlertTitle>Automatic Failover Enabled</AlertTitle>
            <AlertDescription>
              System will automatically switch to backup providers if primary providers become unhealthy.
            </AlertDescription>
          </Alert>

          <div className="grid gap-4">
            {data.failover_status.map((status: FailoverStatus) => (
              <Card key={status.provider} className={
                status.status === 'UNHEALTHY' ? 'border-l-4 border-l-red-500' :
                status.status === 'DEGRADED' ? 'border-l-4 border-l-yellow-500' :
                'border-l-4 border-l-green-500'
              }>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="capitalize">{status.provider}</CardTitle>
                    <Badge variant={getStatusBadgeVariant(status.status)}>
                      {status.status === 'HEALTHY' && <CheckCircle2 className="h-3 w-3 mr-1" />}
                      {status.status === 'DEGRADED' && <AlertTriangle className="h-3 w-3 mr-1" />}
                      {status.status === 'UNHEALTHY' && <XCircle className="h-3 w-3 mr-1" />}
                      {status.status}
                    </Badge>
                  </div>
                  <CardDescription>
                    Last checked: {new Date(status.last_check).toLocaleString()}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <p className="text-xs text-muted-foreground">Uptime</p>
                      <p className="text-sm font-medium">{status.uptime_percentage.toFixed(2)}%</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Consecutive Failures</p>
                      <p className={`text-sm font-medium ${status.consecutive_failures > 0 ? 'text-red-600' : 'text-green-600'}`}>
                        {status.consecutive_failures}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Failover Ready</p>
                      <p className="text-sm font-medium">
                        {status.failover_ready ? (
                          <span className="text-green-600">Yes</span>
                        ) : (
                          <span className="text-gray-600">No</span>
                        )}
                      </p>
                    </div>
                  </div>

                  {status.backup_provider && (
                    <div className="bg-blue-50 p-3 rounded-lg">
                      <p className="text-sm font-medium text-blue-900">
                        Backup Provider: <span className="capitalize">{status.backup_provider}</span>
                      </p>
                      <p className="text-xs text-blue-700 mt-1">
                        Will automatically failover if {status.provider} becomes unhealthy
                      </p>
                    </div>
                  )}

                  {status.status === 'DEGRADED' && (
                    <Alert>
                      <AlertTriangle className="h-4 w-4" />
                      <AlertDescription>
                        Provider experiencing intermittent issues. Monitoring closely.
                      </AlertDescription>
                    </Alert>
                  )}

                  {status.status === 'UNHEALTHY' && (
                    <Alert variant="destructive">
                      <XCircle className="h-4 w-4" />
                      <AlertDescription>
                        Provider is unhealthy. Failover to {status.backup_provider} in progress.
                      </AlertDescription>
                    </Alert>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Cost Comparison Tab */}
        <TabsContent value="comparison" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Provider Cost Comparison</CardTitle>
              <CardDescription>Monthly costs and cost-effectiveness metrics</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={400}>
                <BarChart data={data.provider_comparison}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="provider" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="monthly_cost" fill="#3b82f6" name="Monthly Cost ($)" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Performance Metrics</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={data.provider_comparison}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="provider" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="performance_score" fill="#10b981" name="Performance Score" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Compliance Scores</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={data.provider_comparison}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="provider" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="compliance_score" fill="#f59e0b" name="Compliance Score" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Detailed Comparison</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {data.provider_comparison.map((comp: ProviderComparison) => (
                  <div key={comp.provider} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-semibold capitalize">{comp.provider}</h3>
                      <Badge>Overall Score: {comp.overall_score}</Badge>
                    </div>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <DollarSign className="h-3 w-3" />
                          Cost Score
                        </p>
                        <p className="text-sm font-medium">{comp.cost_score}</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <Zap className="h-3 w-3" />
                          Performance
                        </p>
                        <p className="text-sm font-medium">{comp.performance_score}</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <Shield className="h-3 w-3" />
                          Compliance
                        </p>
                        <p className="text-sm font-medium">{comp.compliance_score}</p>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          Avg Latency
                        </p>
                        <p className="text-sm font-medium">{comp.avg_latency}ms</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Alert>
            <TrendingDown className="h-4 w-4" />
            <AlertTitle>Cost Optimization Opportunity</AlertTitle>
            <AlertDescription>
              Migrating 30% of workloads from AWS to GCP could save approximately $400/month
              while maintaining performance and compliance requirements.
            </AlertDescription>
          </Alert>
        </TabsContent>
      </Tabs>
    </div>
  )
}
