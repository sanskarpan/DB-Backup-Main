import * as React from 'react'
import { clsx } from 'clsx'

export interface CheckboxProps extends React.InputHTMLAttributes<HTMLInputElement> {
  onCheckedChange?: (checked: boolean) => void
}

export const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, onCheckedChange, ...props }, ref) => {
    return (
      <input
        type="checkbox"
        className={clsx(
          'h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary focus:ring-2',
          className
        )}
        onChange={(e) => onCheckedChange?.(e.target.checked)}
        ref={ref}
        {...props}
      />
    )
  }
)

Checkbox.displayName = 'Checkbox'
