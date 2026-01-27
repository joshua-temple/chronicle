import { create } from 'zustand'

interface ScenarioSelection {
  projectId: string
  suiteName?: string
  scenarioName: string
}

interface RunSelectionState {
  // Selected scenarios for batch run
  selections: ScenarioSelection[]

  // Selection mode
  selectionMode: boolean

  // Actions
  toggleSelectionMode: () => void
  enableSelectionMode: () => void
  disableSelectionMode: () => void

  // Selection management
  addSelection: (selection: ScenarioSelection) => void
  removeSelection: (selection: ScenarioSelection) => void
  toggleSelection: (selection: ScenarioSelection) => void
  clearSelections: () => void

  // Bulk operations
  selectAll: (scenarios: ScenarioSelection[]) => void
  selectAllInSuite: (projectId: string, suiteName: string, scenarios: string[]) => void
  deselectAllInSuite: (projectId: string, suiteName: string) => void
  selectAllInProject: (projectId: string, scenarios: ScenarioSelection[]) => void
  deselectAllInProject: (projectId: string) => void

  // Queries
  isSelected: (selection: ScenarioSelection) => boolean
  getSelectionCount: () => number
  getSelectionsByProject: () => Map<string, ScenarioSelection[]>
  getSelectionsBySuite: (projectId: string) => Map<string, ScenarioSelection[]>
}

function selectionKey(s: ScenarioSelection): string {
  return `${s.projectId}:${s.suiteName || ''}:${s.scenarioName}`
}

function selectionsEqual(a: ScenarioSelection, b: ScenarioSelection): boolean {
  return selectionKey(a) === selectionKey(b)
}

export const useRunSelectionStore = create<RunSelectionState>((set, get) => ({
  selections: [],
  selectionMode: false,

  toggleSelectionMode: () => {
    set(state => ({
      selectionMode: !state.selectionMode,
      selections: state.selectionMode ? [] : state.selections,
    }))
  },

  enableSelectionMode: () => set({ selectionMode: true }),

  disableSelectionMode: () => set({ selectionMode: false, selections: [] }),

  addSelection: (selection) => {
    set(state => {
      if (state.selections.some(s => selectionsEqual(s, selection))) {
        return state
      }
      return { selections: [...state.selections, selection] }
    })
  },

  removeSelection: (selection) => {
    set(state => ({
      selections: state.selections.filter(s => !selectionsEqual(s, selection)),
    }))
  },

  toggleSelection: (selection) => {
    const { isSelected, addSelection, removeSelection } = get()
    if (isSelected(selection)) {
      removeSelection(selection)
    } else {
      addSelection(selection)
    }
  },

  clearSelections: () => set({ selections: [] }),

  selectAll: (scenarios) => set({ selections: scenarios }),

  selectAllInSuite: (projectId, suiteName, scenarios) => {
    set(state => {
      const newSelections = scenarios.map(name => ({
        projectId,
        suiteName,
        scenarioName: name,
      }))
      const existing = state.selections.filter(
        s => !(s.projectId === projectId && s.suiteName === suiteName)
      )
      return { selections: [...existing, ...newSelections] }
    })
  },

  deselectAllInSuite: (projectId, suiteName) => {
    set(state => ({
      selections: state.selections.filter(
        s => !(s.projectId === projectId && s.suiteName === suiteName)
      ),
    }))
  },

  selectAllInProject: (projectId, scenarios) => {
    set(state => {
      const existing = state.selections.filter(s => s.projectId !== projectId)
      return { selections: [...existing, ...scenarios] }
    })
  },

  deselectAllInProject: (projectId) => {
    set(state => ({
      selections: state.selections.filter(s => s.projectId !== projectId),
    }))
  },

  isSelected: (selection) => {
    return get().selections.some(s => selectionsEqual(s, selection))
  },

  getSelectionCount: () => get().selections.length,

  getSelectionsByProject: () => {
    const { selections } = get()
    const byProject = new Map<string, ScenarioSelection[]>()

    for (const s of selections) {
      const existing = byProject.get(s.projectId) || []
      byProject.set(s.projectId, [...existing, s])
    }

    return byProject
  },

  getSelectionsBySuite: (projectId) => {
    const { selections } = get()
    const bySuite = new Map<string, ScenarioSelection[]>()

    for (const s of selections) {
      if (s.projectId !== projectId) continue
      const key = s.suiteName || '__default__'
      const existing = bySuite.get(key) || []
      bySuite.set(key, [...existing, s])
    }

    return bySuite
  },
}))

// Convenience hooks
export function useSelectionMode() {
  return useRunSelectionStore(state => state.selectionMode)
}

export function useSelections() {
  return useRunSelectionStore(state => state.selections)
}

export function useSelectionCount() {
  return useRunSelectionStore(state => state.selections.length)
}

export function useIsScenarioSelected(
  projectId: string,
  scenarioName: string,
  suiteName?: string
) {
  return useRunSelectionStore(state =>
    state.selections.some(
      s =>
        s.projectId === projectId &&
        s.scenarioName === scenarioName &&
        s.suiteName === suiteName
    )
  )
}
