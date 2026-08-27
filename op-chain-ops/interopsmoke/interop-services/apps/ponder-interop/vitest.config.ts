import { defineConfig } from 'vitest/config'
import { resolve } from 'node:path'

export default defineConfig({
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  test: {
    testTimeout: 30_000,
    setupFiles: ['./__tests__/setup.ts'],
    environment: 'node',
    globals: true,
  },
})