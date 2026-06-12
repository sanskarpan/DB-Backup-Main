"use client"

import React, { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Search, Filter, Calendar, Database, HardDrive, Tag } from 'lucide-react'

export default function BackupSearch() {
  const [searchQuery, setSearchQuery] = useState('')
  const [results, setResults] = useState([
    { id: 'b-001', database: 'production-db', type: 'full', status: 'success', size_gb: 125.5, provider: 's3', created_at: new Date(Date.now() - 86400000).toISOString(), tags: ['production', 'daily'] },
    { id: 'b-002', database: 'analytics-db', type: 'incremental', status: 'success', size_gb: 45.2, provider: 'azure', created_at: new Date(Date.now() - 172800000).toISOString(), tags: ['analytics', 'hourly'] }
  ])

  return (
    <div className="space-y-6">
      <div><h1 className="text-3xl font-bold flex items-center gap-2"><Search className="h-8 w-8" />Backup Catalog</h1><p className="text-muted-foreground">Advanced search and timeline view</p></div>

      <Card><CardContent className="pt-6"><div className="flex gap-4"><div className="relative flex-1"><Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" /><Input placeholder="Search backups (database:prod status:success type:full)" value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} className="pl-10" /></div><Select><SelectTrigger className="w-40"><SelectValue placeholder="Database" /></SelectTrigger><SelectContent><SelectItem value="all">All Databases</SelectItem><SelectItem value="prod">Production</SelectItem></SelectContent></Select><Select><SelectTrigger className="w-40"><SelectValue placeholder="Status" /></SelectTrigger><SelectContent><SelectItem value="all">All Status</SelectItem><SelectItem value="success">Success</SelectItem><SelectItem value="failed">Failed</SelectItem></SelectContent></Select><Button><Filter className="h-4 w-4 mr-2" />Filter</Button></div></CardContent></Card>

      <div className="grid gap-4">{results.map(backup => <Card key={backup.id}><CardHeader><div className="flex items-center justify-between"><CardTitle className="text-lg">{backup.database}</CardTitle><Badge variant={backup.status === 'success' ? 'default' : 'destructive'}>{backup.status}</Badge></div></CardHeader><CardContent><div className="grid grid-cols-2 md:grid-cols-5 gap-4"><div><p className="text-xs text-muted-foreground flex items-center gap-1"><Database className="h-3 w-3" />Type</p><p className="text-sm font-medium capitalize">{backup.type}</p></div><div><p className="text-xs text-muted-foreground flex items-center gap-1"><HardDrive className="h-3 w-3" />Size</p><p className="text-sm font-medium">{backup.size_gb.toFixed(1)} GB</p></div><div><p className="text-xs text-muted-foreground">Provider</p><p className="text-sm font-medium uppercase">{backup.provider}</p></div><div><p className="text-xs text-muted-foreground flex items-center gap-1"><Calendar className="h-3 w-3" />Created</p><p className="text-sm font-medium">{new Date(backup.created_at).toLocaleDateString()}</p></div><div><p className="text-xs text-muted-foreground flex items-center gap-1"><Tag className="h-3 w-3" />Tags</p><div className="flex gap-1">{backup.tags.map(tag => <Badge key={tag} variant="outline" className="text-xs">{tag}</Badge>)}</div></div></div></CardContent></Card>)}</div>
    </div>
  )
}
