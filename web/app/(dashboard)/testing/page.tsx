'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Play,
  Pause,
  RotateCw,
  Activity,
  Zap,
  Shield,
  TrendingUp,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Clock,
  FileCode,
  Users,
  Cpu,
  Network,
  HardDrive,
} from 'lucide-react'

interface TestRun {
  id: string
  type: 'chaos' | 'load' | 'contract' | 'mutation'
  name: string
  status: 'running' | 'passed' | 'failed' | 'pending'
  duration?: number
  startedAt?: string
  completedAt?: string
}

interface ChaosExperiment {
  name: string
  type: string
  status: 'active' | 'paused' | 'completed'
  duration: string
  lastRun?: string
}

interface LoadTestResult {
  tool: 'k6' | 'locust' | 'jmeter'
  scenario: string
  vus: number
  requests: number
  failures: number
  avgResponseTime: number
  p95ResponseTime: number
}

export default function TestingPage() {
  const [selectedTab, setSelectedTab] = useState<'chaos' | 'load' | 'contract' | 'mutation'>('chaos')

  const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

  // Fetch test runs
  const { data: testRuns } = useQuery<TestRun[]>({
    queryKey: ['test-runs'],
    queryFn: async () => {
      const res = await fetch(`${BACKEND_URL}/testing/runs`)
      if (!res.ok) throw new Error('Failed to fetch test runs')
      return res.json()
    },
    refetchInterval: 10000,
  })

  // Fetch chaos experiments
  const { data: chaosExperiments } = useQuery<ChaosExperiment[]>({
    queryKey: ['chaos-experiments'],
    queryFn: async () => {
      const res = await fetch(`${BACKEND_URL}/testing/chaos/experiments`)
      if (!res.ok) throw new Error('Failed to fetch chaos experiments')
      return res.json()
    },
  })

  // Fetch load test results
  const { data: loadTestResults } = useQuery<LoadTestResult[]>({
    queryKey: ['load-test-results'],
    queryFn: async () => {
      const res = await fetch(`${BACKEND_URL}/testing/load/results`)
      if (!res.ok) throw new Error('Failed to fetch load test results')
      return res.json()
    },
  })

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Testing & Quality Assurance</h1>
          <p className="text-gray-600">Comprehensive testing dashboard for chaos, load, contract, and mutation testing</p>
        </div>

        {/* Test Status Overview */}
        <div className="grid grid-cols-4 gap-6 mb-8">
          <StatusCard
            icon={<Zap className="w-6 h-6" />}
            title="Chaos Tests"
            count={chaosExperiments?.length || 0}
            status="active"
            color="purple"
          />
          <StatusCard
            icon={<Activity className="w-6 h-6" />}
            title="Load Tests"
            count={loadTestResults?.length || 0}
            status="passed"
            color="blue"
          />
          <StatusCard
            icon={<Shield className="w-6 h-6" />}
            title="Contract Tests"
            count={12}
            status="passed"
            color="green"
          />
          <StatusCard
            icon={<FileCode className="w-6 h-6" />}
            title="Mutation Score"
            count={85}
            status="passed"
            color="emerald"
          />
        </div>

        {/* Tab Navigation */}
        <div className="mb-6 flex gap-2">
          {[
            { id: 'chaos', label: 'Chaos Engineering', icon: <Zap className="w-4 h-4" /> },
            { id: 'load', label: 'Load Testing', icon: <Activity className="w-4 h-4" /> },
            { id: 'contract', label: 'Contract Testing', icon: <Shield className="w-4 h-4" /> },
            { id: 'mutation', label: 'Mutation Testing', icon: <FileCode className="w-4 h-4" /> },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setSelectedTab(tab.id as any)}
              className={`flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition-all ${
                selectedTab === tab.id
                  ? 'bg-blue-600 text-white shadow-md'
                  : 'bg-white text-gray-700 hover:bg-gray-100 border border-gray-200'
              }`}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab Content */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8">
          {selectedTab === 'chaos' && <ChaosEngineeringTab experiments={chaosExperiments} />}
          {selectedTab === 'load' && <LoadTestingTab results={loadTestResults} />}
          {selectedTab === 'contract' && <ContractTestingTab />}
          {selectedTab === 'mutation' && <MutationTestingTab />}
        </div>

        {/* Recent Test Runs */}
        <div className="mt-8 bg-white rounded-xl shadow-sm border border-gray-200 p-8">
          <h2 className="text-xl font-bold text-gray-900 mb-6">Recent Test Runs</h2>
          <div className="space-y-3">
            {testRuns?.slice(0, 10).map((run) => (
              <TestRunCard key={run.id} run={run} />
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

interface StatusCardProps {
  icon: React.ReactNode
  title: string
  count: number
  status: 'active' | 'passed' | 'failed' | 'pending'
  color: string
}

function StatusCard({ icon, title, count, status, color }: StatusCardProps) {
  const colorClasses = {
    purple: 'text-purple-600 bg-purple-50 border-purple-200',
    blue: 'text-blue-600 bg-blue-50 border-blue-200',
    green: 'text-green-600 bg-green-50 border-green-200',
    emerald: 'text-emerald-600 bg-emerald-50 border-emerald-200',
  }

  return (
    <div className={`p-6 rounded-xl border-2 ${colorClasses[color as keyof typeof colorClasses]}`}>
      <div className="flex items-center justify-between mb-3">
        {icon}
        <StatusBadge status={status} />
      </div>
      <div className="text-3xl font-bold mb-1">{count}</div>
      <div className="text-sm font-medium opacity-75">{title}</div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const statusConfig = {
    active: { icon: <Play className="w-4 h-4" />, color: 'bg-blue-100 text-blue-700' },
    passed: { icon: <CheckCircle2 className="w-4 h-4" />, color: 'bg-green-100 text-green-700' },
    failed: { icon: <XCircle className="w-4 h-4" />, color: 'bg-red-100 text-red-700' },
    pending: { icon: <Clock className="w-4 h-4" />, color: 'bg-yellow-100 text-yellow-700' },
  }

  const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.pending

  return (
    <span className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${config.color}`}>
      {config.icon}
      {status}
    </span>
  )
}

function ChaosEngineeringTab({ experiments }: { experiments?: ChaosExperiment[] }) {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Chaos Engineering Experiments</h2>

      <div className="grid grid-cols-3 gap-6 mb-8">
        <ExperimentTypeCard icon={<Cpu className="w-8 h-8" />} title="Pod Failures" count={3} active={2} />
        <ExperimentTypeCard icon={<Network className="w-8 h-8" />} title="Network Chaos" count={4} active={3} />
        <ExperimentTypeCard icon={<HardDrive className="w-8 h-8" />} title="Resource Stress" count={2} active={1} />
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">Active Experiments</h3>
        {experiments?.map((exp, idx) => (
          <div key={idx} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
            <div className="flex-1">
              <h4 className="font-semibold text-gray-900">{exp.name}</h4>
              <p className="text-sm text-gray-600">{exp.type} - Duration: {exp.duration}</p>
            </div>
            <div className="flex items-center gap-4">
              <StatusBadge status={exp.status} />
              <button className="p-2 hover:bg-gray-100 rounded-lg">
                <Pause className="w-5 h-5 text-gray-600" />
              </button>
              <button className="p-2 hover:bg-gray-100 rounded-lg">
                <RotateCw className="w-5 h-5 text-gray-600" />
              </button>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-8">
        <button className="w-full py-3 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors font-medium">
          Run New Chaos Scenario
        </button>
      </div>
    </div>
  )
}

function LoadTestingTab({ results }: { results?: LoadTestResult[] }) {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Load Testing Results</h2>

      <div className="grid grid-cols-3 gap-6 mb-8">
        <LoadTestToolCard tool="k6" scenarios={3} lastRun="2 hours ago" />
        <LoadTestToolCard tool="Locust" scenarios={2} lastRun="4 hours ago" />
        <LoadTestToolCard tool="JMeter" scenarios={1} lastRun="6 hours ago" />
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">Recent Test Results</h3>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-900">Tool</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-900">Scenario</th>
                <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">VUs</th>
                <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Requests</th>
                <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Failures</th>
                <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Avg RT</th>
                <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">p95 RT</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {results?.map((result, idx) => (
                <tr key={idx} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium">{result.tool}</td>
                  <td className="px-4 py-3 text-sm text-gray-900">{result.scenario}</td>
                  <td className="px-4 py-3 text-sm text-right">{result.vus}</td>
                  <td className="px-4 py-3 text-sm text-right">{result.requests.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm text-right">
                    <span className={result.failures > 0 ? 'text-red-600' : 'text-green-600'}>
                      {result.failures}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-right">{result.avgResponseTime}ms</td>
                  <td className="px-4 py-3 text-sm text-right">{result.p95ResponseTime}ms</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function ContractTestingTab() {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Contract Testing (Pact)</h2>

      <div className="grid grid-cols-2 gap-6 mb-8">
        <div className="p-6 bg-blue-50 border border-blue-200 rounded-lg">
          <div className="text-3xl font-bold text-blue-900 mb-2">12</div>
          <div className="text-sm font-medium text-blue-700">Consumer Contracts</div>
        </div>
        <div className="p-6 bg-green-50 border border-green-200 rounded-lg">
          <div className="text-3xl font-bold text-green-900 mb-2">8</div>
          <div className="text-sm font-medium text-green-700">Provider Verifications</div>
        </div>
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">Contract Tests</h3>
        {['backup-client → backup-api', 'restore-client → backup-api', 'scheduler → backup-api'].map((contract, idx) => (
          <div key={idx} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
            <div>
              <h4 className="font-semibold text-gray-900">{contract}</h4>
              <p className="text-sm text-gray-600">Last verified: 1 hour ago</p>
            </div>
            <StatusBadge status="passed" />
          </div>
        ))}
      </div>
    </div>
  )
}

function MutationTestingTab() {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Mutation Testing</h2>

      <div className="grid grid-cols-4 gap-6 mb-8">
        <div className="p-6 bg-emerald-50 border border-emerald-200 rounded-lg">
          <div className="text-3xl font-bold text-emerald-900 mb-2">85%</div>
          <div className="text-sm font-medium text-emerald-700">Mutation Score</div>
        </div>
        <div className="p-6 bg-blue-50 border border-blue-200 rounded-lg">
          <div className="text-3xl font-bold text-blue-900 mb-2">234</div>
          <div className="text-sm font-medium text-blue-700">Total Mutations</div>
        </div>
        <div className="p-6 bg-green-50 border border-green-200 rounded-lg">
          <div className="text-3xl font-bold text-green-900 mb-2">199</div>
          <div className="text-sm font-medium text-green-700">Killed</div>
        </div>
        <div className="p-6 bg-yellow-50 border border-yellow-200 rounded-lg">
          <div className="text-3xl font-bold text-yellow-900 mb-2">35</div>
          <div className="text-sm font-medium text-yellow-700">Survived</div>
        </div>
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">Mutation Results by Package</h3>
        {[
          { pkg: 'internal/backup', score: 92, killed: 45, survived: 4 },
          { pkg: 'internal/restore', score: 88, killed: 38, survived: 5 },
          { pkg: 'internal/storage', score: 81, killed: 52, survived: 12 },
          { pkg: 'internal/encryption', score: 95, killed: 64, survived: 3 },
        ].map((result, idx) => (
          <div key={idx} className="p-4 border border-gray-200 rounded-lg">
            <div className="flex items-center justify-between mb-2">
              <h4 className="font-semibold text-gray-900">{result.pkg}</h4>
              <span className={`text-sm font-medium ${result.score >= 80 ? 'text-green-600' : 'text-red-600'}`}>
                {result.score}% score
              </span>
            </div>
            <div className="flex gap-4 text-sm text-gray-600">
              <span>Killed: {result.killed}</span>
              <span>Survived: {result.survived}</span>
            </div>
            <div className="mt-2 w-full bg-gray-200 rounded-full h-2">
              <div
                className={`h-2 rounded-full ${result.score >= 80 ? 'bg-green-600' : 'bg-red-600'}`}
                style={{ width: `${result.score}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function ExperimentTypeCard({ icon, title, count, active }: { icon: React.ReactNode; title: string; count: number; active: number }) {
  return (
    <div className="p-6 border border-gray-200 rounded-lg">
      <div className="text-purple-600 mb-3">{icon}</div>
      <div className="text-2xl font-bold text-gray-900 mb-1">{count}</div>
      <div className="text-sm font-medium text-gray-600">{title}</div>
      <div className="text-xs text-gray-500 mt-2">{active} active</div>
    </div>
  )
}

function LoadTestToolCard({ tool, scenarios, lastRun }: { tool: string; scenarios: number; lastRun: string }) {
  return (
    <div className="p-6 border border-gray-200 rounded-lg">
      <div className="text-lg font-bold text-gray-900 mb-3">{tool}</div>
      <div className="text-sm text-gray-600 mb-1">{scenarios} scenarios</div>
      <div className="text-xs text-gray-500">Last run: {lastRun}</div>
    </div>
  )
}

function TestRunCard({ run }: { run: TestRun }) {
  const typeIcons = {
    chaos: <Zap className="w-4 h-4" />,
    load: <Activity className="w-4 h-4" />,
    contract: <Shield className="w-4 h-4" />,
    mutation: <FileCode className="w-4 h-4" />,
  }

  return (
    <div className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50">
      <div className="flex items-center gap-4">
        <div className="text-gray-600">{typeIcons[run.type]}</div>
        <div>
          <h4 className="font-semibold text-gray-900">{run.name}</h4>
          <p className="text-sm text-gray-600">
            {run.startedAt && `Started ${new Date(run.startedAt).toLocaleString()}`}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-4">
        {run.duration && <span className="text-sm text-gray-600">{run.duration}ms</span>}
        <StatusBadge status={run.status} />
      </div>
    </div>
  )
}
