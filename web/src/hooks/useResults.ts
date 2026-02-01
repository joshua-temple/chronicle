import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchResults, fetchResult, deleteResult } from '@/api/results'

export function useResults() {
  return useQuery({
    queryKey: ['results'],
    queryFn: fetchResults,
  })
}

export function useResult(id: string) {
  return useQuery({
    queryKey: ['results', id],
    queryFn: () => fetchResult(id),
    enabled: !!id,
  })
}

export function useDeleteResult() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteResult,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['results'] })
    },
  })
}
