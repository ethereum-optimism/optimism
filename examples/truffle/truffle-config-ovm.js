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
    }
  },
  compilers: {
    solc: {
      // Add path to the optimism solc fork
      path: 'node_modules/@eth-optimism/solc',
      version: '0.7.6',
      settings: {
        optimizer: {
          enabled: true,
          runs: 1
        },
      }
    }
  }
}
