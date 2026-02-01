import { apiRequest } from './client'
import type {
  Run,
  RunsResponse,
  RunCrossSuiteRequest,
} from './types'

/**
 * Runs API - operates against specific project daemons
 */

export async function fetchRuns(daemonUrl: string): Promise<RunsResponse> {
  const response = await apiRequest<{ runs: Run[]; count: number }>(
    `${daemonUrl}/api/v1/runs`
  )
  return {
    runs: response.runs || [],
    count: response.count || 0,
  }
}

export async function fetchRun(daemonUrl: string, runId: string): Promise<Run> {
  return apiRequest<Run>(`${daemonUrl}/api/v1/runs/${encodeURIComponent(runId)}`)
}

export async function cancelRun(daemonUrl: string, runId: string): Promise<void> {
  await apiRequest(`${daemonUrl}/api/v1/runs/${encodeURIComponent(runId)}`, {
    method: 'DELETE',
  })
}

/**
 * Cross-suite batch run - spans multiple projects
 * Executes runs in parallel across different daemons
 *
 * Note: This function requires daemon URL resolution. Callers should use
 * runCrossSuiteWithUrls instead, which accepts resolved daemon URLs.
 */
export async function runCrossSuite(_request: RunCrossSuiteRequest): Promise<Run[]> {
  // This requires knowing the daemon URL for each project
  // The caller should resolve projectId to daemonUrl
  throw new Error('Cross-suite run requires daemon URL resolution - use runCrossSuiteWithUrls')
}

/**
 * Cross-suite run with resolved daemon URLs
 */
export async function runCrossSuiteWithUrls(
  selections: Array<{
    daemonUrl: string
    projectId: string
    scenarioNames: string[]
  }>
): Promise<Run[]> {
  const runPromises = selections.map(async (selection) => {
    return apiRequest<Run>(`${selection.daemonUrl}/api/v1/runs/batch`, {
      method: 'POST',
      body: JSON.stringify({
        scenarios: selection.scenarioNames,
      }),
    })
  })

  const results = await Promise.allSettled(runPromises)
  return results
    .filter((r): r is PromiseFulfilledResult<Run> => r.status === 'fulfilled')
    .map(r => r.value)
}

/**
 * Fetch runs from all connected projects
 */
export async function fetchAllRuns(
  daemonUrls: Array<{ projectId: string; daemonUrl: string }>
): Promise<RunsResponse> {
  const runPromises = daemonUrls.map(async ({ projectId, daemonUrl }) => {
    try {
      const response = await fetchRuns(daemonUrl)
      return response.runs.map(run => ({ ...run, projectId }))
    } catch {
      return []
    }
  })

  const results = await Promise.allSettled(runPromises)
  const allRuns = results
    .filter((r): r is PromiseFulfilledResult<Run[]> => r.status === 'fulfilled')
    .flatMap(r => r.value)

  return {
    runs: allRuns,
    count: allRuns.length,
  }
}
