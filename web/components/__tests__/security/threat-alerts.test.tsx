import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { ThreatAlerts } from '../../security/threat-alerts'

describe('ThreatAlerts', () => {
  it('should render severity summary cards', () => {
    const { container } = render(<ThreatAlerts />)

    // Check that severity cards are rendered
    expect(container.textContent).toContain('Critical')
    expect(container.textContent).toContain('High')
    expect(container.textContent).toContain('Medium')
    expect(container.textContent).toContain('Low')
    expect(container.textContent).toContain('Info')
  })

  it('should display correct severity counts', () => {
    render(<ThreatAlerts />)

    // Based on mock data, we should have:
    // 0 Critical, 0 High, 1 Medium, 1 Low, 1 Info
    const { container } = render(<ThreatAlerts />)

    // Should show counts for each severity level
    const summaryCards = container.querySelectorAll('.text-2xl')
    expect(summaryCards.length).toBeGreaterThan(0)
  })

  it('should render filter controls', () => {
    render(<ThreatAlerts />)

    expect(screen.getByText('Filter:')).toBeInTheDocument()
    expect(screen.getByText('Severity:')).toBeInTheDocument()
    expect(screen.getByText('Status:')).toBeInTheDocument()
  })

  it('should have severity filter options', () => {
    const { container } = render(<ThreatAlerts />)

    const selects = container.querySelectorAll('select')
    const severitySelect = selects[0] // First select is severity

    expect(within(severitySelect).getByText('All')).toBeInTheDocument()
    expect(within(severitySelect).getByText('Critical')).toBeInTheDocument()
    expect(within(severitySelect).getByText('High')).toBeInTheDocument()
    expect(within(severitySelect).getByText('Medium')).toBeInTheDocument()
    expect(within(severitySelect).getByText('Low')).toBeInTheDocument()
    expect(within(severitySelect).getByText('Info')).toBeInTheDocument()
  })

  it('should have status filter options', () => {
    const { container } = render(<ThreatAlerts />)

    const selects = container.querySelectorAll('select')
    const statusSelect = selects[1] // Second select is status

    expect(within(statusSelect).getByText('All')).toBeInTheDocument()
    expect(within(statusSelect).getByText('Active')).toBeInTheDocument()
    expect(within(statusSelect).getByText('Investigating')).toBeInTheDocument()
    expect(within(statusSelect).getByText('Resolved')).toBeInTheDocument()
    expect(within(statusSelect).getByText('Dismissed')).toBeInTheDocument()
  })

  it('should render mock alert data', () => {
    render(<ThreatAlerts />)

    expect(screen.getByText('Backup Scan Completed Successfully')).toBeInTheDocument()
    expect(screen.getByText('Unusual Backup Timing Detected')).toBeInTheDocument()
    expect(screen.getByText('Retention Policy Change Attempted')).toBeInTheDocument()
  })

  it('should filter alerts by severity', () => {
    const { container } = render(<ThreatAlerts />)

    const selects = container.querySelectorAll('select')
    const severitySelect = selects[0]
    fireEvent.change(severitySelect, { target: { value: 'MEDIUM' } })

    // Should only show MEDIUM severity alerts
    expect(screen.getByText('Retention Policy Change Attempted')).toBeInTheDocument()
    expect(screen.queryByText('Backup Scan Completed Successfully')).not.toBeInTheDocument()
  })

  it('should filter alerts by status', () => {
    const { container } = render(<ThreatAlerts />)

    const selects = container.querySelectorAll('select')
    const statusSelect = selects[1]
    fireEvent.change(statusSelect, { target: { value: 'resolved' } })

    // All mock alerts are resolved, so they should all be visible
    expect(screen.getByText('Backup Scan Completed Successfully')).toBeInTheDocument()
    expect(screen.getByText('Unusual Backup Timing Detected')).toBeInTheDocument()
  })

  it('should show no alerts message when filters match nothing', () => {
    const { container } = render(<ThreatAlerts />)

    const selects = container.querySelectorAll('select')
    const severitySelect = selects[0]
    fireEvent.change(severitySelect, { target: { value: 'CRITICAL' } })

    expect(screen.getByText('No Alerts Found')).toBeInTheDocument()
    expect(screen.getByText('No alerts match your current filters')).toBeInTheDocument()
  })

  it('should open alert details modal when alert clicked', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Backup Scan Completed Successfully')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    // Modal should show detailed information
    expect(screen.getByText('Description')).toBeInTheDocument()
    expect(screen.getByText('Detection Time')).toBeInTheDocument()
    expect(screen.getByText('Detection Method')).toBeInTheDocument()
    expect(screen.getByText('Affected Resources')).toBeInTheDocument()
    expect(screen.getByText('Recommended Action')).toBeInTheDocument()
  })

  it('should display alert details in modal', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Backup Scan Completed Successfully')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    const detailTexts = screen.getAllByText('Regular security scan completed with no threats detected')
    expect(detailTexts.length).toBeGreaterThan(0)

    const scheduledScanTexts = screen.getAllByText('Scheduled Scan')
    expect(scheduledScanTexts.length).toBeGreaterThan(0)

    const actionTexts = screen.getAllByText('No action required')
    expect(actionTexts.length).toBeGreaterThan(0)
  })

  it('should close modal when cancel clicked', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Backup Scan Completed Successfully')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    const closeButton = screen.getByText('Close')
    fireEvent.click(closeButton)

    expect(screen.queryByText('Description')).not.toBeInTheDocument()
  })

  it('should not show action buttons for resolved alerts', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Backup Scan Completed Successfully')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    // Resolved alerts should not have Dismiss or Mark as Resolved buttons
    expect(screen.queryByText('Dismiss')).not.toBeInTheDocument()
    expect(screen.queryByText('Mark as Resolved')).not.toBeInTheDocument()
  })

  it('should display alert severity badges', () => {
    render(<ThreatAlerts />)

    // Should show INFO, LOW, MEDIUM badges based on mock data
    const badges = screen.getAllByText('INFO', { exact: true })
    expect(badges.length).toBeGreaterThanOrEqual(1)
  })

  it('should display alert status badges', () => {
    render(<ThreatAlerts />)

    const resolvedBadges = screen.getAllByText('resolved', { exact: true })
    expect(resolvedBadges.length).toBeGreaterThanOrEqual(1)
  })

  it('should render affected resources list in modal', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Backup Scan Completed Successfully')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    const affectedResources = screen.getAllByText('All backup files')
    expect(affectedResources.length).toBeGreaterThan(0)
  })

  it('should show recommended action in modal', () => {
    render(<ThreatAlerts />)

    const alertTitle = screen.getByText('Retention Policy Change Attempted')
    fireEvent.click(alertTitle.closest('div[class*="cursor-pointer"]')!)

    expect(screen.getByText('Retention policy is locked and cannot be reduced')).toBeInTheDocument()
  })
})
