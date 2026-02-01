import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

// Define mocks using vi.hoisted()
const {
  mockFetchProject,
  mockFetchConfig,
  mockSaveConfig,
  mockValidateConfig,
  mockRunDiscovery,
  mockFetchLocalComponents,
} = vi.hoisted(() => ({
  mockFetchProject: vi.fn(),
  mockFetchConfig: vi.fn(),
  mockSaveConfig: vi.fn(),
  mockValidateConfig: vi.fn(),
  mockRunDiscovery: vi.fn(),
  mockFetchLocalComponents: vi.fn(),
}))

vi.mock('@/api/local', () => ({
  fetchProject: () => mockFetchProject(),
  fetchConfig: () => mockFetchConfig(),
  saveConfig: (config: unknown) => mockSaveConfig(config),
  validateConfig: (config: unknown) => mockValidateConfig(config),
  runDiscovery: () => mockRunDiscovery(),
  fetchLocalComponents: () => mockFetchLocalComponents(),
}))

import {
  useProject,
  useConfig,
  useSaveConfig,
  useValidateConfig,
  useLocalDiscovery,
  useLocalComponents,
} from './useLocalConfig'

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe('useProject Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('should fetch project info', async () => {
    const mockProjectInfo = {
      directory: '/path/to/project',
      config_file: 'chronicle.yaml',
      config_exists: true,
      last_modified: '2024-01-01T00:00:00Z',
    }
    mockFetchProject.mockResolvedValue(mockProjectInfo)

    const { result } = renderHook(() => useProject(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockFetchProject).toHaveBeenCalledTimes(1)
    expect(result.current.data).toEqual(mockProjectInfo)
  })

  it('should handle fetch error', async () => {
    const error = new Error('Project not found')
    mockFetchProject.mockRejectedValue(error)

    const { result } = renderHook(() => useProject(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })

  it('should handle project without config', async () => {
    const mockProjectInfo = {
      directory: '/path/to/project',
      config_file: 'chronicle.yaml',
      config_exists: false,
    }
    mockFetchProject.mockResolvedValue(mockProjectInfo)

    const { result } = renderHook(() => useProject(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.config_exists).toBe(false)
    expect(result.current.data?.last_modified).toBeUndefined()
  })
})

describe('useConfig Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch config', async () => {
    const mockConfig = {
      name: 'Test Project',
      version: '1.0.0',
      scenarios: [
        {
          name: 'test-scenario',
          description: 'A test scenario',
        },
      ],
    }
    mockFetchConfig.mockResolvedValue(mockConfig)

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockFetchConfig).toHaveBeenCalledTimes(1)
    expect(result.current.data).toEqual(mockConfig)
  })

  it('should handle empty config', async () => {
    mockFetchConfig.mockResolvedValue({ version: '1.0.0' })

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.scenarios).toBeUndefined()
  })

  it('should handle config fetch error', async () => {
    const error = new Error('Config not found')
    mockFetchConfig.mockRejectedValue(error)

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })
})

