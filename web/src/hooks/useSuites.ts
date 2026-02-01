import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchSuites, fetchSuite } from '@/api/suites'
import { runBatch } from '@/api/scenarios'
import { fetchConfig, type SuiteConfig } from '@/api/local'
import { useMode } from '@/stores/mode'
import { useActiveProject } from '@/stores/projects'
import { toast } from '@/stores/toast'
import type { Suite, SuitesResponse } from '@/api/types'

export function useSuites() {
  const mode = useMode()
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['suites', mode, activeProject?.id],
    queryFn: async (): Promise<SuitesResponse> => {
      if (mode === 'standalone') {
        // In standalone mode, extract suites from local config
        const config = await fetchConfig()
        const suites: Suite[] = Object.entries(config.suites || {}).map(
          ([name, suite]: [string, SuiteConfig]) => ({
            name,
            projectId: activeProject?.id || '',
            description: suite.description,
            scenarios: suite.scenarios || [],
            tags: suite.tags,
            excludeTags: suite.exclude_tags,
            parallel: suite.parallel,
            failFast: suite.fail_fast,
          })
        )
        return {
          projectId: activeProject?.id || '',
          suites,
          count: suites.length,
        }
      }

      // Daemon mode
      if (!activeProject?.daemonUrl) {
        return { projectId: '', suites: [], count: 0 }
      }

      const response = await fetchSuites(activeProject.daemonUrl)
      return {
        projectId: activeProject.id,
        suites: response.suites.map(s => ({
          ...s,
          projectId: activeProject.id,
        })),
        count: response.count,
      }
    },
  })
}

export function useSuite(name: string) {
  const mode = useMode()
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['suites', name, mode, activeProject?.id],
    queryFn: async () => {
      if (mode === 'standalone') {
        const config = await fetchConfig()
        const suite = config.suites?.[name]
        if (!suite) throw new Error(`Suite ${name} not found`)
        return {
          name,
          projectId: activeProject?.id || '',
          description: suite.description,
          scenarios: suite.scenarios || [],
          tags: suite.tags,
          excludeTags: suite.exclude_tags,
          parallel: suite.parallel,
          failFast: suite.fail_fast,
        }
      }

      // Daemon mode
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }

      const suite = await fetchSuite(activeProject.daemonUrl, name)
      return {
        ...suite,
        projectId: activeProject.id,
      }
    },
    enabled: !!name,
  })
}

export function useRunSuite() {
  const queryClient = useQueryClient()
  const activeProject = useActiveProject()

  return useMutation({
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    mutationFn: async (_suiteName: string) => {
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }
      // Run suite as a batch run with the suite name
      // TODO: Backend needs to support suite name parameter in runBatch
      return runBatch({
        daemonUrl: activeProject.daemonUrl,
        projectId: activeProject.id,
        scenarioNames: [], // Empty array - suite name is handled by backend
      })
    },
    onSuccess: (_data, suiteName) => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      toast.success('Suite started', `Running suite "${suiteName}"`)
    },
    onError: (error, suiteName) => {
      toast.error('Failed to start suite', `Could not run suite "${suiteName}": ${error.message}`)
    },
  })
}
