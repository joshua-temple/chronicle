import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, within } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { ProjectSelector } from './ProjectSelector'
import {
  useProjectsStore,
  _resetOperationFlags,
  POLLING_INTERVAL_ACTIVE,
} from '@/stores/projects'
import type { Project } from '@/stores/projects'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'standalone',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

const mockProjects: Project[] = [
  {
    id: 'proj-1',
    name: 'chronicle-main',
    path: '/home/user/projects/chronicle',
    addedAt: '2024-01-15T10:00:00Z',
    lastOpened: '2024-01-20T15:30:00Z',
    status: { state: 'running', port: 8080 },
  },
  {
    id: 'proj-2',
    name: 'test-service',
    path: '/home/user/projects/test-service',
    addedAt: '2024-01-10T08:00:00Z',
    status: { state: 'stopped' },
  },
  {
    id: 'proj-3',
    name: 'remote-daemon',
    remoteUrl: 'https://chronicle.example.com:8080',
    addedAt: '2024-01-12T12:00:00Z',
    lastOpened: '2024-01-19T09:00:00Z',
    status: { state: 'running', port: 8080 },
  },
]

const mockDiscoveredProjects: Project[] = [
  {
    id: 'discovered-1',
    name: 'new-project',
    path: '/home/user/projects/new-project',
    addedAt: '',
    autoDiscovered: true,
    status: { state: 'unknown' },
  },
]

