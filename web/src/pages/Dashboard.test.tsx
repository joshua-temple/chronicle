import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { Dashboard } from './Dashboard'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'daemon',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

// Mock events store
const mockConnectionState = vi.fn()
const mockRecentEvents = vi.fn()
vi.mock('@/stores/events', () => ({
  useEventsStore: (selector: unknown) => {
    const state = {
      connectionState: mockConnectionState(),
      events: mockRecentEvents(),
    }
    return typeof selector === 'function' ? (selector as (s: typeof state) => unknown)(state) : state
  },
  selectRecentEvents: (state: { events: unknown[] }, limit: number) => state.events.slice(0, limit),
  selectEventsByType: () => [],
  selectActiveRunsArray: () => [],
}))

// Mock toast store
vi.mock('@/stores/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  },
  useToastStore: () => ({ toasts: [], addToast: vi.fn() }),
}))

const mockScenarios = [
  { name: 'test-scenario-1', description: 'First test', flowCount: 3 },
  { name: 'test-scenario-2', description: 'Second test', flowCount: 5 },
]

const mockRuns = [
  {
    id: 'run-1',
    status: 'completed',
    scenarioId: 'test-scenario-1',
    startTime: '2024-01-01T00:00:00Z',
    end_time: '2024-01-01T00:01:00Z',
    duration: '1m0s',
  },
  {
    id: 'run-2',
    status: 'running',
    scenarioId: 'test-scenario-2',
    startTime: '2024-01-01T00:02:00Z',
  },
  {
    id: 'run-3',
    status: 'failed',
    scenarioId: 'test-scenario-1',
    startTime: '2024-01-01T00:03:00Z',
    end_time: '2024-01-01T00:04:00Z',
    duration: '1m0s',
    error: 'timeout exceeded',
  },
]

