/**
 * Team Collaboration Features - Core Managers
 * Comprehensive system for team collaboration, sharing, permissions, and workflows
 */

// ==================== Types & Interfaces ====================

export interface Team {
  id: string
  name: string
  description?: string
  avatar?: string
  createdAt: Date
  createdBy: string
  settings: TeamSettings
  members: TeamMember[]
  isActive: boolean
}

export interface TeamSettings {
  allowMemberInvites: boolean
  requireApprovalForRestores: boolean
  defaultPermissionLevel: PermissionLevel
  notificationPreferences: NotificationPreferences
  retentionPolicy?: string
  autoShareBackups: boolean
}

export interface NotificationPreferences {
  emailNotifications: boolean
  pushNotifications: boolean
  inAppNotifications: boolean
  mentionNotifications: boolean
  digestFrequency: 'realtime' | 'hourly' | 'daily' | 'weekly'
}

export interface TeamMember {
  userId: string
  username: string
  email: string
  displayName: string
  avatar?: string
  role: TeamRole
  permissions: PermissionLevel
  joinedAt: Date
  lastActive?: Date
  status: 'active' | 'inactive' | 'invited' | 'suspended'
}

export type TeamRole = 'owner' | 'admin' | 'member'
export type PermissionLevel = 'viewer' | 'operator' | 'admin'

export interface SharedSchedule {
  id: string
  teamId: string
  name: string
  description?: string
  schedule: ScheduleConfig
  databases: string[]
  sharedWith: SharePermission[]
  createdBy: string
  createdAt: Date
  updatedAt?: Date
  isActive: boolean
  lastRun?: Date
  nextRun?: Date
}

export interface ScheduleConfig {
  type: 'cron' | 'interval' | 'manual'
  cronExpression?: string
  intervalMinutes?: number
  timezone: string
  enabled: boolean
}

export interface SharePermission {
  userId: string
  permissionLevel: 'view' | 'edit' | 'execute' | 'admin'
  grantedBy: string
  grantedAt: Date
  expiresAt?: Date
}

export interface BackupShare {
  id: string
  backupId: string
  backupName: string
  teamId: string
  sharedBy: string
  sharedAt: Date
  permissions: SharePermission[]
  expiresAt?: Date
  accessCount: number
  isActive: boolean
  shareUrl?: string
  requiresApproval: boolean
}

export interface Comment {
  id: string
  resourceType: 'backup' | 'restore' | 'schedule' | 'team'
  resourceId: string
  teamId: string
  authorId: string
  authorName: string
  authorAvatar?: string
  content: string
  mentions: Mention[]
  attachments?: Attachment[]
  createdAt: Date
  updatedAt?: Date
  editedAt?: Date
  isEdited: boolean
  isDeleted: boolean
  parentId?: string // for threaded comments
  reactions: Reaction[]
}

export interface Mention {
  userId: string
  username: string
  displayName: string
  position: number // character position in content
}

export interface Attachment {
  id: string
  filename: string
  filesize: number
  mimeType: string
  url: string
  uploadedAt: Date
}

export interface Reaction {
  emoji: string
  userId: string
  username: string
  createdAt: Date
}

export interface TeamNotification {
  id: string
  teamId: string
  type: NotificationType
  title: string
  message: string
  priority: 'low' | 'medium' | 'high' | 'urgent'
  category: NotificationCategory
  recipients: string[] // user IDs
  mentions: Mention[]
  metadata: Record<string, any>
  createdBy: string
  createdAt: Date
  readBy: string[] // user IDs who have read it
  dismissedBy: string[] // user IDs who dismissed it
  actionUrl?: string
  actionLabel?: string
  expiresAt?: Date
}

export type NotificationType =
  | 'backup_complete'
  | 'backup_failed'
  | 'restore_requested'
  | 'restore_approved'
  | 'restore_rejected'
  | 'schedule_shared'
  | 'comment_added'
  | 'mention'
  | 'permission_changed'
  | 'team_invite'
  | 'audit_alert'
  | 'custom'

export type NotificationCategory =
  | 'backup'
  | 'restore'
  | 'schedule'
  | 'permission'
  | 'team'
  | 'audit'
  | 'system'

export interface ActivityEvent {
  id: string
  teamId: string
  actorId: string
  actorName: string
  actorAvatar?: string
  action: ActivityAction
  resourceType: 'backup' | 'restore' | 'schedule' | 'team' | 'permission' | 'comment'
  resourceId: string
  resourceName: string
  details: Record<string, any>
  timestamp: Date
  visibility: 'team' | 'admins' | 'private'
  tags: string[]
}

export type ActivityAction =
  | 'created'
  | 'updated'
  | 'deleted'
  | 'shared'
  | 'unshared'
  | 'executed'
  | 'approved'
  | 'rejected'
  | 'commented'
  | 'mentioned'
  | 'invited'
  | 'removed'
  | 'promoted'
  | 'demoted'

export interface AuditLogEntry {
  id: string
  teamId: string
  timestamp: Date
  actorId: string
  actorName: string
  actorEmail: string
  actorIp?: string
  action: string
  resourceType: string
  resourceId: string
  resourceName?: string
  changes?: AuditChange[]
  status: 'success' | 'failure' | 'warning'
  severity: 'info' | 'warning' | 'critical'
  metadata: Record<string, any>
  userAgent?: string
  sessionId?: string
}

export interface AuditChange {
  field: string
  oldValue: any
  newValue: any
  timestamp: Date
}

