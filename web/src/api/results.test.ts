import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchResults, fetchResult, deleteResult } from './results'

describe('results API', () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch)
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('fetchResults', () => {
    it('should fetch all result IDs successfully', async () => {
      const mockResponse = {
        results: ['result-1', 'result-2', 'result-3'],
        count: 3
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchResults()

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/results', expect.any(Object))
      expect(result).toEqual(mockResponse)
      expect(result.results).toHaveLength(3)
    })

    it('should handle empty results list', async () => {
      const mockResponse = {
        results: [],
        count: 0
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchResults()

      expect(result.results).toHaveLength(0)
      expect(result.count).toBe(0)
    })

    it('should throw error on failed request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' })
      })

      await expect(fetchResults()).rejects.toThrow()
    })
  })

  describe('fetchResult', () => {
    it('should fetch a specific result by id', async () => {
      const mockResponse = {
        id: 'result-123',
        name: 'test-run',
        start_time: '2024-01-01T00:00:00Z',
        duration: '1m30s',
        stats: {
          passed: 5,
          failed: 1,
          skipped: 0
        },
        scenarios: []
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchResult('result-123')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/results/result-123', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should throw error when result not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Result not found' })
      })

      await expect(fetchResult('non-existent')).rejects.toThrow()
    })

    it('should handle result with detailed scenario data', async () => {
      const mockResponse = {
        id: 'result-456',
        name: 'complex-run',
        start_time: '2024-01-01T12:00:00Z',
        duration: '5m',
        stats: {
          passed: 10,
          failed: 2,
          skipped: 3
        },
        scenarios: [
          {
            scenario_name: 'scenario-1',
            state: 'passed',
            duration: '30s',
            flow_results: []
          },
          {
            scenario_name: 'scenario-2',
            state: 'failed',
            duration: '1m',
            flow_results: []
          }
        ],
        environment: {
          hostname: 'test-host',
          os: 'linux'
        }
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchResult('result-456')

      expect(result.scenarios).toHaveLength(2)
      expect(result.stats.failed).toBe(2)
    })
  })

  describe('deleteResult', () => {
    it('should delete a result successfully', async () => {
      const mockResponse = {
        status: 'deleted'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await deleteResult('result-123')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/results/result-123', expect.objectContaining({
        method: 'DELETE'
      }))
      expect(result).toEqual(mockResponse)
    })

    it('should throw error when result not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Result not found' })
      })

      await expect(deleteResult('non-existent')).rejects.toThrow()
    })

    it('should handle server error during deletion', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Failed to delete result file' })
      })

      await expect(deleteResult('result-123')).rejects.toThrow()
    })
  })
})
