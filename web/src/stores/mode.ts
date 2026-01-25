import { create } from 'zustand'

export type AppMode = 'standalone' | 'daemon' | 'disconnected' | 'detecting'

interface ModeState {
  mode: AppMode
  setMode: (mode: AppMode) => void
  detectMode: () => Promise<void>
}

export const useModeStore = create<ModeState>((set) => ({
  mode: 'detecting',
  setMode: (mode) => set({ mode }),
  detectMode: async () => {
    // Try standalone API first (local UI server)
    try {
      const res = await fetch('/api/local/project')
      if (res.ok) {
        set({ mode: 'standalone' })
        return
      }
    } catch {
      // Not standalone mode
    }

    // Try daemon API
    try {
      const res = await fetch('/api/v1/health')
      if (res.ok) {
        set({ mode: 'daemon' })
        return
      }
    } catch {
      // Not daemon mode
    }

    set({ mode: 'disconnected' })
  },
}))

// Convenience hooks
export function useMode() {
  return useModeStore((state) => state.mode)
}

export function useDetectMode() {
  return useModeStore((state) => state.detectMode)
}

export function useIsStandalone() {
  return useModeStore((state) => state.mode === 'standalone')
}

export function useIsDaemon() {
  return useModeStore((state) => state.mode === 'daemon')
}
