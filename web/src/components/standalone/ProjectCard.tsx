import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Trash2, ExternalLink, Globe, RefreshCw, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Project, ProjectConnectionStatus } from '@/api/types'

interface ProjectCardProps {
  project: Project
  onOpen: (id: string) => void
  onConnect: (id: string) => Promise<void>
  onDisconnect: (id: string) => void
  onRemove: (id: string) => void
  disabled?: boolean
  error?: string
}

const STATUS_COLORS: Record<ProjectConnectionStatus, string> = {
  connected: 'bg-green-500',
  connecting: 'bg-yellow-500 animate-pulse',
  disconnected: 'bg-gray-400',
  error: 'bg-red-500',
}

const STATUS_LABELS: Record<ProjectConnectionStatus, string> = {
  connected: 'Connected',
  connecting: 'Connecting...',
  disconnected: 'Disconnected',
  error: 'Error',
}

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return 'Never'

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} min ago`
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`

  return date.toLocaleDateString()
}

export function ProjectCard({
  project,
  onOpen,
  onConnect,
  onDisconnect,
  onRemove,
  disabled = false,
  error,
}: ProjectCardProps) {
  const [showRemoveConfirm, setShowRemoveConfirm] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  const [localError, setLocalError] = useState<string | null>(null)

  const isConnected = project.status === 'connected'
  const isConnectingStatus = project.status === 'connecting'
  const hasError = project.status === 'error'

  // Combined error from props or local operations
  const displayError = localError || error

  const handleConnect = async () => {
    setLocalError(null)
    setIsConnecting(true)
    try {
      await onConnect(project.id)
    } catch (err) {
      if (err instanceof Error) {
        setLocalError(err.message)
      }
    } finally {
      setIsConnecting(false)
    }
  }

  const handleDisconnect = () => {
    setLocalError(null)
    onDisconnect(project.id)
  }

  const handleRemoveClick = () => {
    if (showRemoveConfirm) {
      onRemove(project.id)
      setShowRemoveConfirm(false)
    } else {
      setShowRemoveConfirm(true)
    }
  }

  const handleCancelRemove = () => {
    setShowRemoveConfirm(false)
  }

  return (
    <Card className="transition-colors hover:bg-secondary/30">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4">
          {/* Left: Status and Info */}
          <div className="flex items-start gap-3 min-w-0 flex-1">
            {/* Status Indicator */}
            <div
              className={cn(
                'mt-1.5 h-3 w-3 shrink-0 rounded-full',
                STATUS_COLORS[project.status]
              )}
              title={STATUS_LABELS[project.status]}
              role="status"
              aria-label={`Status: ${STATUS_LABELS[project.status]}`}
            />

            {/* Project Info */}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-foreground truncate">{project.name}</h3>
                <span title="Daemon URL">
                  <Globe className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                </span>
              </div>
              <p className="text-sm text-muted-foreground truncate">
                {project.daemonUrl}
              </p>
              {project.description && (
                <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                  {project.description}
                </p>
              )}
            </div>
          </div>

          {/* Right: Status Info */}
          <div className="text-right shrink-0">
            <p className="text-sm font-medium">
              {isConnecting || isConnectingStatus
                ? 'Connecting...'
                : STATUS_LABELS[project.status]}
            </p>
            <p className="text-xs text-muted-foreground">
              Last connected: {formatRelativeTime(project.lastConnected)}
            </p>
          </div>
        </div>

        {/* Error Display with Retry */}
        {displayError && (
          <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 p-3">
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 text-destructive shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="text-sm text-destructive">{displayError}</p>
              </div>
              {!isConnected && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleConnect}
                  disabled={disabled || isConnecting}
                  className="shrink-0"
                >
                  <RefreshCw className={cn('h-3 w-3 mr-1', isConnecting && 'animate-spin')} />
                  Retry
                </Button>
              )}
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="mt-4 flex items-center justify-end gap-2">
          {showRemoveConfirm ? (
            <>
              <span className="text-sm text-muted-foreground mr-2">Remove project?</span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCancelRemove}
                disabled={disabled}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={handleRemoveClick}
                disabled={disabled}
              >
                Confirm
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowRemoveConfirm(true)}
                disabled={disabled}
                aria-label={`Remove ${project.name}`}
                title="Remove project"
              >
                <Trash2 className="h-4 w-4" />
              </Button>

              {isConnected ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleDisconnect}
                  disabled={disabled}
                >
                  Disconnect
                </Button>
              ) : hasError ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleConnect}
                  disabled={disabled || isConnecting}
                >
                  <RefreshCw className={cn('h-4 w-4 mr-1', isConnecting && 'animate-spin')} />
                  Retry
                </Button>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleConnect}
                  disabled={disabled || isConnecting || isConnectingStatus}
                >
                  <RefreshCw className={cn('h-4 w-4 mr-1', (isConnecting || isConnectingStatus) && 'animate-spin')} />
                  {isConnecting || isConnectingStatus ? 'Connecting...' : 'Connect'}
                </Button>
              )}

              <Button
                variant="default"
                size="sm"
                onClick={() => onOpen(project.id)}
                disabled={disabled || isConnecting || !isConnected}
                title={!isConnected ? 'Connect to project first' : undefined}
              >
                <ExternalLink className="h-4 w-4 mr-1" />
                Open
              </Button>
            </>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
