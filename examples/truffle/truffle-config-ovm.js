const mnemonicPhrase = "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat"
const HDWalletProvider = require('@truffle/hdwallet-provider')

module.exports = {
  contracts_build_directory: './build-ovm',
  networks: {
    optimism: {
      provider: function () {
        return new HDWalletProvider({
          mnemonic: {
            phrase: mnemonicPhrase
          },
          providerOrUrl: 'http://127.0.0.1:8545'
        })
      },
      network_id: 420,
      host: '127.0.0.1',
      port: 8545,
      gasPrice: 0,
      gas: 54180127,
    }
  },
  compilers: {
    solc: {
      // Add path to the optimism solc fork
      version: './node_modules/@eth-optimism/solc',
      settings: {
        optimizer: {
          enabled: true,
          runs: 1
        },
      }
    }
  }
}
