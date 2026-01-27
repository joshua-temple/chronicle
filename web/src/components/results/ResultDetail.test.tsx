import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { ResultDetail } from './ResultDetail'
import * as useResultsModule from '@/hooks/useResults'

// Mock the useResult hook
vi.mock('@/hooks/useResults', () => ({
  useResult: vi.fn(),
}))

const mockResult = {
  id: 'result-123',
  totalScenarios: 10,
  passed: 7,
  failed: 2,
  skipped: 1,
  duration: '2m 30s',
  scenarios: [
    {
      scenarioName: 'Test Scenario 1',
      state: 'completed',
      duration: '30s',
      error: null,
      flow_results: [
        {
          name: 'Setup Database',
          type: 'setup',
          state: 'completed',
          duration: '5s',
        },
        {
          name: 'Run Tests',
          type: 'task',
          state: 'completed',
          duration: '20s',
        },
      ],
    },
    {
      scenarioName: 'Test Scenario 2',
      state: 'failed',
      duration: '45s',
      error: 'Assertion failed: expected 200 but got 500',
      flow_results: [
        {
          name: 'Setup API',
          type: 'setup',
          state: 'completed',
          duration: '10s',
        },
        {
          name: 'Call Endpoint',
          type: 'task',
          state: 'failed',
          duration: '35s',
        },
      ],
    },
  ],
}

describe('ResultDetail Component', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Loading State', () => {
    it('shows loading spinner when data is loading', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      // Look for the loading spinner (SVG with animate-spin class)
      const spinner = container.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })

    it('shows loading spinner with animation', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const spinner = container.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    it('returns null when no result data and not loading', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(container.firstChild).toBeNull()
    })
  })

  describe('Result Display', () => {
    beforeEach(() => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: mockResult,
        isLoading: false,
        error: null,
      } as any)
    })

    it('renders result header', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Run Result')).toBeInTheDocument()
      expect(screen.getByText('result-123')).toBeInTheDocument()
    })

    it('renders summary statistics', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('10')).toBeInTheDocument() // Total
      expect(screen.getByText('7')).toBeInTheDocument() // Passed
      expect(screen.getByText('2')).toBeInTheDocument() // Failed
      expect(screen.getByText('1')).toBeInTheDocument() // Skipped
    })

    it('renders stat labels', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Total')).toBeInTheDocument()
      expect(screen.getByText('Passed')).toBeInTheDocument()
      expect(screen.getByText('Failed')).toBeInTheDocument()
      expect(screen.getByText('Skipped')).toBeInTheDocument()
    })

    it('renders duration', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText(/Duration: 2m 30s/)).toBeInTheDocument()
    })

    it('renders scenario results section', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Scenario Results')).toBeInTheDocument()
    })

    it('renders scenario names', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Test Scenario 1')).toBeInTheDocument()
      expect(screen.getByText('Test Scenario 2')).toBeInTheDocument()
    })

    it('renders scenario durations', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('30s')).toBeInTheDocument()
      expect(screen.getByText('45s')).toBeInTheDocument()
    })

    it('renders flow results', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Setup Database')).toBeInTheDocument()
      expect(screen.getByText('Run Tests')).toBeInTheDocument()
      expect(screen.getByText('Setup API')).toBeInTheDocument()
      expect(screen.getByText('Call Endpoint')).toBeInTheDocument()
    })

    it('renders flow type badges', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getAllByText('setup').length).toBe(2)
      expect(screen.getAllByText('task').length).toBe(2)
    })

    it('renders error message for failed scenario', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Assertion failed: expected 200 but got 500')).toBeInTheDocument()
    })

    it('shows check icon for completed scenarios', () => {
      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const checkIcons = container.querySelectorAll('.text-green-500')
      expect(checkIcons.length).toBeGreaterThan(0)
    })

    it('shows X icon for failed scenarios', () => {
      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const xIcons = container.querySelectorAll('.text-red-500')
      expect(xIcons.length).toBeGreaterThan(0)
    })
  })

  describe('Close Functionality', () => {
    beforeEach(() => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: mockResult,
        isLoading: false,
        error: null,
      } as any)
    })

    it('calls onClose when close button is clicked', async () => {
      const user = userEvent.setup()
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const closeButton = screen.getByRole('button', { name: /close/i })
      await user.click(closeButton)

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when clicking backdrop', async () => {
      const user = userEvent.setup()
      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      // Click the backdrop (the outer fixed div)
      const backdrop = container.querySelector('.fixed.inset-0')
      if (backdrop) {
        await user.click(backdrop)
      }

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('does not call onClose when clicking card content', async () => {
      const user = userEvent.setup()
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const cardContent = screen.getByText('Run Result')
      await user.click(cardContent)

      // onClose should NOT be called because we stopped propagation
      expect(mockOnClose).not.toHaveBeenCalled()
    })

    it('renders close button with aria-label', () => {
      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const closeButton = screen.getByRole('button', { name: /close/i })
      expect(closeButton).toBeInTheDocument()
    })
  })

  describe('Loading State Close', () => {
    it('calls onClose when clicking backdrop in loading state', async () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      const user = userEvent.setup()
      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const backdrop = container.querySelector('.fixed.inset-0')
      if (backdrop) {
        await user.click(backdrop)
      }

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('Result with no scenarios', () => {
    it('handles result with empty scenarios array', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: {
          ...mockResult,
          scenarios: [],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Scenario Results')).toBeInTheDocument()
    })

    it('handles result with undefined scenarios', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: {
          ...mockResult,
          scenarios: undefined,
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Scenario Results')).toBeInTheDocument()
    })
  })

  describe('Scenario with no flow results', () => {
    it('handles scenario with empty flow_results', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: {
          ...mockResult,
          scenarios: [
            {
              scenarioName: 'Simple Scenario',
              state: 'completed',
              duration: '10s',
              error: null,
              flow_results: [],
            },
          ],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Simple Scenario')).toBeInTheDocument()
    })

    it('handles scenario with undefined flow_results', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: {
          ...mockResult,
          scenarios: [
            {
              scenarioName: 'Simple Scenario',
              state: 'completed',
              duration: '10s',
              error: null,
              flow_results: undefined,
            },
          ],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      expect(screen.getByText('Simple Scenario')).toBeInTheDocument()
    })
  })

  describe('Hook usage', () => {
    it('calls useResult with correct id', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      render(<ResultDetail id="test-id-123" onClose={mockOnClose} />)

      expect(useResultsModule.useResult).toHaveBeenCalledWith('test-id-123')
    })
  })

  describe('Modal behavior', () => {
    it('renders as a modal overlay', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: mockResult,
        isLoading: false,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const overlay = container.querySelector('.fixed.inset-0.z-50')
      expect(overlay).toBeInTheDocument()
    })

    it('has semi-transparent backdrop', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: mockResult,
        isLoading: false,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const overlay = container.querySelector('.bg-black\\/50')
      expect(overlay).toBeInTheDocument()
    })

    it('card is scrollable', () => {
      vi.mocked(useResultsModule.useResult).mockReturnValue({
        data: mockResult,
        isLoading: false,
        error: null,
      } as any)

      const { container } = render(<ResultDetail id="result-123" onClose={mockOnClose} />)

      const card = container.querySelector('.overflow-auto')
      expect(card).toBeInTheDocument()
    })
  })
})
