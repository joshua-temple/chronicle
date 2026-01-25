export interface Scenario {
  name: string
  description?: string
  tags?: string[]
  timeout?: string
  flow_count: number
}

export interface ScenarioDetail extends Scenario {
  flow: FlowStep[]
  flags?: Record<string, unknown>
  options?: Record<string, unknown>
}

export interface FlowStep {
  name: string
  type: string
  component?: string
  parallel?: string[]
}

export interface Run {
  id: string
  status: 'running' | 'completed' | 'failed' | 'canceled'
  scenario_id: string
  start_time: string
  end_time?: string
  duration?: string
  error?: string
  result_id?: string
}

export interface Component {
  name: string
  type: 'setup' | 'task' | 'validation' | 'teardown'
  description?: string
  tags?: string[]
  produces?: string[]
  requires?: string[]
  source_file: string
  scenarios?: string[]
}

export interface RunResult {
  id: string
  project_name: string
  started_at: string
  completed_at: string
  duration: string
  total_scenarios: number
  passed: number
  failed: number
  skipped: number
  scenarios: ScenarioResult[]
}

export interface ScenarioResult {
  scenario_name: string
  state: string
  duration: string
  error?: string
  flow_results: FlowResult[]
}

export interface FlowResult {
  name: string
  type: string
  state: string
  duration: string
  error?: string
}

export interface HealthStatus {
  status: string
  timestamp: string
  version: string
}
