import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@eth-optimism/hardhat-ovm'
import 'hardhat-gas-reporter'

const enableGasReport = !!process.env.ENABLE_GAS_REPORT

const config: HardhatUserConfig = {
  mocha: {
    timeout: 100000,
  },
  networks: {
    optimism: {
      url: process.env.L2_URL || 'http://localhost:8545',
      ovm: true,
    },
  },
  solidity: {
    version: '0.7.6',
    settings: {
      optimizer: { enabled: true, runs: 1 },
    },
  },
  ovm: {
    solcVersion: '0.7.6',
  },
  gasReporter: {
    enabled: enableGasReport,
    currency: 'USD',
    gasPrice: 100,
    outputFile: process.env.CI ? 'gas-report.txt' : undefined,
  },
}

export default config
