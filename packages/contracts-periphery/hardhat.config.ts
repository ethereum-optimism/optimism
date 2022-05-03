import { HardhatUserConfig } from 'hardhat/types'
import { getenv } from '@eth-optimism/core-utils'
import * as dotenv from 'dotenv'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@nomiclabs/hardhat-etherscan'
import 'solidity-coverage'

// Hardhat tasks
import './tasks'

// Load environment variables from .env
dotenv.config()

const config: HardhatUserConfig = {
  networks: {
    optimism: {
      chainId: 10,
      url: 'https://mainnet.optimsim.io',
    },
    opkovan: {
      chainId: 69,
      url: 'https://kovan.optimism.io',
    },
    mainnet: {
      chainId: 1,
      url: 'https://rpc.ankr.com/eth',
    },
  },
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
  etherscan: {
    apiKey: {
      mainnet: getenv('ETHERSCAN_API_KEY'),
      optimisticEthereum: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
      optimisticKovan: getenv('OPTIMISTIC_ETHERSCAN_API_KEY'),
    },
  },
}

export default config
