import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatsCard } from '../../dashboard/stats-card'
import { Database } from 'lucide-react'

describe('StatsCard', () => {
  it('should render title and value', () => {
    render(
      <StatsCard
        title="Total Backups"
        value="42"
        icon={Database}
      />
    )

    expect(screen.getByText('Total Backups')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('should render trend when provided', () => {
    render(
      <StatsCard
        title="Success Rate"
        value="98.5%"
        icon={Database}
        trend="+5% from last month"
        trendUp={true}
      />
    )

    expect(screen.getByText('+5% from last month')).toBeInTheDocument()
  })

  it('should apply correct color for trend up', () => {
    const { container } = render(
      <StatsCard
        title="Test"
        value="100"
        icon={Database}
        trend="+10%"
        trendUp={true}
      />
    )

    const trendElement = container.querySelector('.text-green-600')
    expect(trendElement).toBeInTheDocument()
  })

  it('should apply correct color for trend down', () => {
    const { container } = render(
      <StatsCard
        title="Test"
        value="100"
        icon={Database}
        trend="-10%"
        trendUp={false}
      />
    )

    const trendElement = container.querySelector('.text-red-600')
    expect(trendElement).toBeInTheDocument()
  })

  it('should not render trend when not provided', () => {
    const { container } = render(
      <StatsCard
        title="Test"
        value="100"
        icon={Database}
      />
    )

    expect(container.querySelector('.text-green-600')).not.toBeInTheDocument()
    expect(container.querySelector('.text-red-600')).not.toBeInTheDocument()
  })
})
