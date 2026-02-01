import { type ReactNode } from 'react'
import { FileQuestion, Search, FolderOpen, AlertCircle } from 'lucide-react'
import { Button } from './button'
import { cn } from '@/lib/utils'

type EmptyStateVariant = 'default' | 'search' | 'error' | 'empty'

const VARIANT_ICONS: Record<EmptyStateVariant, typeof FileQuestion> = {
  default: FolderOpen,
  search: Search,
  error: AlertCircle,
  empty: FileQuestion,
}

const VARIANT_COLORS: Record<EmptyStateVariant, string> = {
  default: 'text-muted-foreground',
  search: 'text-muted-foreground',
  error: 'text-destructive',
  empty: 'text-muted-foreground',
}

interface EmptyStateProps {
  variant?: EmptyStateVariant
  icon?: ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export function EmptyState({
  variant = 'default',
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  const Icon = VARIANT_ICONS[variant]
  const iconColor = VARIANT_COLORS[variant]

  return (
    <div className={cn('flex flex-col items-center justify-center py-12 px-4', className)}>
      <div className={cn('mb-4 rounded-full bg-muted p-4', iconColor)}>
        {icon || <Icon className="h-8 w-8" />}
      </div>
      <h3 className="mb-1 text-lg font-semibold">{title}</h3>
      {description && (
        <p className="mb-4 max-w-sm text-center text-sm text-muted-foreground">
          {description}
        </p>
      )}
      {action && (
        <Button variant="outline" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  )
}
