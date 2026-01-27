import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchRuns, fetchRun, createRun, cancelRun } from './runs'

describe('runs API', () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch)
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('fetchRuns', () => {
    it('should fetch all runs successfully', async () => {
      const mockResponse = {
        runs: [
          { id: 'run-1', status: 'completed', scenario_id: 'test-scenario' },
          { id: 'run-2', status: 'running', scenario_id: 'another-scenario' }
        ],
        count: 2
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchRuns()

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/runs', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should handle empty runs list', async () => {
      const mockResponse = {
        runs: [],
        count: 0
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchRuns()

      expect(result.runs).toHaveLength(0)
      expect(result.count).toBe(0)
    })

    it('should throw error on failed request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' })
      })

      await expect(fetchRuns()).rejects.toThrow()
    })
  })

  describe('fetchRun', () => {
    it('should fetch a specific run by id', async () => {
      const mockResponse = {
        id: 'run-123',
        status: 'completed',
        scenario_id: 'test-scenario',
        start_time: '2024-01-01T00:00:00Z'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchRun('run-123')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/runs/run-123', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should throw error when run not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Run not found' })
      })

      await expect(fetchRun('non-existent')).rejects.toThrow()
    })
  })

  describe('createRun', () => {
    it('should create a run with just scenario name', async () => {
      const mockResponse = {
        id: 'run-new',
        status: 'pending',
        scenario_id: 'test-scenario'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await createRun('test-scenario')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/runs', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ scenario_name: 'test-scenario' })
      }))
      expect(result).toEqual(mockResponse)
    })

    it('should create a run with flags and timeout', async () => {
      const mockResponse = {
        id: 'run-new',
        status: 'pending',
        scenario_id: 'test-scenario'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const options = {
        flags: { debug: true, env: 'test' },
        timeout: '10m'
      }

      await createRun('test-scenario', options)

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/runs', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          scenario_name: 'test-scenario',
          flags: { debug: true, env: 'test' },
          timeout: '10m'
        })
      }))
    })

    it('should throw error when scenario not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Scenario not found' })
      })

      await expect(createRun('non-existent-scenario')).rejects.toThrow()
    })

    it('should throw error on validation failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Invalid timeout format' })
      })

      await expect(createRun('test', { timeout: 'invalid' })).rejects.toThrow()
    })
  })

  describe('cancelRun', () => {
    it('should cancel a run successfully', async () => {
      const mockResponse = {
        status: 'cancelled'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await cancelRun('run-123')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/runs/run-123', expect.objectContaining({
        method: 'DELETE'
      }))
      expect(result).toEqual(mockResponse)
    })

    it('should throw error when run not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Run not found' })
      })

      await expect(cancelRun('non-existent')).rejects.toThrow()
    })

    it('should handle already completed run', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Run already completed' })
      })

      await expect(cancelRun('completed-run')).rejects.toThrow()
    })
  })
})
