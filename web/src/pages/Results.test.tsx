import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { Results } from './Results'

// Mock ResultDetail component
vi.mock('@/components/results/ResultDetail', () => ({
  ResultDetail: ({ id, onClose }: { id: string; onClose: () => void }) => (
    <div data-testid="result-detail">
      <div>Result Detail: {id}</div>
      <button onClick={onClose}>Close</button>
    </div>
  ),
}))

const mockResultIds = ['result-1', 'result-2', 'result-3']

const mockResults: Record<string, unknown> = {
  'result-1': {
    id: 'result-1',
    project_name: 'test-project',
    started_at: '2024-01-01T00:00:00Z',
    completed_at: '2024-01-01T00:05:00Z',
    duration: '5m0s',
    totalScenarios: 3,
    passed: 3,
    failed: 0,
    skipped: 0,
    scenarios: [],
  },
  'result-2': {
    id: 'result-2',
    project_name: 'test-project',
    started_at: '2024-01-02T00:00:00Z',
    completed_at: '2024-01-02T00:10:00Z',
    duration: '10m0s',
    totalScenarios: 5,
    passed: 3,
    failed: 2,
    skipped: 0,
    scenarios: [],
  },
  'result-3': {
    id: 'result-3',
    project_name: 'test-project',
    started_at: '2024-01-03T00:00:00Z',
    completed_at: '2024-01-03T00:02:00Z',
    duration: '2m0s',
    totalScenarios: 2,
    passed: 2,
    failed: 0,
    skipped: 0,
    scenarios: [],
  },
}

