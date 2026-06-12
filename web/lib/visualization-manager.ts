/**
 * Advanced Data Visualizations - Core Managers
 * Comprehensive system for data visualization, charts, and analytics
 */

// ==================== Types & Interfaces ====================

export type ChartType =
  | 'line'
  | 'area'
  | 'bar'
  | 'stackedBar'
  | 'pie'
  | 'donut'
  | 'combo'
  | 'scatter'
  | 'heatmap'
  | 'timeline'
  | 'gantt'
  | 'comparison'
  | 'gauge'
  | 'funnel'
  | 'radar'

export type TimeGranularity = 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year'

export type AggregationType = 'sum' | 'avg' | 'min' | 'max' | 'count' | 'median' | 'percentile'

export interface ChartConfig {
  id: string
  type: ChartType
  title: string
  subtitle?: string
  data: DataPoint[]
  options: ChartOptions
  theme?: ChartTheme
  responsive?: boolean
  animated?: boolean
  interactive?: boolean
  exportable?: boolean
  createdAt: Date
  updatedAt?: Date
}

export interface ChartOptions {
  width?: number
  height?: number
  margin?: Margin
  legend?: LegendConfig
  tooltip?: TooltipConfig
  axis?: AxisConfig
  colors?: string[]
  grid?: GridConfig
  annotations?: Annotation[]
  zoom?: boolean
  brush?: boolean
  crosshair?: boolean
  stacked?: boolean
  normalized?: boolean
}

export interface Margin {
  top: number
  right: number
  bottom: number
  left: number
}

export interface LegendConfig {
  enabled: boolean
  position: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  orientation?: 'horizontal' | 'vertical'
}

export interface TooltipConfig {
  enabled: boolean
  shared?: boolean
  formatter?: string
  valuePrefix?: string
  valueSuffix?: string
}

export interface AxisConfig {
  x?: AxisSettings
  y?: AxisSettings
  y2?: AxisSettings
}

export interface AxisSettings {
  label?: string
  type?: 'linear' | 'logarithmic' | 'category' | 'time'
  min?: number
  max?: number
  tickFormat?: string
  grid?: boolean
}

export interface GridConfig {
  x?: boolean
  y?: boolean
  color?: string
  dashArray?: string
}

export interface Annotation {
  type: 'line' | 'rect' | 'text' | 'marker'
  position: number | { x: number; y: number }
  label?: string
  color?: string
  dashArray?: string
}

export interface ChartTheme {
  name: string
  backgroundColor: string
  textColor: string
  gridColor: string
  colors: string[]
  fontFamily: string
  fontSize: number
}

export interface DataPoint {
  timestamp?: Date
  date?: string
  label: string
  value: number
  category?: string
  metadata?: Record<string, any>
}

export interface TimeSeriesData {
  id: string
  name: string
  description?: string
  dataPoints: TimeSeriesPoint[]
  granularity: TimeGranularity
  aggregationType: AggregationType
  unit: string
  createdAt: Date
}

export interface TimeSeriesPoint {
  timestamp: Date
  value: number
  metadata?: Record<string, any>
}

export interface BackupSizeTrend {
  backupId: string
  backupName: string
  timestamp: Date
  sizeBytes: number
  compressedSize: number
  compressionRatio: number
  database: string
}

export interface DatabaseGrowth {
  databaseId: string
  databaseName: string
  timestamp: Date
  totalSize: number
  dataSize: number
  indexSize: number
  growthRate: number
  recordCount: number
}

// Alias for test compatibility
export type DatabaseGrowthPoint = Omit<DatabaseGrowth, 'growthRate'> & {
  sizeBytes: number
  tableCount: number
  rowCount: number
}

export interface CostAnalysis {
  date: Date
  storageCost: number
  computeCost: number
  networkCost: number
  totalCost: number
  currency: string
  provider: string
}

// Alias for test compatibility
export type CostData = {
  date: Date
  database: string
  storageCost: number
  computeCost: number
  networkCost: number
  totalCost: number
  currency: string
}

export interface StorageBreakdown {
  category: string
  sizeBytes: number
  percentage: number
  count: number
  color?: string
}

export interface SuccessFailureRate {
  timestamp: Date
  totalBackups: number
  successfulBackups: number
  failedBackups: number
  successRate: number
  failureRate: number
  averageDuration: number
}

export interface PerformanceMetrics {
  timestamp: Date
  backupSpeed: number // MB/s
  restoreSpeed: number
  cpuUsage: number // percentage
  memoryUsage: number
  diskIo: number
  networkThroughput: number
}

export interface HeatmapData {
  x: string | number
  y: string | number
  value: number
  label?: string
  color?: string
}

export interface NetworkUsage {
  timestamp: Date
  uploadBytes: number
  downloadBytes: number
  uploadSpeed: number // MB/s
  downloadSpeed: number
  latency: number // ms
}

export interface TimelineEvent {
  id: string
  name: string
  startTime: Date
  endTime: Date
  duration: number
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress?: number
  dependencies?: string[]
  resource?: string
  metadata?: Record<string, any>
}

export interface ComparisonData {
  label: string
  values: Record<string, number>
  metadata?: Record<string, any>
}

