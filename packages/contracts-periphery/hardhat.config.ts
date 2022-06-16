import { HardhatUserConfig } from 'hardhat/types'
import { getenv } from '@eth-optimism/core-utils'
import * as dotenv from 'dotenv'

import { configSpec } from './src/config/deploy'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@nomiclabs/hardhat-etherscan'
import '@eth-optimism/hardhat-deploy-config'
import 'solidity-coverage'
import 'hardhat-gas-reporter'
import 'hardhat-deploy'

// Hardhat tasks
import './tasks'

// Load environment variables from .env
dotenv.config()

const config: HardhatUserConfig = {
  networks: {
    optimism: {
      chainId: 10,
      url: 'https://mainnet.optimism.io',
      verify: {
        etherscan: {
          apiKey: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
        },
      },
    },
    'optimism-kovan': {
      chainId: 69,
      url: 'https://kovan.optimism.io',
      verify: {
        etherscan: {
          apiKey: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
        },
      },
    },
    ethereum: {
      chainId: 1,
      url: `https://mainnet.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
    },
    goerli: {
      chainId: 5,
      url: `https://goerli.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
    },
    ropsten: {
      chainId: 3,
      url: `https://ropsten.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
    },
    kovan: {
      chainId: 42,
      url: `https://kovan.infura.io/v3/${getenv('INFURA_PROJECT_ID')}`,
      verify: {
        etherscan: {
          apiKey: getenv('ETHEREUM_ETHERSCAN_API_KEY'),
        },
      },
    },
  },
  paths: {
    deployConfig: './config/deploy',
  },
  deployConfigSpec: configSpec,
  mocha: {
    timeout: 50000,
  },
  solidity: {
    compilers: [
      {
        version: '0.8.9',
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
      default: `ledger://${getenv('LEDGER_ADDRESS')}`,
      hardhat: 0,
    },
  },
}

export default config
