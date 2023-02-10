import assert from 'assert'

import { HardhatUserConfig, subtask } from 'hardhat/config'
import { TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS } from 'hardhat/builtin-tasks/task-names'
import { getenv } from '@eth-optimism/core-utils'
import * as dotenv from 'dotenv'

import { configSpec } from './src/config/deploy'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@nomiclabs/hardhat-etherscan'
import '@eth-optimism/hardhat-deploy-config'
import '@typechain/hardhat'
import 'solidity-coverage'
import 'hardhat-gas-reporter'
import 'hardhat-deploy'

// Hardhat tasks
import './tasks'

// Load environment variables from .env
dotenv.config()

subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS).setAction(
  async (_, __, runSuper) => {
    const paths = await runSuper()

    return paths.filter((p: string) => !p.endsWith('.t.sol'))
  }
)

assert(
  !(getenv('PRIVATE_KEY') && getenv('LEDGER_ADDRESS')),
  'use only one of PRIVATE_KEY or LEDGER_ADDRESS'
)

const accounts = getenv('PRIVATE_KEY')
  ? [getenv('PRIVATE_KEY')]
  : (undefined as any)

const config: HardhatUserConfig = {
  networks: {
    optimism: {
      chainId: 10,
      url: 'https://mainnet.optimism.io',
      accounts,
      verify: {
        etherscan: {
          apiKey: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
        },
      },
      companionNetworks: {
        l1: 'mainnet',
      },
    },
    'optimism-goerli': {
      chainId: 420,
      url: 'https://goerli.optimism.io',
      accounts,
      verify: {
        etherscan: {
          apiKey: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
        },
      },
      companionNetworks: {
        l1: 'goerli',
      },
    },
    mainnet: {
      chainId: 1,
      url: `https://mainnet.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      accounts,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
      companionNetworks: {
        l2: 'optimism',
      },
    },
    goerli: {
      live: true,
      chainId: 5700,
      url: `https://rpc.tanenbaum.io`,
      accounts,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
      companionNetworks: {
        l2: 'optimism-goerli',
      },
    },
    ropsten: {
      chainId: 3,
      url: `https://ropsten.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      accounts,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
    },
    'ops-l2': {
      chainId: 17,
      accounts: [
        '0x3b8d2345102cce2443acb240db6e87c8edd4bb3f821b17fab8ea2c9da08ea132',
        '0xa6aecc98b63bafb0de3b29ae9964b14acb4086057808be29f90150214ebd4a0f',
      ],
      url: 'http://127.0.0.1:8545',
      companionNetworks: {
        l1: 'ops-l1',
      },
    },
    'ops-l1': {
      chainId: 31337,
      accounts: [
        '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
      url: 'http://127.0.0.1:9545',
      companionNetworks: {
        l2: 'ops-l2',
      },
    },
  },
  paths: {
    deployConfig: './config/deploy',
  },
  external: {
    contracts: [
      {
        artifacts: '../contracts/artifacts',
      },
    ],
    deployments: {
      goerli: ['../contracts/deployments/goerli'],
      mainnet: ['../contracts/deployments/mainnet'],
    },
  },
  deployConfigSpec: configSpec,
  mocha: {
    timeout: 50000,
  },
  typechain: {
    outDir: 'dist/types',
    target: 'ethers-v5',
  },
  solidity: {
    compilers: [
      {
        version: '0.8.16',
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
      {
        version: '0.8.15',
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
    ],
    settings: {
      metadata: {
        bytecodeHash: 'none',
      },
      outputSelection: {
        '*': {
          '*': ['metadata', 'storageLayout'],
        },
      },
    },
  },
  namedAccounts: {
    deployer: {
      default: getenv('LEDGER_ADDRESS')
        ? `ledger://${getenv('LEDGER_ADDRESS')}`
        : 0,
      hardhat: 0,
    },
  },
}

export default config
