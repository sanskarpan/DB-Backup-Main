"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  FileCode2,
  GitBranch,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  History,
  Play,
  Pause,
  RefreshCw,
  GitPullRequest,
  Save,
  Upload,
  Download,
  Undo2,
  Eye,
  Edit3,
  Trash2,
  Plus,
} from "lucide-react"

// Types
interface BackupConfig {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    version: string
    created_at: string
    updated_at: string
    labels?: Record<string, string>
  }
  spec: {
    databases: DatabaseConfig[]
    schedules: ScheduleConfig[]
    retention: {
      daily: number
      weekly: number
      monthly: number
      yearly: number
    }
  }
  status?: {
    phase: string
    last_applied: string
    active_backups: number
    failed_backups: number
  }
}

interface DatabaseConfig {
  id: string
  name: string
  type: string
  enabled: boolean
}

interface ScheduleConfig {
  name: string
  type: string
  cron: string
  enabled: boolean
}

interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
  warnings: string[]
}

interface ValidationError {
  field: string
  message: string
  code: string
}

interface GitOpsStatus {
  running: boolean
  last_sync: string
  last_sync_status: string
  last_sync_duration: string
  last_commit_hash: string
  repository: string
  branch: string
  auto_apply: boolean
}

interface SyncRecord {
  timestamp: string
  commit_hash: string
  status: string
  configs_added: number
  configs_updated: number
  configs_deleted: number
  duration: string
  errors: string[]
}

interface ConfigVersion {
  version: string
  created_at: string
  created_by: string
  change_note: string
  tags: string[]
}

interface RollbackPlan {
  config_name: string
  current_version: string
  target_version: string
  changes: VersionChange[]
  impact_analysis: {
    affected_databases: string[]
    affected_schedules: string[]
    risk_level: string
    warnings: string[]
  }
}

interface VersionChange {
  type: string
  path: string
  old_value?: any
  new_value?: any
}

