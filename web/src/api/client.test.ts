import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { ApiError, apiRequest } from './client'

describe('API Client', () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch)
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('ApiError', () => {
    it('should create error with status and message', () => {
      const error = new ApiError(404, 'Not found')

      expect(error).toBeInstanceOf(Error)
      expect(error).toBeInstanceOf(ApiError)
      expect(error.name).toBe('ApiError')
      expect(error.status).toBe(404)
      expect(error.message).toBe('Not found')
    })

    it('should be catchable as Error', () => {
      const error = new ApiError(500, 'Server error')

      try {
        throw error
      } catch (e) {
        expect(e).toBeInstanceOf(Error)
        expect((e as ApiError).status).toBe(500)
      }
    })
  })

  describe('apiRequest', () => {
    it('should make request to correct URL', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: 'test' }),
      })

      await apiRequest('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/test',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      )
    })

    it('should return parsed JSON on success', async () => {
      const responseData = { scenarios: [], count: 0 }
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(responseData),
      })

      const result = await apiRequest('/scenarios')

      expect(result).toEqual(responseData)
    })

    it('should pass through request options', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      })

      await apiRequest('/runs', {
        method: 'POST',
        body: JSON.stringify({ scenario_name: 'test' }),
      })

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/runs',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ scenario_name: 'test' }),
        })
      )
    })

    it('should merge headers', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      })

      await apiRequest('/test', {
        headers: {
          'Authorization': 'Bearer token123',
        },
      })

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/test',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': 'Bearer token123',
          }),
        })
      )
    })

    it('should throw ApiError on non-ok response with error message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Scenario not found' }),
      })

      await expect(apiRequest('/scenarios/nonexistent')).rejects.toThrow(ApiError)

      try {
        await apiRequest('/scenarios/nonexistent')
      } catch (e) {
        const error = e as ApiError
        expect(error.status).toBe(404)
        expect(error.message).toBe('Scenario not found')
      }
    })

    it('should handle response with no error field', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
      })

      try {
        await apiRequest('/test')
      } catch (e) {
        const error = e as ApiError
        expect(error.status).toBe(500)
        expect(error.message).toBe('Request failed')
      }
    })

    it('should handle non-JSON error response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      })

      try {
        await apiRequest('/test')
      } catch (e) {
        const error = e as ApiError
        expect(error.status).toBe(500)
        expect(error.message).toBe('Unknown error')
      }
    })

    it('should propagate network errors', async () => {
      const networkError = new Error('Network error')
      mockFetch.mockRejectedValueOnce(networkError)

      await expect(apiRequest('/test')).rejects.toThrow('Network error')
    })

    it('should handle different HTTP methods', async () => {
      const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

      for (const method of methods) {
        mockFetch.mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({}),
        })

        await apiRequest('/test', { method })

        expect(mockFetch).toHaveBeenLastCalledWith(
          '/api/v1/test',
          expect.objectContaining({ method })
        )
      }
    })

    it('should type return value correctly', async () => {
      interface TestResponse {
        id: string
        name: string
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ id: '123', name: 'test' }),
      })

      const result = await apiRequest<TestResponse>('/test')

      expect(result.id).toBe('123')
      expect(result.name).toBe('test')
    })

    it('should handle empty response body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(null),
      })

      const result = await apiRequest('/test')

      expect(result).toBeNull()
    })

    it('should handle array response', async () => {
      const items = [{ id: 1 }, { id: 2 }, { id: 3 }]
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(items),
      })

      const result = await apiRequest<typeof items>('/items')

      expect(result).toEqual(items)
      expect(result.length).toBe(3)
    })
  })

  describe('Integration scenarios', () => {
    it('should support typical list operation', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          scenarios: [
            { name: 'test-1', description: 'Test 1' },
            { name: 'test-2', description: 'Test 2' },
          ],
          count: 2,
        }),
      })

      const result = await apiRequest<{
        scenarios: Array<{ name: string; description: string }>
        count: number
      }>('/scenarios')

      expect(result.count).toBe(2)
      expect(result.scenarios).toHaveLength(2)
    })

    it('should support typical create operation', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          id: 'run-123',
          status: 'running',
          scenario_id: 'test-scenario',
        }),
      })

      const result = await apiRequest<{
        id: string
        status: string
        scenario_id: string
      }>('/runs', {
        method: 'POST',
        body: JSON.stringify({ scenario_name: 'test-scenario' }),
      })

      expect(result.id).toBe('run-123')
      expect(result.status).toBe('running')
    })

    it('should support typical delete operation', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ status: 'deleted' }),
      })

      const result = await apiRequest<{ status: string }>('/results/123', {
        method: 'DELETE',
      })

      expect(result.status).toBe('deleted')
    })
  })
})
