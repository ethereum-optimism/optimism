import { HardhatUserConfig } from 'hardhat/types'
import { ethers } from 'ethers'

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
    hivenet: {
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
  },
  external: {
    contracts: [
      {
        artifacts: '../contracts-bedrock/artifacts',
      },
    ],
    deployments: {
      hivenet: ['../contracts-bedrock/deployments/hivenet'],
      devnetL1: ['../contracts-bedrock/deployments/devnetL1'],
      goerli: ['../contracts-bedrock/deployments/goerli'],
    },
  },
}

export default config
