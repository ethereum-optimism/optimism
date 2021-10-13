/* External Imports */
import chai = require('chai')
import Mocha from 'mocha'
import chaiAsPromised from 'chai-as-promised'
import * as dotenv from 'dotenv'
import { reqenv, getenv } from '@eth-optimism/core-utils'
import { providers } from 'ethers'
import { SurgeryDataSources } from '../scripts/types'
import { loadSurgeryData } from '../scripts/data'

// Chai plugins go here.
chai.use(chaiAsPromised)

const should = chai.should()
const expect = chai.expect

dotenv.config()

interface TestEnvConfig {
  preL2ProviderUrl: string
  postL2ProviderUrl: string
  stateDumpHeight: string | number
}

const config = (): TestEnvConfig => {
  const height = getenv('REGEN__STATE_DUMP_HEIGHT')
  return {
    preL2ProviderUrl: reqenv('REGEN__PRE_L2_PROVIDER_URL'),
    postL2ProviderUrl: reqenv('REGEN__POST_L2_PROVIDER_URL'),
    stateDumpHeight: parseInt(height, 10) || 'latest',
  }
}

// A TestEnv that contains all of the required test data
class TestEnv {
  // Config
  config: TestEnvConfig

  // An L2 provider configured to be able to query a pre
  // regenesis L2 node. This node should be synced to the
  // height that the state dump was taken
  preL2Provider: providers.StaticJsonRpcProvider

  // An L2 provider configured to be able to query a post
  // regenesis L2 node. This L2 node was initialized with
  // the results of the state surgery script
  postL2Provider: providers.StaticJsonRpcProvider

  // The datasources used for doing state surgery
  surgeryDataSources: SurgeryDataSources

  constructor(opts: TestEnvConfig) {
    this.config = opts
    this.preL2Provider = new providers.StaticJsonRpcProvider(
      opts.preL2ProviderUrl
    )
    this.postL2Provider = new providers.StaticJsonRpcProvider(
      opts.postL2ProviderUrl
    )
  }

  // Read the big files from disk. Without bumping the size of the nodejs heap,
  // this can oom the process. Prefix the test command with:
  // $ NODE_OPTIONS=--max_old_space=8912
  async init() {
    if (this.surgeryDataSources === undefined) {
      this.surgeryDataSources = await loadSurgeryData()
    }
  }
}

// Create a singleton test env that can be imported into each
// test file. It is important that the async operations are only
// called once as they take awhile. Each test file should be sure
// to call `env.init()` in a `before` clause to ensure that
// the files are read from disk at least once
let env: TestEnv
try {
  if (env === undefined) {
    const cfg = config()
    env = new TestEnv(cfg)
  }
} catch (e) {
  console.error(`unable to initialize test env: ${e.toString()}`)
}

export { should, expect, Mocha, env }
