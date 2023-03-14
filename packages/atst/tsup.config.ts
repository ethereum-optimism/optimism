import { defineConfig } from 'tsup'

/**
 * @see https://tsup.egoist.dev/
 */
export default defineConfig({
  name: '@eth-optimism/atst',
  /**
   * This is also a cli app and tsup will automatically make the cli entrypoint executable
   *
   * @see https://tsup.egoist.dev/#building-cli-app
   */
  entry: ['src/index.ts', 'src/cli.ts', 'src/react.ts'],
  outDir: 'dist',
  target: 'es2021',
  // will create a .js file for commonjs and a .cjs file for esm
  format: ['esm', 'cjs'],
  // don't generate .d.ts files.  This is default but being explicit
  dts: false,
  splitting: false,
  sourcemap: true,
  // remove dist folder before building
  clean: true,
})
