'use client';

import React, { useState } from 'react';
import {
  Cloud,
  HardDrive,
  CheckCircle2,
  XCircle,
  Settings,
  Trash2,
  Plus,
  RefreshCw,
  AlertCircle,
  Eye,
  EyeOff,
  TrendingUp,
  Server,
  Shield,
  Zap,
  BarChart3,
  Download,
  Upload,
} from 'lucide-react';

interface StorageProvider {
  id: string;
  name: string;
  type: 'aws' | 'azure' | 'gcp' | 'wasabi' | 'backblaze' | 'local';
  status: 'connected' | 'disconnected' | 'error';
  region: string;
  totalStorage: string;
  usedStorage: string;
  bucketName: string;
  createdAt: Date;
  lastSync?: Date;
  logo: string;
}

interface ProviderConfig {
  accessKey: string;
  secretKey: string;
  region: string;
  bucketName: string;
  endpoint?: string;
}

export default function StorageProvidersPage() {
  const [providers, setProviders] = useState<StorageProvider[]>([
    {
      id: '1',
      name: 'AWS S3 Production',
      type: 'aws',
      status: 'connected',
      region: 'us-west-2',
      totalStorage: '500 GB',
      usedStorage: '287 GB',
      bucketName: 'prod-backups',
      createdAt: new Date('2024-01-15'),
      lastSync: new Date('2024-01-22T14:30:00'),
      logo: '🟠',
    },
    {
      id: '2',
      name: 'Azure Blob Storage',
      type: 'azure',
      status: 'connected',
      region: 'eastus',
      totalStorage: '1 TB',
      usedStorage: '412 GB',
      bucketName: 'backup-container',
      createdAt: new Date('2024-01-10'),
      lastSync: new Date('2024-01-22T12:00:00'),
      logo: '🔵',
    },
    {
      id: '3',
      name: 'GCP Cloud Storage',
      type: 'gcp',
      status: 'disconnected',
      region: 'us-central1',
      totalStorage: '250 GB',
      usedStorage: '0 GB',
      bucketName: 'gcp-backups',
      createdAt: new Date('2024-01-20'),
      logo: '🔴',
    },
  ]);

  const [showAddProvider, setShowAddProvider] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<StorageProvider | null>(null);
  const [testing, setTesting] = useState<{ [key: string]: boolean }>({});
  const [showCredentials, setShowCredentials] = useState<{ [key: string]: boolean }>({});
  const [newProviderType, setNewProviderType] = useState<StorageProvider['type']>('aws');
  const [config, setConfig] = useState<ProviderConfig>({
    accessKey: '',
    secretKey: '',
    region: '',
    bucketName: '',
    endpoint: '',
  });

  const providerTemplates = [
    {
      type: 'aws',
      name: 'Amazon S3',
      logo: '🟠',
      description: 'Industry-leading object storage with 99.999999999% durability',
      features: ['Global infrastructure', 'Unlimited scalability', 'Advanced security'],
    },
    {
      type: 'azure',
      name: 'Azure Blob Storage',
      logo: '🔵',
      description: 'Microsoft Azure cloud storage for large amounts of unstructured data',
      features: ['Integrated with Azure', 'Hot and cool tiers', 'Global redundancy'],
    },
    {
      type: 'gcp',
      name: 'Google Cloud Storage',
      logo: '🔴',
      description: 'Unified object storage for developers and enterprises',
      features: ['Multi-regional storage', 'Strong consistency', 'ML integration'],
    },
    {
      type: 'wasabi',
      name: 'Wasabi Hot Cloud Storage',
      logo: '🟢',
      description: 'Affordable cloud storage with no egress fees',
      features: ['80% cheaper than AWS', 'No API charges', 'No egress fees'],
    },
    {
      type: 'backblaze',
      name: 'Backblaze B2',
      logo: '🔴',
      description: 'Simple, reliable, affordable cloud storage',
      features: ['S3-compatible API', 'Free egress to CDN', 'Easy integration'],
    },
    {
      type: 'local',
      name: 'Local Storage',
      logo: '💾',
      description: 'Store backups on local or network-attached storage',
      features: ['Full control', 'No bandwidth costs', 'Fast access'],
    },
  ];

  const handleTestConnection = async (id: string) => {
    setTesting({ ...testing, [id]: true });

    // Simulate connection test
    setTimeout(() => {
      setTesting({ ...testing, [id]: false });
      // Update provider status
      setProviders(
        providers.map((p) =>
          p.id === id ? { ...p, status: 'connected', lastSync: new Date() } : p
        )
      );
    }, 2000);
  };

  const handleDeleteProvider = (id: string) => {
    setProviders(providers.filter((p) => p.id !== id));
  };

  const handleAddProvider = () => {
    const template = providerTemplates.find((t) => t.type === newProviderType);
    if (!template || !config.bucketName || !config.region) return;

    const newProvider: StorageProvider = {
      id: `${Date.now()}`,
      name: `${template.name} - ${config.bucketName}`,
      type: newProviderType,
      status: 'disconnected',
      region: config.region,
      totalStorage: '0 GB',
      usedStorage: '0 GB',
      bucketName: config.bucketName,
      createdAt: new Date(),
      logo: template.logo,
    };

    setProviders([...providers, newProvider]);
    setShowAddProvider(false);
    setConfig({
      accessKey: '',
      secretKey: '',
      region: '',
      bucketName: '',
      endpoint: '',
    });
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'badge-success';
      case 'disconnected':
        return 'badge-warning';
      case 'error':
        return 'badge-error';
      default:
        return 'badge-info';
    }
  };

  const calculateUsagePercentage = (used: string, total: string) => {
    const usedNum = parseFloat(used);
    const totalNum = parseFloat(total);
    if (isNaN(usedNum) || isNaN(totalNum) || totalNum === 0) return 0;
    return Math.round((usedNum / totalNum) * 100);
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
              <Cloud className="w-8 h-8 text-white" />
            </div>
            <div>
              <h1 className="text-4xl font-bold text-white mb-2">Storage Providers</h1>
              <p className="text-white/90 text-lg">Connect and manage your cloud storage destinations</p>
            </div>
          </div>

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-8">
            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Active Providers</p>
                  <p className="text-3xl font-bold text-white">
                    {providers.filter((p) => p.status === 'connected').length}
                  </p>
                </div>
                <Cloud className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Total Storage</p>
                  <p className="text-3xl font-bold text-white">1.7 TB</p>
                </div>
                <HardDrive className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Used Storage</p>
                  <p className="text-3xl font-bold text-white">699 GB</p>
                </div>
                <TrendingUp className="w-8 h-8 text-white/50" />
              </div>
            </div>

            <div className="glass-card p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm mb-1">Avg Usage</p>
                  <p className="text-3xl font-bold text-white">41%</p>
                </div>
                <BarChart3 className="w-8 h-8 text-white/50" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Add Provider Button */}
        <div className="flex justify-between items-center mb-8">
          <div>
            <h2 className="text-2xl font-bold text-[var(--text-primary)]">Connected Providers</h2>
            <p className="text-[var(--text-secondary)] mt-1">
              Manage your storage destinations for database backups
            </p>
          </div>
          <button
            onClick={() => setShowAddProvider(!showAddProvider)}
            className="btn-primary flex items-center gap-2"
          >
            <Plus className="w-5 h-5" />
            Add Provider
          </button>
        </div>

        {/* Add Provider Form */}
        {showAddProvider && (
          <div className="modern-card mb-8 border-l-4 border-l-[var(--orange-primary)] animate-fade-in">
            <h3 className="text-lg font-semibold text-[var(--text-primary)] mb-6">
              Add New Storage Provider
            </h3>

            {/* Provider Type Selection */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
              {providerTemplates.map((template) => (
                <button
                  key={template.type}
                  onClick={() => setNewProviderType(template.type as any)}
                  className={`p-4 text-left rounded-lg border-2 transition-all ${
                    newProviderType === template.type
                      ? 'border-[var(--orange-primary)] bg-[var(--surface)]'
                      : 'border-[var(--border)] hover:border-[var(--orange-light)]'
                  }`}
                >
                  <div className="text-3xl mb-2">{template.logo}</div>
                  <h4 className="font-semibold text-[var(--text-primary)] mb-1">{template.name}</h4>
                  <p className="text-xs text-[var(--text-secondary)]">{template.description}</p>
                </button>
              ))}
            </div>

            {/* Configuration Form */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div>
                <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                  Access Key / Client ID
                </label>
                <input
                  type="text"
                  value={config.accessKey}
                  onChange={(e) => setConfig({ ...config, accessKey: e.target.value })}
                  placeholder="Enter access key"
                  className="input-modern"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                  Secret Key / Client Secret
                </label>
                <input
                  type="password"
                  value={config.secretKey}
                  onChange={(e) => setConfig({ ...config, secretKey: e.target.value })}
                  placeholder="Enter secret key"
                  className="input-modern"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                  Region
                </label>
                <input
                  type="text"
                  value={config.region}
                  onChange={(e) => setConfig({ ...config, region: e.target.value })}
                  placeholder="e.g., us-west-2"
                  className="input-modern"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                  Bucket / Container Name
                </label>
                <input
                  type="text"
                  value={config.bucketName}
                  onChange={(e) => setConfig({ ...config, bucketName: e.target.value })}
                  placeholder="Enter bucket name"
                  className="input-modern"
                />
              </div>

              {(newProviderType === 'wasabi' || newProviderType === 'backblaze') && (
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-[var(--text-primary)] mb-2">
                    Custom Endpoint (Optional)
                  </label>
                  <input
                    type="text"
                    value={config.endpoint}
                    onChange={(e) => setConfig({ ...config, endpoint: e.target.value })}
                    placeholder="https://s3.wasabisys.com"
                    className="input-modern"
                  />
                </div>
              )}
            </div>

            <div className="flex gap-3">
              <button
                onClick={handleAddProvider}
                disabled={!config.bucketName || !config.region}
                className="btn-primary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <CheckCircle2 className="w-5 h-5" />
                Add Provider
              </button>
              <button onClick={() => setShowAddProvider(false)} className="btn-secondary">
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Provider Cards */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          {providers.map((provider) => (
            <div key={provider.id} className="modern-card">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="text-4xl">{provider.logo}</div>
                  <div>
                    <h3 className="font-semibold text-[var(--text-primary)] text-lg">{provider.name}</h3>
                    <p className="text-sm text-[var(--text-secondary)]">{provider.region}</p>
                  </div>
                </div>
                <span className={`badge ${getStatusColor(provider.status)}`}>
                  {provider.status}
                </span>
              </div>

              {/* Storage Usage */}
              <div className="mb-4">
                <div className="flex justify-between text-sm mb-2">
                  <span className="text-[var(--text-secondary)]">Storage Usage</span>
                  <span className="font-semibold text-[var(--text-primary)]">
                    {provider.usedStorage} / {provider.totalStorage}
                  </span>
                </div>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${calculateUsagePercentage(provider.usedStorage, provider.totalStorage)}%`,
                    }}
                  ></div>
                </div>
              </div>

              {/* Details */}
              <div className="grid grid-cols-2 gap-4 mb-4 text-sm">
                <div>
                  <p className="text-[var(--text-tertiary)] text-xs mb-1">Bucket Name</p>
                  <p className="font-medium text-[var(--text-primary)]">{provider.bucketName}</p>
                </div>
                <div>
                  <p className="text-[var(--text-tertiary)] text-xs mb-1">Created</p>
                  <p className="font-medium text-[var(--text-primary)]">
                    {provider.createdAt.toLocaleDateString()}
                  </p>
                </div>
                {provider.lastSync && (
                  <div className="col-span-2">
                    <p className="text-[var(--text-tertiary)] text-xs mb-1">Last Sync</p>
                    <p className="font-medium text-[var(--text-primary)]">
                      {provider.lastSync.toLocaleString()}
                    </p>
                  </div>
                )}
              </div>

              {/* Actions */}
              <div className="flex gap-2 pt-4 border-t border-[var(--border)]">
                <button
                  onClick={() => handleTestConnection(provider.id)}
                  disabled={testing[provider.id]}
                  className="btn-secondary flex-1 flex items-center justify-center gap-2 disabled:opacity-50"
                >
                  {testing[provider.id] ? (
                    <>
                      <div className="w-4 h-4 border-2 border-[var(--text-primary)] border-t-transparent rounded-full animate-spin"></div>
                      Testing...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="w-4 h-4" />
                      Test Connection
                    </>
                  )}
                </button>
                <button
                  onClick={() => setSelectedProvider(provider)}
                  className="btn-ghost"
                >
                  <Settings className="w-5 h-5" />
                </button>
                <button
                  onClick={() => handleDeleteProvider(provider.id)}
                  className="btn-ghost text-[var(--error)]"
                >
                  <Trash2 className="w-5 h-5" />
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* Empty State */}
        {providers.length === 0 && !showAddProvider && (
          <div className="empty-state py-16">
            <Cloud className="empty-state-icon" />
            <h3 className="empty-state-title">No storage providers configured</h3>
            <p className="empty-state-description">
              Add your first storage provider to start backing up your databases
            </p>
            <button
              onClick={() => setShowAddProvider(true)}
              className="btn-primary flex items-center gap-2 mt-4"
            >
              <Plus className="w-5 h-5" />
              Add Provider
            </button>
          </div>
        )}

        {/* Provider Comparison Table */}
        {providers.length > 0 && (
          <div className="modern-card">
            <h3 className="text-xl font-bold text-[var(--text-primary)] mb-6">
              Storage Provider Comparison
            </h3>
            <div className="overflow-x-auto">
              <table className="table-modern">
                <thead>
                  <tr>
                    <th>Provider</th>
                    <th>Status</th>
                    <th>Region</th>
                    <th>Total Storage</th>
                    <th>Used Storage</th>
                    <th>Usage %</th>
                    <th>Last Sync</th>
                  </tr>
                </thead>
                <tbody>
                  {providers.map((provider) => (
                    <tr key={provider.id}>
                      <td>
                        <div className="flex items-center gap-2">
                          <span className="text-2xl">{provider.logo}</span>
                          <span className="font-medium">{provider.name}</span>
                        </div>
                      </td>
                      <td>
                        <span className={`badge ${getStatusColor(provider.status)}`}>
                          {provider.status}
                        </span>
                      </td>
                      <td>{provider.region}</td>
                      <td>{provider.totalStorage}</td>
                      <td>{provider.usedStorage}</td>
                      <td>
                        <div className="flex items-center gap-2">
                          <div className="w-20 progress-bar h-2">
                            <div
                              className="progress-fill"
                              style={{
                                width: `${calculateUsagePercentage(
                                  provider.usedStorage,
                                  provider.totalStorage
                                )}%`,
                              }}
                            ></div>
                          </div>
                          <span className="text-sm">
                            {calculateUsagePercentage(provider.usedStorage, provider.totalStorage)}%
                          </span>
                        </div>
                      </td>
                      <td className="text-sm text-[var(--text-secondary)]">
                        {provider.lastSync ? provider.lastSync.toLocaleString() : 'Never'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Info Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-8">
          <div className="stats-card">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                <Shield className="w-5 h-5 text-[var(--orange-primary)]" />
              </div>
              <div>
                <h4 className="font-semibold text-[var(--text-primary)] mb-1">Secure Connections</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  All credentials are encrypted at rest and in transit using AES-256 encryption.
                </p>
              </div>
            </div>
          </div>

          <div className="stats-card">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                <Zap className="w-5 h-5 text-[var(--orange-primary)]" />
              </div>
              <div>
                <h4 className="font-semibold text-[var(--text-primary)] mb-1">Multi-Cloud Support</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  Connect to multiple cloud providers simultaneously for redundancy.
                </p>
              </div>
            </div>
          </div>

          <div className="stats-card">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-[var(--orange-primary)]/10 rounded-lg">
                <Server className="w-5 h-5 text-[var(--orange-primary)]" />
              </div>
              <div>
                <h4 className="font-semibold text-[var(--text-primary)] mb-1">Automated Backups</h4>
                <p className="text-sm text-[var(--text-secondary)]">
                  Configure automatic backup schedules to your preferred storage provider.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
