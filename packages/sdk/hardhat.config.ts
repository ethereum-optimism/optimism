import { HardhatUserConfig } from 'hardhat/types'

import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import 'hardhat-deploy'

import './tasks'

const config: HardhatUserConfig = {
  solidity: {
    version: '0.8.9',
  },
  paths: {
    sources: './test/contracts',
  },
  networks: {
    devnetL1: {
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
  },
  external: {
    contracts: [
      {
        artifacts: '../contracts-bedrock/artifacts',
      },
    ],
    deployments: {
      devnetL1: ['../contracts-bedrock/deployments/devnetL1'],
      goerli: ['../contracts-bedrock/deployments/goerli'],
    },
  },
}

export default config
