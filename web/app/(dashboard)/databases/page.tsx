'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, Database } from '@/lib/api'
import { useState } from 'react'
import {
  Plus,
  Trash2,
  Edit,
  TestTube2,
  CheckCircle,
  XCircle,
  Database as DatabaseIcon,
  Server,
  Zap,
  Shield,
  Activity,
  Loader2,
  AlertCircle,
} from 'lucide-react'

export default function DatabasesPage() {
  const [showModal, setShowModal] = useState(false)
  const [editingDatabase, setEditingDatabase] = useState<Database | null>(null)
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({})

  const queryClient = useQueryClient()

  const [mutationError, setMutationError] = useState<string | null>(null)

  const { data: databases, isLoading, isError, error } = useQuery({
    queryKey: ['databases'],
    queryFn: () => api.listDatabases(),
  })

  const createMutation = useMutation({
    mutationFn: (data: Partial<Database>) => api.createDatabase(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['databases'] })
      setShowModal(false)
      setEditingDatabase(null)
      setMutationError(null)
    },
    onError: (err: unknown) => {
      setMutationError(err instanceof Error ? err.message : 'Failed to create database')
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Database> }) =>
      api.updateDatabase(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['databases'] })
      setShowModal(false)
      setEditingDatabase(null)
      setMutationError(null)
    },
    onError: (err: unknown) => {
      setMutationError(err instanceof Error ? err.message : 'Failed to update database')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteDatabase(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['databases'] })
      setMutationError(null)
    },
    onError: (err: unknown) => {
      setMutationError(err instanceof Error ? err.message : 'Failed to delete database')
    },
  })

  const testConnectionMutation = useMutation({
    mutationFn: (id: string) => api.testConnection(id),
    onSuccess: (data, id) => {
      setTestResults((prev) => ({ ...prev, [id]: data }))
      setTimeout(() => {
        setTestResults((prev) => {
          const newResults = { ...prev }
          delete newResults[id]
          return newResults
        })
      }, 5000)
    },
    onError: (err: unknown, id) => {
      setTestResults((prev) => ({
        ...prev,
        [id]: { success: false, message: err instanceof Error ? err.message : 'Connection test failed' },
      }))
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data = {
      name: formData.get('name') as string,
      type: formData.get('type') as Database['type'],
      host: formData.get('host') as string,
      port: parseInt(formData.get('port') as string),
      username: formData.get('username') as string,
      password: formData.get('password') as string,
    }

    if (editingDatabase) {
      updateMutation.mutate({ id: editingDatabase.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const openEditModal = (database: Database) => {
    setEditingDatabase(database)
    setShowModal(true)
  }

  const openCreateModal = () => {
    setEditingDatabase(null)
    setShowModal(true)
  }

  const getDBIcon = (type: string) => {
    const iconMap: Record<string, string> = {
      postgres: '🐘',
      mysql: '🐬',
      mongodb: '🍃',
      redis: '📦',
      cassandra: '🌌',
      elasticsearch: '🔍',
      timescaledb: '⏱️',
      influxdb: '📈',
      dynamodb: '⚡',
    }
    return iconMap[type] || '💾'
  }

  const getDBColor = (type: string) => {
    const colorMap: Record<string, string> = {
      postgres: 'from-blue-500 to-blue-600',
      mysql: 'from-orange-500 to-orange-600',
      mongodb: 'from-green-500 to-green-600',
      redis: 'from-red-500 to-red-600',
      cassandra: 'from-purple-500 to-purple-600',
      elasticsearch: 'from-yellow-500 to-yellow-600',
      timescaledb: 'from-cyan-500 to-cyan-600',
      influxdb: 'from-pink-500 to-pink-600',
      dynamodb: 'from-indigo-500 to-indigo-600',
    }
    return colorMap[type] || 'from-gray-500 to-gray-600'
  }

  const stats = {
    total: databases?.length || 0,
    sql: databases?.filter(db => ['postgres', 'mysql', 'timescaledb'].includes(db.type)).length || 0,
    nosql: databases?.filter(db => ['mongodb', 'redis', 'cassandra', 'dynamodb'].includes(db.type)).length || 0,
    other: databases?.filter(db => ['elasticsearch', 'opensearch', 'influxdb'].includes(db.type)).length || 0,
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Hero Section */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-blue-500 via-blue-600 to-purple-600 p-8 text-white shadow-2xl">
          <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-20"></div>
          <div className="relative z-10">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <DatabaseIcon className="w-10 h-10" />
                  <h1 className="text-4xl font-bold">Database Connections</h1>
                </div>
                <p className="text-blue-100 text-lg max-w-2xl">
                  Configure and manage connections to all your databases across multiple platforms
                </p>
              </div>
              <button
                onClick={openCreateModal}
                className="btn-primary bg-white text-blue-600 hover:bg-blue-50 shadow-xl"
              >
                <Plus className="w-5 h-5 mr-2" />
                Add Database
              </button>
            </div>
          </div>
        </div>

        {/* Error Alerts */}
        {(isError || mutationError) && (
          <div className="p-4 rounded-xl bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-semibold text-red-800 dark:text-red-300">
                {isError ? 'Failed to load databases' : 'Action failed'}
              </p>
              <p className="text-sm text-red-700 dark:text-red-400">
                {isError ? (error instanceof Error ? error.message : 'An error occurred') : mutationError}
              </p>
            </div>
          </div>
        )}

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-lg">
                <DatabaseIcon className="w-6 h-6" />
              </div>
              <Activity className="w-5 h-5 text-blue-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.total}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Total Databases</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-emerald-500 to-emerald-600 text-white shadow-lg">
                <Server className="w-6 h-6" />
              </div>
              <span className="text-2xl">🐘</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.sql}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">SQL Databases</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-lg">
                <Zap className="w-6 h-6" />
              </div>
              <span className="text-2xl">🍃</span>
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.nosql}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">NoSQL Databases</div>
          </div>

          <div className="stats-card bg-white dark:bg-gray-900 dark:border-gray-800">
            <div className="flex items-center justify-between mb-4">
              <div className="p-3 rounded-xl bg-gradient-to-br from-amber-500 to-amber-600 text-white shadow-lg">
                <Shield className="w-6 h-6" />
              </div>
              <CheckCircle className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-3xl font-bold text-gray-900 dark:text-white mb-1">{stats.other}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Other</div>
          </div>
        </div>

        {/* Databases Grid */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="modern-card modern-card-dark p-6 animate-pulse">
                <div className="flex items-start justify-between mb-4">
                  <div className="space-y-3 flex-1">
                    <div className="h-6 w-32 bg-gray-200 dark:bg-gray-700 rounded"></div>
                    <div className="h-5 w-20 bg-gray-200 dark:bg-gray-700 rounded"></div>
                  </div>
                  <div className="flex gap-2">
                    <div className="h-8 w-8 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                    <div className="h-8 w-8 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
                  </div>
                </div>
                <div className="space-y-2 mb-4">
                  <div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded"></div>
                  <div className="h-4 w-3/4 bg-gray-200 dark:bg-gray-700 rounded"></div>
                </div>
                <div className="h-10 bg-gray-200 dark:bg-gray-700 rounded-lg"></div>
              </div>
            ))}
          </div>
        ) : databases && databases.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {databases.map((database) => (
              <div
                key={database.id}
                className="modern-card modern-card-dark p-6 group hover:shadow-2xl transition-all duration-300"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className={`p-3 rounded-xl bg-gradient-to-br ${getDBColor(database.type)} text-white shadow-lg text-2xl`}>
                      {getDBIcon(database.type)}
                    </div>
                    <div>
                      <h3 className="text-lg font-bold text-gray-900 dark:text-white group-hover:text-orange-600 dark:group-hover:text-orange-400 transition-colors">
                        {database.name}
                      </h3>
                      <span className="inline-block mt-1 px-2.5 py-1 text-xs font-semibold bg-gradient-to-r from-blue-100 to-purple-100 dark:from-blue-950 dark:to-purple-950 text-blue-800 dark:text-blue-300 rounded-lg border border-blue-200 dark:border-blue-800">
                        {database.type.toUpperCase()}
                      </span>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => openEditModal(database)}
                      className="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-all hover:scale-110"
                      title="Edit"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Are you sure you want to delete this database?')) {
                          deleteMutation.mutate(database.id)
                        }
                      }}
                      className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950 rounded-lg transition-all hover:scale-110"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                <div className="space-y-3 mb-4 bg-gray-50 dark:bg-gray-800 rounded-xl p-4">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400 flex items-center gap-2">
                      <Server className="w-4 h-4" />
                      Host
                    </span>
                    <span className="font-mono font-medium text-gray-900 dark:text-white">{database.host}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400 flex items-center gap-2">
                      <Zap className="w-4 h-4" />
                      Port
                    </span>
                    <span className="font-mono font-medium text-gray-900 dark:text-white">{database.port}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400 flex items-center gap-2">
                      <Shield className="w-4 h-4" />
                      User
                    </span>
                    <span className="font-mono font-medium text-gray-900 dark:text-white">{database.username}</span>
                  </div>
                </div>

                <button
                  onClick={() => testConnectionMutation.mutate(database.id)}
                  disabled={testConnectionMutation.isPending}
                  className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-gradient-to-r from-gray-100 to-gray-200 dark:from-gray-800 dark:to-gray-700 text-gray-700 dark:text-gray-300 rounded-xl hover:from-orange-500 hover:to-orange-600 hover:text-white transition-all font-medium shadow-sm hover:shadow-lg disabled:opacity-50"
                >
                  <TestTube2 className="w-4 h-4" />
                  Test Connection
                </button>

                {testResults[database.id] && (
                  <div
                    className={`mt-3 p-4 rounded-xl flex items-start gap-3 border ${
                      testResults[database.id].success
                        ? 'bg-emerald-50 dark:bg-emerald-950 text-emerald-800 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800'
                        : 'bg-red-50 dark:bg-red-950 text-red-800 dark:text-red-300 border-red-200 dark:border-red-800'
                    }`}
                  >
                    {testResults[database.id].success ? (
                      <CheckCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                    ) : (
                      <XCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                    )}
                    <span className="text-sm font-medium">{testResults[database.id].message}</span>
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="modern-card modern-card-dark p-16">
            <div className="flex flex-col items-center justify-center text-center">
              <div className="w-20 h-20 mb-6 rounded-full bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-950 dark:to-purple-950 flex items-center justify-center">
                <DatabaseIcon className="w-10 h-10 text-blue-600 dark:text-blue-400" />
              </div>
              <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-3">No databases configured</h3>
              <p className="text-gray-600 dark:text-gray-400 mb-8 max-w-md">
                Get started by adding your first database connection. We support PostgreSQL, MySQL, MongoDB, Redis, and many more.
              </p>
              <button
                onClick={openCreateModal}
                className="btn-primary"
              >
                <Plus className="w-5 h-5 mr-2" />
                Add Your First Database
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-900 rounded-2xl p-8 w-full max-w-2xl shadow-2xl border border-gray-200 dark:border-gray-800 max-h-[90vh] overflow-y-auto custom-scrollbar">
            <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
              {editingDatabase ? 'Edit Database' : 'Add New Database'}
            </h3>
            <form onSubmit={handleSubmit}>
              <div className="space-y-5">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Database Name
                  </label>
                  <input
                    type="text"
                    name="name"
                    required
                    defaultValue={editingDatabase?.name}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all"
                    placeholder="My Production Database"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Database Type
                  </label>
                  <select
                    name="type"
                    required
                    defaultValue={editingDatabase?.type}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all"
                  >
                    <option value="">Select database type...</option>
                    <optgroup label="SQL Databases">
                      <option value="postgres">🐘 PostgreSQL</option>
                      <option value="mysql">🐬 MySQL</option>
                      <option value="timescaledb">⏱️ TimescaleDB</option>
                    </optgroup>
                    <optgroup label="NoSQL Databases">
                      <option value="mongodb">🍃 MongoDB</option>
                      <option value="redis">📦 Redis</option>
                      <option value="cassandra">🌌 Cassandra</option>
                      <option value="scylladb">⚡ ScyllaDB</option>
                      <option value="dynamodb">⚡ DynamoDB</option>
                    </optgroup>
                    <optgroup label="Search & Analytics">
                      <option value="elasticsearch">🔍 Elasticsearch</option>
                      <option value="opensearch">🔎 OpenSearch</option>
                    </optgroup>
                    <optgroup label="Time-Series">
                      <option value="influxdb">📈 InfluxDB</option>
                    </optgroup>
                  </select>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                      Host
                    </label>
                    <input
                      type="text"
                      name="host"
                      required
                      defaultValue={editingDatabase?.host}
                      className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all font-mono"
                      placeholder="localhost"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                      Port
                    </label>
                    <input
                      type="number"
                      name="port"
                      required
                      defaultValue={editingDatabase?.port}
                      className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all font-mono"
                      placeholder="5432"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Username
                  </label>
                  <input
                    type="text"
                    name="username"
                    required
                    defaultValue={editingDatabase?.username}
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all font-mono"
                    placeholder="postgres"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Password
                  </label>
                  <input
                    type="password"
                    name="password"
                    className="w-full px-4 py-3 border border-gray-300 dark:border-gray-700 rounded-xl bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent transition-all font-mono"
                    placeholder="••••••••"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-3 mt-8">
                <button
                  type="button"
                  onClick={() => {
                    setShowModal(false)
                    setEditingDatabase(null)
                  }}
                  className="px-6 py-3 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 rounded-xl hover:bg-gray-200 dark:hover:bg-gray-700 transition-all font-medium"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {createMutation.isPending || updateMutation.isPending ? (
                    <>
                      <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    <>
                      {editingDatabase ? (
                        <>
                          <Edit className="w-5 h-5 mr-2" />
                          Update Database
                        </>
                      ) : (
                        <>
                          <Plus className="w-5 h-5 mr-2" />
                          Add Database
                        </>
                      )}
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
