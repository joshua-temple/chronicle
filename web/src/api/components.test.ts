import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchComponents, fetchComponent } from './components'

describe('components API', () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch)
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('fetchComponents', () => {
    it('should fetch all components successfully', async () => {
      const mockResponse = {
        components: [
          { name: 'SetupDB', type: 'setup', source_file: 'setup.go', tags: ['database'] },
          { name: 'TestTask', type: 'task', source_file: 'task.go', tags: ['unit'] },
          { name: 'Validate', type: 'validation', source_file: 'validate.go', tags: [] }
        ],
        count: 3
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchComponents()

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/components', expect.any(Object))
      expect(result).toEqual(mockResponse)
      expect(result.components).toHaveLength(3)
    })

    it('should handle empty components list', async () => {
      const mockResponse = {
        components: [],
        count: 0
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchComponents()

      expect(result.components).toHaveLength(0)
      expect(result.count).toBe(0)
    })

    it('should throw error on failed request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' })
      })

      await expect(fetchComponents()).rejects.toThrow()
    })

    it('should handle components with dependencies', async () => {
      const mockResponse = {
        components: [
          {
            name: 'ComplexTask',
            type: 'task',
            source_file: 'complex.go',
            dependencies: ['db:*sql.DB', 'cache:*redis.Client'],
            tags: ['integration']
          }
        ],
        count: 1
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchComponents()

      expect(result.components[0].dependencies).toHaveLength(2)
    })
  })

  describe('fetchComponent', () => {
    it('should fetch a specific component by name', async () => {
      const mockResponse = {
        name: 'SetupDB',
        type: 'setup',
        source_file: 'internal/setup/db.go',
        line: 42,
        dependencies: ['config:*Config'],
        produces: ['db:*sql.DB'],
        tags: ['database', 'setup']
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchComponent('SetupDB')

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/components/SetupDB', expect.any(Object))
      expect(result).toEqual(mockResponse)
    })

    it('should encode special characters in component name', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ name: 'My/Component' })
      })

      await fetchComponent('My/Component')

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/components/My%2FComponent',
        expect.any(Object)
      )
    })

    it('should throw error when component not found', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: 'Component not found' })
      })

      await expect(fetchComponent('NonExistent')).rejects.toThrow()
    })

    it('should handle component with all fields', async () => {
      const mockResponse = {
        name: 'FullComponent',
        type: 'task',
        source_file: 'full.go',
        line: 100,
        dependencies: ['db:*sql.DB'],
        produces: ['result:*Result'],
        requires: ['config:*Config'],
        tags: ['full', 'test'],
        description: 'A fully featured component'
      }

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      })

      const result = await fetchComponent('FullComponent')

      expect(result.name).toBe('FullComponent')
      expect(result.type).toBe('task')
      expect(result.tags).toContain('full')
    })
  })
})