describe('ProjectSelector', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    _resetOperationFlags()
    // Stop any existing polling and reset store state
    const state = useProjectsStore.getState()
    if (state.pollingIntervalId !== null) {
      state.stopPolling()
    }
    useProjectsStore.setState({
      projects: [],
      discovered: [],
      loading: false,
      error: null,
      activeProjectId: null,
      pollingIntervalId: null,
      pollingIntervalMs: POLLING_INTERVAL_ACTIVE,
    })
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ projects: mockProjects }),
    } as Response)
  })

  afterEach(() => {
    // Clean up polling
    const state = useProjectsStore.getState()
    if (state.pollingIntervalId !== null) {
      state.stopPolling()
    }
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  // ============================================
  // LOADING STATE TESTS
  // ============================================
  describe('Loading State', () => {
    it('shows loading skeletons when loading with no projects', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      )

      render(<ProjectSelector />)

      // Should show skeleton loaders
      expect(document.querySelector('.animate-pulse')).toBeInTheDocument()
    })

    it('shows refresh button with spinner when loading', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      )

      render(<ProjectSelector />)

      const refreshButton = screen.getByRole('button', { name: /refresh projects/i })
      expect(refreshButton).toBeDisabled()
      // Spinner should have animate-spin class on the SVG
      const spinner = refreshButton.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })
  })

  // ============================================
  // ERROR DISPLAY TESTS
  // ============================================
  describe('Error Display', () => {
    it('displays error message when fetch fails', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
        json: async () => ({}),
      } as Response)

      render(<ProjectSelector />)

      // With polling, both fetch and discover run, so we match any error
      await waitFor(() => {
        expect(screen.getByText(/failed to (fetch|discover) projects/i)).toBeInTheDocument()
      })
    })

    it('shows dismiss button for errors', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
        json: async () => ({}),
      } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /dismiss/i })).toBeInTheDocument()
      })
    })

    it('shows retry button for errors', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
        json: async () => ({}),
      } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
      })
    })

    it('clears error when dismiss is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

      // All calls fail
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
        json: async () => ({}),
      } as Response)

      render(<ProjectSelector />)

      // With polling, both fetch and discover run, so we match any error
      await waitFor(() => {
        expect(screen.getByText(/failed to (fetch|discover) projects/i)).toBeInTheDocument()
      })

      const dismissButton = screen.getByRole('button', { name: /dismiss/i })
      await user.click(dismissButton)

      await waitFor(() => {
        expect(screen.queryByText(/failed to (fetch|discover) projects/i)).not.toBeInTheDocument()
      })
    })
  })

  // ============================================
  // PROJECT LIST RENDERING TESTS
  // ============================================
  describe('Project List Rendering', () => {
    it('renders project list after loading', async () => {
      // Mock both fetchProjects and discover calls
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockProjects }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [] }),
        } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('chronicle-main')).toBeInTheDocument()
      })

      expect(screen.getByText('test-service')).toBeInTheDocument()
      expect(screen.getByText('remote-daemon')).toBeInTheDocument()
    })

    it('shows correct project paths', async () => {
      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('/home/user/projects/chronicle')).toBeInTheDocument()
      })

      expect(screen.getByText('/home/user/projects/test-service')).toBeInTheDocument()
    })

    it('shows remote URL for remote projects', async () => {
      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('https://chronicle.example.com:8080')).toBeInTheDocument()
      })
    })

    it('shows empty state when no projects exist', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('No projects yet')).toBeInTheDocument()
      })

      expect(screen.getByText(/add a chronicle project to get started/i)).toBeInTheDocument()
    })

    it('shows Add Project button in empty state', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /add project/i })).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // DISCOVERED PROJECTS TESTS
  // ============================================
  describe('Discovered Projects', () => {
    it('shows discovered projects section', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockProjects }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockDiscoveredProjects }),
        } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('Discovered Projects')).toBeInTheDocument()
      })

      expect(screen.getByText('new-project')).toBeInTheDocument()
    })

    it('filters out already registered projects from discovered list', async () => {
      const discoveredWithDuplicate = [
        ...mockDiscoveredProjects,
        {
          id: 'discovered-dup',
          name: 'chronicle-main',
          path: '/home/user/projects/chronicle', // Same path as registered project
          addedAt: '',
          autoDiscovered: true,
          status: { state: 'unknown' as const },
        },
      ]

      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockProjects }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: discoveredWithDuplicate }),
        } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('Discovered Projects')).toBeInTheDocument()
      })

      // Should only show new-project, not the duplicate
      const discoveredSection = screen.getByText('Discovered Projects').closest('section')
      expect(discoveredSection).toBeInTheDocument()

      // The duplicate should not appear in discovered section
      const discoveredCards = within(discoveredSection!).queryAllByText('/home/user/projects/chronicle')
      expect(discoveredCards.length).toBe(0)
    })
  })

  // ============================================
  // INTERACTION TESTS
  // ============================================
  describe('Interactions', () => {
    it('opens add project modal when Add Project button is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

      // Mock both fetchProjects and discover calls
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockProjects }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [] }),
        } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('chronicle-main')).toBeInTheDocument()
      })

      // Find the header Add Project button (not the one in empty state)
      const addButtons = screen.getAllByRole('button', { name: /add project/i })
      const headerAddButton = addButtons[0] // Header button is first
      await user.click(headerAddButton)

      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
        expect(screen.getByRole('heading', { name: /add project/i })).toBeInTheDocument()
      })
    })

    it('refreshes projects when refresh button is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('chronicle-main')).toBeInTheDocument()
      })

      vi.mocked(globalThis.fetch).mockClear()
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: mockProjects }),
      } as Response)

      const refreshButton = screen.getByRole('button', { name: /refresh projects/i })
      await user.click(refreshButton)

      await waitFor(() => {
        expect(globalThis.fetch).toHaveBeenCalledWith('/api/standalone/projects')
      })
    })

    it('triggers scan when Scan Again button is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockProjects }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: mockDiscoveredProjects }),
        } as Response)

      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByText('Discovered Projects')).toBeInTheDocument()
      })

      vi.mocked(globalThis.fetch).mockClear()
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: mockDiscoveredProjects }),
      } as Response)

      const scanButton = screen.getByRole('button', { name: /scan again/i })
      await user.click(scanButton)

      await waitFor(() => {
        expect(globalThis.fetch).toHaveBeenCalledWith('/api/standalone/discover', expect.any(Object))
      })
    })
  })

  // ============================================
  // ACCESSIBILITY TESTS
  // ============================================
  describe('Accessibility', () => {
    it('has accessible header', async () => {
      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /chronicle control center/i })).toBeInTheDocument()
      })
    })

    it('has accessible refresh button with aria-label', async () => {
      render(<ProjectSelector />)

      await waitFor(() => {
        const refreshButton = screen.getByRole('button', { name: /refresh projects/i })
        expect(refreshButton).toBeInTheDocument()
      })
    })

    it('project sections have proper headings', async () => {
      render(<ProjectSelector />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /your projects/i })).toBeInTheDocument()
      })
    })
  })
})
