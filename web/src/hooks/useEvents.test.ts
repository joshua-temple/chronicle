import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { act } from '@testing-library/react'
import { useEventsStore } from '@/stores/events'
import type { StoredEvent, ActiveRun } from '@/stores/events'

// Mock the events API to prevent EventSource issues
vi.mock('@/api/events', () => ({
  connectToEvents: vi.fn(() => ({
    close: vi.fn(),
  })),
  parseEventData: vi.fn(),
}))

// Reset the store before each test
beforeEach(() => {
  // Reset the zustand store to initial state
  useEventsStore.setState({
    connectionState: 'disconnected',
    connection: null,
    events: [],
    activeRuns: new Map(),
  })
})

afterEach(() => {
  vi.clearAllMocks()
})

describe('useEventsStore', () => {
  describe('initial state', () => {
    it('should have disconnected connection state', () => {
      const state = useEventsStore.getState()
      expect(state.connectionState).toBe('disconnected')
    })

    it('should have null connection', () => {
      const state = useEventsStore.getState()
      expect(state.connection).toBeNull()
    })

    it('should have empty events array', () => {
      const state = useEventsStore.getState()
      expect(state.events).toEqual([])
    })

    it('should have empty activeRuns map', () => {
      const state = useEventsStore.getState()
      expect(state.activeRuns.size).toBe(0)
    })
  })

  describe('addEvent', () => {
    it('should add event to the beginning of events array', () => {
      const event: StoredEvent = {
        id: '1',
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-1' },
        receivedAt: Date.now(),
      }

      act(() => {
        useEventsStore.getState().addEvent(event)
      })

      const state = useEventsStore.getState()
      expect(state.events).toHaveLength(1)
      expect(state.events[0]).toEqual(event)
    })

    it('should prepend new events', () => {
      const event1: StoredEvent = {
        id: '1',
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
        receivedAt: Date.now(),
      }
      const event2: StoredEvent = {
        id: '2',
        type: 'run:completed',
        timestamp: '2024-01-01T00:01:00Z',
        data: {},
        receivedAt: Date.now(),
      }

      act(() => {
        useEventsStore.getState().addEvent(event1)
        useEventsStore.getState().addEvent(event2)
      })

      const state = useEventsStore.getState()
      expect(state.events[0].id).toBe('2')
      expect(state.events[1].id).toBe('1')
    })

    it('should limit events to MAX_EVENTS (50)', () => {
      const events = Array.from({ length: 60 }, (_, i) => ({
        id: `${i}`,
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
        receivedAt: Date.now(),
      }))

      act(() => {
        events.forEach((event) => {
          useEventsStore.getState().addEvent(event)
        })
      })

      const state = useEventsStore.getState()
      expect(state.events.length).toBe(50)
    })
  })

  describe('clearEvents', () => {
    it('should clear all events', () => {
      const event: StoredEvent = {
        id: '1',
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
        receivedAt: Date.now(),
      }

      act(() => {
        useEventsStore.getState().addEvent(event)
      })

      expect(useEventsStore.getState().events.length).toBe(1)

      act(() => {
        useEventsStore.getState().clearEvents()
      })

      expect(useEventsStore.getState().events).toEqual([])
    })

    it('should clear activeRuns', () => {
      const activeRunsMap = new Map<string, ActiveRun>([
        ['run-1', { id: 'run-1', scenarioId: 'scenario-1', startedAt: '2024-01-01T00:00:00Z' }],
      ])

      useEventsStore.setState({ activeRuns: activeRunsMap })

      act(() => {
        useEventsStore.getState().clearEvents()
      })

      expect(useEventsStore.getState().activeRuns.size).toBe(0)
    })
  })

  describe('connect', () => {
    it('should set connection state to connecting', () => {
      act(() => {
        useEventsStore.getState().connect()
      })

      // Note: The actual connection handling is mocked
      expect(useEventsStore.getState().connectionState).toBe('connecting')
    })

    it('should not reconnect if already connecting', () => {
      useEventsStore.setState({ connectionState: 'connecting' })

      act(() => {
        useEventsStore.getState().connect()
      })

      // Should still be connecting, not reset
      expect(useEventsStore.getState().connectionState).toBe('connecting')
    })
  })

  describe('disconnect', () => {
    it('should set connection state to disconnected', () => {
      const mockClose = vi.fn()
      useEventsStore.setState({
        connectionState: 'connected',
        connection: { close: mockClose } as any,
      })

      act(() => {
        useEventsStore.getState().disconnect()
      })

      expect(useEventsStore.getState().connectionState).toBe('disconnected')
      expect(useEventsStore.getState().connection).toBeNull()
      expect(mockClose).toHaveBeenCalled()
    })

    it('should do nothing if no connection', () => {
      act(() => {
        useEventsStore.getState().disconnect()
      })

      expect(useEventsStore.getState().connectionState).toBe('disconnected')
    })
  })

  describe('handleEvent', () => {
    it('should add stored event and track run:started', () => {
      act(() => {
        useEventsStore.getState().handleEvent('run:started', {
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-1', scenarioId: 'scenario-1' },
        })
      })

      const state = useEventsStore.getState()
      expect(state.events.length).toBe(1)
      expect(state.events[0].type).toBe('run:started')
      expect(state.activeRuns.has('run-1')).toBe(true)
    })

    it('should update active run on run:progress', () => {
      // First start a run
      act(() => {
        useEventsStore.getState().handleEvent('run:started', {
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-1', scenarioId: 'scenario-1' },
        })
      })

      // Then update progress
      act(() => {
        useEventsStore.getState().handleEvent('run:progress', {
          type: 'run:progress',
          timestamp: '2024-01-01T00:01:00Z',
          data: { run_id: 'run-1', step: 'step-1', status: 'running' },
        })
      })

      const state = useEventsStore.getState()
      const activeRun = state.activeRuns.get('run-1')
      expect(activeRun?.currentStep).toBe('step-1')
      expect(activeRun?.stepStatus).toBe('running')
    })

    it('should remove active run on run:completed', () => {
      // First start a run
      act(() => {
        useEventsStore.getState().handleEvent('run:started', {
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-1', scenarioId: 'scenario-1' },
        })
      })

      expect(useEventsStore.getState().activeRuns.has('run-1')).toBe(true)

      // Then complete it
      act(() => {
        useEventsStore.getState().handleEvent('run:completed', {
          type: 'run:completed',
          timestamp: '2024-01-01T00:02:00Z',
          data: { run_id: 'run-1' },
        })
      })

      expect(useEventsStore.getState().activeRuns.has('run-1')).toBe(false)
    })

    it('should remove active run on run:failed', () => {
      act(() => {
        useEventsStore.getState().handleEvent('run:started', {
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-1', scenarioId: 'scenario-1' },
        })
      })

      act(() => {
        useEventsStore.getState().handleEvent('run:failed', {
          type: 'run:failed',
          timestamp: '2024-01-01T00:02:00Z',
          data: { run_id: 'run-1' },
        })
      })

      expect(useEventsStore.getState().activeRuns.has('run-1')).toBe(false)
    })

    it('should remove active run on run:canceled', () => {
      act(() => {
        useEventsStore.getState().handleEvent('run:started', {
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-1', scenarioId: 'scenario-1' },
        })
      })

      act(() => {
        useEventsStore.getState().handleEvent('run:canceled', {
          type: 'run:canceled',
          timestamp: '2024-01-01T00:02:00Z',
          data: { run_id: 'run-1' },
        })
      })

      expect(useEventsStore.getState().activeRuns.has('run-1')).toBe(false)
    })
  })
})

