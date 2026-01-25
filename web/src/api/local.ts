import { ApiError } from './client'

// Types for local API

export interface ProjectInfo {
  directory: string
  config_file: string
  config_exists: boolean
  last_modified?: string
}

export interface ChronicleConfig {
  name?: string
  version: string
  scenarios?: ScenarioConfig[]
  infrastructure?: Record<string, unknown>
  chaos_profiles?: Record<string, ChaosProfile>
  mock_profiles?: Record<string, MockProfile>
  flags?: Record<string, unknown>
  execution?: Record<string, unknown>
  results?: Record<string, unknown>
}

export interface ScenarioConfig {
  name: string
  description?: string
  tags?: string[]
  timeout?: number // Duration in nanoseconds
  flow?: FlowItemConfig[]
  teardown?: FlowItemConfig[]
  abstract?: boolean
  extends?: string
  chaos_profiles?: string[]
  mock_profiles?: string[]
}

export interface FlowItemConfig {
  setup?: string
  task?: string
  validation?: string
  step?: string
  rollup?: string
  teardown?: string
  timeout?: number
  depends_on?: string[]
  params?: Record<string, unknown>
  parallel?: boolean
}

export interface InfrastructureConfig {
  providers?: ProviderConfig[]
}

export interface ProviderConfig {
  name: string
  type: string
  config?: Record<string, unknown>
}

export interface ChaosProfile {
  name: string
  infrastructure?: ChaosInfraConfig[]
  application?: ChaosAppConfig[]
}

export interface ChaosInfraConfig {
  target: string
  action: string
  duration?: string
}

export interface ChaosAppConfig {
  type: string
  config?: Record<string, unknown>
}

export interface MockProfile {
  name: string
  injectors?: MockInjector[]
}

export interface MockInjector {
  target: string
  responses?: MockResponse[]
}

export interface MockResponse {
  match?: Record<string, unknown>
  return?: unknown
}

export interface ValidationResult {
  valid: boolean
  errors: string[]
  warnings: string[]
}

export interface LocalDiscoveredComponent {
  name: string
  type: 'setup' | 'task' | 'validation' | 'teardown'
  description: string
  tags: string[]
  produces: string[]
  requires: string[]
  source_file: string
}

export interface DiscoveryResult {
  components: LocalDiscoveredComponent[]
  discovered_at: string
}

// API base path for local endpoints
const LOCAL_BASE = '/api/local'

// Local API request helper (uses different base path than main API)
async function localApiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${LOCAL_BASE}${endpoint}`
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new ApiError(response.status, error.error || 'Request failed')
  }

  return response.json()
}

// Project info
export async function fetchProject(): Promise<ProjectInfo> {
  return localApiRequest<ProjectInfo>('/project')
}

// Config operations
export async function fetchConfig(): Promise<ChronicleConfig> {
  return localApiRequest<ChronicleConfig>('/config')
}

export async function saveConfig(config: ChronicleConfig): Promise<void> {
  await localApiRequest<{ status: string }>('/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

export async function validateConfig(config: ChronicleConfig): Promise<ValidationResult> {
  return localApiRequest<ValidationResult>('/config/validate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

// Discovery operations
export async function runDiscovery(): Promise<DiscoveryResult> {
  return localApiRequest<DiscoveryResult>('/discover', {
    method: 'POST',
  })
}

export async function fetchLocalComponents(): Promise<DiscoveryResult> {
  return localApiRequest<DiscoveryResult>('/components')
}
