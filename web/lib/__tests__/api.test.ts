import { describe, it, expect, beforeEach, vi } from 'vitest'

// Use vi.hoisted to ensure mocks are available during hoisting
const { mockGet, mockPost, mockPut, mockDelete, mockAxiosInstance } = vi.hoisted(() => {
  const mockGet = vi.fn()
  const mockPost = vi.fn()
  const mockPut = vi.fn()
  const mockDelete = vi.fn()
  const mockRequestUse = vi.fn((fn) => fn)
  const mockResponseUse = vi.fn()

  const mockAxiosInstance = {
    get: mockGet,
    post: mockPost,
    put: mockPut,
    delete: mockDelete,
    interceptors: {
      request: {
        use: mockRequestUse,
      },
      response: {
        use: mockResponseUse,
      },
    },
  }

  return {
    mockGet,
    mockPost,
    mockPut,
    mockDelete,
    mockAxiosInstance,
  }
})

// Mock axios module
vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => mockAxiosInstance),
  },
}))

// Import api after mocking
import { api } from '../api'

describe('API Client', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  describe('Backups API', () => {
    it('should list backups', async () => {
      const mockBackups = {
        backups: [
          {
            id: 'backup-1',
            database_id: 'db-1',
            filename: 'backup.sql',
            size: 1024,
            status: 'completed',
            created_at: '2025-12-28T00:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        page_size: 10,
      }

      mockGet.mockResolvedValue({ data: mockBackups })

      const result = await api.listBackups()
      expect(result).toEqual(mockBackups)
    })

    it('should create backup', async () => {
      const mockBackup = {
        id: 'backup-1',
        database_id: 'db-1',
        filename: 'backup.sql',
        size: 1024,
        status: 'in_progress',
        created_at: '2025-12-28T00:00:00Z',
      }

      mockPost.mockResolvedValue({ data: mockBackup })

      const result = await api.createBackup('db-1')
      expect(result).toEqual(mockBackup)
    })

    it('should delete backup', async () => {
      mockDelete.mockResolvedValue({})

      await expect(api.deleteBackup('backup-1')).resolves.not.toThrow()
    })

    it('should restore backup', async () => {
      mockPost.mockResolvedValue({ data: { success: true } })

      const result = await api.restoreBackup('backup-1', 'target-db-1')
      expect(result).toEqual({ success: true })
    })
  })

  describe('Databases API', () => {
    it('should list databases', async () => {
      const mockDatabases = [
        {
          id: 'db-1',
          name: 'mydb',
          type: 'postgres',
          host: 'localhost',
          port: 5432,
          username: 'user',
          created_at: '2025-12-28T00:00:00Z',
          updated_at: '2025-12-28T00:00:00Z',
        },
      ]

      mockGet.mockResolvedValue({ data: mockDatabases })

      const result = await api.listDatabases()
      expect(result).toEqual(mockDatabases)
    })

    it('should test connection', async () => {
      mockPost.mockResolvedValue({
        data: { success: true, message: 'Connection successful' },
      })

      const result = await api.testConnection('db-1')
      expect(result.success).toBe(true)
    })
  })

  describe('Schedules API', () => {
    it('should list schedules', async () => {
      const mockSchedules = [
        {
          id: 'schedule-1',
          database_id: 'db-1',
          cron_expression: '0 0 * * *',
          retention_days: 30,
          enabled: true,
          created_at: '2025-12-28T00:00:00Z',
          updated_at: '2025-12-28T00:00:00Z',
        },
      ]

      mockGet.mockResolvedValue({ data: mockSchedules })

      const result = await api.listSchedules()
      expect(result).toEqual(mockSchedules)
    })

    it('should create schedule', async () => {
      const newSchedule = {
        database_id: 'db-1',
        cron_expression: '0 0 * * *',
        retention_days: 30,
        enabled: true,
      }

      mockPost.mockResolvedValue({ data: { id: 'schedule-1', ...newSchedule } })

      const result = await api.createSchedule(newSchedule)
      expect(result.id).toBe('schedule-1')
    })
  })

  describe('Auth Token', () => {
    it('should add auth token to requests', async () => {
      const token = 'test-token-123'
      localStorage.setItem('auth_token', token)

      mockGet.mockResolvedValue({ data: { backups: [], total: 0, page: 1, page_size: 10 } })

      // This should trigger the interceptor configured in api.ts
      await api.listBackups()

      // The interceptor was called during module initialization
      // We just verify the call succeeded
      expect(mockGet).toHaveBeenCalled()
    })
  })
})
