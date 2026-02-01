import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ToastContainer } from './toast'

// Mock the toast store
const mockToasts: Array<{
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  description?: string
}> = []
const mockRemoveToast = vi.fn()

vi.mock('@/stores/toast', () => ({
  useToastStore: (selector: (state: { toasts: typeof mockToasts; removeToast: typeof mockRemoveToast }) => unknown) => {
    const state = { toasts: mockToasts, removeToast: mockRemoveToast }
    return selector(state)
  },
}))

describe('ToastContainer Component', () => {
  beforeEach(() => {
    mockToasts.length = 0
    mockRemoveToast.mockClear()
  })

  describe('Rendering', () => {
    it('renders nothing when no toasts', () => {
      render(<ToastContainer />)

      expect(screen.queryByRole('region')).not.toBeInTheDocument()
    })

    it('renders container when toasts exist', () => {
      mockToasts.push({
        id: 'toast-1',
        type: 'success',
        title: 'Success!',
      })

      render(<ToastContainer />)

      expect(screen.getByRole('region', { name: /notifications/i })).toBeInTheDocument()
    })

    it('renders toast title', () => {
      mockToasts.push({
        id: 'toast-1',
        type: 'info',
        title: 'Information',
      })

      render(<ToastContainer />)

      expect(screen.getByText('Information')).toBeInTheDocument()
    })

    it('renders toast description when provided', () => {
      mockToasts.push({
        id: 'toast-1',
        type: 'warning',
        title: 'Warning',
        description: 'Something needs attention',
      })

      render(<ToastContainer />)

      expect(screen.getByText('Something needs attention')).toBeInTheDocument()
    })

    it('does not render description when not provided', () => {
      mockToasts.push({
        id: 'toast-1',
        type: 'success',
        title: 'Success',
      })

      render(<ToastContainer />)

      // Only title should be present
      expect(screen.getByText('Success')).toBeInTheDocument()
      expect(screen.queryByText(/something/i)).not.toBeInTheDocument()
    })

    it('renders multiple toasts', () => {
      mockToasts.push(
        { id: 'toast-1', type: 'success', title: 'First Toast' },
        { id: 'toast-2', type: 'error', title: 'Second Toast' },
        { id: 'toast-3', type: 'info', title: 'Third Toast' }
      )

      render(<ToastContainer />)

      expect(screen.getByText('First Toast')).toBeInTheDocument()
      expect(screen.getByText('Second Toast')).toBeInTheDocument()
      expect(screen.getByText('Third Toast')).toBeInTheDocument()
    })
  })

  describe('Toast Types', () => {
    it('renders success toast with correct icon', () => {
      mockToasts.push({ id: 'toast-1', type: 'success', title: 'Success' })

      render(<ToastContainer />)

      // Check for green success icon
      const icon = document.querySelector('.text-green-500')
      expect(icon).toBeInTheDocument()
    })

    it('renders error toast with correct icon', () => {
      mockToasts.push({ id: 'toast-1', type: 'error', title: 'Error' })

      render(<ToastContainer />)

      const icon = document.querySelector('.text-red-500')
      expect(icon).toBeInTheDocument()
    })

    it('renders warning toast with correct icon', () => {
      mockToasts.push({ id: 'toast-1', type: 'warning', title: 'Warning' })

      render(<ToastContainer />)

      const icon = document.querySelector('.text-yellow-500')
      expect(icon).toBeInTheDocument()
    })

    it('renders info toast with correct icon', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Info' })

      render(<ToastContainer />)

      const icon = document.querySelector('.text-blue-500')
      expect(icon).toBeInTheDocument()
    })
  })

  describe('Toast Styling', () => {
    it('applies success styling', () => {
      mockToasts.push({ id: 'toast-1', type: 'success', title: 'Success' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('bg-green-500/10')
    })

    it('applies error styling', () => {
      mockToasts.push({ id: 'toast-1', type: 'error', title: 'Error' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('bg-red-500/10')
    })

    it('applies warning styling', () => {
      mockToasts.push({ id: 'toast-1', type: 'warning', title: 'Warning' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('bg-yellow-500/10')
    })

    it('applies info styling', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Info' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('bg-blue-500/10')
    })
  })

  describe('Interactions', () => {
    it('calls removeToast when dismiss button is clicked', async () => {
      const user = userEvent.setup()
      mockToasts.push({ id: 'toast-123', type: 'success', title: 'Dismissable' })

      render(<ToastContainer />)

      const dismissButton = screen.getByRole('button', { name: /dismiss notification/i })
      await user.click(dismissButton)

      expect(mockRemoveToast).toHaveBeenCalledWith('toast-123')
    })

    it('renders dismiss button for each toast', () => {
      mockToasts.push(
        { id: 'toast-1', type: 'success', title: 'First' },
        { id: 'toast-2', type: 'error', title: 'Second' }
      )

      render(<ToastContainer />)

      const dismissButtons = screen.getAllByRole('button', { name: /dismiss notification/i })
      expect(dismissButtons).toHaveLength(2)
    })
  })

  describe('Accessibility', () => {
    it('container has role="region"', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      expect(screen.getByRole('region')).toBeInTheDocument()
    })

    it('container has aria-label', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      expect(screen.getByRole('region', { name: /notifications/i })).toBeInTheDocument()
    })

    it('container has aria-live="polite"', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      expect(screen.getByRole('region')).toHaveAttribute('aria-live', 'polite')
    })

    it('each toast has role="alert"', () => {
      mockToasts.push(
        { id: 'toast-1', type: 'success', title: 'First' },
        { id: 'toast-2', type: 'error', title: 'Second' }
      )

      render(<ToastContainer />)

      const alerts = screen.getAllByRole('alert')
      expect(alerts).toHaveLength(2)
    })

    it('dismiss button has accessible label', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      expect(screen.getByRole('button', { name: /dismiss notification/i })).toBeInTheDocument()
    })
  })

  describe('Positioning', () => {
    it('is fixed positioned', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const container = screen.getByRole('region')
      expect(container).toHaveClass('fixed')
    })

    it('is positioned at bottom-right', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const container = screen.getByRole('region')
      expect(container).toHaveClass('bottom-4')
      expect(container).toHaveClass('right-4')
    })

    it('has high z-index', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const container = screen.getByRole('region')
      expect(container).toHaveClass('z-50')
    })
  })

  describe('Toast Layout', () => {
    it('toast has shadow', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('shadow-lg')
    })

    it('toast has rounded corners', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('rounded-lg')
    })

    it('toast has border', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('border')
    })

    it('toast has minimum width', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('min-w-[300px]')
    })

    it('toast has maximum width', () => {
      mockToasts.push({ id: 'toast-1', type: 'info', title: 'Test' })

      render(<ToastContainer />)

      const toast = screen.getByRole('alert')
      expect(toast).toHaveClass('max-w-[400px]')
    })
  })
})
