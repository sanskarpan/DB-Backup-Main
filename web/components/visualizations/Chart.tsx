'use client'

import React, { useEffect, useRef, useState } from 'react'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions as ChartJSOptions,
  TimeScale,
  Filler,
} from 'chart.js'
import { Line, Bar, Pie, Doughnut, Scatter } from 'react-chartjs-2'
import 'chartjs-adapter-date-fns'
import type { ChartConfig, ChartType } from '@/lib/visualization-manager'
import { exportManager } from '@/lib/visualization-manager'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler
)

interface ChartProps {
  config: ChartConfig
  onExport?: (format: 'png' | 'svg' | 'csv' | 'json') => void
}

export default function Chart({ config, onExport }: ChartProps) {
  const chartRef = useRef<any>(null)
  const [isExporting, setIsExporting] = useState(false)

  // Prepare chart data based on type
  const getChartData = () => {
    const labels = config.data.map((d) => d.label || d.date || d.timestamp?.toISOString() || '')
    const values = config.data.map((d) => d.value)

    const baseColors = config.options.colors || config.theme?.colors || [
      '#3b82f6',
      '#10b981',
      '#f59e0b',
      '#ef4444',
      '#8b5cf6',
      '#06b6d4',
    ]

    switch (config.type) {
      case 'line':
      case 'area':
        return {
          labels,
          datasets: [
            {
              label: config.title,
              data: values,
              borderColor: baseColors[0],
              backgroundColor:
                config.type === 'area'
                  ? `${baseColors[0]}33`
                  : 'transparent',
              fill: config.type === 'area',
              tension: 0.4,
            },
          ],
        }

      case 'bar':
      case 'stackedBar':
        return {
          labels,
          datasets: [
            {
              label: config.title,
              data: values,
              backgroundColor: baseColors[0],
              borderColor: baseColors[0],
              borderWidth: 1,
            },
          ],
        }

      case 'pie':
      case 'donut':
        return {
          labels,
          datasets: [
            {
              label: config.title,
              data: values,
              backgroundColor: baseColors,
              borderColor: config.theme?.backgroundColor || '#ffffff',
              borderWidth: 2,
            },
          ],
        }

      case 'scatter':
        return {
          datasets: [
            {
              label: config.title,
              data: config.data.map((d) => ({
                x: d.timestamp?.getTime() || 0,
                y: d.value,
              })),
              backgroundColor: baseColors[0],
            },
          ],
        }

      default:
        return { labels, datasets: [] }
    }
  }

  // Prepare chart options
  const getChartOptions = (): ChartJSOptions<any> => {
    const baseOptions: ChartJSOptions<any> = {
      responsive: config.responsive !== false,
      maintainAspectRatio: false,
      animation: config.animated !== false ? { duration: 750 } : false,
      plugins: {
        title: {
          display: !!config.title,
          text: config.title,
          font: {
            size: config.theme?.fontSize ? config.theme.fontSize + 4 : 16,
            family: config.theme?.fontFamily || 'Inter, system-ui, sans-serif',
          },
          color: config.theme?.textColor || '#333333',
        },
        subtitle: {
          display: !!config.subtitle,
          text: config.subtitle,
          font: {
            size: config.theme?.fontSize || 12,
            family: config.theme?.fontFamily || 'Inter, system-ui, sans-serif',
          },
          color: config.theme?.textColor || '#666666',
        },
        legend: {
          display: config.options.legend?.enabled !== false,
          position: config.options.legend?.position || 'top',
          align: config.options.legend?.align || 'center',
          labels: {
            font: {
              size: config.theme?.fontSize || 12,
              family: config.theme?.fontFamily || 'Inter, system-ui, sans-serif',
            },
            color: config.theme?.textColor || '#333333',
          },
        },
        tooltip: {
          enabled: config.options.tooltip?.enabled !== false,
          mode: config.options.tooltip?.shared ? 'index' : 'nearest',
          intersect: !config.options.tooltip?.shared,
          backgroundColor: 'rgba(0, 0, 0, 0.8)',
          titleColor: '#ffffff',
          bodyColor: '#ffffff',
          borderColor: config.theme?.gridColor || '#e0e0e0',
          borderWidth: 1,
          callbacks: {
            label: (context: any) => {
              let label = context.dataset.label || ''
              if (label) {
                label += ': '
              }
              const prefix = config.options.tooltip?.valuePrefix || ''
              const suffix = config.options.tooltip?.valueSuffix || ''
              label += prefix + context.parsed.y + suffix
              return label
            },
          },
        },
      },
      scales:
        config.type === 'pie' || config.type === 'donut'
          ? undefined
          : {
              x: {
                type: config.options.axis?.x?.type || 'category',
                title: {
                  display: !!config.options.axis?.x?.label,
                  text: config.options.axis?.x?.label || '',
                  color: config.theme?.textColor || '#333333',
                },
                grid: {
                  display: config.options.grid?.x !== false,
                  color: config.theme?.gridColor || '#e0e0e0',
                },
                ticks: {
                  color: config.theme?.textColor || '#333333',
                },
              },
              y: {
                type: config.options.axis?.y?.type || 'linear',
                title: {
                  display: !!config.options.axis?.y?.label,
                  text: config.options.axis?.y?.label || '',
                  color: config.theme?.textColor || '#333333',
                },
                grid: {
                  display: config.options.grid?.y !== false,
                  color: config.theme?.gridColor || '#e0e0e0',
                },
                ticks: {
                  color: config.theme?.textColor || '#333333',
                },
                min: config.options.axis?.y?.min,
                max: config.options.axis?.y?.max,
              },
            },
    }

    // Add stacked configuration if needed
    if (config.type === 'stackedBar' && baseOptions.scales) {
      baseOptions.scales.x = { ...baseOptions.scales.x, stacked: true }
      baseOptions.scales.y = { ...baseOptions.scales.y, stacked: true }
    }

    return baseOptions
  }

  // Handle chart export
  const handleExport = async (format: 'png' | 'svg' | 'csv' | 'json') => {
    setIsExporting(true)
    try {
      const result = await exportManager.exportChart(config, format)

      // Create download link
      const blob = new Blob([result.data], {
        type:
          format === 'json'
            ? 'application/json'
            : format === 'csv'
            ? 'text/csv'
            : format === 'svg'
            ? 'image/svg+xml'
            : 'text/plain',
      })

      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${config.title.replace(/\s+/g, '-').toLowerCase()}.${format}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      if (onExport) {
        onExport(format)
      }
    } catch (error) {
      console.error('Failed to export chart:', error)
    } finally {
      setIsExporting(false)
    }
  }

  // Render appropriate chart type
  const renderChart = () => {
    const data = getChartData()
    const options = getChartOptions()

    switch (config.type) {
      case 'line':
      case 'area':
        return <Line ref={chartRef} data={data as any} options={options} />

      case 'bar':
      case 'stackedBar':
        return <Bar ref={chartRef} data={data as any} options={options} />

      case 'pie':
        return <Pie ref={chartRef} data={data as any} options={options} />

      case 'donut':
        return <Doughnut ref={chartRef} data={data as any} options={options} />

      case 'scatter':
        return <Scatter ref={chartRef} data={data as any} options={options} />

      default:
        return (
          <div className="flex items-center justify-center h-full text-gray-500">
            Chart type "{config.type}" not yet implemented
          </div>
        )
    }
  }

  return (
    <div
      className="relative rounded-lg border bg-card"
      style={{
        backgroundColor: config.theme?.backgroundColor || '#ffffff',
        borderColor: config.theme?.gridColor || '#e0e0e0',
        width: config.options.width || '100%',
        height: config.options.height || 400,
        padding: '1rem',
      }}
    >
      {/* Export buttons */}
      {config.exportable && (
        <div className="absolute top-2 right-2 z-10 flex gap-1">
          <button
            onClick={() => handleExport('png')}
            disabled={isExporting}
            className="px-2 py-1 text-xs rounded bg-gray-100 hover:bg-gray-200 disabled:opacity-50"
            title="Export as PNG"
          >
            PNG
          </button>
          <button
            onClick={() => handleExport('csv')}
            disabled={isExporting}
            className="px-2 py-1 text-xs rounded bg-gray-100 hover:bg-gray-200 disabled:opacity-50"
            title="Export as CSV"
          >
            CSV
          </button>
          <button
            onClick={() => handleExport('json')}
            disabled={isExporting}
            className="px-2 py-1 text-xs rounded bg-gray-100 hover:bg-gray-200 disabled:opacity-50"
            title="Export as JSON"
          >
            JSON
          </button>
        </div>
      )}

      {/* Chart */}
      <div className="h-full">{renderChart()}</div>
    </div>
  )
}
