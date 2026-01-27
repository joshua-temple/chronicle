import { useEffect, useCallback, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import {
  useEventsStore,
  type StoredEvent,
  type ActiveRun,
  type ConnectionState,
} from '@/stores/events'
import type { SSEEventType } from '@/api/events'

/**
 * Hook to access the events store state
 * Returns events, connection state, and actions
 */
export function useEvents() {
  const events = useEventsStore(useShallow((state) => state.events))
  const connectionState = useEventsStore((state) => state.connectionState)
  const activeRuns = useEventsStore(
    useShallow((state) => Array.from(state.activeRuns.values()))
  )
  const clearEvents = useEventsStore((state) => state.clearEvents)

  return {
    events,
    connectionState,
    activeRuns,
    clearEvents,
  }
}

/**
 * Hook to get recent events with a specified limit
 * Uses shallow comparison to prevent infinite re-renders
 */
export function useRecentEvents(limit = 10): StoredEvent[] {
  const events = useEventsStore(useShallow((state) => state.events))
  return useMemo(() => events.slice(0, limit), [events, limit])
}

/**
 * Hook to get events filtered by type
 * Uses shallow comparison to prevent infinite re-renders
 */
export function useEventsByType(type: SSEEventType): StoredEvent[] {
  const events = useEventsStore(useShallow((state) => state.events))
  return useMemo(() => events.filter((event) => event.type === type), [events, type])
}

/**
 * Hook to get the current SSE connection state
 */
export function useConnectionState(): ConnectionState {
  return useEventsStore((state) => state.connectionState)
}

/**
 * Hook to get active runs from SSE events
 * Uses shallow comparison to prevent infinite re-renders
 */
export function useActiveRunsFromEvents(): ActiveRun[] {
  return useEventsStore(
    useShallow((state) => Array.from(state.activeRuns.values()))
  )
}

/**
 * Hook to manage SSE connection lifecycle
 * Connects on mount, disconnects on unmount
 *
 * @example
 * // In your root App component:
 * function App() {
 *   useEventConnection()
 *   return <Routes>...</Routes>
 * }
 */
export function useEventConnection() {
  const connect = useEventsStore((state) => state.connect)
  const disconnect = useEventsStore((state) => state.disconnect)
  const connectionState = useEventsStore((state) => state.connectionState)

  useEffect(() => {
    // Connect when component mounts
    connect()

    // Disconnect when component unmounts
    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  // Return methods to manually control connection if needed
  const reconnect = useCallback(() => {
    disconnect()
    // Small delay to ensure clean disconnect before reconnect
    setTimeout(connect, 100)
  }, [connect, disconnect])

  return {
    connectionState,
    reconnect,
    disconnect,
  }
}

/**
 * Hook to subscribe to specific event types
 * Calls the callback whenever a new event of the specified type is received
 *
 * @example
 * useEventSubscription('run:completed', (event) => {
 *   console.log('Run completed:', event.data)
 * })
 */
export function useEventSubscription(
  type: SSEEventType,
  callback: (event: StoredEvent) => void
) {
  const events = useEventsByType(type)

  useEffect(() => {
    if (events.length > 0) {
      const latestEvent = events[0]
      callback(latestEvent)
    }
  }, [events, callback])
}

export type { StoredEvent, ActiveRun, ConnectionState }