export default function BackupAsCodeDashboard() {
  const [configs, setConfigs] = useState<BackupConfig[]>([])
  const [selectedConfig, setSelectedConfig] = useState<BackupConfig | null>(null)
  const [configYaml, setConfigYaml] = useState<string>("")
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null)
  const [gitopsStatus, setGitopsStatus] = useState<GitOpsStatus | null>(null)
  const [syncHistory, setSyncHistory] = useState<SyncRecord[]>([])
  const [versions, setVersions] = useState<ConfigVersion[]>([])
  const [rollbackPlan, setRollbackPlan] = useState<RollbackPlan | null>(null)
  const [activeTab, setActiveTab] = useState("configs")
  const [editMode, setEditMode] = useState(false)

  useEffect(() => {
    loadConfigs()
    loadGitOpsStatus()
    loadSyncHistory()
  }, [])

  const loadConfigs = () => {
    // Mock data - replace with actual API call
    const mockConfigs: BackupConfig[] = [
      {
        apiVersion: "v1",
        kind: "BackupConfig",
        metadata: {
          name: "production-postgres",
          version: "1.2.0",
          created_at: "2024-01-15T10:00:00Z",
          updated_at: "2024-01-20T15:30:00Z",
          labels: {
            env: "production",
            team: "platform",
          },
        },
        spec: {
          databases: [
            { id: "pg-prod-01", name: "users-db", type: "postgres", enabled: true },
            { id: "pg-prod-02", name: "orders-db", type: "postgres", enabled: true },
          ],
          schedules: [
            { name: "daily-full", type: "full", cron: "0 2 * * *", enabled: true },
            { name: "hourly-incr", type: "incremental", cron: "0 * * * *", enabled: true },
          ],
          retention: {
            daily: 7,
            weekly: 4,
            monthly: 12,
            yearly: 5,
          },
        },
        status: {
          phase: "Active",
          last_applied: "2024-01-20T15:30:00Z",
          active_backups: 145,
          failed_backups: 2,
        },
      },
      {
        apiVersion: "v1",
        kind: "BackupConfig",
        metadata: {
          name: "staging-mongodb",
          version: "1.0.0",
          created_at: "2024-01-10T08:00:00Z",
          updated_at: "2024-01-18T12:00:00Z",
          labels: {
            env: "staging",
            team: "backend",
          },
        },
        spec: {
          databases: [
            { id: "mongo-stg-01", name: "analytics-db", type: "mongodb", enabled: true },
          ],
          schedules: [
            { name: "nightly-full", type: "full", cron: "0 3 * * *", enabled: true },
          ],
          retention: {
            daily: 3,
            weekly: 2,
            monthly: 6,
            yearly: 0,
          },
        },
        status: {
          phase: "Active",
          last_applied: "2024-01-18T12:00:00Z",
          active_backups: 45,
          failed_backups: 0,
        },
      },
    ]
    setConfigs(mockConfigs)
  }

  const loadGitOpsStatus = () => {
    const mockStatus: GitOpsStatus = {
      running: true,
      last_sync: "2024-01-20T16:00:00Z",
      last_sync_status: "success",
      last_sync_duration: "3.2s",
      last_commit_hash: "a7f3c91",
      repository: "https://github.com/company/backup-configs",
      branch: "main",
      auto_apply: true,
    }
    setGitopsStatus(mockStatus)
  }

  const loadSyncHistory = () => {
    const mockHistory: SyncRecord[] = [
      {
        timestamp: "2024-01-20T16:00:00Z",
        commit_hash: "a7f3c91",
        status: "success",
        configs_added: 0,
        configs_updated: 1,
        configs_deleted: 0,
        duration: "3.2s",
        errors: [],
      },
      {
        timestamp: "2024-01-20T15:30:00Z",
        commit_hash: "b2e4d56",
        status: "success",
        configs_added: 1,
        configs_updated: 0,
        configs_deleted: 0,
        duration: "2.8s",
        errors: [],
      },
      {
        timestamp: "2024-01-20T15:00:00Z",
        commit_hash: "c9f1a32",
        status: "partial_success",
        configs_added: 0,
        configs_updated: 1,
        configs_deleted: 0,
        duration: "4.1s",
        errors: ["Warning: Validation warning for staging-mongodb"],
      },
    ]
    setSyncHistory(mockHistory)
  }

  const loadVersions = (configName: string) => {
    const mockVersions: ConfigVersion[] = [
      {
        version: "v3",
        created_at: "2024-01-20T15:30:00Z",
        created_by: "alice@company.com",
        change_note: "Increased retention periods for compliance",
        tags: ["stable"],
      },
      {
        version: "v2",
        created_at: "2024-01-18T10:00:00Z",
        created_by: "bob@company.com",
        change_note: "Added new database to backup",
        tags: [],
      },
      {
        version: "v1",
        created_at: "2024-01-15T10:00:00Z",
        created_by: "alice@company.com",
        change_note: "Initial configuration",
        tags: ["initial"],
      },
    ]
    setVersions(mockVersions)
  }

  const validateConfig = (yaml: string) => {
    // Mock validation - replace with actual API call
    const mockResult: ValidationResult = {
      valid: true,
      errors: [],
      warnings: [
        "spec.retention.yearly: 0 may result in no long-term backups",
      ],
    }
    setValidationResult(mockResult)
  }

  const handleConfigSelect = (config: BackupConfig) => {
    setSelectedConfig(config)
    setConfigYaml(JSON.stringify(config, null, 2))
    loadVersions(config.metadata.name)
    setEditMode(false)
  }

  const handleValidate = () => {
    validateConfig(configYaml)
  }

  const handleSave = () => {
    // Save logic here
    setEditMode(false)
  }

  const handleSync = () => {
    // Trigger GitOps sync
    loadGitOpsStatus()
    loadSyncHistory()
  }

  const createRollbackPlan = (targetVersion: string) => {
    const mockPlan: RollbackPlan = {
      config_name: selectedConfig?.metadata.name || "",
      current_version: "v3",
      target_version: targetVersion,
      changes: [
        {
          type: "modified",
          path: "spec.retention.daily",
          old_value: 7,
          new_value: 3,
        },
      ],
      impact_analysis: {
        affected_databases: ["users-db", "orders-db"],
        affected_schedules: ["daily-full"],
        risk_level: "medium",
        warnings: ["Retention period will be reduced"],
      },
    }
    setRollbackPlan(mockPlan)
  }

  const getStatusBadge = (phase: string) => {
    const variants: Record<string, { variant: "default" | "secondary" | "destructive" | "outline", icon: any }> = {
      Active: { variant: "default", icon: CheckCircle2 },
      Pending: { variant: "secondary", icon: Clock },
      Failed: { variant: "destructive", icon: XCircle },
    }
    const config = variants[phase] || variants.Pending
    const Icon = config.icon
    return (
      <Badge variant={config.variant} className="gap-1">
        <Icon className="h-3 w-3" />
        {phase}
      </Badge>
    )
  }

  const getRiskBadge = (level: string) => {
    const variants: Record<string, "default" | "secondary" | "destructive"> = {
      low: "secondary",
      medium: "default",
      high: "destructive",
    }
    return (
      <Badge variant={variants[level] || "default"}>
        {level.toUpperCase()}
      </Badge>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Backup as Code</h1>
        <p className="text-muted-foreground mt-2">
          Declarative backup configuration with GitOps workflow
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList>
          <TabsTrigger value="configs">Configurations</TabsTrigger>
          <TabsTrigger value="editor">Policy Editor</TabsTrigger>
          <TabsTrigger value="gitops">GitOps</TabsTrigger>
          <TabsTrigger value="versions">Versions</TabsTrigger>
          <TabsTrigger value="validation">Validation</TabsTrigger>
        </TabsList>

        {/* Configurations Tab */}
        <TabsContent value="configs" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Backup Configurations</CardTitle>
                  <CardDescription>
                    Manage declarative backup configurations
                  </CardDescription>
                </div>
                <Button className="gap-2">
                  <Plus className="h-4 w-4" />
                  New Config
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Databases</TableHead>
                    <TableHead>Schedules</TableHead>
                    <TableHead>Backups</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {configs.map((config) => (
                    <TableRow
                      key={config.metadata.name}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => handleConfigSelect(config)}
                    >
                      <TableCell className="font-medium">
                        <div>
                          {config.metadata.name}
                          <div className="flex gap-1 mt-1">
                            {Object.entries(config.metadata.labels || {}).map(([key, value]) => (
                              <Badge key={key} variant="outline" className="text-xs">
                                {key}:{value}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>{config.metadata.version}</TableCell>
                      <TableCell>{getStatusBadge(config.status?.phase || "Pending")}</TableCell>
                      <TableCell>{config.spec.databases.length}</TableCell>
                      <TableCell>{config.spec.schedules.length}</TableCell>
                      <TableCell>
                        <div className="text-sm">
                          <div className="text-green-600">{config.status?.active_backups || 0} active</div>
                          {(config.status?.failed_backups || 0) > 0 && (
                            <div className="text-red-600">{config.status?.failed_backups} failed</div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        {new Date(config.metadata.updated_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button size="sm" variant="ghost">
                            <Eye className="h-4 w-4" />
                          </Button>
                          <Button size="sm" variant="ghost">
                            <Edit3 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {selectedConfig && (
            <Card>
              <CardHeader>
                <CardTitle>Configuration Details: {selectedConfig.metadata.name}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label className="text-sm font-medium">Databases</Label>
                    <div className="mt-2 space-y-2">
                      {selectedConfig.spec.databases.map((db) => (
                        <div key={db.id} className="flex items-center gap-2 p-2 border rounded">
                          <Badge variant={db.enabled ? "default" : "secondary"}>
                            {db.type}
                          </Badge>
                          <span>{db.name}</span>
                          {db.enabled && <CheckCircle2 className="h-4 w-4 text-green-600 ml-auto" />}
                        </div>
                      ))}
                    </div>
                  </div>
                  <div>
                    <Label className="text-sm font-medium">Schedules</Label>
                    <div className="mt-2 space-y-2">
                      {selectedConfig.spec.schedules.map((sched) => (
                        <div key={sched.name} className="p-2 border rounded">
                          <div className="flex items-center justify-between">
                            <span className="font-medium">{sched.name}</span>
                            <Badge variant="outline">{sched.type}</Badge>
                          </div>
                          <div className="text-sm text-muted-foreground mt-1">
                            {sched.cron}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>

                <div>
                  <Label className="text-sm font-medium">Retention Policy</Label>
                  <div className="grid grid-cols-4 gap-4 mt-2">
                    <div className="p-3 border rounded text-center">
                      <div className="text-2xl font-bold">{selectedConfig.spec.retention.daily}</div>
                      <div className="text-sm text-muted-foreground">Daily</div>
                    </div>
                    <div className="p-3 border rounded text-center">
                      <div className="text-2xl font-bold">{selectedConfig.spec.retention.weekly}</div>
                      <div className="text-sm text-muted-foreground">Weekly</div>
                    </div>
                    <div className="p-3 border rounded text-center">
                      <div className="text-2xl font-bold">{selectedConfig.spec.retention.monthly}</div>
                      <div className="text-sm text-muted-foreground">Monthly</div>
                    </div>
                    <div className="p-3 border rounded text-center">
                      <div className="text-2xl font-bold">{selectedConfig.spec.retention.yearly}</div>
                      <div className="text-sm text-muted-foreground">Yearly</div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Policy Editor Tab */}
        <TabsContent value="editor" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Policy Editor</CardTitle>
                  <CardDescription>
                    Edit backup configuration in YAML/JSON format
                  </CardDescription>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" className="gap-2">
                    <Upload className="h-4 w-4" />
                    Import
                  </Button>
                  <Button variant="outline" className="gap-2">
                    <Download className="h-4 w-4" />
                    Export
                  </Button>
                  {editMode ? (
                    <>
                      <Button variant="outline" onClick={() => setEditMode(false)}>
                        Cancel
                      </Button>
                      <Button onClick={handleSave} className="gap-2">
                        <Save className="h-4 w-4" />
                        Save
                      </Button>
                    </>
                  ) : (
                    <Button onClick={() => setEditMode(true)} className="gap-2">
                      <Edit3 className="h-4 w-4" />
                      Edit
                    </Button>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Textarea
                  value={configYaml}
                  onChange={(e) => setConfigYaml(e.target.value)}
                  className="font-mono text-sm min-h-[400px]"
                  readOnly={!editMode}
                />
              </div>
              <div className="flex justify-end">
                <Button onClick={handleValidate} variant="outline" className="gap-2">
                  <CheckCircle2 className="h-4 w-4" />
                  Validate
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* GitOps Tab */}
        <TabsContent value="gitops" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>GitOps Status</CardTitle>
                  <CardDescription>
                    Git repository synchronization and automation
                  </CardDescription>
                </div>
                <Button onClick={handleSync} className="gap-2">
                  <RefreshCw className="h-4 w-4" />
                  Sync Now
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {gitopsStatus && (
                <>
                  <div className="grid grid-cols-2 gap-6">
                    <div className="space-y-4">
                      <div>
                        <Label className="text-sm text-muted-foreground">Status</Label>
                        <div className="flex items-center gap-2 mt-1">
                          {gitopsStatus.running ? (
                            <Badge variant="default" className="gap-1">
                              <Play className="h-3 w-3" />
                              Running
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="gap-1">
                              <Pause className="h-3 w-3" />
                              Stopped
                            </Badge>
                          )}
                        </div>
                      </div>
                      <div>
                        <Label className="text-sm text-muted-foreground">Repository</Label>
                        <div className="flex items-center gap-2 mt-1">
                          <GitBranch className="h-4 w-4 text-muted-foreground" />
                          <span className="text-sm font-mono">{gitopsStatus.repository}</span>
                        </div>
                      </div>
                      <div>
                        <Label className="text-sm text-muted-foreground">Branch</Label>
                        <div className="text-sm mt-1">{gitopsStatus.branch}</div>
                      </div>
                    </div>

                    <div className="space-y-4">
                      <div>
                        <Label className="text-sm text-muted-foreground">Last Sync</Label>
                        <div className="text-sm mt-1">
                          {new Date(gitopsStatus.last_sync).toLocaleString()}
                        </div>
                      </div>
                      <div>
                        <Label className="text-sm text-muted-foreground">Last Commit</Label>
                        <div className="flex items-center gap-2 mt-1">
                          <Badge variant="outline" className="font-mono">
                            {gitopsStatus.last_commit_hash}
                          </Badge>
                          <Badge variant={gitopsStatus.last_sync_status === "success" ? "default" : "destructive"}>
                            {gitopsStatus.last_sync_status}
                          </Badge>
                        </div>
                      </div>
                      <div>
                        <Label className="text-sm text-muted-foreground">Auto Apply</Label>
                        <div className="text-sm mt-1">
                          {gitopsStatus.auto_apply ? "Enabled" : "Disabled"}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="text-sm font-medium mb-3">Sync History</h3>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Timestamp</TableHead>
                          <TableHead>Commit</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Changes</TableHead>
                          <TableHead>Duration</TableHead>
                          <TableHead>Details</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {syncHistory.map((record, idx) => (
                          <TableRow key={idx}>
                            <TableCell>{new Date(record.timestamp).toLocaleString()}</TableCell>
                            <TableCell>
                              <Badge variant="outline" className="font-mono">
                                {record.commit_hash}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant={record.status === "success" ? "default" : record.status === "partial_success" ? "secondary" : "destructive"}>
                                {record.status}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="text-sm space-y-1">
                                {record.configs_added > 0 && (
                                  <div className="text-green-600">+{record.configs_added} added</div>
                                )}
                                {record.configs_updated > 0 && (
                                  <div className="text-blue-600">~{record.configs_updated} updated</div>
                                )}
                                {record.configs_deleted > 0 && (
                                  <div className="text-red-600">-{record.configs_deleted} deleted</div>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>{record.duration}</TableCell>
                            <TableCell>
                              {record.errors.length > 0 && (
                                <Badge variant="destructive" className="gap-1">
                                  <AlertTriangle className="h-3 w-3" />
                                  {record.errors.length} errors
                                </Badge>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Versions Tab */}
        <TabsContent value="versions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Version History</CardTitle>
              <CardDescription>
                Configuration versions and rollback management
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {selectedConfig ? (
                <>
                  <div className="text-sm text-muted-foreground mb-4">
                    Viewing versions for: <strong>{selectedConfig.metadata.name}</strong>
                  </div>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Version</TableHead>
                        <TableHead>Created</TableHead>
                        <TableHead>Author</TableHead>
                        <TableHead>Change Note</TableHead>
                        <TableHead>Tags</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {versions.map((version) => (
                        <TableRow key={version.version}>
                          <TableCell>
                            <Badge variant="outline">{version.version}</Badge>
                          </TableCell>
                          <TableCell>{new Date(version.created_at).toLocaleDateString()}</TableCell>
                          <TableCell>{version.created_by}</TableCell>
                          <TableCell>{version.change_note}</TableCell>
                          <TableCell>
                            <div className="flex gap-1">
                              {version.tags.map((tag) => (
                                <Badge key={tag} variant="secondary">{tag}</Badge>
                              ))}
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex gap-2">
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => createRollbackPlan(version.version)}
                                className="gap-1"
                              >
                                <Undo2 className="h-3 w-3" />
                                Rollback
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  Select a configuration to view version history
                </div>
              )}

              {rollbackPlan && (
                <Card className="border-orange-200 bg-orange-50">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <AlertTriangle className="h-5 w-5 text-orange-600" />
                      Rollback Plan
                    </CardTitle>
                    <CardDescription>
                      Review the impact before rolling back
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label className="text-sm">Current Version</Label>
                        <div className="font-mono mt-1">{rollbackPlan.current_version}</div>
                      </div>
                      <div>
                        <Label className="text-sm">Target Version</Label>
                        <div className="font-mono mt-1">{rollbackPlan.target_version}</div>
                      </div>
                    </div>

                    <div>
                      <Label className="text-sm">Risk Level</Label>
                      <div className="mt-1">
                        {getRiskBadge(rollbackPlan.impact_analysis.risk_level)}
                      </div>
                    </div>

                    <div>
                      <Label className="text-sm">Affected Resources</Label>
                      <div className="mt-2 space-y-2">
                        <div className="text-sm">
                          <strong>Databases:</strong> {rollbackPlan.impact_analysis.affected_databases.join(", ")}
                        </div>
                        <div className="text-sm">
                          <strong>Schedules:</strong> {rollbackPlan.impact_analysis.affected_schedules.join(", ")}
                        </div>
                      </div>
                    </div>

                    {rollbackPlan.impact_analysis.warnings.length > 0 && (
                      <div>
                        <Label className="text-sm">Warnings</Label>
                        <div className="mt-2 space-y-1">
                          {rollbackPlan.impact_analysis.warnings.map((warning, idx) => (
                            <div key={idx} className="flex items-start gap-2 text-sm text-orange-700">
                              <AlertTriangle className="h-4 w-4 mt-0.5" />
                              {warning}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    <div className="flex gap-2 pt-4">
                      <Button variant="outline" onClick={() => setRollbackPlan(null)}>
                        Cancel
                      </Button>
                      <Button variant="destructive">
                        Confirm Rollback
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Validation Tab */}
        <TabsContent value="validation" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Configuration Validation</CardTitle>
              <CardDescription>
                Validate backup policies against schema and best practices
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {validationResult ? (
                <>
                  <div className="flex items-center gap-2">
                    {validationResult.valid ? (
                      <Badge variant="default" className="gap-1">
                        <CheckCircle2 className="h-4 w-4" />
                        Valid Configuration
                      </Badge>
                    ) : (
                      <Badge variant="destructive" className="gap-1">
                        <XCircle className="h-4 w-4" />
                        Invalid Configuration
                      </Badge>
                    )}
                  </div>

                  {validationResult.errors.length > 0 && (
                    <div>
                      <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                        <XCircle className="h-4 w-4 text-red-600" />
                        Errors ({validationResult.errors.length})
                      </h3>
                      <div className="space-y-2">
                        {validationResult.errors.map((error, idx) => (
                          <div key={idx} className="p-3 border border-red-200 rounded bg-red-50">
                            <div className="flex items-start gap-2">
                              <Badge variant="destructive">{error.code}</Badge>
                              <div className="flex-1">
                                <div className="text-sm font-medium">{error.field}</div>
                                <div className="text-sm text-muted-foreground mt-1">
                                  {error.message}
                                </div>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {validationResult.warnings.length > 0 && (
                    <div>
                      <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                        <AlertTriangle className="h-4 w-4 text-orange-600" />
                        Warnings ({validationResult.warnings.length})
                      </h3>
                      <div className="space-y-2">
                        {validationResult.warnings.map((warning, idx) => (
                          <div key={idx} className="p-3 border border-orange-200 rounded bg-orange-50">
                            <div className="text-sm">{warning}</div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  Click "Validate" in the Policy Editor to check your configuration
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
