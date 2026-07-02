import { FlaskConical } from 'lucide-react'

/**
 * Visible marker for UI that renders illustrative/sample data rather than
 * values from a real backend source. Used so demo numbers are never presented
 * as live metrics.
 */
export function SampleDataBadge({ className = '' }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950 dark:text-amber-400 dark:border-amber-800 ${className}`}
      title="This panel shows sample data, not live backend metrics."
    >
      <FlaskConical className="w-3 h-3" />
      Sample data
    </span>
  )
}
