import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useToastStore, toast, type ToastType } from './toast'

describe('Toast Store', () => {
  beforeEach(() => {
    // Reset store state
    useToastStore.setState({ toasts: [] })
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('Initial State', () => {
    it('should start with empty toasts array', () => {
      const state = useToastStore.getState()
      expect(state.toasts).toEqual([])
    })
  })

  describe('addToast', () => {
    it('should add a toast with generated id', () => {
      useToastStore.getState().addToast({
        type: 'success',
        title: 'Test Toast',
      })

      const state = useToastStore.getState()
      expect(state.toasts).toHaveLength(1)
      expect(state.toasts[0].title).toBe('Test Toast')
      expect(state.toasts[0].type).toBe('success')
      expect(state.toasts[0].id).toMatch(/^toast-\d+$/)
    })

    it('should add toast with description', () => {
      useToastStore.getState().addToast({
        type: 'error',
        title: 'Error Title',
        description: 'Error description',
      })

      const state = useToastStore.getState()
      expect(state.toasts[0].description).toBe('Error description')
    })

    it('should auto-remove toast after default duration', () => {
      useToastStore.getState().addToast({
        type: 'info',
        title: 'Auto-remove test',
      })

      expect(useToastStore.getState().toasts).toHaveLength(1)

      // Fast-forward past default duration (4000ms)
      vi.advanceTimersByTime(4001)

      expect(useToastStore.getState().toasts).toHaveLength(0)
    })

    it('should respect custom duration', () => {
      useToastStore.getState().addToast({
        type: 'warning',
        title: 'Custom duration',
        duration: 10000,
      })

      expect(useToastStore.getState().toasts).toHaveLength(1)

      // Should still be there after 4 seconds
      vi.advanceTimersByTime(4001)
      expect(useToastStore.getState().toasts).toHaveLength(1)

      // Should be gone after 10 seconds
      vi.advanceTimersByTime(6000)
      expect(useToastStore.getState().toasts).toHaveLength(0)
    })

    it('should not auto-remove when duration is 0', () => {
      useToastStore.getState().addToast({
        type: 'info',
        title: 'Persistent toast',
        duration: 0,
      })

      expect(useToastStore.getState().toasts).toHaveLength(1)

      // Should still be there after a long time
      vi.advanceTimersByTime(100000)
      expect(useToastStore.getState().toasts).toHaveLength(1)
    })

    it('should add multiple toasts', () => {
      const { addToast } = useToastStore.getState()

      addToast({ type: 'success', title: 'Toast 1' })
      addToast({ type: 'error', title: 'Toast 2' })
      addToast({ type: 'info', title: 'Toast 3' })

      expect(useToastStore.getState().toasts).toHaveLength(3)
    })

    it('should generate unique ids for each toast', () => {
      const { addToast } = useToastStore.getState()

      addToast({ type: 'success', title: 'Toast 1' })
      addToast({ type: 'success', title: 'Toast 2' })
      addToast({ type: 'success', title: 'Toast 3' })

      const toasts = useToastStore.getState().toasts
      const ids = toasts.map(t => t.id)
      const uniqueIds = new Set(ids)

      expect(uniqueIds.size).toBe(3)
    })
  })

  describe('removeToast', () => {
    it('should remove a specific toast by id', () => {
      const { addToast, removeToast } = useToastStore.getState()

      addToast({ type: 'success', title: 'Toast 1', duration: 0 })
      addToast({ type: 'error', title: 'Toast 2', duration: 0 })
      addToast({ type: 'info', title: 'Toast 3', duration: 0 })

      const toasts = useToastStore.getState().toasts
      const idToRemove = toasts[1].id

      removeToast(idToRemove)

      const updatedToasts = useToastStore.getState().toasts
      expect(updatedToasts).toHaveLength(2)
      expect(updatedToasts.find(t => t.id === idToRemove)).toBeUndefined()
      expect(updatedToasts[0].title).toBe('Toast 1')
      expect(updatedToasts[1].title).toBe('Toast 3')
    })

    it('should handle removing non-existent toast', () => {
      useToastStore.getState().addToast({ type: 'success', title: 'Test', duration: 0 })

      expect(useToastStore.getState().toasts).toHaveLength(1)

      // Remove non-existent id
      useToastStore.getState().removeToast('non-existent-id')

      expect(useToastStore.getState().toasts).toHaveLength(1)
    })
  })

  describe('clearToasts', () => {
    it('should remove all toasts', () => {
      const { addToast, clearToasts } = useToastStore.getState()

      addToast({ type: 'success', title: 'Toast 1', duration: 0 })
      addToast({ type: 'error', title: 'Toast 2', duration: 0 })
      addToast({ type: 'info', title: 'Toast 3', duration: 0 })

      expect(useToastStore.getState().toasts).toHaveLength(3)

      clearToasts()

      expect(useToastStore.getState().toasts).toHaveLength(0)
    })

    it('should handle clearing when already empty', () => {
      expect(useToastStore.getState().toasts).toHaveLength(0)

      useToastStore.getState().clearToasts()

      expect(useToastStore.getState().toasts).toHaveLength(0)
    })
  })

  describe('Toast Convenience Functions', () => {
    it('should create success toast', () => {
      toast.success('Success!', 'Operation completed')

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(1)
      expect(toasts[0].type).toBe('success')
      expect(toasts[0].title).toBe('Success!')
      expect(toasts[0].description).toBe('Operation completed')
    })

    it('should create error toast', () => {
      toast.error('Error!', 'Something went wrong')

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(1)
      expect(toasts[0].type).toBe('error')
      expect(toasts[0].title).toBe('Error!')
      expect(toasts[0].description).toBe('Something went wrong')
    })

    it('should create info toast', () => {
      toast.info('Info', 'Just so you know')

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(1)
      expect(toasts[0].type).toBe('info')
      expect(toasts[0].title).toBe('Info')
    })

    it('should create warning toast', () => {
      toast.warning('Warning!', 'Be careful')

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(1)
      expect(toasts[0].type).toBe('warning')
      expect(toasts[0].title).toBe('Warning!')
    })

    it('should create toast without description', () => {
      toast.success('Just title')

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(1)
      expect(toasts[0].title).toBe('Just title')
      expect(toasts[0].description).toBeUndefined()
    })
  })

  describe('Toast Types', () => {
    it('should support all toast types', () => {
      const types: ToastType[] = ['success', 'error', 'info', 'warning']

      types.forEach((type, index) => {
        useToastStore.getState().addToast({
          type,
          title: `Toast ${index}`,
          duration: 0,
        })
      })

      const toasts = useToastStore.getState().toasts
      expect(toasts).toHaveLength(4)
      expect(toasts.map(t => t.type)).toEqual(types)
    })
  })

  describe('Toast Interface', () => {
    it('should have correct structure', () => {
      useToastStore.getState().addToast({
        type: 'success',
        title: 'Test',
        description: 'Description',
        duration: 5000,
      })

      const toast = useToastStore.getState().toasts[0]

      expect(toast).toHaveProperty('id')
      expect(toast).toHaveProperty('type')
      expect(toast).toHaveProperty('title')
      expect(toast).toHaveProperty('description')
      expect(toast).toHaveProperty('duration')
    })
  })
})
