"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  DollarSign, TrendingDown, TrendingUp, ArrowRight, Cloud,
  Server, Database, AlertCircle, CheckCircle, Info, Loader2,
  Download, RefreshCw, BarChart3, PieChart as PieChartIcon
} from 'lucide-react'
import {
  BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  Area, AreaChart, ComposedChart
} from 'recharts'

const PROVIDER_COLORS: Record<string, string> = {
  aws: '#FF9900',
  azure: '#0078D4',
  gcp: '#4285F4',
  local: '#6B7280'
}

const TIER_COLORS: Record<string, string> = {
  hot: '#EF4444',
  warm: '#F59E0b',
  cold: '#3B82F6',
  archive: '#6B7280'
}

interface ProviderCost {
  provider: string
  total_cost: number
  storage_cost: number
  transfer_cost: number
  compute_cost: number
  cost_per_gb: number
  bandwidth_gb: number
  regions_count: number
}

interface CloudComparison {
  database_id: string
  database_name: string
  current_size_gb: number
  providers: {
    name: string
    monthly_cost: number
    storage_cost: number
    bandwidth_cost: number
    cost_per_gb: number
    tier_recommendations: {
      tier: string
      monthly_cost: number
      retrieval_time: string
    }[]
  }[]
  recommendation: {
    cheapest_provider: string
    current_provider: string
    potential_savings: number
    break_even_days: number
    migration_cost: number
  }
}

interface TierRecommendation {
  database: string
  current_tier: string
  recommended_tier: string
  current_cost: number
  recommended_cost: number
  monthly_savings: number
  access_frequency: number
  last_accessed_days: number
}

interface CostTrend {
  date: string
  aws: number
  azure: number
  gcp: number
  local: number
  total: number
}

interface BudgetAlert {
  scope: string
  budget: number
  spent: number
  percentage: number
  status: 'ok' | 'warning' | 'critical'
  forecast_overage: number
}

