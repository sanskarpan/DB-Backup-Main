"use client"

import React, { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Search, Filter, Calendar, Database, HardDrive, Tag,
  Clock, TrendingUp, Download, ChevronRight, ChevronLeft,
  X, Info, AlertCircle, CheckCircle, XCircle, Loader2
} from 'lucide-react'
import { format, formatDistance } from 'date-fns'

interface SearchResult {
  backup_id: string
  database_name: string
  database_type: string
  backup_type: string
  status: string
  size_bytes: number
  duration: number
  storage_path: string
  storage_provider: string
  created_at: string
  expires_at?: string
  tags?: Record<string, string>
  tables?: string[]
  schemas?: string[]
  score?: number
}

interface SearchFilters {
  text: string
  database_names: string[]
  database_types: string[]
  backup_types: string[]
  status: string[]
  storage_providers: string[]
  tags: Record<string, string>
  date_from?: string
  date_to?: string
  min_size?: number
  max_size?: number
  min_duration?: number
  max_duration?: number
  tables: string[]
  schemas: string[]
  include_expired: boolean
  sort_by: string
  sort_order: string
}

interface CatalogStats {
  total_backups: number
  total_size: number
  average_size: number
  average_duration: number
  by_status: Record<string, number>
  by_type: Record<string, number>
  by_database_type: Record<string, number>
  by_provider: Record<string, number>
}

