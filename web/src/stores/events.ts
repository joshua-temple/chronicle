import { create } from 'zustand'
import {
  connectToEvents,
  parseEventData,
  type SSEConnection,
  type SSEEventType,
  type SSEEventData,
  type RunStartedData,
  type RunProgressData,
  type RunCompletedData,
  type RunFailedData,
  type RunCanceledData,
  type ConfigReloadedData,
} from '@/api/events'

/**
 * Maximum number of events to keep in the store
 */
const MAX_EVENTS = 50

/**
 * Represents a stored event with normalized structure
 */
export interface StoredEvent {
  id: string
  type: SSEEventType
  timestamp: string
  data: unknown
  receivedAt: number
}

/**
 * Represents an active run being tracked
 */
export interface ActiveRun {
  id: string
  scenarioId: string
  startedAt: string
  currentStep?: string
  stepStatus?: string
}

/**
 * Connection state for the SSE connection
 */
export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

/**
 * Events store state interface
 */
interface EventsState {
  // Connection state
  connectionState: ConnectionState
  connection: SSEConnection | null

  // Events
  events: StoredEvent[]
  activeRuns: Map<string, ActiveRun>

  // Actions
  connect: () => void
  disconnect: () => void
  addEvent: (event: StoredEvent) => void
  clearEvents: () => void

  // Internal handlers
  handleEvent: (type: SSEEventType, data: SSEEventData<unknown>) => void
}

/**
 * Generate a unique ID for events
 */
function generateEventId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

/**
 * Zustand store for managing SSE events and connection state
 */
export const useEventsStore = create<EventsState>((set, get) => ({
  // Initial state
  connectionState: 'disconnected',
  connection: null,
  events: [],
  activeRuns: new Map(),

  // Connect to SSE endpoint
  connect: () => {
    const { connection, connectionState } = get()

    // Don't reconnect if already connected or connecting
    if (connection || connectionState === 'connecting') {
      return
    }

    set({ connectionState: 'connecting' })

    const sseConnection = connectToEvents({
      onOpen: () => {
        set({ connectionState: 'connected' })
      },
      onError: () => {
        // Only set error state if we're not intentionally disconnected
        const { connectionState } = get()
        if (connectionState !== 'disconnected') {
          set({ connectionState: 'error' })
        }
      },
      eventHandlers: {
        connected: (event) => {
          const data = parseEventData(event)
          if (data) {
            get().handleEvent('connected', data)
          }
        },
        'run:started': (event) => {
          const data = parseEventData<RunStartedData>(event)
          if (data) {
            get().handleEvent('run:started', data)
          }
        },
        'run:progress': (event) => {
          const data = parseEventData<RunProgressData>(event)
          if (data) {
            get().handleEvent('run:progress', data)
          }
        },
        'run:completed': (event) => {
          const data = parseEventData<RunCompletedData>(event)
          if (data) {
            get().handleEvent('run:completed', data)
          }
        },
        'run:failed': (event) => {
          const data = parseEventData<RunFailedData>(event)
          if (data) {
            get().handleEvent('run:failed', data)
          }
        },
        'run:canceled': (event) => {
          const data = parseEventData<RunCanceledData>(event)
          if (data) {
            get().handleEvent('run:canceled', data)
          }
        },
        'config:reloaded': (event) => {
          const data = parseEventData<ConfigReloadedData>(event)
          if (data) {
            get().handleEvent('config:reloaded', data)
          }
        },
      },
    })

    set({ connection: sseConnection })
  },

  // Disconnect from SSE endpoint
  disconnect: () => {
    const { connection } = get()
    if (connection) {
      connection.close()
      set({ connection: null, connectionState: 'disconnected' })
    }
  },

  // Add an event to the store
  addEvent: (event) => {
    set((state) => {
      const newEvents = [event, ...state.events].slice(0, MAX_EVENTS)
      return { events: newEvents }
    })
  },

  // Clear all events
  clearEvents: () => {
    set({ events: [], activeRuns: new Map() })
  },

  // Handle incoming SSE events
  handleEvent: (type, eventData) => {
    const storedEvent: StoredEvent = {
      id: generateEventId(),
      type,
      timestamp: eventData.timestamp,
      data: eventData.data,
      receivedAt: Date.now(),
    }

    get().addEvent(storedEvent)

    // Update active runs based on event type
    set((state) => {
      const activeRuns = new Map(state.activeRuns)

      switch (type) {
        case 'run:started': {
          const data = eventData.data as RunStartedData
          activeRuns.set(data.run_id, {
            id: data.run_id,
            scenarioId: data.scenario_id,
            startedAt: eventData.timestamp,
          })
          break
        }
        case 'run:progress': {
          const data = eventData.data as RunProgressData
          const existing = activeRuns.get(data.run_id)
          if (existing) {
            activeRuns.set(data.run_id, {
              ...existing,
              currentStep: data.step,
              stepStatus: data.status,
            })
          }
          break
        }
        case 'run:completed':
        case 'run:failed':
        case 'run:canceled': {
          const data = eventData.data as { run_id: string }
          activeRuns.delete(data.run_id)
          break
        }
      }

      return { activeRuns }
    })
  },
}))

/**
 * Selector for getting recent events
 */
export function selectRecentEvents(state: EventsState, limit = 10): StoredEvent[] {
  return state.events.slice(0, limit)
}

/**
 * Selector for getting events by type
 */
export function selectEventsByType(state: EventsState, type: SSEEventType): StoredEvent[] {
  return state.events.filter((event) => event.type === type)
}

/**
 * Selector for getting active runs as array
 */
export function selectActiveRunsArray(state: EventsState): ActiveRun[] {
  return Array.from(state.activeRuns.values())
}
