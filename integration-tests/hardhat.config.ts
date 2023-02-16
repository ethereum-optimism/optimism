import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import 'hardhat-gas-reporter'
import './tasks/check-block-hashes'
import { envConfig } from './test/shared/utils'

const enableGasReport = !!process.env.ENABLE_GAS_REPORT

const config: HardhatUserConfig = {
  networks: {
    optimism: {
      url: process.env.L2_URL || 'http://localhost:8545',
    },
  },
  mocha: {
    timeout: envConfig.MOCHA_TIMEOUT,
    bail: envConfig.MOCHA_BAIL,
  },
  solidity: {
    compilers: [
      {
        version: '0.7.6',
        settings: {},
      },
      {
        version: '0.8.9',
        settings: {
          optimizer: { enabled: true, runs: 200 },
          metadata: {
            bytecodeHash: 'none',
          },
          outputSelection: {
            '*': {
              '*': ['storageLayout'],
            },
          },
        },
      },
      {
        version: '0.8.15',
        settings: {
          optimizer: { enabled: true, runs: 200 },
          metadata: {
            bytecodeHash: 'none',
          },
          outputSelection: {
            '*': {
              '*': ['storageLayout'],
            },
          },
        },
      },
    ],
  },
  gasReporter: {
    enabled: enableGasReport,
    currency: 'USD',
    gasPrice: 100,
    outputFile: process.env.CI ? 'gas-report.txt' : undefined,
  },
}

export default config
