import { defineConfig } from 'vitest/config'

// @see https://vitest.dev/config/
export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'istanbul',
    },
  },
})
