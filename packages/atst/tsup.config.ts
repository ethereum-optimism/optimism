import { defineConfig } from 'tsup'

export default defineConfig({
  name: '@eth-optimism/atst',
  /**
   * This is also a cli app and tsup will automatically make the cli entrypoint executable
   * @see https://tsup.egoist.dev/#building-cli-app
   */
  entry: ['src/index.ts', 'src/cli.ts'],
  outDir: 'dist',
  format: ['esm', 'cjs'],
  splitting: false,
  sourcemap: true,
  clean: true,
})
