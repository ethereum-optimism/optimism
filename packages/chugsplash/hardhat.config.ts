import { HardhatUserConfig } from 'hardhat/config'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'
import '@typechain/hardhat'

const config: HardhatUserConfig = {
  solidity: {
    version: '0.8.9',
    settings: {
      outputSelection: {
        '*': {
          '*': ['storageLayout'],
        },
      },
    },
  },
  typechain: {
    outDir: 'dist/types',
    target: 'ethers-v5',
  },
}

export default config
