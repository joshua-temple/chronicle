/// <reference types="vitest/globals" />

// Extend globalThis for test mocking
declare global {
  // Allow fetch to be reassigned for mocking
  interface globalThis {
    fetch: typeof fetch
  }

  // Document visibility API for polling tests
  interface Document {
    visibilityState: 'visible' | 'hidden' | 'prerender' | 'unloaded'
  }
}

export {}
