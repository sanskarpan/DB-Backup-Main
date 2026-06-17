'use client'

import { useState, useEffect } from 'react'
import {
  Shield, FileText, Globe, CheckCircle, XCircle,
  AlertTriangle, Download, Trash2, Lock, Users,
  MapPin, Clock, ChevronRight, BarChart3, Settings
} from 'lucide-react'

export default function CompliancePage() {
  const [activeTab, setActiveTab] = useState('overview')
  const [refreshTime, setRefreshTime] = useState(new Date())

  useEffect(() => {
    const interval = setInterval(() => {
      setRefreshTime(new Date())
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
            <Shield className="w-8 h-8 text-blue-600" />
            Compliance & Governance
          </h1>
          <p className="text-gray-600 mt-2">
            GDPR compliance, data residency, and policy enforcement
          </p>
          <p className="text-sm text-gray-500 mt-1">
            Last updated: {refreshTime.toLocaleTimeString()}
          </p>
        </div>

        {/* Stats Overview */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="Consents"
            value="1,247"
            status="active"
            icon={<Users className="w-6 h-6" />}
            change="+12 this week"
          />
          <StatCard
            title="Erasure Requests"
            value="8"
            status="pending"
            icon={<Trash2 className="w-6 h-6" />}
            change="2 in grace period"
          />
          <StatCard
            title="Export Requests"
            value="23"
            status="active"
            icon={<Download className="w-6 h-6" />}
            change="15 completed"
          />
          <StatCard
            title="Policy Violations"
            value="3"
            status="warning"
            icon={<AlertTriangle className="w-6 h-6" />}
            change="1 resolved today"
          />
        </div>

        {/* Tabs */}
        <div className="bg-white rounded-lg shadow-sm mb-6">
          <div className="border-b border-gray-200">
            <nav className="flex -mb-px">
              <TabButton
                active={activeTab === 'overview'}
                onClick={() => setActiveTab('overview')}
                icon={<BarChart3 className="w-4 h-4" />}
              >
                Overview
              </TabButton>
              <TabButton
                active={activeTab === 'gdpr'}
                onClick={() => setActiveTab('gdpr')}
                icon={<FileText className="w-4 h-4" />}
              >
                GDPR Compliance
              </TabButton>
              <TabButton
                active={activeTab === 'residency'}
                onClick={() => setActiveTab('residency')}
                icon={<Globe className="w-4 h-4" />}
              >
                Data Residency
              </TabButton>
              <TabButton
                active={activeTab === 'policies'}
                onClick={() => setActiveTab('policies')}
                icon={<Settings className="w-4 h-4" />}
              >
                Policy Engine
              </TabButton>
            </nav>
          </div>

          <div className="p-6">
            {activeTab === 'overview' && <OverviewTab />}
            {activeTab === 'gdpr' && <GDPRTab />}
            {activeTab === 'residency' && <ResidencyTab />}
            {activeTab === 'policies' && <PoliciesTab />}
          </div>
        </div>
      </div>
    </div>
  )
}

// Tab Components
function OverviewTab() {
  return (
    <div className="space-y-6">
      {/* Compliance Score */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Compliance Score</h2>
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <div className="text-5xl font-bold text-blue-600">94%</div>
            <p className="text-gray-600 mt-2">Overall compliance rating</p>
          </div>
          <div className="w-32 h-32">
            <CircularProgress value={94} />
          </div>
        </div>
      </div>

      {/* Compliance Frameworks */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Compliance Frameworks</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <FrameworkCard name="GDPR" status="compliant" score={96} />
          <FrameworkCard name="CCPA" status="compliant" score={92} />
          <FrameworkCard name="HIPAA" status="partial" score={88} />
        </div>
      </div>

      {/* Recent Activity */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Compliance Activity</h3>
        <div className="space-y-3">
          <ActivityItem
            action="Consent granted"
            user="user@example.com"
            purpose="Backup Storage"
            time="2 minutes ago"
            status="success"
          />
          <ActivityItem
            action="Export completed"
            user="john@example.com"
            purpose="Data Portability"
            time="15 minutes ago"
            status="success"
          />
          <ActivityItem
            action="Policy violation detected"
            user="system"
            purpose="Data Residency"
            time="1 hour ago"
            status="warning"
          />
        </div>
      </div>
    </div>
  )
}

function GDPRTab() {
  const [selectedSection, setSelectedSection] = useState<'consents' | 'erasure' | 'portability'>('consents')

  return (
    <div className="space-y-6">
      {/* Section Selector */}
      <div className="flex gap-2">
        <button
          onClick={() => setSelectedSection('consents')}
          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
            selectedSection === 'consents'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Consent Management
        </button>
        <button
          onClick={() => setSelectedSection('erasure')}
          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
            selectedSection === 'erasure'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Right to Erasure
        </button>
        <button
          onClick={() => setSelectedSection('portability')}
          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
            selectedSection === 'portability'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Data Portability
        </button>
      </div>

      {selectedSection === 'consents' && <ConsentsSection />}
      {selectedSection === 'erasure' && <ErasureSection />}
      {selectedSection === 'portability' && <PortabilitySection />}
    </div>
  )
}

function ConsentsSection() {
  const consents = [
    { id: 1, user: 'user@example.com', purpose: 'Backup Storage', status: 'granted', date: '2024-01-09' },
    { id: 2, user: 'john@example.com', purpose: 'Data Processing', status: 'granted', date: '2024-01-08' },
    { id: 3, user: 'jane@example.com', purpose: 'Analytics', status: 'revoked', date: '2024-01-07' },
    { id: 4, user: 'admin@example.com', purpose: 'Third Party Sharing', status: 'granted', date: '2024-01-06' },
  ]

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">User Consents</h3>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
          Request New Consent
        </button>
      </div>
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Purpose</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Date</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {consents.map((consent) => (
              <tr key={consent.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{consent.user}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{consent.purpose}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <StatusBadge status={consent.status} />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{consent.date}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <button className="text-blue-600 hover:text-blue-800 mr-3">View</button>
                  {consent.status === 'granted' && (
                    <button className="text-red-600 hover:text-red-800">Revoke</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ErasureSection() {
  const requests = [
    { id: 1, user: 'user@example.com', status: 'pending', requested: '2024-01-09', gracePeriod: '7 days remaining' },
    { id: 2, user: 'old@example.com', status: 'processing', requested: '2024-01-03', gracePeriod: 'Expired' },
    { id: 3, user: 'deleted@example.com', status: 'completed', requested: '2023-12-15', gracePeriod: 'N/A' },
  ]

  return (
    <div>
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
        <div className="flex items-start">
          <AlertTriangle className="w-5 h-5 text-yellow-600 mt-0.5 mr-3" />
          <div>
            <h4 className="text-sm font-medium text-yellow-800">Grace Period Active</h4>
            <p className="text-sm text-yellow-700 mt-1">
              2 erasure requests are in the 30-day grace period. Users can still cancel these requests.
            </p>
          </div>
        </div>
      </div>

      <h3 className="text-lg font-semibold text-gray-900 mb-4">Erasure Requests</h3>
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Requested</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Grace Period</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {requests.map((request) => (
              <tr key={request.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{request.user}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <StatusBadge status={request.status} />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{request.requested}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{request.gracePeriod}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <button className="text-blue-600 hover:text-blue-800 mr-3">View</button>
                  {request.status === 'pending' && (
                    <button className="text-green-600 hover:text-green-800">Process</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function PortabilitySection() {
  const exports = [
    { id: 1, user: 'user@example.com', format: 'JSON', status: 'completed', size: '2.3 MB', requested: '2024-01-09' },
    { id: 2, user: 'john@example.com', format: 'ZIP', status: 'processing', size: '-', requested: '2024-01-09' },
    { id: 3, user: 'jane@example.com', format: 'XML', status: 'completed', size: '1.8 MB', requested: '2024-01-08' },
  ]

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Data Export Requests</h3>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
          Create Export
        </button>
      </div>
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Format</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Size</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Requested</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {exports.map((exp) => (
              <tr key={exp.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{exp.user}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{exp.format}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <StatusBadge status={exp.status} />
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{exp.size}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{exp.requested}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {exp.status === 'completed' && (
                    <button className="flex items-center text-blue-600 hover:text-blue-800">
                      <Download className="w-4 h-4 mr-1" />
                      Download
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ResidencyTab() {
  const regions = [
    { region: 'EU-West', policies: 12, violations: 0, compliance: 100, status: 'compliant' },
    { region: 'US-East', policies: 8, violations: 1, compliance: 95, status: 'warning' },
    { region: 'AP-South', policies: 6, violations: 0, compliance: 100, status: 'compliant' },
    { region: 'CA-Central', policies: 4, violations: 0, compliance: 100, status: 'compliant' },
  ]

  const transfers = [
    { id: 1, from: 'EU-West', to: 'US-East', resource: 'backup-123', status: 'pending', requested: '2024-01-09' },
    { id: 2, from: 'US-East', to: 'EU-West', resource: 'db-456', status: 'approved', requested: '2024-01-08' },
    { id: 3, from: 'AP-South', to: 'AP-Southeast', resource: 'backup-789', status: 'completed', requested: '2024-01-07' },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Regional Compliance</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {regions.map((region) => (
            <div key={region.region} className="bg-white border border-gray-200 rounded-lg p-4">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <MapPin className="w-5 h-5 text-blue-600" />
                  <h4 className="font-semibold text-gray-900">{region.region}</h4>
                </div>
                <StatusBadge status={region.status} />
              </div>
              <div className="grid grid-cols-3 gap-2 text-sm">
                <div>
                  <div className="text-gray-500">Policies</div>
                  <div className="font-semibold text-gray-900">{region.policies}</div>
                </div>
                <div>
                  <div className="text-gray-500">Violations</div>
                  <div className={`font-semibold ${region.violations > 0 ? 'text-red-600' : 'text-green-600'}`}>
                    {region.violations}
                  </div>
                </div>
                <div>
                  <div className="text-gray-500">Compliance</div>
                  <div className="font-semibold text-gray-900">{region.compliance}%</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Cross-Border Transfer Requests</h3>
        <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Source</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Destination</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Resource</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Requested</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {transfers.map((transfer) => (
                <tr key={transfer.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{transfer.from}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 flex items-center gap-2">
                    <ChevronRight className="w-4 h-4 text-gray-400" />
                    {transfer.to}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{transfer.resource}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={transfer.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{transfer.requested}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {transfer.status === 'pending' && (
                      <>
                        <button className="text-green-600 hover:text-green-800 mr-3">Approve</button>
                        <button className="text-red-600 hover:text-red-800">Reject</button>
                      </>
                    )}
                    {transfer.status !== 'pending' && (
                      <button className="text-blue-600 hover:text-blue-800">View</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function PoliciesTab() {
  const policies = [
    { name: 'Backup Authorization', path: 'backup/authorization', status: 'active', evaluations: 1234 },
    { name: 'Storage Compliance', path: 'storage/compliance', status: 'active', evaluations: 892 },
    { name: 'Security Access', path: 'security/access', status: 'active', evaluations: 2456 },
    { name: 'Data Residency', path: 'compliance/residency', status: 'active', evaluations: 567 },
  ]

  return (
    <div className="space-y-6">
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-start">
          <Shield className="w-5 h-5 text-blue-600 mt-0.5 mr-3" />
          <div>
            <h4 className="text-sm font-medium text-blue-800">Open Policy Agent Integration</h4>
            <p className="text-sm text-blue-700 mt-1">
              All policies are enforced through OPA (Open Policy Agent) with Rego policy language.
              Policies are evaluated in real-time for every request.
            </p>
          </div>
        </div>
      </div>

      <div>
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Loaded Policies</h3>
          <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
            Load New Policy
          </button>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {policies.map((policy) => (
            <div key={policy.path} className="bg-white border border-gray-200 rounded-lg p-4">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Lock className="w-5 h-5 text-blue-600" />
                  <h4 className="font-semibold text-gray-900">{policy.name}</h4>
                </div>
                <StatusBadge status={policy.status} />
              </div>
              <div className="text-sm text-gray-600 mb-3">
                <code className="bg-gray-100 px-2 py-1 rounded">{policy.path}</code>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-gray-500">{policy.evaluations.toLocaleString()} evaluations</span>
                <div className="space-x-2">
                  <button className="text-blue-600 hover:text-blue-800">Edit</button>
                  <button className="text-blue-600 hover:text-blue-800">Test</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Policy Evaluations</h3>
        <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
          <PolicyEvaluation
            policy="backup/authorization"
            decision="allow"
            user="user@example.com"
            action="create_backup"
            time="2 min ago"
          />
          <PolicyEvaluation
            policy="security/access"
            decision="deny"
            user="attacker@example.com"
            action="delete_data"
            time="5 min ago"
          />
          <PolicyEvaluation
            policy="storage/compliance"
            decision="allow"
            user="system"
            action="store_backup"
            time="8 min ago"
          />
        </div>
      </div>
    </div>
  )
}

// UI Components
function StatCard({ title, value, status, icon, change }: any) {
  const statusColors = {
    active: 'bg-green-50 text-green-700',
    pending: 'bg-yellow-50 text-yellow-700',
    warning: 'bg-orange-50 text-orange-700',
  }

  return (
    <div className="bg-white rounded-lg shadow-sm p-6 border border-gray-200">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-medium text-gray-600">{title}</h3>
        <div className={`p-2 rounded-lg ${statusColors[status as keyof typeof statusColors]}`}>
          {icon}
        </div>
      </div>
      <div className="text-2xl font-bold text-gray-900 mb-1">{value}</div>
      <p className="text-xs text-gray-500">{change}</p>
    </div>
  )
}

function TabButton({ active, onClick, icon, children }: any) {
  return (
    <button
      onClick={onClick}
      className={`px-6 py-3 font-medium text-sm transition-colors flex items-center gap-2 ${
        active
          ? 'border-b-2 border-blue-600 text-blue-600'
          : 'text-gray-600 hover:text-gray-900'
      }`}
    >
      {icon}
      {children}
    </button>
  )
}

function StatusBadge({ status }: { status: string }) {
  const statusConfig: Record<string, { bg: string; text: string; icon: any }> = {
    granted: { bg: 'bg-green-100', text: 'text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
    revoked: { bg: 'bg-red-100', text: 'text-red-800', icon: <XCircle className="w-3 h-3" /> },
    pending: { bg: 'bg-yellow-100', text: 'text-yellow-800', icon: <Clock className="w-3 h-3" /> },
    processing: { bg: 'bg-blue-100', text: 'text-blue-800', icon: <Clock className="w-3 h-3" /> },
    completed: { bg: 'bg-green-100', text: 'text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
    active: { bg: 'bg-green-100', text: 'text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
    compliant: { bg: 'bg-green-100', text: 'text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
    warning: { bg: 'bg-orange-100', text: 'text-orange-800', icon: <AlertTriangle className="w-3 h-3" /> },
    approved: { bg: 'bg-green-100', text: 'text-green-800', icon: <CheckCircle className="w-3 h-3" /> },
  }

  const config = statusConfig[status] || { bg: 'bg-gray-100', text: 'text-gray-800', icon: null }

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${config.bg} ${config.text}`}>
      {config.icon}
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}

function CircularProgress({ value }: { value: number }) {
  const circumference = 2 * Math.PI * 45
  const offset = circumference - (value / 100) * circumference

  return (
    <svg className="w-full h-full transform -rotate-90">
      <circle
        cx="50%"
        cy="50%"
        r="45"
        stroke="#e5e7eb"
        strokeWidth="8"
        fill="none"
      />
      <circle
        cx="50%"
        cy="50%"
        r="45"
        stroke="#2563eb"
        strokeWidth="8"
        fill="none"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        strokeLinecap="round"
      />
    </svg>
  )
}

function FrameworkCard({ name, status, score }: any) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center justify-between mb-2">
        <h4 className="font-semibold text-gray-900">{name}</h4>
        <StatusBadge status={status} />
      </div>
      <div className="text-2xl font-bold text-blue-600">{score}%</div>
      <div className="w-full bg-gray-200 rounded-full h-2 mt-2">
        <div
          className="bg-blue-600 h-2 rounded-full transition-all"
          style={{ width: `${score}%` }}
        />
      </div>
    </div>
  )
}

function ActivityItem({ action, user, purpose, time, status }: any) {
  return (
    <div className="flex items-center justify-between p-3 bg-white border border-gray-200 rounded-lg">
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${
          status === 'success' ? 'bg-green-50 text-green-600' : 'bg-yellow-50 text-yellow-600'
        }`}>
          {status === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}
        </div>
        <div>
          <div className="font-medium text-gray-900">{action}</div>
          <div className="text-sm text-gray-600">{user} • {purpose}</div>
        </div>
      </div>
      <div className="text-sm text-gray-500">{time}</div>
    </div>
  )
}

function PolicyEvaluation({ policy, decision, user, action, time }: any) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
      <div className="flex items-center gap-3">
        <div className={`p-1.5 rounded ${
          decision === 'allow' ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'
        }`}>
          {decision === 'allow' ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
        </div>
        <div>
          <div className="text-sm font-medium text-gray-900">
            <code className="text-xs bg-gray-100 px-1 py-0.5 rounded">{policy}</code>
          </div>
          <div className="text-xs text-gray-600">{user} • {action}</div>
        </div>
      </div>
      <div className="text-xs text-gray-500">{time}</div>
    </div>
  )
}
