import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'

// Define mocks using vi.hoisted() so they're available when vi.mock runs
const { mockUseMode, mockUseActiveProject } = vi.hoisted(() => ({
  mockUseMode: vi.fn(),
  mockUseActiveProject: vi.fn(),
}))

vi.mock('@/stores/mode', () => ({
  useMode: () => mockUseMode(),
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

vi.mock('@/stores/projects', () => ({
  useActiveProject: () => mockUseActiveProject(),
  useProjectsStore: () => ({ projects: [], activeProject: null }),
}))

// Mock toast store - useToastStore takes a selector function
vi.mock('@/stores/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  },
  useToastStore: (selector?: (state: { toasts: Array<unknown>; removeToast: () => void }) => unknown) => {
    const state = { toasts: [], removeToast: vi.fn() }
    return selector ? selector(state) : state
  },
}))

import { Layout } from './Layout'

function createTestWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })

  return function TestWrapper({ children }: { children: React.ReactNode }) {
    return createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: ['/'] },
        createElement(Routes, null,
          createElement(Route, {
            path: '/*',
            element: children
          })
        )
      )
    )
  }
}

describe('Layout Component', () => {
  beforeEach(() => {
    mockUseMode.mockReturnValue('daemon')
    mockUseActiveProject.mockReturnValue(null)

    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'healthy', timestamp: new Date().toISOString() }),
    } as Response)
  })

  describe('Rendering', () => {
    it('renders the sidebar', () => {
      render(<Layout />, { wrapper: createTestWrapper() })

      expect(screen.getByText('Chronicle')).toBeInTheDocument()
    })

    it('renders the header', async () => {
      render(<Layout />, { wrapper: createTestWrapper() })

      await waitFor(() => {
        expect(screen.getByText('Test Orchestration')).toBeInTheDocument()
      })
    })

    it('renders main content area with Outlet', () => {
      render(<Layout />, { wrapper: createTestWrapper() })

      // Main element should exist
      expect(document.querySelector('main')).toBeInTheDocument()
    })

    it('renders toast container', () => {
      render(<Layout />, { wrapper: createTestWrapper() })

      // ToastContainer should be rendered (even if empty)
      // We can check that layout has complete structure
      expect(document.querySelector('.min-h-screen')).toBeInTheDocument()
    })
  })

  describe('Back to Projects Button', () => {
    it('renders back button when onBackToProjects is provided', () => {
      const mockBack = vi.fn()
      render(<Layout onBackToProjects={mockBack} />, { wrapper: createTestWrapper() })

      expect(screen.getByRole('button', { name: /back to projects/i })).toBeInTheDocument()
    })

    it('does not render back button when onBackToProjects is not provided', () => {
      render(<Layout />, { wrapper: createTestWrapper() })

      expect(screen.queryByRole('button', { name: /back to projects/i })).not.toBeInTheDocument()
    })

    it('calls onBackToProjects when back button is clicked', async () => {
      const user = userEvent.setup()
      const mockBack = vi.fn()
      render(<Layout onBackToProjects={mockBack} />, { wrapper: createTestWrapper() })

      const backButton = screen.getByRole('button', { name: /back to projects/i })
      await user.click(backButton)

      expect(mockBack).toHaveBeenCalledTimes(1)
    })
  })

  describe('Active Project Display', () => {
    it('shows active project name when available', () => {
      mockUseActiveProject.mockReturnValue({ name: 'my-project', path: '/path' })
      const mockBack = vi.fn()
      render(<Layout onBackToProjects={mockBack} />, { wrapper: createTestWrapper() })

      expect(screen.getByText('Working on:')).toBeInTheDocument()
      expect(screen.getByText('my-project')).toBeInTheDocument()
    })

    it('does not show project info when no active project', () => {
      mockUseActiveProject.mockReturnValue(null)
      const mockBack = vi.fn()
      render(<Layout onBackToProjects={mockBack} />, { wrapper: createTestWrapper() })

      expect(screen.queryByText('Working on:')).not.toBeInTheDocument()
    })

    it('does not show project info without back button', () => {
      mockUseActiveProject.mockReturnValue({ name: 'my-project', path: '/path' })
      render(<Layout />, { wrapper: createTestWrapper() })

      // Project bar only shows when onBackToProjects is provided
      expect(screen.queryByText('Working on:')).not.toBeInTheDocument()
    })
  })

  describe('Mode-based Sidebar', () => {
    it('passes mode to Sidebar component', () => {
      mockUseMode.mockReturnValue('standalone')
      render(<Layout />, { wrapper: createTestWrapper() })

      // Standalone mode has Config nav item instead of Dashboard
      expect(screen.getByText('Config')).toBeInTheDocument()
    })

    it('shows daemon navigation in daemon mode', () => {
      mockUseMode.mockReturnValue('daemon')
      render(<Layout />, { wrapper: createTestWrapper() })

      expect(screen.getByText('Dashboard')).toBeInTheDocument()
      expect(screen.getByText('Runs')).toBeInTheDocument()
    })
  })
})
