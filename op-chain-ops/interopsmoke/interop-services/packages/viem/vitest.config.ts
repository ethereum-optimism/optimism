import path from 'path'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    testTimeout: 30_000,
    globalSetup: ['./src/test/globalSetup.ts'],
    fileParallelism: false,
  },
})
