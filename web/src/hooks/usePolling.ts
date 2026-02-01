import { useEffect, useCallback, useRef } from 'react'
import { useProjectsStore } from '@/stores/projects'

// Polling intervals
export const POLLING_INTERVAL_ACTIVE = 5000 // 5 seconds when tab is visible
export const POLLING_INTERVAL_HIDDEN = 60000 // 60 seconds when tab is hidden

interface UsePollingOptions {
  /** Polling interval when tab is visible (default: 5000ms) */
  activeInterval?: number
  /** Polling interval when tab is hidden (default: 60000ms) */
  hiddenInterval?: number
  /** Whether to start polling automatically on mount (default: true) */
  autoStart?: boolean
}

/**
 * Hook to manage project status polling with visibility-aware intervals.
 *
 * Polls more frequently when the tab is visible and reduces frequency
 * when the tab is hidden to conserve resources.
 *
 * @example
 * ```tsx
 * // In ProjectSelector component
 * usePolling() // Uses defaults: 5s active, 60s hidden
 *
 * // With custom intervals
 * usePolling({ activeInterval: 3000, hiddenInterval: 30000 })
 * ```
 */
export function usePolling(options: UsePollingOptions = {}) {
  const {
    activeInterval = POLLING_INTERVAL_ACTIVE,
    hiddenInterval = POLLING_INTERVAL_HIDDEN,
    autoStart = true,
  } = options

  const refreshAllHealth = useProjectsStore((state) => state.refreshAllHealth)
  const intervalRef = useRef<number | null>(null)
  const currentIntervalMs = useRef(activeInterval)

  const startPolling = useCallback((intervalMs: number) => {
    // Clear any existing interval
    if (intervalRef.current !== null) {
      window.clearInterval(intervalRef.current)
    }

    currentIntervalMs.current = intervalMs
    intervalRef.current = window.setInterval(() => {
      refreshAllHealth()
    }, intervalMs)
  }, [refreshAllHealth])

  const stopPolling = useCallback(() => {
    if (intervalRef.current !== null) {
      window.clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  const setPollingInterval = useCallback((intervalMs: number) => {
    if (intervalRef.current !== null) {
      startPolling(intervalMs)
    }
  }, [startPolling])

  const handleVisibilityChange = useCallback(() => {
    if (document.hidden) {
      setPollingInterval(hiddenInterval)
    } else {
      setPollingInterval(activeInterval)
    }
  }, [activeInterval, hiddenInterval, setPollingInterval])

  useEffect(() => {
    if (!autoStart) {
      return
    }

    // Start polling with the appropriate interval based on current visibility
    const initialInterval = document.hidden ? hiddenInterval : activeInterval
    startPolling(initialInterval)

    // Listen for visibility changes
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      stopPolling()
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [autoStart, activeInterval, hiddenInterval, startPolling, stopPolling, handleVisibilityChange])

  return {
    startPolling,
    stopPolling,
    setPollingInterval,
  }
}