export interface ApprovalRequest {
  id: string
  teamId: string
  type: 'restore' | 'delete' | 'permission_change' | 'custom'
  title: string
  description: string
  requestedBy: string
  requestedByName: string
  requestedAt: Date
  resourceType: string
  resourceId: string
  resourceName: string
  approvers: Approver[]
  requiredApprovals: number
  currentApprovals: number
  status: 'pending' | 'approved' | 'rejected' | 'cancelled' | 'expired'
  expiresAt?: Date
  completedAt?: Date
  completedBy?: string
  metadata: Record<string, any>
}

export interface Approver {
  userId: string
  username: string
  displayName: string
  status: 'pending' | 'approved' | 'rejected'
  respondedAt?: Date
  comment?: string
  order: number // for sequential approval
}

export interface CalendarEvent {
  id: string
  teamId: string
  type: 'scheduled_backup' | 'maintenance' | 'deadline' | 'meeting' | 'custom'
  title: string
  description?: string
  startTime: Date
  endTime?: Date
  duration?: number // in minutes
  schedule?: SharedSchedule
  location?: string
  attendees: string[] // user IDs
  reminders: Reminder[]
  recurrence?: RecurrenceRule
  color?: string
  isAllDay: boolean
  createdBy: string
  createdAt: Date
  updatedAt?: Date
}

export interface Reminder {
  type: 'email' | 'push' | 'sms'
  minutesBefore: number
  sentAt?: Date
}

export interface RecurrenceRule {
  frequency: 'daily' | 'weekly' | 'monthly' | 'yearly'
  interval: number
  endDate?: Date
  count?: number // number of occurrences
  byDay?: string[] // ['MO', 'TU', 'WE', 'TH', 'FR']
  byMonthDay?: number[]
}

export interface TeamPresence {
  userId: string
  username: string
  status: 'online' | 'away' | 'busy' | 'offline'
  lastSeen: Date
  currentView?: string
  currentResource?: string
}

export interface CollaborativeDashboard {
  id: string
  teamId: string
  name: string
  description?: string
  layout: DashboardLayout
  widgets: DashboardWidget[]
  sharedWith: string[] // user IDs
  isPublic: boolean
  createdBy: string
  createdAt: Date
  updatedAt?: Date
}

export interface DashboardLayout {
  columns: number
  rowHeight: number
  breakpoints?: Record<string, number>
}

export interface DashboardWidget {
  id: string
  type: string
  title: string
  config: Record<string, any>
  position: { x: number; y: number; w: number; h: number }
  permissions: {
    canView: string[]
    canEdit: string[]
  }
}

// ==================== Team Manager ====================

export class TeamManager {
  private teams: Map<string, Team> = new Map()
  private currentTeam: string | null = null
  private listeners: Set<(team: Team) => void> = new Set()

  constructor() {
    this.loadTeams()
  }

  createTeam(name: string, createdBy: string, settings?: Partial<TeamSettings>): Team {
    const team: Team = {
      id: `team-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      name,
      description: '',
      createdAt: new Date(),
      createdBy,
      settings: {
        allowMemberInvites: true,
        requireApprovalForRestores: true,
        defaultPermissionLevel: 'operator',
        notificationPreferences: {
          emailNotifications: true,
          pushNotifications: true,
          inAppNotifications: true,
          mentionNotifications: true,
          digestFrequency: 'realtime'
        },
        autoShareBackups: false,
        ...settings
      },
      members: [
        {
          userId: createdBy,
          username: createdBy,
          email: `${createdBy}@example.com`,
          displayName: createdBy,
          role: 'owner',
          permissions: 'admin',
          joinedAt: new Date(),
          status: 'active'
        }
      ],
      isActive: true
    }

    this.teams.set(team.id, team)
    this.saveTeams()
    this.notifyListeners(team)
    return team
  }

  getTeam(teamId: string): Team | undefined {
    return this.teams.get(teamId)
  }

  getAllTeams(): Team[] {
    return Array.from(this.teams.values())
  }

  updateTeam(teamId: string, updates: Partial<Team>): boolean {
    const team = this.teams.get(teamId)
    if (!team) return false

    Object.assign(team, updates)
    this.saveTeams()
    this.notifyListeners(team)
    return true
  }

  deleteTeam(teamId: string): boolean {
    const deleted = this.teams.delete(teamId)
    if (deleted) {
      this.saveTeams()
      if (this.currentTeam === teamId) {
        this.currentTeam = null
      }
    }
    return deleted
  }

  setCurrentTeam(teamId: string): boolean {
    if (!this.teams.has(teamId)) return false
    this.currentTeam = teamId
    this.saveCurrentTeam()
    return true
  }

  getCurrentTeam(): Team | null {
    return this.currentTeam ? this.teams.get(this.currentTeam) || null : null
  }

  addMember(teamId: string, member: Omit<TeamMember, 'joinedAt' | 'status'>): boolean {
    const team = this.teams.get(teamId)
    if (!team) return false

    const newMember: TeamMember = {
      ...member,
      joinedAt: new Date(),
      status: 'invited'
    }

    team.members.push(newMember)
    this.saveTeams()
    this.notifyListeners(team)
    return true
  }

  removeMember(teamId: string, userId: string): boolean {
    const team = this.teams.get(teamId)
    if (!team) return false

    team.members = team.members.filter(m => m.userId !== userId)
    this.saveTeams()
    this.notifyListeners(team)
    return true
  }

  updateMemberRole(teamId: string, userId: string, role: TeamRole, permissions: PermissionLevel): boolean {
    const team = this.teams.get(teamId)
    if (!team) return false

    const member = team.members.find(m => m.userId === userId)
    if (!member) return false

    member.role = role
    member.permissions = permissions
    this.saveTeams()
    this.notifyListeners(team)
    return true
  }

  getMemberPermissions(teamId: string, userId: string): PermissionLevel | null {
    const team = this.teams.get(teamId)
    if (!team) return null

    const member = team.members.find(m => m.userId === userId)
    return member?.permissions || null
  }

  subscribe(listener: (team: Team) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(team: Team): void {
    this.listeners.forEach(listener => listener(team))
  }

  private loadTeams(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-teams')
    if (stored) {
      try {
        const teams = JSON.parse(stored)
        teams.forEach((team: Team) => {
          // Convert date strings back to Date objects
          team.createdAt = new Date(team.createdAt)
          team.members.forEach(m => {
            m.joinedAt = new Date(m.joinedAt)
            if (m.lastActive) m.lastActive = new Date(m.lastActive)
          })
          this.teams.set(team.id, team)
        })
      } catch (e) {
        console.error('Failed to load teams:', e)
      }
    }

    const currentTeam = localStorage.getItem('collaboration-current-team')
    if (currentTeam) {
      this.currentTeam = currentTeam
    }
  }

  private saveTeams(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('collaboration-teams', JSON.stringify(Array.from(this.teams.values())))
  }

  private saveCurrentTeam(): void {
    if (typeof window === 'undefined') return
    if (this.currentTeam) {
      localStorage.setItem('collaboration-current-team', this.currentTeam)
    } else {
      localStorage.removeItem('collaboration-current-team')
    }
  }

  exportTeam(teamId: string): string {
    const team = this.teams.get(teamId)
    if (!team) throw new Error('Team not found')
    return JSON.stringify(team, null, 2)
  }

  importTeam(teamJson: string): Team {
    const team = JSON.parse(teamJson) as Team
    team.createdAt = new Date(team.createdAt)
    team.members.forEach(m => {
      m.joinedAt = new Date(m.joinedAt)
      if (m.lastActive) m.lastActive = new Date(m.lastActive)
    })
    this.teams.set(team.id, team)
    this.saveTeams()
    return team
  }
}

// ==================== Shared Schedule Manager ====================

export class SharedScheduleManager {
  private schedules: Map<string, SharedSchedule> = new Map()
  private listeners: Set<(schedule: SharedSchedule) => void> = new Set()

  constructor() {
    this.loadSchedules()
  }

  createSchedule(
    teamId: string,
    name: string,
    schedule: ScheduleConfig,
    databases: string[],
    createdBy: string
  ): SharedSchedule {
    const sharedSchedule: SharedSchedule = {
      id: `schedule-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      teamId,
      name,
      schedule,
      databases,
      sharedWith: [],
      createdBy,
      createdAt: new Date(),
      isActive: true
    }

    this.schedules.set(sharedSchedule.id, sharedSchedule)
    this.saveSchedules()
    this.notifyListeners(sharedSchedule)
    return sharedSchedule
  }

