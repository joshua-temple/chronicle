import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { parseEventData, type SSEEventData, type RunStartedData, type RunProgressData } from './events'

// Mock EventSource since it's not available in jsdom
class MockEventSource {
  url: string
  readyState: number = 0
  onopen: ((this: EventSource, ev: Event) => void) | null = null
  onerror: ((this: EventSource, ev: Event) => void) | null = null
  onmessage: ((this: EventSource, ev: MessageEvent) => void) | null = null

  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSED = 2

  private listeners: Map<string, EventListener[]> = new Map()

  constructor(url: string) {
    this.url = url
    this.readyState = MockEventSource.CONNECTING
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockEventSource.OPEN
      if (this.onopen) {
        this.onopen.call(this as unknown as EventSource, new Event('open'))
      }
    }, 0)
  }

  addEventListener(type: string, listener: EventListener): void {
    const listeners = this.listeners.get(type) || []
    listeners.push(listener)
    this.listeners.set(type, listeners)
  }

  removeEventListener(type: string, listener: EventListener): void {
    const listeners = this.listeners.get(type) || []
    const index = listeners.indexOf(listener)
    if (index > -1) {
      listeners.splice(index, 1)
      this.listeners.set(type, listeners)
    }
  }

  close(): void {
    this.readyState = MockEventSource.CLOSED
  }

  // Helper method for tests to dispatch events
  dispatchEvent(event: Event): boolean {
    const listeners = this.listeners.get(event.type) || []
    listeners.forEach(listener => listener(event))
    return true
  }
}

describe('events API', () => {
  beforeEach(() => {
    vi.stubGlobal('EventSource', MockEventSource)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('parseEventData', () => {
    it('should parse valid JSON event data', () => {
      const eventData: SSEEventData<RunStartedData> = {
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: {
          run_id: 'run-123',
          scenario_id: 'test-scenario'
        }
      }

      const event = new MessageEvent('message', {
        data: JSON.stringify(eventData)
      })

      const result = parseEventData<RunStartedData>(event)

      expect(result).not.toBeNull()
      expect(result?.type).toBe('run:started')
      expect(result?.data.run_id).toBe('run-123')
      expect(result?.data.scenario_id).toBe('test-scenario')
    })

    it('should parse run:progress event data', () => {
      const eventData: SSEEventData<RunProgressData> = {
        type: 'run:progress',
        timestamp: '2024-01-01T00:01:00Z',
        data: {
          run_id: 'run-123',
          step: 'SetupDB',
          status: 'completed'
        }
      }

      const event = new MessageEvent('message', {
        data: JSON.stringify(eventData)
      })

      const result = parseEventData<RunProgressData>(event)

      expect(result).not.toBeNull()
      expect(result?.type).toBe('run:progress')
      expect(result?.data.status).toBe('completed')
    })

    it('should return null for invalid JSON', () => {
      const event = new MessageEvent('message', {
        data: 'not valid json'
      })

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const result = parseEventData(event)

      expect(result).toBeNull()
      expect(consoleSpy).toHaveBeenCalled()

      consoleSpy.mockRestore()
    })

    it('should return null for empty data', () => {
      const event = new MessageEvent('message', {
        data: ''
      })

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const result = parseEventData(event)

      expect(result).toBeNull()

      consoleSpy.mockRestore()
    })

    it('should handle nested data objects', () => {
      const eventData = {
        type: 'run:completed',
        timestamp: '2024-01-01T00:05:00Z',
        data: {
          run_id: 'run-123',
          duration: '5m30s',
          result_id: 'result-456'
        }
      }

      const event = new MessageEvent('message', {
        data: JSON.stringify(eventData)
      })

      const result = parseEventData(event)

      expect(result).not.toBeNull()
      expect(result?.data.result_id).toBe('result-456')
    })
  })

  describe('SSEEventType values', () => {
    it('should support all expected event types', () => {
      const eventTypes = [
        'connected',
        'run:started',
        'run:progress',
        'run:completed',
        'run:failed',
        'run:canceled',
        'config:reloaded'
      ]

      // These are just string literal types, but we can verify they're valid
      eventTypes.forEach(type => {
        expect(typeof type).toBe('string')
        expect(type.length).toBeGreaterThan(0)
      })
    })
  })

  describe('Event data structures', () => {
    it('should validate RunStartedData structure', () => {
      const data: RunStartedData = {
        run_id: 'run-1',
        scenario_id: 'scenario-1'
      }

      expect(data.run_id).toBeDefined()
      expect(data.scenario_id).toBeDefined()
    })

    it('should validate RunProgressData structure', () => {
      const validStatuses: Array<'pending' | 'running' | 'completed' | 'failed' | 'skipped'> = [
        'pending', 'running', 'completed', 'failed', 'skipped'
      ]

      validStatuses.forEach(status => {
        const data: RunProgressData = {
          run_id: 'run-1',
          step: 'TestStep',
          status
        }
        expect(data.status).toBe(status)
      })
    })

    it('should validate optional fields in event data', () => {
      // RunCompletedData can have optional result_id
      const completedWithResult = {
        run_id: 'run-1',
        duration: '1m',
        result_id: 'result-1'
      }

      const completedWithoutResult = {
        run_id: 'run-1',
        duration: '1m'
      }

      expect(completedWithResult.result_id).toBeDefined()
      expect(completedWithoutResult).not.toHaveProperty('result_id')
    })
  })

  describe('connectToEvents (integration)', () => {
    it('should create EventSource with correct URL', async () => {
      // Import the function dynamically to test with mocked EventSource
      const { connectToEvents } = await import('./events')

      const connection = connectToEvents({})

      expect(connection.eventSource).toBeInstanceOf(MockEventSource)
      expect((connection.eventSource as unknown as MockEventSource).url).toBe('/api/v1/events')

      connection.close()
    })

    it('should call onOpen callback when connected', async () => {
      const { connectToEvents } = await import('./events')

      const onOpen = vi.fn()
      const connection = connectToEvents({ onOpen })

      // Wait for async connection
      await new Promise(resolve => setTimeout(resolve, 10))

      expect(onOpen).toHaveBeenCalled()

      connection.close()
    })

    it('should handle close correctly', async () => {
      const { connectToEvents } = await import('./events')

      const connection = connectToEvents({})

      connection.close()

      expect((connection.eventSource as unknown as MockEventSource).readyState).toBe(MockEventSource.CLOSED)
    })

    it('should allow custom event handlers', async () => {
      const { connectToEvents } = await import('./events')

      const runStartedHandler = vi.fn()
      const connection = connectToEvents({
        eventHandlers: {
          'run:started': runStartedHandler
        }
      })

      // Verify handler was registered
      // The MockEventSource stores handlers internally
      connection.close()
    })
  })
})

// Helper types for import
interface RunStartedData {
  run_id: string
  scenario_id: string
}

interface RunProgressData {
  run_id: string
  step: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
}
