import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  useEventsStore,
  selectRecentEvents,
  selectEventsByType,
  selectActiveRunsArray,
  type StoredEvent,
  type ActiveRun,
  type ConnectionState,
} from './events'

// Mock the events API
vi.mock('@/api/events', () => ({
  connectToEvents: vi.fn(() => ({
    close: vi.fn(),
  })),
  parseEventData: vi.fn((event) => event),
}))

describe('Events Store', () => {
  beforeEach(() => {
    // Reset store state
    useEventsStore.setState({
      connectionState: 'disconnected',
      connection: null,
      events: [],
      activeRuns: new Map(),
    })
  })

  describe('Initial State', () => {
    it('should have disconnected connection state', () => {
      const state = useEventsStore.getState()
      expect(state.connectionState).toBe('disconnected')
    })

    it('should have no connection', () => {
      const state = useEventsStore.getState()
      expect(state.connection).toBeNull()
    })

    it('should have empty events array', () => {
      const state = useEventsStore.getState()
      expect(state.events).toEqual([])
    })

    it('should have empty active runs map', () => {
      const state = useEventsStore.getState()
      expect(state.activeRuns.size).toBe(0)
    })
  })

  describe('addEvent', () => {
    it('should add event to the beginning of events array', () => {
      const event1: StoredEvent = {
        id: 'evt-1',
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-1' },
        receivedAt: Date.now(),
      }

      const event2: StoredEvent = {
        id: 'evt-2',
        type: 'run:completed',
        timestamp: '2024-01-01T00:01:00Z',
        data: { run_id: 'run-1' },
        receivedAt: Date.now(),
      }

      useEventsStore.getState().addEvent(event1)
      useEventsStore.getState().addEvent(event2)

      const events = useEventsStore.getState().events
      expect(events).toHaveLength(2)
      // Most recent should be first
      expect(events[0].id).toBe('evt-2')
      expect(events[1].id).toBe('evt-1')
    })

    it('should limit events to MAX_EVENTS (50)', () => {
      // Add 60 events
      for (let i = 0; i < 60; i++) {
        useEventsStore.getState().addEvent({
          id: `evt-${i}`,
          type: 'run:started',
          timestamp: new Date().toISOString(),
          data: { run_id: `run-${i}` },
          receivedAt: Date.now(),
        })
      }

      const events = useEventsStore.getState().events
      expect(events).toHaveLength(50)
      // Should have the most recent 50 events
      expect(events[0].id).toBe('evt-59')
    })
  })

  describe('clearEvents', () => {
    it('should clear all events', () => {
      // Add some events
      useEventsStore.getState().addEvent({
        id: 'evt-1',
        type: 'run:started',
        timestamp: new Date().toISOString(),
        data: {},
        receivedAt: Date.now(),
      })
      useEventsStore.getState().addEvent({
        id: 'evt-2',
        type: 'run:completed',
        timestamp: new Date().toISOString(),
        data: {},
        receivedAt: Date.now(),
      })

      expect(useEventsStore.getState().events).toHaveLength(2)

      useEventsStore.getState().clearEvents()

      expect(useEventsStore.getState().events).toHaveLength(0)
    })

    it('should clear active runs', () => {
      useEventsStore.setState({
        activeRuns: new Map([
          ['run-1', { id: 'run-1', scenarioId: 'test', startedAt: new Date().toISOString() }],
        ]),
      })

      expect(useEventsStore.getState().activeRuns.size).toBe(1)

      useEventsStore.getState().clearEvents()

      expect(useEventsStore.getState().activeRuns.size).toBe(0)
    })
  })

  describe('handleEvent', () => {
    it('should handle run:started event', () => {
      const eventData = {
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {
          run_id: 'run-123',
          scenarioId: 'test-scenario',
        },
      }

      useEventsStore.getState().handleEvent('run:started', eventData)

      const state = useEventsStore.getState()
      expect(state.events).toHaveLength(1)
      expect(state.events[0].type).toBe('run:started')
      expect(state.activeRuns.has('run-123')).toBe(true)
      expect(state.activeRuns.get('run-123')?.scenarioId).toBe('test-scenario')
    })

    it('should handle run:progress event', () => {
      // First start a run
      useEventsStore.getState().handleEvent('run:started', {
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-123', scenarioId: 'test' },
      })

      // Then update progress
      useEventsStore.getState().handleEvent('run:progress', {
        type: 'run:progress' as const,
        timestamp: '2024-01-01T00:00:01Z',
        data: {
          run_id: 'run-123',
          step: 'Setup',
          status: 'running',
        },
      })

      const activeRun = useEventsStore.getState().activeRuns.get('run-123')
      expect(activeRun?.currentStep).toBe('Setup')
      expect(activeRun?.stepStatus).toBe('running')
    })

    it('should handle run:completed event', () => {
      // Start a run
      useEventsStore.getState().handleEvent('run:started', {
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-123', scenarioId: 'test' },
      })

      expect(useEventsStore.getState().activeRuns.has('run-123')).toBe(true)

      // Complete the run
      useEventsStore.getState().handleEvent('run:completed', {
        type: 'run:completed' as const,
        timestamp: '2024-01-01T00:01:00Z',
        data: { run_id: 'run-123' },
      })

      expect(useEventsStore.getState().activeRuns.has('run-123')).toBe(false)
    })

    it('should handle run:failed event', () => {
      // Start a run
      useEventsStore.getState().handleEvent('run:started', {
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-123', scenarioId: 'test' },
      })

      // Fail the run
      useEventsStore.getState().handleEvent('run:failed', {
        type: 'run:failed' as const,
        timestamp: '2024-01-01T00:01:00Z',
        data: { run_id: 'run-123' },
      })

      expect(useEventsStore.getState().activeRuns.has('run-123')).toBe(false)
    })

    it('should handle run:canceled event', () => {
      // Start a run
      useEventsStore.getState().handleEvent('run:started', {
        type: 'run:started' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-123', scenarioId: 'test' },
      })

      // Cancel the run
      useEventsStore.getState().handleEvent('run:canceled', {
        type: 'run:canceled' as const,
        timestamp: '2024-01-01T00:01:00Z',
        data: { run_id: 'run-123' },
      })

      expect(useEventsStore.getState().activeRuns.has('run-123')).toBe(false)
    })

    it('should handle config:reloaded event', () => {
      useEventsStore.getState().handleEvent('config:reloaded', {
        type: 'config:reloaded' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
      })

      const events = useEventsStore.getState().events
      expect(events).toHaveLength(1)
      expect(events[0].type).toBe('config:reloaded')
    })

    it('should handle connected event', () => {
      useEventsStore.getState().handleEvent('connected', {
        type: 'connected' as const,
        timestamp: '2024-01-01T00:00:00Z',
        data: {},
      })

      const events = useEventsStore.getState().events
      expect(events).toHaveLength(1)
      expect(events[0].type).toBe('connected')
    })
  })

  describe('Selectors', () => {
    beforeEach(() => {
      // Add test events
      const events: StoredEvent[] = [
        { id: 'evt-1', type: 'run:started', timestamp: '2024-01-01T00:00:00Z', data: {}, receivedAt: 1 },
        { id: 'evt-2', type: 'run:completed', timestamp: '2024-01-01T00:01:00Z', data: {}, receivedAt: 2 },
        { id: 'evt-3', type: 'run:started', timestamp: '2024-01-01T00:02:00Z', data: {}, receivedAt: 3 },
        { id: 'evt-4', type: 'run:failed', timestamp: '2024-01-01T00:03:00Z', data: {}, receivedAt: 4 },
        { id: 'evt-5', type: 'config:reloaded', timestamp: '2024-01-01T00:04:00Z', data: {}, receivedAt: 5 },
      ]
      useEventsStore.setState({ events })
    })

    describe('selectRecentEvents', () => {
      it('should return recent events with default limit', () => {
        const state = useEventsStore.getState()
        const recent = selectRecentEvents(state)
        expect(recent).toHaveLength(5) // All 5 events (limit defaults to 10)
      })

      it('should respect custom limit', () => {
        const state = useEventsStore.getState()
        const recent = selectRecentEvents(state, 3)
        expect(recent).toHaveLength(3)
        expect(recent[0].id).toBe('evt-1')
        expect(recent[2].id).toBe('evt-3')
      })
    })

    describe('selectEventsByType', () => {
      it('should filter events by type', () => {
        const state = useEventsStore.getState()

        const started = selectEventsByType(state, 'run:started')
        expect(started).toHaveLength(2)

        const completed = selectEventsByType(state, 'run:completed')
        expect(completed).toHaveLength(1)

        const reloaded = selectEventsByType(state, 'config:reloaded')
        expect(reloaded).toHaveLength(1)
      })

      it('should return empty array for non-existent type', () => {
        const state = useEventsStore.getState()
        const events = selectEventsByType(state, 'run:progress')
        expect(events).toHaveLength(0)
      })
    })

    describe('selectActiveRunsArray', () => {
      it('should return active runs as array', () => {
        const activeRuns = new Map<string, ActiveRun>([
          ['run-1', { id: 'run-1', scenarioId: 'test-1', startedAt: '2024-01-01T00:00:00Z' }],
          ['run-2', { id: 'run-2', scenarioId: 'test-2', startedAt: '2024-01-01T00:01:00Z' }],
        ])
        useEventsStore.setState({ activeRuns })

        const state = useEventsStore.getState()
        const runs = selectActiveRunsArray(state)

        expect(runs).toHaveLength(2)
        expect(runs.map(r => r.id).sort()).toEqual(['run-1', 'run-2'])
      })

      it('should return empty array when no active runs', () => {
        const state = useEventsStore.getState()
        const runs = selectActiveRunsArray(state)
        expect(runs).toHaveLength(0)
      })
    })
  })

  describe('Connection State', () => {
    it('should track connection state changes', () => {
      const states: ConnectionState[] = ['disconnected', 'connecting', 'connected', 'error']

      states.forEach(state => {
        useEventsStore.setState({ connectionState: state })
        expect(useEventsStore.getState().connectionState).toBe(state)
      })
    })
  })

  describe('StoredEvent Interface', () => {
    it('should have correct structure', () => {
      const event: StoredEvent = {
        id: 'test-id',
        type: 'run:started',
        timestamp: '2024-01-01T00:00:00Z',
        data: { run_id: 'run-123' },
        receivedAt: Date.now(),
      }

      expect(event).toHaveProperty('id')
      expect(event).toHaveProperty('type')
      expect(event).toHaveProperty('timestamp')
      expect(event).toHaveProperty('data')
      expect(event).toHaveProperty('receivedAt')
    })
  })

  describe('ActiveRun Interface', () => {
    it('should have correct structure', () => {
      const run: ActiveRun = {
        id: 'run-123',
        scenarioId: 'test-scenario',
        startedAt: '2024-01-01T00:00:00Z',
        currentStep: 'Setup',
        stepStatus: 'running',
      }

      expect(run).toHaveProperty('id')
      expect(run).toHaveProperty('scenarioId')
      expect(run).toHaveProperty('startedAt')
      expect(run).toHaveProperty('currentStep')
      expect(run).toHaveProperty('stepStatus')
    })

    it('should allow optional step fields', () => {
      const run: ActiveRun = {
        id: 'run-123',
        scenarioId: 'test-scenario',
        startedAt: '2024-01-01T00:00:00Z',
      }

      expect(run.currentStep).toBeUndefined()
      expect(run.stepStatus).toBeUndefined()
    })
  })
})
