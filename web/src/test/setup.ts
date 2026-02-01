import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Mock fetch for API tests
globalThis.fetch = vi.fn()

// Reset mocks between tests
beforeEach(() => {
  vi.clearAllMocks()
})
