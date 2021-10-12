/* External Imports */
import chai = require('chai')
import Mocha from 'mocha'
import chaiAsPromised from 'chai-as-promised'
import * as dotenv from 'dotenv'
import { reqenv } from '@eth-optimism/core-utils'
import { providers } from 'ethers'
import { GenesisFile, StateDump, EtherscanDump, SurgeryDataSources } from '../scripts/types'
import {
  readDumpFile,
  readEtherscanFile,
  readGenesisStateDump,
} from '../scripts/utils'

// Chai plugins go here.
chai.use(chaiAsPromised)

const should = chai.should()
const expect = chai.expect

dotenv.config()

interface TestEnvConfig {
  stateDumpFilePath: string
  etherscanFilePath: string
  genesisFilePath: string
  preL2ProviderUrl: string
  postL2ProviderUrl: string
}

const config = (): TestEnvConfig => {
  return {
    stateDumpFilePath: reqenv('REGEN__STATE_DUMP_FILE'),
    etherscanFilePath: reqenv('REGEN__ETHERSCAN_FILE'),
    genesisFilePath: reqenv('REGEN__GENESIS_FILE'),
    preL2ProviderUrl: reqenv('REGEN__PRE_L2_PROVIDER_URL'),
    postL2ProviderUrl: reqenv('REGEN__POST_L2_PROVIDER_URL'),
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
    this.preL2Provider = new providers.StaticJsonRpcProvider(opts.preL2ProviderUrl)
    this.postL2Provider = new providers.StaticJsonRpcProvider(opts.postL2ProviderUrl)
    // TODO: initialize this better for more safety
    this.surgeryDataSources = {} as SurgeryDataSources
  }

  // Read the big files from disk. Without bumping the size of the nodejs heap,
  // this can oom the process. Prefix the test command with:
  // $ NODE_OPTIONS=--max_old_space=8912
  async init() {
    if (this.surgeryDataSources.dump === undefined) {
      try {
        console.log('Reading state dump...')
        this.surgeryDataSources.dump = await readDumpFile(this.config.stateDumpFilePath)
        console.log(`${this.surgeryDataSources.dump.length} entries`)
      } catch (e) {
        console.error(e)
      }
    }

    if (this.surgeryDataSources.etherscanDump === undefined) {
      try {
        console.log('Reading etherscan dump...')
        this.surgeryDataSources.etherscanDump = await readEtherscanFile(this.config.etherscanFilePath)
        console.log(`${this.surgeryDataSources.etherscanDump.length} entries`)
      } catch (e) {
        console.error(e)
      }
    }

    if (this.surgeryDataSources.genesis === undefined) {
      try {
        console.log('Reading genesis file...')
        this.surgeryDataSources.genesis = await readGenesisStateDump(this.config.genesisFilePath)
        console.log(`${this.surgeryDataSources.genesis.length} entries`)
      } catch (e) {
        console.error(e)
      }
    }
  }
}

// Create a singleton test env that can be imported into each
// test file. It is important that the async operations are only
// called once as they take awhile. Each test file should be sure
// to call `testEnv.init()` in a `before` clause to ensure that
// the files are read from disk at least once
let testEnv: TestEnv
try {
  if (testEnv === undefined) {
    const cfg = config()
    testEnv = new TestEnv(cfg)
  }
} catch (e) {
  console.error(`unable to initialize test env: ${e.toString()}`)
}

export { should, expect, Mocha, testEnv }
