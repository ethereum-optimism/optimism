import { ethers } from 'ethers'
import { HardhatUserConfig } from 'hardhat/config'
import dotenv from 'dotenv'

// Hardhat plugins
import '@eth-optimism/hardhat-deploy-config'
import '@foundry-rs/hardhat-forge'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

// Hardhat tasks
import './tasks'

// Deploy configuration
import { deployConfigSpec } from './scripts/deploy-config'

// Load environment variables
dotenv.config()

const config: HardhatUserConfig = {
  networks: {
    hardhat: {
      live: false,
    },
    local: {
      live: true,
      url: 'http://localhost:8545',
      saveDeployments: !!process.env.SAVE_DEPLOYMENTS || false,
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    'hardhat-local': {
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: true,
    },
    'boba-sepolia': {
      chainId: 11155111,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: true,
    },
  },
  foundry: {
    buildInfo: true,
  },
  paths: {
    deploy: './deploy',
    deployments: './deployments',
    deployConfig: './deploy-config',
  },
  namedAccounts: {
    deployer: {
      default: 0,
    },
  },
  deployConfigSpec,
  external: {
    contracts: [
      {
        artifacts: '../contracts/artifacts',
      },
    ],
    deployments: {
      local: ['../contracts/deployments/local'],
      'hardhat-local': ['../contracts/deployments/hardhat-local'],
      'boba-sepolia': ['../contracts/deployments/boba-sepolia'],
    },
  },
  solidity: {
    compilers: [
      {
        version: '0.8.15',
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
      {
        version: '0.5.17', // Required for WETH9
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
    ],
    settings: {
      metadata: {
        bytecodeHash:
          process.env.FOUNDRY_PROFILE === 'echidna' ? 'ipfs' : 'none',
      },
      outputSelection: {
        '*': {
          '*': ['metadata', 'storageLayout'],
        },
      },
    },
  },
}

export default config