export default function AdvancedBackupSearch() {
  const [filters, setFilters] = useState<SearchFilters>({
    text: '',
    database_names: [],
    database_types: [],
    backup_types: [],
    status: [],
    storage_providers: [],
    tags: {},
    tables: [],
    schemas: [],
    include_expired: false,
    sort_by: 'created_at',
    sort_order: 'desc'
  })

  const [results, setResults] = useState<SearchResult[]>([])
  const [stats, setStats] = useState<CatalogStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [total, setTotal] = useState(0)
  const [limit] = useState(20)
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [took, setTook] = useState(0)
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false)
  const [suggestions, setSuggestions] = useState<Record<string, string[]>>({})
  const [queryExamples, setQueryExamples] = useState<any[]>([])

  // Fetch catalog statistics
  useEffect(() => {
    fetchStats()
    fetchQueryExamples()
  }, [])

  const fetchStats = async () => {
    try {
      const response = await fetch('/api/v1/catalog/stats')
      if (response.ok) {
        const data = await response.json()
        setStats(data)
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err)
    }
  }

  const fetchQueryExamples = async () => {
    try {
      const response = await fetch('/api/v1/catalog/query-examples')
      if (response.ok) {
        const data = await response.json()
        setQueryExamples(data.examples || [])
      }
    } catch (err) {
      console.error('Failed to fetch examples:', err)
    }
  }

  // Fetch autocomplete suggestions
  const fetchSuggestions = async (field: string, prefix: string) => {
    if (prefix.length < 2) return
    try {
      const response = await fetch(`/api/v1/catalog/suggest?field=${field}&prefix=${prefix}&limit=10`)
      if (response.ok) {
        const data = await response.json()
        setSuggestions(prev => ({ ...prev, [field]: data.suggestions }))
      }
    } catch (err) {
      console.error('Failed to fetch suggestions:', err)
    }
  }

  // Search backups
  const searchBackups = useCallback(async (resetOffset = false) => {
    setLoading(true)
    setError(null)

    const currentOffset = resetOffset ? 0 : offset
    if (resetOffset) setOffset(0)

    try {
      const body = {
        ...filters,
        limit,
        offset: currentOffset
      }

      const response = await fetch('/api/v1/catalog/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })

      if (!response.ok) {
        throw new Error(`Search failed: ${response.statusText}`)
      }

      const data = await response.json()
      setResults(data.results || [])
      setTotal(data.total || 0)
      setHasMore(data.has_more || false)
      setTook(data.took_ms || 0)
    } catch (err: any) {
      setError(err.message || 'Search failed')
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [filters, limit, offset])

  // Simple natural language search
  const handleSimpleSearch = async (query: string) => {
    setLoading(true)
    setError(null)

    try {
      const response = await fetch(`/api/v1/catalog/search?q=${encodeURIComponent(query)}&limit=${limit}&offset=0`)

      if (!response.ok) {
        throw new Error(`Search failed: ${response.statusText}`)
      }

      const data = await response.json()
      setResults(data.results || [])
      setTotal(data.total || 0)
      setHasMore(data.has_more || false)
      setTook(data.took_ms || 0)
      setOffset(0)
    } catch (err: any) {
      setError(err.message || 'Search failed')
      setResults([])
    } finally {
      setLoading(false)
    }
  }

  // Format bytes to human readable
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // Format duration to human readable
  const formatDuration = (seconds: number): string => {
    if (seconds < 60) return `${seconds.toFixed(1)}s`
    if (seconds < 3600) return `${(seconds / 60).toFixed(1)}m`
    return `${(seconds / 3600).toFixed(1)}h`
  }

  // Status badge variant
  const getStatusVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (status.toLowerCase()) {
      case 'success': case 'completed': return 'default'
      case 'failed': case 'error': return 'destructive'
      case 'in_progress': case 'running': return 'secondary'
      default: return 'outline'
    }
  }

  // Status icon
  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'success': case 'completed': return <CheckCircle className="h-4 w-4 text-green-600" />
      case 'failed': case 'error': return <XCircle className="h-4 w-4 text-red-600" />
      case 'in_progress': case 'running': return <Loader2 className="h-4 w-4 animate-spin text-blue-600" />
      default: return <AlertCircle className="h-4 w-4 text-gray-600" />
    }
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Search className="h-8 w-8" />
          Backup Catalog Search
        </h1>
        <p className="text-muted-foreground mt-2">
          Advanced search with Elasticsearch-powered full-text indexing
        </p>
      </div>

      {/* Statistics Overview */}
      {stats && (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total_backups.toLocaleString()}</div>
              <p className="text-xs text-muted-foreground mt-1">
                {formatBytes(stats.total_size)} total
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Average Size</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatBytes(stats.average_size)}</div>
              <p className="text-xs text-muted-foreground mt-1">
                per backup
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">
                {((stats.by_status['success'] || 0) / stats.total_backups * 100).toFixed(1)}%
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {stats.by_status['success'] || 0} successful
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Avg Duration</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatDuration(stats.average_duration)}</div>
              <p className="text-xs text-muted-foreground mt-1">
                backup time
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Search Interface */}
      <Tabs defaultValue="simple" className="w-full">
        <TabsList className="grid w-full grid-cols-2 max-w-md">
          <TabsTrigger value="simple">Simple Search</TabsTrigger>
          <TabsTrigger value="advanced">Advanced Search</TabsTrigger>
        </TabsList>

        {/* Simple Search Tab */}
        <TabsContent value="simple" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Natural Language Query</CardTitle>
              <CardDescription>
                Search using natural language (e.g., "database:production type:mysql status:success")
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="database:mydb type:postgres status:success size:>1GB"
                    value={filters.text}
                    onChange={(e) => setFilters({ ...filters, text: e.target.value })}
                    onKeyPress={(e) => e.key === 'Enter' && handleSimpleSearch(filters.text)}
                    className="pl-10"
                  />
                </div>
                <Button onClick={() => handleSimpleSearch(filters.text)} disabled={loading}>
                  {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Search'}
                </Button>
              </div>

              {/* Query Examples */}
              {queryExamples.length > 0 && (
                <div className="space-y-2">
                  <Label className="text-sm font-medium">Example Queries</Label>
                  <div className="grid gap-2">
                    {queryExamples.slice(0, 3).map((example, idx) => (
                      <button
                        key={idx}
                        onClick={() => {
                          setFilters({ ...filters, text: example.query })
                          handleSimpleSearch(example.query)
                        }}
                        className="text-left p-2 rounded border hover:bg-accent text-sm"
                      >
                        <div className="font-medium">{example.description}</div>
                        <code className="text-xs text-muted-foreground">{example.query}</code>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Advanced Search Tab */}
        <TabsContent value="advanced" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Advanced Filters</CardTitle>
              <CardDescription>
                Use multiple filters for precise search results
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Text Search */}
              <div className="space-y-2">
                <Label>Free Text Search</Label>
                <Input
                  placeholder="Search in database names, paths, tables..."
                  value={filters.text}
                  onChange={(e) => setFilters({ ...filters, text: e.target.value })}
                />
              </div>

              {/* Database Filters */}
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label>Database Type</Label>
                  <Select
                    value={filters.database_types[0] || 'all'}
                    onValueChange={(value) => setFilters({
                      ...filters,
                      database_types: value === 'all' ? [] : [value]
                    })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="All Types" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Types</SelectItem>
                      {stats && Object.keys(stats.by_database_type).map(type => (
                        <SelectItem key={type} value={type}>{type} ({stats.by_database_type[type]})</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Backup Type</Label>
                  <Select
                    value={filters.backup_types[0] || 'all'}
                    onValueChange={(value) => setFilters({
                      ...filters,
                      backup_types: value === 'all' ? [] : [value]
                    })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="All Types" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Types</SelectItem>
                      {stats && Object.keys(stats.by_type).map(type => (
                        <SelectItem key={type} value={type}>{type} ({stats.by_type[type]})</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Status</Label>
                  <Select
                    value={filters.status[0] || 'all'}
                    onValueChange={(value) => setFilters({
                      ...filters,
                      status: value === 'all' ? [] : [value]
                    })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="All Status" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Status</SelectItem>
                      {stats && Object.keys(stats.by_status).map(status => (
                        <SelectItem key={status} value={status}>{status} ({stats.by_status[status]})</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Storage Provider */}
              <div className="space-y-2">
                <Label>Storage Provider</Label>
                <Select
                  value={filters.storage_providers[0] || 'all'}
                  onValueChange={(value) => setFilters({
                    ...filters,
                    storage_providers: value === 'all' ? [] : [value]
                  })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All Providers" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Providers</SelectItem>
                    {stats && Object.keys(stats.by_provider).map(provider => (
                      <SelectItem key={provider} value={provider}>{provider.toUpperCase()} ({stats.by_provider[provider]})</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Date Range */}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>Date From</Label>
                  <Input
                    type="datetime-local"
                    value={filters.date_from || ''}
                    onChange={(e) => setFilters({ ...filters, date_from: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Date To</Label>
                  <Input
                    type="datetime-local"
                    value={filters.date_to || ''}
                    onChange={(e) => setFilters({ ...filters, date_to: e.target.value })}
                  />
                </div>
              </div>

              {/* Size Range */}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>Min Size (bytes)</Label>
                  <Input
                    type="number"
                    placeholder="e.g., 1073741824 (1GB)"
                    value={filters.min_size || ''}
                    onChange={(e) => setFilters({ ...filters, min_size: e.target.value ? Number(e.target.value) : undefined })}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Max Size (bytes)</Label>
                  <Input
                    type="number"
                    placeholder="e.g., 10737418240 (10GB)"
                    value={filters.max_size || ''}
                    onChange={(e) => setFilters({ ...filters, max_size: e.target.value ? Number(e.target.value) : undefined })}
                  />
                </div>
              </div>

              {/* Sort Options */}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>Sort By</Label>
                  <Select
                    value={filters.sort_by}
                    onValueChange={(value) => setFilters({ ...filters, sort_by: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="created_at">Created Date</SelectItem>
                      <SelectItem value="size_bytes">Size</SelectItem>
                      <SelectItem value="duration">Duration</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Sort Order</Label>
                  <Select
                    value={filters.sort_order}
                    onValueChange={(value) => setFilters({ ...filters, sort_order: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="desc">Descending</SelectItem>
                      <SelectItem value="asc">Ascending</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Include Expired */}
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="include_expired"
                  checked={filters.include_expired}
                  onCheckedChange={(checked) => setFilters({ ...filters, include_expired: !!checked })}
                />
                <Label htmlFor="include_expired" className="cursor-pointer">
                  Include expired backups
                </Label>
              </div>

              {/* Search Button */}
              <div className="flex gap-2">
                <Button onClick={() => searchBackups(true)} disabled={loading} className="flex-1">
                  {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Search className="h-4 w-4 mr-2" />}
                  Search
                </Button>
                <Button
                  variant="outline"
                  onClick={() => {
                    setFilters({
                      text: '',
                      database_names: [],
                      database_types: [],
                      backup_types: [],
                      status: [],
                      storage_providers: [],
                      tags: {},
                      tables: [],
                      schemas: [],
                      include_expired: false,
                      sort_by: 'created_at',
                      sort_order: 'desc'
                    })
                    setResults([])
                  }}
                >
                  <X className="h-4 w-4 mr-2" />
                  Clear
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

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

      {/* Results Header */}
      {!loading && results.length > 0 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Found {total.toLocaleString()} results in {took}ms
            {offset > 0 && ` (showing ${offset + 1}-${Math.min(offset + limit, total)})`}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setOffset(Math.max(0, offset - limit))
                searchBackups(false)
              }}
              disabled={offset === 0 || loading}
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setOffset(offset + limit)
                searchBackups(false)
              }}
              disabled={!hasMore || loading}
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Results */}
      <div className="grid gap-4">
        {loading && (
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
        )}

        {!loading && results.length === 0 && (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8 text-muted-foreground">
                <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>No backups found matching your search criteria</p>
              </div>
            </CardContent>
          </Card>
        )}

        {!loading && results.map((backup) => (
          <Card key={backup.backup_id} className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Database className="h-5 w-5" />
                    {backup.database_name}
                    {backup.score && backup.score > 0 && (
                      <Badge variant="outline" className="ml-2">
                        Score: {backup.score.toFixed(2)}
                      </Badge>
                    )}
                  </CardTitle>
                  <CardDescription className="mt-1">
                    ID: {backup.backup_id}
                  </CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  {getStatusIcon(backup.status)}
                  <Badge variant={getStatusVariant(backup.status)}>
                    {backup.status}
                  </Badge>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-6 gap-4">
                <div>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <Database className="h-3 w-3" />
                    Type
                  </p>
                  <p className="text-sm font-medium capitalize">{backup.database_type}</p>
                  <p className="text-xs text-muted-foreground capitalize">{backup.backup_type}</p>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    Size
                  </p>
                  <p className="text-sm font-medium">{formatBytes(backup.size_bytes)}</p>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    Duration
                  </p>
                  <p className="text-sm font-medium">{formatDuration(backup.duration)}</p>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground">Provider</p>
                  <p className="text-sm font-medium uppercase">{backup.storage_provider}</p>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    Created
                  </p>
                  <p className="text-sm font-medium">
                    {format(new Date(backup.created_at), 'MMM d, yyyy')}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {formatDistance(new Date(backup.created_at), new Date(), { addSuffix: true })}
                  </p>
                </div>

                <div>
                  <p className="text-xs text-muted-foreground flex items-center gap-1">
                    <Download className="h-3 w-3" />
                    Path
                  </p>
                  <p className="text-sm font-medium truncate" title={backup.storage_path}>
                    {backup.storage_path.split('/').pop()}
                  </p>
                </div>
              </div>

              {/* Tags */}
              {backup.tags && Object.keys(backup.tags).length > 0 && (
                <div className="mt-4 pt-4 border-t">
                  <p className="text-xs text-muted-foreground flex items-center gap-1 mb-2">
                    <Tag className="h-3 w-3" />
                    Tags
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {Object.entries(backup.tags).map(([key, value]) => (
                      <Badge key={key} variant="secondary" className="text-xs">
                        {key}: {value}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Tables/Schemas */}
              {((backup.tables && backup.tables.length > 0) || (backup.schemas && backup.schemas.length > 0)) && (
                <div className="mt-4 pt-4 border-t">
                  {backup.tables && backup.tables.length > 0 && (
                    <div className="mb-2">
                      <p className="text-xs text-muted-foreground mb-1">
                        Tables ({backup.tables.length})
                      </p>
                      <div className="flex flex-wrap gap-1">
                        {backup.tables.slice(0, 10).map((table) => (
                          <Badge key={table} variant="outline" className="text-xs">
                            {table}
                          </Badge>
                        ))}
                        {backup.tables.length > 10 && (
                          <Badge variant="outline" className="text-xs">
                            +{backup.tables.length - 10} more
                          </Badge>
                        )}
                      </div>
                    </div>
                  )}
                  {backup.schemas && backup.schemas.length > 0 && (
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">
                        Schemas ({backup.schemas.length})
                      </p>
                      <div className="flex flex-wrap gap-1">
                        {backup.schemas.map((schema) => (
                          <Badge key={schema} variant="outline" className="text-xs">
                            {schema}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Expiration */}
              {backup.expires_at && (
                <div className="mt-4 pt-4 border-t">
                  <p className="text-xs text-muted-foreground">
                    Expires: {format(new Date(backup.expires_at), 'MMM d, yyyy')}
                    ({formatDistance(new Date(backup.expires_at), new Date(), { addSuffix: true })})
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Pagination Footer */}
      {!loading && results.length > 0 && (
        <div className="flex items-center justify-between pt-4">
          <div className="text-sm text-muted-foreground">
            Page {Math.floor(offset / limit) + 1} of {Math.ceil(total / limit)}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setOffset(Math.max(0, offset - limit))
                searchBackups(false)
              }}
              disabled={offset === 0 || loading}
            >
              <ChevronLeft className="h-4 w-4 mr-2" />
              Previous
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                setOffset(offset + limit)
                searchBackups(false)
              }}
              disabled={!hasMore || loading}
            >
              Next
              <ChevronRight className="h-4 w-4 ml-2" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
