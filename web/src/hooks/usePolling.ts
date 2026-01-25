import { useEffect, useCallback } from 'react'
import {
  useProjectsStore,
  POLLING_INTERVAL_ACTIVE,
  POLLING_INTERVAL_HIDDEN,
} from '@/stores/projects'

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

  const startPolling = useProjectsStore((state) => state.startPolling)
  const stopPolling = useProjectsStore((state) => state.stopPolling)
  const setPollingInterval = useProjectsStore((state) => state.setPollingInterval)

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
