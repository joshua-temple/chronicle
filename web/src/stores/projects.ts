import { create } from 'zustand'
import { subscribeWithSelector } from 'zustand/middleware'
import {
  getStoredProjects,
  addStoredProject,
  updateStoredProject,
  removeStoredProject,
  connectToProject,
  discoverProjects,
} from '@/api/projects'
import type { Project, ProjectHealth } from '@/api/types'

interface ProjectsState {
  // Data
  projects: Project[]
  discovered: Project[]
  health: Map<string, ProjectHealth>

  // Selection
  activeProjectId: string | null
  activeSuiteId: string | null

  // Loading states
  loading: boolean
  discovering: boolean
  connecting: Map<string, boolean>

  // Error state
  error: string | null

  // Actions
  loadProjects: () => void
  addProject: (project: Omit<Project, 'id' | 'addedAt' | 'status'>) => Project
  removeProject: (id: string) => void
  updateProject: (id: string, updates: Partial<Project>) => void

  // Connection
  connectProject: (id: string) => Promise<void>
  disconnectProject: (id: string) => void
  connectAll: () => Promise<void>

  // Discovery
  runDiscovery: () => Promise<void>
  addDiscovered: (project: Project) => void
  dismissDiscovered: (id: string) => void

  // Selection
  setActiveProject: (id: string | null) => void
  setActiveSuite: (id: string | null) => void

  // Health
  refreshHealth: (id: string) => Promise<void>
  refreshAllHealth: () => Promise<void>

  // Utility
  getProject: (id: string) => Project | undefined
  getActiveProject: () => Project | undefined
  getConnectedProjects: () => Project[]
  clearError: () => void
}

export const useProjectsStore = create<ProjectsState>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    projects: [],
    discovered: [],
    health: new Map(),
    activeProjectId: null,
    activeSuiteId: null,
    loading: false,
    discovering: false,
    connecting: new Map(),
    error: null,

    // Load projects from localStorage
    loadProjects: () => {
      const projects = getStoredProjects()
      set({ projects })
    },

    // Add a new project
    addProject: (projectData) => {
      const project = addStoredProject(projectData)
      set(state => ({ projects: [...state.projects, project] }))
      return project
    },

    // Remove a project
    removeProject: (id) => {
      removeStoredProject(id)
      set(state => ({
        projects: state.projects.filter(p => p.id !== id),
        activeProjectId: state.activeProjectId === id ? null : state.activeProjectId,
        health: new Map([...state.health].filter(([k]) => k !== id)),
      }))
    },

    // Update a project
    updateProject: (id, updates) => {
      const updated = updateStoredProject(id, updates)
      if (updated) {
        set(state => ({
          projects: state.projects.map(p => p.id === id ? updated : p),
        }))
      }
    },

    // Connect to a project's daemon
    connectProject: async (id) => {
      const project = get().projects.find(p => p.id === id)
      if (!project) return

      set(state => ({
        connecting: new Map(state.connecting).set(id, true),
      }))

      try {
        const health = await connectToProject(project)

        set(state => {
          const newHealth = new Map(state.health)
          newHealth.set(id, health)

          const newConnecting = new Map(state.connecting)
          newConnecting.delete(id)

          return {
            health: newHealth,
            connecting: newConnecting,
            projects: state.projects.map(p =>
              p.id === id
                ? { ...p, status: health.status, lastConnected: new Date().toISOString() }
                : p
            ),
          }
        })

        // Persist updated status
        updateStoredProject(id, {
          status: health.status,
          lastConnected: new Date().toISOString(),
        })
      } catch (error) {
        set(state => {
          const newConnecting = new Map(state.connecting)
          newConnecting.delete(id)

          return {
            connecting: newConnecting,
            error: `Failed to connect to ${project.name}`,
          }
        })
      }
    },

    // Disconnect from a project
    disconnectProject: (id) => {
      set(state => {
        const newHealth = new Map(state.health)
        newHealth.delete(id)

        return {
          health: newHealth,
          projects: state.projects.map(p =>
            p.id === id ? { ...p, status: 'disconnected' as const } : p
          ),
        }
      })

      updateStoredProject(id, { status: 'disconnected' })
    },

    // Connect to all projects
    connectAll: async () => {
      const { projects, connectProject } = get()
      await Promise.allSettled(projects.map(p => connectProject(p.id)))
    },

    // Run auto-discovery
    runDiscovery: async () => {
      set({ discovering: true })

      try {
        const discovered = await discoverProjects()
        set({ discovered, discovering: false })
      } catch {
        set({ discovering: false, error: 'Discovery failed' })
      }
    },

    // Add a discovered project to managed list
    addDiscovered: (project) => {
      const added = addStoredProject({
        name: project.name,
        daemonUrl: project.daemonUrl,
        description: project.description,
        autoDiscovered: true,
      })

      set(state => ({
        projects: [...state.projects, added],
        discovered: state.discovered.filter(p => p.id !== project.id),
      }))
    },

    // Dismiss a discovered project
    dismissDiscovered: (id) => {
      set(state => ({
        discovered: state.discovered.filter(p => p.id !== id),
      }))
    },

    // Set active project
    setActiveProject: (id) => {
      set({ activeProjectId: id, activeSuiteId: null })
    },

    // Set active suite
    setActiveSuite: (id) => {
      set({ activeSuiteId: id })
    },

    // Refresh health for a project
    refreshHealth: async (id) => {
      const project = get().projects.find(p => p.id === id)
      if (!project) return

      const health = await connectToProject(project)

      set(state => {
        const newHealth = new Map(state.health)
        newHealth.set(id, health)
        return { health: newHealth }
      })
    },

    // Refresh health for all projects
    refreshAllHealth: async () => {
      const { projects, refreshHealth } = get()
      await Promise.allSettled(projects.map(p => refreshHealth(p.id)))
    },

    // Get a specific project
    getProject: (id) => {
      return get().projects.find(p => p.id === id)
    },

    // Get the active project
    getActiveProject: () => {
      const { projects, activeProjectId } = get()
      return projects.find(p => p.id === activeProjectId)
    },

    // Get all connected projects
    getConnectedProjects: () => {
      return get().projects.filter(p => p.status === 'connected')
    },

    // Clear error
    clearError: () => {
      set({ error: null })
    },
  }))
)

// Convenience hooks
export function useProjects() {
  return useProjectsStore(state => state.projects)
}

export function useActiveProject() {
  return useProjectsStore(state => {
    const { projects, activeProjectId } = state
    return projects.find(p => p.id === activeProjectId)
  })
}

export function useProjectHealth(projectId: string) {
  return useProjectsStore(state => state.health.get(projectId))
}

export function useConnectedProjects() {
  return useProjectsStore(state =>
    state.projects.filter(p => p.status === 'connected')
  )
}
