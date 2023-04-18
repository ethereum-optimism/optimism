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
import { deployConfigSpec } from './src/deploy-config'

// Load environment variables
dotenv.config()

const config: HardhatUserConfig = {
  networks: {
    hardhat: {
      live: false,
    },
    local: {
      live: false,
      url: 'http://localhost:8545',
      saveDeployments: false,
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    // NOTE: The 'mainnet' network is currently being used for mainnet rehearsals.
    mainnet: {
      url: process.env.L1_RPC || 'https://mainnet-l1-rehearsal.optimism.io',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    devnetL1: {
      live: false,
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    devnetL2: {
      live: false,
      url: process.env.RPC_URL || 'http://localhost:9545',
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
      companionNetworks: {
        l2: 'optimism-goerli',
      },
    },
    'optimism-goerli': {
      chainId: 420,
      url: process.env.L2_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      companionNetworks: {
        l1: 'goerli',
      },
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
    'goerli-forked': {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: true,
    },
    'final-migration-rehearsal': {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: true,
    },
    'internal-devnet': {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: true,
    },
    'getting-started': {
      chainId: 5,
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
      goerli: [
        '../contracts/deployments/goerli',
        '../contracts-periphery/deployments/goerli',
      ],
      mainnet: [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
      'mainnet-forked': [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
      'goerli-forked': [
        '../contracts/deployments/goerli',
        '../contracts-periphery/deployments/goerli',
      ],
      'final-migration-rehearsal': [
        '../contracts/deployments/goerli',
        '../contracts-periphery/deployments/goerli',
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
