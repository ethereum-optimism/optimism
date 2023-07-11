import { defineConfig } from 'tsup'

export default defineConfig({
  name: '@eth-optimsim/contracts-ts',
  entry: ['src/index.ts', 'src/actions.ts', 'src/react.ts'],
  outDir: 'dist',
  format: ['esm', 'cjs'],
  splitting: false,
  sourcemap: true,
  clean: false,
})
