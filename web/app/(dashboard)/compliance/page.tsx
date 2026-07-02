'use client'

import { Shield } from 'lucide-react'
import { FeatureUnavailable } from '@/components/ui/feature-unavailable'

// The compliance/governance endpoints (/compliance/*) are not served by the
// backend in this deployment. The previous implementation rendered only
// hardcoded sample data (consents, erasure requests, policy evaluations) with
// no real source, so it is gated behind an honest not-available state.
export default function CompliancePage() {
  return (
    <FeatureUnavailable
      title="Compliance & Governance Not Available"
      icon={<Shield className="w-8 h-8" />}
      description="GDPR consent, data residency, and policy engine dashboards are not enabled in this deployment. The backend does not expose the /compliance endpoints required by this page."
    />
  )
}