  getSchedule(scheduleId: string): SharedSchedule | undefined {
    return this.schedules.get(scheduleId)
  }

  getTeamSchedules(teamId: string): SharedSchedule[] {
    return Array.from(this.schedules.values()).filter(s => s.teamId === teamId)
  }

  updateSchedule(scheduleId: string, updates: Partial<SharedSchedule>): boolean {
    const schedule = this.schedules.get(scheduleId)
    if (!schedule) return false

    Object.assign(schedule, updates, { updatedAt: new Date() })
    this.saveSchedules()
    this.notifyListeners(schedule)
    return true
  }

  deleteSchedule(scheduleId: string): boolean {
    const deleted = this.schedules.delete(scheduleId)
    if (deleted) this.saveSchedules()
    return deleted
  }

  shareWithUser(scheduleId: string, userId: string, permission: SharePermission): boolean {
    const schedule = this.schedules.get(scheduleId)
    if (!schedule) return false

    schedule.sharedWith.push(permission)
    schedule.updatedAt = new Date()
    this.saveSchedules()
    this.notifyListeners(schedule)
    return true
  }

  revokeAccess(scheduleId: string, userId: string): boolean {
    const schedule = this.schedules.get(scheduleId)
    if (!schedule) return false

    schedule.sharedWith = schedule.sharedWith.filter(p => p.userId !== userId)
    schedule.updatedAt = new Date()
    this.saveSchedules()
    this.notifyListeners(schedule)
    return true
  }

  subscribe(listener: (schedule: SharedSchedule) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(schedule: SharedSchedule): void {
    this.listeners.forEach(listener => listener(schedule))
  }

  private loadSchedules(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-schedules')
    if (stored) {
      try {
        const schedules = JSON.parse(stored)
        schedules.forEach((schedule: SharedSchedule) => {
          schedule.createdAt = new Date(schedule.createdAt)
          if (schedule.updatedAt) schedule.updatedAt = new Date(schedule.updatedAt)
          if (schedule.lastRun) schedule.lastRun = new Date(schedule.lastRun)
          if (schedule.nextRun) schedule.nextRun = new Date(schedule.nextRun)
          schedule.sharedWith.forEach(p => {
            p.grantedAt = new Date(p.grantedAt)
            if (p.expiresAt) p.expiresAt = new Date(p.expiresAt)
          })
          this.schedules.set(schedule.id, schedule)
        })
      } catch (e) {
        console.error('Failed to load schedules:', e)
      }
    }
  }

  private saveSchedules(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('collaboration-schedules', JSON.stringify(Array.from(this.schedules.values())))
  }
}

// ==================== Team Notification Manager ====================

export class TeamNotificationManager {
  private notifications: TeamNotification[] = []
  private listeners: Set<(notification: TeamNotification) => void> = new Set()

