module.exports = {
  skipFiles: [
    './test-helpers',
    './test-libraries',
    './L2/predeploys/OVM_DeployerWhitelist.sol'
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
