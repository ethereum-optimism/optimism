import dotenv from 'dotenv'
import { HardhatUserConfig } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-etherscan'
import '@nomiclabs/hardhat-waffle'
import 'hardhat-gas-reporter'
import 'solidity-coverage'

import './scripts/deploy-token'
import './scripts/multi-send'
import './scripts/mint-initial-supply'
import './scripts/generate-merkle-root'
import './scripts/create-airdrop-json'
import './scripts/deploy-distributor'
import './scripts/test-claims'
import './scripts/create-distributor-json'
import './scripts/deposit'

dotenv.config()

const privKey = process.env.PRIVATE_KEY || '0x' + '11'.repeat(32)

const config: HardhatUserConfig = {
  solidity: {
    version: '0.8.12',
    settings: {
      outputSelection: {
        '*': {
          '*': ['metadata', 'storageLayout'],
        },
      },
    },
  },
  networks: {
    optimism: {
      chainId: 17,
      url: 'http://localhost:8545',
    },
    'optimism-kovan': {
      chainId: 69,
      url: 'https://kovan.optimism.io',
      accounts: [privKey],
    },
    'optimism-goerli': {
      chainId: 420,
      url: 'https://goerli.optimism.io',
      accounts: [privKey],
    },
    'optimism-nightly': {
      chainId: 421,
      url: 'https://goerli-nightly-us-central1-a-sequencer.optimism.io',
      accounts: [privKey],
    },
    'optimism-mainnet': {
      chainId: 10,
      url: 'https://mainnet.optimism.io',
      accounts: [privKey],
    },
    'hardhat-node': {
      url: 'http://localhost:8545',
    },
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    currency: 'USD',
  },
  etherscan: {
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
}

export default config
