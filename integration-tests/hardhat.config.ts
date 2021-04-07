import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-ovm'

const config: HardhatUserConfig = {
  mocha: {
    timeout: 100000,
  },
  networks: {
    optimism: {
      url: 'http://localhost:8545',
      ovm: true,
    },
  },
}

export default config
