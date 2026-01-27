import { useQuery } from '@tanstack/react-query'
import { fetchPlugins, fetchPlugin } from '@/api/plugins'
import { fetchLocalComponents } from '@/api/local'
import { useMode } from '@/stores/mode'
import { useActiveProject } from '@/stores/projects'
import type { Plugin } from '@/api/types'

export function useComponents() {
  const mode = useMode()
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['components', mode, activeProject?.id],
    queryFn: async (): Promise<{ components: Plugin[]; count: number }> => {
      if (mode === 'standalone') {
        // In standalone mode, use local discovery endpoint
        const result = await fetchLocalComponents()
        const components: Plugin[] = result.components.map(c => ({
          name: c.name,
          type: c.type as Plugin['type'],
          projectId: activeProject?.id || '',
          description: c.description || undefined,
          tags: c.tags?.length ? c.tags : undefined,
          produces: c.produces?.length ? c.produces : undefined,
          requires: c.requires?.length ? c.requires : undefined,
          sourceFile: c.source_file,
        }))
        return { components, count: components.length }
      }

      // Daemon mode - use active project's daemon URL
      if (!activeProject?.daemonUrl) {
        return { components: [], count: 0 }
      }

      const response = await fetchPlugins(activeProject.daemonUrl)
      return {
        components: response.plugins.map(p => ({
          ...p,
          projectId: activeProject.id,
        })),
        count: response.count,
      }
    },
  })
}

export function useComponent(name: string) {
  const mode = useMode()
  const activeProject = useActiveProject()

  return useQuery({
    queryKey: ['components', name, mode, activeProject?.id],
    queryFn: async () => {
      if (mode === 'standalone') {
        const result = await fetchLocalComponents()
        const component = result.components.find(c => c.name === name)
        if (!component) throw new Error(`Component ${name} not found`)
        return {
          name: component.name,
          type: component.type as Plugin['type'],
          projectId: activeProject?.id || '',
          description: component.description || undefined,
          tags: component.tags?.length ? component.tags : undefined,
          produces: component.produces?.length ? component.produces : undefined,
          requires: component.requires?.length ? component.requires : undefined,
          sourceFile: component.source_file,
        }
      }

      // Daemon mode
      if (!activeProject?.daemonUrl) {
        throw new Error('No active project')
      }

      const plugin = await fetchPlugin(activeProject.daemonUrl, name)
      return {
        ...plugin,
        projectId: activeProject.id,
      }
    },
    enabled: !!name,
  })
}
