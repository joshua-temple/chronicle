import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchScenarios, fetchScenario } from '@/api/scenarios'
import { createRun } from '@/api/runs'

export function useScenarios() {
  return useQuery({
    queryKey: ['scenarios'],
    queryFn: fetchScenarios,
  })
}

export function useScenario(name: string) {
  return useQuery({
    queryKey: ['scenarios', name],
    queryFn: () => fetchScenario(name),
    enabled: !!name,
  })
}

export function useRunScenario() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (scenarioName: string) => createRun(scenarioName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
    },
  })
}
