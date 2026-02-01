import { useResult } from '@/hooks/useResults'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { X, Loader2, CheckCircle2, XCircle, Clock } from 'lucide-react'

interface ResultDetailProps {
  id: string
  onClose: () => void
}

export function ResultDetail({ id, onClose }: ResultDetailProps) {
  const { data: result, isLoading } = useResult(id)

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
        <Card className="w-full max-w-3xl" onClick={(e) => e.stopPropagation()}>
          <CardContent className="flex items-center justify-center p-8">
            <Loader2 className="h-8 w-8 animate-spin" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!result) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <Card className="w-full max-w-3xl max-h-[90vh] overflow-auto" onClick={(e) => e.stopPropagation()}>
        <CardHeader className="flex flex-row items-center justify-between sticky top-0 bg-card z-10">
          <div>
            <CardTitle>Run Result</CardTitle>
            <p className="text-sm text-muted-foreground">{result.id}</p>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close">
            <X className="h-5 w-5" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Summary */}
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border border-border p-3 text-center">
              <div className="text-2xl font-bold">{result.totalScenarios}</div>
              <div className="text-xs text-muted-foreground">Total</div>
            </div>
            <div className="rounded-lg border border-border p-3 text-center">
              <div className="text-2xl font-bold text-green-500">{result.passed}</div>
              <div className="text-xs text-muted-foreground">Passed</div>
            </div>
            <div className="rounded-lg border border-border p-3 text-center">
              <div className="text-2xl font-bold text-red-500">{result.failed}</div>
              <div className="text-xs text-muted-foreground">Failed</div>
            </div>
            <div className="rounded-lg border border-border p-3 text-center">
              <div className="text-2xl font-bold text-yellow-500">{result.skipped}</div>
              <div className="text-xs text-muted-foreground">Skipped</div>
            </div>
          </div>

          {/* Duration */}
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="h-4 w-4" />
            Duration: {result.duration}
          </div>

          {/* Scenarios */}
          <div>
            <h4 className="mb-3 font-semibold">Scenario Results</h4>
            <div className="space-y-3">
              {result.scenarios?.map((scenario, index) => (
                <div key={index} className="rounded-lg border border-border p-4">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      {scenario.state === 'passed' ? (
                        <CheckCircle2 className="h-5 w-5 text-green-500" />
                      ) : (
                        <XCircle className="h-5 w-5 text-red-500" />
                      )}
                      <span className="font-medium">{scenario.scenarioName}</span>
                    </div>
                    <span className="text-sm text-muted-foreground">{scenario.duration}</span>
                  </div>

                  {scenario.error && (
                    <div className="mt-2 rounded bg-red-500/10 p-2 text-sm text-red-400">
                      {scenario.error}
                    </div>
                  )}

                  {/* Flow results */}
                  {scenario.flowResults && scenario.flowResults.length > 0 && (
                    <div className="mt-3 space-y-1">
                      {scenario.flowResults.map((flow: { name: string; type: string; state: string; duration: string }, flowIndex: number) => (
                        <div
                          key={flowIndex}
                          className="flex items-center justify-between text-sm py-1 px-2 rounded hover:bg-secondary/50"
                        >
                          <div className="flex items-center gap-2">
                            {flow.state === 'passed' ? (
                              <CheckCircle2 className="h-4 w-4 text-green-500" />
                            ) : (
                              <XCircle className="h-4 w-4 text-red-500" />
                            )}
                            <span>{flow.name}</span>
                            <Badge variant="outline" className="text-xs">
                              {flow.type}
                            </Badge>
                          </div>
                          <span className="text-muted-foreground">{flow.duration}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
