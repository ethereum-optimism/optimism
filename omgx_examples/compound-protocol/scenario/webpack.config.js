'use strict'
const webpack = require('webpack');

module.exports = {
  stats: 'verbose',
  devtool: 'source-map',
  externals: {
    file: '{}',
    fs: '{}',
    tls: '{}',
    net: '{}',
    xmlhttprequest: '{}',
    'truffle-flattener': '{}',
    'request': '{}'
  },
  optimization: {
    minimize: true
  },
  plugins: [
    new webpack.IgnorePlugin(/^\.\/locale$/, /moment$/),
    new webpack.DefinePlugin({
        'process.env': {
            // This has effect on the react lib size
            'NODE_ENV': JSON.stringify('production'),
        }
    })
  ]
}
