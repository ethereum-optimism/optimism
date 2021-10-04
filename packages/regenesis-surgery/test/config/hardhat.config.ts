import { HardhatUserConfig } from 'hardhat/config'

const config: HardhatUserConfig = {
  // All paths relative to ** this file **.
  paths: {
    tests: '../../test',
    cache: '../temp/cache',
    artifacts: '../temp/artifacts',
  },
  mocha: {
    timeout: 100000,
  },
}

export default config
