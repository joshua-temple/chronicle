import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/utils'
import { Components } from './Components'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'standalone',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

const mockComponents = {
  components: [
    {
      name: 'CreateUser',
      type: 'setup',
      description: 'Creates a test user',
      tags: ['user', 'setup'],
      produces: ['user_id'],
      requires: [],
      source_file: '/path/to/setup.go',
    },
    {
      name: 'ProcessOrder',
      type: 'task',
      description: 'Processes an order',
      tags: ['order'],
      produces: ['order_id'],
      requires: ['user_id'],
      source_file: '/path/to/tasks.go',
    },
    {
      name: 'ValidateOrder',
      type: 'validation',
      description: 'Validates order was created',
      tags: ['order', 'validation'],
      produces: [],
      requires: ['order_id'],
      source_file: '/path/to/validations.go',
    },
    {
      name: 'CleanupUser',
      type: 'teardown',
      description: 'Removes test user',
      tags: ['cleanup'],
      produces: [],
      requires: ['user_id'],
      source_file: '/path/to/teardown.go',
    },
  ],
  discovered_at: '2026-01-25T10:00:00Z',
}

describe('Components Page', () => {
  beforeEach(() => {
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => mockComponents,
    } as Response)
  })

  it('renders components list', async () => {
    render(<Components />)

    await waitFor(() => {
      expect(screen.getByText('CreateUser')).toBeInTheDocument()
    })

    expect(screen.getByText('ProcessOrder')).toBeInTheDocument()
    expect(screen.getByText('ValidateOrder')).toBeInTheDocument()
    expect(screen.getByText('CleanupUser')).toBeInTheDocument()
  })

  it('displays component types', async () => {
    render(<Components />)

    await waitFor(() => {
      expect(screen.getByText('CreateUser')).toBeInTheDocument()
    })

    // Types appear as badges on cards and in filter, use getAllByText
    const setupBadges = screen.getAllByText('setup')
    expect(setupBadges.length).toBeGreaterThan(0)

    const taskBadges = screen.getAllByText('task')
    expect(taskBadges.length).toBeGreaterThan(0)

    const validationBadges = screen.getAllByText('validation')
    expect(validationBadges.length).toBeGreaterThan(0)

    const teardownBadges = screen.getAllByText('teardown')
    expect(teardownBadges.length).toBeGreaterThan(0)
  })

  it('shows total component count', async () => {
    render(<Components />)

    await waitFor(() => {
      expect(screen.getByText('4 total')).toBeInTheDocument()
    })
  })

  it('displays type filter badges', async () => {
    render(<Components />)

    await waitFor(() => {
      expect(screen.getByText('CreateUser')).toBeInTheDocument()
    })

    // Should have "All" filter badge
    expect(screen.getByRole('group', { name: /filter by type/i })).toBeInTheDocument()
  })

  it('shows loading state initially', () => {
    vi.mocked(global.fetch).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    )

    render(<Components />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('shows empty state when no components', async () => {
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ components: [], discovered_at: '' }),
    } as Response)

    render(<Components />)

    await waitFor(
      () => {
        // The actual message includes "matching your filters"
        expect(screen.getByText(/No components found/i)).toBeInTheDocument()
      },
      { timeout: 3000 }
    )
  })
})
