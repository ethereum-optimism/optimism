const HDWalletProvider = require('truffle-hdwallet-provider');

const mnemonic = 'candy maple cake sugar pudding cream honey rich smooth crumble sweet treat';

// Set this to the desired Execution Manager Address -- required for the transpiler
process.env.EXECUTION_MANAGER_ADDRESS = process.env.EXECUTION_MANAGER_ADDRESS || "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA";
const gasPrice = process.env.OVM_DEFAULT_GAS_PRICE || 0;
const gas = process.env.OVM_DEFAULT_GAS || 9000000;
const chainId = process.env.OVM_CHAIN_ID || 108;

module.exports = {
  /**
   * Note: this expects the local fullnode to be running:
   * // TODO: Run `yarn server:fullnode` in rollup-full-node before executing this test
   *
   * To run tests:
   * $ truffle test ./truffle-tests/test-erc20.js --config truffle-config-ovm.js
   */
  networks: {
    // Note: Requires running the rollup-full-node locally.
    test: {
      network_id: 108,
      provider: function() {
        const wallet = new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10);
        const sendAsync = wallet.sendAsync

        wallet.sendAsync = function (...args) {
          if (args[0].method === 'eth_sendTransaction') {
            // HACK TO PROPERLY SET CHAIN ID
            args[0].params[0].chainId = chainId
          }
          sendAsync.apply(this, args)
        };
        return wallet;
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
