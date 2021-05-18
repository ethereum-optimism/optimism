import { HardhatUserConfig } from 'hardhat/config'

import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'

const config: HardhatUserConfig = {
  paths: {
    sources: './test/test-contracts',
  },
  solidity: {
    version: '0.7.6',
  },
}

export default config
