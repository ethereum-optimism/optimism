const HDWalletProvider = require("truffle-hdwallet-provider");
const ProviderWrapper = require("@eth-optimism/ovm-truffle-provider-wrapper");
const mnemonic = "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat";

// Set this to the desired Execution Manager Address -- required for the transpiler
process.env.EXECUTION_MANAGER_ADDRESS = process.env.EXECUTION_MANAGER_ADDRESS || "0x6454c9d69a4721feba60e26a367bd4d56196ee7c";
const gasPrice = process.env.OVM_DEFAULT_GAS_PRICE || 0;
const gas = process.env.OVM_DEFAULT_GAS || 1000000000;


module.exports = {
  contracts_build_directory: './build/truffle',
  /**
   * Note: Using the `test` network will start a local node at 'http://127.0.0.1:8545/'
   *
   * To run tests:
   * $ truffle test <test file here> --config truffle-config-ovm.js
   */
  networks: {
    test: {
      network_id: 108,
      provider: function() {
        return ProviderWrapper.wrapProviderAndStartLocalNode(new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10));
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
