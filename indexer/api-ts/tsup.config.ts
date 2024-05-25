import packageJson from './package.json'

export default {
  name: packageJson.name,
  entry: ['indexer.ts'],
  outDir: '.',
  format: ['esm', 'cjs'],
  splitting: false,
  sourcemap: true,
  clean: false,
}
