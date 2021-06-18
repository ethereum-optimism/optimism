import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@eth-optimism/hardhat-ovm'
import 'hardhat-gas-reporter'

const enableGasReport = !!process.env.ENABLE_GAS_REPORT

const config: HardhatUserConfig = {
  networks: {
    optimism: {
      url: process.env.L2_URL || 'http://localhost:8545',
      ovm: true,
      timeout: 50000,
    },
    'optimism-live': {
      url: process.env.L2_URL || 'http://localhost:8545',
      ovm: true,
      timeout: 150000,
    },
  },
  solidity: '0.7.6',
  ovm: {
    solcVersion: '0.7.6+commit.3b061308',
  },
  gasReporter: {
    enabled: enableGasReport,
    currency: 'USD',
    gasPrice: 100,
    outputFile: process.env.CI ? 'gas-report.txt' : undefined,
  },
}

export default config
