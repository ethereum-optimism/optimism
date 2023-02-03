import { defineConfig } from 'tsup'

/**
 * @see https://tsup.egoist.dev/
 */
export default defineConfig({
  name: '@eth-optimism/atst',
  entry: ['src/index.ts'],
  outDir: 'dist',
  target: 'es2015',
  // will create a .js file for commonjs and a .cjs file for esm
  format: ['esm', 'cjs'],
  // don't generate .d.ts files.  This is default but being explicit
  // note this means we need to generate our own .d.ts files
  // this means we need to run the typechecker as a linter
  // which is a general best practice anyways.  It also means
  // we must use our .ts files as "types" in package.json
  // this is better dx since it means the types take you to
  // the source code instead of just the .d.ts file in editors
  dts: false,
  splitting: false,
  sourcemap: true,
  // remove dist folder before building
  clean: true,
})
