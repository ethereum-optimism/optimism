import { HardhatUserConfig } from 'hardhat/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@eth-optimism/hardhat-ovm'
import './tasks/deploy'

const config: HardhatUserConfig = {
  mocha: {
    timeout: 300000,
  },
  networks: {
    optimism: {
      url: 'http://localhost:8545',
      saveDeployments: false,
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
    deployer: {
      default: 0,
    },
  },
}

export default config