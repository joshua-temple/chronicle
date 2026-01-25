import { apiRequest } from './client'
import type { Scenario, ScenarioDetail } from './types'

export async function fetchScenarios(): Promise<{ scenarios: Scenario[]; count: number }> {
  return apiRequest('/scenarios')
}

export async function fetchScenario(name: string): Promise<ScenarioDetail> {
  return apiRequest(`/scenarios/${encodeURIComponent(name)}`)
}
