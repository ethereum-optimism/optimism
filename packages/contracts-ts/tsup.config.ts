import { defineConfig } from 'tsup'
import packageJson from './package.json'

export default defineConfig({
  name: packageJson.name,
  entry: ['src/index.ts'],
  outDir: 'dist',
  format: ['esm', 'cjs'],
  dts: true,
  splitting: false,
  sourcemap: true,
  clean: false,
})
