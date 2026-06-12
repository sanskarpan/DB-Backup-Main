'use client'

import React from 'react'
import { Activity, TrendingUp, Clock, Database, Zap, AlertCircle } from 'lucide-react'
import { clsx } from 'clsx'

export interface DataPoint {
  timestamp: Date
  value: number
  label?: string
}

export interface ChartSeries {
  name: string
  data: DataPoint[]
  color: string
  unit?: string
}

interface RealTimeChartsProps {
  series: ChartSeries[]
  height?: number
  maxDataPoints?: number
  refreshInterval?: number
  showLegend?: boolean
  showGrid?: boolean
  yAxisLabel?: string
  xAxisLabel?: string
  className?: string
}

export function RealTimeCharts({
  series,
  height = 300,
  maxDataPoints = 60,
  refreshInterval,
  showLegend = true,
  showGrid = true,
  yAxisLabel,
  xAxisLabel,
  className,
}: RealTimeChartsProps) {
  const [hoveredPoint, setHoveredPoint] = React.useState<{
    seriesIndex: number
    pointIndex: number
  } | null>(null)

  // Calculate chart dimensions
  const padding = { top: 20, right: 20, bottom: 40, left: 60 }
  const chartWidth = 800 // Fixed width, can be made responsive
  const chartHeight = height
  const innerWidth = chartWidth - padding.left - padding.right
  const innerHeight = chartHeight - padding.top - padding.bottom

  // Get all data points for scaling
  const allDataPoints = series.flatMap((s) => s.data)
  const minValue = Math.min(...allDataPoints.map((p) => p.value), 0)
  const maxValue = Math.max(...allDataPoints.map((p) => p.value), 1)
  const valueRange = maxValue - minValue || 1

  // Get time range
  const allTimestamps = allDataPoints.map((p) => p.timestamp.getTime())
  const minTime = Math.min(...allTimestamps)
  const maxTime = Math.max(...allTimestamps)
  const timeRange = maxTime - minTime || 1

  // Scale functions
  const scaleX = (timestamp: Date) => {
    return ((timestamp.getTime() - minTime) / timeRange) * innerWidth
  }

  const scaleY = (value: number) => {
    return innerHeight - ((value - minValue) / valueRange) * innerHeight
  }

  // Generate path for line chart
  const generatePath = (data: DataPoint[]) => {
    if (data.length === 0) return ''

    const points = data.map((point) => ({
      x: scaleX(point.timestamp),
      y: scaleY(point.value),
    }))

    let path = `M ${points[0].x} ${points[0].y}`
    for (let i = 1; i < points.length; i++) {
      path += ` L ${points[i].x} ${points[i].y}`
    }

    return path
  }

  // Generate grid lines
  const gridLines = showGrid
    ? Array.from({ length: 5 }, (_, i) => {
        const y = (innerHeight / 4) * i
        const value = maxValue - (valueRange / 4) * i
        return { y, value }
      })
    : []

  return (
    <div className={clsx('bg-white rounded-lg shadow-sm border border-gray-200 p-6', className)}>
      <div className="space-y-4">
        {/* Legend */}
        {showLegend && (
          <div className="flex items-center gap-4 flex-wrap">
            {series.map((s, index) => (
              <div key={index} className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full" style={{ backgroundColor: s.color }} />
                <span className="text-sm text-gray-700 font-medium">{s.name}</span>
                {s.unit && <span className="text-xs text-gray-500">({s.unit})</span>}
              </div>
            ))}
          </div>
        )}

        {/* Chart */}
        <div className="relative overflow-x-auto">
          <svg width={chartWidth} height={chartHeight} className="select-none">
            <g transform={`translate(${padding.left}, ${padding.top})`}>
              {/* Grid lines */}
              {gridLines.map((line, i) => (
                <g key={i}>
                  <line
                    x1={0}
                    y1={line.y}
                    x2={innerWidth}
                    y2={line.y}
                    stroke="#e5e7eb"
                    strokeWidth="1"
                    strokeDasharray="4 2"
                  />
                  <text x={-10} y={line.y} textAnchor="end" fontSize="12" fill="#6b7280" dy="0.3em">
                    {line.value.toFixed(0)}
                  </text>
                </g>
              ))}

              {/* X-axis */}
              <line x1={0} y1={innerHeight} x2={innerWidth} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />

              {/* Y-axis */}
              <line x1={0} y1={0} x2={0} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />

              {/* Data lines */}
              {series.map((s, seriesIndex) => (
                <g key={seriesIndex}>
                  <path
                    d={generatePath(s.data)}
                    fill="none"
                    stroke={s.color}
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />

                  {/* Data points */}
                  {s.data.map((point, pointIndex) => {
                    const cx = scaleX(point.timestamp)
                    const cy = scaleY(point.value)
                    const isHovered =
                      hoveredPoint?.seriesIndex === seriesIndex && hoveredPoint?.pointIndex === pointIndex

                    return (
                      <circle
                        key={pointIndex}
                        cx={cx}
                        cy={cy}
                        r={isHovered ? 6 : 4}
                        fill={s.color}
                        stroke="white"
                        strokeWidth="2"
                        className="cursor-pointer transition-all"
                        onMouseEnter={() => setHoveredPoint({ seriesIndex, pointIndex })}
                        onMouseLeave={() => setHoveredPoint(null)}
                      />
                    )
                  })}
                </g>
              ))}

              {/* Axis labels */}
              {yAxisLabel && (
                <text
                  x={-innerHeight / 2}
                  y={-45}
                  transform="rotate(-90)"
                  textAnchor="middle"
                  fontSize="12"
                  fill="#4b5563"
                  fontWeight="500"
                >
                  {yAxisLabel}
                </text>
              )}

              {xAxisLabel && (
                <text
                  x={innerWidth / 2}
                  y={innerHeight + 35}
                  textAnchor="middle"
                  fontSize="12"
                  fill="#4b5563"
                  fontWeight="500"
                >
                  {xAxisLabel}
                </text>
              )}

              {/* Time labels */}
              {series.length > 0 && series[0].data.length > 0 && (
                <>
                  <text x={0} y={innerHeight + 20} textAnchor="start" fontSize="10" fill="#6b7280">
                    {new Date(minTime).toLocaleTimeString()}
                  </text>
                  <text x={innerWidth} y={innerHeight + 20} textAnchor="end" fontSize="10" fill="#6b7280">
                    {new Date(maxTime).toLocaleTimeString()}
                  </text>
                </>
              )}
            </g>
          </svg>
        </div>

        {/* Tooltip */}
        {hoveredPoint !== null && (
          <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg border border-gray-200">
            <div className="flex items-center gap-3">
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: series[hoveredPoint.seriesIndex].color }}
              />
              <div>
                <div className="text-sm font-medium text-gray-900">
                  {series[hoveredPoint.seriesIndex].name}
                </div>
                <div className="text-xs text-gray-500">
                  {series[hoveredPoint.seriesIndex].data[hoveredPoint.pointIndex].timestamp.toLocaleString()}
                </div>
              </div>
            </div>
            <div className="text-lg font-bold text-gray-900">
              {series[hoveredPoint.seriesIndex].data[hoveredPoint.pointIndex].value.toFixed(2)}
              {series[hoveredPoint.seriesIndex].unit && (
                <span className="text-sm text-gray-500 ml-1">{series[hoveredPoint.seriesIndex].unit}</span>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// Area Chart Variant
export function RealTimeAreaChart({
  series,
  height = 300,
  maxDataPoints = 60,
  showLegend = true,
  showGrid = true,
  yAxisLabel,
  xAxisLabel,
  className,
}: RealTimeChartsProps) {
  const padding = { top: 20, right: 20, bottom: 40, left: 60 }
  const chartWidth = 800
  const chartHeight = height
  const innerWidth = chartWidth - padding.left - padding.right
  const innerHeight = chartHeight - padding.top - padding.bottom

  const allDataPoints = series.flatMap((s) => s.data)
  const minValue = Math.min(...allDataPoints.map((p) => p.value), 0)
  const maxValue = Math.max(...allDataPoints.map((p) => p.value), 1)
  const valueRange = maxValue - minValue || 1

  const allTimestamps = allDataPoints.map((p) => p.timestamp.getTime())
  const minTime = Math.min(...allTimestamps)
  const maxTime = Math.max(...allTimestamps)
  const timeRange = maxTime - minTime || 1

  const scaleX = (timestamp: Date) => {
    return ((timestamp.getTime() - minTime) / timeRange) * innerWidth
  }

  const scaleY = (value: number) => {
    return innerHeight - ((value - minValue) / valueRange) * innerHeight
  }

  const generateAreaPath = (data: DataPoint[]) => {
    if (data.length === 0) return ''

    const points = data.map((point) => ({
      x: scaleX(point.timestamp),
      y: scaleY(point.value),
    }))

    let path = `M ${points[0].x} ${innerHeight}`
    path += ` L ${points[0].x} ${points[0].y}`

    for (let i = 1; i < points.length; i++) {
      path += ` L ${points[i].x} ${points[i].y}`
    }

    path += ` L ${points[points.length - 1].x} ${innerHeight}`
    path += ' Z'

    return path
  }

  return (
    <div className={clsx('bg-white rounded-lg shadow-sm border border-gray-200 p-6', className)}>
      <div className="space-y-4">
        {showLegend && (
          <div className="flex items-center gap-4 flex-wrap">
            {series.map((s, index) => (
              <div key={index} className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full" style={{ backgroundColor: s.color }} />
                <span className="text-sm text-gray-700 font-medium">{s.name}</span>
              </div>
            ))}
          </div>
        )}

        <div className="relative overflow-x-auto">
          <svg width={chartWidth} height={chartHeight}>
            <defs>
              {series.map((s, index) => (
                <linearGradient key={index} id={`gradient-${index}`} x1="0%" y1="0%" x2="0%" y2="100%">
                  <stop offset="0%" stopColor={s.color} stopOpacity="0.3" />
                  <stop offset="100%" stopColor={s.color} stopOpacity="0.05" />
                </linearGradient>
              ))}
            </defs>

            <g transform={`translate(${padding.left}, ${padding.top})`}>
              {/* Grid */}
              {showGrid &&
                Array.from({ length: 5 }, (_, i) => {
                  const y = (innerHeight / 4) * i
                  return (
                    <line
                      key={i}
                      x1={0}
                      y1={y}
                      x2={innerWidth}
                      y2={y}
                      stroke="#e5e7eb"
                      strokeWidth="1"
                      strokeDasharray="4 2"
                    />
                  )
                })}

              {/* Areas */}
              {series.map((s, index) => (
                <path
                  key={index}
                  d={generateAreaPath(s.data)}
                  fill={`url(#gradient-${index})`}
                  stroke={s.color}
                  strokeWidth="2"
                />
              ))}

              {/* Axes */}
              <line x1={0} y1={innerHeight} x2={innerWidth} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />
              <line x1={0} y1={0} x2={0} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />
            </g>
          </svg>
        </div>
      </div>
    </div>
  )
}

// Bar Chart for discrete values
interface BarChartProps {
  data: { label: string; value: number; color?: string }[]
  height?: number
  yAxisLabel?: string
  className?: string
}

export function RealTimeBarChart({ data, height = 300, yAxisLabel, className }: BarChartProps) {
  const padding = { top: 20, right: 20, bottom: 60, left: 60 }
  const chartWidth = 800
  const chartHeight = height
  const innerWidth = chartWidth - padding.left - padding.right
  const innerHeight = chartHeight - padding.top - padding.bottom

  const maxValue = Math.max(...data.map((d) => d.value), 1)
  const barWidth = innerWidth / data.length - 10

  return (
    <div className={clsx('bg-white rounded-lg shadow-sm border border-gray-200 p-6', className)}>
      <svg width={chartWidth} height={chartHeight}>
        <g transform={`translate(${padding.left}, ${padding.top})`}>
          {/* Bars */}
          {data.map((item, index) => {
            const x = (innerWidth / data.length) * index + 5
            const barHeight = (item.value / maxValue) * innerHeight
            const y = innerHeight - barHeight

            return (
              <g key={index}>
                <rect
                  x={x}
                  y={y}
                  width={barWidth}
                  height={barHeight}
                  fill={item.color || '#3b82f6'}
                  rx="4"
                  className="hover:opacity-80 transition-opacity cursor-pointer"
                />
                <text
                  x={x + barWidth / 2}
                  y={innerHeight + 15}
                  textAnchor="middle"
                  fontSize="10"
                  fill="#6b7280"
                  transform={`rotate(-45, ${x + barWidth / 2}, ${innerHeight + 15})`}
                >
                  {item.label}
                </text>
                <text
                  x={x + barWidth / 2}
                  y={y - 5}
                  textAnchor="middle"
                  fontSize="12"
                  fill="#374151"
                  fontWeight="500"
                >
                  {item.value.toFixed(0)}
                </text>
              </g>
            )
          })}

          {/* Axes */}
          <line x1={0} y1={innerHeight} x2={innerWidth} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />
          <line x1={0} y1={0} x2={0} y2={innerHeight} stroke="#9ca3af" strokeWidth="2" />

          {/* Y-axis label */}
          {yAxisLabel && (
            <text
              x={-innerHeight / 2}
              y={-45}
              transform="rotate(-90)"
              textAnchor="middle"
              fontSize="12"
              fill="#4b5563"
              fontWeight="500"
            >
              {yAxisLabel}
            </text>
          )}
        </g>
      </svg>
    </div>
  )
}

// Sparkline for compact display
interface SparklineProps {
  data: number[]
  width?: number
  height?: number
  color?: string
  showArea?: boolean
  className?: string
}

export function Sparkline({
  data,
  width = 100,
  height = 30,
  color = '#3b82f6',
  showArea = true,
  className,
}: SparklineProps) {
  if (data.length === 0) return null

  const minValue = Math.min(...data)
  const maxValue = Math.max(...data)
  const valueRange = maxValue - minValue || 1

  const points = data.map((value, index) => {
    const x = (width / (data.length - 1)) * index
    const y = height - ((value - minValue) / valueRange) * height
    return { x, y }
  })

  const linePath = `M ${points.map((p) => `${p.x},${p.y}`).join(' L ')}`

  const areaPath = showArea
    ? `M 0,${height} L ${points.map((p) => `${p.x},${p.y}`).join(' L ')} L ${width},${height} Z`
    : ''

  return (
    <svg width={width} height={height} className={className}>
      {showArea && <path d={areaPath} fill={color} fillOpacity="0.2" />}
      <path d={linePath} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

// Sample data generators
export function generateTimeSeriesData(points: number = 60, baseValue: number = 50): DataPoint[] {
  const now = Date.now()
  return Array.from({ length: points }, (_, i) => ({
    timestamp: new Date(now - (points - i) * 1000), // 1 second intervals
    value: baseValue + Math.random() * 20 - 10 + Math.sin(i / 10) * 10,
  }))
}

export function generateMultiSeriesData(seriesCount: number = 3, points: number = 60): ChartSeries[] {
  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6']
  const names = ['CPU Usage', 'Memory Usage', 'Disk I/O', 'Network Traffic', 'Requests/sec']
  const units = ['%', '%', 'MB/s', 'MB/s', 'req/s']

  return Array.from({ length: seriesCount }, (_, i) => ({
    name: names[i % names.length],
    color: colors[i % colors.length],
    unit: units[i % units.length],
    data: generateTimeSeriesData(points, 50 + i * 10),
  }))
}