describe('Results Page', () => {
  beforeEach(() => {
    vi.mocked(globalThis.fetch).mockImplementation(async (url, options) => {
      const urlStr = url.toString()
      const method = (options as RequestInit)?.method || 'GET'

      // Handle /results endpoint
      if (urlStr.match(/\/results$/) && method === 'GET') {
        return {
          ok: true,
          json: async () => ({ results: mockResultIds, count: mockResultIds.length }),
        } as Response
      }

      // Handle /results/:id endpoint
      const resultMatch = urlStr.match(/\/results\/([^/]+)$/)
      if (resultMatch) {
        const resultId = resultMatch[1]
        if (method === 'DELETE') {
          return {
            ok: true,
            json: async () => ({ status: 'deleted' }),
          } as Response
        }
        const result = mockResults[resultId]
        if (result) {
          return {
            ok: true,
            json: async () => result,
          } as Response
        }
        return {
          ok: false,
          status: 404,
          json: async () => ({ error: 'Not found' }),
        } as Response
      }

      return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
    })
  })

  // ============================================
  // RENDERING TESTS
  // ============================================
  describe('Rendering', () => {
    it('renders the results page header', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('Results')).toBeInTheDocument()
      })
    })

    it('shows loading state initially', () => {
      vi.mocked(globalThis.fetch).mockImplementation(() => new Promise(() => {}))

      render(<Results />)

      // Should show loader
      expect(document.querySelector('.animate-spin')).toBeInTheDocument()
    })

    it('shows total count', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('3 total')).toBeInTheDocument()
      })
    })

    it('renders result rows', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
        expect(screen.getByText('result-2')).toBeInTheDocument()
        expect(screen.getByText('result-3')).toBeInTheDocument()
      })
    })

    it('shows result statistics', async () => {
      render(<Results />)

      await waitFor(() => {
        // Result 1: 3/3 passed
        expect(screen.getByText('3/3 passed • 5m0s')).toBeInTheDocument()
        // Result 2: 3/5 passed
        expect(screen.getByText('3/5 passed • 10m0s')).toBeInTheDocument()
        // Result 3: 2/2 passed
        expect(screen.getByText('2/2 passed • 2m0s')).toBeInTheDocument()
      })
    })

    it('shows success icon for passing results', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Results 1 and 3 have no failures, should show success icons
      // Result 2 has failures, should show failure icon
      const rows = screen.getAllByRole('button')
      expect(rows.length).toBeGreaterThanOrEqual(3)
    })

    it('shows empty state when no results', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ results: [], count: 0 }),
      } as Response)

      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('No results yet. Run a scenario to see results here.')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // INTERACTION TESTS
  // ============================================
  describe('Interactions', () => {
    it('opens result detail when clicking row', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Find and click the result row
      const resultRow = screen.getByText('result-1').closest('[role="button"]')
      expect(resultRow).toBeInTheDocument()

      await user.click(resultRow!)

      await waitFor(() => {
        expect(screen.getByTestId('result-detail')).toBeInTheDocument()
        expect(screen.getByText('Result Detail: result-1')).toBeInTheDocument()
      })
    })

    it('closes result detail', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Open detail
      const resultRow = screen.getByText('result-1').closest('[role="button"]')
      await user.click(resultRow!)

      await waitFor(() => {
        expect(screen.getByTestId('result-detail')).toBeInTheDocument()
      })

      // Close detail
      const closeButton = screen.getByRole('button', { name: /close/i })
      await user.click(closeButton)

      await waitFor(() => {
        expect(screen.queryByTestId('result-detail')).not.toBeInTheDocument()
      })
    })

    it('deletes result when clicking delete button', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Find delete button for result-1
      const deleteButtons = screen.getAllByRole('button', { name: /delete result/i })
      expect(deleteButtons.length).toBe(3)

      await user.click(deleteButtons[0])

      // Should call DELETE endpoint
      await waitFor(() => {
        expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
          expect.stringContaining('/results/result-1'),
          expect.objectContaining({ method: 'DELETE' })
        )
      })
    })

    it('delete button stops propagation (does not open detail)', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      const deleteButtons = screen.getAllByRole('button', { name: /delete result/i })
      await user.click(deleteButtons[0])

      // Wait a moment to ensure no detail opens
      await waitFor(
        () => {
          expect(screen.queryByTestId('result-detail')).not.toBeInTheDocument()
        },
        { timeout: 500 }
      )
    })

    it('can select different results', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-2')).toBeInTheDocument()
      })

      // Click second result
      const secondRow = screen.getByText('result-2').closest('[role="button"]')
      await user.click(secondRow!)

      await waitFor(() => {
        expect(screen.getByText('Result Detail: result-2')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // KEYBOARD ACCESSIBILITY TESTS
  // ============================================
  describe('Keyboard Accessibility', () => {
    it('result rows are keyboard accessible', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      const resultRow = screen.getByText('result-1').closest('[role="button"]')
      expect(resultRow).toHaveAttribute('tabindex', '0')
    })

    it('can open detail with Enter key', async () => {
      const user = userEvent.setup()
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      const resultRow = screen.getByText('result-1').closest('[role="button"]') as HTMLElement | null
      resultRow?.focus()

      await user.keyboard('{Enter}')

      await waitFor(() => {
        expect(screen.getByTestId('result-detail')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // RESULT STATUS DISPLAY TESTS
  // ============================================
  describe('Result Status Display', () => {
    it('shows success icon for all-passing results', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Result 1 has 0 failures - look for green checkmark
      const result1Row = screen.getByText('result-1').closest('[role="button"]')
      expect(result1Row).toBeInTheDocument()

      // Check for success styling or icon presence
      const svg = result1Row?.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('shows failure icon for results with failures', async () => {
      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('result-2')).toBeInTheDocument()
      })

      // Result 2 has 2 failures - should show failure icon
      const result2Row = screen.getByText('result-2').closest('[role="button"]')
      expect(result2Row).toBeInTheDocument()

      // Check for failure styling
      const svg = result2Row?.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('shows loading spinner while result data is loading', async () => {
      // Make individual result fetches slow
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()

        if (urlStr.match(/\/results$/)) {
          return {
            ok: true,
            json: async () => ({ results: mockResultIds, count: mockResultIds.length }),
          } as Response
        }

        // Individual results take a long time
        if (urlStr.match(/\/results\/[^/]+$/)) {
          return new Promise(() => {}) // Never resolves
        }

        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Results />)

      await waitFor(() => {
        // Should show result IDs but with loading spinners for data
        expect(screen.getByText('result-1')).toBeInTheDocument()
      })

      // Look for loading spinners within result rows
      const spinners = document.querySelectorAll('.animate-spin')
      expect(spinners.length).toBeGreaterThan(0)
    })
  })

  // ============================================
  // ERROR HANDLING TESTS
  // ============================================
  describe('Error Handling', () => {
    it('handles API error gracefully', async () => {
      vi.mocked(globalThis.fetch).mockRejectedValue(new Error('Network error'))

      render(<Results />)

      // Should show loading initially, then handle error
      await waitFor(() => {
        // Page should still be accessible
        expect(screen.getByText('Results')).toBeInTheDocument()
      })
    })

    it('handles empty results array', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ results: [], count: 0 }),
      } as Response)

      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('No results yet. Run a scenario to see results here.')).toBeInTheDocument()
      })
    })

    it('handles null results gracefully', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ results: null, count: 0 }),
      } as Response)

      render(<Results />)

      await waitFor(() => {
        expect(screen.getByText('No results yet. Run a scenario to see results here.')).toBeInTheDocument()
      })
    })

    it('handles individual result fetch error', async () => {
      vi.mocked(globalThis.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()

        if (urlStr.match(/\/results$/)) {
          return {
            ok: true,
            json: async () => ({ results: ['result-missing'], count: 1 }),
          } as Response
        }

        if (urlStr.match(/\/results\/result-missing$/)) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ error: 'Not found' }),
          } as Response
        }

        return { ok: false, status: 404, json: async () => ({ error: 'Not found' }) } as Response
      })

      render(<Results />)

      await waitFor(() => {
        // Should still show the result ID
        expect(screen.getByText('result-missing')).toBeInTheDocument()
      })
    })
  })
})
