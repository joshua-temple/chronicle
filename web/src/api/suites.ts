import { apiRequest } from './client'
import type {
  Suite,
  SuitesResponse,
  SuiteSettings,
  RunSuiteRequest,
  Run,
} from './types'

/**
 * Suite API - operates against a specific project's daemon
 */

export async function fetchSuites(daemonUrl: string): Promise<SuitesResponse> {
  const response = await apiRequest<{ suites: Suite[]; count: number }>(
    `${daemonUrl}/api/v1/suites`
  )
  return {
    projectId: '', // Caller will set this
    suites: response.suites || [],
    count: response.count || 0,
  }
}

export async function fetchSuite(
  daemonUrl: string,
  suiteName: string
): Promise<Suite> {
  return apiRequest<Suite>(`${daemonUrl}/api/v1/suites/${encodeURIComponent(suiteName)}`)
}

export async function fetchSuiteSettings(
  daemonUrl: string,
  suiteName: string
): Promise<SuiteSettings> {
  return apiRequest<SuiteSettings>(
    `${daemonUrl}/api/v1/suites/${encodeURIComponent(suiteName)}/settings`
  )
}

export async function updateSuiteSettings(
  daemonUrl: string,
  suiteName: string,
  settings: Partial<SuiteSettings>
): Promise<SuiteSettings> {
  return apiRequest<SuiteSettings>(
    `${daemonUrl}/api/v1/suites/${encodeURIComponent(suiteName)}/settings`,
    {
      method: 'PUT',
      body: JSON.stringify(settings),
    }
  )
}

export async function runSuite(request: RunSuiteRequest & { daemonUrl: string }): Promise<Run> {
  return apiRequest<Run>(`${request.daemonUrl}/api/v1/runs`, {
    method: 'POST',
    body: JSON.stringify({
      suite: request.suiteName,
    }),
  })
}
