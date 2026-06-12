"use client"

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Play, CheckCircle2, XCircle, Clock, FileText, AlertTriangle } from 'lucide-react'

export default function DRTestingDashboard() {
  const [data] = useState({
    stats: { total_tests: 45, passing: 42, failing: 3, last_run: new Date().toISOString(), rto_compliance: 95.5, rpo_compliance: 98.2 },
    recent_tests: [
      { id: 'test-001', database: 'production-db', status: 'PASSING', rto_actual: 180, rto_target: 240, rpo_actual: 10, rpo_target: 15, run_at: new Date(Date.now() - 3600000).toISOString(), recommendations: ['Excellent performance', 'Within all SLA targets'] },
      { id: 'test-002', database: 'analytics-db', status: 'FAILING', rto_actual: 350, rto_target: 240, rpo_actual: 12, rpo_target: 15, run_at: new Date(Date.now() - 7200000).toISOString(), recommendations: ['Optimize restore process', 'Review database size'] }
    ]
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2"><FileText className="h-8 w-8" />DR Testing Dashboard</h1>
          <p className="text-muted-foreground">Automated disaster recovery validation</p>
        </div>
        <Button><Play className="h-4 w-4 mr-2" />Run Test</Button>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Total Tests</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">{data.stats.total_tests}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">Passing</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold text-green-600">{data.stats.passing}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">RTO Compliance</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">{data.stats.rto_compliance}%</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">RPO Compliance</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">{data.stats.rpo_compliance}%</div></CardContent></Card>
      </div>

      <Tabs defaultValue="results"><TabsList><TabsTrigger value="results">Test Results</TabsTrigger><TabsTrigger value="schedule">Schedule</TabsTrigger></TabsList>
        <TabsContent value="results" className="space-y-4">{data.recent_tests.map(test => (<Card key={test.id} className={test.status === 'PASSING' ? 'border-l-4 border-l-green-500' : 'border-l-4 border-l-red-500'}><CardHeader><div className="flex items-center justify-between"><CardTitle>{test.database}</CardTitle><Badge variant={test.status === 'PASSING' ? 'default' : 'destructive'}>{test.status}</Badge></div><CardDescription>{new Date(test.run_at).toLocaleString()}</CardDescription></CardHeader><CardContent className="space-y-4"><div className="grid grid-cols-2 gap-4"><div><p className="text-xs text-muted-foreground">RTO (Target: {test.rto_target}s)</p><p className={`text-sm font-medium ${test.rto_actual <= test.rto_target ? 'text-green-600' : 'text-red-600'}`}>{test.rto_actual}s {test.rto_actual <= test.rto_target ? <CheckCircle2 className="inline h-4 w-4" /> : <XCircle className="inline h-4 w-4" />}</p></div><div><p className="text-xs text-muted-foreground">RPO (Target: {test.rpo_target}m)</p><p className={`text-sm font-medium ${test.rpo_actual <= test.rpo_target ? 'text-green-600' : 'text-red-600'}`}>{test.rpo_actual}m {test.rpo_actual <= test.rpo_target ? <CheckCircle2 className="inline h-4 w-4" /> : <XCircle className="inline h-4 w-4" />}</p></div></div><div><p className="text-xs text-muted-foreground mb-2">Recommendations</p><ul className="space-y-1">{test.recommendations.map((rec, i) => <li key={i} className="text-sm flex items-start gap-2">{test.status === 'PASSING' ? <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5" /> : <AlertTriangle className="h-4 w-4 text-orange-500 mt-0.5" />}{rec}</li>)}</ul></div></CardContent></Card>))}</TabsContent>
        <TabsContent value="schedule"><Card><CardHeader><CardTitle>Test Schedule</CardTitle><CardDescription>Automated DR testing schedule</CardDescription></CardHeader><CardContent><div className="space-y-4"><div className="flex items-center justify-between"><span className="text-sm">Daily automated tests</span><Badge>Active</Badge></div><div className="flex items-center justify-between"><span className="text-sm">Monthly full validation</span><Badge>Active</Badge></div><div className="flex items-center justify-between"><span className="text-sm">Quarterly compliance audit</span><Badge>Active</Badge></div></div></CardContent></Card></TabsContent>
      </Tabs>
    </div>
  )
}
