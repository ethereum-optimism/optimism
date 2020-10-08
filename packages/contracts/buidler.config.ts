import * as path from 'path'
import { usePlugin, BuidlerConfig, task } from '@nomiclabs/buidler/config'

import {
  DEFAULT_ACCOUNTS_BUIDLER,
  GAS_LIMIT,
} from './test/test-helpers/constants'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')
usePlugin('@nomiclabs/buidler-solpp')
usePlugin('solidity-coverage')

import './plugins/hijack-compiler'

const parseSolppFlags = (): { [flag: string]: boolean } => {
  const flags: { [flag: string]: boolean } = {}

  const solppEnv = process.env.SOLPP_FLAGS
  if (!solppEnv) {
    return flags
  }

  for (const flag of solppEnv.split(',')) {
    flags[flag] = true
  }

  return flags
}

task('compile')
  .addFlag('ovm', 'Compile using OVM solc compiler')
  .setAction(async (taskArguments, bre: any, runSuper) => {
    if (taskArguments.ovm) {
      bre.config.solc = {
        path: path.resolve(__dirname, '../../node_modules/@eth-optimism/solc'),
      }
      bre.config.paths.artifacts = './build/ovm_artifacts'
    }
    await runSuper(taskArguments)
  })

const config: BuidlerConfig = {
  networks: {
    buidlerevm: {
      accounts: DEFAULT_ACCOUNTS_BUIDLER,
      blockGasLimit: GAS_LIMIT * 2,
      allowUnlimitedContractSize: true, // TEMPORARY: Will be fixed by AddressResolver PR.
    },
    coverage: {
      url: 'http://localhost:8555',
    },
  },
  mocha: {
    timeout: 50000,
  },
  solpp: {
    defs: {
      ...parseSolppFlags(),
    },
    collapseEmptyLines: true,
  },
  solc: {
    optimizer: { enabled: true, runs: 200 },
  },
  analytics: {
    enabled: false,
  },
}

export default config
