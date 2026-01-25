import { useScenario, useRunScenario } from '@/hooks/useScenarios'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { X, PlayCircle, Loader2 } from 'lucide-react'

interface ScenarioDetailProps {
  name: string
  onClose: () => void
}

export function ScenarioDetail({ name, onClose }: ScenarioDetailProps) {
  const { data: scenario, isLoading } = useScenario(name)
  const runScenario = useRunScenario()

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
        <Card className="w-full max-w-2xl" onClick={(e) => e.stopPropagation()}>
          <CardContent className="flex items-center justify-center p-8">
            <Loader2 className="h-8 w-8 animate-spin" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!scenario) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <Card className="w-full max-w-2xl max-h-[80vh] overflow-auto" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{scenario.name}</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close">
            <X className="h-5 w-5" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {scenario.description && (
            <p className="text-muted-foreground">{scenario.description}</p>
          )}

          {scenario.tags && scenario.tags.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {scenario.tags.map((tag) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ))}
            </div>
          )}

          <div>
            <h4 className="mb-2 font-semibold">Flow ({scenario.flow?.length || 0} steps)</h4>
            <div className="space-y-2">
              {scenario.flow?.map((step, index) => (
                <div
                  key={index}
                  className="flex items-center gap-3 rounded-lg border border-border p-3"
                >
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-secondary text-xs">
                    {index + 1}
                  </span>
                  <div>
                    <div className="font-medium">{step.name || step.component}</div>
                    <div className="text-xs text-muted-foreground">{step.type}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={onClose}>
              Close
            </Button>
            <Button
              onClick={() => {
                runScenario.mutate(scenario.name)
                onClose()
              }}
              disabled={runScenario.isPending}
            >
              <PlayCircle className="mr-2 h-4 w-4" />
              Run Scenario
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