describe('useSaveConfig Hook', () => {
  let queryClient: QueryClient

  beforeEach(() => {
    vi.clearAllMocks()
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          gcTime: 0,
        },
      },
    })
  })

  function createWrapperWithQueryClient() {
    return function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }
  }

  it('should save config', async () => {
    const configToSave = {
      version: '1.0.0',
      scenarios: [{ name: 'new-scenario' }],
    }
    mockSaveConfig.mockResolvedValue(undefined)

    const { result } = renderHook(() => useSaveConfig(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate(configToSave as any)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockSaveConfig).toHaveBeenCalledWith(configToSave)
  })

  it('should invalidate queries on success', async () => {
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    mockSaveConfig.mockResolvedValue(undefined)

    const { result } = renderHook(() => useSaveConfig(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate({ version: '1.0.0' } as any)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['local', 'config'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['local', 'project'] })
  })

  it('should handle save error', async () => {
    const error = new Error('Failed to save config')
    mockSaveConfig.mockRejectedValue(error)

    const { result } = renderHook(() => useSaveConfig(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate({ version: '1.0.0' } as any)

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })
})

describe('useValidateConfig Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should validate config successfully', async () => {
    const mockValidationResult = {
      valid: true,
      errors: [],
      warnings: [],
    }
    mockValidateConfig.mockResolvedValue(mockValidationResult)

    const { result } = renderHook(() => useValidateConfig(), {
      wrapper: createWrapper(),
    })

    result.current.mutate({ version: '1.0.0' } as any)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockValidateConfig).toHaveBeenCalledWith({ version: '1.0.0' })
    expect(result.current.data).toEqual(mockValidationResult)
  })

  it('should return validation errors', async () => {
    const mockValidationResult = {
      valid: false,
      errors: ['Invalid scenario name', 'Missing required field'],
      warnings: ['Deprecated feature used'],
    }
    mockValidateConfig.mockResolvedValue(mockValidationResult)

    const { result } = renderHook(() => useValidateConfig(), {
      wrapper: createWrapper(),
    })

    result.current.mutate({ version: '1.0.0' } as any)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.valid).toBe(false)
    expect(result.current.data?.errors).toHaveLength(2)
    expect(result.current.data?.warnings).toHaveLength(1)
  })

  it('should handle validation API error', async () => {
    const error = new Error('Validation service unavailable')
    mockValidateConfig.mockRejectedValue(error)

    const { result } = renderHook(() => useValidateConfig(), {
      wrapper: createWrapper(),
    })

    result.current.mutate({ version: '1.0.0' } as any)

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })
})

describe('useLocalDiscovery Hook', () => {
  let queryClient: QueryClient

  beforeEach(() => {
    vi.clearAllMocks()
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          gcTime: 0,
        },
      },
    })
  })

  function createWrapperWithQueryClient() {
    return function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }
  }

  it('should run discovery', async () => {
    const mockDiscoveryResult = {
      components: [
        {
          name: 'setup-db',
          type: 'setup' as const,
          description: 'Set up database',
          tags: ['db', 'setup'],
          produces: ['db_connection'],
          requires: [],
          source_file: 'setup.go',
        },
      ],
      discovered_at: '2024-01-01T00:00:00Z',
    }
    mockRunDiscovery.mockResolvedValue(mockDiscoveryResult)

    const { result } = renderHook(() => useLocalDiscovery(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockRunDiscovery).toHaveBeenCalledTimes(1)
    expect(result.current.data).toEqual(mockDiscoveryResult)
  })

  it('should invalidate components query on success', async () => {
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    mockRunDiscovery.mockResolvedValue({
      components: [],
      discovered_at: '2024-01-01T00:00:00Z',
    })

    const { result } = renderHook(() => useLocalDiscovery(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['local', 'components'] })
  })

  it('should handle discovery error', async () => {
    const error = new Error('Discovery failed')
    mockRunDiscovery.mockRejectedValue(error)

    const { result } = renderHook(() => useLocalDiscovery(), {
      wrapper: createWrapperWithQueryClient(),
    })

    result.current.mutate()

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })
})

describe('useLocalComponents Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch local components', async () => {
    const mockDiscoveryResult = {
      components: [
        {
          name: 'task-1',
          type: 'task' as const,
          description: 'Task 1',
          tags: ['core'],
          produces: ['result'],
          requires: ['input'],
          source_file: 'task.go',
        },
        {
          name: 'validation-1',
          type: 'validation' as const,
          description: 'Validation 1',
          tags: [],
          produces: [],
          requires: ['result'],
          source_file: 'validate.go',
        },
      ],
      discovered_at: '2024-01-01T00:00:00Z',
    }
    mockFetchLocalComponents.mockResolvedValue(mockDiscoveryResult)

    const { result } = renderHook(() => useLocalComponents(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockFetchLocalComponents).toHaveBeenCalledTimes(1)
    expect(result.current.data?.components).toHaveLength(2)
    expect(result.current.data?.components[0].name).toBe('task-1')
  })

  it('should handle empty components', async () => {
    mockFetchLocalComponents.mockResolvedValue({
      components: [],
      discovered_at: '2024-01-01T00:00:00Z',
    })

    const { result } = renderHook(() => useLocalComponents(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.components).toEqual([])
  })

  it('should handle fetch error', async () => {
    const error = new Error('Failed to fetch components')
    mockFetchLocalComponents.mockRejectedValue(error)

    const { result } = renderHook(() => useLocalComponents(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toBe(error)
  })
})
