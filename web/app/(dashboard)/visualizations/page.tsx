'use client'

import { useEffect, useState } from 'react'
import {
  visualizationDataManager,
  chartConfigManager,
  backupSizeTrendManager,
  databaseGrowthManager,
  costAnalysisManager,
  storageBreakdownManager,
  successFailureRateManager,
  performanceMetricsManager,
  heatmapManager,
  networkUsageManager,
  timelineManager,
  comparisonManager,
  dashboardManager,
  statisticalAnalysisManager,
  type ChartConfig,
  type DashboardConfig,
} from '@/lib/visualization-manager'
import Chart from '@/components/visualizations/Chart'
import Dashboard from '@/components/visualizations/Dashboard'
import { SampleDataBadge } from '@/components/ui/sample-data-badge'

export default function VisualizationsPage() {
  const [activeTab, setActiveTab] = useState<'charts' | 'dashboards'>('charts')
  const [charts, setCharts] = useState<ChartConfig[]>([])
  const [dashboards, setDashboards] = useState<DashboardConfig[]>([])
  const [selectedDashboard, setSelectedDashboard] = useState<string | null>(null)

  useEffect(() => {
    // Initialize sample data
    initializeSampleData()

    // Subscribe to chart updates
    const unsubscribeCharts = chartConfigManager.subscribe(() => {
      setCharts(chartConfigManager.getAllCharts())
    })

    // Subscribe to dashboard updates
    const unsubscribeDashboards = dashboardManager.subscribe(() => {
      setDashboards(dashboardManager.getAllDashboards())
    })

    // Load initial data
    setCharts(chartConfigManager.getAllCharts())
    setDashboards(dashboardManager.getAllDashboards())

    return () => {
      unsubscribeCharts()
      unsubscribeDashboards()
    }
  }, [])

  const initializeSampleData = () => {
    // Only initialize if no data exists
    if (chartConfigManager.getAllCharts().length > 0) return

    // 1. Backup Size Trends (Line Chart)
    const backupTrends = []
    for (let i = 0; i < 30; i++) {
      backupSizeTrendManager.addTrend({
        backupId: `backup-${i}`,
        backupName: `Daily Backup ${i}`,
        database: 'production-db',
        timestamp: new Date(Date.now() - (29 - i) * 24 * 60 * 60 * 1000),
        sizeBytes: 1073741824 * (100 + Math.random() * 50 + i * 2), // Growing trend
        compressedSize: 536870912 * (100 + Math.random() * 50 + i * 2),
      })
    }

    const backupChartData = backupSizeTrendManager.getChartData()
    const backupChart = chartConfigManager.createChart({
      id: 'backup-size-trend',
      type: 'line',
      title: 'Backup Size Trends (Last 30 Days)',
      subtitle: 'Daily backup sizes with compression',
      data: backupChartData,
      options: {
        width: 800,
        height: 400,
        axis: {
          x: { label: 'Date', type: 'category' },
          y: { label: 'Size (GB)', type: 'linear' },
        },
        tooltip: { enabled: true, shared: true, valueSuffix: ' GB' },
      },
    })

    // 2. Database Growth (Area Chart)
    for (let i = 0; i < 12; i++) {
      const size = 10737418240 * (10 + i * 1.5)
      databaseGrowthManager.addGrowthPoint({
        databaseId: 'prod-db',
        databaseName: 'Production Database',
        timestamp: new Date(Date.now() - (11 - i) * 30 * 24 * 60 * 60 * 1000),
        sizeBytes: size,
        totalSize: size,
        dataSize: size * 0.7,
        indexSize: size * 0.3,
        recordCount: 1000000 * (10 + i * 1.2),
        tableCount: 50 + i * 2,
        rowCount: 1000000 * (10 + i * 1.2),
      })
    }

    const growthChartData = databaseGrowthManager.getChartData()
    chartConfigManager.createChart({
      id: 'database-growth',
      type: 'area',
      title: 'Database Growth Over Time',
      subtitle: 'Monthly database size progression',
      data: growthChartData,
      options: {
        width: 800,
        height: 400,
        axis: {
          x: { label: 'Month', type: 'category' },
          y: { label: 'Size (GB)', type: 'linear' },
        },
      },
    })

    // 3. Cost Analysis (Stacked Bar Chart)
    const databases = ['prod-db', 'staging-db', 'dev-db']
    for (let i = 0; i < 6; i++) {
      databases.forEach((db) => {
        costAnalysisManager.addCost({
          date: new Date(Date.now() - (5 - i) * 30 * 24 * 60 * 60 * 1000),
          database: db,
          storageCost: 100 + Math.random() * 50,
          computeCost: 50 + Math.random() * 30,
          networkCost: 25 + Math.random() * 15,
          totalCost: 175 + Math.random() * 95,
          currency: 'USD',
        })
      })
    }

    const costChartData = costAnalysisManager.getStackedBarData()
    chartConfigManager.createChart({
      id: 'cost-analysis',
      type: 'stackedBar',
      title: 'Cost Analysis by Category',
      subtitle: 'Storage, compute, and network costs',
      data: costChartData.map((d: any) => ({
        label: d.date,
        value: d.total,
        timestamp: new Date(d.date),
        metadata: {
          storage: d.storage,
          compute: d.compute,
          network: d.network,
        },
      })),
      options: {
        width: 800,
        height: 400,
        axis: {
          x: { label: 'Month', type: 'category' },
          y: { label: 'Cost (USD)', type: 'linear' },
        },
        tooltip: { enabled: true, valuePrefix: '$' },
        stacked: true,
      },
    })

    // 4. Storage Breakdown (Pie Chart)
    storageBreakdownManager.setBreakdown({
      timestamp: new Date(),
      categories: [
        { name: 'Full Backups', sizeBytes: 10737418240, percentage: 0 },
        { name: 'Incremental Backups', sizeBytes: 5368709120, percentage: 0 },
        { name: 'Logs', sizeBytes: 2147483648, percentage: 0 },
        { name: 'Archives', sizeBytes: 3221225472, percentage: 0 },
      ],
      totalSize: 21474836480,
    })

    const storageChartData = storageBreakdownManager.getPieChartData(
      new Date().toISOString()
    )
    chartConfigManager.createChart({
      id: 'storage-breakdown',
      type: 'pie',
      title: 'Storage Breakdown by Category',
      subtitle: 'Current storage distribution',
      data: storageChartData,
      options: {
        width: 600,
        height: 400,
        legend: { enabled: true, position: 'right' },
      },
    })

    // 5. Success/Failure Rate (Combo Chart)
    for (let i = 0; i < 14; i++) {
      successFailureRateManager.addRate({
        timestamp: new Date(Date.now() - (13 - i) * 24 * 60 * 60 * 1000),
        totalBackups: 100,
        successfulBackups: 90 + Math.floor(Math.random() * 8),
        failedBackups: 10 - Math.floor(Math.random() * 8),
        averageDuration: 3600000 + Math.random() * 1800000,
      })
    }

    const rateChartData = successFailureRateManager.getComboChartData()
    chartConfigManager.createChart({
      id: 'success-failure-rate',
      type: 'bar',
      title: 'Backup Success/Failure Rates',
      subtitle: 'Daily backup completion statistics',
      data: rateChartData.map((d: any) => ({
        label: d.date,
        value: d.successRate,
        timestamp: new Date(d.date),
        metadata: {
          successCount: d.successCount,
          failureCount: d.failureCount,
        },
      })),
      options: {
        width: 800,
        height: 400,
        axis: {
          x: { label: 'Date', type: 'category' },
          y: { label: 'Success Rate (%)', type: 'linear', min: 0, max: 100 },
        },
      },
    })

    // 6. Performance Metrics (Multi-series Line Chart)
    for (let i = 0; i < 24; i++) {
      performanceMetricsManager.addMetric({
        timestamp: new Date(Date.now() - (23 - i) * 60 * 60 * 1000),
        backupSpeed: 100 + Math.random() * 50,
        restoreSpeed: 80 + Math.random() * 40,
        cpuUsage: 30 + Math.random() * 40,
        memoryUsage: 40 + Math.random() * 30,
        diskIo: 50 + Math.random() * 40,
        networkThroughput: 120 + Math.random() * 60,
      })
    }

    const perfChartData = performanceMetricsManager.getMultiSeriesData()
    chartConfigManager.createChart({
      id: 'performance-metrics',
      type: 'line',
      title: 'Performance Metrics (Last 24 Hours)',
      subtitle: 'CPU, memory, and I/O utilization',
      data: perfChartData.map((d: any) => ({
        label: d.date,
        value: d.cpuUsage,
        timestamp: new Date(d.date),
        metadata: d,
      })),
      options: {
        width: 800,
        height: 400,
        axis: {
          x: { label: 'Time', type: 'category' },
          y: { label: 'Usage (%)', type: 'linear', min: 0, max: 100 },
        },
      },
    })

    // Create a main dashboard
    const allCharts = chartConfigManager.getAllCharts()
    dashboardManager.createDashboard({
      id: 'main-dashboard',
      name: 'Backup Operations Dashboard',
      description: 'Comprehensive overview of backup operations and metrics',
      layout: {
        type: 'grid',
        columns: 2,
        gap: 4,
        positions: allCharts.map((chart, index) => ({
          chartId: chart.id,
          x: index % 2,
          y: Math.floor(index / 2),
          width: 1,
          height: 1,
        })),
      },
      charts: allCharts,
      autoRefresh: true,
      refreshInterval: 60000, // 1 minute
      filters: [
        {
          id: 'time-range',
          label: 'Time Range',
          type: 'select',
          options: ['Last 24 Hours', 'Last 7 Days', 'Last 30 Days', 'Last Year'],
          defaultValue: 'Last 30 Days',
        },
        {
          id: 'database',
          label: 'Database',
          type: 'select',
          options: ['All', 'Production', 'Staging', 'Development'],
          defaultValue: 'All',
        },
      ],
    })

    // Set the first dashboard as selected
    setSelectedDashboard('main-dashboard')

    // Refresh state
    setCharts(chartConfigManager.getAllCharts())
    setDashboards(dashboardManager.getAllDashboards())
  }

  return (
    <div className="container mx-auto p-6">
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-4xl font-bold">Data Visualizations</h1>
          <SampleDataBadge />
        </div>
        <p className="text-gray-600">
          Advanced charts and dashboards for backup operations monitoring and analytics
        </p>
        <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          These charts render generated sample data to demonstrate the visualization
          engine. They are not backed by live backup metrics.
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b mb-6">
        <nav className="flex gap-4">
          <button
            onClick={() => setActiveTab('charts')}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === 'charts'
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            Individual Charts ({charts.length})
          </button>
          <button
            onClick={() => setActiveTab('dashboards')}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === 'dashboards'
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            Dashboards ({dashboards.length})
          </button>
        </nav>
      </div>

      {/* Content */}
      {activeTab === 'charts' ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {charts.length === 0 ? (
            <div className="col-span-2 text-center py-12">
              <p className="text-gray-500 mb-4">No charts available</p>
              <button
                onClick={initializeSampleData}
                className="px-6 py-3 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90"
              >
                Initialize Sample Data
              </button>
            </div>
          ) : (
            charts.map((chart) => (
              <div key={chart.id} className="flex flex-col">
                <Chart config={chart} />
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="space-y-6">
          {dashboards.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-gray-500 mb-4">No dashboards available</p>
              <button
                onClick={initializeSampleData}
                className="px-6 py-3 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90"
              >
                Initialize Sample Data
              </button>
            </div>
          ) : selectedDashboard ? (
            <>
              {/* Dashboard selector */}
              <div className="flex gap-2">
                {dashboards.map((dashboard) => (
                  <button
                    key={dashboard.id}
                    onClick={() => setSelectedDashboard(dashboard.id)}
                    className={`px-4 py-2 rounded-lg transition-colors ${
                      selectedDashboard === dashboard.id
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                    }`}
                  >
                    {dashboard.name}
                  </button>
                ))}
              </div>

              {/* Dashboard content */}
              <Dashboard dashboardId={selectedDashboard} autoRefresh />
            </>
          ) : null}
        </div>
      )}

      {/* Statistics Panel */}
      {charts.length > 0 && (
        <div className="mt-12 p-6 bg-gray-50 rounded-lg">
          <h3 className="text-lg font-semibold mb-4">System Statistics</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-white p-4 rounded-lg">
              <div className="text-sm text-gray-600">Total Charts</div>
              <div className="text-2xl font-bold text-primary">{charts.length}</div>
            </div>
            <div className="bg-white p-4 rounded-lg">
              <div className="text-sm text-gray-600">Dashboards</div>
              <div className="text-2xl font-bold text-primary">{dashboards.length}</div>
            </div>
            <div className="bg-white p-4 rounded-lg">
              <div className="text-sm text-gray-600">Data Points</div>
              <div className="text-2xl font-bold text-primary">
                {charts.reduce((sum, chart) => sum + chart.data.length, 0)}
              </div>
            </div>
            <div className="bg-white p-4 rounded-lg">
              <div className="text-sm text-gray-600">Chart Types</div>
              <div className="text-2xl font-bold text-primary">
                {new Set(charts.map((c) => c.type)).size}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
