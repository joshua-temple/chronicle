import { apiRequest } from './client'
import type { RunResult } from './types'

export async function fetchResults(): Promise<{ results: string[]; count: number }> {
  return apiRequest('/results')
}

export async function fetchResult(id: string): Promise<RunResult> {
  return apiRequest(`/results/${id}`)
}

export async function deleteResult(id: string): Promise<{ status: string }> {
  return apiRequest(`/results/${id}`, { method: 'DELETE' })
}
