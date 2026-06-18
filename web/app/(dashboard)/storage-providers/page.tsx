'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, StorageProvider, StorageProviderType } from '@/lib/api'
import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit, TestTube2, CheckCircle, XCircle, HardDrive, Cloud, Database, Lock } from 'lucide-react'

export default function StorageProvidersPage() {
  const [showModal, setShowModal] = useState(false)
  const [editingProvider, setEditingProvider] = useState<StorageProvider | null>(null)
  const [selectedType, setSelectedType] = useState<StorageProviderType>('s3')
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({})

  const queryClient = useQueryClient()

  const { data: providers, isLoading } = useQuery({
    queryKey: ['storage-providers'],
    queryFn: () => api.listStorageProviders(),
  })

  const createMutation = useMutation({
    mutationFn: (data: Partial<StorageProvider>) => api.createStorageProvider(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['storage-providers'] })
      setShowModal(false)
      setEditingProvider(null)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<StorageProvider> }) =>
      api.updateStorageProvider(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['storage-providers'] })
      setShowModal(false)
      setEditingProvider(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteStorageProvider(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['storage-providers'] })
    },
  })

  const testConnectionMutation = useMutation({
    mutationFn: (id: string) => api.testStorageProvider(id),
    onSuccess: (data: { success: boolean; message: string }, id) => {
      setTestResults((prev) => ({ ...prev, [id]: data }))
      setTimeout(() => {
        setTestResults((prev) => {
          const newResults = { ...prev }
          delete newResults[id]
          return newResults
        })
      }, 5000)
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)

    const name = formData.get('name') as string
    const type = formData.get('type') as StorageProviderType
    const enabled = formData.get('enabled') === 'on'

    let config: any = { type }

    // Build config based on provider type
    switch (type) {
      case 'local':
        config.path = formData.get('path') as string
        break
      case 's3':
        config.region = formData.get('region') as string
        config.bucket = formData.get('bucket') as string
        config.access_key = formData.get('access_key') as string
        config.secret_key = formData.get('secret_key') as string
        config.endpoint = formData.get('endpoint') as string || undefined
        break
      case 'minio':
        config.endpoint = formData.get('endpoint') as string
        config.access_key = formData.get('access_key') as string
        config.secret_key = formData.get('secret_key') as string
        config.bucket = formData.get('bucket') as string
        config.use_ssl = formData.get('use_ssl') === 'on'
        config.region = formData.get('region') as string || 'us-east-1'
        break
      case 'wasabi':
        config.endpoint = formData.get('endpoint') as string
        config.region = formData.get('region') as string
        config.access_key = formData.get('access_key') as string
        config.secret_key = formData.get('secret_key') as string
        config.bucket = formData.get('bucket') as string
        config.use_ssl = formData.get('use_ssl') === 'on'
        config.immutable = formData.get('immutable') === 'on'
        config.retention_days = parseInt(formData.get('retention_days') as string) || undefined
        break
      case 'b2':
        config.account_id = formData.get('account_id') as string
        config.application_key = formData.get('application_key') as string
        config.bucket = formData.get('bucket') as string
        config.immutable = formData.get('immutable') === 'on'
        config.retention_days = parseInt(formData.get('retention_days') as string) || undefined
        break
      case 'gcs':
        config.project = formData.get('project') as string
        config.bucket = formData.get('bucket') as string
        config.credentials_file = formData.get('credentials_file') as string
        break
      case 'azure':
        config.account_name = formData.get('account_name') as string
        config.account_key = formData.get('account_key') as string
        config.container = formData.get('container') as string
        break
    }

    const data = { name, type, config, enabled }

    if (editingProvider) {
      updateMutation.mutate({ id: editingProvider.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const openEditModal = (provider: StorageProvider) => {
    setEditingProvider(provider)
    setSelectedType(provider.type)
    setShowModal(true)
  }

  const openCreateModal = () => {
    setEditingProvider(null)
    setSelectedType('s3')
    setShowModal(true)
  }

  const closeModal = () => {
    setShowModal(false)
    setEditingProvider(null)
  }

  useEffect(() => {
    if (!showModal) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeModal()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [showModal])

  const getProviderIcon = (type: StorageProviderType) => {
    switch (type) {
      case 'local':
        return <HardDrive className="w-5 h-5" />
      case 's3':
      case 'minio':
      case 'wasabi':
        return <Cloud className="w-5 h-5" />
      case 'b2':
        return <Database className="w-5 h-5" />
      case 'gcs':
      case 'azure':
        return <Cloud className="w-5 h-5" />
      default:
        return <Database className="w-5 h-5" />
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Storage Providers</h1>
          <p className="text-gray-500 mt-1">Manage backup storage destinations</p>
        </div>
        <button
          onClick={openCreateModal}
          className="flex items-center px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        >
          <Plus className="w-5 h-5 mr-2" />
          Add Provider
        </button>
      </div>

      {/* Providers Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {isLoading ? (
          <div className="col-span-full text-center py-12 text-gray-500">Loading...</div>
        ) : providers && providers.length > 0 ? (
          providers.map((provider) => (
            <div
              key={provider.id}
              className={`bg-white rounded-lg border ${
                provider.enabled ? 'border-gray-200' : 'border-gray-300 opacity-60'
              } p-6 hover:shadow-lg transition-shadow`}
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3">
                  <div className="p-2 bg-blue-100 rounded-lg text-blue-600">
                    {getProviderIcon(provider.type)}
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">{provider.name}</h3>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="inline-block px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                        {provider.type.toUpperCase()}
                      </span>
                      {provider.enabled ? (
                        <span className="inline-block px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded">
                          ENABLED
                        </span>
                      ) : (
                        <span className="inline-block px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded">
                          DISABLED
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex space-x-2">
                  <button
                    onClick={() => openEditModal(provider)}
                    className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
                    title="Edit"
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => {
                      if (confirm('Are you sure you want to delete this storage provider?')) {
                        deleteMutation.mutate(provider.id)
                      }
                    }}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <div className="space-y-2 text-sm text-gray-600 mb-4">
                {provider.type === 's3' && (
                  <>
                    <div className="flex justify-between">
                      <span>Region:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).region}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Bucket:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).bucket}</span>
                    </div>
                  </>
                )}
                {(provider.type === 'minio' || provider.type === 'wasabi') && (
                  <>
                    <div className="flex justify-between">
                      <span>Endpoint:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).endpoint}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Bucket:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).bucket}</span>
                    </div>
                  </>
                )}
                {provider.type === 'b2' && (
                  <>
                    <div className="flex justify-between">
                      <span>Account ID:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).account_id?.substring(0, 8)}...</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Bucket:</span>
                      <span className="font-medium text-gray-900">{(provider.config as any).bucket}</span>
                    </div>
                  </>
                )}
                {provider.type === 'local' && (
                  <div className="flex justify-between">
                    <span>Path:</span>
                    <span className="font-medium text-gray-900">{(provider.config as any).path}</span>
                  </div>
                )}
                {((provider.config as any).immutable) && (
                  <div className="flex items-center gap-1 text-amber-600">
                    <Lock className="w-4 h-4" />
                    <span>Immutable ({(provider.config as any).retention_days} days)</span>
                  </div>
                )}
              </div>

              <button
                onClick={() => testConnectionMutation.mutate(provider.id)}
                disabled={testConnectionMutation.isPending || !provider.enabled}
                className="w-full flex items-center justify-center px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                <TestTube2 className="w-4 h-4 mr-2" />
                Test Connection
              </button>

              {testResults[provider.id] && (
                <div
                  className={`mt-3 p-3 rounded-lg flex items-start ${
                    testResults[provider.id].success
                      ? 'bg-green-50 text-green-800'
                      : 'bg-red-50 text-red-800'
                  }`}
                >
                  {testResults[provider.id].success ? (
                    <CheckCircle className="w-5 h-5 mr-2 flex-shrink-0 mt-0.5" />
                  ) : (
                    <XCircle className="w-5 h-5 mr-2 flex-shrink-0 mt-0.5" />
                  )}
                  <span className="text-sm">{testResults[provider.id].message}</span>
                </div>
              )}
            </div>
          ))
        ) : (
          <div className="col-span-full text-center py-12 text-gray-500">
            No storage providers configured. Add one to get started.
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 overflow-y-auto" onClick={closeModal}>
          <div className="bg-white rounded-lg p-6 w-full max-w-2xl my-8 max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              {editingProvider ? 'Edit Storage Provider' : 'Add New Storage Provider'}
            </h3>
            <form onSubmit={handleSubmit}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Name</label>
                  <input
                    type="text"
                    name="name"
                    required
                    defaultValue={editingProvider?.name}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                    placeholder="My Storage Provider"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Type</label>
                  <select
                    name="type"
                    required
                    value={selectedType}
                    onChange={(e) => setSelectedType(e.target.value as StorageProviderType)}
                    disabled={!!editingProvider}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent disabled:bg-gray-100"
                  >
                    <option value="">Select type...</option>
                    <optgroup label="Cloud Storage">
                      <option value="s3">Amazon S3</option>
                      <option value="gcs">Google Cloud Storage</option>
                      <option value="azure">Azure Blob Storage</option>
                    </optgroup>
                    <optgroup label="S3-Compatible">
                      <option value="minio">MinIO</option>
                      <option value="wasabi">Wasabi</option>
                      <option value="b2">Backblaze B2</option>
                    </optgroup>
                    <optgroup label="Local">
                      <option value="local">Local Storage</option>
                    </optgroup>
                  </select>
                </div>

                <div>
                  <label className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      name="enabled"
                      defaultChecked={editingProvider?.enabled ?? true}
                      className="rounded border-gray-300 text-primary focus:ring-primary"
                    />
                    <span className="text-sm font-medium text-gray-700">Enabled</span>
                  </label>
                </div>

                {/* Provider-specific fields */}
                {selectedType === 'local' && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Path</label>
                    <input
                      type="text"
                      name="path"
                      required
                      defaultValue={editingProvider?.type === 'local' ? (editingProvider.config as any).path : ''}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      placeholder="/var/backups"
                    />
                  </div>
                )}

                {selectedType === 's3' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Region</label>
                      <input
                        type="text"
                        name="region"
                        required
                        defaultValue={editingProvider?.type === 's3' ? (editingProvider.config as any).region : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="us-east-1"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Bucket</label>
                      <input
                        type="text"
                        name="bucket"
                        required
                        defaultValue={editingProvider?.type === 's3' ? (editingProvider.config as any).bucket : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="my-backup-bucket"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Access Key</label>
                      <input
                        type="text"
                        name="access_key"
                        required
                        defaultValue={editingProvider?.type === 's3' ? (editingProvider.config as any).access_key : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Secret Key</label>
                      <input
                        type="password"
                        name="secret_key"
                        required
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Endpoint (optional)</label>
                      <input
                        type="text"
                        name="endpoint"
                        defaultValue={editingProvider?.type === 's3' ? (editingProvider.config as any).endpoint : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="https://s3.amazonaws.com"
                      />
                    </div>
                  </>
                )}

                {selectedType === 'minio' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Endpoint</label>
                      <input
                        type="text"
                        name="endpoint"
                        required
                        defaultValue={editingProvider?.type === 'minio' ? (editingProvider.config as any).endpoint : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="localhost:9000"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Access Key</label>
                      <input
                        type="text"
                        name="access_key"
                        required
                        defaultValue={editingProvider?.type === 'minio' ? (editingProvider.config as any).access_key : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Secret Key</label>
                      <input
                        type="password"
                        name="secret_key"
                        required
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Bucket</label>
                      <input
                        type="text"
                        name="bucket"
                        required
                        defaultValue={editingProvider?.type === 'minio' ? (editingProvider.config as any).bucket : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Region</label>
                      <input
                        type="text"
                        name="region"
                        defaultValue={editingProvider?.type === 'minio' ? (editingProvider.config as any).region : 'us-east-1'}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="us-east-1"
                      />
                    </div>
                    <div>
                      <label className="flex items-center space-x-2">
                        <input
                          type="checkbox"
                          name="use_ssl"
                          defaultChecked={editingProvider?.type === 'minio' ? (editingProvider.config as any).use_ssl : false}
                          className="rounded border-gray-300 text-primary focus:ring-primary"
                        />
                        <span className="text-sm font-medium text-gray-700">Use SSL</span>
                      </label>
                    </div>
                  </>
                )}

                {selectedType === 'wasabi' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Endpoint</label>
                      <input
                        type="text"
                        name="endpoint"
                        required
                        defaultValue={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).endpoint : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="s3.wasabisys.com"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Region</label>
                      <input
                        type="text"
                        name="region"
                        required
                        defaultValue={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).region : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="us-east-1"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Access Key</label>
                      <input
                        type="text"
                        name="access_key"
                        required
                        defaultValue={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).access_key : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Secret Key</label>
                      <input
                        type="password"
                        name="secret_key"
                        required
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Bucket</label>
                      <input
                        type="text"
                        name="bucket"
                        required
                        defaultValue={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).bucket : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div className="border-t pt-4">
                      <h4 className="text-sm font-semibold text-gray-700 mb-3">Immutability Settings</h4>
                      <div className="space-y-3">
                        <div>
                          <label className="flex items-center space-x-2">
                            <input
                              type="checkbox"
                              name="immutable"
                              defaultChecked={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).immutable : false}
                              className="rounded border-gray-300 text-primary focus:ring-primary"
                            />
                            <span className="text-sm font-medium text-gray-700">Enable Object Lock (Immutability)</span>
                          </label>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">Retention Days</label>
                          <input
                            type="number"
                            name="retention_days"
                            min="1"
                            defaultValue={editingProvider?.type === 'wasabi' ? (editingProvider.config as any).retention_days : 30}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                            placeholder="30"
                          />
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {selectedType === 'b2' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Account ID</label>
                      <input
                        type="text"
                        name="account_id"
                        required
                        defaultValue={editingProvider?.type === 'b2' ? (editingProvider.config as any).account_id : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Application Key</label>
                      <input
                        type="password"
                        name="application_key"
                        required
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Bucket</label>
                      <input
                        type="text"
                        name="bucket"
                        required
                        defaultValue={editingProvider?.type === 'b2' ? (editingProvider.config as any).bucket : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div className="border-t pt-4">
                      <h4 className="text-sm font-semibold text-gray-700 mb-3">Immutability Settings</h4>
                      <div className="space-y-3">
                        <div>
                          <label className="flex items-center space-x-2">
                            <input
                              type="checkbox"
                              name="immutable"
                              defaultChecked={editingProvider?.type === 'b2' ? (editingProvider.config as any).immutable : false}
                              className="rounded border-gray-300 text-primary focus:ring-primary"
                            />
                            <span className="text-sm font-medium text-gray-700">Enable File Lock (Immutability)</span>
                          </label>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">Retention Days</label>
                          <input
                            type="number"
                            name="retention_days"
                            min="1"
                            defaultValue={editingProvider?.type === 'b2' ? (editingProvider.config as any).retention_days : 30}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                            placeholder="30"
                          />
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {selectedType === 'gcs' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Project</label>
                      <input
                        type="text"
                        name="project"
                        required
                        defaultValue={editingProvider?.type === 'gcs' ? (editingProvider.config as any).project : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Bucket</label>
                      <input
                        type="text"
                        name="bucket"
                        required
                        defaultValue={editingProvider?.type === 'gcs' ? (editingProvider.config as any).bucket : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Credentials File Path</label>
                      <input
                        type="text"
                        name="credentials_file"
                        required
                        defaultValue={editingProvider?.type === 'gcs' ? (editingProvider.config as any).credentials_file : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                        placeholder="/path/to/credentials.json"
                      />
                    </div>
                  </>
                )}

                {selectedType === 'azure' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Account Name</label>
                      <input
                        type="text"
                        name="account_name"
                        required
                        defaultValue={editingProvider?.type === 'azure' ? (editingProvider.config as any).account_name : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Account Key</label>
                      <input
                        type="password"
                        name="account_key"
                        required
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Container</label>
                      <input
                        type="text"
                        name="container"
                        required
                        defaultValue={editingProvider?.type === 'azure' ? (editingProvider.config as any).container : ''}
                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                      />
                    </div>
                  </>
                )}
              </div>

              <div className="flex justify-end space-x-3 mt-6">
                <button
                  type="button"
                  onClick={() => {
                    setShowModal(false)
                    setEditingProvider(null)
                  }}
                  className="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 transition-colors"
                >
                  {createMutation.isPending || updateMutation.isPending
                    ? 'Saving...'
                    : editingProvider
                    ? 'Update'
                    : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