  constructor() {
    this.loadNotifications()
  }

  send(notification: Omit<TeamNotification, 'id' | 'createdAt' | 'readBy' | 'dismissedBy'>): TeamNotification {
    const newNotification: TeamNotification = {
      ...notification,
      id: `notif-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      createdAt: new Date(),
      readBy: [],
      dismissedBy: []
    }

    this.notifications.unshift(newNotification) // Add to beginning for chronological order
    this.saveNotifications()
    this.notifyListeners(newNotification)
    return newNotification
  }

  getNotifications(teamId: string, userId?: string): TeamNotification[] {
    let filtered = this.notifications.filter(n => n.teamId === teamId)

    if (userId) {
      filtered = filtered.filter(n =>
        n.recipients.includes(userId) || n.recipients.includes('all')
      )
    }

    return filtered
  }

  getUnreadCount(teamId: string, userId: string): number {
    return this.notifications.filter(n =>
      n.teamId === teamId &&
      (n.recipients.includes(userId) || n.recipients.includes('all')) &&
      !n.readBy.includes(userId)
    ).length
  }

  markAsRead(notificationId: string, userId: string): boolean {
    const notification = this.notifications.find(n => n.id === notificationId)
    if (!notification) return false

    if (!notification.readBy.includes(userId)) {
      notification.readBy.push(userId)
      this.saveNotifications()
    }
    return true
  }

  markAllAsRead(teamId: string, userId: string): number {
    let count = 0
    this.notifications.forEach(n => {
      if (n.teamId === teamId && !n.readBy.includes(userId)) {
        n.readBy.push(userId)
        count++
      }
    })
    if (count > 0) this.saveNotifications()
    return count
  }

  dismiss(notificationId: string, userId: string): boolean {
    const notification = this.notifications.find(n => n.id === notificationId)
    if (!notification) return false

    if (!notification.dismissedBy.includes(userId)) {
      notification.dismissedBy.push(userId)
      this.saveNotifications()
    }
    return true
  }

  parseMentions(content: string): Mention[] {
    const mentions: Mention[] = []
    const mentionRegex = /@(\w+)/g
    let match

    while ((match = mentionRegex.exec(content)) !== null) {
      mentions.push({
        userId: match[1],
        username: match[1],
        displayName: match[1],
        position: match.index
      })
    }

    return mentions
  }

  subscribe(listener: (notification: TeamNotification) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(notification: TeamNotification): void {
    this.listeners.forEach(listener => listener(notification))
  }

  private loadNotifications(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-notifications')
    if (stored) {
      try {
        this.notifications = JSON.parse(stored)
        this.notifications.forEach(n => {
          n.createdAt = new Date(n.createdAt)
          if (n.expiresAt) n.expiresAt = new Date(n.expiresAt)
        })
      } catch (e) {
        console.error('Failed to load notifications:', e)
      }
    }
  }

  private saveNotifications(): void {
    if (typeof window === 'undefined') return
    // Keep only last 1000 notifications
    const toSave = this.notifications.slice(0, 1000)
    localStorage.setItem('collaboration-notifications', JSON.stringify(toSave))
  }
}

// ==================== Share Manager ====================

export class ShareManager {
  private shares: Map<string, BackupShare> = new Map()
  private listeners: Set<(share: BackupShare) => void> = new Set()

  constructor() {
    this.loadShares()
  }

  shareBackup(
    backupId: string,
    backupName: string,
    teamId: string,
    sharedBy: string,
    permissions: SharePermission[],
    options?: { expiresAt?: Date; requiresApproval?: boolean }
  ): BackupShare {
    const share: BackupShare = {
      id: `share-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      backupId,
      backupName,
      teamId,
      sharedBy,
      sharedAt: new Date(),
      permissions,
      expiresAt: options?.expiresAt,
      accessCount: 0,
      isActive: true,
      shareUrl: `https://backup.example.com/shared/${backupId}`,
      requiresApproval: options?.requiresApproval || false
    }

    this.shares.set(share.id, share)
    this.saveShares()
    this.notifyListeners(share)
    return share
  }

  getShare(shareId: string): BackupShare | undefined {
    return this.shares.get(shareId)
  }

  getTeamShares(teamId: string): BackupShare[] {
    return Array.from(this.shares.values()).filter(s => s.teamId === teamId && s.isActive)
  }

  getBackupShares(backupId: string): BackupShare[] {
    return Array.from(this.shares.values()).filter(s => s.backupId === backupId && s.isActive)
  }

  revokeShare(shareId: string): boolean {
    const share = this.shares.get(shareId)
    if (!share) return false

    share.isActive = false
    this.saveShares()
    this.notifyListeners(share)
    return true
  }

  trackAccess(shareId: string): boolean {
    const share = this.shares.get(shareId)
    if (!share) return false

    share.accessCount++
    this.saveShares()
    return true
  }

  hasAccess(shareId: string, userId: string, requiredPermission: 'view' | 'edit' | 'execute' | 'admin'): boolean {
    const share = this.shares.get(shareId)
    if (!share || !share.isActive) return false

    // Check expiration
    if (share.expiresAt && share.expiresAt < new Date()) return false

    const userPermission = share.permissions.find(p => p.userId === userId)
    if (!userPermission) return false

    // Check permission level
    const permissionLevels = ['view', 'edit', 'execute', 'admin']
    const userLevel = permissionLevels.indexOf(userPermission.permissionLevel)
    const requiredLevel = permissionLevels.indexOf(requiredPermission)

    return userLevel >= requiredLevel
  }

