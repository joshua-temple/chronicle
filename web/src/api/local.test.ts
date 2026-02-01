import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import {
  fetchProject,
  fetchConfig,
  saveConfig,
  validateConfig,
  runDiscovery,
  fetchLocalComponents,
  type ChronicleConfig,
  type ValidationResult,
  type ProjectInfo,
  type DiscoveryResult,
} from './local'
import { ApiError } from './client'

// Mock global fetch
const mockFetch = vi.fn()
globalThis.fetch = mockFetch

describe('Local API Functions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('fetchProject', () => {
    it('should fetch project info', async () => {
      const mockProjectInfo: ProjectInfo = {
        directory: '/path/to/project',
        config_file: 'chronicle.yaml',
        config_exists: true,
        last_modified: '2024-01-01T00:00:00Z',
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockProjectInfo,
      })

      const result = await fetchProject()

      expect(mockFetch).toHaveBeenCalledWith('/api/local/project', {
        headers: {
          'Content-Type': 'application/json',
        },
      })
      expect(result).toEqual(mockProjectInfo)
    })

    it('should throw ApiError on failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Project not found' }),
      })

      await expect(fetchProject()).rejects.toThrow(ApiError)
      await expect(fetchProject()).rejects.toThrow('Project not found')
    })

    it('should handle unknown error format', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => {
          throw new Error('Invalid JSON')
        },
      })

      await expect(fetchProject()).rejects.toThrow(ApiError)
    })
  })

  describe('fetchConfig', () => {
    it('should fetch config', async () => {
      const mockConfig: ChronicleConfig = {
        version: '1.0.0',
        scenarios: [
          {
            name: 'test-scenario',
            description: 'Test scenario',
            tags: ['unit'],
          },
        ],
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockConfig,
      })

      const result = await fetchConfig()

      expect(mockFetch).toHaveBeenCalledWith('/api/local/config', {
        headers: {
          'Content-Type': 'application/json',
        },
      })
      expect(result).toEqual(mockConfig)
    })

    it('should throw ApiError when config not found', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Config file not found' }),
      })

      await expect(fetchConfig()).rejects.toThrow(ApiError)
    })
  })

  describe('saveConfig', () => {
    it('should save config', async () => {
      const configToSave: ChronicleConfig = {
        version: '1.0.0',
        scenarios: [{ name: 'new-scenario' }],
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'ok' }),
      })

      await saveConfig(configToSave)

      expect(mockFetch).toHaveBeenCalledWith('/api/local/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(configToSave),
      })
    })

    it('should throw ApiError on save failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Invalid config format' }),
      })

      await expect(saveConfig({ version: '1.0.0' })).rejects.toThrow(ApiError)
    })
  })

  describe('validateConfig', () => {
    it('should validate config successfully', async () => {
      const configToValidate: ChronicleConfig = {
        version: '1.0.0',
        scenarios: [],
      }

      const mockValidationResult: ValidationResult = {
        valid: true,
        errors: [],
        warnings: [],
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockValidationResult,
      })

      const result = await validateConfig(configToValidate)

      expect(mockFetch).toHaveBeenCalledWith('/api/local/config/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(configToValidate),
      })
      expect(result).toEqual(mockValidationResult)
    })

    it('should return validation errors', async () => {
      const invalidConfig: ChronicleConfig = {
        version: '1.0.0',
        scenarios: [{ name: '' }], // Invalid empty name
      }

      const mockValidationResult: ValidationResult = {
        valid: false,
        errors: ['Scenario name is required'],
        warnings: [],
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockValidationResult,
      })

      const result = await validateConfig(invalidConfig)

      expect(result.valid).toBe(false)
      expect(result.errors).toContain('Scenario name is required')
    })

    it('should throw ApiError on validation endpoint failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Validation service error' }),
      })

      await expect(validateConfig({ version: '1.0.0' })).rejects.toThrow(ApiError)
    })
  })

  describe('runDiscovery', () => {
    it('should run discovery', async () => {
      const mockDiscoveryResult: DiscoveryResult = {
        components: [
          {
            name: 'setup-component',
            type: 'setup',
            description: 'Sets up the environment',
            tags: ['core'],
            produces: ['env'],
            requires: [],
            source_file: 'setup.go',
          },
        ],
        discovered_at: '2024-01-01T00:00:00Z',
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockDiscoveryResult,
      })

      const result = await runDiscovery()

      expect(mockFetch).toHaveBeenCalledWith('/api/local/discover', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      expect(result).toEqual(mockDiscoveryResult)
    })

    it('should throw ApiError on discovery failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Discovery failed' }),
      })

      await expect(runDiscovery()).rejects.toThrow(ApiError)
    })
  })

  describe('fetchLocalComponents', () => {
    it('should fetch local components', async () => {
      const mockDiscoveryResult: DiscoveryResult = {
        components: [
          {
            name: 'task-component',
            type: 'task',
            description: 'Performs a task',
            tags: ['task'],
            produces: ['result'],
            requires: ['input'],
            source_file: 'task.go',
          },
          {
            name: 'validation-component',
            type: 'validation',
            description: 'Validates results',
            tags: ['validation'],
            produces: [],
            requires: ['result'],
            source_file: 'validate.go',
          },
        ],
        discovered_at: '2024-01-01T00:00:00Z',
      }

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockDiscoveryResult,
      })

      const result = await fetchLocalComponents()

      expect(mockFetch).toHaveBeenCalledWith('/api/local/components', {
        headers: {
          'Content-Type': 'application/json',
        },
      })
      expect(result).toEqual(mockDiscoveryResult)
      expect(result.components).toHaveLength(2)
    })

    it('should return empty components list', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          components: [],
          discovered_at: '2024-01-01T00:00:00Z',
        }),
      })

      const result = await fetchLocalComponents()

      expect(result.components).toEqual([])
    })

    it('should throw ApiError on failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Failed to fetch components' }),
      })

      await expect(fetchLocalComponents()).rejects.toThrow(ApiError)
    })
  })
})

