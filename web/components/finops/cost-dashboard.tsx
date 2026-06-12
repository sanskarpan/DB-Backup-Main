"use client"

import React, { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DollarSign, TrendingDown, AlertCircle, Database } from 'lucide-react'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, PieChart, Pie, Cell } from 'recharts'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444']

export default function CostDashboard() {
  const [data] = useState({
    overview: { total_cost: 4567.82, monthly_projection: 5234.00, savings_opportunities: 892.50, budget_used: 87.2 },
    by_provider: [
      { name: 'AWS S3', cost: 2156.30, percentage: 47.2 },
      { name: 'Azure', cost: 1234.50, percentage: 27.0 },
      { name: 'GCP', cost: 890.12, percentage: 19.5 },
      { name: 'Local', cost: 286.90, percentage: 6.3 }
    ],
    top_databases: [
      { name: 'prod-db-1', cost: 1245.00, size_gb: 2500, provider: 'AWS' },
      { name: 'analytics', cost: 980.50, size_gb: 1800, provider: 'Azure' },
      { name: 'staging', cost: 456.20, size_gb: 890, provider: 'GCP' }
    ],
    recommendations: [
      { database: 'prod-db-1', action: 'Switch to Azure', current_cost: 1245, estimated_cost: 987, savings: 258, breakeven_days: 15 },
      { database: 'analytics', action: 'Use cold storage', current_cost: 980, estimated_cost: 735, savings: 245, breakeven_days: 5 }
    ],
    trend: Array.from({ length: 30 }, (_, i) => ({ day: i + 1, cost: 150 + Math.random() * 50 }))
  })

  return (
    <div className="space-y-6">
      <div><h1 className="text-3xl font-bold flex items-center gap-2"><DollarSign className="h-8 w-8" />Cost Optimization</h1><p className="text-muted-foreground">FinOps dashboard and savings opportunities</p></div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Total Cost (MTD)</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">${data.overview.total_cost.toFixed(2)}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Monthly Projection</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">${data.overview.monthly_projection.toFixed(2)}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Potential Savings</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold text-green-600">${data.overview.savings_opportunities.toFixed(2)}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Budget Used</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">{data.overview.budget_used.toFixed(1)}%</div></CardContent></Card>
      </div>

      <Tabs defaultValue="overview"><TabsList><TabsTrigger value="overview">Overview</TabsTrigger><TabsTrigger value="recommendations">Recommendations</TabsTrigger><TabsTrigger value="trends">Trends</TabsTrigger></TabsList>
        <TabsContent value="overview"><div className="grid gap-4 md:grid-cols-2"><Card><CardHeader><CardTitle>Cost by Provider</CardTitle></CardHeader><CardContent><ResponsiveContainer width="100%" height={300}><PieChart><Pie data={data.by_provider} dataKey="cost" nameKey="name" cx="50%" cy="50%" outerRadius={100} label>{data.by_provider.map((_, index) => <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />)}</Pie><Tooltip /></PieChart></ResponsiveContainer></CardContent></Card>
        <Card><CardHeader><CardTitle>Top Databases by Cost</CardTitle></CardHeader><CardContent><div className="space-y-4">{data.top_databases.map(db => <div key={db.name} className="flex items-center justify-between"><div><p className="font-medium">{db.name}</p><p className="text-xs text-muted-foreground">{db.size_gb} GB · {db.provider}</p></div><div className="text-right"><p className="font-medium">${db.cost.toFixed(2)}</p></div></div>)}</div></CardContent></Card></div></TabsContent>

        <TabsContent value="recommendations" className="space-y-4">{data.recommendations.map((rec, idx) => <Card key={idx} className="border-l-4 border-l-green-500"><CardHeader><CardTitle>{rec.database}</CardTitle></CardHeader><CardContent><div className="space-y-2"><div className="flex justify-between"><span className="text-sm">Recommendation:</span><Badge>{rec.action}</Badge></div><div className="flex justify-between"><span className="text-sm">Current Cost:</span><span className="font-medium">${rec.current_cost}</span></div><div className="flex justify-between"><span className="text-sm">Estimated Cost:</span><span className="font-medium text-green-600">${rec.estimated_cost}</span></div><div className="flex justify-between"><span className="text-sm">Monthly Savings:</span><span className="font-medium text-green-600">${rec.savings}</span></div><div className="flex justify-between"><span className="text-sm">Break-even:</span><span className="text-sm text-muted-foreground">{rec.breakeven_days} days</span></div></div></CardContent></Card>)}</TabsContent>

        <TabsContent value="trends"><Card><CardHeader><CardTitle>Cost Trend (30 Days)</CardTitle></CardHeader><CardContent><ResponsiveContainer width="100%" height={400}><LineChart data={data.trend}><CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="day" /><YAxis /><Tooltip /><Legend /><Line type="monotone" dataKey="cost" stroke="#3b82f6" name="Daily Cost ($)" /></LineChart></ResponsiveContainer></CardContent></Card></TabsContent>
      </Tabs>
    </div>
  )
}
