import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EmptyState } from './empty-state'

describe('EmptyState Component', () => {
  describe('Rendering', () => {
    it('renders with title', () => {
      render(<EmptyState title="No items found" />)

      expect(screen.getByRole('heading', { name: /no items found/i })).toBeInTheDocument()
    })

    it('renders with description when provided', () => {
      render(
        <EmptyState
          title="Empty"
          description="There are no items to display"
        />
      )

      expect(screen.getByText('There are no items to display')).toBeInTheDocument()
    })

    it('does not render description when not provided', () => {
      render(<EmptyState title="Empty" />)

      // Only the title should be present, no description paragraph
      expect(screen.queryByRole('paragraph')).not.toBeInTheDocument()
    })

    it('applies custom className', () => {
      render(<EmptyState title="Empty" className="custom-class" />)

      // Container should have the custom class
      expect(screen.getByRole('heading').parentElement).toHaveClass('custom-class')
    })
  })

  describe('Variants', () => {
    it('uses default variant icon', () => {
      render(<EmptyState title="Default" variant="default" />)

      // Default variant uses FolderOpen icon
      const icon = document.querySelector('svg')
      expect(icon).toBeInTheDocument()
    })

    it('uses search variant icon', () => {
      render(<EmptyState title="No results" variant="search" />)

      const icon = document.querySelector('svg')
      expect(icon).toBeInTheDocument()
    })

    it('uses error variant with destructive color', () => {
      render(<EmptyState title="Error occurred" variant="error" />)

      const iconContainer = document.querySelector('.text-destructive')
      expect(iconContainer).toBeInTheDocument()
    })

    it('uses empty variant icon', () => {
      render(<EmptyState title="Empty" variant="empty" />)

      const icon = document.querySelector('svg')
      expect(icon).toBeInTheDocument()
    })

    it('defaults to default variant when not specified', () => {
      render(<EmptyState title="Default" />)

      // Should have muted foreground color (not destructive)
      const iconContainer = document.querySelector('.text-muted-foreground')
      expect(iconContainer).toBeInTheDocument()
    })
  })

  describe('Custom Icon', () => {
    it('renders custom icon when provided', () => {
      render(
        <EmptyState
          title="Custom"
          icon={<svg data-testid="custom-icon" />}
        />
      )

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
    })

    it('custom icon overrides variant icon', () => {
      render(
        <EmptyState
          title="Custom"
          variant="error"
          icon={<div data-testid="custom-element">Custom Icon</div>}
        />
      )

      expect(screen.getByTestId('custom-element')).toBeInTheDocument()
    })
  })

  describe('Action Button', () => {
    it('renders action button when provided', () => {
      render(
        <EmptyState
          title="Empty"
          action={{ label: 'Add Item', onClick: () => {} }}
        />
      )

      expect(screen.getByRole('button', { name: /add item/i })).toBeInTheDocument()
    })

    it('does not render action button when not provided', () => {
      render(<EmptyState title="Empty" />)

      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })

    it('calls onClick when action button is clicked', async () => {
      const user = userEvent.setup()
      const handleClick = vi.fn()

      render(
        <EmptyState
          title="Empty"
          action={{ label: 'Refresh', onClick: handleClick }}
        />
      )

      await user.click(screen.getByRole('button', { name: /refresh/i }))

      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('action button has outline variant', () => {
      render(
        <EmptyState
          title="Empty"
          action={{ label: 'Action', onClick: () => {} }}
        />
      )

      const button = screen.getByRole('button', { name: /action/i })
      expect(button).toHaveClass('border')
    })
  })

  describe('Layout', () => {
    it('centers content', () => {
      render(<EmptyState title="Centered" />)

      const container = screen.getByRole('heading').parentElement
      expect(container).toHaveClass('flex')
      expect(container).toHaveClass('flex-col')
      expect(container).toHaveClass('items-center')
      expect(container).toHaveClass('justify-center')
    })

    it('has vertical padding', () => {
      render(<EmptyState title="Padded" />)

      const container = screen.getByRole('heading').parentElement
      expect(container).toHaveClass('py-12')
    })

    it('icon container has muted background', () => {
      render(<EmptyState title="Icon" />)

      const iconContainer = document.querySelector('.bg-muted')
      expect(iconContainer).toBeInTheDocument()
    })

    it('icon container is rounded', () => {
      render(<EmptyState title="Rounded" />)

      const iconContainer = document.querySelector('.rounded-full')
      expect(iconContainer).toBeInTheDocument()
    })

    it('description has max width and is centered', () => {
      render(
        <EmptyState
          title="Test"
          description="A long description that should be constrained"
        />
      )

      const description = screen.getByText(/long description/)
      expect(description).toHaveClass('max-w-sm')
      expect(description).toHaveClass('text-center')
    })
  })

  describe('Accessibility', () => {
    it('title is rendered as h3', () => {
      render(<EmptyState title="Accessible Title" />)

      expect(screen.getByRole('heading', { level: 3 })).toBeInTheDocument()
    })

    it('description has muted foreground for visual hierarchy', () => {
      render(
        <EmptyState
          title="Test"
          description="Secondary text"
        />
      )

      const description = screen.getByText('Secondary text')
      expect(description).toHaveClass('text-muted-foreground')
    })
  })

  describe('Full Example', () => {
    it('renders complete empty state', async () => {
      const user = userEvent.setup()
      const handleRefresh = vi.fn()

      render(
        <EmptyState
          variant="search"
          title="No results found"
          description="Try adjusting your search or filters to find what you're looking for."
          action={{ label: 'Clear filters', onClick: handleRefresh }}
          className="my-8"
        />
      )

      expect(screen.getByRole('heading', { name: /no results found/i })).toBeInTheDocument()
      expect(screen.getByText(/try adjusting/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /clear filters/i })).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: /clear filters/i }))
      expect(handleRefresh).toHaveBeenCalled()
    })
  })
})