describe('Dashboard Page', () => {
  beforeEach(() => {
    mockConnectionState.mockReturnValue('connected')
    mockRecentEvents.mockReturnValue([])

    vi.mocked(globalThis.fetch).mockImplementation(async (url: RequestInfo | URL) => {
      const urlStr = url.toString()
      if (urlStr.includes('/scenarios')) {
        return {
          ok: true,
          json: async () => ({ scenarios: mockScenarios, count: mockScenarios.length }),
        } as Response
      }
      if (urlStr.includes('/runs')) {
        return {
          ok: true,
          json: async () => ({ runs: mockRuns, count: mockRuns.length }),
        } as Response
      }
      return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
    })
  })

  // ============================================
  // RENDERING TESTS
  // ============================================
  describe('Rendering', () => {
    it('renders the dashboard header', async () => {
      render(<Dashboard />)

      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })

    it('shows connection status indicator', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Connected')).toBeInTheDocument()
      })
    })

    it('shows refresh button', async () => {
      render(<Dashboard />)

      const refreshButton = screen.getByRole('button', { name: /refresh runs/i })
      expect(refreshButton).toBeInTheDocument()
    })

    it('displays quick actions section with scenario selector', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Quick Actions')).toBeInTheDocument()
      })

      const selectElement = screen.getByRole('combobox', { name: /select scenario/i })
      expect(selectElement).toBeInTheDocument()
    })

    it('displays statistics cards', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Total Scenarios')).toBeInTheDocument()
      })

      expect(screen.getByText('Active Runs')).toBeInTheDocument()
      expect(screen.getByText('Recent Passed')).toBeInTheDocument()
      expect(screen.getByText('Recent Failed')).toBeInTheDocument()
    })

    it('shows correct scenario count', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        // The total scenarios count should be displayed
        const statsSection = screen.getByText('Total Scenarios').parentElement?.parentElement
        expect(statsSection).toBeInTheDocument()
        expect(within(statsSection!).getByText('2')).toBeInTheDocument()
      })
    })

    it('displays live activity section', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Live Activity')).toBeInTheDocument()
      })
    })

    it('displays recent runs section', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Recent Runs')).toBeInTheDocument()
      })
    })

    it('shows active runs when present', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText(/Active Runs \(1\)/)).toBeInTheDocument()
      })
    })

    it('shows empty events message when no events', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('No events yet. Waiting for activity...')).toBeInTheDocument()
      })
    })

    it('displays events when available', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { scenarioId: 'test-scenario', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
        {
          id: 'evt-2',
          type: 'run:completed',
          timestamp: '2024-01-01T00:01:00Z',
          data: { duration: '1m0s', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Run Started')).toBeInTheDocument()
        expect(screen.getByText('Run Completed')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // CONNECTION STATUS TESTS
  // ============================================
  describe('Connection Status', () => {
    it('shows connected status', async () => {
      mockConnectionState.mockReturnValue('connected')
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Connected')).toBeInTheDocument()
      })
    })

    it('shows connecting status', async () => {
      mockConnectionState.mockReturnValue('connecting')
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Connecting...')).toBeInTheDocument()
      })
    })

    it('shows disconnected status', async () => {
      mockConnectionState.mockReturnValue('disconnected')
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Disconnected')).toBeInTheDocument()
      })
    })

    it('shows error status', async () => {
      mockConnectionState.mockReturnValue('error')
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Connection Error')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // INTERACTION TESTS
  // ============================================
  describe('Interactions', () => {
    it('allows selecting a scenario', async () => {
      const user = userEvent.setup()
      render(<Dashboard />)

      // Wait for scenarios to load (options appear in select)
      await waitFor(() => {
        expect(screen.getByRole('option', { name: 'test-scenario-1' })).toBeInTheDocument()
      })

      const select = screen.getByRole('combobox', { name: /select scenario/i })
      await user.selectOptions(select, 'test-scenario-1')

      expect(select).toHaveValue('test-scenario-1')
    })

    it('enables run button when scenario is selected', async () => {
      const user = userEvent.setup()
      render(<Dashboard />)

      // Wait for scenarios to load
      await waitFor(() => {
        expect(screen.getByRole('option', { name: 'test-scenario-1' })).toBeInTheDocument()
      })

      const runButton = screen.getByRole('button', { name: /run scenario/i })
      expect(runButton).toBeDisabled()

      const select = screen.getByRole('combobox', { name: /select scenario/i })
      await user.selectOptions(select, 'test-scenario-1')

      expect(runButton).not.toBeDisabled()
    })

    it('can click run scenario button', async () => {
      const user = userEvent.setup()

      // Mock POST for run creation
      vi.mocked(globalThis.fetch).mockImplementation(async (url, options) => {
        const urlStr = url.toString()
        const method = (options as RequestInit)?.method || 'GET'

        if (urlStr.includes('/scenarios')) {
          return {
            ok: true,
            json: async () => ({ scenarios: mockScenarios, count: mockScenarios.length }),
          } as Response
        }
        if (urlStr.includes('/runs') && method === 'POST') {
          return {
            ok: true,
            json: async () => ({
              id: 'run-new',
              status: 'running',
              scenarioId: 'test-scenario-1',
              startTime: new Date().toISOString(),
            }),
          } as Response
        }
        if (urlStr.includes('/runs')) {
          return {
            ok: true,
            json: async () => ({ runs: mockRuns, count: mockRuns.length }),
          } as Response
        }
        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Dashboard />)

      // Wait for scenarios to load
      await waitFor(() => {
        expect(screen.getByRole('option', { name: 'test-scenario-1' })).toBeInTheDocument()
      })

      const select = screen.getByRole('combobox', { name: /select scenario/i })
      await user.selectOptions(select, 'test-scenario-1')

      const runButton = screen.getByRole('button', { name: /run scenario/i })
      await user.click(runButton)

      // After running, the select should be cleared
      await waitFor(() => {
        expect(select).toHaveValue('')
      })
    })

    it('can refresh runs', async () => {
      const user = userEvent.setup()
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /refresh runs/i })).toBeInTheDocument()
      })

      const refreshButton = screen.getByRole('button', { name: /refresh runs/i })
      await user.click(refreshButton)

      // Should trigger a refetch (fetch should be called again)
      await waitFor(() => {
        expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
          expect.stringContaining('/runs'),
          expect.anything()
        )
      })
    })
  })

  // ============================================
  // STATISTICS TESTS
  // ============================================
  describe('Statistics', () => {
    it('shows correct active run count', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        const activeRunsCard = screen.getByText('Active Runs').parentElement?.parentElement
        expect(activeRunsCard).toBeInTheDocument()
        expect(within(activeRunsCard!).getByText('1')).toBeInTheDocument()
      })
    })

    it('shows correct passed run count', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        const passedCard = screen.getByText('Recent Passed').parentElement?.parentElement
        expect(passedCard).toBeInTheDocument()
        expect(within(passedCard!).getByText('1')).toBeInTheDocument()
      })
    })

    it('shows correct failed run count', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        const failedCard = screen.getByText('Recent Failed').parentElement?.parentElement
        expect(failedCard).toBeInTheDocument()
        expect(within(failedCard!).getByText('1')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // RECENT RUNS DISPLAY TESTS
  // ============================================
  describe('Recent Runs Display', () => {
    it('shows recent runs with status icons', async () => {
      render(<Dashboard />)

      // Wait for Recent Runs section to show runs
      await waitFor(() => {
        // Look specifically in the Recent Runs section (not the dropdown)
        const recentRunsSection = screen.getByText('Recent Runs').parentElement?.parentElement
        expect(recentRunsSection).toBeInTheDocument()

        // The runs show scenarioId with font-medium class
        const runItems = document.querySelectorAll('.font-medium')
        const runTexts = Array.from(runItems).map(el => el.textContent)
        expect(runTexts).toContain('test-scenario-1')
        expect(runTexts).toContain('test-scenario-2')
      })
    })

    it('shows loading state when runs are loading', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/scenarios')) {
          return {
            ok: true,
            json: async () => ({ scenarios: mockScenarios, count: mockScenarios.length }),
          } as Response
        }
        if (urlStr.includes('/runs')) {
          return new Promise(() => {}) // Never resolves
        }
        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Loading...')).toBeInTheDocument()
      })
    })

    it('shows empty state when no runs', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/scenarios')) {
          return {
            ok: true,
            json: async () => ({ scenarios: mockScenarios, count: mockScenarios.length }),
          } as Response
        }
        if (urlStr.includes('/runs')) {
          return {
            ok: true,
            json: async () => ({ runs: [], count: 0 }),
          } as Response
        }
        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('No runs yet')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // EVENT FORMATTING TESTS
  // ============================================
  describe('Event Display', () => {
    it('formats run:started event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:started',
          timestamp: '2024-01-01T00:00:00Z',
          data: { scenarioId: 'my-scenario', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Run Started')).toBeInTheDocument()
        expect(screen.getByText('Scenario: my-scenario')).toBeInTheDocument()
      })
    })

    it('formats run:progress event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:progress',
          timestamp: '2024-01-01T00:00:00Z',
          data: { step: 'Setup', status: 'running', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Step Progress')).toBeInTheDocument()
        expect(screen.getByText('Step: Setup (running)')).toBeInTheDocument()
      })
    })

    it('formats run:completed event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:completed',
          timestamp: '2024-01-01T00:00:00Z',
          data: { duration: '2m30s', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Run Completed')).toBeInTheDocument()
        expect(screen.getByText('Duration: 2m30s')).toBeInTheDocument()
      })
    })

    it('formats run:failed event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:failed',
          timestamp: '2024-01-01T00:00:00Z',
          data: { error: 'timeout exceeded', run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Run Failed')).toBeInTheDocument()
        expect(screen.getByText('Error: timeout exceeded')).toBeInTheDocument()
      })
    })

    it('formats run:canceled event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'run:canceled',
          timestamp: '2024-01-01T00:00:00Z',
          data: { run_id: 'run-123' },
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Run Canceled')).toBeInTheDocument()
        expect(screen.getByText('Run ID: run-123')).toBeInTheDocument()
      })
    })

    it('formats config:reloaded event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'config:reloaded',
          timestamp: '2024-01-01T00:00:00Z',
          data: {},
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText('Config Reloaded')).toBeInTheDocument()
        expect(screen.getByText('Configuration reloaded')).toBeInTheDocument()
      })
    })

    it('formats connected event correctly', async () => {
      mockRecentEvents.mockReturnValue([
        {
          id: 'evt-1',
          type: 'connected',
          timestamp: '2024-01-01T00:00:00Z',
          data: {},
          receivedAt: Date.now(),
        },
      ])

      render(<Dashboard />)

      await waitFor(() => {
        const eventCards = screen.getAllByText('Connected')
        expect(eventCards.length).toBeGreaterThan(0)
      })
    })
  })

  // ============================================
  // ERROR HANDLING TESTS
  // ============================================
  describe('Error Handling', () => {
    it('handles scenarios API error gracefully', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/scenarios')) {
          return {
            ok: false,
            status: 500,
            json: async () => ({ error: 'Internal server error' }),
          } as Response
        }
        if (urlStr.includes('/runs')) {
          return {
            ok: true,
            json: async () => ({ runs: [], count: 0 }),
          } as Response
        }
        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Dashboard />)

      // Dashboard should still render even with API error
      await waitFor(() => {
        expect(screen.getByText('Dashboard')).toBeInTheDocument()
      })
    })

    it('handles runs API error gracefully', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/scenarios')) {
          return {
            ok: true,
            json: async () => ({ scenarios: mockScenarios, count: mockScenarios.length }),
          } as Response
        }
        if (urlStr.includes('/runs')) {
          return {
            ok: false,
            status: 500,
            json: async () => ({ error: 'Internal server error' }),
          } as Response
        }
        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Dashboard />)

      // Dashboard should still render
      await waitFor(() => {
        expect(screen.getByText('Dashboard')).toBeInTheDocument()
      })
    })
  })
})