  subscribe(listener: (share: BackupShare) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(share: BackupShare): void {
    this.listeners.forEach(listener => listener(share))
  }

  private loadShares(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-shares')
    if (stored) {
      try {
        const shares = JSON.parse(stored)
        shares.forEach((share: BackupShare) => {
          share.sharedAt = new Date(share.sharedAt)
          if (share.expiresAt) share.expiresAt = new Date(share.expiresAt)
          share.permissions.forEach(p => {
            p.grantedAt = new Date(p.grantedAt)
            if (p.expiresAt) p.expiresAt = new Date(p.expiresAt)
          })
          this.shares.set(share.id, share)
        })
      } catch (e) {
        console.error('Failed to load shares:', e)
      }
    }
  }

  private saveShares(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('collaboration-shares', JSON.stringify(Array.from(this.shares.values())))
  }
}

// ==================== Comment Manager ====================

export class CommentManager {
  private comments: Comment[] = []
  private listeners: Set<(comment: Comment) => void> = new Set()

  constructor() {
    this.loadComments()
  }

  addComment(
    resourceType: Comment['resourceType'],
    resourceId: string,
    teamId: string,
    authorId: string,
    authorName: string,
    content: string,
    options?: { parentId?: string; attachments?: Attachment[] }
  ): Comment {
    const mentions = this.parseMentions(content)

    const comment: Comment = {
      id: `comment-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      resourceType,
      resourceId,
      teamId,
      authorId,
      authorName,
      content,
      mentions,
      attachments: options?.attachments,
      createdAt: new Date(),
      isEdited: false,
      isDeleted: false,
      parentId: options?.parentId,
      reactions: []
    }

    this.comments.unshift(comment)
    this.saveComments()
    this.notifyListeners(comment)
    return comment
  }

  getComments(resourceType: string, resourceId: string): Comment[] {
    return this.comments.filter(c =>
      c.resourceType === resourceType &&
      c.resourceId === resourceId &&
      !c.isDeleted
    )
  }

  getThreadedComments(resourceType: string, resourceId: string): Comment[] {
    const comments = this.getComments(resourceType, resourceId)
    return comments.filter(c => !c.parentId) // Return only top-level comments
  }

  getReplies(commentId: string): Comment[] {
    return this.comments.filter(c => c.parentId === commentId && !c.isDeleted)
  }

  updateComment(commentId: string, content: string): boolean {
    const comment = this.comments.find(c => c.id === commentId)
    if (!comment || comment.isDeleted) return false

    comment.content = content
    comment.mentions = this.parseMentions(content)
    comment.editedAt = new Date()
    comment.isEdited = true
    this.saveComments()
    this.notifyListeners(comment)
    return true
  }

  deleteComment(commentId: string): boolean {
    const comment = this.comments.find(c => c.id === commentId)
    if (!comment) return false

    comment.isDeleted = true
    comment.content = '[Deleted]'
    this.saveComments()
    return true
  }

  addReaction(commentId: string, emoji: string, userId: string, username: string): boolean {
    const comment = this.comments.find(c => c.id === commentId)
    if (!comment || comment.isDeleted) return false

    // Check if user already reacted with this emoji
    const existing = comment.reactions.find(r => r.emoji === emoji && r.userId === userId)
    if (existing) return false

    comment.reactions.push({
      emoji,
      userId,
      username,
      createdAt: new Date()
    })

    this.saveComments()
    this.notifyListeners(comment)
    return true
  }

  removeReaction(commentId: string, emoji: string, userId: string): boolean {
    const comment = this.comments.find(c => c.id === commentId)
    if (!comment) return false

    comment.reactions = comment.reactions.filter(r =>
      !(r.emoji === emoji && r.userId === userId)
    )

    this.saveComments()
    this.notifyListeners(comment)
    return true
  }

  private parseMentions(content: string): Mention[] {
    const mentions: Mention[] = []
    const mentionRegex = /@(\w+)/g
    let match

    while ((match = mentionRegex.exec(content)) !== null) {
      mentions.push({
        userId: match[1],
        username: match[1],
        displayName: match[1],
        position: match.index
      })
    }

    return mentions
  }

  subscribe(listener: (comment: Comment) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(comment: Comment): void {
    this.listeners.forEach(listener => listener(comment))
  }

  private loadComments(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-comments')
    if (stored) {
      try {
        this.comments = JSON.parse(stored)
        this.comments.forEach(c => {
          c.createdAt = new Date(c.createdAt)
          if (c.updatedAt) c.updatedAt = new Date(c.updatedAt)
          if (c.editedAt) c.editedAt = new Date(c.editedAt)
          c.reactions.forEach(r => r.createdAt = new Date(r.createdAt))
        })
      } catch (e) {
        console.error('Failed to load comments:', e)
      }
    }
  }

  private saveComments(): void {
    if (typeof window === 'undefined') return
    // Keep only last 5000 comments
    const toSave = this.comments.slice(0, 5000)
    localStorage.setItem('collaboration-comments', JSON.stringify(toSave))
  }
}

// ==================== Activity Feed Manager ====================

export class ActivityFeedManager {
  private activities: ActivityEvent[] = []
  private listeners: Set<(activity: ActivityEvent) => void> = new Set()

  constructor() {
    this.loadActivities()
  }

  log(
    teamId: string,
    actorId: string,
    actorName: string,
    action: ActivityAction,
    resourceType: ActivityEvent['resourceType'],
    resourceId: string,
    resourceName: string,
    details: Record<string, any>,
    options?: { visibility?: ActivityEvent['visibility']; tags?: string[] }
  ): ActivityEvent {
    const activity: ActivityEvent = {
      id: `activity-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      teamId,
      actorId,
      actorName,
      action,
      resourceType,
      resourceId,
      resourceName,
      details,
      timestamp: new Date(),
      visibility: options?.visibility || 'team',
      tags: options?.tags || []
    }

    this.activities.unshift(activity)
    this.saveActivities()
    this.notifyListeners(activity)
    return activity
  }

