'use client'

import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

export function StorageChart() {
  const { data: backups } = useQuery({
    queryKey: ['backups'],
    queryFn: () => api.listBackups(),
  })

  // Group backups by date and calculate storage
  const chartData = backups?.backups
    ? Object.entries(
        backups.backups.reduce((acc: Record<string, number>, backup) => {
          const date = new Date(backup.created_at).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
          })
          acc[date] = (acc[date] || 0) + backup.size
          return acc
        }, {})
      )
        .slice(-7) // Last 7 days
        .map(([date, size]) => ({
          date,
          size: Math.round(size / (1024 * 1024)), // Convert to MB
        }))
    : []

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Storage Usage (Last 7 Days)</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis dataKey="date" stroke="#9ca3af" fontSize={12} />
          <YAxis stroke="#9ca3af" fontSize={12} label={{ value: 'MB', angle: -90, position: 'insideLeft' }} />
          <Tooltip
            contentStyle={{
              backgroundColor: '#fff',
              border: '1px solid #e5e7eb',
              borderRadius: '0.5rem',
            }}
          />
          <Bar dataKey="size" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
