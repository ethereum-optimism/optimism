import { defineConfig } from 'tsup'

/**
 * @see https://tsup.egoist.dev/
 */
export default defineConfig({
  name: '@eth-optimism/core-utils',
  entry: ['src/index.ts'],
  outDir: 'dist',
  target: 'es2015',
  // will create a .js file for commonjs and a .cjs file for esm
  format: ['esm'],
  // don't generate .d.ts files
  dts: false,
  splitting: false,
  sourcemap: true,
  // remove dist folder before building
  clean: true,
})
