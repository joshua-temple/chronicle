import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@/test/utils'
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
      return { ok: false, status: 404 } as Response
    })
  })

  it('renders config editor with tabs', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    // Check tabs are present (they're buttons)
    expect(screen.getByRole('button', { name: /general/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /scenarios/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /infrastructure/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /chaos/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /mocks/i })).toBeInTheDocument()
  })

  it('displays version in general tab', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('1.0')).toBeInTheDocument()
    })
  })

  it('shows scenarios in scenarios tab', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    // Click scenarios tab
    fireEvent.click(screen.getByRole('button', { name: /scenarios/i }))

    await waitFor(() => {
      expect(screen.getByText('happy_path')).toBeInTheDocument()
    })

    expect(screen.getByText('error_case')).toBeInTheDocument()
  })

  it('shows infrastructure in infrastructure tab', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    // Click infrastructure tab
    fireEvent.click(screen.getByRole('button', { name: /infrastructure/i }))

    await waitFor(() => {
      expect(screen.getByText('postgres')).toBeInTheDocument()
    })

    expect(screen.getByText('redis')).toBeInTheDocument()
  })

  it('shows chaos profiles in chaos tab', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    // Click chaos tab
    fireEvent.click(screen.getByRole('button', { name: /chaos/i }))

    await waitFor(() => {
      expect(screen.getByText('network_chaos')).toBeInTheDocument()
    })
  })

  it('shows mock profiles in mocks tab', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByText('Configuration')).toBeInTheDocument()
    })

    // Click mocks tab
    fireEvent.click(screen.getByRole('button', { name: /mocks/i }))

    await waitFor(() => {
      expect(screen.getByText('happy_mocks')).toBeInTheDocument()
    })
  })

  it('has a save button', async () => {
    render(<ConfigEditor />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
    })
  })

  it('shows loading state initially', () => {
    vi.mocked(global.fetch).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    )

    render(<ConfigEditor />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
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
})
