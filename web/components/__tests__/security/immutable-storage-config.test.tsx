import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { ImmutableStorageConfig } from '../../security/immutable-storage-config'

describe('ImmutableStorageConfig', () => {
  it('should render overview section', () => {
    render(<ImmutableStorageConfig />)

    expect(screen.getByText('Multi-Cloud Immutable Storage Protection')).toBeInTheDocument()
    expect(screen.getByText(/WORM \(Write-Once-Read-Many\) protection/)).toBeInTheDocument()
    expect(screen.getByText('3 Providers Active')).toBeInTheDocument()
    expect(screen.getByText('156 Protected Backups')).toBeInTheDocument()
  })

  it('should render all storage providers', () => {
    render(<ImmutableStorageConfig />)

    expect(screen.getByText('AWS S3 Production')).toBeInTheDocument()
    expect(screen.getByText('Azure Blob Storage')).toBeInTheDocument()
    expect(screen.getByText('Google Cloud Storage')).toBeInTheDocument()
  })

  it('should display provider configurations', () => {
    render(<ImmutableStorageConfig />)

    // Check S3 configuration
    expect(screen.getByText('GOVERNANCE')).toBeInTheDocument()
    expect(screen.getByText('30 days')).toBeInTheDocument()
    expect(screen.getByText('87')).toBeInTheDocument()

    // Check Azure configuration
    expect(screen.getByText('LOCKED')).toBeInTheDocument()
    expect(screen.getByText('90 days')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()

    // Check GCS configuration
    expect(screen.getByText('UNLOCKED')).toBeInTheDocument()
    expect(screen.getByText('60 days')).toBeInTheDocument()
    expect(screen.getByText('27')).toBeInTheDocument()
  })

  it('should show enabled status for all providers', () => {
    render(<ImmutableStorageConfig />)

    const enabledLabels = screen.getAllByText('Enabled', { exact: false })
    expect(enabledLabels.length).toBeGreaterThanOrEqual(3)
  })

  it('should render configure buttons for each provider', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    expect(configureButtons).toHaveLength(3)
  })

  it('should open configuration modal when configure clicked', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    expect(screen.getByText('Configure AWS S3 Production')).toBeInTheDocument()
  })

  it('should render modal configuration options', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    expect(screen.getByText('Enable Immutability')).toBeInTheDocument()
    expect(screen.getByText('Retention Period (days)')).toBeInTheDocument()
    expect(screen.getByText('Protection Mode')).toBeInTheDocument()

    const legalHoldTexts = screen.getAllByText('Legal Hold')
    expect(legalHoldTexts.length).toBeGreaterThan(0)
  })

  it('should show different modes for S3', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    const governanceOption = screen.getByRole('option', { name: /Governance/i })
    expect(governanceOption).toBeInTheDocument()

    const complianceOption = screen.getByRole('option', { name: /Compliance/i })
    expect(complianceOption).toBeInTheDocument()
  })

  it('should show warning for locked modes', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[1]) // Azure with LOCKED mode

    // Azure provider has LOCKED mode selected
    const warningTexts = screen.getAllByText(/Once locked, this policy cannot be removed or reduced/)
    expect(warningTexts.length).toBeGreaterThan(0)
  })

  it('should close modal on cancel', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    const cancelButton = screen.getByText('Cancel')
    fireEvent.click(cancelButton)

    expect(screen.queryByText('Configure AWS S3 Production')).not.toBeInTheDocument()
  })

  it('should render feature comparison table', () => {
    render(<ImmutableStorageConfig />)

    expect(screen.getByText('Feature Comparison')).toBeInTheDocument()
    expect(screen.getByText('S3 Object Lock')).toBeInTheDocument()
    expect(screen.getByText('Azure Immutable Blobs')).toBeInTheDocument()
    expect(screen.getByText('GCS Retention')).toBeInTheDocument()
  })

  it('should display feature comparison rows', () => {
    render(<ImmutableStorageConfig />)

    expect(screen.getByText('Protection Modes')).toBeInTheDocument()

    const legalHoldTexts = screen.getAllByText('Legal Hold')
    expect(legalHoldTexts.length).toBeGreaterThan(0)

    expect(screen.getByText('Bypass Protection')).toBeInTheDocument()
    expect(screen.getByText('Max Retention')).toBeInTheDocument()
  })

  it('should update retention days in modal', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    const retentionInput = screen.getByDisplayValue('30')
    fireEvent.change(retentionInput, { target: { value: '60' } })

    expect(screen.getByDisplayValue('60')).toBeInTheDocument()
  })

  it('should toggle immutability checkbox', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    const immutabilityCheckbox = screen.getByRole('checkbox', { name: /Enable Immutability/i })
    expect(immutabilityCheckbox).toBeChecked()

    fireEvent.click(immutabilityCheckbox)
    expect(immutabilityCheckbox).not.toBeChecked()
  })

  it('should toggle legal hold checkbox', () => {
    render(<ImmutableStorageConfig />)

    const configureButtons = screen.getAllByText('Configure')
    fireEvent.click(configureButtons[0])

    const legalHoldCheckbox = screen.getByRole('checkbox', { name: /Legal Hold/i })
    expect(legalHoldCheckbox).not.toBeChecked()

    fireEvent.click(legalHoldCheckbox)
    expect(legalHoldCheckbox).toBeChecked()
  })
})
