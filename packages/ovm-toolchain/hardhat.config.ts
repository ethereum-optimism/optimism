import path from 'path'
import { task } from 'hardhat/config'

import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-waffle'

import './src/plugins/hardhat/ovm-compiler'

const config: any = {
  paths: {
    sources: './test/common/contracts',
    tests: './test/test-buidler',
    cache: './test/temp/build/hardhat/cache',
    artifacts: './test/temp/build/hardhat/artifacts',
  },
  mocha: {
    timeout: 50000,
  },
  solidity: {
    version: '0.5.16',
    settings: {},
  }
}

export default config