export interface DashboardConfig {
  id: string
  name: string
  description?: string
  layout: DashboardLayout
  charts: ChartConfig[]
  refreshInterval?: number
  autoRefresh?: boolean
  filters?: DashboardFilter[]
  createdAt: Date
  updatedAt?: Date
}

export interface DashboardLayout {
  type: 'grid' | 'flex' | 'masonry'
  columns: number
  rowHeight?: number
  gap: number
  positions: ChartPosition[]
}

export interface ChartPosition {
  chartId: string
  x: number
  y: number
  width: number
  height: number
}

export interface DashboardFilter {
  id: string
  label: string
  type: 'date' | 'select' | 'multi-select' | 'range'
  options?: string[]
  defaultValue?: any
}

export interface ExportOptions {
  format: 'png' | 'svg' | 'pdf' | 'csv' | 'json'
  width?: number
  height?: number
  quality?: number
  backgroundColor?: string
  scale?: number
}

export interface StatisticalAnalysis {
  mean: number
  median: number
  mode: number
  stdDev: number
  variance: number
  min: number
  max: number
  q1: number
  q3: number
  iqr: number
  outliers: number[]
}

export interface TrendAnalysis {
  direction: 'up' | 'down' | 'stable'
  slope: number
  correlation: number
  forecast: number[]
  confidence: number
}

// ==================== Visualization Data Manager ====================

export class VisualizationDataManager {
  private datasets: Map<string, TimeSeriesData> = new Map()
  private listeners: Set<(data: TimeSeriesData) => void> = new Set()

  constructor() {
    this.loadDatasets()
  }

  createDataset(dataset: Omit<TimeSeriesData, 'createdAt'>): TimeSeriesData {
    const fullDataset: TimeSeriesData = {
      ...dataset,
      createdAt: new Date()
    }

    this.datasets.set(fullDataset.id, fullDataset)
    this.saveDatasets()
    this.notifyListeners(fullDataset)
    return fullDataset
  }

  getDataset(datasetId: string): TimeSeriesData | undefined {
    return this.datasets.get(datasetId)
  }

  getAllDatasets(): TimeSeriesData[] {
    return Array.from(this.datasets.values())
  }

  updateDataset(datasetId: string, updates: Partial<TimeSeriesData>): boolean {
    const dataset = this.datasets.get(datasetId)
    if (!dataset) return false

    Object.assign(dataset, updates)
    this.saveDatasets()
    this.notifyListeners(dataset)
    return true
  }

  appendDataPoint(datasetId: string, dataPoint: TimeSeriesPoint): boolean {
    const dataset = this.datasets.get(datasetId)
    if (!dataset) return false

    dataset.dataPoints.push(dataPoint)
    this.saveDatasets()
    this.notifyListeners(dataset)
    return true
  }

  deleteDataset(datasetId: string): boolean {
    const deleted = this.datasets.delete(datasetId)
    if (deleted) this.saveDatasets()
    return deleted
  }

