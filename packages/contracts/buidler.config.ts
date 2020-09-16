import { usePlugin, BuidlerConfig } from '@nomiclabs/buidler/config'

import {
  DEFAULT_ACCOUNTS_BUIDLER,
  GAS_LIMIT,
} from './test/helpers/constants'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')

import './test/helpers/buidler/modify-compiler'

const config: BuidlerConfig = {
  networks: {
    buidlerevm: {
      accounts: DEFAULT_ACCOUNTS_BUIDLER,
      blockGasLimit: GAS_LIMIT * 2,
    },
  },
  mocha: {
    timeout: 50000,
  },
  solc: {
    version: "0.7.0",
    optimizer: { enabled: true, runs: 200 },
  },
}

export default config
