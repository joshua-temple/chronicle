import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Button } from './button'
import { createRef } from 'react'

describe('Button Component', () => {
  describe('Rendering', () => {
    it('renders with children', () => {
      render(<Button>Click me</Button>)

      expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument()
    })

    it('renders as a button element', () => {
      render(<Button>Test</Button>)

      expect(screen.getByRole('button')).toBeInTheDocument()
    })

    it('applies custom className', () => {
      render(<Button className="custom-class">Test</Button>)

      expect(screen.getByRole('button')).toHaveClass('custom-class')
    })

    it('forwards ref to button element', () => {
      const ref = createRef<HTMLButtonElement>()
      render(<Button ref={ref}>Test</Button>)

      expect(ref.current).toBeInstanceOf(HTMLButtonElement)
    })

    it('spreads additional props to button', () => {
      render(<Button data-testid="test-button" type="submit">Test</Button>)

      const button = screen.getByTestId('test-button')
      expect(button).toHaveAttribute('type', 'submit')
    })
  })

  describe('Variants', () => {
    it('applies default variant styles', () => {
      render(<Button variant="default">Default</Button>)

      expect(screen.getByRole('button')).toHaveClass('bg-primary')
    })

    it('applies destructive variant styles', () => {
      render(<Button variant="destructive">Destructive</Button>)

      expect(screen.getByRole('button')).toHaveClass('bg-destructive')
    })

    it('applies outline variant styles', () => {
      render(<Button variant="outline">Outline</Button>)

      expect(screen.getByRole('button')).toHaveClass('border')
    })

    it('applies secondary variant styles', () => {
      render(<Button variant="secondary">Secondary</Button>)

      expect(screen.getByRole('button')).toHaveClass('bg-secondary')
    })

    it('applies ghost variant styles', () => {
      render(<Button variant="ghost">Ghost</Button>)

      // Ghost has hover styles but no background
      expect(screen.getByRole('button')).not.toHaveClass('bg-primary')
      expect(screen.getByRole('button')).not.toHaveClass('bg-secondary')
    })

    it('applies link variant styles', () => {
      render(<Button variant="link">Link</Button>)

      expect(screen.getByRole('button')).toHaveClass('underline-offset-4')
    })

    it('defaults to default variant when not specified', () => {
      render(<Button>Default</Button>)

      expect(screen.getByRole('button')).toHaveClass('bg-primary')
    })
  })

  describe('Sizes', () => {
    it('applies default size', () => {
      render(<Button size="default">Default</Button>)

      expect(screen.getByRole('button')).toHaveClass('h-10')
    })

    it('applies small size', () => {
      render(<Button size="sm">Small</Button>)

      expect(screen.getByRole('button')).toHaveClass('h-9')
    })

    it('applies large size', () => {
      render(<Button size="lg">Large</Button>)

      expect(screen.getByRole('button')).toHaveClass('h-11')
    })

    it('applies icon size', () => {
      render(<Button size="icon">Icon</Button>)

      expect(screen.getByRole('button')).toHaveClass('h-10')
      expect(screen.getByRole('button')).toHaveClass('w-10')
    })

    it('defaults to default size when not specified', () => {
      render(<Button>Default</Button>)

      expect(screen.getByRole('button')).toHaveClass('h-10')
    })
  })

  describe('States', () => {
    it('can be disabled', () => {
      render(<Button disabled>Disabled</Button>)

      expect(screen.getByRole('button')).toBeDisabled()
      expect(screen.getByRole('button')).toHaveClass('disabled:opacity-50')
    })

    it('applies focus-visible ring styles', () => {
      render(<Button>Focus</Button>)

      expect(screen.getByRole('button')).toHaveClass('focus-visible:ring-2')
    })
  })

  describe('Interactions', () => {
    it('calls onClick when clicked', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()

      render(<Button onClick={handleClick}>Click me</Button>)

      await user.click(screen.getByRole('button'))

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('does not call onClick when disabled', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()

      render(<Button disabled onClick={handleClick}>Disabled</Button>)

      await user.click(screen.getByRole('button'))

      expect(handleClick).not.toHaveBeenCalled()
    })

    it('can be focused with Tab', async () => {
      const user = userEvent.setup()
      render(<Button>Focus me</Button>)

      await user.tab()

      expect(screen.getByRole('button')).toHaveFocus()
    })

    it('can be activated with Enter key', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()

      render(<Button onClick={handleClick}>Enter</Button>)

      screen.getByRole('button').focus()
      await user.keyboard('{Enter}')

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('can be activated with Space key', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()

      render(<Button onClick={handleClick}>Space</Button>)

      screen.getByRole('button').focus()
      await user.keyboard(' ')

      expect(handleClick).toHaveBeenCalledTimes(1)
    })
  })

  describe('Accessibility', () => {
    it('has displayName', () => {
      expect(Button.displayName).toBe('Button')
    })

    it('supports aria-label', () => {
      render(<Button aria-label="Custom label">Icon</Button>)

      expect(screen.getByRole('button', { name: /custom label/i })).toBeInTheDocument()
    })

    it('supports aria-disabled', () => {
      render(<Button aria-disabled="true">Aria disabled</Button>)

      expect(screen.getByRole('button')).toHaveAttribute('aria-disabled', 'true')
    })
  })
})
