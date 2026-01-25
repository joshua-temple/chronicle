import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Play, Square, Trash2, ExternalLink, Globe, Folder, RefreshCw, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Project, ProjectState } from '@/stores/projects'

interface ProjectCardProps {
  project: Project
  onOpen: (id: string) => void
  onLaunch: (id: string) => Promise<void>
  onStop: (id: string) => Promise<void>
  onRemove: (id: string) => Promise<void>
  disabled?: boolean
}

const STATUS_COLORS: Record<ProjectState, string> = {
  running: 'bg-green-500',
  stopped: 'bg-gray-400',
  starting: 'bg-yellow-500 animate-pulse',
  unhealthy: 'bg-red-500',
  unknown: 'bg-gray-300',
}

const STATUS_LABELS: Record<ProjectState, string> = {
  running: 'Running',
  stopped: 'Stopped',
  starting: 'Starting',
  unhealthy: 'Unhealthy',
  unknown: 'Unknown',
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
  onLaunch,
  onStop,
  onRemove,
  disabled = false,
}: ProjectCardProps) {
  const [showRemoveConfirm, setShowRemoveConfirm] = useState(false)
  const [isRetrying, setIsRetrying] = useState(false)
  const [localError, setLocalError] = useState<string | null>(null)
  const isLocal = Boolean(project.path)
  const isRemote = Boolean(project.remoteUrl)
  const isRunning = project.status.state === 'running'
  const isStarting = project.status.state === 'starting'
  const isUnhealthy = project.status.state === 'unhealthy'
  const canControl = isLocal && !isStarting && !isRetrying

  // Combined error from status or local operations
  const displayError = localError || project.status.error

  const handleLaunch = async () => {
    setLocalError(null)
    setIsRetrying(true)
    try {
      await onLaunch(project.id)
    } catch (error) {
      // Error is already handled by the store, but we can show local feedback
      if (error instanceof Error) {
        setLocalError(error.message)
      }
    } finally {
      setIsRetrying(false)
    }
  }

  const handleRetryLaunch = async () => {
    setLocalError(null)
    await handleLaunch()
  }

  const handleStop = async () => {
    setLocalError(null)
    setIsRetrying(true)
    try {
      await onStop(project.id)
    } catch (error) {
      if (error instanceof Error) {
        setLocalError(error.message)
      }
    } finally {
      setIsRetrying(false)
    }
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
                STATUS_COLORS[project.status.state]
              )}
              title={STATUS_LABELS[project.status.state]}
              role="status"
              aria-label={`Status: ${STATUS_LABELS[project.status.state]}`}
            />

            {/* Project Info */}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-foreground truncate">{project.name}</h3>
                {isRemote && (
                  <span title="Remote daemon">
                    <Globe className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                  </span>
                )}
                {isLocal && (
                  <span title="Local project">
                    <Folder className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                  </span>
                )}
              </div>
              <p className="text-sm text-muted-foreground truncate">
                {project.path || project.remoteUrl}
              </p>
            </div>
          </div>

          {/* Right: Status Info */}
          <div className="text-right shrink-0">
            <p className="text-sm font-medium">
              {isRetrying
                ? 'Processing...'
                : isRunning && project.status.port
                  ? `Running on :${project.status.port}`
                  : STATUS_LABELS[project.status.state]}
            </p>
            <p className="text-xs text-muted-foreground">
              Last run: {formatRelativeTime(project.lastOpened)}
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
              {isLocal && !isRunning && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleRetryLaunch}
                  disabled={disabled || isRetrying}
                  className="shrink-0"
                >
                  <RefreshCw className={cn('h-3 w-3 mr-1', isRetrying && 'animate-spin')} />
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

              {canControl && (
                isRunning ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleStop}
                    disabled={disabled || isRetrying}
                  >
                    <Square className="h-4 w-4 mr-1" />
                    Stop
                  </Button>
                ) : isUnhealthy ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleRetryLaunch}
                    disabled={disabled || isRetrying}
                  >
                    <RefreshCw className={cn('h-4 w-4 mr-1', isRetrying && 'animate-spin')} />
                    Retry
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleLaunch}
                    disabled={disabled || isRetrying}
                  >
                    <Play className={cn('h-4 w-4 mr-1', isRetrying && 'animate-spin')} />
                    {isRetrying ? 'Launching...' : 'Launch'}
                  </Button>
                )
              )}

              <Button
                variant="default"
                size="sm"
                onClick={() => onOpen(project.id)}
                disabled={disabled || isRetrying || (!isRunning && !isRemote)}
                title={!isRunning && !isRemote ? 'Launch project first' : undefined}
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
