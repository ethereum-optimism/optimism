'use strict'

const path = require('path')

module.exports = {
  module: {
    rules: [
      {
        test: /\.ts$/,
        exclude: ['/node_modules/'],
        use: {
          loader: 'ts-loader',
        },
      },
    ],
  },
  node: {
    child_process: 'empty',
    fs: 'empty',
    net: 'empty',
  },
  entry: './src/connect-contracts.ts',
  target: 'web',
  output: {
    path: path.resolve(__dirname, 'build'),
    filename: 'index.js',
    libraryTarget: 'umd',
  },
  resolve: {
    extensions: ['.ts', '.js'],
  },
  plugins: [],
}
