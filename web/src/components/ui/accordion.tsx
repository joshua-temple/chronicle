import * as React from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

interface AccordionProps {
  children: React.ReactNode
  className?: string
}

export function Accordion({ children, className }: AccordionProps) {
  return (
    <div className={cn('space-y-1', className)}>
      {children}
    </div>
  )
}

interface AccordionItemProps {
  children: React.ReactNode
  className?: string
}

export function AccordionItem({ children, className }: AccordionItemProps) {
  return (
    <div className={cn('', className)}>
      {children}
    </div>
  )
}

interface AccordionTriggerProps {
  children: React.ReactNode
  expanded: boolean
  onToggle: () => void
  className?: string
  icon?: React.ReactNode
  actions?: React.ReactNode
}

export function AccordionTrigger({
  children,
  expanded,
  onToggle,
  className,
  icon,
  actions,
}: AccordionTriggerProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm font-medium',
        'hover:bg-accent hover:text-accent-foreground',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        className
      )}
    >
      <ChevronRight
        className={cn(
          'h-4 w-4 shrink-0 transition-transform duration-200',
          expanded && 'rotate-90'
        )}
      />
      {icon && <span className="shrink-0">{icon}</span>}
      <span className="flex-1 truncate text-left">{children}</span>
      {actions && (
        <span
          className="shrink-0"
          onClick={(e) => e.stopPropagation()}
        >
          {actions}
        </span>
      )}
    </button>
  )
}

interface AccordionContentProps {
  children: React.ReactNode
  expanded: boolean
  className?: string
}

export function AccordionContent({
  children,
  expanded,
  className,
}: AccordionContentProps) {
  if (!expanded) return null

  return (
    <div className={cn('pl-4', className)}>
      {children}
    </div>
  )
}
