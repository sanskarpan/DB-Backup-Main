'use client';

import React, { useState, useEffect } from 'react';
import {
  Clock,
  Database,
  Server,
  Play,
  CheckCircle2,
  AlertCircle,
  History,
  Search,
  Filter,
  Download,
  RotateCcw,
  Calendar,
  FileText,
  Zap,
  ArrowRight,
  ChevronRight,
  Info,
} from 'lucide-react';

interface Backup {
  id: string;
  name: string;
  database: string;
  size: string;
  timestamp: Date;
  type: 'full' | 'incremental' | 'differential';
  status: 'completed' | 'failed' | 'in_progress';
  retention: string;
}

interface RestoreJob {
  id: string;
  backupId: string;
  backupName: string;
  targetDatabase: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  startTime: Date;
  endTime?: Date;
  pointInTime?: Date;
}

export default function RestorePage() {
  const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null);
  const [selectedPointInTime, setSelectedPointInTime] = useState<Date | null>(null);
  const [targetDatabase, setTargetDatabase] = useState('');
  const [restoreType, setRestoreType] = useState<'full' | 'pitr'>('full');
  const [isRestoring, setIsRestoring] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'full' | 'incremental' | 'differential'>('all');
  const [loading, setLoading] = useState(true);
  const [showHistory, setShowHistory] = useState(false);

  const [backups, setBackups] = useState<Backup[]>([]);
  const [restoreHistory, setRestoreHistory] = useState<RestoreJob[]>([]);
  const [currentRestore, setCurrentRestore] = useState<RestoreJob | null>(null);

  useEffect(() => {
    // Simulate loading data
    setTimeout(() => {
      setBackups([
        {
          id: '1',
          name: 'production-db-full-backup',
          database: 'PostgreSQL - production',
          size: '2.4 GB',
          timestamp: new Date('2024-01-22T02:00:00'),
          type: 'full',
          status: 'completed',
          retention: '30 days',
        },
        {
          id: '2',
          name: 'production-db-incremental',
          database: 'PostgreSQL - production',
          size: '156 MB',
          timestamp: new Date('2024-01-22T14:00:00'),
          type: 'incremental',
          status: 'completed',
          retention: '7 days',
        },
        {
          id: '3',
          name: 'staging-db-full-backup',
          database: 'MySQL - staging',
          size: '1.8 GB',
          timestamp: new Date('2024-01-21T02:00:00'),
          type: 'full',
          status: 'completed',
          retention: '14 days',
        },
        {
          id: '4',
          name: 'analytics-db-differential',
          database: 'MongoDB - analytics',
          size: '432 MB',
          timestamp: new Date('2024-01-22T08:00:00'),
          type: 'differential',
          status: 'completed',
          retention: '7 days',
        },
      ]);

      setRestoreHistory([
        {
          id: 'r1',
          backupId: '1',
          backupName: 'production-db-full-backup',
          targetDatabase: 'PostgreSQL - production-restore',
          status: 'completed',
          progress: 100,
          startTime: new Date('2024-01-20T10:00:00'),
          endTime: new Date('2024-01-20T10:45:00'),
        },
        {
          id: 'r2',
          backupId: '3',
          backupName: 'staging-db-full-backup',
          targetDatabase: 'MySQL - staging-restore',
          status: 'completed',
          progress: 100,
          startTime: new Date('2024-01-19T15:30:00'),
          endTime: new Date('2024-01-19T16:00:00'),
        },
        {
          id: 'r3',
          backupId: '2',
          backupName: 'production-db-incremental',
          targetDatabase: 'PostgreSQL - test',
          status: 'failed',
          progress: 45,
          startTime: new Date('2024-01-18T09:00:00'),
          endTime: new Date('2024-01-18T09:15:00'),
        },
      ]);

      setLoading(false);
    }, 1000);
  }, []);

  const filteredBackups = backups.filter((backup) => {
    const matchesSearch = backup.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         backup.database.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = filterType === 'all' || backup.type === filterType;
    return matchesSearch && matchesFilter;
  });

  const handleRestore = () => {
    if (!selectedBackup || !targetDatabase) return;

    const newRestore: RestoreJob = {
      id: `r${Date.now()}`,
      backupId: selectedBackup.id,
      backupName: selectedBackup.name,
      targetDatabase,
      status: 'running',
      progress: 0,
      startTime: new Date(),
      pointInTime: restoreType === 'pitr' ? selectedPointInTime || undefined : undefined,
    };

    setCurrentRestore(newRestore);
    setIsRestoring(true);

    // Simulate restore progress
    let progress = 0;
    const interval = setInterval(() => {
      progress += Math.random() * 15;
      if (progress >= 100) {
        progress = 100;
        clearInterval(interval);
        setCurrentRestore((prev) => prev ? { ...prev, status: 'completed', progress: 100, endTime: new Date() } : null);
        setIsRestoring(false);
        setRestoreHistory((prev) => [{ ...newRestore, status: 'completed', progress: 100, endTime: new Date() }, ...prev]);
      } else {
        setCurrentRestore((prev) => prev ? { ...prev, progress } : null);
      }
    }, 500);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'badge-success';
      case 'running':
      case 'in_progress':
        return 'badge-info';
      case 'failed':
        return 'badge-error';
      default:
        return 'badge-warning';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'full':
        return 'badge-orange';
      case 'incremental':
        return 'badge-info';
      case 'differential':
        return 'badge-warning';
      default:
        return 'badge-success';
    }
  };

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
              <RotateCcw className="w-8 h-8 text-white" />
            </div>
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">Database Restore</h1>
              <p className="text-white/90 text-lg">Restore your databases from backups with point-in-time recovery</p>
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-8">
            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Available Backups</p>
                  <p className="text-3xl font-bold text-white">{backups.length}</p>
                </div>
                <Database className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Successful Restores</p>
                  <p className="text-3xl font-bold text-white">
                    {restoreHistory.filter(r => r.status === 'completed').length}
                  </p>
                </div>
                <CheckCircle2 className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Total Data</p>
                  <p className="text-3xl font-bold text-white">4.8 GB</p>
                </div>
                <Server className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Avg Restore Time</p>
                  <p className="text-3xl font-bold text-white">32m</p>
                </div>
                <Clock className="w-8 h-8 text-white/50" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Active Restore Progress */}
        {currentRestore && (
          <div className="modern-card mb-8 border-l-4 border-l-[var(--orange-primary)] animate-fade-in">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-1">
                  Restore in Progress
                </h3>
                <p className="text-sm text-[var(--text-secondary)]">
                  {currentRestore.backupName} → {currentRestore.targetDatabase}
                </p>
              </div>
              <span className={`badge ${getStatusColor(currentRestore.status)}`}>
                {currentRestore.status}
              </span>
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-[var(--text-secondary)]">Progress</span>
                <span className="font-semibold text-[var(--text-primary)]">
                  {Math.round(currentRestore.progress)}%
                </span>
              </div>
              <div className="progress-bar">
                <div
                  className="progress-fill"
                  style={{ width: `${currentRestore.progress}%` }}
                ></div>
              </div>

              {currentRestore.pointInTime && (
                <div className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                  <Clock className="w-4 h-4" />
                  <span>Point-in-time: {currentRestore.pointInTime.toLocaleString()}</span>
                </div>
              )}
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Left Column - Backup Selection */}
          <div className="lg:col-span-2 space-y-6">
            {/* Search and Filter */}
            <div className="modern-card">
              <div className="flex flex-col sm:flex-row gap-4 mb-6">
                <div className="flex-1 relative">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-[var(--text-tertiary)]" />
                  <input
                    type="text"
                    placeholder="Search backups..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="input-modern pl-10"
                  />
                </div>
                <div className="flex gap-2">
                  <select
                    value={filterType}
                    onChange={(e) => setFilterType(e.target.value as any)}
                    className="input-modern"
                  >
                    <option value="all">All Types</option>
                    <option value="full">Full Backup</option>
                    <option value="incremental">Incremental</option>
                    <option value="differential">Differential</option>
                  </select>
                  <button
                    onClick={() => setShowHistory(!showHistory)}
                    className="btn-secondary flex items-center gap-2"
                  >
                    <History className="w-4 h-4" />
                    History
                  </button>
                </div>
              </div>

              {/* Backup List */}
              <div className="space-y-3">
                {loading ? (
                  Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="skeleton h-24 rounded-lg"></div>
                  ))
                ) : filteredBackups.length === 0 ? (
                  <div className="empty-state py-12">
                    <Database className="empty-state-icon" />
                    <h3 className="empty-state-title">No backups found</h3>
                    <p className="empty-state-description">
                      Try adjusting your search or filter criteria
                    </p>
                  </div>
                ) : (
                  filteredBackups.map((backup) => (
                    <div
                      key={backup.id}
                      onClick={() => setSelectedBackup(backup)}
                      className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                        selectedBackup?.id === backup.id
                          ? 'border-[var(--orange-primary)] bg-[var(--surface)]'
                          : 'border-[var(--border)] hover:border-[var(--orange-light)]'
                      }`}
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex-1">
                          <h4 className="font-semibold text-[var(--text-primary)] mb-1">
                            {backup.name}
                          </h4>
                          <p className="text-sm text-[var(--text-secondary)] flex items-center gap-2">
                            <Database className="w-4 h-4" />
                            {backup.database}
                          </p>
                        </div>
                        <span className={`badge ${getTypeColor(backup.type)}`}>
                          {backup.type}
                        </span>
                      </div>

                      <div className="grid grid-cols-3 gap-4 text-sm">
                        <div>
                          <p className="text-[var(--text-tertiary)] text-xs mb-1">Size</p>
                          <p className="font-medium text-[var(--text-primary)]">{backup.size}</p>
                        </div>
                        <div>
                          <p className="text-[var(--text-tertiary)] text-xs mb-1">Date</p>
                          <p className="font-medium text-[var(--text-primary)]">
                            {backup.timestamp.toLocaleDateString()}
                          </p>
                        </div>
                        <div>
                          <p className="text-[var(--text-tertiary)] text-xs mb-1">Status</p>
                          <span className={`badge ${getStatusColor(backup.status)}`}>
                            {backup.status}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Restore History */}
            {showHistory && (
              <div className="modern-card animate-fade-in">
                <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-4 flex items-center gap-2">
                  <History className="w-5 h-5" />
                  Restore History
                </h3>
                <div className="space-y-3">
                  {restoreHistory.map((restore) => (
                    <div key={restore.id} className="p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)]">
                      <div className="flex items-start justify-between mb-2">
                        <div>
                          <h4 className="font-medium text-[var(--text-primary)]">{restore.backupName}</h4>
                          <p className="text-sm text-[var(--text-secondary)]">{restore.targetDatabase}</p>
                        </div>
                        <span className={`badge ${getStatusColor(restore.status)}`}>
                          {restore.status}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-xs text-[var(--text-secondary)]">
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {restore.startTime.toLocaleString()}
                        </span>
                        {restore.endTime && (
                          <span>
                            Duration: {Math.round((restore.endTime.getTime() - restore.startTime.getTime()) / 60000)}m
                          </span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Right Column - Restore Configuration */}
          <div className="space-y-6">
            {/* Restore Type Selection */}
            <div className="modern-card">
              <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-4">
                Restore Configuration
              </h3>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                    Restore Type
                  </label>
                  <div className="grid grid-cols-2 gap-3">
                    <button
                      onClick={() => setRestoreType('full')}
                      className={`p-4 rounded-lg border-2 transition-all ${
                        restoreType === 'full'
                          ? 'border-[var(--orange-primary)] bg-[var(--surface)]'
                          : 'border-[var(--border)]'
                      }`}
                    >
                      <Database className="w-6 h-6 mb-2 mx-auto text-[var(--orange-primary)]" />
                      <p className="text-sm font-medium text-[var(--text-primary)]">Full Restore</p>
                    </button>
                    <button
                      onClick={() => setRestoreType('pitr')}
                      className={`p-4 rounded-lg border-2 transition-all ${
                        restoreType === 'pitr'
                          ? 'border-[var(--orange-primary)] bg-[var(--surface)]'
                          : 'border-[var(--border)]'
                      }`}
                    >
                      <Clock className="w-6 h-6 mb-2 mx-auto text-[var(--orange-primary)]" />
                      <p className="text-sm font-medium text-[var(--text-primary)]">Point-in-Time</p>
                    </button>
                  </div>
                </div>

                {restoreType === 'pitr' && (
                  <div className="animate-fade-in">
                    <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                      Select Point in Time
                    </label>
                    <input
                      type="datetime-local"
                      onChange={(e) => setSelectedPointInTime(new Date(e.target.value))}
                      className="input-modern"
                    />
                    <p className="text-xs text-[var(--text-secondary)] mt-1 flex items-start gap-1">
                      <Info className="w-3 h-3 mt-0.5 flex-shrink-0" />
                      <span>Restore database to a specific point in time using transaction logs</span>
                    </p>
                  </div>
                )}

                <div>
                  <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                    Target Database
                  </label>
                  <input
                    type="text"
                    placeholder="e.g., production-restore"
                    value={targetDatabase}
                    onChange={(e) => setTargetDatabase(e.target.value)}
                    className="input-modern"
                  />
                </div>

                {selectedBackup && (
                  <div className="p-4 bg-[var(--surface)] rounded-lg border border-[var(--border)] animate-fade-in">
                    <h4 className="text-sm font-semibold text-[var(--text-primary)] mb-3">
                      Selected Backup
                    </h4>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-[var(--text-secondary)]">Name:</span>
                        <span className="font-medium text-[var(--text-primary)]">{selectedBackup.name}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-[var(--text-secondary)]">Size:</span>
                        <span className="font-medium text-[var(--text-primary)]">{selectedBackup.size}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-[var(--text-secondary)]">Type:</span>
                        <span className={`badge ${getTypeColor(selectedBackup.type)}`}>
                          {selectedBackup.type}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-[var(--text-secondary)]">Date:</span>
                        <span className="font-medium text-[var(--text-primary)]">
                          {selectedBackup.timestamp.toLocaleString()}
                        </span>
                      </div>
                    </div>
                  </div>
                )}

                <button
                  onClick={handleRestore}
                  disabled={!selectedBackup || !targetDatabase || isRestoring}
                  className="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isRestoring ? (
                    <>
                      <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      Restoring...
                    </>
                  ) : (
                    <>
                      <Play className="w-5 h-5" />
                      Start Restore
                    </>
                  )}
                </button>
              </div>
            </div>

            {/* Quick Info */}
            <div className="stats-card">
              <div className="flex items-start gap-3">
                <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                  <Zap className="w-5 h-5 text-[var(--orange-primary)]" />
                </div>
                <div>
                  <h4 className="font-semibold text-[var(--text-primary)] mb-1">Quick Restore</h4>
                  <p className="text-sm text-[var(--text-secondary)]">
                    Most restores complete in under 30 minutes with our optimized recovery process.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
