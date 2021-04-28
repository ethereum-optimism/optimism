import { nodeResolve } from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import { babel } from '@rollup/plugin-babel';
import json from '@rollup/plugin-json'

export default [{
  input: 'dist/index.js',
  output: {
    name: "core_utils",
    file: './dist-browser/index.js',
    format: 'iife',
    sourcemap: true,
  },
  plugins: [
    json(),
    nodeResolve(),
    commonjs(),
    babel({ comments: false }),
  ],
}];