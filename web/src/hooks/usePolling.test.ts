import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePolling } from './usePolling'
import {
  useProjectsStore,
  _resetOperationFlags,
  POLLING_INTERVAL_ACTIVE,
  POLLING_INTERVAL_HIDDEN,
} from '@/stores/projects'

describe('usePolling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    _resetOperationFlags()

    // Reset store state
    useProjectsStore.setState({
      projects: [],
      discovered: [],
      loading: false,
      error: null,
      activeProjectId: null,
      pollingIntervalId: null,
      pollingIntervalMs: POLLING_INTERVAL_ACTIVE,
    })

    // Mock fetch
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ projects: [] }),
    } as Response)
  })

  afterEach(() => {
    // Stop any active polling
    const state = useProjectsStore.getState()
    if (state.pollingIntervalId !== null) {
      state.stopPolling()
    }

    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('starts polling on mount by default', () => {
    renderHook(() => usePolling())

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalId).not.toBeNull()
  })

  it('does not start polling if autoStart is false', () => {
    renderHook(() => usePolling({ autoStart: false }))

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalId).toBeNull()
  })

  it('stops polling on unmount', () => {
    const { unmount } = renderHook(() => usePolling())

    expect(useProjectsStore.getState().pollingIntervalId).not.toBeNull()

    unmount()

    expect(useProjectsStore.getState().pollingIntervalId).toBeNull()
  })

  it('uses custom active interval', () => {
    renderHook(() => usePolling({ activeInterval: 3000 }))

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalMs).toBe(3000)
  })

  it('adjusts interval on visibility change to hidden', async () => {
    renderHook(() => usePolling())

    // Simulate tab becoming hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true })

    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalMs).toBe(POLLING_INTERVAL_HIDDEN)
  })

  it('adjusts interval on visibility change to visible', async () => {
    // Start with hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true })

    renderHook(() => usePolling())

    // Simulate tab becoming visible
    Object.defineProperty(document, 'hidden', { value: false, writable: true })

    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalMs).toBe(POLLING_INTERVAL_ACTIVE)
  })

  it('uses custom hidden interval', async () => {
    renderHook(() => usePolling({ hiddenInterval: 30000 }))

    // Simulate tab becoming hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true })

    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalMs).toBe(30000)
  })

  it('removes visibility listener on unmount', () => {
    const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')

    const { unmount } = renderHook(() => usePolling())

    unmount()

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      'visibilitychange',
      expect.any(Function)
    )
  })

  it('returns polling control functions', () => {
    const { result } = renderHook(() => usePolling())

    expect(result.current.startPolling).toBeDefined()
    expect(result.current.stopPolling).toBeDefined()
    expect(result.current.setPollingInterval).toBeDefined()
  })

  it('uses hidden interval if tab is hidden on mount', () => {
    Object.defineProperty(document, 'hidden', { value: true, writable: true })

    renderHook(() => usePolling())

    const state = useProjectsStore.getState()
    expect(state.pollingIntervalMs).toBe(POLLING_INTERVAL_HIDDEN)
  })

  it('fetches projects immediately on start', () => {
    renderHook(() => usePolling())

    // Should have made an initial fetch
    expect(globalThis.fetch).toHaveBeenCalledWith('/api/standalone/projects')
  })
})
