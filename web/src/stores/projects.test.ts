import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'

// We need to dynamically import the module to reset the store between tests
let useProjectsStore: typeof import('./projects').useProjectsStore
let useProjects: typeof import('./projects').useProjects
let useDiscoveredProjects: typeof import('./projects').useDiscoveredProjects
let useActiveProject: typeof import('./projects').useActiveProject
let useProjectsLoading: typeof import('./projects').useProjectsLoading
let useProjectsError: typeof import('./projects').useProjectsError
let _resetOperationFlags: typeof import('./projects')._resetOperationFlags

const mockProject = {
  id: 'proj-1',
  name: 'Test Project',
  path: '/path/to/project',
  addedAt: '2024-01-01T00:00:00Z',
  status: { state: 'stopped' as const },
}

const mockDiscoveredProject = {
  id: 'disc-1',
  name: 'Discovered Project',
  path: '/path/to/discovered',
  addedAt: '2024-01-02T00:00:00Z',
  autoDiscovered: true,
  status: { state: 'unknown' as const },
}

describe('Projects Store', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    vi.resetModules()
    // Re-import the module fresh
    const module = await import('./projects')
    useProjectsStore = module.useProjectsStore
    useProjects = module.useProjects
    useDiscoveredProjects = module.useDiscoveredProjects
    useActiveProject = module.useActiveProject
    useProjectsLoading = module.useProjectsLoading
    useProjectsError = module.useProjectsError
    _resetOperationFlags = module._resetOperationFlags
    // Reset operation flags before each test
    _resetOperationFlags()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ============================================
  // INITIAL STATE TESTS
  // ============================================
  describe('Initial State', () => {
    it('starts with empty projects array', () => {
      const { result } = renderHook(() => useProjects())
      expect(result.current).toEqual([])
    })

    it('starts with empty discovered array', () => {
      const { result } = renderHook(() => useDiscoveredProjects())
      expect(result.current).toEqual([])
    })

    it('starts with loading false', () => {
      const { result } = renderHook(() => useProjectsLoading())
      expect(result.current).toBe(false)
    })

    it('starts with null error', () => {
      const { result } = renderHook(() => useProjectsError())
      expect(result.current).toBeNull()
    })

    it('starts with null activeProjectId', () => {
      const { result } = renderHook(() => useProjectsStore())
      expect(result.current.activeProjectId).toBeNull()
    })

    it('useActiveProject returns null initially', () => {
      const { result } = renderHook(() => useActiveProject())
      expect(result.current).toBeNull()
    })
  })

  // ============================================
  // FETCH PROJECTS TESTS
  // ============================================
  describe('fetchProjects', () => {
    it('fetches projects successfully', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.projects).toEqual([mockProject])
      expect(result.current.loading).toBe(false)
      expect(result.current.error).toBeNull()
      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects')
    })

    it('sets loading state during fetch', async () => {
      let resolvePromise: (value: Response) => void
      const fetchPromise = new Promise<Response>((resolve) => {
        resolvePromise = resolve
      })
      vi.mocked(global.fetch).mockReturnValueOnce(fetchPromise)

      const { result } = renderHook(() => useProjectsStore())

      // Start the fetch
      act(() => {
        result.current.fetchProjects()
      })

      // Check loading state
      expect(result.current.loading).toBe(true)

      // Resolve the fetch
      await act(async () => {
        resolvePromise!({
          ok: true,
          json: async () => ({ projects: [] }),
        } as Response)
      })

      expect(result.current.loading).toBe(false)
    })

    it('handles fetch error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).toBe('Failed to fetch projects: Internal Server Error')
      expect(result.current.loading).toBe(false)
    })

    it('handles network error', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'))

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).toContain('network error')
      expect(result.current.loading).toBe(false)
    })

    it('handles empty projects response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: null }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.projects).toEqual([])
    })
  })

  // ============================================
  // ADD PROJECT TESTS
  // ============================================
  describe('addProject', () => {
    it('adds a project successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ id: 'new-proj' }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockProject, { ...mockProject, id: 'new-proj' }] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.addProject({ name: 'New Project', path: '/new/path' })
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'New Project', path: '/new/path' }),
      })
      expect(result.current.error).toBeNull()
    })

    it('handles add project error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Bad Request',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        // addProject now re-throws error, so we need to catch it
        try {
          await result.current.addProject({ name: 'New Project' })
        } catch {
          // Expected to throw
        }
      })

      expect(result.current.error).toContain('Failed to add project')
    })

    it('refreshes projects after adding', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ id: 'new-proj' }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockProject] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.addProject({ name: 'New Project' })
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
      expect(global.fetch).toHaveBeenNthCalledWith(2, '/api/standalone/projects')
    })
  })

  // ============================================
  // REMOVE PROJECT TESTS
  // ============================================
  describe('removeProject', () => {
    it('removes a project successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.removeProject('proj-1')
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects/proj-1', {
        method: 'DELETE',
      })
      expect(result.current.error).toBeNull()
    })

    it('handles remove project error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Not Found',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await expect(result.current.removeProject('nonexistent')).rejects.toThrow(
          'Failed to remove project: Not Found'
        )
      })

      expect(result.current.error).toBe('Failed to remove project: Not Found')
    })
  })

  // ============================================
  // UPDATE PROJECT TESTS
  // ============================================
  describe('updateProject', () => {
    it('updates a project successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [{ ...mockProject, name: 'Updated Name' }] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.updateProject('proj-1', { name: 'Updated Name' })
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects/proj-1', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'Updated Name' }),
      })
      expect(result.current.error).toBeNull()
    })

    it('handles update project error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Forbidden',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await expect(result.current.updateProject('proj-1', { name: 'New Name' })).rejects.toThrow(
          'Failed to update project: Forbidden'
        )
      })

      expect(result.current.error).toBe('Failed to update project: Forbidden')
    })
  })

  // ============================================
  // LAUNCH PROJECT TESTS
  // ============================================
  describe('launchProject', () => {
    it('launches a project successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ port: 8080 }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            projects: [{ ...mockProject, status: { state: 'running', port: 8080 } }],
          }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.launchProject('proj-1')
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects/proj-1/launch', {
        method: 'POST',
      })
      expect(result.current.error).toBeNull()
    })

    it('handles launch project error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Service Unavailable',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.launchProject('proj-1')
      })

      expect(result.current.error).toContain('Failed to launch daemon')
    })

    it('refreshes projects after launching to get updated status', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockProject] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.launchProject('proj-1')
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
    })
  })

  // ============================================
  // STOP PROJECT TESTS
  // ============================================
  describe('stopProject', () => {
    it('stops a project successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            projects: [{ ...mockProject, status: { state: 'stopped' } }],
          }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.stopProject('proj-1')
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/projects/proj-1/stop', {
        method: 'POST',
      })
      expect(result.current.error).toBeNull()
    })

    it('handles stop project error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await expect(result.current.stopProject('proj-1')).rejects.toThrow('Failed to stop daemon')
      })

      expect(result.current.error).toContain('Failed to stop daemon')
    })
  })

  // ============================================
  // SET ACTIVE PROJECT TESTS
  // ============================================
  describe('setActiveProject', () => {
    it('sets active project id', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      // First fetch projects
      await act(async () => {
        await result.current.fetchProjects()
      })

      // Then set active project
      act(() => {
        result.current.setActiveProject('proj-1')
      })

      expect(result.current.activeProjectId).toBe('proj-1')
    })

    it('clears active project when set to null', () => {
      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.setActiveProject('proj-1')
      })
      expect(result.current.activeProjectId).toBe('proj-1')

      act(() => {
        result.current.setActiveProject(null)
      })
      expect(result.current.activeProjectId).toBeNull()
    })

    it('useActiveProject returns the active project', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const storeHook = renderHook(() => useProjectsStore())

      await act(async () => {
        await storeHook.result.current.fetchProjects()
      })

      act(() => {
        storeHook.result.current.setActiveProject('proj-1')
      })

      const { result } = renderHook(() => useActiveProject())
      expect(result.current).toEqual(mockProject)
    })

    it('useActiveProject returns null for nonexistent project', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const storeHook = renderHook(() => useProjectsStore())

      await act(async () => {
        await storeHook.result.current.fetchProjects()
      })

      act(() => {
        storeHook.result.current.setActiveProject('nonexistent')
      })

      const { result } = renderHook(() => useActiveProject())
      expect(result.current).toBeNull()
    })
  })

  // ============================================
  // DISCOVER TESTS
  // ============================================
  describe('discover', () => {
    it('discovers projects successfully', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockDiscoveredProject] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.discovered).toEqual([mockDiscoveredProject])
      expect(result.current.loading).toBe(false)
      expect(result.current.error).toBeNull()
      expect(global.fetch).toHaveBeenCalledWith('/api/standalone/discover', {
        method: 'POST',
      })
    })

    it('handles discover error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.error).toBe('Failed to discover projects: Internal Server Error')
      expect(result.current.loading).toBe(false)
    })

    it('handles empty discovered projects', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: null }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.discovered).toEqual([])
    })

    it('useDiscoveredProjects returns discovered projects', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [mockDiscoveredProject] }),
      } as Response)

      const storeHook = renderHook(() => useProjectsStore())

      await act(async () => {
        await storeHook.result.current.discover()
      })

      const { result } = renderHook(() => useDiscoveredProjects())
      expect(result.current).toEqual([mockDiscoveredProject])
    })
  })

  // ============================================
  // CLEAR ERROR TESTS
  // ============================================
  describe('clearError', () => {
    it('clears the error state', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Error',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).not.toBeNull()

      act(() => {
        result.current.clearError()
      })

      expect(result.current.error).toBeNull()
    })
  })

  // ============================================
  // EDGE CASES
  // ============================================
  describe('Edge Cases', () => {
    it('handles non-Error thrown objects', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce('String error')

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).toContain('Failed to fetch projects')
    })

    it('clears error before starting new operation', async () => {
      // First call fails
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        statusText: 'Error',
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })
      expect(result.current.error).not.toBeNull()

      // Second call succeeds
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      await act(async () => {
        await result.current.fetchProjects()
      })
      expect(result.current.error).toBeNull()
    })

    it('handles multiple projects in response', async () => {
      const multipleProjects = [
        mockProject,
        { ...mockProject, id: 'proj-2', name: 'Project 2' },
        { ...mockProject, id: 'proj-3', name: 'Project 3' },
      ]

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ projects: multipleProjects }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.projects).toHaveLength(3)
      expect(result.current.projects).toEqual(multipleProjects)
    })
  })

  // ============================================
  // CONCURRENT OPERATIONS TESTS
  // ============================================
  describe('Concurrent Operations', () => {
    it('prevents concurrent fetch operations', async () => {
      let resolveFirst: (value: Response) => void
      const firstPromise = new Promise<Response>((resolve) => {
        resolveFirst = resolve
      })

      vi.mocked(global.fetch).mockReturnValueOnce(firstPromise)

      const { result } = renderHook(() => useProjectsStore())

      // Start first fetch (doesn't await)
      act(() => {
        result.current.fetchProjects()
      })

      // Try to start second fetch while first is in progress
      act(() => {
        result.current.fetchProjects()
      })

      // Resolve the first fetch
      await act(async () => {
        resolveFirst!({
          ok: true,
          json: async () => ({ projects: [mockProject] }),
        } as Response)
      })

      // Only one fetch should have been made
      expect(global.fetch).toHaveBeenCalledTimes(1)
      expect(result.current.projects).toEqual([mockProject])
    })

    it('prevents concurrent discover operations', async () => {
      let resolveFirst: (value: Response) => void
      const firstPromise = new Promise<Response>((resolve) => {
        resolveFirst = resolve
      })

      vi.mocked(global.fetch).mockReturnValueOnce(firstPromise)

      const { result } = renderHook(() => useProjectsStore())

      // Start first discover (doesn't await)
      act(() => {
        result.current.discover()
      })

      // Try to start second discover while first is in progress
      act(() => {
        result.current.discover()
      })

      // Resolve the first discover
      await act(async () => {
        resolveFirst!({
          ok: true,
          json: async () => ({ projects: [mockDiscoveredProject] }),
        } as Response)
      })

      // Only one discover should have been made
      expect(global.fetch).toHaveBeenCalledTimes(1)
      expect(result.current.discovered).toEqual([mockDiscoveredProject])
    })

    it('allows fetch after previous fetch completes', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockProject] }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [{ ...mockProject, name: 'Updated' }] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      // First fetch
      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.projects[0].name).toBe('Test Project')

      // Second fetch should work since first completed
      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
      expect(result.current.projects[0].name).toBe('Updated')
    })

    it('allows discover after previous discover completes', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockDiscoveredProject] }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [{ ...mockDiscoveredProject, name: 'Updated Discovered' }] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      // First discover
      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.discovered[0].name).toBe('Discovered Project')

      // Second discover should work since first completed
      await act(async () => {
        await result.current.discover()
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
      expect(result.current.discovered[0].name).toBe('Updated Discovered')
    })

    it('resets fetch flag after error', async () => {
      vi.mocked(global.fetch)
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockProject] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      // First fetch fails
      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).toContain('network error')

      // Second fetch should work since flag was reset
      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
      expect(result.current.projects).toEqual([mockProject])
    })

    it('resets discover flag after error', async () => {
      vi.mocked(global.fetch)
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ projects: [mockDiscoveredProject] }),
        } as Response)

      const { result } = renderHook(() => useProjectsStore())

      // First discover fails
      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.error).toContain('network error')

      // Second discover should work since flag was reset
      await act(async () => {
        await result.current.discover()
      })

      expect(global.fetch).toHaveBeenCalledTimes(2)
      expect(result.current.discovered).toEqual([mockDiscoveredProject])
    })
  })

  // ============================================
  // JSON PARSING ERROR TESTS
  // ============================================
  describe('JSON Parsing Errors', () => {
    it('handles JSON parsing error during fetch', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => {
          throw new Error('Invalid JSON')
        },
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.fetchProjects()
      })

      expect(result.current.error).toContain('parse server response')
      expect(result.current.loading).toBe(false)
    })

    it('handles JSON parsing error during discover', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => {
          throw new Error('Invalid JSON')
        },
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      await act(async () => {
        await result.current.discover()
      })

      expect(result.current.error).toContain('parse server response')
      expect(result.current.loading).toBe(false)
    })
  })

  // ============================================
  // POLLING TESTS
  // ============================================
  describe('Polling', () => {
    beforeEach(() => {
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('starts with null polling interval', () => {
      const { result } = renderHook(() => useProjectsStore())
      expect(result.current.pollingIntervalId).toBeNull()
    })

    it('starts polling with default interval', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling()
      })

      // Should have an interval ID set
      expect(result.current.pollingIntervalId).not.toBeNull()
      expect(result.current.pollingIntervalMs).toBe(5000) // POLLING_INTERVAL_ACTIVE

      // Initial fetch should have been called
      expect(global.fetch).toHaveBeenCalledTimes(1)

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('starts polling with custom interval', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling(10000)
      })

      expect(result.current.pollingIntervalMs).toBe(10000)

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('stops polling and clears interval', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling()
      })

      expect(result.current.pollingIntervalId).not.toBeNull()

      act(() => {
        result.current.stopPolling()
      })

      expect(result.current.pollingIntervalId).toBeNull()
    })

    it('fetches projects at each interval', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [mockProject] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling(1000) // 1 second interval
      })

      // Initial fetch
      expect(global.fetch).toHaveBeenCalledTimes(1)

      // Advance by 1 second - use advanceTimersByTimeAsync to handle async operations
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000)
      })

      // Should have triggered another fetch
      expect(global.fetch).toHaveBeenCalledTimes(2)

      // Advance by another second
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000)
      })

      expect(global.fetch).toHaveBeenCalledTimes(3)

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('prevents multiple polling intervals when called multiple times', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling()
      })

      const firstIntervalId = result.current.pollingIntervalId

      act(() => {
        result.current.startPolling() // Second call should be ignored
      })

      // Should still have the same interval ID
      expect(result.current.pollingIntervalId).toBe(firstIntervalId)

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('setPollingInterval adjusts timing when polling', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling(5000)
      })

      expect(result.current.pollingIntervalMs).toBe(5000)

      act(() => {
        result.current.setPollingInterval(60000)
      })

      expect(result.current.pollingIntervalMs).toBe(60000)
      // Should still have an interval running
      expect(result.current.pollingIntervalId).not.toBeNull()

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('setPollingInterval does nothing when not polling', () => {
      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.setPollingInterval(60000)
      })

      // Should still be null since polling was never started
      expect(result.current.pollingIntervalId).toBeNull()
      // Interval setting should remain at default
      expect(result.current.pollingIntervalMs).toBe(5000) // POLLING_INTERVAL_ACTIVE
    })

    it('setPollingInterval does nothing if interval unchanged', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling(5000)
      })

      const originalIntervalId = result.current.pollingIntervalId

      act(() => {
        result.current.setPollingInterval(5000) // Same interval
      })

      // Should have same interval ID (no restart)
      expect(result.current.pollingIntervalId).toBe(originalIntervalId)

      // Cleanup
      act(() => {
        result.current.stopPolling()
      })
    })

    it('stopPolling does nothing when not polling', () => {
      const { result } = renderHook(() => useProjectsStore())

      // Should not throw
      act(() => {
        result.current.stopPolling()
      })

      expect(result.current.pollingIntervalId).toBeNull()
    })

    it('stops fetching after stopPolling is called', async () => {
      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({ projects: [] }),
      } as Response)

      const { result } = renderHook(() => useProjectsStore())

      act(() => {
        result.current.startPolling(1000)
      })

      // Initial fetch
      expect(global.fetch).toHaveBeenCalledTimes(1)

      act(() => {
        result.current.stopPolling()
      })

      // Advance time - should NOT trigger more fetches
      await act(async () => {
        vi.advanceTimersByTime(3000)
      })

      // Should still be just 1 fetch (the initial one)
      expect(global.fetch).toHaveBeenCalledTimes(1)
    })
  })
})
