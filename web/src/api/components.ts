import { apiRequest } from './client'
import type { Component } from './types'

export async function fetchComponents(): Promise<{ components: Component[]; count: number }> {
  return apiRequest('/components')
}

export async function fetchComponent(name: string): Promise<Component> {
  return apiRequest(`/components/${encodeURIComponent(name)}`)
}
