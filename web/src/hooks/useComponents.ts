import { useQuery } from '@tanstack/react-query'
import { fetchComponents, fetchComponent } from '@/api/components'
import { fetchLocalComponents } from '@/api/local'
import { useMode } from '@/stores/mode'
import type { Component } from '@/api/types'

export function useComponents() {
  const mode = useMode()

  return useQuery({
    queryKey: ['components', mode],
    queryFn: async (): Promise<{ components: Component[]; count: number }> => {
      if (mode === 'standalone') {
        // In standalone mode, use local discovery endpoint
        const result = await fetchLocalComponents()
        const components: Component[] = result.components.map(c => ({
          name: c.name,
          type: c.type as Component['type'],
          description: c.description || undefined,
          tags: c.tags?.length ? c.tags : undefined,
          produces: c.produces?.length ? c.produces : undefined,
          requires: c.requires?.length ? c.requires : undefined,
          source_file: c.source_file,
        }))
        return { components, count: components.length }
      }
      return fetchComponents()
    },
  })
}

export function useComponent(name: string) {
  const mode = useMode()

  return useQuery({
    queryKey: ['components', name, mode],
    queryFn: async () => {
      if (mode === 'standalone') {
        const result = await fetchLocalComponents()
        const component = result.components.find(c => c.name === name)
        if (!component) throw new Error(`Component ${name} not found`)
        return {
          name: component.name,
          type: component.type as Component['type'],
          description: component.description || undefined,
          tags: component.tags?.length ? component.tags : undefined,
          produces: component.produces?.length ? component.produces : undefined,
          requires: component.requires?.length ? component.requires : undefined,
          source_file: component.source_file,
        }
      }
      return fetchComponent(name)
    },
    enabled: !!name,
  })
}
