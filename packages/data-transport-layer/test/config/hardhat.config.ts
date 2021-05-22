import { HardhatUserConfig } from 'hardhat/config'

const config: HardhatUserConfig = {
  // All paths relative to ** this file **.
  paths: {
    tests: '../../test',
    cache: '../temp/cache',
    artifacts: '../temp/artifacts',
  },
}

export default config