  aggregateData(datasetId: string, newGranularity: TimeGranularity, aggregationType?: AggregationType): TimeSeriesPoint[] {
    const dataset = this.datasets.get(datasetId)
    if (!dataset) return []

    const aggType = aggregationType || dataset.aggregationType

    // Group by new granularity and aggregate
    const grouped = new Map<string, number[]>()

    dataset.dataPoints.forEach(point => {
      const key = this.getTimeKey(point.timestamp, newGranularity)
      if (!grouped.has(key)) {
        grouped.set(key, [])
      }
      grouped.get(key)!.push(point.value)
    })

    const aggregated: TimeSeriesPoint[] = []
    grouped.forEach((values, key) => {
      aggregated.push({
        timestamp: new Date(key),
        value: this.aggregate(values, aggType)
      })
    })

    return aggregated.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  private getTimeKey(date: Date, granularity: TimeGranularity): string {
    const year = date.getFullYear()
    const month = date.getMonth()
    const day = date.getDate()
    const hour = date.getHours()

    switch (granularity) {
      case 'hour':
        return `${year}-${month}-${day}-${hour}`
      case 'day':
        return `${year}-${month}-${day}`
      case 'week':
        const week = Math.floor(day / 7)
        return `${year}-${month}-${week}`
      case 'month':
        return `${year}-${month}`
      case 'quarter':
        const quarter = Math.floor(month / 3)
        return `${year}-${quarter}`
      case 'year':
        return `${year}`
      default:
        return date.toISOString()
    }
  }

  private aggregate(values: number[], type: AggregationType): number {
    if (values.length === 0) return 0

    switch (type) {
      case 'sum':
        return values.reduce((a, b) => a + b, 0)
      case 'avg':
        return values.reduce((a, b) => a + b, 0) / values.length
      case 'min':
        return Math.min(...values)
      case 'max':
        return Math.max(...values)
      case 'count':
        return values.length
      case 'median':
        const sorted = [...values].sort((a, b) => a - b)
        const mid = Math.floor(sorted.length / 2)
        return sorted.length % 2 === 0
          ? (sorted[mid - 1] + sorted[mid]) / 2
          : sorted[mid]
      case 'percentile':
        const s = [...values].sort((a, b) => a - b)
        const index = Math.ceil(s.length * 0.95) - 1
        return s[index]
      default:
        return 0
    }
  }

  subscribe(listener: (data: TimeSeriesData) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(data: TimeSeriesData): void {
    this.listeners.forEach(listener => listener(data))
  }

  private loadDatasets(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-datasets')
    if (stored) {
      try {
        const datasets = JSON.parse(stored)
        datasets.forEach((ds: TimeSeriesData) => {
          ds.createdAt = new Date(ds.createdAt)
          ds.dataPoints.forEach(p => p.timestamp = new Date(p.timestamp))
          this.datasets.set(ds.id, ds)
        })
      } catch (e) {
        console.error('Failed to load datasets:', e)
      }
    }
  }

  private saveDatasets(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-datasets', JSON.stringify(Array.from(this.datasets.values())))
  }
}

// ==================== Chart Config Manager ====================

export class ChartConfigManager {
  private configs: Map<string, ChartConfig> = new Map()
  private themes: Map<string, ChartTheme> = new Map()
  private listeners: Set<(config: ChartConfig) => void> = new Set()

  constructor() {
    this.initializeDefaultThemes()
    this.loadConfigs()
  }

  private initializeDefaultThemes(): void {
    const defaultTheme: ChartTheme = {
      name: 'default',
      backgroundColor: '#ffffff',
      textColor: '#333333',
      gridColor: '#e0e0e0',
      colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4'],
      fontFamily: 'Inter, system-ui, sans-serif',
      fontSize: 12
    }

    const darkTheme: ChartTheme = {
      name: 'dark',
      backgroundColor: '#1f2937',
      textColor: '#f9fafb',
      gridColor: '#374151',
      colors: ['#60a5fa', '#34d399', '#fbbf24', '#f87171', '#a78bfa', '#22d3ee'],
      fontFamily: 'Inter, system-ui, sans-serif',
      fontSize: 12
    }

    this.themes.set('default', defaultTheme)
    this.themes.set('dark', darkTheme)
  }

  createChart(chart: Omit<ChartConfig, 'createdAt'>): ChartConfig {
    const defaultOptions: ChartOptions = {
      width: 800,
      height: 400,
      margin: { top: 20, right: 30, bottom: 50, left: 60 },
      legend: { enabled: true, position: 'top', align: 'center' },
      tooltip: { enabled: true, shared: true },
      axis: {
        x: { label: 'X Axis', type: 'category', grid: true },
        y: { label: 'Y Axis', type: 'linear', grid: true }
      },
      colors: ['#3b82f6', '#10b981', '#f59e0b'],
      grid: { x: true, y: true, color: '#e0e0e0' },
      annotations: [],
      zoom: false,
      brush: false
    }

    const config: ChartConfig = {
      ...chart,
      options: {
        ...defaultOptions,
        ...chart.options
      },
      theme: chart.theme || this.themes.get('default'),
      responsive: chart.responsive !== undefined ? chart.responsive : true,
      animated: chart.animated !== undefined ? chart.animated : true,
      interactive: chart.interactive !== undefined ? chart.interactive : true,
      exportable: chart.exportable !== undefined ? chart.exportable : true,
      createdAt: new Date()
    }

    this.configs.set(config.id, config)
    this.saveConfigs()
    this.notifyListeners(config)
    return config
  }

  getChart(chartId: string): ChartConfig | undefined {
    return this.configs.get(chartId)
  }

  getAllCharts(): ChartConfig[] {
    return Array.from(this.configs.values())
  }

  updateChart(chartId: string, updates: Partial<ChartConfig>): boolean {
    const config = this.configs.get(chartId)
    if (!config) return false

    Object.assign(config, updates, { updatedAt: new Date() })
    this.saveConfigs()
    this.notifyListeners(config)
    return true
  }

  deleteChart(chartId: string): boolean {
    const deleted = this.configs.delete(chartId)
    if (deleted) this.saveConfigs()
    return deleted
  }

  setTheme(chartId: string, themeName: string): boolean {
    const config = this.configs.get(chartId)
    const theme = this.themes.get(themeName)
    if (!config || !theme) return false

    config.theme = theme
    config.updatedAt = new Date()
    this.saveConfigs()
    this.notifyListeners(config)
    return true
  }

  subscribe(listener: (config: ChartConfig) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(config: ChartConfig): void {
    this.listeners.forEach(listener => listener(config))
  }

  private loadConfigs(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-chart-configs')
    if (stored) {
      try {
        const configs = JSON.parse(stored)
        configs.forEach((cfg: ChartConfig) => {
          cfg.createdAt = new Date(cfg.createdAt)
          if (cfg.updatedAt) cfg.updatedAt = new Date(cfg.updatedAt)
          cfg.data.forEach(d => {
            if (d.timestamp) d.timestamp = new Date(d.timestamp)
          })
          this.configs.set(cfg.id, cfg)
        })
      } catch (e) {
        console.error('Failed to load chart configs:', e)
      }
    }
  }

  private saveConfigs(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-chart-configs', JSON.stringify(Array.from(this.configs.values())))
  }
}

// ==================== Backup Size Trend Manager ====================

export class BackupSizeTrendManager {
  private trends: BackupSizeTrend[] = []

  constructor() {
    this.loadTrends()
  }

  addTrend(trend: Omit<BackupSizeTrend, 'compressionRatio'>): BackupSizeTrend {
    const fullTrend: BackupSizeTrend = {
      ...trend,
      compressionRatio: trend.sizeBytes > 0 ? trend.compressedSize / trend.sizeBytes : 0
    }

    this.trends.push(fullTrend)
    this.saveTrends()
    return fullTrend
  }

  getTrends(databaseId?: string, startDate?: Date, endDate?: Date): BackupSizeTrend[] {
    let filtered = [...this.trends]

    if (databaseId) {
      filtered = filtered.filter(t => t.database === databaseId)
    }

    if (startDate) {
      filtered = filtered.filter(t => t.timestamp >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(t => t.timestamp <= endDate)
    }

    return filtered.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  getChartData(databaseId?: string, startDate?: Date, endDate?: Date): DataPoint[] {
    const trends = this.getTrends(databaseId, startDate, endDate)

    return trends.map(t => ({
      timestamp: t.timestamp,
      date: t.timestamp.toISOString().split('T')[0],
      label: t.backupName,
      value: t.sizeBytes / (1024 * 1024 * 1024), // Convert to GB
      category: t.database,
      metadata: {
        backupId: t.backupId,
        compressedSize: t.compressedSize,
        compressionRatio: t.compressionRatio
      }
    }))
  }

  private loadTrends(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-backup-size-trends')
    if (stored) {
      try {
        this.trends = JSON.parse(stored)
        this.trends.forEach(t => t.timestamp = new Date(t.timestamp))
      } catch (e) {
        console.error('Failed to load backup size trends:', e)
      }
    }
  }

  private saveTrends(): void {
    if (typeof window === 'undefined') return
    // Keep last 10000 trends
    const toSave = this.trends.slice(-10000)
    localStorage.setItem('visualization-backup-size-trends', JSON.stringify(toSave))
  }
}

// ==================== Database Growth Manager ====================

export class DatabaseGrowthManager {
  private growth: DatabaseGrowth[] = []

  constructor() {
    this.loadGrowth()
  }

  addGrowthPoint(point: DatabaseGrowthPoint): DatabaseGrowthPoint & { growthRate: number } {
    // Calculate growth rate based on previous point
    const previousPoints = this.growth.filter((g: any) =>
      g.databaseId === point.databaseId &&
      g.timestamp < point.timestamp
    ).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())

    let growthRate = 0
    if (previousPoints.length > 0) {
      const prev: any = previousPoints[0]
      const timeDiff = point.timestamp.getTime() - prev.timestamp.getTime()
      const sizeDiff = point.sizeBytes - prev.sizeBytes
      growthRate = prev.sizeBytes > 0 ? sizeDiff / prev.sizeBytes : 0
    }

    const fullPoint: DatabaseGrowthPoint & { growthRate: number } = {
      ...point,
      growthRate
    }

    this.growth.push(fullPoint as any)
    this.saveGrowth()
    return fullPoint
  }

  getGrowth(databaseId?: string, startDate?: Date, endDate?: Date): DatabaseGrowth[] {
    let filtered = [...this.growth]

    if (databaseId) {
      filtered = filtered.filter(g => g.databaseId === databaseId)
    }

    if (startDate) {
      filtered = filtered.filter(g => g.timestamp >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(g => g.timestamp <= endDate)
    }

    return filtered.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  getChartData(databaseId?: string, startDate?: Date, endDate?: Date): DataPoint[] {
    const growth = this.getGrowth(databaseId, startDate, endDate)

    return growth.map((g: any) => ({
      timestamp: g.timestamp,
      date: g.timestamp.toISOString().split('T')[0],
      label: g.databaseName,
      value: (g.totalSize || g.sizeBytes) / (1024 * 1024 * 1024), // Convert to GB
      category: g.databaseName,
      metadata: {
        dataSize: g.dataSize,
        indexSize: g.indexSize,
        growthRate: g.growthRate,
        recordCount: g.recordCount || g.rowCount,
        tableCount: g.tableCount
      }
    }))
  }

  private loadGrowth(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-database-growth')
    if (stored) {
      try {
        this.growth = JSON.parse(stored)
        this.growth.forEach(g => g.timestamp = new Date(g.timestamp))
      } catch (e) {
        console.error('Failed to load database growth:', e)
      }
    }
  }

  private saveGrowth(): void {
    if (typeof window === 'undefined') return
    const toSave = this.growth.slice(-10000)
    localStorage.setItem('visualization-database-growth', JSON.stringify(toSave))
  }
}

// ==================== Cost Analysis Manager ====================

export class CostAnalysisManager {
  private costs: CostAnalysis[] = []

  constructor() {
    this.loadCosts()
  }

  addCost(cost: CostData): CostData {
    this.costs.push(cost as any)
    this.saveCosts()
    return cost
  }

  getCosts(database?: string, startDate?: Date, endDate?: Date): CostData[] {
    let filtered: any[] = [...this.costs]

    if (database) {
      filtered = filtered.filter((c: any) => c.database === database)
    }

    if (startDate) {
      filtered = filtered.filter((c: any) => c.date >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter((c: any) => c.date <= endDate)
    }

    return filtered.sort((a: any, b: any) => a.date.getTime() - b.date.getTime())
  }

  getStackedBarData(database?: string, startDate?: Date, endDate?: Date): any[] {
    const costs = this.getCosts(database, startDate, endDate)

    return costs.map((c: any) => ({
      date: c.date.toISOString().split('T')[0],
      storage: c.storageCost,
      compute: c.computeCost,
      network: c.networkCost,
      total: c.totalCost
    }))
  }

  getTotalCost(database?: string, startDate?: Date, endDate?: Date): number {
    const costs = this.getCosts(database, startDate, endDate)
    return costs.reduce((sum: number, c: any) => sum + c.totalCost, 0)
  }

  private loadCosts(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-cost-analysis')
    if (stored) {
      try {
        this.costs = JSON.parse(stored)
        this.costs.forEach(c => c.date = new Date(c.date))
      } catch (e) {
        console.error('Failed to load cost analysis:', e)
      }
    }
  }

  private saveCosts(): void {
    if (typeof window === 'undefined') return
    const toSave = this.costs.slice(-10000)
    localStorage.setItem('visualization-cost-analysis', JSON.stringify(toSave))
  }
}

// ==================== Storage Breakdown Manager ====================

export class StorageBreakdownManager {
  private breakdowns: Map<string, StorageBreakdown[]> = new Map()

  constructor() {
    this.loadBreakdowns()
  }

  setBreakdown(breakdown: { timestamp: Date, categories: Array<{name: string, sizeBytes: number, percentage: number}>, totalSize: number }): { timestamp: Date, categories: Array<{name: string, sizeBytes: number, percentage: number}>, totalSize: number } {
    // Calculate percentages
    const total = breakdown.categories.reduce((sum, b) => sum + b.sizeBytes, 0)
    breakdown.categories.forEach(b => {
      b.percentage = total > 0 ? (b.sizeBytes / total) * 100 : 0
    })

    const id = breakdown.timestamp.toISOString()
    this.breakdowns.set(id, breakdown.categories.map(c => ({
      category: c.name,
      sizeBytes: c.sizeBytes,
      percentage: c.percentage,
      count: 0
    })))
    this.saveBreakdowns()
    return breakdown
  }

  getBreakdown(id: string): StorageBreakdown[] {
    return this.breakdowns.get(id) || []
  }

  getPieChartData(id: string): DataPoint[] {
    const breakdown = this.getBreakdown(id)

    return breakdown.map(b => ({
      label: b.category,
      value: b.sizeBytes / (1024 * 1024 * 1024), // GB
      category: b.category,
      metadata: {
        percentage: b.percentage,
        count: b.count,
        color: b.color
      }
    }))
  }

  private loadBreakdowns(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-storage-breakdown')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        this.breakdowns = new Map(data)
      } catch (e) {
        console.error('Failed to load storage breakdowns:', e)
      }
    }
  }

  private saveBreakdowns(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-storage-breakdown', JSON.stringify(Array.from(this.breakdowns.entries())))
  }
}

// ==================== Success/Failure Rate Manager ====================

export class SuccessFailureRateManager {
  private rates: SuccessFailureRate[] = []

  constructor() {
    this.loadRates()
  }

  addRate(rate: Omit<SuccessFailureRate, 'successRate' | 'failureRate'>): SuccessFailureRate {
    const fullRate: SuccessFailureRate = {
      ...rate,
      successRate: rate.totalBackups > 0 ? (rate.successfulBackups / rate.totalBackups) * 100 : 0,
      failureRate: rate.totalBackups > 0 ? (rate.failedBackups / rate.totalBackups) * 100 : 0
    }

    this.rates.push(fullRate)
    this.saveRates()
    return fullRate
  }

  getRates(startDate?: Date, endDate?: Date): SuccessFailureRate[] {
    let filtered = [...this.rates]

    if (startDate) {
      filtered = filtered.filter(r => r.timestamp >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(r => r.timestamp <= endDate)
    }

    return filtered.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  getComboChartData(database?: string, startDate?: Date, endDate?: Date): any[] {
    const rates = this.getRates(startDate, endDate)

    return rates.map((r: any) => ({
      date: r.timestamp ? r.timestamp.toISOString().split('T')[0] : r.date?.toISOString().split('T')[0] || '',
      successCount: r.successfulBackups,
      failureCount: r.failedBackups,
      successRate: r.successRate,
      failureRate: r.failureRate,
      totalBackups: r.totalBackups
    }))
  }

  private loadRates(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-success-failure-rates')
    if (stored) {
      try {
        this.rates = JSON.parse(stored)
        this.rates.forEach(r => r.timestamp = new Date(r.timestamp))
      } catch (e) {
        console.error('Failed to load success/failure rates:', e)
      }
    }
  }

  private saveRates(): void {
    if (typeof window === 'undefined') return
    const toSave = this.rates.slice(-10000)
    localStorage.setItem('visualization-success-failure-rates', JSON.stringify(toSave))
  }
}

// ==================== Performance Metrics Manager ====================

export class PerformanceMetricsManager {
  private metrics: PerformanceMetrics[] = []

  constructor() {
    this.loadMetrics()
  }

  addMetric(metric: PerformanceMetrics): PerformanceMetrics {
    this.metrics.push(metric)
    this.saveMetrics()
    return metric
  }

  getMetrics(startDate?: Date, endDate?: Date): PerformanceMetrics[] {
    let filtered = [...this.metrics]

    if (startDate) {
      filtered = filtered.filter(m => m.timestamp >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(m => m.timestamp <= endDate)
    }

    return filtered.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  getMultiSeriesData(startDate?: Date, endDate?: Date): any[] {
    const metrics = this.getMetrics(startDate, endDate)

    return metrics.map((m: any) => ({
      date: m.timestamp.toISOString().split('T')[0],
      backupSpeed: m.backupSpeed,
      restoreSpeed: m.restoreSpeed,
      cpuUsage: m.cpuUsage,
      memoryUsage: m.memoryUsage,
      diskIO: m.diskIo,
      networkThroughput: m.networkThroughput
    }))
  }

  private loadMetrics(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-performance-metrics')
    if (stored) {
      try {
        this.metrics = JSON.parse(stored)
        this.metrics.forEach(m => m.timestamp = new Date(m.timestamp))
      } catch (e) {
        console.error('Failed to load performance metrics:', e)
      }
    }
  }

  private saveMetrics(): void {
    if (typeof window === 'undefined') return
    const toSave = this.metrics.slice(-10000)
    localStorage.setItem('visualization-performance-metrics', JSON.stringify(toSave))
  }
}

// ==================== Heatmap Manager ====================

export class HeatmapManager {
  private heatmaps: Map<string, HeatmapData[]> = new Map()

  constructor() {
    this.loadHeatmaps()
  }

  setHeatmap(heatmap: { id: string, title: string, data: number[][], rowLabels: string[], columnLabels: string[] }): { id: string, title: string, data: number[][], rowLabels: string[], columnLabels: string[] } {
    const heatmapData: HeatmapData[] = []
    heatmap.data.forEach((row, rowIdx) => {
      row.forEach((value, colIdx) => {
        heatmapData.push({
          x: colIdx,
          y: rowIdx,
          value,
          label: `${heatmap.rowLabels[rowIdx]} - ${heatmap.columnLabels[colIdx]}`
        })
      })
    })
    this.heatmaps.set(heatmap.id, heatmapData)
    this.saveHeatmaps()
    return heatmap
  }

  getHeatmap(id: string): HeatmapData[] {
    return this.heatmaps.get(id) || []
  }

  generateBackupActivityHeatmap(backups: { timestamp: Date; count: number }[]): { rowLabels: string[], columnLabels: string[], data: number[][] } {
    // Create heatmap: x = hour of day, y = day of week
    const heatmap: Map<string, number> = new Map()

    backups.forEach(backup => {
      const hour = backup.timestamp.getHours()
      const day = backup.timestamp.getDay()
      const key = `${day}-${hour}`

      heatmap.set(key, (heatmap.get(key) || 0) + backup.count)
    })

    const rowLabels = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
    const columnLabels = Array.from({ length: 24 }, (_, i) => `${i}:00`)

    const data: number[][] = []
    for (let day = 0; day < 7; day++) {
      const row: number[] = []
      for (let hour = 0; hour < 24; hour++) {
        const key = `${day}-${hour}`
        row.push(heatmap.get(key) || 0)
      }
      data.push(row)
    }

    return { rowLabels, columnLabels, data }
  }

  private loadHeatmaps(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-heatmaps')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        this.heatmaps = new Map(data)
      } catch (e) {
        console.error('Failed to load heatmaps:', e)
      }
    }
  }

  private saveHeatmaps(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-heatmaps', JSON.stringify(Array.from(this.heatmaps.entries())))
  }
}

// ==================== Network Usage Manager ====================

export class NetworkUsageManager {
  private usage: NetworkUsage[] = []

  constructor() {
    this.loadUsage()
  }

  addUsage(usage: NetworkUsage): NetworkUsage {
    this.usage.push(usage)
    this.saveUsage()
    return usage
  }

  getUsage(startDate?: Date, endDate?: Date): NetworkUsage[] {
    let filtered = [...this.usage]

    if (startDate) {
      filtered = filtered.filter(u => u.timestamp >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(u => u.timestamp <= endDate)
    }

    return filtered.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())
  }

  getChartData(startDate?: Date, endDate?: Date): any[] {
    const usage = this.getUsage(startDate, endDate)

    return usage.map((u: any) => ({
      date: u.timestamp.toISOString().split('T')[0],
      upload: u.uploadSpeed,
      download: u.downloadSpeed,
      latency: u.latency
    }))
  }

  private loadUsage(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-network-usage')
    if (stored) {
      try {
        this.usage = JSON.parse(stored)
        this.usage.forEach(u => u.timestamp = new Date(u.timestamp))
      } catch (e) {
        console.error('Failed to load network usage:', e)
      }
    }
  }

  private saveUsage(): void {
    if (typeof window === 'undefined') return
    const toSave = this.usage.slice(-10000)
    localStorage.setItem('visualization-network-usage', JSON.stringify(toSave))
  }
}

// ==================== Timeline Manager ====================

export class TimelineManager {
  private events: Map<string, TimelineEvent[]> = new Map()

  constructor() {
    this.loadTimelines()
  }

  addEvent(event: TimelineEvent): TimelineEvent {
    const timelineId = 'default'
    if (!this.events.has(timelineId)) {
      this.events.set(timelineId, [])
    }

    const fullEvent: TimelineEvent = {
      ...event,
      duration: event.endTime.getTime() - event.startTime.getTime()
    }

    this.events.get(timelineId)!.push(fullEvent)
    this.saveTimelines()
    return fullEvent
  }

  getEvents(): TimelineEvent[] {
    const timelineId = 'default'
    return this.events.get(timelineId) || []
  }

  updateEvent(eventId: string, updates: Partial<TimelineEvent>): boolean {
    const timelineId = 'default'
    const events = this.events.get(timelineId)
    if (!events) return false

    const event = events.find(e => e.id === eventId)
    if (!event) return false

    Object.assign(event, updates)
    if (updates.startTime || updates.endTime) {
      event.duration = event.endTime.getTime() - event.startTime.getTime()
    }
    this.saveTimelines()
    return true
  }

  getGanttData(): TimelineEvent[] {
    return this.getEvents().sort((a, b) => a.startTime.getTime() - b.startTime.getTime())
  }

  private loadTimelines(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-timelines')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        data.forEach(([id, events]: [string, TimelineEvent[]]) => {
          events.forEach(e => {
            e.startTime = new Date(e.startTime)
            e.endTime = new Date(e.endTime)
          })
          this.events.set(id, events)
        })
      } catch (e) {
        console.error('Failed to load timelines:', e)
      }
    }
  }

  private saveTimelines(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-timelines', JSON.stringify(Array.from(this.events.entries())))
  }
}

// ==================== Comparison Manager ====================

export class ComparisonManager {
  private comparisons: Map<string, ComparisonData[]> = new Map()

  constructor() {
    this.loadComparisons()
  }

  setComparison(comparison: { id: string, title: string, items: Array<{name: string, metrics: Record<string, number>}> }): { id: string, title: string, items: Array<{name: string, metrics: Record<string, number>}> } {
    const comparisonData: ComparisonData[] = comparison.items.map(item => ({
      label: item.name,
      values: item.metrics
    }))

    this.comparisons.set(comparison.id, comparisonData)
    this.saveComparisons()
    return comparison
  }

  getComparison(id: string): ComparisonData[] {
    return this.comparisons.get(id) || []
  }

  getComparisonChartData(id: string): any[] {
    const comparison = this.comparisons.get(id) || []

    return comparison.map(item => ({
      name: item.label,
      ...item.values
    }))
  }

  private loadComparisons(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-comparisons')
    if (stored) {
      try {
        const data = JSON.parse(stored)
        this.comparisons = new Map(data)
      } catch (e) {
        console.error('Failed to load comparisons:', e)
      }
    }
  }

  private saveComparisons(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-comparisons', JSON.stringify(Array.from(this.comparisons.entries())))
  }
}

// ==================== Dashboard Manager ====================

export class DashboardManager {
  private dashboards: Map<string, DashboardConfig> = new Map()
  private listeners: Set<(dashboard: DashboardConfig) => void> = new Set()

  constructor() {
    this.loadDashboards()
  }

  createDashboard(dashboard: Omit<DashboardConfig, 'createdAt'>): DashboardConfig {
    const fullDashboard: DashboardConfig = {
      ...dashboard,
      createdAt: new Date()
    }

    this.dashboards.set(fullDashboard.id, fullDashboard)
    this.saveDashboards()
    this.notifyListeners(fullDashboard)
    return fullDashboard
  }

  getDashboard(dashboardId: string): DashboardConfig | undefined {
    return this.dashboards.get(dashboardId)
  }

  getAllDashboards(): DashboardConfig[] {
    return Array.from(this.dashboards.values())
  }

  updateDashboard(dashboardId: string, updates: Partial<DashboardConfig>): boolean {
    const dashboard = this.dashboards.get(dashboardId)
    if (!dashboard) return false

    Object.assign(dashboard, updates, { updatedAt: new Date() })
    this.saveDashboards()
    this.notifyListeners(dashboard)
    return true
  }

  deleteDashboard(dashboardId: string): boolean {
    const deleted = this.dashboards.delete(dashboardId)
    if (deleted) this.saveDashboards()
    return deleted
  }

  subscribe(listener: (dashboard: DashboardConfig) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(dashboard: DashboardConfig): void {
    this.listeners.forEach(listener => listener(dashboard))
  }

  private loadDashboards(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('visualization-dashboards')
    if (stored) {
      try {
        const dashboards = JSON.parse(stored)
        dashboards.forEach((d: DashboardConfig) => {
          d.createdAt = new Date(d.createdAt)
          if (d.updatedAt) d.updatedAt = new Date(d.updatedAt)
          this.dashboards.set(d.id, d)
        })
      } catch (e) {
        console.error('Failed to load dashboards:', e)
      }
    }
  }

  private saveDashboards(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('visualization-dashboards', JSON.stringify(Array.from(this.dashboards.values())))
  }
}

// ==================== Export Manager ====================

export class ExportManager {
  async exportChart(chart: ChartConfig, format: string): Promise<{ format: string; data: string }> {
    // This would integrate with a charting library like Chart.js or D3
    // For now, we return a mock implementation

    switch (format) {
      case 'json':
        const jsonData = JSON.stringify({
          id: chart.id,
          type: chart.type,
          title: chart.title,
          data: chart.data,
          exportedAt: new Date().toISOString()
        }, null, 2)
        return { format: 'json', data: jsonData }

      case 'csv':
        const csvData = this.convertToCSV(chart)
        return { format: 'csv', data: csvData }

      case 'svg':
        const svgData = `<svg><title>${chart.title}</title></svg>`
        return { format: 'svg', data: svgData }

      case 'png':
        // Would use canvas.toDataURL() in real implementation
        const pngData = `data:image/png;base64,${btoa(JSON.stringify(chart))}`
        return { format: 'png', data: pngData }

      default:
        return { format: 'json', data: JSON.stringify(chart) }
    }
  }

  private convertToCSV(chart: ChartConfig): string {
    // Convert chart data to CSV format
    const headers = ['timestamp', 'value', 'label']
    const rows = chart.data.map(point => {
      const timestamp = point.timestamp ? point.timestamp.toISOString() : point.date || ''
      return `${timestamp},${point.value},${point.label}`
    })

    return [headers.join(','), ...rows].join('\n')
  }

  downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }
}

// ==================== Statistical Analysis Manager ====================

export class StatisticalAnalysisManager {
  analyzeDataset(values: number[]): StatisticalAnalysis {
    if (values.length === 0) {
      return {
        mean: 0, median: 0, mode: 0, stdDev: 0, variance: 0,
        min: 0, max: 0, q1: 0, q3: 0, iqr: 0, outliers: []
      }
    }

    const sorted = [...values].sort((a, b) => a - b)
    const n = values.length

    const mean = values.reduce((a, b) => a + b, 0) / n
    const median = n % 2 === 0
      ? (sorted[n / 2 - 1] + sorted[n / 2]) / 2
      : sorted[Math.floor(n / 2)]

    // Mode
    const frequency = new Map<number, number>()
    values.forEach(v => frequency.set(v, (frequency.get(v) || 0) + 1))
    const mode = Array.from(frequency.entries()).reduce((a, b) => a[1] > b[1] ? a : b)[0]

    // Variance and standard deviation
    const variance = values.reduce((sum, v) => sum + Math.pow(v - mean, 2), 0) / n
    const stdDev = Math.sqrt(variance)

    // Quartiles
    const q1 = sorted[Math.floor(n * 0.25)]
    const q3 = sorted[Math.floor(n * 0.75)]
    const iqr = q3 - q1

    // Outliers (values beyond 1.5 * IQR from quartiles)
    const lowerBound = q1 - 1.5 * iqr
    const upperBound = q3 + 1.5 * iqr
    const outliers = values.filter(v => v < lowerBound || v > upperBound)

    return {
      mean,
      median,
      mode,
      stdDev,
      variance,
      min: sorted[0],
      max: sorted[n - 1],
      q1,
      q3,
      iqr,
      outliers
    }
  }

  detectTrend(dataPoints: TimeSeriesPoint[]): TrendAnalysis {
    if (dataPoints.length < 2) {
      return {
        direction: 'stable',
        slope: 0,
        correlation: 0,
        forecast: [],
        confidence: 0
      }
    }

    // Simple linear regression
    const n = dataPoints.length
    const x = dataPoints.map((_, i) => i)
    const y = dataPoints.map(p => p.value)

    const sumX = x.reduce((a, b) => a + b, 0)
    const sumY = y.reduce((a, b) => a + b, 0)
    const sumXY = x.reduce((sum, xi, i) => sum + xi * y[i], 0)
    const sumX2 = x.reduce((sum, xi) => sum + xi * xi, 0)

    const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX)
    const intercept = (sumY - slope * sumX) / n

    // Correlation
    const meanX = sumX / n
    const meanY = sumY / n
    const correlation = x.reduce((sum, xi, i) => sum + (xi - meanX) * (y[i] - meanY), 0) /
      Math.sqrt(
        x.reduce((sum, xi) => sum + Math.pow(xi - meanX, 2), 0) *
        y.reduce((sum, yi) => sum + Math.pow(yi - meanY, 2), 0)
      )

    // Forecast next 5 points
    const forecast = []
    for (let i = 0; i < 5; i++) {
      forecast.push(slope * (n + i) + intercept)
    }

    const direction = slope > 0.01 ? 'up' : slope < -0.01 ? 'down' : 'stable'

    return {
      direction,
      slope,
      correlation,
      forecast,
      confidence: Math.abs(correlation) * 100
    }
  }
}

// ==================== Global Instances ====================

export const visualizationDataManager = new VisualizationDataManager()
export const chartConfigManager = new ChartConfigManager()
export const backupSizeTrendManager = new BackupSizeTrendManager()
export const databaseGrowthManager = new DatabaseGrowthManager()
export const costAnalysisManager = new CostAnalysisManager()
export const storageBreakdownManager = new StorageBreakdownManager()
export const successFailureRateManager = new SuccessFailureRateManager()
export const performanceMetricsManager = new PerformanceMetricsManager()
export const heatmapManager = new HeatmapManager()
export const networkUsageManager = new NetworkUsageManager()
export const timelineManager = new TimelineManager()
export const comparisonManager = new ComparisonManager()
export const dashboardManager = new DashboardManager()
export const exportManager = new ExportManager()
export const statisticalAnalysisManager = new StatisticalAnalysisManager()
