import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge } from './badge'
import { createRef } from 'react'

describe('Badge Component', () => {
  describe('Rendering', () => {
    it('renders with children', () => {
      render(<Badge>Status</Badge>)

      expect(screen.getByText('Status')).toBeInTheDocument()
    })

    it('renders as a div element', () => {
      render(<Badge data-testid="badge">Test</Badge>)

      expect(screen.getByTestId('badge').tagName).toBe('DIV')
    })

    it('applies custom className', () => {
      render(<Badge className="custom-class">Test</Badge>)

      expect(screen.getByText('Test')).toHaveClass('custom-class')
    })

    it('forwards ref to div element', () => {
      const ref = createRef<HTMLDivElement>()
      render(<Badge ref={ref}>Test</Badge>)

      expect(ref.current).toBeInstanceOf(HTMLDivElement)
    })

    it('spreads additional props', () => {
      render(<Badge data-testid="test-badge" aria-label="Status badge">Test</Badge>)

      const badge = screen.getByTestId('test-badge')
      expect(badge).toHaveAttribute('aria-label', 'Status badge')
    })
  })

  describe('Variants', () => {
    it('applies default variant styles', () => {
      render(<Badge variant="default">Default</Badge>)

      expect(screen.getByText('Default')).toHaveClass('bg-primary')
    })

    it('applies secondary variant styles', () => {
      render(<Badge variant="secondary">Secondary</Badge>)

      expect(screen.getByText('Secondary')).toHaveClass('bg-secondary')
    })

    it('applies destructive variant styles', () => {
      render(<Badge variant="destructive">Destructive</Badge>)

      expect(screen.getByText('Destructive')).toHaveClass('bg-destructive')
    })

    it('applies outline variant styles', () => {
      render(<Badge variant="outline">Outline</Badge>)

      expect(screen.getByText('Outline')).toHaveClass('text-foreground')
      // Outline doesn't have bg-* classes
      expect(screen.getByText('Outline')).not.toHaveClass('bg-primary')
    })

    it('applies success variant styles', () => {
      render(<Badge variant="success">Success</Badge>)

      expect(screen.getByText('Success')).toHaveClass('bg-green-500/20')
      expect(screen.getByText('Success')).toHaveClass('text-green-400')
    })

    it('defaults to default variant when not specified', () => {
      render(<Badge>Default</Badge>)

      expect(screen.getByText('Default')).toHaveClass('bg-primary')
    })
  })

  describe('Base Styles', () => {
    it('applies rounded-full class', () => {
      render(<Badge>Rounded</Badge>)

      expect(screen.getByText('Rounded')).toHaveClass('rounded-full')
    })

    it('applies inline-flex class', () => {
      render(<Badge>Flex</Badge>)

      expect(screen.getByText('Flex')).toHaveClass('inline-flex')
    })

    it('applies border class', () => {
      render(<Badge>Border</Badge>)

      expect(screen.getByText('Border')).toHaveClass('border')
    })

    it('applies text-xs class', () => {
      render(<Badge>Small</Badge>)

      expect(screen.getByText('Small')).toHaveClass('text-xs')
    })

    it('applies font-semibold class', () => {
      render(<Badge>Bold</Badge>)

      expect(screen.getByText('Bold')).toHaveClass('font-semibold')
    })

    it('applies padding classes', () => {
      render(<Badge>Padded</Badge>)

      expect(screen.getByText('Padded')).toHaveClass('px-2.5')
      expect(screen.getByText('Padded')).toHaveClass('py-0.5')
    })
  })

  describe('Accessibility', () => {
    it('has displayName', () => {
      expect(Badge.displayName).toBe('Badge')
    })

    it('supports role attribute', () => {
      render(<Badge role="status">Status</Badge>)

      expect(screen.getByRole('status')).toBeInTheDocument()
    })

    it('can contain interactive elements', () => {
      render(
        <Badge>
          <span>Active</span>
          <button>Close</button>
        </Badge>
      )

      expect(screen.getByRole('button', { name: /close/i })).toBeInTheDocument()
    })
  })

  describe('Complex Content', () => {
    it('renders with icon and text', () => {
      render(
        <Badge>
          <svg data-testid="icon" />
          Active
        </Badge>
      )

      expect(screen.getByTestId('icon')).toBeInTheDocument()
      expect(screen.getByText('Active')).toBeInTheDocument()
    })

    it('renders with multiple children', () => {
      render(
        <Badge>
          <span data-testid="child-1">Part 1</span>
          <span data-testid="child-2">Part 2</span>
        </Badge>
      )

      expect(screen.getByTestId('child-1')).toBeInTheDocument()
      expect(screen.getByTestId('child-2')).toBeInTheDocument()
    })
  })
})
