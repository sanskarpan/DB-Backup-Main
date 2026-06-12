import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SecurityStats } from '../../security/security-stats'

describe('SecurityStats', () => {
  it('should render all security stat cards', () => {
    render(<SecurityStats />)

    expect(screen.getByText('Protected Backups')).toBeInTheDocument()
    expect(screen.getByText('Active Scans')).toBeInTheDocument()
    expect(screen.getByText('Threats Blocked')).toBeInTheDocument()
    expect(screen.getByText('Security Score')).toBeInTheDocument()
  })

  it('should display correct stat values', () => {
    render(<SecurityStats />)

    expect(screen.getByText('156')).toBeInTheDocument()
    expect(screen.getByText('24/7')).toBeInTheDocument()
    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getByText('98%')).toBeInTheDocument()
  })

  it('should display stat changes', () => {
    render(<SecurityStats />)

    expect(screen.getByText('+12%')).toBeInTheDocument()
    expect(screen.getByText('Continuous')).toBeInTheDocument()
    expect(screen.getByText('Last 30 days')).toBeInTheDocument()
    expect(screen.getByText('+3%')).toBeInTheDocument()
  })

  it('should render with proper grid layout', () => {
    const { container } = render(<SecurityStats />)

    const gridContainer = container.querySelector('.grid')
    expect(gridContainer).toBeInTheDocument()
    expect(gridContainer).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-4')
  })

  it('should render stat cards with hover effects', () => {
    const { container } = render(<SecurityStats />)

    const cards = container.querySelectorAll('.hover\\:shadow-lg')
    expect(cards.length).toBe(4)
  })
})
