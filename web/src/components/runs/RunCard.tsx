import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Loader2, XCircle, CheckCircle2, AlertCircle } from 'lucide-react'
import type { Run } from '@/api/types'

interface RunCardProps {
  run: Run
  onCancel?: (id: string) => void
}

export function RunCard({ run, onCancel }: RunCardProps) {
  const statusConfig = {
    running: { icon: Loader2, color: 'default' as const, animate: true },
    completed: { icon: CheckCircle2, color: 'success' as const, animate: false },
    failed: { icon: AlertCircle, color: 'destructive' as const, animate: false },
    canceled: { icon: XCircle, color: 'secondary' as const, animate: false },
  }

  const config = statusConfig[run.status] || statusConfig.running
  const Icon = config.icon

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{run.scenario_id}</CardTitle>
          <Badge variant={config.color}>
            <Icon className={`mr-1 h-3 w-3 ${config.animate ? 'animate-spin' : ''}`} />
            {run.status}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-sm text-muted-foreground">
          Started: {new Date(run.start_time).toLocaleTimeString()}
        </div>
        {run.duration && (
          <div className="text-sm text-muted-foreground">Duration: {run.duration}</div>
        )}
        {run.status === 'running' && onCancel && (
          <Button
            variant="ghost"
            size="sm"
            className="mt-2"
            onClick={() => onCancel(run.id)}
          >
            Cancel
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