  getActivities(teamId: string, filters?: {
    actorId?: string
    resourceType?: string
    action?: ActivityAction
    startDate?: Date
    endDate?: Date
    limit?: number
  }): ActivityEvent[] {
    let filtered = this.activities.filter(a => a.teamId === teamId)

    if (filters?.actorId) {
      filtered = filtered.filter(a => a.actorId === filters.actorId)
    }

    if (filters?.resourceType) {
      filtered = filtered.filter(a => a.resourceType === filters.resourceType)
    }

    if (filters?.action) {
      filtered = filtered.filter(a => a.action === filters.action)
    }

    if (filters?.startDate) {
      filtered = filtered.filter(a => a.timestamp >= filters.startDate!)
    }

    if (filters?.endDate) {
      filtered = filtered.filter(a => a.timestamp <= filters.endDate!)
    }

    if (filters?.limit) {
      filtered = filtered.slice(0, filters.limit)
    }

    return filtered
  }

  getRecentActivity(teamId: string, hours: number = 24): ActivityEvent[] {
    const cutoff = new Date(Date.now() - hours * 60 * 60 * 1000)
    return this.activities.filter(a =>
      a.teamId === teamId && a.timestamp >= cutoff
    )
  }

  subscribe(listener: (activity: ActivityEvent) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(activity: ActivityEvent): void {
    this.listeners.forEach(listener => listener(activity))
  }

  private loadActivities(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-activities')
    if (stored) {
      try {
        this.activities = JSON.parse(stored)
        this.activities.forEach(a => a.timestamp = new Date(a.timestamp))
      } catch (e) {
        console.error('Failed to load activities:', e)
      }
    }
  }

  private saveActivities(): void {
    if (typeof window === 'undefined') return
    // Keep only last 10000 activities
    const toSave = this.activities.slice(0, 10000)
    localStorage.setItem('collaboration-activities', JSON.stringify(toSave))
  }
}

// ==================== Audit Log Manager ====================

export class AuditLogManager {
  private auditLogs: AuditLogEntry[] = []
  private listeners: Set<(entry: AuditLogEntry) => void> = new Set()

  constructor() {
    this.loadAuditLogs()
  }

  log(
    teamId: string,
    actorId: string,
    actorName: string,
    actorEmail: string,
    action: string,
    resourceType: string,
    resourceId: string,
    options?: {
      resourceName?: string
      changes?: AuditChange[]
      status?: AuditLogEntry['status']
      severity?: AuditLogEntry['severity']
      metadata?: Record<string, any>
      actorIp?: string
      userAgent?: string
      sessionId?: string
    }
  ): AuditLogEntry {
    const entry: AuditLogEntry = {
      id: `audit-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      teamId,
      timestamp: new Date(),
      actorId,
      actorName,
      actorEmail,
      actorIp: options?.actorIp,
      action,
      resourceType,
      resourceId,
      resourceName: options?.resourceName,
      changes: options?.changes,
      status: options?.status || 'success',
      severity: options?.severity || 'info',
      metadata: options?.metadata || {},
      userAgent: options?.userAgent,
      sessionId: options?.sessionId
    }

    this.auditLogs.unshift(entry)
    this.saveAuditLogs()
    this.notifyListeners(entry)
    return entry
  }

  getLogs(teamId: string, filters?: {
    actorId?: string
    action?: string
    resourceType?: string
    severity?: AuditLogEntry['severity']
    startDate?: Date
    endDate?: Date
    limit?: number
  }): AuditLogEntry[] {
    let filtered = this.auditLogs.filter(l => l.teamId === teamId)

    if (filters?.actorId) {
      filtered = filtered.filter(l => l.actorId === filters.actorId)
    }

    if (filters?.action) {
      filtered = filtered.filter(l => l.action === filters.action)
    }

    if (filters?.resourceType) {
      filtered = filtered.filter(l => l.resourceType === filters.resourceType)
    }

    if (filters?.severity) {
      filtered = filtered.filter(l => l.severity === filters.severity)
    }

    if (filters?.startDate) {
      filtered = filtered.filter(l => l.timestamp >= filters.startDate!)
    }

    if (filters?.endDate) {
      filtered = filtered.filter(l => l.timestamp <= filters.endDate!)
    }

    if (filters?.limit) {
      filtered = filtered.slice(0, filters.limit)
    }

    return filtered
  }

  getCriticalLogs(teamId: string, days: number = 7): AuditLogEntry[] {
    const cutoff = new Date(Date.now() - days * 24 * 60 * 60 * 1000)
    return this.auditLogs.filter(l =>
      l.teamId === teamId &&
      l.severity === 'critical' &&
      l.timestamp >= cutoff
    )
  }

  exportLogs(teamId: string, startDate?: Date, endDate?: Date): string {
    const logs = this.getLogs(teamId, { startDate, endDate })
    return JSON.stringify(logs, null, 2)
  }

  subscribe(listener: (entry: AuditLogEntry) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(entry: AuditLogEntry): void {
    this.listeners.forEach(listener => listener(entry))
  }

  private loadAuditLogs(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-audit-logs')
    if (stored) {
      try {
        this.auditLogs = JSON.parse(stored)
        this.auditLogs.forEach(l => {
          l.timestamp = new Date(l.timestamp)
          if (l.changes) {
            l.changes.forEach(c => c.timestamp = new Date(c.timestamp))
          }
        })
      } catch (e) {
        console.error('Failed to load audit logs:', e)
      }
    }
  }

  private saveAuditLogs(): void {
    if (typeof window === 'undefined') return
    // Keep only last 10000 audit logs
    const toSave = this.auditLogs.slice(0, 10000)
    localStorage.setItem('collaboration-audit-logs', JSON.stringify(toSave))
  }
}

// ==================== Approval Workflow Manager ====================

export class ApprovalWorkflowManager {
  private requests: Map<string, ApprovalRequest> = new Map()
  private listeners: Set<(request: ApprovalRequest) => void> = new Set()

  constructor() {
    this.loadRequests()
  }

  createRequest(
    teamId: string,
    type: ApprovalRequest['type'],
    title: string,
    description: string,
    requestedBy: string,
    requestedByName: string,
    resourceType: string,
    resourceId: string,
    resourceName: string,
    approvers: Omit<Approver, 'status' | 'order'>[],
    requiredApprovals: number,
    options?: { expiresAt?: Date; metadata?: Record<string, any> }
  ): ApprovalRequest {
    const request: ApprovalRequest = {
      id: `approval-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      teamId,
      type,
      title,
      description,
      requestedBy,
      requestedByName,
      requestedAt: new Date(),
      resourceType,
      resourceId,
      resourceName,
      approvers: approvers.map((a, index) => ({
        ...a,
        status: 'pending',
        order: index
      })),
      requiredApprovals,
      currentApprovals: 0,
      status: 'pending',
      expiresAt: options?.expiresAt,
      metadata: options?.metadata || {}
    }

    this.requests.set(request.id, request)
    this.saveRequests()
    this.notifyListeners(request)
    return request
  }

