import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { RansomwareDetectionDashboard } from '../../security/ransomware-detection-dashboard'

describe('RansomwareDetectionDashboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('should render detection metrics', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Files Scanned')).toBeInTheDocument()
    expect(screen.getByText('1,247')).toBeInTheDocument()
    expect(screen.getByText('Threats Detected')).toBeInTheDocument()
    expect(screen.getByText('Monitoring Status')).toBeInTheDocument()
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('should render scan controls', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Scan Controls')).toBeInTheDocument()
    expect(screen.getByText('Start Full Scan')).toBeInTheDocument()
  })

  it('should start scan when button clicked', async () => {
    render(<RansomwareDetectionDashboard />)

    const scanButton = screen.getByText('Start Full Scan')

    act(() => {
      fireEvent.click(scanButton)
    })

    // Advance timers in act
    act(() => {
      vi.advanceTimersByTime(100)
    })

    // Scanning should have started
    expect(screen.getByText('Scanning...')).toBeInTheDocument()
  })

  it('should show scan progress', async () => {
    const { container } = render(<RansomwareDetectionDashboard />)

    const scanButton = screen.getByText('Start Full Scan')

    act(() => {
      fireEvent.click(scanButton)
    })

    act(() => {
      vi.advanceTimersByTime(100)
    })

    expect(screen.getByText('Scanning backups...')).toBeInTheDocument()

    // Should show progress bar
    const progressBar = container.querySelector('.bg-blue-600')
    expect(progressBar).toBeTruthy()
  })

  it('should disable scan button during scanning', async () => {
    render(<RansomwareDetectionDashboard />)

    const scanButton = screen.getByText('Start Full Scan')

    act(() => {
      fireEvent.click(scanButton)
    })

    act(() => {
      vi.advanceTimersByTime(100)
    })

    expect(scanButton).toBeDisabled()
  })

  it('should render detection methods', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Active Detection Methods')).toBeInTheDocument()
    expect(screen.getByText('Entropy Analysis')).toBeInTheDocument()
    expect(screen.getByText('Rapid Modification Detection')).toBeInTheDocument()
    expect(screen.getByText('Signature Matching')).toBeInTheDocument()
    expect(screen.getByText('Extension Detection')).toBeInTheDocument()
  })

  it('should render thresholds', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Entropy Threshold')).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(screen.getByText('Modification Threshold')).toBeInTheDocument()
    expect(screen.getByText('10 files/5min')).toBeInTheDocument()
  })

  it('should render recent scan results table', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Recent Scan Results')).toBeInTheDocument()
    expect(screen.getByText('Export Report')).toBeInTheDocument()

    // Check table headers
    expect(screen.getByText('File')).toBeInTheDocument()
    expect(screen.getByText('Threat Type')).toBeInTheDocument()
    expect(screen.getByText('Level')).toBeInTheDocument()
    expect(screen.getByText('Entropy')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
    expect(screen.getByText('Time')).toBeInTheDocument()
  })

  it('should display scan result data', () => {
    const { container } = render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('backup-prod-db.sql')).toBeInTheDocument()
    expect(screen.getByText('/backups/prod/backup-prod-db.sql')).toBeInTheDocument()

    // Entropy value might be formatted, so use a more flexible check
    expect(container.textContent).toContain('4.2')
  })

  it('should show clean status for safe files', () => {
    render(<RansomwareDetectionDashboard />)

    expect(screen.getByText('Clean')).toBeInTheDocument()
  })
})
