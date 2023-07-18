import { defineConfig } from 'tsup'
import packageJson from './package.json'

export default defineConfig({
  name: packageJson.name,
  entry: ['src/index.ts', 'src/actions.ts', 'src/react.ts'],
  outDir: 'dist',
  format: ['esm', 'cjs'],
  splitting: false,
  sourcemap: true,
  clean: false,
})