describe('Type Definitions', () => {
  it('should accept valid ProjectInfo', () => {
    const projectInfo: ProjectInfo = {
      directory: '/path/to/project',
      config_file: 'chronicle.yaml',
      config_exists: true,
      last_modified: '2024-01-01T00:00:00Z',
    }
    expect(projectInfo.directory).toBeDefined()
  })

  it('should accept ProjectInfo without last_modified', () => {
    const projectInfo: ProjectInfo = {
      directory: '/path/to/project',
      config_file: 'chronicle.yaml',
      config_exists: false,
    }
    expect(projectInfo.last_modified).toBeUndefined()
  })

  it('should accept valid ChronicleConfig', () => {
    const config: ChronicleConfig = {
      name: 'Test Project',
      version: '1.0.0',
      scenarios: [
        {
          name: 'test',
          description: 'Test scenario',
          tags: ['unit'],
          timeout: 5000000000,
          flow: [{ setup: 'init' }, { task: 'work' }],
          teardown: [{ teardown: 'cleanup' }],
        },
      ],
      infrastructure: { providers: [] },
      chaos_profiles: {},
      mock_profiles: {},
      flags: {},
      execution: {},
      results: {},
    }
    expect(config.version).toBe('1.0.0')
  })

  it('should accept minimal ChronicleConfig', () => {
    const config: ChronicleConfig = {
      version: '1.0.0',
    }
    expect(config.scenarios).toBeUndefined()
  })

  it('should accept ValidationResult', () => {
    const result: ValidationResult = {
      valid: false,
      errors: ['Error 1', 'Error 2'],
      warnings: ['Warning 1'],
    }
    expect(result.errors).toHaveLength(2)
  })

  it('should accept DiscoveryResult', () => {
    const result: DiscoveryResult = {
      components: [
        {
          name: 'component',
          type: 'setup',
          description: 'Description',
          tags: ['tag1'],
          produces: ['output'],
          requires: ['input'],
          source_file: 'file.go',
        },
      ],
      discovered_at: '2024-01-01T00:00:00Z',
    }
    expect(result.components).toHaveLength(1)
  })
})
