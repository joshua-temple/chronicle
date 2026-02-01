const BASE_URL = '/api/v1'

export interface SSEConnection {
  eventSource: EventSource
  close: () => void
}

export type EventHandler = (event: MessageEvent) => void

export interface ConnectOptions {
  onOpen?: () => void
  onError?: (error: Event) => void
  onMessage?: EventHandler
  eventHandlers?: Record<string, EventHandler>
  reconnectDelay?: number
  maxReconnectAttempts?: number
}

/**
 * Creates an EventSource connection to the SSE events endpoint.
 * Handles automatic reconnection on disconnect.
 */
export function connectToEvents(options: ConnectOptions = {}): SSEConnection {
  const {
    onOpen,
    onError,
    onMessage,
    eventHandlers = {},
    reconnectDelay = 3000,
    maxReconnectAttempts = 10,
  } = options

  let reconnectAttempts = 0
  let isClosedManually = false
  let currentEventSource: EventSource | null = null

  const createConnection = (): EventSource => {
    const eventSource = new EventSource(`${BASE_URL}/events`)
    currentEventSource = eventSource

    eventSource.onopen = () => {
      reconnectAttempts = 0
      onOpen?.()
    }

    eventSource.onerror = (error) => {
      onError?.(error)

      // EventSource will automatically attempt to reconnect on error
      // We track attempts to give up after max attempts
      if (!isClosedManually && eventSource.readyState === EventSource.CLOSED) {
        reconnectAttempts++
        if (reconnectAttempts < maxReconnectAttempts) {
          setTimeout(() => {
            if (!isClosedManually) {
              const newSource = createConnection()
              currentEventSource = newSource
            }
          }, reconnectDelay * Math.min(reconnectAttempts, 5))
        }
      }
    }

    // Generic message handler (for events without specific type)
    if (onMessage) {
      eventSource.onmessage = onMessage
    }

    // Register handlers for specific event types
    Object.entries(eventHandlers).forEach(([eventType, handler]) => {
      eventSource.addEventListener(eventType, handler)
    })

    return eventSource
  }

  const eventSource = createConnection()

  return {
    eventSource,
    close: () => {
      isClosedManually = true
      currentEventSource?.close()
    },
  }
}

/**
 * Event types emitted by the Chronicle SSE endpoint
 */
export type SSEEventType =
  | 'connected'
  | 'run:started'
  | 'run:progress'
  | 'run:completed'
  | 'run:failed'
  | 'run:canceled'
  | 'config:reloaded'

/**
 * Base SSE event data structure from the server
 */
export interface SSEEventData<T = unknown> {
  type: SSEEventType
  timestamp: string
  data: T
}

/**
 * Payload for run:started event
 */
export interface RunStartedData {
  run_id: string
  scenario_id: string
}

/**
 * Payload for run:progress event
 */
export interface RunProgressData {
  run_id: string
  step: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
}

/**
 * Payload for run:completed event
 */
export interface RunCompletedData {
  run_id: string
  duration: string
  result_id?: string
}

/**
 * Payload for run:failed event
 */
export interface RunFailedData {
  run_id: string
  duration: string
  error: string
}

/**
 * Payload for run:canceled event
 */
export interface RunCanceledData {
  run_id: string
  duration?: string
}

/**
 * Payload for config:reloaded event
 */
export interface ConfigReloadedData {
  timestamp: string
  scenarios_count?: number
  components_count?: number
}

/**
 * Payload for connected event
 */
export interface ConnectedData {
  status: string
}

/**
 * Union type for all possible event payloads
 */
export type EventPayload =
  | SSEEventData<RunStartedData>
  | SSEEventData<RunProgressData>
  | SSEEventData<RunCompletedData>
  | SSEEventData<RunFailedData>
  | SSEEventData<RunCanceledData>
  | SSEEventData<ConfigReloadedData>
  | SSEEventData<ConnectedData>

/**
 * Parse SSE event data from a MessageEvent
 */
export function parseEventData<T>(event: MessageEvent): SSEEventData<T> | null {
  try {
    return JSON.parse(event.data) as SSEEventData<T>
  } catch {
    console.error('Failed to parse SSE event data:', event.data)
    return null
  }
}
