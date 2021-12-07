import { HardhatUserConfig } from 'hardhat/types'

import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'

const config: HardhatUserConfig = {
  solidity: {
    version: '0.8.9',
  },
  paths: {
    sources: './test/contracts',
  },
}

export default config
