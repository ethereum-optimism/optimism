import { defineConfig } from 'vitest/config'

// @see https://vitest.dev/config/
export default defineConfig({
  test: {
    setupFiles: './setupVitest.ts',
    environment: 'jsdom',
    coverage: {
      provider: 'istanbul',
    },
  },
})
