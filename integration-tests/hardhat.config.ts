import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-ovm'

const config: HardhatUserConfig = {
  mocha: {
    timeout: 100000,
  },
}

export default config
