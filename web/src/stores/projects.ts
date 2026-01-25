import { create } from 'zustand'

export type ProjectState = 'unknown' | 'stopped' | 'starting' | 'running' | 'unhealthy'

export interface ProjectStatus {
  state: ProjectState
  port?: number
  version?: string
  error?: string
}

export interface Project {
  id: string
  name: string
  path?: string
  remoteUrl?: string
  addedAt: string
  lastOpened?: string
  lastScenarios?: string[]
  preferences?: Record<string, string>
  autoDiscovered?: boolean
  status: ProjectStatus
}

// Polling interval constants
export const POLLING_INTERVAL_ACTIVE = 5000 // 5s for active view
export const POLLING_INTERVAL_BACKGROUND = 30000 // 30s for background projects
export const POLLING_INTERVAL_HIDDEN = 60000 // 60s when tab hidden

interface ProjectsState {
  projects: Project[]
  discovered: Project[]
  loading: boolean
  error: string | null
  activeProjectId: string | null

  // Polling state
  pollingIntervalId: ReturnType<typeof setInterval> | null
  pollingIntervalMs: number

  fetchProjects: () => Promise<void>
  addProject: (project: Partial<Project>) => Promise<void>
  removeProject: (id: string) => Promise<void>
  updateProject: (id: string, updates: Partial<Project>) => Promise<void>
  launchProject: (id: string) => Promise<void>
  stopProject: (id: string) => Promise<void>
  setActiveProject: (id: string | null) => void
  discover: () => Promise<void>
  clearError: () => void

  // Polling methods
  startPolling: (intervalMs?: number) => void
  stopPolling: () => void
  setPollingInterval: (intervalMs: number) => void
}

const API_BASE = '/api/standalone'

// Operation flags to prevent concurrent requests
let isFetching = false
let isDiscovering = false

// Track if polling is being set up to prevent multiple intervals
let isSettingUpPolling = false

// Export for testing purposes
export const _resetOperationFlags = () => {
  isFetching = false
  isDiscovering = false
  isSettingUpPolling = false
}

// Helper to extract error message from API response
async function extractErrorMessage(response: Response, defaultMessage: string): Promise<string> {
  try {
    const data = await response.json()
    if (data.error && typeof data.error === 'string') {
      return data.error
    }
  } catch {
    // Could not parse JSON response
  }
  // Fall back to status text or default
  if (response.statusText && response.statusText !== 'OK') {
    return `${defaultMessage}: ${response.statusText}`
  }
  return defaultMessage
}

// Helper to get user-friendly error messages for network errors
function getNetworkErrorMessage(error: unknown, context: string): string {
  if (error instanceof TypeError && error.message === 'Failed to fetch') {
    return `Cannot connect to server. Please ensure Chronicle is running.`
  }
  if (error instanceof Error) {
    // Check for common network errors
    const msg = error.message.toLowerCase()
    if (msg.includes('timeout') || msg.includes('timed out')) {
      return `${context}: request timed out. The server may be busy or unreachable.`
    }
    if (msg.includes('network') || msg.includes('connection')) {
      return `${context}: network error. Please check your connection.`
    }
    return error.message
  }
  return `${context}: an unexpected error occurred`
}

