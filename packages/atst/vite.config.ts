import { defineConfig } from 'vitest/config'

/**
 * @see https://vitejs.dev/config/
 */
export default defineConfig({
  test: {
    environment: 'jsdom',
    testTimeout: 10000,
  },
})
