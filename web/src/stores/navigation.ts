import { create } from 'zustand'

export type NavigationLevel = 'global' | 'project' | 'suite'

export type GlobalView = 'dashboard' | 'settings'
export type ProjectView = 'overview' | 'settings'
export type SuiteView = 'scenarios' | 'plugins' | 'runs' | 'results' | 'config'

interface NavigationState {
  // Current level
  level: NavigationLevel

  // Expanded state for accordion
  expandedProjects: Set<string>
  expandedSuites: Set<string>

  // Current views
  globalView: GlobalView
  projectView: ProjectView
  suiteView: SuiteView

  // Actions
  setLevel: (level: NavigationLevel) => void

  // Accordion controls
  toggleProject: (projectId: string) => void
  toggleSuite: (suiteId: string) => void
  expandProject: (projectId: string) => void
  collapseProject: (projectId: string) => void
  expandSuite: (suiteId: string) => void
  collapseSuite: (suiteId: string) => void
  collapseAll: () => void

  // View controls
  setGlobalView: (view: GlobalView) => void
  setProjectView: (view: ProjectView) => void
  setSuiteView: (view: SuiteView) => void

  // Navigation helpers
  navigateToDashboard: () => void
  navigateToProject: (projectId: string) => void
  navigateToSuite: (projectId: string, suiteId: string) => void
  navigateToSettings: (level: NavigationLevel, id?: string) => void
}

export const useNavigationStore = create<NavigationState>((set, get) => ({
  // Initial state
  level: 'global',
  expandedProjects: new Set(),
  expandedSuites: new Set(),
  globalView: 'dashboard',
  projectView: 'overview',
  suiteView: 'scenarios',

  // Set navigation level
  setLevel: (level) => set({ level }),

  // Toggle project expansion
  toggleProject: (projectId) => {
    set(state => {
      const expanded = new Set(state.expandedProjects)
      if (expanded.has(projectId)) {
        expanded.delete(projectId)
      } else {
        expanded.add(projectId)
      }
      return { expandedProjects: expanded }
    })
  },

  // Toggle suite expansion
  toggleSuite: (suiteId) => {
    set(state => {
      const expanded = new Set(state.expandedSuites)
      if (expanded.has(suiteId)) {
        expanded.delete(suiteId)
      } else {
        expanded.add(suiteId)
      }
      return { expandedSuites: expanded }
    })
  },

  // Expand project
  expandProject: (projectId) => {
    set(state => ({
      expandedProjects: new Set(state.expandedProjects).add(projectId),
    }))
  },

  // Collapse project
  collapseProject: (projectId) => {
    set(state => {
      const expanded = new Set(state.expandedProjects)
      expanded.delete(projectId)
      return { expandedProjects: expanded }
    })
  },

  // Expand suite
  expandSuite: (suiteId) => {
    set(state => ({
      expandedSuites: new Set(state.expandedSuites).add(suiteId),
    }))
  },

  // Collapse suite
  collapseSuite: (suiteId) => {
    set(state => {
      const expanded = new Set(state.expandedSuites)
      expanded.delete(suiteId)
      return { expandedSuites: expanded }
    })
  },

  // Collapse all
  collapseAll: () => {
    set({
      expandedProjects: new Set(),
      expandedSuites: new Set(),
    })
  },

  // Set views
  setGlobalView: (view) => set({ globalView: view, level: 'global' }),
  setProjectView: (view) => set({ projectView: view }),
  setSuiteView: (view) => set({ suiteView: view }),

  // Navigation helpers
  navigateToDashboard: () => {
    set({
      level: 'global',
      globalView: 'dashboard',
    })
  },

  navigateToProject: (projectId) => {
    const { expandProject } = get()
    expandProject(projectId)
    set({
      level: 'project',
      projectView: 'overview',
    })
  },

  navigateToSuite: (projectId, suiteId) => {
    const { expandProject, expandSuite } = get()
    expandProject(projectId)
    expandSuite(suiteId)
    set({
      level: 'suite',
      suiteView: 'scenarios',
    })
  },

  navigateToSettings: (level, _id) => {
    if (level === 'global') {
      set({ level: 'global', globalView: 'settings' })
    } else if (level === 'project') {
      set({ level: 'project', projectView: 'settings' })
    } else if (level === 'suite') {
      set({ level: 'suite', suiteView: 'config' })
    }
  },
}))

// Convenience hooks
export function useNavigationLevel() {
  return useNavigationStore(state => state.level)
}

export function useExpandedProjects() {
  return useNavigationStore(state => state.expandedProjects)
}

export function useExpandedSuites() {
  return useNavigationStore(state => state.expandedSuites)
}

export function useIsProjectExpanded(projectId: string) {
  return useNavigationStore(state => state.expandedProjects.has(projectId))
}

export function useIsSuiteExpanded(suiteId: string) {
  return useNavigationStore(state => state.expandedSuites.has(suiteId))
}
