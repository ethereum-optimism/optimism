module.exports = {
  skipFiles: [
    './test-helpers',
    './test-libraries',
    './L2/predeploys/OVM_DeployerWhitelist.sol',
    './lib',
    './L1/rollup/L2OutputOracle.sol',
    './test',
  ],
  mocha: {
    grep: '@skip-on-coverage',
    invert: true,
  },
}