export default function MultiCloudCostComparison() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [providerCosts, setProviderCosts] = useState<ProviderCost[]>([])
  const [comparisons, setComparisons] = useState<CloudComparison[]>([])
  const [tierRecommendations, setTierRecommendations] = useState<TierRecommendation[]>([])
  const [costTrends, setCostTrends] = useState<CostTrend[]>([])
  const [budgetAlerts, setBudgetAlerts] = useState<BudgetAlert[]>([])
  const [selectedDatabase, setSelectedDatabase] = useState<string>('all')
  const [selectedPeriod, setSelectedPeriod] = useState<'7d' | '30d' | '90d'>('30d')
  const [totalSavingsOpportunity, setTotalSavingsOpportunity] = useState(0)

  useEffect(() => {
    fetchCostData()
  }, [selectedPeriod])

  const fetchCostData = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch provider costs (mock for now - replace with actual API)
      const providerData: ProviderCost[] = [
        {
          provider: 'aws',
          total_cost: 2156.30,
          storage_cost: 1800.00,
          transfer_cost: 256.30,
          compute_cost: 100.00,
          cost_per_gb: 0.023,
          bandwidth_gb: 512.6,
          regions_count: 3
        },
        {
          provider: 'azure',
          total_cost: 1234.50,
          storage_cost: 1050.00,
          transfer_cost: 134.50,
          compute_cost: 50.00,
          cost_per_gb: 0.020,
          bandwidth_gb: 269.0,
          regions_count: 2
        },
        {
          provider: 'gcp',
          total_cost: 890.12,
          storage_cost: 750.00,
          transfer_cost: 90.12,
          compute_cost: 50.00,
          cost_per_gb: 0.018,
          bandwidth_gb: 180.24,
          regions_count: 2
        },
        {
          provider: 'local',
          total_cost: 286.90,
          storage_cost: 250.00,
          transfer_cost: 0,
          compute_cost: 36.90,
          cost_per_gb: 0.005,
          bandwidth_gb: 0,
          regions_count: 1
        }
      ]
      setProviderCosts(providerData)

      // Fetch cloud comparisons (mock for now)
      const comparisonData: CloudComparison[] = [
        {
          database_id: 'db-001',
          database_name: 'production-db',
          current_size_gb: 2500,
          providers: [
            {
              name: 'aws',
              monthly_cost: 1245.00,
              storage_cost: 1100.00,
              bandwidth_cost: 145.00,
              cost_per_gb: 0.023,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 1245.00, retrieval_time: 'instant' },
                { tier: 'warm', monthly_cost: 935.00, retrieval_time: '< 1 hour' },
                { tier: 'cold', monthly_cost: 625.00, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 312.50, retrieval_time: '< 48 hours' }
              ]
            },
            {
              name: 'azure',
              monthly_cost: 987.00,
              storage_cost: 875.00,
              bandwidth_cost: 112.00,
              cost_per_gb: 0.020,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 987.00, retrieval_time: 'instant' },
                { tier: 'warm', monthly_cost: 750.00, retrieval_time: '< 1 hour' },
                { tier: 'cold', monthly_cost: 500.00, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 250.00, retrieval_time: '< 48 hours' }
              ]
            },
            {
              name: 'gcp',
              monthly_cost: 875.00,
              storage_cost: 775.00,
              bandwidth_cost: 100.00,
              cost_per_gb: 0.018,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 875.00, retrieval_time: 'instant' },
                { tier: 'warm', monthly_cost: 665.00, retrieval_time: '< 1 hour' },
                { tier: 'cold', monthly_cost: 445.00, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 220.00, retrieval_time: '< 48 hours' }
              ]
            }
          ],
          recommendation: {
            cheapest_provider: 'gcp',
            current_provider: 'aws',
            potential_savings: 370.00,
            break_even_days: 15,
            migration_cost: 185.00
          }
        },
        {
          database_id: 'db-002',
          database_name: 'analytics-db',
          current_size_gb: 1800,
          providers: [
            {
              name: 'aws',
              monthly_cost: 980.50,
              storage_cost: 865.00,
              bandwidth_cost: 115.50,
              cost_per_gb: 0.023,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 980.50, retrieval_time: 'instant' },
                { tier: 'cold', monthly_cost: 490.25, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 245.12, retrieval_time: '< 48 hours' }
              ]
            },
            {
              name: 'azure',
              monthly_cost: 735.00,
              storage_cost: 650.00,
              bandwidth_cost: 85.00,
              cost_per_gb: 0.020,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 735.00, retrieval_time: 'instant' },
                { tier: 'cold', monthly_cost: 367.50, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 183.75, retrieval_time: '< 48 hours' }
              ]
            },
            {
              name: 'gcp',
              monthly_cost: 650.00,
              storage_cost: 575.00,
              bandwidth_cost: 75.00,
              cost_per_gb: 0.018,
              tier_recommendations: [
                { tier: 'hot', monthly_cost: 650.00, retrieval_time: 'instant' },
                { tier: 'cold', monthly_cost: 325.00, retrieval_time: '< 12 hours' },
                { tier: 'archive', monthly_cost: 162.50, retrieval_time: '< 48 hours' }
              ]
            }
          ],
          recommendation: {
            cheapest_provider: 'azure',
            current_provider: 'azure',
            potential_savings: 0,
            break_even_days: 0,
            migration_cost: 0
          }
        }
      ]
      setComparisons(comparisonData)

      // Calculate total savings opportunity
      const totalSavings = comparisonData.reduce((sum, comp) => sum + comp.recommendation.potential_savings, 0)
      setTotalSavingsOpportunity(totalSavings)

      // Fetch tier recommendations
      const tierData: TierRecommendation[] = [
        {
          database: 'production-db',
          current_tier: 'hot',
          recommended_tier: 'warm',
          current_cost: 1245.00,
          recommended_cost: 935.00,
          monthly_savings: 310.00,
          access_frequency: 45,
          last_accessed_days: 3
        },
        {
          database: 'analytics-db',
          current_tier: 'hot',
          recommended_tier: 'cold',
          current_cost: 735.00,
          recommended_cost: 367.50,
          monthly_savings: 367.50,
          access_frequency: 12,
          last_accessed_days: 15
        },
        {
          database: 'staging-db',
          current_tier: 'warm',
          recommended_tier: 'archive',
          current_cost: 456.20,
          recommended_cost: 114.05,
          monthly_savings: 342.15,
          access_frequency: 2,
          last_accessed_days: 45
        }
      ]
      setTierRecommendations(tierData)

      // Generate cost trends
      const trends: CostTrend[] = Array.from({ length: 30 }, (_, i) => ({
        date: new Date(Date.now() - (29 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
        aws: 70 + Math.random() * 15,
        azure: 40 + Math.random() * 10,
        gcp: 28 + Math.random() * 8,
        local: 9 + Math.random() * 3,
        total: 0
      }))
      trends.forEach(t => {
        t.total = t.aws + t.azure + t.gcp + t.local
      })
      setCostTrends(trends)

      // Budget alerts
      const alerts: BudgetAlert[] = [
        {
          scope: 'Total Monthly',
          budget: 5000,
          spent: 4567.82,
          percentage: 91.4,
          status: 'critical',
          forecast_overage: 234.18
        },
        {
          scope: 'AWS',
          budget: 2500,
          spent: 2156.30,
          percentage: 86.3,
          status: 'warning',
          forecast_overage: 0
        },
        {
          scope: 'Azure',
          budget: 1500,
          spent: 1234.50,
          percentage: 82.3,
          status: 'ok',
          forecast_overage: 0
        }
      ]
      setBudgetAlerts(alerts)

    } catch (err: any) {
      setError(err.message || 'Failed to fetch cost data')
    } finally {
      setLoading(false)
    }
  }

  const formatCurrency = (amount: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2
    }).format(amount)
  }

  const getAlertVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (status) {
      case 'ok': return 'default'
      case 'warning': return 'secondary'
      case 'critical': return 'destructive'
      default: return 'outline'
    }
  }

  const getAlertIcon = (status: string) => {
    switch (status) {
      case 'ok': return <CheckCircle className="h-5 w-5 text-green-600" />
      case 'warning': return <AlertCircle className="h-5 w-5 text-yellow-600" />
      case 'critical': return <AlertCircle className="h-5 w-5 text-red-600" />
      default: return <Info className="h-5 w-5 text-gray-600" />
    }
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Cloud className="h-8 w-8" />
            Multi-Cloud Cost Comparison
          </h1>
          <p className="text-muted-foreground mt-2">
            Compare costs across AWS, Azure, GCP and optimize spending
          </p>
        </div>
        <div className="flex gap-2">
          <Select value={selectedPeriod} onValueChange={(v) => setSelectedPeriod(v as any)}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="7d">7 Days</SelectItem>
              <SelectItem value="30d">30 Days</SelectItem>
              <SelectItem value="90d">90 Days</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={fetchCostData} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <Card className="border-red-500">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 text-red-600">
              <AlertCircle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Overview Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Cost (MTD)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatCurrency(providerCosts.reduce((sum, p) => sum + p.total_cost, 0))}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Across all providers
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Potential Savings</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {formatCurrency(totalSavingsOpportunity)}
            </div>
            <div className="flex items-center gap-1 mt-1">
              <TrendingDown className="h-4 w-4 text-green-600" />
              <p className="text-xs text-green-600">
                {((totalSavingsOpportunity / providerCosts.reduce((sum, p) => sum + p.total_cost, 0)) * 100).toFixed(1)}% reduction
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Cost Per GB</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatCurrency(
                providerCosts.reduce((sum, p) => sum + p.total_cost, 0) /
                providerCosts.reduce((sum, p) => sum + (p.bandwidth_gb || 1), 0)
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Average across all storage
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Providers</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{providerCosts.length}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {providerCosts.reduce((sum, p) => sum + p.regions_count, 0)} regions total
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Budget Alerts */}
      {budgetAlerts.length > 0 && (
        <div className="grid gap-4 md:grid-cols-3">
          {budgetAlerts.map((alert, idx) => (
            <Card key={idx} className={`border-l-4 ${
              alert.status === 'critical' ? 'border-l-red-500' :
              alert.status === 'warning' ? 'border-l-yellow-500' :
              'border-l-green-500'
            }`}>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-medium">{alert.scope}</CardTitle>
                  {getAlertIcon(alert.status)}
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-muted-foreground">Budget</span>
                      <span className="font-medium">{formatCurrency(alert.budget)}</span>
                    </div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-muted-foreground">Spent</span>
                      <span className="font-medium">{formatCurrency(alert.spent)}</span>
                    </div>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${
                        alert.status === 'critical' ? 'bg-red-500' :
                        alert.status === 'warning' ? 'bg-yellow-500' :
                        'bg-green-500'
                      }`}
                      style={{ width: `${Math.min(alert.percentage, 100)}%` }}
                    />
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium">{alert.percentage.toFixed(1)}% used</span>
                    {alert.forecast_overage > 0 && (
                      <Badge variant="destructive" className="text-xs">
                        Forecast: +{formatCurrency(alert.forecast_overage)}
                      </Badge>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Main Tabs */}
      <Tabs defaultValue="comparison" className="w-full">
        <TabsList className="grid w-full grid-cols-4 max-w-2xl">
          <TabsTrigger value="comparison">Provider Comparison</TabsTrigger>
          <TabsTrigger value="tier">Tier Optimization</TabsTrigger>
          <TabsTrigger value="trends">Cost Trends</TabsTrigger>
          <TabsTrigger value="breakdown">Cost Breakdown</TabsTrigger>
        </TabsList>

        {/* Provider Comparison Tab */}
        <TabsContent value="comparison" className="space-y-4">
          {/* Provider Overview */}
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Cost by Provider</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={providerCosts}
                      dataKey="total_cost"
                      nameKey="provider"
                      cx="50%"
                      cy="50%"
                      outerRadius={100}
                      label={(entry) => `${entry.provider.toUpperCase()}: ${formatCurrency(entry.total_cost)}`}
                    >
                      {providerCosts.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={PROVIDER_COLORS[entry.provider] || '#6B7280'} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Cost Per GB Comparison</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={providerCosts}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="provider" />
                    <YAxis />
                    <Tooltip formatter={(value: number) => formatCurrency(value)} />
                    <Bar dataKey="cost_per_gb" fill="#3B82F6" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>

          {/* Database-specific Comparisons */}
          {comparisons.map((comparison) => (
            <Card key={comparison.database_id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Database className="h-5 w-5" />
                      {comparison.database_name}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      {comparison.current_size_gb.toLocaleString()} GB
                    </CardDescription>
                  </div>
                  {comparison.recommendation.potential_savings > 0 && (
                    <Badge variant="outline" className="text-green-600 border-green-600">
                      Save {formatCurrency(comparison.recommendation.potential_savings)}/mo
                    </Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Provider Comparison */}
                  <div className="grid gap-3">
                    {comparison.providers.map((provider) => (
                      <div
                        key={provider.name}
                        className={`p-4 rounded-lg border-2 ${
                          provider.name === comparison.recommendation.cheapest_provider
                            ? 'border-green-500 bg-green-50'
                            : provider.name === comparison.recommendation.current_provider
                            ? 'border-blue-500 bg-blue-50'
                            : 'border-gray-200'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div
                              className="w-3 h-3 rounded-full"
                              style={{ backgroundColor: PROVIDER_COLORS[provider.name] }}
                            />
                            <div>
                              <div className="font-semibold uppercase">{provider.name}</div>
                              <div className="text-sm text-muted-foreground">
                                {formatCurrency(provider.cost_per_gb)}/GB
                              </div>
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="text-2xl font-bold">
                              {formatCurrency(provider.monthly_cost)}
                            </div>
                            <div className="text-sm text-muted-foreground">per month</div>
                          </div>
                        </div>
                        <div className="mt-3 grid grid-cols-3 gap-2 text-sm">
                          <div>
                            <span className="text-muted-foreground">Storage: </span>
                            <span className="font-medium">{formatCurrency(provider.storage_cost)}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">Bandwidth: </span>
                            <span className="font-medium">{formatCurrency(provider.bandwidth_cost)}</span>
                          </div>
                          <div className="text-right">
                            {provider.name === comparison.recommendation.cheapest_provider && (
                              <Badge variant="outline" className="text-green-600">Cheapest</Badge>
                            )}
                            {provider.name === comparison.recommendation.current_provider && (
                              <Badge variant="outline" className="text-blue-600">Current</Badge>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>

                  {/* Recommendation */}
                  {comparison.recommendation.potential_savings > 0 && (
                    <div className="p-4 bg-green-50 border border-green-200 rounded-lg">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <TrendingDown className="h-5 w-5 text-green-600" />
                            <span className="font-semibold text-green-900">Migration Recommendation</span>
                          </div>
                          <div className="space-y-2 text-sm">
                            <div>
                              <span className="text-gray-700">Migrate from </span>
                              <span className="font-medium">{comparison.recommendation.current_provider.toUpperCase()}</span>
                              <ArrowRight className="inline h-4 w-4 mx-2" />
                              <span className="font-medium">{comparison.recommendation.cheapest_provider.toUpperCase()}</span>
                            </div>
                            <div className="grid grid-cols-3 gap-2 mt-3">
                              <div>
                                <div className="text-xs text-gray-600">Monthly Savings</div>
                                <div className="font-bold text-green-600">
                                  {formatCurrency(comparison.recommendation.potential_savings)}
                                </div>
                              </div>
                              <div>
                                <div className="text-xs text-gray-600">Migration Cost</div>
                                <div className="font-semibold">
                                  {formatCurrency(comparison.recommendation.migration_cost)}
                                </div>
                              </div>
                              <div>
                                <div className="text-xs text-gray-600">Break-even</div>
                                <div className="font-semibold">
                                  {comparison.recommendation.break_even_days} days
                                </div>
                              </div>
                            </div>
                          </div>
                        </div>
                        <Button variant="default" className="ml-4">
                          <Download className="h-4 w-4 mr-2" />
                          Start Migration
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        {/* Tier Optimization Tab */}
        <TabsContent value="tier" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Storage Tier Recommendations</CardTitle>
              <CardDescription>
                Optimize costs by moving data to appropriate storage tiers based on access patterns
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {tierRecommendations.map((rec, idx) => (
                  <div key={idx} className="p-4 border rounded-lg">
                    <div className="flex items-center justify-between mb-3">
                      <div>
                        <div className="font-semibold text-lg">{rec.database}</div>
                        <div className="text-sm text-muted-foreground">
                          Access frequency: {rec.access_frequency}/month | Last accessed: {rec.last_accessed_days} days ago
                        </div>
                      </div>
                      <Badge variant="outline" className="text-green-600 border-green-600">
                        Save {formatCurrency(rec.monthly_savings)}/mo
                      </Badge>
                    </div>
                    <div className="grid grid-cols-3 gap-4 mb-3">
                      <div className="text-center p-3 bg-blue-50 rounded">
                        <div className="text-xs text-muted-foreground mb-1">Current Tier</div>
                        <Badge style={{ backgroundColor: TIER_COLORS[rec.current_tier] }}>
                          {rec.current_tier.toUpperCase()}
                        </Badge>
                        <div className="text-lg font-bold mt-2">{formatCurrency(rec.current_cost)}</div>
                      </div>
                      <div className="flex items-center justify-center">
                        <ArrowRight className="h-8 w-8 text-green-600" />
                      </div>
                      <div className="text-center p-3 bg-green-50 rounded">
                        <div className="text-xs text-muted-foreground mb-1">Recommended Tier</div>
                        <Badge style={{ backgroundColor: TIER_COLORS[rec.recommended_tier] }}>
                          {rec.recommended_tier.toUpperCase()}
                        </Badge>
                        <div className="text-lg font-bold text-green-600 mt-2">{formatCurrency(rec.recommended_cost)}</div>
                      </div>
                    </div>
                    <Button variant="outline" className="w-full">
                      Apply Recommendation
                    </Button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Cost Trends Tab */}
        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Cost Trends (Last 30 Days)</CardTitle>
              <CardDescription>
                Daily cost breakdown by provider
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={400}>
                <AreaChart data={costTrends}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" tickFormatter={(value) => new Date(value).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} />
                  <YAxis tickFormatter={(value) => `$${value}`} />
                  <Tooltip formatter={(value: number) => formatCurrency(value)} />
                  <Legend />
                  <Area type="monotone" dataKey="aws" stackId="1" stroke={PROVIDER_COLORS.aws} fill={PROVIDER_COLORS.aws} />
                  <Area type="monotone" dataKey="azure" stackId="1" stroke={PROVIDER_COLORS.azure} fill={PROVIDER_COLORS.azure} />
                  <Area type="monotone" dataKey="gcp" stackId="1" stroke={PROVIDER_COLORS.gcp} fill={PROVIDER_COLORS.gcp} />
                  <Area type="monotone" dataKey="local" stackId="1" stroke={PROVIDER_COLORS.local} fill={PROVIDER_COLORS.local} />
                </AreaChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Total Cost Trend</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={costTrends}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tickFormatter={(value) => new Date(value).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} />
                    <YAxis tickFormatter={(value) => `$${value}`} />
                    <Tooltip formatter={(value: number) => formatCurrency(value)} />
                    <Line type="monotone" dataKey="total" stroke="#3B82F6" strokeWidth={2} />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Provider Comparison</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <BarChart data={providerCosts}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="provider" />
                    <YAxis tickFormatter={(value) => `$${value}`} />
                    <Tooltip formatter={(value: number) => formatCurrency(value)} />
                    <Legend />
                    <Bar dataKey="storage_cost" fill="#3B82F6" name="Storage" />
                    <Bar dataKey="transfer_cost" fill="#10B981" name="Transfer" />
                    <Bar dataKey="compute_cost" fill="#F59E0B" name="Compute" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Cost Breakdown Tab */}
        <TabsContent value="breakdown" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {providerCosts.map((provider) => (
              <Card key={provider.provider}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <div
                      className="w-4 h-4 rounded-full"
                      style={{ backgroundColor: PROVIDER_COLORS[provider.provider] }}
                    />
                    {provider.provider.toUpperCase()}
                  </CardTitle>
                  <CardDescription>
                    {provider.regions_count} regions | {formatCurrency(provider.cost_per_gb)}/GB
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex justify-between items-center pb-2 border-b">
                    <span className="text-2xl font-bold">{formatCurrency(provider.total_cost)}</span>
                    <Badge>Total</Badge>
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Storage</span>
                      <span className="font-medium">{formatCurrency(provider.storage_cost)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Transfer</span>
                      <span className="font-medium">{formatCurrency(provider.transfer_cost)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Compute</span>
                      <span className="font-medium">{formatCurrency(provider.compute_cost)}</span>
                    </div>
                    {provider.bandwidth_gb > 0 && (
                      <div className="flex justify-between pt-2 border-t">
                        <span className="text-sm text-muted-foreground">Bandwidth</span>
                        <span className="font-medium">{provider.bandwidth_gb.toFixed(2)} GB</span>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
