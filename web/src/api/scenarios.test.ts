import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchScenarios, fetchScenario } from './scenarios'

describe('scenarios API', () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch)
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('fetchScenarios', () => {
    it('should fetch all scenarios successfully', async () => {
      const mockResponse = {
        scenarios: [
          { name: 'scenario-1', description: 'First scenario', tags: ['unit'] },
          { name: 'scenario-2', description: 'Second scenario', tags: ['integration'] }
        ],
        count: 2
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchScenarios()

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/scenarios', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should handle empty scenarios list', async () => {
      const mockResponse = {
        scenarios: [],
        count: 0
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchScenarios()

      expect(result.scenarios).toHaveLength(0)
      expect(result.count).toBe(0)
    })

    it('should throw error on failed request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' })
      })

      await expect(fetchScenarios()).rejects.toThrow()
    })
  })

  describe('fetchScenario', () => {
    it('should fetch a specific scenario by name', async () => {
      const mockResponse = {
        name: 'test-scenario',
        description: 'A test scenario',
        tags: ['unit', 'fast'],
        timeout: '5m',
        flow_count: 3
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchScenario('test-scenario')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/scenarios/test-scenario', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should encode special characters in scenario name', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ name: 'scenario/with/slashes' })
      })

      await fetchScenario('scenario/with/slashes')

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/scenarios/scenario%2Fwith%2Fslashes',
        expect.any(Object)
      )
    })

    it('should throw error when scenario not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Scenario not found' })
      })

      await expect(fetchScenario('non-existent')).rejects.toThrow()
    })
  })
})
