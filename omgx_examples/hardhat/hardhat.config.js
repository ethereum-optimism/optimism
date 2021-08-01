require('@nomiclabs/hardhat-ethers')
require('@nomiclabs/hardhat-waffle')
require('hardhat-deploy')
require('@eth-optimism/hardhat-ovm')
require('dotenv').config();
const env = process.env;


module.exports = {
  networks: {
    // Add this network to your config!
    optimism: {
      url: 'http://127.0.0.1:8545',
      // instantiate with a mnemonic so that you have >1 accounts available
      accounts: {
        mnemonic: env.mnemonic
      },
      // accounts: [env.privateKey1, env.privateKey2],
      gasPrice: 15000000,
      ovm: true // This sets the network as using the ovm and ensure contract will be compiled against that.
    },
    // Add this network to your config!
    omgx_rinkeby: {
      url: 'https://rinkeby.omgx.network',
      // instantiate with a mnemonic so that you have >1 accounts available
      accounts: {
        mnemonic: env.mnemonic
      },
      // accounts: [env.privateKey1, env.privateKey2],
      gasPrice: 15000000,
      ovm: true // This sets the network as using the ovm and ensure contract will be compiled against that.
    },
  },
  solidity: '0.7.6',
  ovm: {
    solcVersion: '0.7.6'
  },
  namedAccounts: {
    deployer: 0
  },
}
