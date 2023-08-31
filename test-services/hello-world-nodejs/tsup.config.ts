import { defineConfig } from 'tsup'
import packageJson from './package.json'

// @see https://tsup.egoist.dev/
export default defineConfig({
  name: packageJson.name,
  entry: ['index.ts'],
  outDir: 'dist',
  format: ['esm', 'cjs'],
  splitting: false,
  sourcemap: true,
  clean: false,
  dts: true
})
