import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Skeleton, SkeletonCard, SkeletonList, SkeletonText } from './skeleton'

describe('Skeleton Components', () => {
  describe('Skeleton', () => {
    it('renders a div element', () => {
      render(<Skeleton data-testid="skeleton" />)
      const skeleton = screen.getByTestId('skeleton')
      expect(skeleton.tagName).toBe('DIV')
    })

    it('applies default classes', () => {
      render(<Skeleton data-testid="skeleton" />)
      const skeleton = screen.getByTestId('skeleton')
      expect(skeleton).toHaveClass('animate-pulse', 'rounded-md', 'bg-muted')
    })

    it('applies custom className', () => {
      render(<Skeleton className="h-4 w-full" data-testid="skeleton" />)
      const skeleton = screen.getByTestId('skeleton')
      expect(skeleton).toHaveClass('h-4', 'w-full')
    })

    it('spreads additional props', () => {
      render(<Skeleton data-testid="skeleton" aria-hidden="true" />)
      expect(screen.getByTestId('skeleton')).toHaveAttribute('aria-hidden', 'true')
    })

    it('can have custom dimensions', () => {
      render(<Skeleton className="h-10 w-32" data-testid="skeleton" />)
      const skeleton = screen.getByTestId('skeleton')
      expect(skeleton).toHaveClass('h-10', 'w-32')
    })
  })

  describe('SkeletonCard', () => {
    it('renders card structure', () => {
      render(<SkeletonCard />)
      const card = document.querySelector('.rounded-lg.border')
      expect(card).toBeInTheDocument()
    })

    it('contains multiple skeletons', () => {
      const { container } = render(<SkeletonCard />)
      const skeletons = container.querySelectorAll('.animate-pulse')
      expect(skeletons.length).toBeGreaterThan(0)
    })

    it('has card styling', () => {
      const { container } = render(<SkeletonCard />)
      const card = container.firstChild as HTMLElement
      expect(card).toHaveClass('rounded-lg', 'border', 'p-4', 'shadow-sm')
    })

    it('renders title skeleton', () => {
      const { container } = render(<SkeletonCard />)
      const titleSkeleton = container.querySelector('.h-5.w-2\\/3')
      expect(titleSkeleton).toBeInTheDocument()
    })

    it('renders description skeleton', () => {
      const { container } = render(<SkeletonCard />)
      const descSkeleton = container.querySelector('.h-4.w-full')
      expect(descSkeleton).toBeInTheDocument()
    })

    it('renders tag skeletons', () => {
      const { container } = render(<SkeletonCard />)
      const tagSkeletons = container.querySelectorAll('.rounded-full')
      expect(tagSkeletons.length).toBe(2)
    })
  })

  describe('SkeletonList', () => {
    it('renders default 6 cards', () => {
      const { container } = render(<SkeletonList />)
      const cards = container.querySelectorAll('.rounded-lg.border')
      expect(cards.length).toBe(6)
    })

    it('renders custom count of cards', () => {
      const { container } = render(<SkeletonList count={3} />)
      const cards = container.querySelectorAll('.rounded-lg.border')
      expect(cards.length).toBe(3)
    })

    it('renders grid layout', () => {
      const { container } = render(<SkeletonList />)
      const grid = container.firstChild as HTMLElement
      expect(grid).toHaveClass('grid', 'gap-4')
    })

    it('has responsive columns', () => {
      const { container } = render(<SkeletonList />)
      const grid = container.firstChild as HTMLElement
      expect(grid).toHaveClass('md:grid-cols-2', 'lg:grid-cols-3')
    })

    it('renders 0 cards when count is 0', () => {
      const { container } = render(<SkeletonList count={0} />)
      const cards = container.querySelectorAll('.rounded-lg.border')
      expect(cards.length).toBe(0)
    })

    it('renders 10 cards when count is 10', () => {
      const { container } = render(<SkeletonList count={10} />)
      const cards = container.querySelectorAll('.rounded-lg.border')
      expect(cards.length).toBe(10)
    })
  })

  describe('SkeletonText', () => {
    it('renders default 3 lines', () => {
      const { container } = render(<SkeletonText />)
      const lines = container.querySelectorAll('.animate-pulse')
      expect(lines.length).toBe(3)
    })

    it('renders custom number of lines', () => {
      const { container } = render(<SkeletonText lines={5} />)
      const lines = container.querySelectorAll('.animate-pulse')
      expect(lines.length).toBe(5)
    })

    it('has vertical spacing', () => {
      const { container } = render(<SkeletonText />)
      const wrapper = container.firstChild as HTMLElement
      expect(wrapper).toHaveClass('space-y-2')
    })

    it('makes last line shorter', () => {
      const { container } = render(<SkeletonText lines={3} />)
      const lines = container.querySelectorAll('.animate-pulse')
      // Last line should have w-3/4 class
      expect(lines[2]).toHaveClass('w-3/4')
    })

    it('makes non-last lines full width', () => {
      const { container } = render(<SkeletonText lines={3} />)
      const lines = container.querySelectorAll('.animate-pulse')
      // First two lines should have w-full class
      expect(lines[0]).toHaveClass('w-full')
      expect(lines[1]).toHaveClass('w-full')
    })

    it('renders lines with height', () => {
      const { container } = render(<SkeletonText />)
      const lines = container.querySelectorAll('.animate-pulse')
      lines.forEach((line) => {
        expect(line).toHaveClass('h-4')
      })
    })

    it('renders 1 line correctly', () => {
      const { container } = render(<SkeletonText lines={1} />)
      const lines = container.querySelectorAll('.animate-pulse')
      expect(lines.length).toBe(1)
      // Single line should be shorter (w-3/4) since it's the last
      expect(lines[0]).toHaveClass('w-3/4')
    })
  })

  describe('Animation', () => {
    it('all skeletons have animate-pulse class', () => {
      const { container } = render(
        <>
          <Skeleton data-testid="single" />
          <SkeletonCard />
          <SkeletonText lines={2} />
        </>
      )
      const allSkeletons = container.querySelectorAll('.animate-pulse')
      allSkeletons.forEach((skeleton) => {
        expect(skeleton).toHaveClass('animate-pulse')
      })
    })
  })

  describe('Accessibility', () => {
    it('skeletons can be marked as decorative', () => {
      render(<Skeleton aria-hidden="true" data-testid="skeleton" />)
      expect(screen.getByTestId('skeleton')).toHaveAttribute('aria-hidden', 'true')
    })

    it('can have accessible label for loading states', () => {
      render(
        <div role="status" aria-label="Loading">
          <Skeleton data-testid="skeleton" />
        </div>
      )
      expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Loading')
    })
  })
})
