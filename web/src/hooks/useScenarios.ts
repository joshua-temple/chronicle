import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchScenarios, fetchScenario } from '@/api/scenarios'
import { fetchConfig, type FlowItemConfig } from '@/api/local'
import { createRun } from '@/api/runs'
import { useMode } from '@/stores/mode'
import { toast } from '@/stores/toast'
import type { Scenario } from '@/api/types'

function getFlowItemName(f: FlowItemConfig): string {
  return f.setup || f.task || f.validation || f.step || f.rollup || f.teardown || 'unknown'
}

function getFlowItemType(f: FlowItemConfig): string {
  if (f.setup) return 'setup'
  if (f.task) return 'task'
  if (f.validation) return 'validation'
  if (f.step) return 'step'
  if (f.rollup) return 'rollup'
  if (f.teardown) return 'teardown'
  return 'unknown'
}

export function useScenarios() {
  const mode = useMode()

  return useQuery({
    queryKey: ['scenarios', mode],
    queryFn: async (): Promise<{ scenarios: Scenario[]; count: number }> => {
      if (mode === 'standalone') {
        // In standalone mode, extract scenarios from local config
        const config = await fetchConfig()
        const scenarios: Scenario[] = (config.scenarios || [])
          .filter(s => !s.abstract) // Don't show abstract scenarios
          .map(s => ({
            name: s.name,
            description: s.description,
            tags: s.tags,
            timeout: s.timeout ? `${s.timeout}ns` : undefined,
            flow_count: s.flow?.length || 0,
          }))
        return { scenarios, count: scenarios.length }
      }
      return fetchScenarios()
    },
  })
}

export function useScenario(name: string) {
  const mode = useMode()

  return useQuery({
    queryKey: ['scenarios', name, mode],
    queryFn: async () => {
      if (mode === 'standalone') {
        const config = await fetchConfig()
        const scenario = config.scenarios?.find(s => s.name === name)
        if (!scenario) throw new Error(`Scenario ${name} not found`)
        return {
          name: scenario.name,
          description: scenario.description,
          tags: scenario.tags,
          timeout: scenario.timeout ? `${scenario.timeout}ns` : undefined,
          flow_count: scenario.flow?.length || 0,
          flow: scenario.flow?.map(f => ({
            name: getFlowItemName(f),
            type: getFlowItemType(f),
            component: getFlowItemName(f),
          })) || [],
        }
      }
      return fetchScenario(name)
    },
    enabled: !!name,
  })
}

export function useRunScenario() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (scenarioName: string) => createRun(scenarioName),
    onSuccess: (_data, scenarioName) => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      toast.success('Scenario started', `Running "${scenarioName}"`)
    },
    onError: (error, scenarioName) => {
      toast.error('Failed to start scenario', `Could not run "${scenarioName}": ${error.message}`)
    },
  })
}
