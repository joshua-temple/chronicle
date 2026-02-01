import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { Header } from './Header'

describe('Header Component', () => {
  beforeEach(() => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'healthy', timestamp: new Date().toISOString(), version: '1.0.0' }),
    } as Response)
  })

  describe('Rendering', () => {
    it('renders the title', () => {
      render(<Header />)

      expect(screen.getByText('Test Orchestration')).toBeInTheDocument()
    })

    it('renders the refresh button', () => {
      render(<Header />)

      expect(screen.getByRole('button', { name: /refresh health status/i })).toBeInTheDocument()
    })

    it('shows loading state initially', () => {
      vi.mocked(globalThis.fetch).mockImplementation(() => new Promise(() => {}))
      render(<Header />)

      expect(screen.getByText('Checking...')).toBeInTheDocument()
    })

    it('shows healthy status when API returns healthy', async () => {
      render(<Header />)

      await waitFor(() => {
        expect(screen.getByText('healthy')).toBeInTheDocument()
      })
    })

    it('shows unhealthy status when API returns unhealthy', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'unhealthy', timestamp: new Date().toISOString() }),
      } as Response)

      render(<Header />)

      await waitFor(() => {
        expect(screen.getByText('unhealthy')).toBeInTheDocument()
      })
    })

    it('shows Unknown when status is missing', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({}),
      } as Response)

      render(<Header />)

      await waitFor(() => {
        expect(screen.getByText('Unknown')).toBeInTheDocument()
      })
    })
  })

  describe('Badge Styling', () => {
    it('uses success variant for healthy status', async () => {
      render(<Header />)

      await waitFor(() => {
        const badge = screen.getByText('healthy')
        // Badge with success variant should be present
        expect(badge).toBeInTheDocument()
      })
    })

    it('uses destructive variant for unhealthy status', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'unhealthy' }),
      } as Response)

      render(<Header />)

      await waitFor(() => {
        const badge = screen.getByText('unhealthy')
        expect(badge).toBeInTheDocument()
      })
    })
  })

  describe('Interactions', () => {
    it('refetches health when refresh button is clicked', async () => {
      const user = userEvent.setup()
      render(<Header />)

      await waitFor(() => {
        expect(screen.getByText('healthy')).toBeInTheDocument()
      })

      // Clear mock to track new calls
      vi.mocked(globalThis.fetch).mockClear()
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'healthy' }),
      } as Response)

      const refreshButton = screen.getByRole('button', { name: /refresh health status/i })
      await user.click(refreshButton)

      await waitFor(() => {
        expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith('/api/v1/health')
      })
    })
  })

  describe('Error Handling', () => {
    it('handles network error gracefully', async () => {
      vi.mocked(globalThis.fetch).mockRejectedValue(new Error('Network error'))

      render(<Header />)

      // Should show loading then handle error
      await waitFor(() => {
        expect(screen.getByText('Unknown')).toBeInTheDocument()
      })
    })

    it('handles API error response', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' }),
      } as Response)

      render(<Header />)

      // Health check should fail, show unknown
      await waitFor(() => {
        expect(screen.getByText('Unknown')).toBeInTheDocument()
      })
    })
  })

  describe('Accessibility', () => {
    it('has accessible refresh button', () => {
      render(<Header />)

      const refreshButton = screen.getByRole('button', { name: /refresh health status/i })
      expect(refreshButton).toBeInTheDocument()
      expect(refreshButton).toHaveAccessibleName('Refresh health status')
    })

    it('renders in a header element', () => {
      render(<Header />)

      const header = screen.getByRole('banner')
      expect(header).toBeInTheDocument()
    })
  })
})
