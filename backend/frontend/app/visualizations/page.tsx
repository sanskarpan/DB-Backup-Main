'use client';

import React, { useState, useEffect } from 'react';
import {
  BarChart3,
  LineChart,
  PieChart,
  TrendingUp,
  TrendingDown,
  Download,
  Calendar,
  Filter,
  RefreshCw,
  Database,
  HardDrive,
  Clock,
  Activity,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Server,
  Zap,
  Eye,
  Settings,
  Share2,
} from 'lucide-react';

interface ChartData {
  label: string;
  value: number;
  color?: string;
}

interface TimeSeriesData {
  timestamp: Date;
  backups: number;
  storage: number;
  restores: number;
}

type TimeRange = '7d' | '30d' | '90d' | '1y';
type MetricType = 'backups' | 'storage' | 'restores' | 'success_rate';

export default function VisualizationsPage() {
  const [timeRange, setTimeRange] = useState<TimeRange>('30d');
  const [selectedMetrics, setSelectedMetrics] = useState<MetricType[]>(['backups', 'storage']);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);

  // Sample data
  const [timeSeriesData, setTimeSeriesData] = useState<TimeSeriesData[]>([]);
  const [databaseDistribution, setDatabaseDistribution] = useState<ChartData[]>([
    { label: 'PostgreSQL', value: 45, color: '#3B82F6' },
    { label: 'MySQL', value: 30, color: '#10B981' },
    { label: 'MongoDB', value: 15, color: '#F59E0B' },
    { label: 'Redis', value: 10, color: '#EF4444' },
  ]);

  const [backupTypeDistribution, setBackupTypeDistribution] = useState<ChartData[]>([
    { label: 'Full', value: 60, color: '#FF6B35' },
    { label: 'Incremental', value: 30, color: '#F7931E' },
    { label: 'Differential', value: 10, color: '#FFB88C' },
  ]);

  const [storageProviders, setStorageProviders] = useState<ChartData[]>([
    { label: 'AWS S3', value: 287, color: '#FF6B35' },
    { label: 'Azure', value: 412, color: '#3B82F6' },
    { label: 'GCP', value: 156, color: '#EF4444' },
  ]);

  useEffect(() => {
    generateTimeSeriesData();
  }, [timeRange]);

  useEffect(() => {
    if (autoRefresh) {
      const interval = setInterval(() => {
        generateTimeSeriesData();
      }, 30000); // Refresh every 30 seconds
      return () => clearInterval(interval);
    }
  }, [autoRefresh]);

  const generateTimeSeriesData = () => {
    setLoading(true);
    const days = timeRange === '7d' ? 7 : timeRange === '30d' ? 30 : timeRange === '90d' ? 90 : 365;
    const data: TimeSeriesData[] = [];

    for (let i = days - 1; i >= 0; i--) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      data.push({
        timestamp: date,
        backups: Math.floor(Math.random() * 20) + 10,
        storage: Math.floor(Math.random() * 50) + 100,
        restores: Math.floor(Math.random() * 5),
      });
    }

    setTimeSeriesData(data);
    setTimeout(() => setLoading(false), 500);
  };

  const toggleMetric = (metric: MetricType) => {
    if (selectedMetrics.includes(metric)) {
      setSelectedMetrics(selectedMetrics.filter((m) => m !== metric));
    } else {
      setSelectedMetrics([...selectedMetrics, metric]);
    }
  };

  const handleExport = (format: 'csv' | 'json' | 'pdf') => {
    // Export functionality would go here
    console.log(`Exporting data as ${format}`);
  };

  const stats = [
    {
      label: 'Total Backups',
      value: '1,234',
      change: '+12.5%',
      trend: 'up',
      icon: Database,
    },
    {
      label: 'Total Storage',
      value: '855 GB',
      change: '+8.3%',
      trend: 'up',
      icon: HardDrive,
    },
    {
      label: 'Avg Backup Time',
      value: '14m',
      change: '-5.2%',
      trend: 'down',
      icon: Clock,
    },
    {
      label: 'Success Rate',
      value: '98.7%',
      change: '+0.3%',
      trend: 'up',
      icon: CheckCircle2,
    },
  ];

  const maxBackups = Math.max(...timeSeriesData.map((d) => d.backups), 1);
  const maxStorage = Math.max(...timeSeriesData.map((d) => d.storage), 1);
  const maxRestores = Math.max(...timeSeriesData.map((d) => d.restores), 1);

  return (
    <div className="min-h-screen bg-[var(--background)]">
      {/* Hero Section */}
      <div className="gradient-hero relative overflow-hidden py-16 px-6">
        <div className="absolute inset-0 opacity-10">
          <div className="absolute top-20 left-10 w-72 h-72 bg-white rounded-full blur-3xl"></div>
          <div className="absolute bottom-20 right-10 w-96 h-96 bg-white rounded-full blur-3xl"></div>
        </div>

        <div className="max-w-7xl mx-auto relative z-10">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-white/20 backdrop-blur-sm rounded-xl">
              <BarChart3 className="w-8 h-8 text-white" />
            </div>
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">Analytics & Visualizations</h1>
              <p className="text-white/90 text-lg">Monitor backup performance and storage trends</p>
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-8">
            {stats.map((stat, index) => (
              <div key={index} className="glass-card p-6">
                <div className="flex items-center justify-between mb-3">
                  <stat.icon className="w-8 h-8 text-white/50" />
                  <div className={`flex items-center gap-1 text-sm ${
                    stat.trend === 'up' ? 'text-green-300' : 'text-red-300'
                  }`}>
                    {stat.trend === 'up' ? (
                      <TrendingUp className="w-4 h-4" />
                    ) : (
                      <TrendingDown className="w-4 h-4" />
                    )}
                    <span>{stat.change}</span>
                  </div>
                </div>
                <p className="text-white/70 text-sm mb-1">{stat.label}</p>
                <p className="text-3xl font-bold text-white">{stat.value}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Controls */}
        <div className="modern-card mb-8">
          <div className="flex flex-col lg:flex-row gap-4 items-start lg:items-center justify-between">
            {/* Time Range Selector */}
            <div className="flex items-center gap-3">
              <Calendar className="w-5 h-5 text-[var(--text-secondary)]" />
              <div className="flex gap-2">
                {[
                  { label: '7 Days', value: '7d' as TimeRange },
                  { label: '30 Days', value: '30d' as TimeRange },
                  { label: '90 Days', value: '90d' as TimeRange },
                  { label: '1 Year', value: '1y' as TimeRange },
                ].map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setTimeRange(range.value)}
                    className={`px-4 py-2 rounded-lg font-medium transition-all ${
                      timeRange === range.value
                        ? 'bg-gradient-orange text-white'
                        : 'bg-[var(--surface)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => setAutoRefresh(!autoRefresh)}
                className={`btn-ghost flex items-center gap-2 ${autoRefresh ? 'text-[var(--orange-primary)]' : ''}`}
              >
                <RefreshCw className={`w-5 h-5 ${autoRefresh ? 'animate-spin' : ''}`} />
                Auto Refresh
              </button>

              <div className="relative group">
                <button className="btn-secondary flex items-center gap-2">
                  <Download className="w-5 h-5" />
                  Export
                </button>
                <div className="absolute right-0 mt-2 w-48 bg-[var(--surface-elevated)] border border-[var(--border)] rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-10">
                  <button
                    onClick={() => handleExport('csv')}
                    className="w-full px-4 py-2 text-left hover:bg-[var(--surface)] text-[var(--text-primary)] transition-colors"
                  >
                    Export as CSV
                  </button>
                  <button
                    onClick={() => handleExport('json')}
                    className="w-full px-4 py-2 text-left hover:bg-[var(--surface)] text-[var(--text-primary)] transition-colors"
                  >
                    Export as JSON
                  </button>
                  <button
                    onClick={() => handleExport('pdf')}
                    className="w-full px-4 py-2 text-left hover:bg-[var(--surface)] text-[var(--text-primary)] transition-colors"
                  >
                    Export as PDF
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Metric Toggles */}
          <div className="flex flex-wrap gap-2 mt-4 pt-4 border-t border-[var(--border)]">
            <span className="text-sm text-[var(--text-secondary)] mr-2">Show Metrics:</span>
            {[
              { label: 'Backups', value: 'backups' as MetricType, color: '#FF6B35' },
              { label: 'Storage', value: 'storage' as MetricType, color: '#3B82F6' },
              { label: 'Restores', value: 'restores' as MetricType, color: '#10B981' },
              { label: 'Success Rate', value: 'success_rate' as MetricType, color: '#F59E0B' },
            ].map((metric) => (
              <button
                key={metric.value}
                onClick={() => toggleMetric(metric.value)}
                className={`px-3 py-1 rounded-lg text-sm font-medium transition-all ${
                  selectedMetrics.includes(metric.value)
                    ? 'bg-gradient-orange text-white'
                    : 'bg-[var(--surface)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                }`}
              >
                {metric.label}
              </button>
            ))}
          </div>
        </div>

        {/* Main Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          {/* Backup Activity Chart */}
          <div className="modern-card">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-[var(--text-primary)] flex items-center gap-2">
                <Activity className="w-5 h-5" />
                Backup Activity
              </h3>
              <button className="btn-ghost p-2">
                <Settings className="w-5 h-5" />
              </button>
            </div>

            {loading ? (
              <div className="skeleton h-64 rounded-lg"></div>
            ) : (
              <div className="relative h-64">
                {/* Simple Bar Chart */}
                <div className="flex items-end justify-between h-full gap-1 pb-8">
                  {timeSeriesData.map((data, index) => (
                    <div
                      key={index}
                      className="flex-1 relative group"
                    >
                      {selectedMetrics.includes('backups') && (
                        <div
                          className="bg-gradient-orange rounded-t transition-all hover:opacity-80 cursor-pointer"
                          style={{
                            height: `${(data.backups / maxBackups) * 100}%`,
                            minHeight: '4px',
                          }}
                        >
                          <div className="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 bg-[var(--gray-900)] text-white text-xs px-2 py-1 rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                            {data.timestamp.toLocaleDateString()}<br />
                            {data.backups} backups
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
                <div className="absolute bottom-0 left-0 right-0 flex justify-between text-xs text-[var(--text-tertiary)] border-t border-[var(--border)] pt-2">
                  <span>{timeSeriesData[0]?.timestamp.toLocaleDateString()}</span>
                  <span>{timeSeriesData[timeSeriesData.length - 1]?.timestamp.toLocaleDateString()}</span>
                </div>
              </div>
            )}
          </div>

          {/* Storage Growth Chart */}
          <div className="modern-card">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-[var(--text-primary)] flex items-center gap-2">
                <TrendingUp className="w-5 h-5" />
                Storage Growth
              </h3>
              <button className="btn-ghost p-2">
                <Settings className="w-5 h-5" />
              </button>
            </div>

            {loading ? (
              <div className="skeleton h-64 rounded-lg"></div>
            ) : (
              <div className="relative h-64">
                {/* Simple Line Chart */}
                <svg className="w-full h-full" viewBox="0 0 400 200" preserveAspectRatio="none">
                  <defs>
                    <linearGradient id="storageGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                      <stop offset="0%" stopColor="#3B82F6" stopOpacity="0.3" />
                      <stop offset="100%" stopColor="#3B82F6" stopOpacity="0.05" />
                    </linearGradient>
                  </defs>

                  {/* Grid lines */}
                  {[0, 1, 2, 3, 4].map((i) => (
                    <line
                      key={i}
                      x1="0"
                      y1={i * 50}
                      x2="400"
                      y2={i * 50}
                      stroke="var(--border)"
                      strokeWidth="1"
                    />
                  ))}

                  {/* Area fill */}
                  {selectedMetrics.includes('storage') && (
                    <>
                      <path
                        d={`M 0,${200 - (timeSeriesData[0]?.storage / maxStorage) * 180} ${timeSeriesData
                          .map(
                            (d, i) =>
                              `L ${(i / (timeSeriesData.length - 1)) * 400},${
                                200 - (d.storage / maxStorage) * 180
                              }`
                          )
                          .join(' ')} L 400,200 L 0,200 Z`}
                        fill="url(#storageGradient)"
                      />
                      <path
                        d={`M 0,${200 - (timeSeriesData[0]?.storage / maxStorage) * 180} ${timeSeriesData
                          .map(
                            (d, i) =>
                              `L ${(i / (timeSeriesData.length - 1)) * 400},${
                                200 - (d.storage / maxStorage) * 180
                              }`
                          )
                          .join(' ')}`}
                        fill="none"
                        stroke="#3B82F6"
                        strokeWidth="3"
                      />
                    </>
                  )}
                </svg>
              </div>
            )}
          </div>
        </div>

        {/* Distribution Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
          {/* Database Distribution */}
          <div className="modern-card">
            <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-6 flex items-center gap-2">
              <Database className="w-5 h-5" />
              Database Types
            </h3>

            <div className="space-y-4">
              {databaseDistribution.map((item, index) => (
                <div key={index}>
                  <div className="flex items-center justify-between text-sm mb-2">
                    <div className="flex items-center gap-2">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: item.color }}
                      ></div>
                      <span className="text-[var(--text-primary)] font-medium">{item.label}</span>
                    </div>
                    <span className="text-[var(--text-secondary)]">{item.value}%</span>
                  </div>
                  <div className="progress-bar">
                    <div
                      className="h-full rounded-lg transition-all"
                      style={{
                        width: `${item.value}%`,
                        backgroundColor: item.color,
                      }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Backup Type Distribution */}
          <div className="modern-card">
            <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-6 flex items-center gap-2">
              <PieChart className="w-5 h-5" />
              Backup Types
            </h3>

            <div className="space-y-4">
              {backupTypeDistribution.map((item, index) => (
                <div key={index}>
                  <div className="flex items-center justify-between text-sm mb-2">
                    <div className="flex items-center gap-2">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: item.color }}
                      ></div>
                      <span className="text-[var(--text-primary)] font-medium">{item.label}</span>
                    </div>
                    <span className="text-[var(--text-secondary)]">{item.value}%</span>
                  </div>
                  <div className="progress-bar">
                    <div
                      className="h-full rounded-lg transition-all"
                      style={{
                        width: `${item.value}%`,
                        backgroundColor: item.color,
                      }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Storage Provider Usage */}
          <div className="modern-card">
            <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-6 flex items-center gap-2">
              <Server className="w-5 h-5" />
              Storage by Provider
            </h3>

            <div className="space-y-4">
              {storageProviders.map((item, index) => {
                const total = storageProviders.reduce((sum, p) => sum + p.value, 0);
                const percentage = Math.round((item.value / total) * 100);
                return (
                  <div key={index}>
                    <div className="flex items-center justify-between text-sm mb-2">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: item.color }}
                        ></div>
                        <span className="text-[var(--text-primary)] font-medium">{item.label}</span>
                      </div>
                      <span className="text-[var(--text-secondary)]">{item.value} GB</span>
                    </div>
                    <div className="progress-bar">
                      <div
                        className="h-full rounded-lg transition-all"
                        style={{
                          width: `${percentage}%`,
                          backgroundColor: item.color,
                        }}
                      ></div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Recent Activity Table */}
        <div className="modern-card">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-6 flex items-center gap-2">
            <Activity className="w-5 h-5" />
            Recent Activity Summary
          </h3>

          <div className="overflow-x-auto">
            <table className="table-modern">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Backups</th>
                  <th>Storage (GB)</th>
                  <th>Restores</th>
                  <th>Success Rate</th>
                </tr>
              </thead>
              <tbody>
                {timeSeriesData.slice(-7).reverse().map((data, index) => (
                  <tr key={index}>
                    <td className="font-medium">{data.timestamp.toLocaleDateString()}</td>
                    <td>{data.backups}</td>
                    <td>{data.storage}</td>
                    <td>{data.restores}</td>
                    <td>
                      <span className="badge badge-success">
                        {Math.floor(Math.random() * 5) + 95}%
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Info Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-8">
          <div className="stats-card">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                <Zap className="w-5 h-5 text-[var(--orange-primary)]" />
              </div>
              <div>
                <h4 className="font-semibold text-[var(--text-primary)] mb-1">Real-time Insights</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  Monitor your backup operations in real-time with auto-refresh capabilities and live data updates.
                </p>
              </div>
            </div>
          </div>

          <div className="stats-card">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                <Share2 className="w-5 h-5 text-[var(--orange-primary)]" />
              </div>
              <div>
                <h4 className="font-semibold text-[var(--text-primary)] mb-1">Export & Share</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  Export your analytics data in multiple formats and share insights with your team.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
