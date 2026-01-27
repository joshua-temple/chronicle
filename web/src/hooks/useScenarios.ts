import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchScenarios, fetchScenario, runScenario } from '@/api/scenarios'
import { fetchConfig, type FlowItemConfig } from '@/api/local'
import { useMode } from '@/stores/mode'
import { useActiveProject } from '@/stores/projects'
import { toast } from '@/stores/toast'
import type { Scenario, ScenariosResponse } from '@/api/types'

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
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['scenarios', mode, activeProject?.id],
    queryFn: async (): Promise<ScenariosResponse> => {
      if (mode === 'standalone') {
        // In standalone mode, extract scenarios from local config
        const config = await fetchConfig()
        const scenarios: Scenario[] = (config.scenarios || [])
          .filter(s => !s.abstract) // Don't show abstract scenarios
          .map(s => ({
            name: s.name,
            projectId: activeProject?.id || '',
            description: s.description,
            tags: s.tags,
            timeout: s.timeout ? `${s.timeout}ns` : undefined,
            flowCount: s.flow?.length || 0,
          }))
        return {
          projectId: activeProject?.id || '',
          scenarios,
          count: scenarios.length,
        }
      }

      // Daemon mode
      if (!activeProject?.daemonUrl) {
        return { projectId: '', scenarios: [], count: 0 }
      }

      const response = await fetchScenarios(activeProject.daemonUrl)
      return {
        projectId: activeProject.id,
        scenarios: response.scenarios.map(s => ({
          ...s,
          projectId: activeProject.id,
        })),
        count: response.count,
      }
    },
  })
}

export function useScenario(name: string) {
  const mode = useMode()
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['scenarios', name, mode, activeProject?.id],
    queryFn: async () => {
      if (mode === 'standalone') {
        const config = await fetchConfig()
        const scenario = config.scenarios?.find(s => s.name === name)
        if (!scenario) throw new Error(`Scenario ${name} not found`)
        return {
          name: scenario.name,
          projectId: activeProject?.id || '',
          description: scenario.description,
          tags: scenario.tags,
          timeout: scenario.timeout ? `${scenario.timeout}ns` : undefined,
          flowCount: scenario.flow?.length || 0,
          flow: scenario.flow?.map(f => ({
            name: getFlowItemName(f),
            type: getFlowItemType(f) as 'setup' | 'task' | 'validation' | 'teardown' | 'step' | 'rollup',
            plugin: getFlowItemName(f),
          })) || [],
        }
      }

      // Daemon mode
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }

      const scenario = await fetchScenario(activeProject.daemonUrl, name)
      return {
        ...scenario,
        projectId: activeProject.id,
      }
    },
    enabled: !!name,
  })
}

export function useRunScenario() {
  const queryClient = useQueryClient()
  const activeProject = useActiveProject()

  return useMutation({
    mutationFn: async (scenarioName: string) => {
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }
      return runScenario({
        daemonUrl: activeProject.daemonUrl,
        projectId: activeProject.id,
        scenarioName,
      })
    },
    onSuccess: (_data, scenarioName) => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      toast.success('Scenario started', `Running "${scenarioName}"`)
    },
    onError: (error, scenarioName) => {
      toast.error('Failed to start scenario', `Could not run "${scenarioName}": ${error.message}`)
    },
  })
}
