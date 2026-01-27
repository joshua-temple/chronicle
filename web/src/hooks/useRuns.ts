import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchRuns, fetchRun, cancelRun } from '@/api/runs'
import { runBatch } from '@/api/scenarios'
import { toast } from '@/stores/toast'
import { useActiveProject } from '@/stores/projects'
import type { BatchRunRequest, RunsResponse, Run } from '@/api/types'

export function useRuns() {
  const activeProject = useActiveProject()
  const daemonUrl = activeProject?.daemonUrl

  return useQuery({
    queryKey: ['runs', activeProject?.id],
    queryFn: async (): Promise<RunsResponse> => {
      if (!daemonUrl) {
        return { runs: [], count: 0 }
      }
      return fetchRuns(daemonUrl)
    },
    refetchInterval: 2000,
    enabled: !!daemonUrl,
  })
}

export function useRun(id: string) {
  const activeProject = useActiveProject()
  const daemonUrl = activeProject?.daemonUrl

  return useQuery({
    queryKey: ['runs', activeProject?.id, id],
    queryFn: async (): Promise<Run> => {
      if (!daemonUrl) {
        throw new Error('No active project')
      }
      return fetchRun(daemonUrl, id)
    },
    enabled: !!id && !!daemonUrl,
    refetchInterval: (query) => (query.state.data?.status === 'running' ? 1000 : false),
  })
}

export function useCancelRun() {
  const queryClient = useQueryClient()
  const activeProject = useActiveProject()

  return useMutation({
    mutationFn: async (runId: string) => {
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }
      return cancelRun(activeProject.daemonUrl, runId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      toast.info('Run cancelled', 'The scenario run has been stopped')
    },
    onError: (error) => {
      toast.error('Failed to cancel', error.message)
    },
  })
}

export function useRunBatch() {
  const queryClient = useQueryClient()
  const activeProject = useActiveProject()

  return useMutation({
    mutationFn: async (request: Partial<BatchRunRequest>) => {
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }
      // Use runBatch from scenarios API with scenario names
      return runBatch({
        daemonUrl: activeProject.daemonUrl,
        projectId: activeProject.id,
        scenarioNames: request.scenarios || [],
      })
    },
    onSuccess: (_data, request) => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      const count = request.scenarios?.length || 0
      const description = count > 0
        ? `Running ${count} scenario(s)`
        : 'Running batch'
      toast.success('Batch started', description)
    },
    onError: (error) => {
      toast.error('Failed to start batch', error.message)
    },
  })
}
