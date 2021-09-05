
require('dotenv').config();
const env = process.env;
const mnemonicPhrase = env.mnemonic
const HDWalletProvider = require('@truffle/hdwallet-provider')

const pk_0 = env.pk_0
const pk_1 = env.pk_1
const pk_2 = env.pk_2

module.exports = {
  contracts_build_directory: './build-ovm',
  networks: {
    rinkeby_l2: {
      provider: function () {
        return new HDWalletProvider({
          privateKeys: [ pk_0, pk_1, pk_2 ],
          providerOrUrl: 'https://rinkeby.boba.network',
        })
      },
      network_id: 28,
      host: 'https://rinkeby.boba.network',
      gasPrice: 15000000,
      gas: 803900000,
    }
  },
  compilers: {
    solc: {
      version: 'node_modules/@eth-optimism/solc',
      settings: {
        optimizer: {
          enabled: true,
          runs: 1,
        },
      },
    },
  },
}
