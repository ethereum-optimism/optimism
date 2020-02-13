const HDWalletProvider = require('truffle-hdwallet-provider');
// const infuraKey = "fj4jll3k.....";
//
// const fs = require('fs');
const mnemonic = 'candy maple cake sugar pudding cream honey rich smooth crumble sweet treat'; // fs.readFileSync(".secret").toString().trim();

module.exports = {
  /**
   * Networks define how you connect to your ethereum client and let you set the
   * defaults web3 uses to send transactions. If you don't specify one truffle
   * will spin up a development blockchain for you on port 9545 when you
   * run `develop` or `test`. You can ask a truffle command to use a specific
   * network from the command line, e.g
   *
   * $ truffle test --network fullnode ./truffle-tests/test-erc20.js
   */

  networks: {
    // Note: Requires running the rollup-full-node locally.
    fullnode: {
      network_id: 108,
      provider: function() {
        const wallet = new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10);

        const sendAsync = wallet.sendAsync

        wallet.sendAsync = function (...args) {
          if (args[0].method === 'eth_sendTransaction') {
            // HACK TO PROPERLY SET CHAIN ID
            args[0].params[0].chainId = 108
          }
          sendAsync.apply(this, args)
        };

        return wallet;
      },
      gasPrice: 0,
      gas: 9000000,
    },
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    timeout: 100000
  },


  compilers: {
    solc: {
      // Add path to the solc-transpiler
      // TODO: will need EXECUTION_MANAGER_ADDRESS environment variable set.
      version: "../../node_modules/@eth-optimism/solc-transpiler",
    }
  }
}
