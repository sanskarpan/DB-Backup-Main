'use client'

import { FlaskConical } from 'lucide-react'
import { FeatureUnavailable } from '@/components/ui/feature-unavailable'

// The testing endpoints (/testing/*) are not served by the backend in this
// deployment. Rather than query nonexistent routes and error perpetually, we
// show an honest not-available state.
export default function TestingPage() {
  return (
    <FeatureUnavailable
      title="Testing Dashboard Not Available"
      icon={<FlaskConical className="w-8 h-8" />}
      description="Chaos, load, contract, and mutation testing dashboards are not enabled in this deployment. The backend does not expose the /testing endpoints required by this page."
    />
  )
}
