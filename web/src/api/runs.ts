import { apiRequest } from './client'
import type { Run } from './types'

export async function fetchRuns(): Promise<{ runs: Run[]; count: number }> {
  return apiRequest('/runs')
}

export async function fetchRun(id: string): Promise<Run> {
  return apiRequest(`/runs/${id}`)
}

export async function createRun(scenarioName: string, options?: {
  flags?: Record<string, unknown>
  timeout?: string
}): Promise<Run> {
  return apiRequest('/runs', {
    method: 'POST',
    body: JSON.stringify({
      scenario_name: scenarioName,
      ...options,
    }),
  })
}

export async function cancelRun(id: string): Promise<{ status: string }> {
  return apiRequest(`/runs/${id}`, { method: 'DELETE' })
}
