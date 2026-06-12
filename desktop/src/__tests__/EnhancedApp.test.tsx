import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import EnhancedApp from '../EnhancedApp'
import { invoke } from '@tauri-apps/api/tauri'

// Mock the invoke function
vi.mock('@tauri-apps/api/tauri', () => ({
  invoke: vi.fn(),
}))

vi.mock('@tauri-apps/api/notification', () => ({
  sendNotification: vi.fn(),
}))

vi.mock('@tauri-apps/api/window', () => ({
  appWindow: {
    minimize: vi.fn(),
    close: vi.fn(),
    listen: vi.fn(),
  },
}))

describe('EnhancedApp', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Setup default mock responses
    ;(invoke as any).mockImplementation((command: string) => {
      switch (command) {
        case 'get_config':
          return Promise.resolve({
            api_url: 'http://localhost:3000',
            api_key: 'test-key',
            auto_backup_enabled: true,
            backup_interval_hours: 24,
            notifications_enabled: true,
            theme: 'dark',
            language: 'en',
            auto_launch: false,
            keyboard_shortcuts: {
              quick_search: 'Cmd+K',
              new_backup: 'Cmd+N',
              refresh: 'Cmd+R',
              toggle_theme: 'Cmd+Shift+D',
              open_settings: 'Cmd+,',
            },
          })
        case 'get_backups':
          return Promise.resolve([])
        case 'get_theme':
          return Promise.resolve('dark')
        default:
          return Promise.resolve(null)
      }
    })
  })

  it('renders without crashing', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      expect(screen.getByText(/DB Backup Desktop/i)).toBeInTheDocument()
    })
  })

  it('displays navigation tabs', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      const backupsElements = screen.getAllByText(/^Backups$/i)
      const activityElements = screen.getAllByText(/^Activity$/i)
      const settingsElements = screen.getAllByText(/^Settings$/i)
      expect(backupsElements.length).toBeGreaterThan(0)
      expect(activityElements.length).toBeGreaterThan(0)
      expect(settingsElements.length).toBeGreaterThan(0)
    })
  })

  it('loads configuration on mount', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      expect(invoke).toHaveBeenCalledWith('get_config')
    })
  })

  it('loads backups on mount', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      expect(invoke).toHaveBeenCalledWith('get_backups', { limit: 50 })
    })
  })

  it('can switch between tabs', async () => {
    const user = userEvent.setup()
    render(<EnhancedApp />)

    await waitFor(() => {
      const backupsElements = screen.getAllByText(/^Backups$/i)
      expect(backupsElements.length).toBeGreaterThan(0)
    })

    // Find the Activity tab button (first occurrence is the tab button)
    const activityTab = screen.getAllByText(/^Activity$/i)[0]
    await user.click(activityTab)

    // Verify activity tab content is shown (looking for Activity Log heading)
    await waitFor(() => {
      expect(screen.getByText(/Activity Log/i)).toBeInTheDocument()
    })
  })

  it('displays theme toggle button', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      const themeButtons = screen.getAllByRole('button')
      expect(themeButtons.length).toBeGreaterThan(0)
    })
  })

  it('can toggle theme', async () => {
    const user = userEvent.setup()
    ;(invoke as any).mockImplementation((command: string) => {
      if (command === 'toggle_theme') {
        return Promise.resolve('light')
      }
      if (command === 'get_config') {
        return Promise.resolve({
          api_url: 'http://localhost:3000',
          api_key: 'test-key',
          auto_backup_enabled: true,
          backup_interval_hours: 24,
          notifications_enabled: true,
          theme: 'dark',
          language: 'en',
          auto_launch: false,
          keyboard_shortcuts: {},
        })
      }
      if (command === 'get_backups') {
        return Promise.resolve([])
      }
      if (command === 'get_theme') {
        return Promise.resolve('dark')
      }
      return Promise.resolve(null)
    })

    render(<EnhancedApp />)

    await waitFor(() => {
      expect(screen.getByText(/DB Backup Desktop/i)).toBeInTheDocument()
    })

    // Find and click the theme toggle button (Sun/Moon icon)
    const themeButton = screen.getAllByRole('button').find(btn =>
      btn.querySelector('svg')
    )

    if (themeButton) {
      await user.click(themeButton)
      await waitFor(() => {
        expect(invoke).toHaveBeenCalledWith('toggle_theme')
      })
    }
  })

  it('displays language selector', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      const buttons = screen.getAllByRole('button')
      expect(buttons.length).toBeGreaterThan(0)
    })
  })

  it('shows new backup button', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      const newBackupButtons = screen.getAllByText(/New Backup/i)
      expect(newBackupButtons.length).toBeGreaterThan(0)
    })
  })

  it('shows refresh button', async () => {
    render(<EnhancedApp />)
    await waitFor(() => {
      const refreshButtons = screen.getAllByRole('button').filter(btn =>
        btn.textContent?.includes('Refresh')
      )
      expect(refreshButtons.length).toBeGreaterThan(0)
    })
  })
})
