import { apiRequest } from './client'
import type { Plugin, PluginsResponse } from './types'

/**
 * Plugin API - operates against a specific project's daemon
 * Renamed from "Components" to emphasize language-agnostic nature
 */

export async function fetchPlugins(daemonUrl: string): Promise<PluginsResponse> {
  // Backend still uses /components endpoint for now
  const response = await apiRequest<{ components: Plugin[]; count: number }>(
    `${daemonUrl}/api/v1/components`
  )
  return {
    projectId: '', // Caller will set this
    plugins: (response.components || []).map(c => ({
      ...c,
      projectId: '',
    })),
    count: response.count || 0,
  }
}

export async function fetchPlugin(
  daemonUrl: string,
  pluginName: string
): Promise<Plugin> {
  // Backend still uses /components endpoint for now
  const component = await apiRequest<Plugin>(
    `${daemonUrl}/api/v1/components/${encodeURIComponent(pluginName)}`
  )
  return {
    ...component,
    projectId: '',
  }
}