export const useProjectsStore = create<ProjectsState>((set, get) => ({
  // Initial state
  projects: [],
  discovered: [],
  loading: false,
  error: null,
  activeProjectId: null,

  // Polling state
  pollingIntervalId: null,
  pollingIntervalMs: POLLING_INTERVAL_ACTIVE,

  // Fetch all projects
  fetchProjects: async () => {
    if (isFetching) return
    isFetching = true
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects`)
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to fetch projects')
        throw new Error(errorMsg)
      }
      const data = await response.json().catch(() => {
        throw new Error('Failed to parse server response')
      })
      set({ projects: data.projects || [], loading: false })
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to fetch projects')
      set({ error: message, loading: false })
    } finally {
      isFetching = false
    }
  },

  // Add a new project
  addProject: async (project) => {
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(project),
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to add project')
        throw new Error(errorMsg)
      }
      // Refresh projects list after adding - fetchProjects will set loading: false
      await get().fetchProjects()
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to add project')
      set({ error: message, loading: false })
      throw error // Re-throw so caller can handle it
    }
  },

  // Remove a project
  removeProject: async (id) => {
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects/${id}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to remove project')
        throw new Error(errorMsg)
      }
      // Refresh projects list after removing
      await get().fetchProjects()
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to remove project')
      set({ error: message, loading: false })
      throw error // Re-throw so caller can handle it
    }
  },

  // Update a project
  updateProject: async (id, updates) => {
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to update project')
        throw new Error(errorMsg)
      }
      // Refresh projects list after updating
      await get().fetchProjects()
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to update project')
      set({ error: message, loading: false })
      throw error // Re-throw so caller can handle it
    }
  },

  // Launch daemon for a project
  launchProject: async (id) => {
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects/${id}/launch`, {
        method: 'POST',
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to launch daemon')
        throw new Error(errorMsg)
      }
      // Refresh projects list to get updated status
      await get().fetchProjects()
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to launch daemon')
      set({ error: message, loading: false })
    }
  },

  // Stop daemon for a project
  stopProject: async (id) => {
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects/${id}/stop`, {
        method: 'POST',
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to stop daemon')
        throw new Error(errorMsg)
      }
      // Refresh projects list to get updated status
      await get().fetchProjects()
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to stop daemon')
      set({ error: message, loading: false })
      throw error // Re-throw so caller can handle it
    }
  },

  // Set the active project
  setActiveProject: (id) => {
    set({ activeProjectId: id })
  },

  // Discover Chronicle projects
  discover: async () => {
    if (isDiscovering) return
    isDiscovering = true
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/discover`, {
        method: 'POST',
      })
      if (!response.ok) {
        const errorMsg = await extractErrorMessage(response, 'Failed to discover projects')
        throw new Error(errorMsg)
      }
      const data = await response.json().catch(() => {
        throw new Error('Failed to parse server response')
      })
      set({ discovered: data.projects || [], loading: false })
    } catch (error) {
      const message = getNetworkErrorMessage(error, 'Failed to discover projects')
      set({ error: message, loading: false })
    } finally {
      isDiscovering = false
    }
  },

  // Clear any error state
  clearError: () => {
    set({ error: null })
  },

  // Start periodic polling for project status updates
  startPolling: (intervalMs = POLLING_INTERVAL_ACTIVE) => {
    const state = get()

    // Prevent multiple polling intervals
    if (state.pollingIntervalId !== null || isSettingUpPolling) {
      return
    }

    isSettingUpPolling = true

    // Fetch immediately, then set up interval
    get().fetchProjects()

    const intervalId = setInterval(() => {
      get().fetchProjects()
    }, intervalMs)

    set({ pollingIntervalId: intervalId, pollingIntervalMs: intervalMs })
    isSettingUpPolling = false
  },

  // Stop periodic polling
  stopPolling: () => {
    const state = get()
    if (state.pollingIntervalId !== null) {
      clearInterval(state.pollingIntervalId)
      set({ pollingIntervalId: null })
    }
  },

  // Adjust polling interval (e.g., when tab visibility changes)
  setPollingInterval: (intervalMs: number) => {
    const state = get()

    // Only adjust if currently polling
    if (state.pollingIntervalId === null) {
      return
    }

    // Don't restart if interval hasn't changed
    if (state.pollingIntervalMs === intervalMs) {
      return
    }

    // Clear existing interval
    clearInterval(state.pollingIntervalId)

    // Set up new interval with updated timing
    const intervalId = setInterval(() => {
      get().fetchProjects()
    }, intervalMs)

    set({ pollingIntervalId: intervalId, pollingIntervalMs: intervalMs })
  },
}))

// Convenience hooks
export function useProjects() {
  return useProjectsStore((state) => state.projects)
}

export function useDiscoveredProjects() {
  return useProjectsStore((state) => state.discovered)
}

export function useActiveProject() {
  return useProjectsStore((state) => {
    if (!state.activeProjectId) return null
    return state.projects.find((p) => p.id === state.activeProjectId) ?? null
  })
}

export function useProjectsLoading() {
  return useProjectsStore((state) => state.loading)
}

export function useProjectsError() {
  return useProjectsStore((state) => state.error)
}

export function usePollingState() {
  return useProjectsStore((state) => ({
    isPolling: state.pollingIntervalId !== null,
    intervalMs: state.pollingIntervalMs,
  }))
}
