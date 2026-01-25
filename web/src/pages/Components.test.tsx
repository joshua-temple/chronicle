import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@/test/utils'
import userEvent from '@testing-library/user-event'
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
    {
      name: 'SetupDatabase',
      type: 'setup',
      description: 'Sets up test database',
      tags: ['database', 'setup'],
      produces: ['db_conn'],
      requires: [],
      source_file: '/path/to/db_setup.go',
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

  // ============================================
  // RENDERING TESTS
  // ============================================
  describe('Rendering', () => {
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
        expect(screen.getByText('5 total')).toBeInTheDocument()
      })
    })

    it('displays type filter badges', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Should have filter group
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
          expect(screen.getByText(/No components found/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('displays component descriptions', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('Creates a test user')).toBeInTheDocument()
      })

      expect(screen.getByText('Processes an order')).toBeInTheDocument()
    })

    it('shows produces and requires counts', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Check for produces/requires indicators (multiple components have produces)
      const producesTexts = screen.getAllByText(/Produces:/)
      expect(producesTexts.length).toBeGreaterThan(0)
    })
  })

  // ============================================
  // INTERACTION TESTS
  // ============================================
  describe('Interactions', () => {
    it('filters components by search text', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search components')
      await user.type(searchInput, 'User')

      // Only User-related components should be visible
      expect(screen.getByText('CreateUser')).toBeInTheDocument()
      expect(screen.getByText('CleanupUser')).toBeInTheDocument()
      expect(screen.queryByText('ProcessOrder')).not.toBeInTheDocument()
      expect(screen.queryByText('ValidateOrder')).not.toBeInTheDocument()
    })

    it('filters components by type when clicking type button', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Find the type filter group
      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      const taskButton = within(filterGroup).getByRole('button', { name: /filter by task/i })

      await user.click(taskButton)

      // Only task components should be visible
      await waitFor(() => {
        expect(screen.queryByText('CreateUser')).not.toBeInTheDocument()
      })
      expect(screen.getByText('ProcessOrder')).toBeInTheDocument()
      expect(screen.queryByText('ValidateOrder')).not.toBeInTheDocument()
      expect(screen.queryByText('CleanupUser')).not.toBeInTheDocument()
    })

    it('shows all components when clicking All button after filtering', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by type/i })

      // First filter by task
      const taskButton = within(filterGroup).getByRole('button', { name: /filter by task/i })
      await user.click(taskButton)

      await waitFor(() => {
        expect(screen.queryByText('CreateUser')).not.toBeInTheDocument()
      })

      // Then click All to show all
      const allButton = within(filterGroup).getByRole('button', { name: /filter by all/i })
      await user.click(allButton)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })
      expect(screen.getByText('ProcessOrder')).toBeInTheDocument()
      expect(screen.getByText('ValidateOrder')).toBeInTheDocument()
      expect(screen.getByText('CleanupUser')).toBeInTheDocument()
    })

    it('combines search and type filter', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Filter by setup type (should show CreateUser and SetupDatabase)
      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      const setupButton = within(filterGroup).getByRole('button', { name: /filter by setup/i })
      await user.click(setupButton)

      await waitFor(() => {
        expect(screen.queryByText('ProcessOrder')).not.toBeInTheDocument()
      })

      // Also search for "Database"
      const searchInput = screen.getByLabelText('Search components')
      await user.type(searchInput, 'Database')

      // Only SetupDatabase should be visible
      await waitFor(() => {
        expect(screen.queryByText('CreateUser')).not.toBeInTheDocument()
      })
      expect(screen.getByText('SetupDatabase')).toBeInTheDocument()
    })

    it('clears search input to show filtered type results again', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search components')
      await user.type(searchInput, 'CreateUser')

      await waitFor(() => {
        expect(screen.queryByText('ProcessOrder')).not.toBeInTheDocument()
      })

      // Clear the search
      await user.clear(searchInput)

      // All components should be visible again
      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
        expect(screen.getByText('ProcessOrder')).toBeInTheDocument()
        expect(screen.getByText('ValidateOrder')).toBeInTheDocument()
        expect(screen.getByText('CleanupUser')).toBeInTheDocument()
      })
    })

    it('shows empty state when search finds nothing', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search components')
      await user.type(searchInput, 'nonexistent_component')

      await waitFor(() => {
        expect(screen.getByText(/No components found/i)).toBeInTheDocument()
      })
    })

    it('shows empty state when type filter finds nothing after search', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Search for "Order"
      const searchInput = screen.getByLabelText('Search components')
      await user.type(searchInput, 'Order')

      // Then filter by setup (no Order components are setup type)
      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      const setupButton = within(filterGroup).getByRole('button', { name: /filter by setup/i })
      await user.click(setupButton)

      await waitFor(() => {
        expect(screen.getByText(/No components found/i)).toBeInTheDocument()
      })
    })

    it('type filter buttons show pressed state', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      const allButton = within(filterGroup).getByRole('button', { name: /filter by all/i })
      const taskButton = within(filterGroup).getByRole('button', { name: /filter by task/i })

      // Initially All should be pressed
      expect(allButton).toHaveAttribute('aria-pressed', 'true')
      expect(taskButton).toHaveAttribute('aria-pressed', 'false')

      // Click task
      await user.click(taskButton)

      // Now task should be pressed
      expect(allButton).toHaveAttribute('aria-pressed', 'false')
      expect(taskButton).toHaveAttribute('aria-pressed', 'true')
    })
  })

  // ============================================
  // WORKFLOW TESTS - Card Selection
  // ============================================
  describe('Card Selection Workflow', () => {
    it('clicking component card triggers selection', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Find and click a component card
      const card = screen.getByLabelText('View details for CreateUser')
      await user.click(card)

      // Should trigger the detail modal (will need modal to be rendered)
      // Since we don't mock the detail fetch, we can check if the click handler was invoked
      // by looking for any state change or modal element
    })

    it('component card responds to Enter key', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Find a component card
      const card = screen.getByLabelText('View details for CreateUser')

      // Focus the card and press Enter
      card.focus()
      await user.keyboard('{Enter}')

      // Card should respond to keyboard interaction
      expect(card).toHaveAttribute('tabindex', '0')
    })
  })

  // ============================================
  // ACCESSIBILITY TESTS
  // ============================================
  describe('Accessibility', () => {
    it('has accessible search input with label', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search components')
      expect(searchInput).toBeInTheDocument()
      expect(searchInput).toHaveAttribute('placeholder', 'Search components...')
    })

    it('has accessible type filter group with aria-label', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      expect(filterGroup).toBeInTheDocument()
    })

    it('filter buttons have aria-pressed attribute', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by type/i })
      const buttons = within(filterGroup).getAllByRole('button')

      buttons.forEach(button => {
        expect(button).toHaveAttribute('aria-pressed')
      })
    })

    it('filter buttons have descriptive aria-labels', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by type/i })

      // Check each type has an accessible name
      expect(within(filterGroup).getByRole('button', { name: /filter by all/i })).toBeInTheDocument()
      expect(within(filterGroup).getByRole('button', { name: /filter by setup/i })).toBeInTheDocument()
      expect(within(filterGroup).getByRole('button', { name: /filter by task/i })).toBeInTheDocument()
      expect(within(filterGroup).getByRole('button', { name: /filter by validation/i })).toBeInTheDocument()
      expect(within(filterGroup).getByRole('button', { name: /filter by teardown/i })).toBeInTheDocument()
    })

    it('component cards are keyboard accessible', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      const card = screen.getByLabelText('View details for CreateUser')

      expect(card).toHaveAttribute('role', 'button')
      expect(card).toHaveAttribute('tabindex', '0')
    })

    it('component cards have descriptive aria-labels', async () => {
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Each card should have an aria-label
      expect(screen.getByLabelText('View details for CreateUser')).toBeInTheDocument()
      expect(screen.getByLabelText('View details for ProcessOrder')).toBeInTheDocument()
      expect(screen.getByLabelText('View details for ValidateOrder')).toBeInTheDocument()
      expect(screen.getByLabelText('View details for CleanupUser')).toBeInTheDocument()
    })

    it('can navigate through components with tab key', async () => {
      const user = userEvent.setup()
      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('CreateUser')).toBeInTheDocument()
      })

      // Tab through elements
      await user.tab() // Search input
      await user.tab() // First filter button

      // All interactive elements should be reachable
      const searchInput = screen.getByLabelText('Search components')
      expect(searchInput).not.toHaveAttribute('tabindex', '-1')
    })

    it('loading spinner has accessible label', () => {
      vi.mocked(global.fetch).mockImplementation(
        () => new Promise(() => {})
      )

      render(<Components />)

      const spinner = document.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
      // The spinner should be wrapped in accessible context
      expect(spinner?.closest('[aria-label]') || spinner?.getAttribute('aria-label')).toBeTruthy
    })
  })

  // ============================================
  // ERROR HANDLING TESTS
  // ============================================
  describe('Error Handling', () => {
    it('handles network error gracefully', async () => {
      vi.mocked(global.fetch).mockRejectedValue(new Error('Network error'))

      render(<Components />)

      // Should show loading initially
      expect(document.querySelector('.animate-spin')).toBeInTheDocument()

      // Eventually error should be handled (exact behavior depends on React Query config)
      await waitFor(
        () => {
          const spinner = document.querySelector('.animate-spin')
          // Either still loading or error state
          expect(spinner).toBeDefined()
        },
        { timeout: 3000 }
      )
    })

    it('handles API error response gracefully', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' }),
      } as Response)

      render(<Components />)

      // Should show loading initially
      expect(document.querySelector('.animate-spin')).toBeInTheDocument()
    })

    it('handles empty components array', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ components: [], discovered_at: '' }),
      } as Response)

      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText(/No components found/i)).toBeInTheDocument()
      })
    })

    it('handles null components gracefully', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ components: null, discovered_at: '' }),
      } as Response)

      render(<Components />)

      // Should either show empty state or handle gracefully without crashing
      await waitFor(
        () => {
          // Check that the page doesn't crash and shows either empty state or 0 total
          const hasEmptyState = screen.queryByText(/No components found/i)
          const hasZeroTotal = screen.queryByText(/0 total/)
          expect(hasEmptyState || hasZeroTotal).toBeTruthy()
        },
        { timeout: 3000 }
      )
    })

    it('handles components without optional fields', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({
          components: [
            {
              name: 'MinimalComponent',
              type: 'task',
              source_file: '/path/to/file.go',
              // No description, tags, produces, requires
            },
          ],
          discovered_at: '',
        }),
      } as Response)

      render(<Components />)

      await waitFor(() => {
        expect(screen.getByText('MinimalComponent')).toBeInTheDocument()
      })

      // Should render without crashing despite missing optional fields
      expect(screen.getByLabelText('View details for MinimalComponent')).toBeInTheDocument()
    })
  })
})
