// Plugins
require('@nomiclabs/hardhat-ethers')
require('@nomiclabs/hardhat-waffle')
require('@eth-optimism/hardhat-ovm')
require('hardhat-deploy')

module.exports = {
  networks: {
    // Add this network to your config!
    optimism: {
      url: 'http://127.0.0.1:8545',
      // This sets the gas price to 0 for all transactions on L2. We do this
      // because account balances are not automatically initiated with an ETH
      // balance.
      gasPrice: 0,
      ovm: true // This sets the network as using the ovm and ensure contract will be compiled against that.
    },
    omgx_rinkeby: {
      url: 'https://rinkeby.omgx.network',
      // instantiate with a mnemonic so that you have >1 accounts available
      accounts: {
        mnemonic: 'test test test test test test test test test test test junk'
      },
      // This sets the gas price to 0 for all transactions on L2. We do this
      // because account balances are not automatically initiated with an ETH
      // balance (yet, sorry!).
      gasPrice: 0,
      ovm: true // This sets the network as using the ovm and ensure contract will be compiled against that.
    },
  },
  solidity: '0.7.6',
  ovm: {
    solcVersion: '0.7.6'
  },
  mocha: {
    timeout: 300000
  },
  namedAccounts: {
    deployer: 0
  },
}
