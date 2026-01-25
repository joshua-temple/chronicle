import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/utils'
import userEvent from '@testing-library/user-event'
import { ConfigEditor } from './ConfigEditor'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'standalone',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

const mockProject = {
  directory: '/path/to/project',
  config_file: 'chronicle.yaml',
  config_exists: true,
  last_modified: '2026-01-25T10:00:00Z',
}

const mockConfig = {
  name: 'test-project',
  version: '1.0',
  scenarios: [
    {
      name: 'happy_path',
      description: 'Happy path test',
      tags: ['smoke'],
      flow: [{ task: 'DoThing' }],
    },
    {
      name: 'error_case',
      description: 'Error handling test',
      tags: ['error'],
      flow: [{ task: 'FailThing' }],
    },
  ],
  infrastructure: {
    postgres: { provider: 'testcontainers', image: 'postgres:15' },
    redis: { provider: 'testcontainers', image: 'redis:7' },
  },
  chaos_profiles: {
    network_chaos: { name: 'Network Chaos Profile' },
  },
  mock_profiles: {
    happy_mocks: { name: 'Happy Path Mocks' },
  },
}

describe('ConfigEditor Page', () => {
  beforeEach(() => {
    vi.mocked(global.fetch).mockImplementation(async (url) => {
      const urlStr = url.toString()
      if (urlStr.includes('/api/local/project')) {
        return { ok: true, json: async () => mockProject } as Response
      }
      if (urlStr.includes('/api/local/config')) {
        return { ok: true, json: async () => mockConfig } as Response
      }
      if (urlStr.includes('/api/local/validate')) {
        return { ok: true, json: async () => ({ valid: true, errors: [] }) } as Response
      }
      if (urlStr.includes('/api/local/save')) {
        return { ok: true, json: async () => ({ success: true }) } as Response
      }
      return { ok: false, status: 404 } as Response
    })
  })

  // ============================================
  // RENDERING TESTS
  // ============================================
  describe('Rendering', () => {
    it('renders config editor with tabs', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Check tabs are present (they're buttons)
      expect(screen.getByRole('tab', { name: /general/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /scenarios/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /infrastructure/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /chaos/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /mocks/i })).toBeInTheDocument()
    })

    it('displays version in general tab', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('1.0')).toBeInTheDocument()
      })
    })

    it('shows scenarios in scenarios tab', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Click scenarios tab
      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        expect(screen.getByText('happy_path')).toBeInTheDocument()
      })

      expect(screen.getByText('error_case')).toBeInTheDocument()
    })

    it('shows infrastructure in infrastructure tab', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Click infrastructure tab
      await user.click(screen.getByRole('tab', { name: /infrastructure/i }))

      await waitFor(() => {
        expect(screen.getByText('postgres')).toBeInTheDocument()
      })

      expect(screen.getByText('redis')).toBeInTheDocument()
    })

    it('shows chaos profiles in chaos tab', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Click chaos tab
      await user.click(screen.getByRole('tab', { name: /chaos/i }))

      await waitFor(() => {
        expect(screen.getByText('network_chaos')).toBeInTheDocument()
      })
    })

    it('shows mock profiles in mocks tab', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Click mocks tab
      await user.click(screen.getByRole('tab', { name: /mocks/i }))

      await waitFor(() => {
        expect(screen.getByText('happy_mocks')).toBeInTheDocument()
      })
    })

    it('shows read-only badge', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Read-only')).toBeInTheDocument()
      })
    })

    it('shows loading state initially', () => {
      vi.mocked(global.fetch).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      )

      render(<ConfigEditor />)

      expect(document.querySelector('.animate-pulse')).toBeInTheDocument()
    })

    it('shows error when config not found', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return {
            ok: true,
            json: async () => ({ ...mockProject, config_exists: false }),
          } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ error: 'config file not found' }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('shows empty infrastructure message when none configured', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return { ok: true, json: async () => ({ ...mockConfig, infrastructure: {} }) } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /infrastructure/i }))

      await waitFor(() => {
        expect(screen.getByText(/no infrastructure providers configured/i)).toBeInTheDocument()
      })
    })

    it('shows empty chaos profiles message when none configured', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return { ok: true, json: async () => ({ ...mockConfig, chaos_profiles: {} }) } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /chaos/i }))

      await waitFor(() => {
        expect(screen.getByText(/no chaos profiles configured/i)).toBeInTheDocument()
      })
    })

    it('shows empty mock profiles message when none configured', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return { ok: true, json: async () => ({ ...mockConfig, mock_profiles: {} }) } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /mocks/i }))

      await waitFor(() => {
        expect(screen.getByText(/no mock profiles configured/i)).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // INTERACTION TESTS - Tab Navigation
  // ============================================
  describe('Tab Navigation', () => {
    it('switches to scenarios tab when clicked', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      const scenariosTab = screen.getByRole('tab', { name: /scenarios/i })
      await user.click(scenariosTab)

      // Should show scenarios content
      await waitFor(() => {
        expect(screen.getByText('happy_path')).toBeInTheDocument()
      })
    })

    it('switches to infrastructure tab when clicked', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      const infraTab = screen.getByRole('tab', { name: /infrastructure/i })
      await user.click(infraTab)

      await waitFor(() => {
        expect(screen.getByText('postgres')).toBeInTheDocument()
      })
    })

    it('switches to chaos tab when clicked', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      const chaosTab = screen.getByRole('tab', { name: /chaos/i })
      await user.click(chaosTab)

      await waitFor(() => {
        expect(screen.getByText('network_chaos')).toBeInTheDocument()
      })
    })

    it('switches to mocks tab when clicked', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      const mocksTab = screen.getByRole('tab', { name: /mocks/i })
      await user.click(mocksTab)

      await waitFor(() => {
        expect(screen.getByText('happy_mocks')).toBeInTheDocument()
      })
    })

    it('returns to general tab from other tabs', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Go to scenarios
      await user.click(screen.getByRole('tab', { name: /scenarios/i }))
      await waitFor(() => {
        expect(screen.getByText('happy_path')).toBeInTheDocument()
      })

      // Return to general
      await user.click(screen.getByRole('tab', { name: /general/i }))
      await waitFor(() => {
        expect(screen.getByText('1.0')).toBeInTheDocument()
      })
    })

    it('can navigate through all tabs sequentially', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // General tab (default)
      expect(screen.getByText('1.0')).toBeInTheDocument()

      // Scenarios
      await user.click(screen.getByRole('tab', { name: /scenarios/i }))
      await waitFor(() => {
        expect(screen.getByText('happy_path')).toBeInTheDocument()
      })

      // Infrastructure
      await user.click(screen.getByRole('tab', { name: /infrastructure/i }))
      await waitFor(() => {
        expect(screen.getByText('postgres')).toBeInTheDocument()
      })

      // Chaos
      await user.click(screen.getByRole('tab', { name: /chaos/i }))
      await waitFor(() => {
        expect(screen.getByText('network_chaos')).toBeInTheDocument()
      })

      // Mocks
      await user.click(screen.getByRole('tab', { name: /mocks/i }))
      await waitFor(() => {
        expect(screen.getByText('happy_mocks')).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // WORKFLOW TESTS - Configuration Display
  // ============================================
  describe('Configuration Display', () => {
    it('shows read-only indicator', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
        expect(screen.getByText('Read-only')).toBeInTheDocument()
      })
    })

    it('shows scenario step counts', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        // Step count format is "N step(s)" - singular or plural depending on count
        // Multiple scenarios have step counts, so use getAllByText
        const stepCounts = screen.getAllByText(/\d+ steps?/)
        expect(stepCounts.length).toBeGreaterThan(0)
      })
    })

    it('shows scenario tags in scenarios tab', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        expect(screen.getByText(/smoke/)).toBeInTheDocument()
      })
    })
  })

  // ============================================
  // ACCESSIBILITY TESTS
  // ============================================
  describe('Accessibility', () => {
    it('tabs have proper ARIA roles', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Check tablist exists
      expect(screen.getByRole('tablist')).toBeInTheDocument()

      // All tabs should have role="tab"
      const tabs = screen.getAllByRole('tab')
      expect(tabs.length).toBe(5) // general, scenarios, infrastructure, chaos, mocks
    })

    it('error state has accessible error message', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: false,
            status: 500,
            json: async () => ({ error: 'Server error' }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )

      // Error message should be visible
      const errorElement = screen.getByText(/failed to load configuration/i)
      expect(errorElement).toBeInTheDocument()
    })

    it('loading state shows skeleton', () => {
      vi.mocked(global.fetch).mockImplementation(
        () => new Promise(() => {})
      )

      render(<ConfigEditor />)

      const skeleton = document.querySelector('.animate-pulse')
      expect(skeleton).toBeInTheDocument()
    })

    it('page header has proper heading hierarchy', async () => {
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Should have h1 for main title
      const heading = screen.getByRole('heading', { level: 1 })
      expect(heading).toHaveTextContent('Configuration')
    })

    it('tab content updates when tab changes for screen readers', async () => {
      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // When switching tabs, content should update
      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        // The card title should update to reflect the tab
        // Multiple h3s exist (card title + scenario names), first one is the section title
        const headings = screen.getAllByRole('heading', { level: 3 })
        expect(headings[0]).toHaveTextContent('Scenarios')
      })
    })
  })

  // ============================================
  // ERROR HANDLING TESTS
  // ============================================
  describe('Error Handling', () => {
    it('handles network error gracefully', async () => {
      vi.mocked(global.fetch).mockRejectedValue(new Error('Network error'))

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('handles 404 error gracefully', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Not found' }),
      } as Response)

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('handles 500 error gracefully', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' }),
      } as Response)

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('handles missing config file', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return {
            ok: true,
            json: async () => ({ ...mockProject, config_exists: false }),
          } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ error: 'Config not found' }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      render(<ConfigEditor />)

      await waitFor(
        () => {
          expect(screen.getByText(/failed to load configuration/i)).toBeInTheDocument()
        },
        { timeout: 3000 }
      )
    })

    it('handles malformed config gracefully', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          // Return config with missing required fields
          return { ok: true, json: async () => ({ version: '1.0' }) } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      // Should handle missing scenarios gracefully - renders without crashing
      const user = userEvent.setup()
      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      // The scenarios tab should be visible - card title shows "Scenarios"
      await waitFor(() => {
        const cardTitle = screen.getByRole('heading', { level: 3 })
        expect(cardTitle).toHaveTextContent('Scenarios')
      })
    })

    it('handles empty scenarios array', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return { ok: true, json: async () => ({ ...mockConfig, scenarios: [] }) } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      // With empty array, no scenario cards should be shown
      // Card title shows "Scenarios" (h3 heading)
      await waitFor(() => {
        const cardTitle = screen.getByRole('heading', { level: 3 })
        expect(cardTitle).toHaveTextContent('Scenarios')
      })

      // Verify no scenario names are present
      expect(screen.queryByText('happy_path')).not.toBeInTheDocument()
      expect(screen.queryByText('error_case')).not.toBeInTheDocument()
    })
  })

  // ============================================
  // EDGE CASES
  // ============================================
  describe('Edge Cases', () => {
    it('handles config with only required fields', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: true,
            json: async () => ({
              version: '1.0',
              scenarios: [],
            }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      expect(screen.getByText('1.0')).toBeInTheDocument()
    })

    it('handles scenarios with no flow steps', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: true,
            json: async () => ({
              ...mockConfig,
              scenarios: [{ name: 'empty_scenario', flow: [] }],
            }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        expect(screen.getByText('empty_scenario')).toBeInTheDocument()
      })

      // Should show 0 steps
      expect(screen.getByText(/0 steps/)).toBeInTheDocument()
    })

    it('handles scenarios with no tags', async () => {
      vi.mocked(global.fetch).mockImplementation(async (url) => {
        const urlStr = url.toString()
        if (urlStr.includes('/api/local/project')) {
          return { ok: true, json: async () => mockProject } as Response
        }
        if (urlStr.includes('/api/local/config')) {
          return {
            ok: true,
            json: async () => ({
              ...mockConfig,
              scenarios: [{ name: 'no_tags_scenario', flow: [{ task: 'Test' }] }],
            }),
          } as Response
        }
        return { ok: false, status: 404 } as Response
      })

      const user = userEvent.setup()
      render(<ConfigEditor />)

      await waitFor(() => {
        expect(screen.getByText('Configuration Viewer')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('tab', { name: /scenarios/i }))

      await waitFor(() => {
        expect(screen.getByText('no_tags_scenario')).toBeInTheDocument()
      })
    })
  })
})
