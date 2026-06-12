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
  Eye,
  EyeOff,
  Shield,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Database,
  Hash,
  Shuffle,
  FileText,
  Settings,
  Activity,
  Lock
} from 'lucide-react'
import { BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

const STRATEGY_COLORS: Record<string, string> = {
  'redaction': '#ef4444',
  'hashing': '#f59e0b',
  'pseudonymization': '#10b981',
  'synthetic': '#3b82f6',
  'nullification': '#6b7280'
}

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']

interface MaskingRule {
  id: string
  table: string
  column: string
  strategy: string
  enabled: boolean
  pii_type: string
  confidence: number
  created_at: string
  last_applied: string
  records_masked: number
}

interface PIIDetection {
  table: string
  column: string
  pii_type: string
  detection_count: number
  sample_values: string[]
  confidence: number
  suggested_strategy: string
}

interface MaskingJob {
  id: string
  database: string
  tables: string[]
  status: string
  progress: number
  total_rows: number
  processed_rows: number
  masked_fields: number
  started_at: string
  estimated_completion: string
}

interface ReferentialRule {
  id: string
  source_table: string
  source_column: string
  target_table: string
  target_column: string
  strategy: string
}

export default function DataMaskingDashboard() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [selectedStrategy, setSelectedStrategy] = useState<string>('all')

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    // Mock data
    const mockData = {
      stats: {
        total_rules: 45,
        enabled_rules: 38,
        total_pii_detected: 126,
        masking_jobs_running: 2,
        avg_masking_time: 1.2,
        protection_coverage: 87.5
      },
      masking_rules: [
        {
          id: 'rule-001',
          table: 'users',
          column: 'email',
          strategy: 'pseudonymization',
          enabled: true,
          pii_type: 'email',
          confidence: 98.5,
          created_at: new Date(Date.now() - 86400000 * 15).toISOString(),
          last_applied: new Date(Date.now() - 3600000).toISOString(),
          records_masked: 125430
        },
        {
          id: 'rule-002',
          table: 'users',
          column: 'ssn',
          strategy: 'hashing',
          enabled: true,
          pii_type: 'ssn',
          confidence: 99.9,
          created_at: new Date(Date.now() - 86400000 * 20).toISOString(),
          last_applied: new Date(Date.now() - 3600000).toISOString(),
          records_masked: 125430
        },
        {
          id: 'rule-003',
          table: 'users',
          column: 'phone',
          strategy: 'redaction',
          enabled: true,
          pii_type: 'phone',
          confidence: 95.2,
          created_at: new Date(Date.now() - 86400000 * 10).toISOString(),
          last_applied: new Date(Date.now() - 3600000).toISOString(),
          records_masked: 98765
        },
        {
          id: 'rule-004',
          table: 'orders',
          column: 'credit_card',
          strategy: 'redaction',
          enabled: true,
          pii_type: 'credit_card',
          confidence: 99.5,
          created_at: new Date(Date.now() - 86400000 * 8).toISOString(),
          last_applied: new Date(Date.now() - 7200000).toISOString(),
          records_masked: 45678
        }
      ],
      pii_detections: [
        {
          table: 'customers',
          column: 'billing_address',
          pii_type: 'address',
          detection_count: 8945,
          sample_values: ['123 Main St, City, ST 12345', '456 Oak Ave, Town, ST 67890'],
          confidence: 92.3,
          suggested_strategy: 'synthetic'
        },
        {
          table: 'employees',
          column: 'date_of_birth',
          pii_type: 'date_of_birth',
          detection_count: 523,
          sample_values: ['1985-03-15', '1992-07-22'],
          confidence: 88.7,
          suggested_strategy: 'pseudonymization'
        }
      ],
      active_jobs: [
        {
          id: 'job-001',
          database: 'production',
          tables: ['users', 'orders'],
          status: 'running',
          progress: 67,
          total_rows: 500000,
          processed_rows: 335000,
          masked_fields: 892000,
          started_at: new Date(Date.now() - 300000).toISOString(),
          estimated_completion: new Date(Date.now() + 180000).toISOString()
        },
        {
          id: 'job-002',
          database: 'analytics',
          tables: ['events'],
          status: 'completed',
          progress: 100,
          total_rows: 1000000,
          processed_rows: 1000000,
          masked_fields: 2500000,
          started_at: new Date(Date.now() - 600000).toISOString(),
          estimated_completion: new Date(Date.now() - 60000).toISOString()
        }
      ],
      referential_rules: [
        {
          id: 'ref-001',
          source_table: 'users',
          source_column: 'user_id',
          target_table: 'orders',
          target_column: 'user_id',
          strategy: 'pseudonymization'
        },
        {
          id: 'ref-002',
          source_table: 'products',
          source_column: 'product_id',
          target_table: 'order_items',
          target_column: 'product_id',
          strategy: 'pseudonymization'
        }
      ],
      strategy_distribution: [
        { name: 'Redaction', value: 15, color: STRATEGY_COLORS.redaction },
        { name: 'Hashing', value: 12, color: STRATEGY_COLORS.hashing },
        { name: 'Pseudonymization', value: 18, color: STRATEGY_COLORS.pseudonymization },
        { name: 'Synthetic', value: 8, color: STRATEGY_COLORS.synthetic },
        { name: 'Nullification', value: 2, color: STRATEGY_COLORS.nullification }
      ],
      pii_distribution: [
        { type: 'Email', count: 35 },
        { type: 'Phone', count: 28 },
        { type: 'SSN', count: 12 },
        { type: 'Credit Card', count: 18 },
        { type: 'Address', count: 22 },
        { type: 'IP Address', count: 11 }
      ]
    }

    setData(mockData)
    setLoading(false)
  }

  const getStrategyIcon = (strategy: string) => {
    switch (strategy) {
      case 'redaction': return <EyeOff className="h-4 w-4" />
      case 'hashing': return <Hash className="h-4 w-4" />
      case 'pseudonymization': return <Shuffle className="h-4 w-4" />
      case 'synthetic': return <FileText className="h-4 w-4" />
      case 'nullification': return <XCircle className="h-4 w-4" />
      default: return <Settings className="h-4 w-4" />
    }
  }

  const getStatusBadgeVariant = (status: string): "default" | "destructive" | "outline" | "secondary" => {
    if (status === 'completed') return 'default'
    if (status === 'failed') return 'destructive'
    if (status === 'running') return 'secondary'
    return 'outline'
  }

  if (loading || !data) {
    return <div className="flex items-center justify-center h-96"><Activity className="h-12 w-12 animate-spin" /></div>
  }

  const filteredRules = selectedStrategy === 'all'
    ? data.masking_rules
    : data.masking_rules.filter((r: MaskingRule) => r.strategy === selectedStrategy)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Eye className="h-8 w-8 text-primary" />
            Data Masking & Anonymization
          </h1>
          <p className="text-muted-foreground mt-1">PII detection, column-level masking, and referential integrity</p>
        </div>
        <Button onClick={fetchData}><Activity className="h-4 w-4 mr-2" />Refresh</Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Masking Rules</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.total_rules}</div>
            <p className="text-xs text-green-600">{data.stats.enabled_rules} enabled</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">PII Detected</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.total_pii_detected}</div>
            <p className="text-xs text-muted-foreground">Across all databases</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Jobs</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.masking_jobs_running}</div>
            <p className="text-xs text-blue-600">In progress</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Protection Coverage</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.protection_coverage}%</div>
            <p className="text-xs text-green-600">Good coverage</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="rules">
        <TabsList>
          <TabsTrigger value="rules">Masking Rules</TabsTrigger>
          <TabsTrigger value="detections">PII Detections</TabsTrigger>
          <TabsTrigger value="jobs">Masking Jobs</TabsTrigger>
          <TabsTrigger value="referential">Referential Integrity</TabsTrigger>
          <TabsTrigger value="analytics">Analytics</TabsTrigger>
        </TabsList>

        {/* Masking Rules Tab */}
        <TabsContent value="rules" className="space-y-4">
          <div className="flex items-center gap-4 mb-4">
            <Select value={selectedStrategy} onValueChange={setSelectedStrategy}>
              <SelectTrigger className="w-64">
                <SelectValue placeholder="Filter by strategy" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Strategies</SelectItem>
                <SelectItem value="redaction">Redaction</SelectItem>
                <SelectItem value="hashing">Hashing</SelectItem>
                <SelectItem value="pseudonymization">Pseudonymization</SelectItem>
                <SelectItem value="synthetic">Synthetic</SelectItem>
                <SelectItem value="nullification">Nullification</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline">
              <Settings className="h-4 w-4 mr-2" />
              Add Rule
            </Button>
          </div>

          <div className="grid gap-4">
            {filteredRules.map((rule: MaskingRule) => (
              <Card key={rule.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base">{rule.table}.{rule.column}</CardTitle>
                      <CardDescription>
                        Detected as {rule.pii_type} with {rule.confidence.toFixed(1)}% confidence
                      </CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={rule.enabled ? 'default' : 'outline'}>
                        {rule.enabled ? <CheckCircle2 className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                        {rule.enabled ? 'Active' : 'Inactive'}
                      </Badge>
                      <Badge variant="secondary">
                        {getStrategyIcon(rule.strategy)}
                        <span className="ml-1 capitalize">{rule.strategy}</span>
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <p className="text-xs text-muted-foreground">Records Masked</p>
                      <p className="text-sm font-medium">{rule.records_masked.toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Created</p>
                      <p className="text-sm font-medium">{new Date(rule.created_at).toLocaleDateString()}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Last Applied</p>
                      <p className="text-sm font-medium">{new Date(rule.last_applied).toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">PII Type</p>
                      <Badge variant="outline" className="text-xs capitalize">{rule.pii_type.replace('_', ' ')}</Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* PII Detections Tab */}
        <TabsContent value="detections" className="space-y-4">
          <Alert>
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Unprotected PII Detected</AlertTitle>
            <AlertDescription>
              {data.pii_detections.length} columns with unprotected PII detected. Review and apply masking rules.
            </AlertDescription>
          </Alert>

          {data.pii_detections.map((detection: PIIDetection, idx: number) => (
            <Card key={idx} className="border-l-4 border-l-orange-500">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">{detection.table}.{detection.column}</CardTitle>
                    <CardDescription>{detection.detection_count.toLocaleString()} instances detected</CardDescription>
                  </div>
                  <Badge variant="outline" className="capitalize">{detection.pii_type.replace('_', ' ')}</Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Confidence Score</p>
                    <div className="flex items-center gap-2">
                      <Progress value={detection.confidence} className="flex-1" />
                      <span className="text-sm font-medium">{detection.confidence.toFixed(1)}%</span>
                    </div>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Suggested Strategy</p>
                    <Badge variant="secondary">
                      {getStrategyIcon(detection.suggested_strategy)}
                      <span className="ml-1 capitalize">{detection.suggested_strategy}</span>
                    </Badge>
                  </div>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground mb-2">Sample Values</p>
                  <div className="bg-muted p-3 rounded-lg space-y-1">
                    {detection.sample_values.map((sample, i) => (
                      <p key={i} className="text-sm font-mono">{sample}</p>
                    ))}
                  </div>
                </div>

                <Button variant="outline" size="sm">
                  <Shield className="h-4 w-4 mr-2" />
                  Apply Suggested Masking
                </Button>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Masking Jobs Tab */}
        <TabsContent value="jobs" className="space-y-4">
          {data.active_jobs.map((job: MaskingJob) => (
            <Card key={job.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">{job.database}</CardTitle>
                    <CardDescription>
                      {job.tables.join(', ')}
                    </CardDescription>
                  </div>
                  <Badge variant={getStatusBadgeVariant(job.status)}>
                    {job.status === 'running' && <Activity className="h-3 w-3 mr-1 animate-spin" />}
                    {job.status === 'completed' && <CheckCircle2 className="h-3 w-3 mr-1" />}
                    {job.status.toUpperCase()}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {job.status === 'running' && (
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-medium">Progress</span>
                      <span className="text-sm text-muted-foreground">
                        {job.processed_rows.toLocaleString()} / {job.total_rows.toLocaleString()} rows
                      </span>
                    </div>
                    <Progress value={job.progress} />
                    <p className="text-xs text-muted-foreground mt-1">{job.progress}% complete</p>
                  </div>
                )}

                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Masked Fields</p>
                    <p className="text-sm font-medium">{job.masked_fields.toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Started</p>
                    <p className="text-sm font-medium">{new Date(job.started_at).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">{job.status === 'completed' ? 'Completed' : 'ETA'}</p>
                    <p className="text-sm font-medium">{new Date(job.estimated_completion).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Tables</p>
                    <p className="text-sm font-medium">{job.tables.length}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Referential Integrity Tab */}
        <TabsContent value="referential" className="space-y-4">
          <Alert>
            <Lock className="h-4 w-4" />
            <AlertTitle>Referential Integrity Protection Enabled</AlertTitle>
            <AlertDescription>
              Ensuring consistent masking across foreign key relationships
            </AlertDescription>
          </Alert>

          <div className="grid gap-4">
            {data.referential_rules.map((rule: ReferentialRule) => (
              <Card key={rule.id}>
                <CardHeader>
                  <CardTitle className="text-base flex items-center gap-2">
                    {rule.source_table}.{rule.source_column}
                    <span className="text-muted-foreground">→</span>
                    {rule.target_table}.{rule.target_column}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">Masking Strategy</p>
                      <Badge variant="secondary" className="mt-1">
                        {getStrategyIcon(rule.strategy)}
                        <span className="ml-1 capitalize">{rule.strategy}</span>
                      </Badge>
                    </div>
                    <Badge variant="outline">
                      <CheckCircle2 className="h-3 w-3 mr-1 text-green-600" />
                      Integrity Preserved
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Analytics Tab */}
        <TabsContent value="analytics">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Strategy Distribution</CardTitle>
                <CardDescription>Masking strategies across all rules</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={data.strategy_distribution}
                      dataKey="value"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      outerRadius={100}
                      label
                    >
                      {data.strategy_distribution.map((entry: any, index: number) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>PII Type Distribution</CardTitle>
                <CardDescription>Detected PII types across columns</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={data.pii_distribution}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="type" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="count" fill="#3b82f6" />
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
