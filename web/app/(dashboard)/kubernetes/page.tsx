'use client'

import { Boxes } from 'lucide-react'
import { FeatureUnavailable } from '@/components/ui/feature-unavailable'

// The Kubernetes operator endpoints (/kubernetes/*) are not served by the
// backend in this deployment. Rather than query nonexistent routes and error
// perpetually, we show an honest not-available state.
export default function KubernetesPage() {
  return (
    <FeatureUnavailable
      title="Kubernetes Operator Not Available"
      icon={<Boxes className="w-8 h-8" />}
      description="Kubernetes operator management is not enabled in this deployment. The backend does not expose the /kubernetes endpoints required by this page."
    />
  )
}
