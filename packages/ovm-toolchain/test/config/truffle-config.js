const mnemonic = "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat";
const { ganache } = require('@eth-optimism/ovm-toolchain')

const GAS_LIMIT = 10000000
const GAS_PRICE = 0

module.exports = {
  contracts_directory: './test/common/contracts',
  contracts_build_directory: './test/temp/build/truffle',

  networks: {
    test: {
      network_id: 108,
      networkCheckTimeout: 100000,
      provider: function() {
        return ganache.provider({
          mnemonic: mnemonic,
          network_id: 108,
          default_balance_ether: 100,
          gasLimit: GAS_LIMIT,
          gasPrice: GAS_PRICE,
        })
      },
      gas: GAS_LIMIT,
      gasPrice: GAS_PRICE,
    },
  },

  mocha: {
    timeout: 100000
  },

  compilers: {
    solc: {
      version: "../../node_modules/@eth-optimism/solc",
    }
  }
}
