import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchRuns, fetchRun, cancelRun } from '@/api/runs'
import { toast } from '@/stores/toast'

export function useRuns() {
  return useQuery({
    queryKey: ['runs'],
    queryFn: fetchRuns,
    refetchInterval: 2000,
  })
}

export function useRun(id: string) {
  return useQuery({
    queryKey: ['runs', id],
    queryFn: () => fetchRun(id),
    enabled: !!id,
    refetchInterval: (query) => (query.state.data?.status === 'running' ? 1000 : false),
  })
}

export function useCancelRun() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: cancelRun,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['runs'] })
      toast.info('Run cancelled', 'The scenario run has been stopped')
    },
    onError: (error) => {
      toast.error('Failed to cancel', error.message)
    },
  })
}
