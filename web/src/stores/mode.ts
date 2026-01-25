import { create } from 'zustand'

export type AppMode = 'standalone' | 'daemon' | 'disconnected' | 'detecting'

interface ModeState {
  mode: AppMode
  setMode: (mode: AppMode) => void
  detectMode: () => Promise<void>
}

// Cache detected mode to avoid repeated fetches
let cachedMode: AppMode | null = null
// Flag to prevent concurrent detection calls
let isDetecting = false

export const useModeStore = create<ModeState>((set) => ({
  mode: cachedMode ?? 'detecting',
  setMode: (mode) => set({ mode }),
  detectMode: async () => {
    // Return cached mode if already detected
    if (cachedMode !== null) {
      set({ mode: cachedMode })
      return
    }

    // Prevent concurrent detection calls
    if (isDetecting) {
      return
    }

    isDetecting = true

    try {
      // Try standalone API first (local UI server)
      // The /api/standalone/mode endpoint returns {"mode": "standalone"} in standalone mode
      try {
        const res = await fetch('/api/standalone/mode')
        if (res.ok) {
          try {
            const data = await res.json()
            if (data?.mode === 'standalone') {
              cachedMode = 'standalone'
              set({ mode: 'standalone' })
              return
            }
          } catch {
            // Failed to parse JSON, fall through to daemon check
          }
        }
      } catch {
        // Network error, fall through to daemon check
      }

      // Try daemon API
      try {
        const res = await fetch('/api/v1/health')
        if (res.ok) {
          cachedMode = 'daemon'
          set({ mode: 'daemon' })
          return
        }
      } catch {
        // Not daemon mode
      }

      cachedMode = 'disconnected'
      set({ mode: 'disconnected' })
    } finally {
      isDetecting = false
    }
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
