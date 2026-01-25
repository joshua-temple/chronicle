import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'

// We need to dynamically import the module to reset the cached mode between tests
let useModeStore: typeof import('./mode').useModeStore
let useMode: typeof import('./mode').useMode
let useIsStandalone: typeof import('./mode').useIsStandalone
let useIsDaemon: typeof import('./mode').useIsDaemon

describe('Mode Store', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    // Reset module cache to clear cachedMode between tests
    vi.resetModules()
    // Re-import the module fresh
    const module = await import('./mode')
    useModeStore = module.useModeStore
    useMode = module.useMode
    useIsStandalone = module.useIsStandalone
    useIsDaemon = module.useIsDaemon
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ============================================
  // INITIAL STATE TESTS
  // ============================================
  describe('Initial State', () => {
    it('starts with detecting mode when not cached', () => {
      const { result } = renderHook(() => useMode())
      expect(result.current).toBe('detecting')
    })

    it('useIsStandalone returns false initially', () => {
      const { result } = renderHook(() => useIsStandalone())
      expect(result.current).toBe(false)
    })

    it('useIsDaemon returns false initially', () => {
      const { result } = renderHook(() => useIsDaemon())
      expect(result.current).toBe(false)
    })
  })

  // ============================================
  // STANDALONE MODE DETECTION TESTS
  // ============================================
  describe('Standalone Mode Detection', () => {
    it('detects standalone mode from /api/standalone/mode endpoint', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ mode: 'standalone' }),
      } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('standalone')
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/standalone/mode')
    })

    it('useIsStandalone returns true in standalone mode', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ mode: 'standalone' }),
      } as Response)

      const storeHook = renderHook(() => useModeStore())
      await act(async () => {
        await storeHook.result.current.detectMode()
      })

      const { result } = renderHook(() => useIsStandalone())
      expect(result.current).toBe(true)
    })
  })

  // ============================================
  // DAEMON MODE DETECTION TESTS
  // ============================================
  describe('Daemon Mode Detection', () => {
    it('falls back to daemon mode when standalone endpoint returns 404', async () => {
      // First call to standalone endpoint fails
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: false,
          status: 404,
        } as Response)
        // Second call to health endpoint succeeds
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('daemon')
      expect(globalThis.fetch).toHaveBeenCalledTimes(2)
      expect(globalThis.fetch).toHaveBeenNthCalledWith(1, '/api/standalone/mode')
      expect(globalThis.fetch).toHaveBeenNthCalledWith(2, '/api/v1/health')
    })

    it('useIsDaemon returns true in daemon mode', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({ ok: false, status: 404 } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const storeHook = renderHook(() => useModeStore())
      await act(async () => {
        await storeHook.result.current.detectMode()
      })

      const { result } = renderHook(() => useIsDaemon())
      expect(result.current).toBe(true)
    })
  })

  // ============================================
  // DISCONNECTED MODE TESTS
  // ============================================
  describe('Disconnected Mode', () => {
    it('sets disconnected when both endpoints fail', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({ ok: false, status: 404 } as Response)
        .mockResolvedValueOnce({ ok: false, status: 404 } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('disconnected')
    })

    it('sets disconnected when network errors occur', async () => {
      vi.mocked(globalThis.fetch)
        .mockRejectedValueOnce(new Error('Network error'))
        .mockRejectedValueOnce(new Error('Network error'))

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('disconnected')
    })
  })

  // ============================================
  // CACHING BEHAVIOR TESTS
  // ============================================
  describe('Caching Behavior', () => {
    it('caches mode after first detection', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ mode: 'standalone' }),
      } as Response)

      const { result } = renderHook(() => useModeStore())

      // First detection
      await act(async () => {
        await result.current.detectMode()
      })

      expect(globalThis.fetch).toHaveBeenCalledTimes(1)

      // Second detection should use cache
      await act(async () => {
        await result.current.detectMode()
      })

      // Should not make additional fetch calls
      expect(globalThis.fetch).toHaveBeenCalledTimes(1)
      expect(result.current.mode).toBe('standalone')
    })

    it('prevents concurrent detection calls', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ mode: 'standalone' }),
      } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await Promise.all([
          result.current.detectMode(),
          result.current.detectMode(),
        ])
      })

      expect(globalThis.fetch).toHaveBeenCalledTimes(1)
    })
  })

  // ============================================
  // SET MODE TESTS
  // ============================================
  describe('setMode', () => {
    it('allows manually setting mode', () => {
      const { result } = renderHook(() => useModeStore())

      act(() => {
        result.current.setMode('daemon')
      })

      expect(result.current.mode).toBe('daemon')
    })
  })

  // ============================================
  // EDGE CASES
  // ============================================
  describe('Edge Cases', () => {
    it('handles malformed response from standalone endpoint', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ notMode: 'something' }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      // Should fall back to daemon mode since mode !== 'standalone'
      expect(result.current.mode).toBe('daemon')
    })

    it('handles response with wrong mode value', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ mode: 'other' }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('daemon')
    })

    it('handles null response from standalone endpoint', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => null,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      expect(result.current.mode).toBe('daemon')
    })

    it('handles JSON parsing error from standalone endpoint', async () => {
      vi.mocked(globalThis.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async (): Promise<unknown> => {
            throw new Error('Invalid JSON')
          },
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' }),
        } as Response)

      const { result } = renderHook(() => useModeStore())

      await act(async () => {
        await result.current.detectMode()
      })

      // Should fall back to daemon mode since JSON parsing failed
      expect(result.current.mode).toBe('daemon')
    })
  })
})
