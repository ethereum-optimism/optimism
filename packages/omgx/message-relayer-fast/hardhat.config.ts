import { HardhatUserConfig } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@eth-optimism/hardhat-ovm'
import 'hardhat-deploy'

const config: HardhatUserConfig = {
  mocha: {
    timeout: 200000,
  },
  networks: {
    omgx: {
      url: 'http://localhost:8545',
      // This sets the gas price to 0 for all transactions on L2. We do this
      // because account balances are not automatically initiated with an ETH
      // balance.
      gasPrice: 0,
      ovm: true,
    },
    localhost: {
			url: "http://localhost:9545",
			allowUnlimitedContractSize: true,
      timeout: 1800000,
      accounts: ['0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80']
		},
  },
  solidity: '0.7.6',
  ovm: {
    solcVersion: '0.7.6',
  },
  namedAccounts: {
    deployer: 0
  }
}

export default config
