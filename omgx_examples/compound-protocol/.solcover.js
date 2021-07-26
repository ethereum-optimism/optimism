const {execSync} = require('child_process');
async function moveCoverage(config) {
  execSync('mv ./.coverage_artifacts/contracts ./networks/coverage-contracts');
}

async function moveCoverageBack() {
  execSync('mv ./networks/coverage-contracts ./.coverage_artifacts/contracts');
}

module.exports = {
  port: 8555,
  providerOpts: 
    { // See example coverage settings at https://github.com/sc-forks/solidity-coverage
      gas: 0xfffffff,
      gasPrice: 0x01
    },
  mocha: {
    enableTimeouts: false,
    grep: /@gas|@no-cov/,
    invert: true
  },
  onCompileComplete: moveCoverage,
  onTestsComplete: moveCoverageBack,
  skipFiles: ['test'].concat(
    process.env['SKIP_UNITROLLER'] ? ['Unitroller.sol'] : []),
};
