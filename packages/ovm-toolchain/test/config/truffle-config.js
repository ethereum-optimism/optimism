const mnemonic = "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat";
const { ganache } = require('@eth-optimism/ovm-toolchain')

// Set this to the desired Execution Manager Address -- required for the transpiler
process.env.EXECUTION_MANAGER_ADDRESS = process.env.EXECUTION_MANAGER_ADDRESS || "0x6454c9d69a4721feba60e26a367bd4d56196ee7c";
const gasPrice = process.env.OVM_DEFAULT_GAS_PRICE || 0;
const gas = process.env.OVM_DEFAULT_GAS || 10000000;


module.exports = {
  contracts_directory: './test/common/contracts',
  contracts_build_directory: './test/temp/build/truffle',
  /**
   * Note: Using the `test` network will start a local node at 'http://127.0.0.1:8545/'
   *
   * To run tests:
   * $ truffle test ./truffle-tests/test-erc20.js --config truffle-config-ovm.js
   */
  networks: {
    test: {
      network_id: 108,
      networkCheckTimeout: 100000,
      provider: function() {
        return ganache.provider({
          mnemonic: mnemonic,
          network_id: 108,
          default_balance_ether: 100,
          gasLimit: 10000000,
          gasPrice: 0,
        })
      },
      gasPrice: gasPrice,
      gas: gas,
    },
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    timeout: 100000
  },

  compilers: {
    solc: {
      // Add path to the solc-transpiler
      version: "../../node_modules/@eth-optimism/solc-transpiler",
    }
  }
}
