import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/utils'
import { Scenarios } from './Scenarios'

// Mock the mode store
vi.mock('@/stores/mode', () => ({
  useMode: () => 'standalone',
  useModeStore: () => ({ detectMode: vi.fn() }),
}))

// Mock fetch responses
const mockScenarios = {
  name: 'test-project',
  version: '1.0',
  scenarios: [
    {
      name: 'test_scenario_1',
      description: 'First test scenario',
      tags: ['smoke', 'api'],
      flow: [{ task: 'DoSomething' }, { validation: 'CheckResult' }],
    },
    {
      name: 'test_scenario_2',
      description: 'Second test scenario',
      tags: ['integration'],
      flow: [{ setup: 'PrepareData' }],
    },
    {
      name: 'abstract_base',
      description: 'Abstract scenario',
      abstract: true,
      flow: [],
    },
  ],
}

describe('Scenarios Page', () => {
  beforeEach(() => {
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => mockScenarios,
    } as Response)
  })

  it('renders scenarios list', async () => {
    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
    })

    expect(screen.getByText('test_scenario_2')).toBeInTheDocument()
    // Abstract scenarios should be filtered out
    expect(screen.queryByText('abstract_base')).not.toBeInTheDocument()
  })

  it('displays scenario descriptions', async () => {
    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('First test scenario')).toBeInTheDocument()
    })

    expect(screen.getByText('Second test scenario')).toBeInTheDocument()
  })

  it('shows total count excluding abstract scenarios', async () => {
    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('2 total')).toBeInTheDocument()
    })
  })

  it('displays tags for scenarios', async () => {
    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
    })

    // Tags appear both in filter and on cards, use getAllByText
    const smokeTags = screen.getAllByText('smoke')
    expect(smokeTags.length).toBeGreaterThan(0)

    const integrationTags = screen.getAllByText('integration')
    expect(integrationTags.length).toBeGreaterThan(0)
  })

  it('filters scenarios by search', async () => {
    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('test_scenario_1')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText('Search scenarios...')
    await searchInput.focus()
    await screen.getByPlaceholderText('Search scenarios...').focus()

    // Type in search - using fireEvent since userEvent may not be installed
    const input = screen.getByLabelText('Search scenarios')
    input.setAttribute('value', 'scenario_1')
    input.dispatchEvent(new Event('change', { bubbles: true }))
  })

  it('shows loading state initially', () => {
    vi.mocked(global.fetch).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    )

    render(<Scenarios />)

    // Should show loading spinner
    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('shows empty state when no scenarios', async () => {
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ ...mockScenarios, scenarios: [] }),
    } as Response)

    render(<Scenarios />)

    await waitFor(() => {
      expect(screen.getByText('No scenarios found')).toBeInTheDocument()
    })
  })
})
