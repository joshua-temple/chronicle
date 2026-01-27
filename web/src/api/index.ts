// Core client
export { apiRequest, ApiError } from './client'

// Types
export * from './types'

// Project management
export {
  getStoredProjects,
  saveStoredProjects,
  addStoredProject,
  updateStoredProject,
  removeStoredProject,
  checkDaemonHealth,
  connectToProject,
  discoverProjects,
  fetchProjectSettings,
  updateProjectSettings,
  fetchAggregateHealth,
} from './projects'

// Suites
export {
  fetchSuites,
  fetchSuite,
  fetchSuiteSettings,
  updateSuiteSettings,
  runSuite,
} from './suites'

// Scenarios
export {
  fetchScenarios,
  fetchScenario,
  runScenario,
  runBatch,
} from './scenarios'

// Plugins (formerly Components)
export {
  fetchPlugins,
  fetchPlugin,
} from './plugins'

// Runs
export {
  fetchRuns,
  fetchRun,
  cancelRun,
  runCrossSuiteWithUrls,
  fetchAllRuns,
} from './runs'

// Results
export {
  fetchResults,
  fetchResult,
  deleteResult,
} from './results'

// Events (SSE)
export {
  connectToEvents,
  parseEventData,
  type SSEConnection,
  type SSEEventType,
  type SSEEventData,
} from './events'

// UI Settings
export {
  getUISettings,
  saveUISettings,
  resetUISettings,
} from './settings'
