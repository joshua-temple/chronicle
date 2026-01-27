import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Modal } from './modal'

describe('Modal Component', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('Rendering', () => {
    it('renders when open is true', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>Modal content</div>
        </Modal>
      )

      expect(screen.getByText('Modal content')).toBeInTheDocument()
    })

    it('does not render when open is false', () => {
      render(
        <Modal open={false} onClose={() => {}}>
          <div>Modal content</div>
        </Modal>
      )

      expect(screen.queryByText('Modal content')).not.toBeInTheDocument()
    })

    it('renders title when provided', () => {
      render(
        <Modal open={true} onClose={() => {}} title="Test Title">
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('heading', { name: /test title/i })).toBeInTheDocument()
    })

    it('renders close button when title is provided', () => {
      render(
        <Modal open={true} onClose={() => {}} title="Test">
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('button', { name: /close dialog/i })).toBeInTheDocument()
    })

    it('does not render close button when no title', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>Content only</div>
        </Modal>
      )

      expect(screen.queryByRole('button', { name: /close dialog/i })).not.toBeInTheDocument()
    })

    it('applies custom className', () => {
      render(
        <Modal open={true} onClose={() => {}} className="custom-modal">
          <div>Content</div>
        </Modal>
      )

      // The Card inside should have the custom class
      const modal = screen.getByRole('dialog')
      expect(modal.querySelector('.custom-modal')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('has role="dialog"', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    it('has aria-modal="true"', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    })

    it('applies aria-labelledby when titleId is provided', () => {
      render(
        <Modal open={true} onClose={() => {}} title="Test" titleId="modal-title">
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('dialog')).toHaveAttribute('aria-labelledby', 'modal-title')
    })

    it('title has the correct id', () => {
      render(
        <Modal open={true} onClose={() => {}} title="Test Title" titleId="my-title">
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByRole('heading', { name: /test title/i })).toHaveAttribute('id', 'my-title')
    })
  })

  describe('Interactions', () => {
    it('calls onClose when close button is clicked', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const handleClose = vi.fn()

      render(
        <Modal open={true} onClose={handleClose} title="Test">
          <div>Content</div>
        </Modal>
      )

      await user.click(screen.getByRole('button', { name: /close dialog/i }))

      expect(handleClose).toHaveBeenCalledTimes(1)
    })

    it('calls onClose when clicking backdrop', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const handleClose = vi.fn()

      render(
        <Modal open={true} onClose={handleClose}>
          <div>Content</div>
        </Modal>
      )

      const backdrop = screen.getByRole('dialog')
      await user.click(backdrop)

      expect(handleClose).toHaveBeenCalledTimes(1)
    })

    it('does not call onClose when clicking modal content', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const handleClose = vi.fn()

      render(
        <Modal open={true} onClose={handleClose}>
          <div data-testid="modal-content">Content</div>
        </Modal>
      )

      await user.click(screen.getByTestId('modal-content'))

      expect(handleClose).not.toHaveBeenCalled()
    })

    it('calls onClose when Escape key is pressed', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
      const handleClose = vi.fn()

      render(
        <Modal open={true} onClose={handleClose}>
          <div>Content</div>
        </Modal>
      )

      await vi.runAllTimersAsync()
      await user.keyboard('{Escape}')

      expect(handleClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('Focus Management', () => {
    it('locks body scroll when open', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(document.body.style.overflow).toBe('hidden')
    })

    it('restores body scroll when closed', () => {
      const { rerender } = render(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      rerender(
        <Modal open={false} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(document.body.style.overflow).not.toBe('hidden')
    })

    it('focuses first focusable element on open', async () => {
      render(
        <Modal open={true} onClose={() => {}} title="Test">
          <div>
            <button>First button</button>
            <button>Second button</button>
          </div>
        </Modal>
      )

      await vi.runAllTimersAsync()

      // The close button should be focused first (it comes before content)
      expect(screen.getByRole('button', { name: /close dialog/i })).toHaveFocus()
    })
  })

  describe('Focus Trap', () => {
    it('traps focus within modal', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

      render(
        <Modal open={true} onClose={() => {}} title="Test">
          <div>
            <button data-testid="btn-1">Button 1</button>
            <button data-testid="btn-2">Button 2</button>
          </div>
        </Modal>
      )

      await vi.runAllTimersAsync()

      // Tab through all focusable elements
      const closeBtn = screen.getByRole('button', { name: /close dialog/i })
      const btn1 = screen.getByTestId('btn-1')
      const btn2 = screen.getByTestId('btn-2')

      expect(closeBtn).toHaveFocus()

      await user.tab()
      expect(btn1).toHaveFocus()

      await user.tab()
      expect(btn2).toHaveFocus()

      // Tab from last should wrap to first
      await user.tab()
      expect(closeBtn).toHaveFocus()
    })

    it('wraps focus backward with Shift+Tab', async () => {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })

      render(
        <Modal open={true} onClose={() => {}} title="Test">
          <div>
            <button data-testid="btn-1">Button 1</button>
          </div>
        </Modal>
      )

      await vi.runAllTimersAsync()

      const closeBtn = screen.getByRole('button', { name: /close dialog/i })
      const btn1 = screen.getByTestId('btn-1')

      expect(closeBtn).toHaveFocus()

      // Shift+Tab from first should wrap to last
      await user.tab({ shift: true })
      expect(btn1).toHaveFocus()
    })
  })

  describe('Opening and Closing', () => {
    it('transitions from closed to open', () => {
      const { rerender } = render(
        <Modal open={false} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.queryByText('Content')).not.toBeInTheDocument()

      rerender(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByText('Content')).toBeInTheDocument()
    })

    it('transitions from open to closed', () => {
      const { rerender } = render(
        <Modal open={true} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.getByText('Content')).toBeInTheDocument()

      rerender(
        <Modal open={false} onClose={() => {}}>
          <div>Content</div>
        </Modal>
      )

      expect(screen.queryByText('Content')).not.toBeInTheDocument()
    })
  })

  describe('Complex Content', () => {
    it('renders form inside modal', () => {
      render(
        <Modal open={true} onClose={() => {}} title="Form Modal">
          <form>
            <input type="text" placeholder="Name" />
            <button type="submit">Submit</button>
          </form>
        </Modal>
      )

      expect(screen.getByPlaceholderText('Name')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /submit/i })).toBeInTheDocument()
    })

    it('renders nested content', () => {
      render(
        <Modal open={true} onClose={() => {}}>
          <div>
            <h2>Nested Title</h2>
            <p>Paragraph text</p>
            <ul>
              <li>Item 1</li>
              <li>Item 2</li>
            </ul>
          </div>
        </Modal>
      )

      expect(screen.getByRole('heading', { name: /nested title/i })).toBeInTheDocument()
      expect(screen.getByText('Paragraph text')).toBeInTheDocument()
      expect(screen.getByText('Item 1')).toBeInTheDocument()
    })
  })
})
