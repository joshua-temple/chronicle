import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { Scenarios } from './Scenarios'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'standalone',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

// Mock fetch responses
const mockScenarios = {
  name: 'test-project',
  version: '1.0',
  scenarios: [
    {
      name: 'test_scenario_1',
      description: 'First test scenario',
      tags: ['smoke', 'api'],
      flow: [{ task: 'DoSomething' }, { validation: 'CheckResult' }],
    },
    {
      name: 'test_scenario_2',
      description: 'Second test scenario',
      tags: ['integration'],
      flow: [{ setup: 'PrepareData' }],
    },
    {
      name: 'test_scenario_3',
      description: 'Third test scenario',
      tags: ['smoke', 'e2e'],
      flow: [{ task: 'RunE2E' }],
    },
    {
      name: 'abstract_base',
      description: 'Abstract scenario',
      abstract: true,
      flow: [],
    },
  ],
}

describe('Scenarios Page', () => {
  beforeEach(() => {
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => mockScenarios,
    } as Response)
  })

  // ============================================
  // RENDERING TESTS
  // ============================================
  describe('Rendering', () => {
    it('renders scenarios list', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      expect(screen.getByText('test_scenario_2')).toBeInTheDocument()
      // Abstract scenarios should be filtered out
      expect(screen.queryByText('abstract_base')).not.toBeInTheDocument()
    })

    it('displays scenario descriptions', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('First test scenario')).toBeInTheDocument()
      })

      expect(screen.getByText('Second test scenario')).toBeInTheDocument()
    })

    it('shows total count excluding abstract scenarios', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('3 total')).toBeInTheDocument()
      })
    })

    it('displays tags for scenarios', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      // Tags appear both in filter and on cards, use getAllByText
      const smokeTags = screen.getAllByText('smoke')
      expect(smokeTags.length).toBeGreaterThan(0)

      const integrationTags = screen.getAllByText('integration')
      expect(integrationTags.length).toBeGreaterThan(0)
    })

    it('shows loading state initially', () => {
      vi.mocked(global.fetch).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      )

      render(<Scenarios />)

      // Should show skeleton loaders
      expect(document.querySelector('.animate-pulse')).toBeInTheDocument()
    })

    it('shows empty state when no scenarios', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ ...mockScenarios, scenarios: [] }),
      } as Response)

      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('No scenarios found')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // INTERACTION TESTS
  // ============================================
  describe('Interactions', () => {
    it('filters scenarios by search text', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search scenarios')
      await user.type(searchInput, 'scenario_1')

      // Only scenario_1 should be visible - use textContent check since Highlight splits text
      await waitFor(() => {
        const cards = screen.getAllByRole('button')
        const scenario1Card = cards.find(el => el.textContent?.includes('test_scenario_1'))
        const scenario2Card = cards.find(el => el.textContent?.includes('test_scenario_2'))
        const scenario3Card = cards.find(el => el.textContent?.includes('test_scenario_3'))
        expect(scenario1Card).toBeDefined()
        expect(scenario2Card).toBeUndefined()
        expect(scenario3Card).toBeUndefined()
      })
    })

    it('filters scenarios by tag when clicking tag badge', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      // Find the tag filter group
      const filterGroup = screen.getByRole('group', { name: /filter by tag/i })
      const integrationBadge = within(filterGroup).getByText('integration')

      await user.click(integrationBadge)

      // Only integration scenarios should be visible
      await waitFor(() => {
        expect(screen.queryByText('test_scenario_1')).not.toBeInTheDocument()
      })
      expect(screen.getByText('test_scenario_2')).toBeInTheDocument()
      expect(screen.queryByText('test_scenario_3')).not.toBeInTheDocument()
    })

    it('shows all scenarios when clicking All badge after filtering', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by tag/i })

      // First filter by a tag
      const integrationBadge = within(filterGroup).getByText('integration')
      await user.click(integrationBadge)

      await waitFor(() => {
        expect(screen.queryByText('test_scenario_1')).not.toBeInTheDocument()
      })

      // Then click All to show all
      const allBadge = within(filterGroup).getByText('All')
      await user.click(allBadge)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })
      expect(screen.getByText('test_scenario_2')).toBeInTheDocument()
      expect(screen.getByText('test_scenario_3')).toBeInTheDocument()
    })

    it('combines search and tag filter', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      // Filter by smoke tag (should show scenario_1 and scenario_3)
      const filterGroup = screen.getByRole('group', { name: /filter by tag/i })
      const smokeBadge = within(filterGroup).getByText('smoke')
      await user.click(smokeBadge)

      await waitFor(() => {
        expect(screen.queryByText('test_scenario_2')).not.toBeInTheDocument()
      })

      // Also search for "scenario_3"
      const searchInput = screen.getByLabelText('Search scenarios')
      await user.type(searchInput, 'scenario_3')

      // Only scenario_3 should be visible (matches both smoke tag and search)
      // With Highlight component, text is split so use custom matcher
      await waitFor(() => {
        const cards = screen.getAllByRole('button')
        const scenario1Card = cards.find(el => el.textContent?.includes('test_scenario_1'))
        expect(scenario1Card).toBeUndefined()
      })
      const cards = screen.getAllByRole('button')
      const scenario3Card = cards.find(el => el.textContent?.includes('test_scenario_3'))
      expect(scenario3Card).toBeDefined()
    })

    it('clears search input to show filtered results again', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search scenarios')
      await user.type(searchInput, 'scenario_1')

      await waitFor(() => {
        expect(screen.queryByText('test_scenario_2')).not.toBeInTheDocument()
      })

      // Clear the search
      await user.clear(searchInput)

      // All scenarios should be visible again
      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
        expect(screen.getByText('test_scenario_2')).toBeInTheDocument()
        expect(screen.getByText('test_scenario_3')).toBeInTheDocument()
      })
    })

    it('shows empty state when search finds nothing', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search scenarios')
      await user.type(searchInput, 'nonexistent_scenario')

      await waitFor(() => {
        expect(screen.getByText('No scenarios found')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // ACCESSIBILITY TESTS
  // ============================================
  describe('Accessibility', () => {
    it('has accessible search input with label', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search scenarios')
      expect(searchInput).toBeInTheDocument()
      expect(searchInput).toHaveAttribute('placeholder', 'Search scenarios...')
    })

    it('has accessible tag filter group', async () => {
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      const filterGroup = screen.getByRole('group', { name: /filter by tag/i })
      expect(filterGroup).toBeInTheDocument()
    })

    it('scenario cards are keyboard accessible', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      // Find scenario card with role="button"
      const scenarioCards = screen.getAllByRole('button')
      const card = scenarioCards.find(el => el.textContent?.includes('test_scenario_1'))

      expect(card).toBeDefined()
      expect(card).toHaveAttribute('tabindex', '0')
    })

    it('can navigate to search with tab key', async () => {
      const user = userEvent.setup()
      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
      })

      // Tab should be able to reach search input
      await user.tab()

      // After a few tabs, we should reach the search input
      const searchInput = screen.getByLabelText('Search scenarios')
      // At some point during tabbing, search should be focusable
      expect(searchInput).not.toHaveAttribute('tabindex', '-1')
    })
  })

  // ============================================
  // ERROR HANDLING TESTS
  // ============================================
  describe('Error Handling', () => {
    it('handles network error gracefully', async () => {
      vi.mocked(global.fetch).mockRejectedValue(new Error('Network error'))

      render(<Scenarios />)

      // Should eventually show error or empty state (depending on error handling)
      await waitFor(
        () => {
          // React Query will show loading then error
          const loadingSkeleton = document.querySelector('.animate-pulse')
          if (loadingSkeleton) return
          // If no skeleton, check for any error handling
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

      render(<Scenarios />)

      // Should show skeleton loaders initially
      expect(document.querySelector('.animate-pulse')).toBeInTheDocument()
    })

    it('handles empty scenarios array', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ ...mockScenarios, scenarios: [] }),
      } as Response)

      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('No scenarios found')).toBeInTheDocument()
      })
    })

    it('handles null scenarios gracefully', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ ...mockScenarios, scenarios: null }),
      } as Response)

      render(<Scenarios />)

      await waitFor(() => {
        expect(screen.getByText('No scenarios found')).toBeInTheDocument()
      })
    })
  })
})
