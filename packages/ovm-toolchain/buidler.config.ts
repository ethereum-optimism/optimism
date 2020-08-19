import { usePlugin } from '@nomiclabs/buidler/config'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')

import './src/buidler-plugins/buidler-ovm-compiler'
import './src/buidler-plugins/buidler-ovm-node'

const config: any = {
  networks: {
    buidlerevm: {
      blockGasLimit: 100000000,
    },
  },
  paths: {
    sources: './test/common/contracts',
    tests: './test/test-buidler',
    cache: './test/temp/build/buidler/cache',
    artifacts: './test/temp/build/buidler/artifacts',
  },
  mocha: {
    timeout: 50000,
  },
  solc: {
    path: '../../node_modules/@eth-optimism/solc-transpiler',
    executionManagerAddress: '0x6454c9d69a4721feba60e26a367bd4d56196ee7c',
  },
}

export default config
