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
      network_id: 28,
      host: '127.0.0.1',
      port: 8545,
      gasPrice: 0,
      gas: 54180127,
    },
    omgx_rinkeby: {
      provider: function () {
        return new HDWalletProvider({
          mnemonic: {
            phrase: mnemonicPhrase
          },
          providerOrUrl: 'http://rinkeby.omgx.network'
        })
      },
      network_id: 28,
      host: 'http://rinkeby.omgx.network',
      gasPrice: 0,
      gas: 0,
    }
  },
  compilers: {
    solc: {
      version: '../../node_modules/@eth-optimism/solc',
      settings: {
        optimizer: {
          enabled: true,
          runs: 1
        },
      }
    }
  }
}