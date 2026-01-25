import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useScenarios, useRunScenario } from '@/hooks/useScenarios'
import { useRuns, useCancelRun } from '@/hooks/useRuns'
import { RunCard } from '@/components/runs/RunCard'
import { PlayCircle, RefreshCw, CheckCircle2, XCircle, Clock } from 'lucide-react'

export function Dashboard() {
  const [selectedScenario, setSelectedScenario] = useState<string>('')
  const { data: scenariosData } = useScenarios()
  const { data: runsData, isLoading: runsLoading, refetch: refetchRuns } = useRuns()
  const runScenario = useRunScenario()
  const cancelRun = useCancelRun()

  const activeRuns = runsData?.runs?.filter((r) => r.status === 'running') || []
  const recentRuns = runsData?.runs?.slice(0, 10) || []

  const handleRunScenario = () => {
    if (selectedScenario) {
      runScenario.mutate(selectedScenario)
      setSelectedScenario('')
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <Button variant="ghost" size="icon" onClick={() => refetchRuns()} aria-label="Refresh runs">
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Quick Actions</CardTitle>
        </CardHeader>
        <CardContent className="flex gap-4">
          <select
            className="flex h-10 w-64 rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={selectedScenario}
            onChange={(e) => setSelectedScenario(e.target.value)}
            aria-label="Select scenario"
          >
            <option value="">Select a scenario...</option>
            {scenariosData?.scenarios?.map((s) => (
              <option key={s.name} value={s.name}>
                {s.name}
              </option>
            ))}
          </select>
          <Button onClick={handleRunScenario} disabled={!selectedScenario || runScenario.isPending}>
            <PlayCircle className="mr-2 h-4 w-4" />
            Run Scenario
          </Button>
        </CardContent>
      </Card>

      {/* Active Runs */}
      {activeRuns.length > 0 && (
        <div>
          <h2 className="mb-4 text-xl font-semibold">Active Runs ({activeRuns.length})</h2>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {activeRuns.map((run) => (
              <RunCard key={run.id} run={run} onCancel={(id) => cancelRun.mutate(id)} />
            ))}
          </div>
        </div>
      )}

      {/* Statistics */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Scenarios
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{scenariosData?.count || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Active Runs</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeRuns.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Recent Passed
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">
              {recentRuns.filter((r) => r.status === 'completed').length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Recent Failed
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-500">
              {recentRuns.filter((r) => r.status === 'failed').length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Runs */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent Runs</CardTitle>
        </CardHeader>
        <CardContent>
          {runsLoading ? (
            <div className="text-muted-foreground">Loading...</div>
          ) : recentRuns.length === 0 ? (
            <div className="text-muted-foreground">No runs yet</div>
          ) : (
            <div className="space-y-2">
              {recentRuns.map((run) => (
                <div
                  key={run.id}
                  className="flex items-center justify-between rounded-lg border border-border p-3"
                >
                  <div className="flex items-center gap-3">
                    {run.status === 'completed' && (
                      <CheckCircle2 className="h-5 w-5 text-green-500" />
                    )}
                    {run.status === 'failed' && <XCircle className="h-5 w-5 text-red-500" />}
                    {run.status === 'running' && (
                      <Clock className="h-5 w-5 animate-pulse text-blue-500" />
                    )}
                    {run.status === 'canceled' && (
                      <XCircle className="h-5 w-5 text-gray-500" />
                    )}
                    <span className="font-medium">{run.scenario_id}</span>
                  </div>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    {run.duration && <span>{run.duration}</span>}
                    <span>{new Date(run.start_time).toLocaleTimeString()}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
