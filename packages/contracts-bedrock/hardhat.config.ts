import { ethers } from 'ethers'
import { HardhatUserConfig } from 'hardhat/config'

// Hardhat plugins
import '@eth-optimism/hardhat-deploy-config'
import '@foundry-rs/hardhat-forge'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

// Hardhat tasks
import './tasks'

// Deploy configuration
import { deployConfigSpec } from './src/deploy-config'

let bytecodeHash = 'none'
if (process.env.FOUNDRY_PROFILE === 'echidna') {
  bytecodeHash = 'ipfs'
}

const config: HardhatUserConfig = {
  networks: {
    hardhat: {
      live: false,
    },
    devnetL1: {
      live: false,
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    hivenet: {
      chainId: Number(process.env.CHAIN_ID),
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    goerli: {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    'alpha-1': {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    deployer: {
      chainId: Number(process.env.CHAIN_ID),
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: process.env.VERIFY_CONTRACTS === 'true',
    },
    'mainnet-forked': {
      chainId: 1,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: false,
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
      {
        artifacts: '../contracts-governance/artifacts',
      },
    ],
    deployments: {
      goerli: ['../contracts/deployments/goerli'],
      mainnet: [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
      'mainnet-forked': [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
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
        bytecodeHash,
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
