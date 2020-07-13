import { usePlugin, BuidlerConfig } from '@nomiclabs/buidler/config'

import {
  DEFAULT_ACCOUNTS_BUIDLER,
  GAS_LIMIT,
} from './test/test-helpers/constants'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')
usePlugin('solidity-coverage')

import './plugins/hijack-compiler'

const config: BuidlerConfig = {
  networks: {
    buidlerevm: {
      accounts: DEFAULT_ACCOUNTS_BUIDLER,
      blockGasLimit: GAS_LIMIT * 2
    },
    coverage: {
      url: 'http://localhost:8555',
    },
  },
}

export default config
