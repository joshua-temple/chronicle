// =============================================================================
// Core Hierarchy Types
// =============================================================================

/**
 * Project represents a Chronicle daemon connection.
 * Projects are the top-level organizational unit.
 */
export interface Project {
  id: string
  name: string
  description?: string

  // Connection info
  daemonUrl: string
  status: ProjectConnectionStatus

  // Metadata
  addedAt: string
  lastConnected?: string
  autoDiscovered?: boolean

  // Cached data (populated when connected)
  suiteCount?: number
  scenarioCount?: number
  pluginCount?: number
}

export type ProjectConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'error'

export interface ProjectHealth {
  projectId: string
  status: ProjectConnectionStatus
  version?: string
  uptime?: string
  activeRuns: number
  error?: string
}

/**
 * Suite is a named collection of scenarios within a project.
 * This is the second level of the hierarchy.
 */
export interface Suite {
  name: string
  description?: string
  projectId: string

  // Scenario references
  scenarios: string[]
  resolvedScenarios?: string[]

  // Filtering
  tags?: string[]
  excludeTags?: string[]

  // Execution config
  parallel?: number
  failFast?: boolean

  // Stats (populated from daemon)
  scenarioCount?: number
  lastRun?: RunSummary
}

/**
 * Scenario is an individual test definition.
 * This is the third level of the hierarchy.
 */
export interface Scenario {
  name: string
  description?: string
  projectId: string
  suiteName?: string

  // Definition
  tags?: string[]
  timeout?: string
  flowCount: number

  // Detailed flow (when fetched individually)
  flow?: FlowStep[]
  teardown?: FlowStep[]

  // Execution options
  flags?: Record<string, unknown>
  options?: Record<string, unknown>

  // Stats
  lastRun?: RunSummary
}

export interface FlowStep {
  name: string
  type: 'setup' | 'task' | 'validation' | 'teardown' | 'step' | 'rollup'
  plugin?: string
  parallel?: string[]
  timeout?: number
  dependsOn?: string[]
  params?: Record<string, unknown>
}

/**
 * Plugin represents a language-agnostic extension.
 * Renamed from "Component" to emphasize language independence.
 */
export interface Plugin {
  name: string
  type: 'setup' | 'task' | 'validation' | 'teardown' | 'step' | 'rollup'
  projectId: string

  description?: string
  tags?: string[]

  // Data flow
  produces?: string[]
  requires?: string[]

  // Source info (language-agnostic)
  sourceFile?: string
  language?: string

  // Usage
  usedInScenarios?: string[]
}

/**
 * Backward compatibility alias.
 * @deprecated Use Plugin instead
 */
export type Component = Plugin

/**
 * ScenarioDetail extends Scenario with fully resolved flow information.
 * Used when fetching individual scenario details.
 */
export interface ScenarioDetail extends Scenario {
  // Detailed flow information (fetched individually)
  flow: FlowStep[]
  teardown?: FlowStep[]
}

// =============================================================================
// Run Types
// =============================================================================

export interface Run {
  id: string
  projectId: string
  status: RunStatus

  // What was run
  runType: 'single' | 'suite' | 'batch' | 'cross-suite'
  scenarioId?: string
  suiteId?: string
  scenarioIds?: string[]

  // Timing
  startTime: string
  endTime?: string
  duration?: string

  // Results
  error?: string
  resultId?: string
  progress?: RunProgress
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'canceled'

export interface RunProgress {
  total: number
  completed: number
  failed: number
  currentScenario?: string
  currentStep?: string
}

export interface RunSummary {
  runId: string
  status: RunStatus
  startTime: string
  duration?: string
  passed?: number
  failed?: number
}

export interface RunResult {
  id: string
  projectId: string
  projectName: string

  // Timing
  startedAt: string
  completedAt: string
  duration: string

  // Stats
  totalScenarios: number
  passed: number
  failed: number
  skipped: number

  // Details
  scenarios: ScenarioResult[]
}

export interface ScenarioResult {
  scenarioId: string
  scenarioName: string
  state: 'passed' | 'failed' | 'skipped'
  startTime: string
  endTime: string
  duration: string
  error?: string
  skipReason?: string
  flowResults: FlowItemResult[]
}

export interface FlowItemResult {
  name: string
  type: string
  state: 'passed' | 'failed' | 'skipped'
  startTime: string
  endTime: string
  duration: string
  error?: string
  output?: unknown
}

// =============================================================================
// Run Request Types
// =============================================================================

export interface RunSingleRequest {
  projectId: string
  scenarioName: string
}

export interface RunSuiteRequest {
  projectId: string
  suiteName: string
}

export interface RunBatchRequest {
  projectId: string
  scenarioNames: string[]
}

/**
 * BatchRunRequest for daemon batch execution.
 * Supports filtering by scenarios, suite, tags, and execution options.
 */
export interface BatchRunRequest {
  projectId: string
  scenarios?: string[]
  suite?: string
  tags?: string[]
  excludeTags?: string[]
  flags?: Record<string, unknown>
  parallel?: number
  failFast?: boolean
  timeout?: string
}

export interface RunCrossSuiteRequest {
  // Can span multiple projects
  selections: Array<{
    projectId: string
    scenarioNames: string[]
  }>
}

// =============================================================================
// Settings Types
// =============================================================================

/**
 * UI Settings - stored in localStorage
 */
export interface UISettings {
  theme: 'light' | 'dark' | 'system'
  sidebarCollapsed: boolean
  defaultView: 'dashboard' | 'projects'
  refreshInterval: number
  notifications: {
    runCompleted: boolean
    runFailed: boolean
    connectionLost: boolean
  }
}

/**
 * Project Settings - stored on daemon
 */
export interface ProjectSettings {
  projectId: string
  defaultParallelism: number
  defaultTimeout: string
  failFast: boolean
  autoReload: boolean
  watchPaths?: string[]
}

/**
 * Suite Settings - stored on daemon
 */
export interface SuiteSettings {
  projectId: string
  suiteName: string
  parallel?: number
  failFast?: boolean
  timeout?: string
  tags?: string[]
  excludeTags?: string[]
}

// =============================================================================
// API Response Types
// =============================================================================

export interface ProjectsResponse {
  projects: Project[]
  discovered: Project[]
}

export interface SuitesResponse {
  projectId: string
  suites: Suite[]
  count: number
}

export interface ScenariosResponse {
  projectId: string
  suiteName?: string
  scenarios: Scenario[]
  count: number
}

export interface PluginsResponse {
  projectId: string
  plugins: Plugin[]
  count: number
}

export interface RunsResponse {
  projectId?: string
  runs: Run[]
  count: number
}

export interface ResultsResponse {
  projectId?: string
  results: string[]
  count: number
}

export interface HealthResponse {
  projects: ProjectHealth[]
  aggregateStatus: 'healthy' | 'degraded' | 'unhealthy'
}