  getRequest(requestId: string): ApprovalRequest | undefined {
    return this.requests.get(requestId)
  }

  getPendingRequests(teamId: string, userId?: string): ApprovalRequest[] {
    let filtered = Array.from(this.requests.values()).filter(r =>
      r.teamId === teamId && r.status === 'pending'
    )

    if (userId) {
      filtered = filtered.filter(r =>
        r.approvers.some(a => a.userId === userId && a.status === 'pending')
      )
    }

    return filtered
  }

  approve(requestId: string, userId: string, comment?: string): boolean {
    const request = this.requests.get(requestId)
    if (!request || request.status !== 'pending') return false

    const approver = request.approvers.find(a => a.userId === userId)
    if (!approver || approver.status !== 'pending') return false

    approver.status = 'approved'
    approver.respondedAt = new Date()
    approver.comment = comment

    request.currentApprovals++

    // Check if enough approvals
    if (request.currentApprovals >= request.requiredApprovals) {
      request.status = 'approved'
      request.completedAt = new Date()
      request.completedBy = userId
    }

    this.saveRequests()
    this.notifyListeners(request)
    return true
  }

  reject(requestId: string, userId: string, comment?: string): boolean {
    const request = this.requests.get(requestId)
    if (!request || request.status !== 'pending') return false

    const approver = request.approvers.find(a => a.userId === userId)
    if (!approver || approver.status !== 'pending') return false

    approver.status = 'rejected'
    approver.respondedAt = new Date()
    approver.comment = comment

    request.status = 'rejected'
    request.completedAt = new Date()
    request.completedBy = userId

    this.saveRequests()
    this.notifyListeners(request)
    return true
  }

  cancel(requestId: string): boolean {
    const request = this.requests.get(requestId)
    if (!request || request.status !== 'pending') return false

    request.status = 'cancelled'
    request.completedAt = new Date()

    this.saveRequests()
    this.notifyListeners(request)
    return true
  }

  checkExpired(): number {
    const now = new Date()
    let count = 0

    this.requests.forEach(request => {
      if (
        request.status === 'pending' &&
        request.expiresAt &&
        request.expiresAt < now
      ) {
        request.status = 'expired'
        request.completedAt = now
        count++
      }
    })

    if (count > 0) {
      this.saveRequests()
    }

    return count
  }

  subscribe(listener: (request: ApprovalRequest) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(request: ApprovalRequest): void {
    this.listeners.forEach(listener => listener(request))
  }

  private loadRequests(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-approval-requests')
    if (stored) {
      try {
        const requests = JSON.parse(stored)
        requests.forEach((request: ApprovalRequest) => {
          request.requestedAt = new Date(request.requestedAt)
          if (request.expiresAt) request.expiresAt = new Date(request.expiresAt)
          if (request.completedAt) request.completedAt = new Date(request.completedAt)
          request.approvers.forEach(a => {
            if (a.respondedAt) a.respondedAt = new Date(a.respondedAt)
          })
          this.requests.set(request.id, request)
        })
      } catch (e) {
        console.error('Failed to load approval requests:', e)
      }
    }
  }

  private saveRequests(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('collaboration-approval-requests', JSON.stringify(Array.from(this.requests.values())))
  }
}

// ==================== Team Calendar Manager ====================

export class TeamCalendarManager {
  private events: Map<string, CalendarEvent> = new Map()
  private listeners: Set<(event: CalendarEvent) => void> = new Set()

  constructor() {
    this.loadEvents()
  }

