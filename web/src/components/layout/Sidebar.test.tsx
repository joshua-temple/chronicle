import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar } from './Sidebar'

// Mock the stores
vi.mock('@/stores/projects', () => ({
  useProjectsStore: vi.fn((selector) => {
    const state = {
      projects: [],
      discovered: [],
      loading: false,
      discovering: false,
      loadProjects: vi.fn(),
      runDiscovery: vi.fn(),
      connectAll: vi.fn(),
      addDiscovered: vi.fn(),
      dismissDiscovered: vi.fn(),
    }
    return selector(state)
  }),
}))

vi.mock('@/stores/settings', () => ({
  useSidebarCollapsed: vi.fn(() => false),
}))

function renderSidebar() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Sidebar />
    </MemoryRouter>
  )
}

describe('Sidebar Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('renders the Chronicle branding', () => {
      renderSidebar()

      expect(screen.getByText('Chronicle')).toBeInTheDocument()
    })

    it('renders as an aside element', () => {
      renderSidebar()

      expect(screen.getByRole('complementary')).toBeInTheDocument()
    })

    it('renders navigation element', () => {
      renderSidebar()

      expect(screen.getByRole('navigation')).toBeInTheDocument()
    })
  })

  describe('Global Navigation', () => {
    it('shows Dashboard link', () => {
      renderSidebar()

      expect(screen.getByRole('link', { name: /dashboard/i })).toBeInTheDocument()
    })

    it('shows UI Settings link', () => {
      renderSidebar()

      expect(screen.getByRole('link', { name: /ui settings/i })).toBeInTheDocument()
    })

    it('has correct href for Dashboard', () => {
      renderSidebar()

      expect(screen.getByRole('link', { name: /dashboard/i })).toHaveAttribute('href', '/')
    })

    it('has correct href for Settings', () => {
      renderSidebar()

      expect(screen.getByRole('link', { name: /ui settings/i })).toHaveAttribute('href', '/settings')
    })
  })

  describe('Footer Actions', () => {
    it('shows Discover button', () => {
      renderSidebar()

      expect(screen.getByRole('button', { name: /discover/i })).toBeInTheDocument()
    })

    it('shows Add Project link', () => {
      renderSidebar()

      expect(screen.getByRole('link', { name: /add project/i })).toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    it('shows "No projects yet" when no projects', () => {
      renderSidebar()

      expect(screen.getByText('No projects yet')).toBeInTheDocument()
    })
  })

  describe('Section Labels', () => {
    it('shows Overview section label', () => {
      renderSidebar()

      expect(screen.getByText('Overview')).toBeInTheDocument()
    })

    it('shows Projects section label', () => {
      renderSidebar()

      expect(screen.getByText('Projects')).toBeInTheDocument()
    })

    it('shows Settings section label', () => {
      renderSidebar()

      expect(screen.getByText('Settings')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('links are keyboard accessible', () => {
      renderSidebar()

      const links = screen.getAllByRole('link')
      links.forEach(link => {
        expect(link).not.toHaveAttribute('tabindex', '-1')
      })
    })
  })
})
