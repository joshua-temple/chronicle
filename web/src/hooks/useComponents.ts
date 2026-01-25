import { useQuery } from '@tanstack/react-query'
import { fetchComponents, fetchComponent } from '@/api/components'

export function useComponents() {
  return useQuery({
    queryKey: ['components'],
    queryFn: fetchComponents,
  })
}

export function useComponent(name: string) {
  return useQuery({
    queryKey: ['components', name],
    queryFn: () => fetchComponent(name),
    enabled: !!name,
  })
}
