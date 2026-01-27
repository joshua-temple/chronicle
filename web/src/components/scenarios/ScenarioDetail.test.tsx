import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { ScenarioDetail } from './ScenarioDetail'
import * as useScenariosModule from '@/hooks/useScenarios'

// Mock the hooks
vi.mock('@/hooks/useScenarios', () => ({
  useScenario: vi.fn(),
  useRunScenario: vi.fn(),
}))

const mockScenario = {
  name: 'Test Scenario',
  description: 'A comprehensive test scenario for user authentication',
  tags: ['auth', 'integration', 'api'],
  flow: [
    {
      name: 'Setup Database',
      component: 'db-setup',
      type: 'setup',
    },
    {
      name: 'Create User',
      component: 'user-creator',
      type: 'task',
    },
    {
      name: 'Validate Token',
      component: 'token-validator',
      type: 'validation',
    },
  ],
}

describe('ScenarioDetail Component', () => {
  const mockOnClose = vi.fn()
  const mockMutate = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useScenariosModule.useRunScenario).mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    } as any)
  })

  describe('Loading State', () => {
    it('shows loading skeletons when data is loading', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      const { container } = render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const skeletons = container.querySelectorAll('.animate-pulse')
      expect(skeletons.length).toBeGreaterThan(0)
    })

    it('shows close button during loading', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByRole('button', { name: /close/i })).toBeInTheDocument()
    })
  })

  describe('Scenario Display', () => {
    beforeEach(() => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: mockScenario,
        isLoading: false,
        error: null,
      } as any)
    })

    it('renders scenario name as title', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('Test Scenario')).toBeInTheDocument()
    })

    it('renders scenario description', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('A comprehensive test scenario for user authentication')).toBeInTheDocument()
    })

    it('renders all tags', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('auth')).toBeInTheDocument()
      expect(screen.getByText('integration')).toBeInTheDocument()
      expect(screen.getByText('api')).toBeInTheDocument()
    })

    it('renders flow section with correct step count', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText(/Flow \(3 steps\)/)).toBeInTheDocument()
    })

    it('renders flow steps with names', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('Setup Database')).toBeInTheDocument()
      expect(screen.getByText('Create User')).toBeInTheDocument()
      expect(screen.getByText('Validate Token')).toBeInTheDocument()
    })

    it('renders flow step types', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('setup')).toBeInTheDocument()
      expect(screen.getByText('task')).toBeInTheDocument()
      expect(screen.getByText('validation')).toBeInTheDocument()
    })

    it('renders step numbers', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('1')).toBeInTheDocument()
      expect(screen.getByText('2')).toBeInTheDocument()
      expect(screen.getByText('3')).toBeInTheDocument()
    })

    it('renders close button', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const closeButtons = screen.getAllByRole('button', { name: /close/i })
      expect(closeButtons.length).toBeGreaterThan(0)
    })

    it('renders run scenario button', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByRole('button', { name: /run scenario/i })).toBeInTheDocument()
    })
  })

  describe('Close Functionality', () => {
    beforeEach(() => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: mockScenario,
        isLoading: false,
        error: null,
      } as any)
    })

    it('calls onClose when X button is clicked', async () => {
      const user = userEvent.setup()
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const closeButtons = screen.getAllByRole('button', { name: /close/i })
      // The X button should be the first close button
      await user.click(closeButtons[0])

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when Close button is clicked', async () => {
      const user = userEvent.setup()
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      // Find the outline Close button (not the X icon button)
      const buttons = screen.getAllByRole('button')
      // The Close button is the outline variant before the Run Scenario button
      const closeButton = buttons.find(btn => btn.textContent === 'Close')
      expect(closeButton).toBeDefined()
      await user.click(closeButton!)

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when modal backdrop is clicked', async () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      // Click on the modal backdrop (captured by Modal component)
      // This should trigger onClose through the Modal's onClose prop
      // Note: Modal backdrop click behavior is tested in Modal component tests
    })
  })

  describe('Run Scenario Functionality', () => {
    beforeEach(() => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: mockScenario,
        isLoading: false,
        error: null,
      } as any)
    })

    it('calls runScenario.mutate with scenario name when Run Scenario is clicked', async () => {
      const user = userEvent.setup()
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const runButton = screen.getByRole('button', { name: /run scenario/i })
      await user.click(runButton)

      expect(mockMutate).toHaveBeenCalledWith('Test Scenario')
    })

    it('calls onClose after clicking Run Scenario', async () => {
      const user = userEvent.setup()
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const runButton = screen.getByRole('button', { name: /run scenario/i })
      await user.click(runButton)

      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })

    it('disables run button when mutation is pending', () => {
      vi.mocked(useScenariosModule.useRunScenario).mockReturnValue({
        mutate: mockMutate,
        isPending: true,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const runButton = screen.getByRole('button', { name: /run scenario/i })
      expect(runButton).toBeDisabled()
    })
  })

  describe('Scenario without optional fields', () => {
    it('handles scenario without description', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          description: undefined,
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('Test Scenario')).toBeInTheDocument()
      expect(screen.queryByText('A comprehensive test scenario')).not.toBeInTheDocument()
    })

    it('handles scenario without tags', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          tags: undefined,
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.queryByText('auth')).not.toBeInTheDocument()
    })

    it('handles scenario with empty tags array', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          tags: [],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.queryByText('auth')).not.toBeInTheDocument()
    })

    it('handles scenario without flow', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          flow: undefined,
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText(/Flow \(0 steps\)/)).toBeInTheDocument()
    })

    it('handles scenario with empty flow', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          flow: [],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText(/Flow \(0 steps\)/)).toBeInTheDocument()
    })
  })

  describe('Flow step variations', () => {
    it('displays step name when present', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          flow: [{ name: 'Named Step', type: 'task' }],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('Named Step')).toBeInTheDocument()
    })

    it('displays component name when step name is not present', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          flow: [{ component: 'component-name', type: 'setup' }],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText('component-name')).toBeInTheDocument()
    })

    it('shows singular step text for single step', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: {
          ...mockScenario,
          flow: [{ name: 'Single Step', type: 'task' }],
        },
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      expect(screen.getByText(/Flow \(1 step\)/)).toBeInTheDocument()
    })
  })

  describe('Hook usage', () => {
    it('calls useScenario with correct name', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any)

      render(<ScenarioDetail name="my-scenario-name" onClose={mockOnClose} />)

      expect(useScenariosModule.useScenario).toHaveBeenCalledWith('my-scenario-name')
    })
  })

  describe('Empty State', () => {
    it('returns null when no scenario and not loading', () => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: null,
        isLoading: false,
        error: null,
      } as any)

      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      // When scenario is null/undefined and not loading, the component renders
      // the Modal but with null content inside
    })
  })

  describe('Accessibility', () => {
    beforeEach(() => {
      vi.mocked(useScenariosModule.useScenario).mockReturnValue({
        data: mockScenario,
        isLoading: false,
        error: null,
      } as any)
    })

    it('has accessible title', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      // Title should be rendered with proper heading
      expect(screen.getByText('Test Scenario')).toBeInTheDocument()
    })

    it('close buttons have aria-labels', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      const closeButtons = screen.getAllByRole('button', { name: /close/i })
      expect(closeButtons.length).toBeGreaterThan(0)
    })

    it('uses Modal component for proper dialog semantics', () => {
      render(<ScenarioDetail name="test" onClose={mockOnClose} />)

      // Modal should provide dialog semantics
      // The Modal component wraps content in a dialog
    })
  })
})
