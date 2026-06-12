'use client'

import * as React from 'react'
import { clsx } from 'clsx'

interface SelectContextValue {
  value: string
  onValueChange: (value: string) => void
}

const SelectContext = React.createContext<SelectContextValue | undefined>(undefined)

export interface SelectProps {
  value?: string
  defaultValue?: string
  onValueChange?: (value: string) => void
  children: React.ReactNode
}

export function Select({ value, defaultValue, onValueChange, children }: SelectProps) {
  const [internalValue, setInternalValue] = React.useState(value || defaultValue || '')

  React.useEffect(() => {
    if (value !== undefined) {
      setInternalValue(value)
    }
  }, [value])

  const handleChange = (newValue: string) => {
    if (value === undefined) {
      setInternalValue(newValue)
    }
    onValueChange?.(newValue)
  }

  return (
    <SelectContext.Provider value={{ value: internalValue, onValueChange: handleChange }}>
      {children}
    </SelectContext.Provider>
  )
}

export interface SelectTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  className?: string
}

export function SelectTrigger({ className, children, ...props }: SelectTriggerProps) {
  return (
    <button
      type="button"
      className={clsx(
        'flex h-10 w-full items-center justify-between rounded-md border border-gray-300 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
        className
      )}
      {...props}
    >
      {children}
    </button>
  )
}

export interface SelectValueProps {
  placeholder?: string
}

export function SelectValue({ placeholder }: SelectValueProps) {
  const context = React.useContext(SelectContext)
  return <span>{context?.value || placeholder}</span>
}

export interface SelectContentProps {
  children: React.ReactNode
}

export function SelectContent({ children }: SelectContentProps) {
  return (
    <div className="relative z-50 min-w-[8rem] overflow-hidden rounded-md border border-gray-200 bg-white text-gray-900 shadow-md">
      {children}
    </div>
  )
}

export interface SelectItemProps extends React.HTMLAttributes<HTMLDivElement> {
  value: string
  children: React.ReactNode
}

export function SelectItem({ value, children, className, ...props }: SelectItemProps) {
  const context = React.useContext(SelectContext)
  if (!context) return null

  return (
    <div
      onClick={() => context.onValueChange(value)}
      className={clsx(
        'relative flex cursor-pointer select-none items-center rounded-sm py-1.5 px-2 text-sm outline-none hover:bg-gray-100 focus:bg-gray-100',
        context.value === value && 'bg-gray-100',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}
