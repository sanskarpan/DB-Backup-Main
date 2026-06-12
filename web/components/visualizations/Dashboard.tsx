'use client'

import React, { useEffect, useState } from 'react'
import type { DashboardConfig } from '@/lib/visualization-manager'
import { dashboardManager } from '@/lib/visualization-manager'
import Chart from './Chart'

interface DashboardProps {
  dashboardId: string
  autoRefresh?: boolean
  onRefresh?: () => void
}

export default function Dashboard({
  dashboardId,
  autoRefresh,
  onRefresh,
}: DashboardProps) {
  const [dashboard, setDashboard] = useState<DashboardConfig | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())

  useEffect(() => {
    // Load dashboard
    const loadDashboard = () => {
      const config = dashboardManager.getDashboard(dashboardId)
      if (config) {
        setDashboard(config)
        setLastRefresh(new Date())
      }
    }

    loadDashboard()

    // Subscribe to dashboard updates
    const unsubscribe = dashboardManager.subscribe((updatedDashboard) => {
      if (updatedDashboard.id === dashboardId) {
        setDashboard(updatedDashboard)
      }
    })

    return () => {
      unsubscribe()
    }
  }, [dashboardId])

  useEffect(() => {
    // Setup auto-refresh
    if (autoRefresh && dashboard?.autoRefresh && dashboard?.refreshInterval) {
      const interval = setInterval(() => {
        setLastRefresh(new Date())
        if (onRefresh) {
          onRefresh()
        }
      }, dashboard.refreshInterval)

      return () => {
        clearInterval(interval)
      }
    }
  }, [autoRefresh, dashboard, onRefresh])

  if (!dashboard) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-gray-500">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  const getLayoutClasses = () => {
    switch (dashboard.layout.type) {
      case 'grid':
        return `grid gap-${dashboard.layout.gap || 4} grid-cols-${
          dashboard.layout.columns || 2
        }`

      case 'flex':
        return `flex flex-wrap gap-${dashboard.layout.gap || 4}`

      case 'masonry':
        return `columns-${dashboard.layout.columns || 2} gap-${
          dashboard.layout.gap || 4
        }`

      default:
        return 'grid grid-cols-2 gap-4'
    }
  }

  const getChartContainerStyle = (chartId: string) => {
    if (dashboard.layout.type === 'grid' && dashboard.layout.positions) {
      const position = dashboard.layout.positions.find(
        (p) => p.chartId === chartId
      )
      if (position) {
        return {
          gridColumn: `${position.x + 1} / span ${position.width}`,
          gridRow: `${position.y + 1} / span ${position.height}`,
        }
      }
    }
    return {}
  }

  return (
    <div className="space-y-4">
      {/* Dashboard Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">{dashboard.name}</h2>
          {dashboard.description && (
            <p className="text-gray-600 mt-1">{dashboard.description}</p>
          )}
        </div>

        <div className="flex items-center gap-4">
          {/* Refresh indicator */}
          {dashboard.autoRefresh && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
              <span>
                Auto-refresh: {(dashboard.refreshInterval || 0) / 1000}s
              </span>
            </div>
          )}

          {/* Last refresh time */}
          <div className="text-sm text-gray-500">
            Last updated: {lastRefresh.toLocaleTimeString()}
          </div>

          {/* Manual refresh button */}
          <button
            onClick={() => {
              setLastRefresh(new Date())
              if (onRefresh) {
                onRefresh()
              }
            }}
            className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Dashboard Filters */}
      {dashboard.filters && dashboard.filters.length > 0 && (
        <div className="flex gap-4 p-4 bg-gray-50 rounded-lg">
          {dashboard.filters.map((filter) => (
            <div key={filter.id} className="flex flex-col gap-2">
              <label className="text-sm font-medium">{filter.label}</label>
              {filter.type === 'select' && (
                <select className="px-3 py-2 border rounded-lg">
                  {filter.options?.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              )}
              {filter.type === 'date' && (
                <input
                  type="date"
                  className="px-3 py-2 border rounded-lg"
                  defaultValue={filter.defaultValue}
                />
              )}
              {filter.type === 'range' && (
                <input
                  type="range"
                  className="w-full"
                  defaultValue={filter.defaultValue}
                />
              )}
            </div>
          ))}
        </div>
      )}

      {/* Charts Grid/Layout */}
      {dashboard.charts.length === 0 ? (
        <div className="flex items-center justify-center h-64 border-2 border-dashed rounded-lg">
          <div className="text-center text-gray-500">
            <p className="text-lg font-medium">No charts in this dashboard</p>
            <p className="text-sm mt-1">Add charts to get started</p>
          </div>
        </div>
      ) : (
        <div className={getLayoutClasses()}>
          {dashboard.charts.map((chart) => (
            <div
              key={chart.id}
              style={getChartContainerStyle(chart.id)}
              className={
                dashboard.layout.type === 'masonry' ? 'break-inside-avoid mb-4' : ''
              }
            >
              <Chart config={chart} />
            </div>
          ))}
        </div>
      )}

      {/* Dashboard Footer */}
      <div className="flex items-center justify-between text-sm text-gray-500 pt-4 border-t">
        <div>
          Dashboard ID: <code className="text-xs bg-gray-100 px-2 py-1 rounded">{dashboard.id}</code>
        </div>
        <div>
          {dashboard.charts.length} chart{dashboard.charts.length !== 1 ? 's' : ''}
        </div>
      </div>
    </div>
  )
}
