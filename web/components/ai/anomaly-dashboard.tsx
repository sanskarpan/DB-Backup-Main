"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Brain, TrendingUp, AlertOctagon, Activity, Zap, Target } from 'lucide-react'
import { Line, LineChart, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

interface Anomaly {
  id: string
  backup_id: string
  database_name: string
  detected_at: string
  anomaly_type: string
  score: number
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  details: {
    expected_value: number
    actual_value: number
    deviation: number
  }
  recommendations: string[]
}

interface PredictedFailure {
  database_name: string
  failure_probability: number
  predicted_time: string
  reasons: string[]
  preventive_actions: string[]
  confidence: number
}

interface OptimizationRecommendation {
  database_name: string
  current_compression: string
  recommended_compression: string
  potential_savings: number
  optimal_window: string
  parallelism_recommendation: number
  estimated_cost_reduction: number
}

export default function AnomalyDashboard() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 60000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    const mockData = {
      stats: {
        total_anomalies: 15,
        critical_anomalies: 2,
        trend: -8.3,
        detection_rate: 98.5
      },
      recent_anomalies: [
        {
          id: 'anom-001',
          backup_id: 'backup-123',
          database_name: 'production-db',
          detected_at: new Date(Date.now() - 1800000).toISOString(),
          anomaly_type: 'size_anomaly',
          score: 85.2,
          severity: 'HIGH' as const,
          details: {
            expected_value: 5.2,
            actual_value: 8.9,
            deviation: 71.2
          },
          recommendations: ['Check for data growth', 'Review retention policy']
        },
        {
          id: 'anom-002',
          backup_id: 'backup-124',
          database_name: 'analytics-db',
          detected_at: new Date(Date.now() - 3600000).toISOString(),
          anomaly_type: 'duration_anomaly',
          score: 92.7,
          severity: 'CRITICAL' as const,
          details: {
            expected_value: 120,
            actual_value: 380,
            deviation: 216.7
          },
          recommendations: ['Investigate slow queries', 'Check database performance']
        }
      ],
      predicted_failures: [
        {
          database_name: 'staging-db',
          failure_probability: 78.5,
          predicted_time: new Date(Date.now() + 86400000).toISOString(),
          reasons: ['Disk space trend', 'Increasing backup duration'],
          preventive_actions: ['Free disk space', 'Optimize backup strategy'],
          confidence: 85.0
        }
      ],
      optimization_recommendations: [
        {
          database_name: 'production-db',
          current_compression: 'gzip',
          recommended_compression: 'zstd',
          potential_savings: 35.2,
          optimal_window: '02:00-04:00',
          parallelism_recommendation: 4,
          estimated_cost_reduction: 250.0
        }
      ],
      trend_data: Array.from({ length: 24 }, (_, i) => ({
        hour: `${i}:00`,
        anomalies: Math.floor(Math.random() * 10),
        expected: 5
      }))
    }
    setData(mockData)
    setLoading(false)
  }

  const getSeverityColor = (severity: string) => {
    const colors = {
      CRITICAL: 'text-red-600',
      HIGH: 'text-orange-600',
      MEDIUM: 'text-yellow-600',
      LOW: 'text-blue-600'
    }
    return colors[severity as keyof typeof colors] || 'text-gray-600'
  }

  if (loading || !data) {
    return <div className="flex items-center justify-center h-96"><Activity className="h-12 w-12 animate-spin" /></div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Brain className="h-8 w-8 text-primary" />
            AI Anomaly Detection
          </h1>
          <p className="text-muted-foreground mt-1">Machine learning-powered backup anomaly detection</p>
        </div>
        <Button onClick={fetchData}><Activity className="h-4 w-4 mr-2" />Refresh</Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Anomalies</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.total_anomalies}</div>
            <p className="text-xs text-muted-foreground">{data.stats.critical_anomalies} critical</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Detection Rate</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.stats.detection_rate}%</div>
            <p className="text-xs text-green-600">Highly accurate</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Predicted Failures</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.predicted_failures.length}</div>
            <p className="text-xs text-orange-600">Next 48 hours</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Optimizations</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.optimization_recommendations.length}</div>
            <p className="text-xs text-blue-600">Available</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="anomalies">
        <TabsList>
          <TabsTrigger value="anomalies">Recent Anomalies</TabsTrigger>
          <TabsTrigger value="predictions">Failure Predictions</TabsTrigger>
          <TabsTrigger value="optimization">Optimizations</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
        </TabsList>

        <TabsContent value="anomalies" className="space-y-4">
          {data.recent_anomalies.map((anomaly: Anomaly) => (
            <Card key={anomaly.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{anomaly.database_name}</CardTitle>
                  <Badge variant={anomaly.severity === 'CRITICAL' ? 'destructive' : 'default'}>
                    {anomaly.severity}
                  </Badge>
                </div>
                <CardDescription>{new Date(anomaly.detected_at).toLocaleString()}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Type</p>
                    <p className="text-sm font-medium capitalize">{anomaly.anomaly_type.replace(/_/g, ' ')}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Score</p>
                    <p className="text-sm font-medium">{anomaly.score.toFixed(1)}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Deviation</p>
                    <p className="text-sm font-medium text-red-600">{anomaly.details.deviation.toFixed(1)}%</p>
                  </div>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground mb-2">Recommendations</p>
                  <ul className="space-y-1">
                    {anomaly.recommendations.map((rec, idx) => (
                      <li key={idx} className="text-sm flex items-start gap-2">
                        <Target className="h-4 w-4 text-primary mt-0.5" />
                        {rec}
                      </li>
                    ))}
                  </ul>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="predictions" className="space-y-4">
          {data.predicted_failures.map((prediction: PredictedFailure, idx: number) => (
            <Card key={idx} className="border-l-4 border-l-orange-500">
              <CardHeader>
                <CardTitle>{prediction.database_name}</CardTitle>
                <CardDescription>
                  Predicted failure: {new Date(prediction.predicted_time).toLocaleString()}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Failure Probability</span>
                  <span className="text-lg font-bold text-orange-600">{prediction.failure_probability.toFixed(1)}%</span>
                </div>
                <div>
                  <p className="text-sm font-medium mb-2">Reasons</p>
                  <ul className="space-y-1">
                    {prediction.reasons.map((reason, i) => (
                      <li key={i} className="text-sm text-muted-foreground flex items-start gap-2">
                        <AlertOctagon className="h-4 w-4 text-orange-500 mt-0.5" />
                        {reason}
                      </li>
                    ))}
                  </ul>
                </div>
                <div>
                  <p className="text-sm font-medium mb-2">Preventive Actions</p>
                  <ul className="space-y-1">
                    {prediction.preventive_actions.map((action, i) => (
                      <li key={i} className="text-sm flex items-start gap-2">
                        <Zap className="h-4 w-4 text-green-500 mt-0.5" />
                        {action}
                      </li>
                    ))}
                  </ul>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="optimization" className="space-y-4">
          {data.optimization_recommendations.map((opt: OptimizationRecommendation, idx: number) => (
            <Card key={idx}>
              <CardHeader>
                <CardTitle>{opt.database_name}</CardTitle>
                <CardDescription>Potential savings: ${opt.estimated_cost_reduction}/month</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">Current Compression</p>
                    <p className="text-sm font-medium">{opt.current_compression}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Recommended</p>
                    <p className="text-sm font-medium text-green-600">{opt.recommended_compression}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Optimal Window</p>
                    <p className="text-sm font-medium">{opt.optimal_window}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Parallelism</p>
                    <p className="text-sm font-medium">{opt.parallelism_recommendation} threads</p>
                  </div>
                </div>
                <div className="bg-green-50 p-4 rounded-lg">
                  <p className="text-sm font-medium text-green-900">
                    Expected savings: {opt.potential_savings.toFixed(1)}% reduction in backup time
                  </p>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="trends">
          <Card>
            <CardHeader>
              <CardTitle>Anomaly Trends (24 Hours)</CardTitle>
              <CardDescription>Detected vs expected anomalies</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={400}>
                <LineChart data={data.trend_data}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="hour" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Line type="monotone" dataKey="anomalies" stroke="#ef4444" name="Detected" />
                  <Line type="monotone" dataKey="expected" stroke="#10b981" name="Expected" />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
