import { apiRequest } from './client'
import type {
  Scenario,
  ScenariosResponse,
  RunSingleRequest,
  RunBatchRequest,
  Run,
} from './types'

/**
 * Scenario API - operates against a specific project's daemon
 */

export async function fetchScenarios(
  daemonUrl: string,
  suiteName?: string
): Promise<ScenariosResponse> {
  const url = suiteName
    ? `${daemonUrl}/api/v1/suites/${encodeURIComponent(suiteName)}/scenarios`
    : `${daemonUrl}/api/v1/scenarios`

  const response = await apiRequest<{ scenarios: Scenario[]; count: number }>(url)
  return {
    projectId: '', // Caller will set this
    suiteName,
    scenarios: response.scenarios || [],
    count: response.count || 0,
  }
}

export async function fetchScenario(
  daemonUrl: string,
  scenarioName: string
): Promise<Scenario> {
  return apiRequest<Scenario>(
    `${daemonUrl}/api/v1/scenarios/${encodeURIComponent(scenarioName)}`
  )
}

export async function runScenario(
  request: RunSingleRequest & { daemonUrl: string }
): Promise<Run> {
  return apiRequest<Run>(`${request.daemonUrl}/api/v1/runs`, {
    method: 'POST',
    body: JSON.stringify({
      scenario: request.scenarioName,
    }),
  })
}

export async function runBatch(
  request: RunBatchRequest & { daemonUrl: string }
): Promise<Run> {
  return apiRequest<Run>(`${request.daemonUrl}/api/v1/runs/batch`, {
    method: 'POST',
    body: JSON.stringify({
      scenarios: request.scenarioNames,
    }),
  })
}