  createEvent(
    teamId: string,
    type: CalendarEvent['type'],
    title: string,
    startTime: Date,
    createdBy: string,
    options?: {
      description?: string
      endTime?: Date
      duration?: number
      schedule?: SharedSchedule
      location?: string
      attendees?: string[]
      reminders?: Reminder[]
      recurrence?: RecurrenceRule
      color?: string
      isAllDay?: boolean
    }
  ): CalendarEvent {
    const event: CalendarEvent = {
      id: `event-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      teamId,
      type,
      title,
      description: options?.description,
      startTime,
      endTime: options?.endTime,
      duration: options?.duration,
      schedule: options?.schedule,
      location: options?.location,
      attendees: options?.attendees || [],
      reminders: options?.reminders || [],
      recurrence: options?.recurrence,
      color: options?.color,
      isAllDay: options?.isAllDay || false,
      createdBy,
      createdAt: new Date()
    }

    this.events.set(event.id, event)
    this.saveEvents()
    this.notifyListeners(event)
    return event
  }

  getEvent(eventId: string): CalendarEvent | undefined {
    return this.events.get(eventId)
  }

  getTeamEvents(teamId: string, startDate?: Date, endDate?: Date): CalendarEvent[] {
    let filtered = Array.from(this.events.values()).filter(e => e.teamId === teamId)

    if (startDate) {
      filtered = filtered.filter(e => e.startTime >= startDate)
    }

    if (endDate) {
      filtered = filtered.filter(e => e.startTime <= endDate)
    }

    return filtered.sort((a, b) => a.startTime.getTime() - b.startTime.getTime())
  }

  getUserEvents(teamId: string, userId: string, startDate?: Date, endDate?: Date): CalendarEvent[] {
    return this.getTeamEvents(teamId, startDate, endDate).filter(e =>
      e.createdBy === userId || e.attendees.includes(userId)
    )
  }

  updateEvent(eventId: string, updates: Partial<CalendarEvent>): boolean {
    const event = this.events.get(eventId)
    if (!event) return false

    Object.assign(event, updates, { updatedAt: new Date() })
    this.saveEvents()
    this.notifyListeners(event)
    return true
  }

  deleteEvent(eventId: string): boolean {
    const deleted = this.events.delete(eventId)
    if (deleted) this.saveEvents()
    return deleted
  }

  addAttendee(eventId: string, userId: string): boolean {
    const event = this.events.get(eventId)
    if (!event) return false

    if (!event.attendees.includes(userId)) {
      event.attendees.push(userId)
      event.updatedAt = new Date()
      this.saveEvents()
      this.notifyListeners(event)
    }

    return true
  }

  removeAttendee(eventId: string, userId: string): boolean {
    const event = this.events.get(eventId)
    if (!event) return false

    event.attendees = event.attendees.filter(id => id !== userId)
    event.updatedAt = new Date()
    this.saveEvents()
    this.notifyListeners(event)
    return true
  }

  subscribe(listener: (event: CalendarEvent) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(event: CalendarEvent): void {
    this.listeners.forEach(listener => listener(event))
  }

  private loadEvents(): void {
    if (typeof window === 'undefined') return

    const stored = localStorage.getItem('collaboration-calendar-events')
    if (stored) {
      try {
        const events = JSON.parse(stored)
        events.forEach((event: CalendarEvent) => {
          event.startTime = new Date(event.startTime)
          if (event.endTime) event.endTime = new Date(event.endTime)
          event.createdAt = new Date(event.createdAt)
          if (event.updatedAt) event.updatedAt = new Date(event.updatedAt)
          event.reminders.forEach(r => {
            if (r.sentAt) r.sentAt = new Date(r.sentAt)
          })
          if (event.recurrence?.endDate) {
            event.recurrence.endDate = new Date(event.recurrence.endDate)
          }
          this.events.set(event.id, event)
        })
      } catch (e) {
        console.error('Failed to load calendar events:', e)
      }
    }
  }

  private saveEvents(): void {
    if (typeof window === 'undefined') return
    localStorage.setItem('collaboration-calendar-events', JSON.stringify(Array.from(this.events.values())))
  }
}

// ==================== Global Instances ====================

export const teamManager = new TeamManager()
export const sharedScheduleManager = new SharedScheduleManager()
export const teamNotificationManager = new TeamNotificationManager()
export const shareManager = new ShareManager()
export const commentManager = new CommentManager()
export const activityFeedManager = new ActivityFeedManager()
export const auditLogManager = new AuditLogManager()
export const approvalWorkflowManager = new ApprovalWorkflowManager()
export const teamCalendarManager = new TeamCalendarManager()

// ==================== Utility Functions ====================

export function exportTeamData(teamId: string): string {
  return JSON.stringify({
    team: teamManager.getTeam(teamId),
    schedules: sharedScheduleManager.getTeamSchedules(teamId),
    notifications: teamNotificationManager.getNotifications(teamId),
    shares: shareManager.getTeamShares(teamId),
    activities: activityFeedManager.getActivities(teamId),
    auditLogs: auditLogManager.getLogs(teamId),
    calendarEvents: teamCalendarManager.getTeamEvents(teamId),
    exportedAt: new Date().toISOString()
  }, null, 2)
}

export function getTeamStats(teamId: string) {
  const team = teamManager.getTeam(teamId)
  if (!team) return null

  return {
    memberCount: team.members.length,
    activeMembers: team.members.filter(m => m.status === 'active').length,
    schedulesCount: sharedScheduleManager.getTeamSchedules(teamId).length,
    activeShares: shareManager.getTeamShares(teamId).length,
    unreadNotifications: teamNotificationManager.getNotifications(teamId).filter(n => n.readBy.length === 0).length,
    recentActivity: activityFeedManager.getRecentActivity(teamId, 24).length,
    pendingApprovals: approvalWorkflowManager.getPendingRequests(teamId).length,
    upcomingEvents: teamCalendarManager.getTeamEvents(teamId, new Date()).slice(0, 5).length
  }
}
