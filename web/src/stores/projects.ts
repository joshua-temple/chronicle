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

interface ProjectsState {
  projects: Project[]
  discovered: Project[]
  loading: boolean
  error: string | null
  activeProjectId: string | null

  fetchProjects: () => Promise<void>
  addProject: (project: Partial<Project>) => Promise<void>
  removeProject: (id: string) => Promise<void>
  updateProject: (id: string, updates: Partial<Project>) => Promise<void>
  launchProject: (id: string) => Promise<void>
  stopProject: (id: string) => Promise<void>
  setActiveProject: (id: string | null) => void
  discover: () => Promise<void>
  clearError: () => void
}

const API_BASE = '/api/standalone'

// Operation flags to prevent concurrent requests
let isFetching = false
let isDiscovering = false

// Export for testing purposes
export const _resetOperationFlags = () => {
  isFetching = false
  isDiscovering = false
}

export const useProjectsStore = create<ProjectsState>((set, get) => ({
  // Initial state
  projects: [],
  discovered: [],
  loading: false,
  error: null,
  activeProjectId: null,

  // Fetch all projects
  fetchProjects: async () => {
    if (isFetching) return
    isFetching = true
    set({ loading: true, error: null })
    try {
      const response = await fetch(`${API_BASE}/projects`)
      if (!response.ok) {
        throw new Error(`Failed to fetch projects: ${response.statusText}`)
      }
      const data = await response.json().catch(() => {
        throw new Error('Failed to parse server response')
      })
      set({ projects: data.projects || [], loading: false })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to fetch projects'
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
        throw new Error(`Failed to add project: ${response.statusText}`)
      }
      // Refresh projects list after adding - fetchProjects will set loading: false
      await get().fetchProjects()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to add project'
      set({ error: message, loading: false })
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
        throw new Error(`Failed to remove project: ${response.statusText}`)
      }
      // Refresh projects list after removing
      await get().fetchProjects()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to remove project'
      set({ error: message, loading: false })
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
        throw new Error(`Failed to update project: ${response.statusText}`)
      }
      // Refresh projects list after updating
      await get().fetchProjects()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update project'
      set({ error: message, loading: false })
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
        throw new Error(`Failed to launch project: ${response.statusText}`)
      }
      // Refresh projects list to get updated status
      await get().fetchProjects()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to launch project'
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
        throw new Error(`Failed to stop project: ${response.statusText}`)
      }
      // Refresh projects list to get updated status
      await get().fetchProjects()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to stop project'
      set({ error: message, loading: false })
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
        throw new Error(`Failed to discover projects: ${response.statusText}`)
      }
      const data = await response.json().catch(() => {
        throw new Error('Failed to parse server response')
      })
      set({ discovered: data.projects || [], loading: false })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to discover projects'
      set({ error: message, loading: false })
    } finally {
      isDiscovering = false
    }
  },

  // Clear any error state
  clearError: () => {
    set({ error: null })
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
