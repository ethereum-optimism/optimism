import * as path from 'path'
import { usePlugin, BuidlerConfig } from '@nomiclabs/buidler/config'

import { DEFAULT_ACCOUNTS_BUIDLER } from './src/constants'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')

import './plugins/hijack-compiler'

const config: BuidlerConfig = {
  networks: {
    buidlerevm: {
      accounts: DEFAULT_ACCOUNTS_BUIDLER
    }
  }
}

export default config