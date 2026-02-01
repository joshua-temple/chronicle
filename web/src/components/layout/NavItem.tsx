import * as React from 'react'
import { Link, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'

interface NavItemProps {
  to: string
  icon?: React.ReactNode
  children: React.ReactNode
  className?: string
  indent?: number
  badge?: React.ReactNode
  onClick?: () => void
}

export function NavItem({
  to,
  icon,
  children,
  className,
  indent = 0,
  badge,
  onClick,
}: NavItemProps) {
  const location = useLocation()
  const isActive = location.pathname === to

  return (
    <Link
      to={to}
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 rounded-md px-2 py-1.5 text-sm',
        'hover:bg-accent hover:text-accent-foreground',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isActive && 'bg-accent text-accent-foreground',
        className
      )}
      style={{ paddingLeft: `${(indent * 12) + 8}px` }}
    >
      {icon && <span className="shrink-0">{icon}</span>}
      <span className="flex-1 truncate">{children}</span>
      {badge && <span className="shrink-0">{badge}</span>}
    </Link>
  )
}

interface NavGroupProps {
  label: string
  children: React.ReactNode
  className?: string
}

export function NavGroup({ label, children, className }: NavGroupProps) {
  return (
    <div className={cn('space-y-1', className)}>
      <div className="px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      {children}
    </div>
  )
}

interface NavDividerProps {
  className?: string
}

export function NavDivider({ className }: NavDividerProps) {
  return <div className={cn('my-2 h-px bg-border', className)} />
}