describe('Selectors', () => {
  describe('selectRecentEvents', () => {
    it('should return recent events with default limit', async () => {
      const { selectRecentEvents } = await import('@/stores/events')

      const mockEvents = Array.from({ length: 15 }, (_, i) => ({
        id: `${i}`,
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
        receivedAt: Date.now() - i * 1000,
      }))

      useEventsStore.setState({ events: mockEvents })

      const result = selectRecentEvents(useEventsStore.getState(), 10)
      expect(result).toHaveLength(10)
    })

    it('should return all events if fewer than limit', async () => {
      const { selectRecentEvents } = await import('@/stores/events')

      const mockEvents = Array.from({ length: 3 }, (_, i) => ({
        id: `${i}`,
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
        receivedAt: Date.now(),
      }))

      useEventsStore.setState({ events: mockEvents })

      const result = selectRecentEvents(useEventsStore.getState(), 10)
      expect(result).toHaveLength(3)
    })
  })

  describe('selectEventsByType', () => {
    it('should filter events by type', async () => {
      const { selectEventsByType } = await import('@/stores/events')

      const mockEvents: StoredEvent[] = [
        { id: '1', type: 'run:started', timestamp: '2024-01-01T00:00:00Z', data: {}, receivedAt: Date.now() },
        { id: '2', type: 'run:completed', timestamp: '2024-01-01T00:01:00Z', data: {}, receivedAt: Date.now() },
        { id: '3', type: 'run:started', timestamp: '2024-01-01T00:02:00Z', data: {}, receivedAt: Date.now() },
      ]

      useEventsStore.setState({ events: mockEvents })

      const result = selectEventsByType(useEventsStore.getState(), 'run:started')
      expect(result).toHaveLength(2)
      expect(result.every((e) => e.type === 'run:started')).toBe(true)
    })

    it('should return empty array when no matching events', async () => {
      const { selectEventsByType } = await import('@/stores/events')

      const mockEvents: StoredEvent[] = [
        { id: '1', type: 'run:started', timestamp: '2024-01-01T00:00:00Z', data: {}, receivedAt: Date.now() },
      ]

      useEventsStore.setState({ events: mockEvents })

      const result = selectEventsByType(useEventsStore.getState(), 'run:failed')
      expect(result).toHaveLength(0)
    })
  })

  describe('selectActiveRunsArray', () => {
    it('should return active runs as array', async () => {
      const { selectActiveRunsArray } = await import('@/stores/events')

      const activeRunsMap = new Map<string, ActiveRun>([
        ['run-1', { id: 'run-1', scenarioId: 'scenario-1', startedAt: '2024-01-01T00:00:00Z' }],
        ['run-2', { id: 'run-2', scenarioId: 'scenario-2', startedAt: '2024-01-01T00:01:00Z' }],
      ])

      useEventsStore.setState({ activeRuns: activeRunsMap })

      const result = selectActiveRunsArray(useEventsStore.getState())
      expect(result).toHaveLength(2)
    })

    it('should return empty array when no active runs', async () => {
      const { selectActiveRunsArray } = await import('@/stores/events')

      useEventsStore.setState({ activeRuns: new Map() })

      const result = selectActiveRunsArray(useEventsStore.getState())
      expect(result).toEqual([])
    })
  })
})
