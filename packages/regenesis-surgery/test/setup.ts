/* External Imports */
import chai = require('chai')
import Mocha from 'mocha'
import chaiAsPromised from 'chai-as-promised'
import * as dotenv from 'dotenv'
import { getenv, remove0x } from '@eth-optimism/core-utils'
import { providers, BigNumber } from 'ethers'
import { solidity } from 'ethereum-waffle'
import { SurgeryDataSources, Account, AccountType } from '../scripts/types'
import { loadSurgeryData } from '../scripts/data'
import { classify, classifiers } from '../scripts/classifiers'
import { GenesisJsonProvider } from './provider'

// Chai plugins go here.
chai.use(chaiAsPromised)
chai.use(solidity)

const should = chai.should()
const expect = chai.expect

dotenv.config()

export const NUM_ACCOUNTS_DIVISOR = 4096
export const ERC20_ABI = [
  'function balanceOf(address owner) view returns (uint256)',
]

interface TestEnvConfig {
  preL2ProviderUrl: string | null
  postL2ProviderUrl: string | null
  postSurgeryGenesisFilePath: string
  stateDumpHeight: string | number
}

const config = (): TestEnvConfig => {
  const height = getenv('REGEN__STATE_DUMP_HEIGHT')
  return {
    // Optional config params for running against live nodes
    preL2ProviderUrl: getenv('REGEN__PRE_L2_PROVIDER_URL'),
    postL2ProviderUrl: getenv('REGEN__POST_L2_PROVIDER_URL'),
    // File path to the post regenesis file to read
    postSurgeryGenesisFilePath: getenv('REGEN__POST_GENESIS_FILE_PATH'),
    stateDumpHeight: parseInt(height, 10) || 'latest',
  }
}

interface TypedAccount extends Account {
  type: AccountType
}

// A TestEnv that contains all of the required test data
class TestEnv {
  // Config
  config: TestEnvConfig

  // An L2 provider configured to be able to query a pre
  // regenesis L2 node. This node should be synced to the
  // height that the state dump was taken
  preL2Provider: providers.StaticJsonRpcProvider | GenesisJsonProvider

  // An L2 provider configured to be able to query a post
  // regenesis L2 node. This L2 node was initialized with
  // the results of the state surgery script
  postL2Provider: providers.StaticJsonRpcProvider | GenesisJsonProvider

  // The datasources used for doing state surgery
  surgeryDataSources: SurgeryDataSources

  // List of typed accounts in the input dump
  accounts: TypedAccount[] = []

  // List of erc20 contracts in input dump
  erc20s: Account[] = []

  constructor(opts: TestEnvConfig) {
    this.config = opts
    // If the pre provider url is provided, use a json rpc provider.
    // Otherwise, initialize a preL2Provider in the init function
    // since it depends on suregery data sources
    if (opts.preL2ProviderUrl) {
      this.preL2Provider = new providers.StaticJsonRpcProvider(
        opts.preL2ProviderUrl
      )
    }
    if (opts.postL2ProviderUrl) {
      this.postL2Provider = new providers.StaticJsonRpcProvider(
        opts.postL2ProviderUrl
      )
    } else {
      if (!opts.postSurgeryGenesisFilePath) {
        throw new Error('Must configure REGEN__POST_GENESIS_FILE_PATH')
      }
      console.log('Using GenesisJsonProvider for postL2Provider')
      this.postL2Provider = new GenesisJsonProvider(
        opts.postSurgeryGenesisFilePath
      )
    }
  }

  // Read the big files from disk. Without bumping the size of the nodejs heap,
  // this can oom the process. Prefix the test command with:
  // $ NODE_OPTIONS=--max_old_space=8912
  async init() {
    if (this.surgeryDataSources === undefined) {
      this.surgeryDataSources = await loadSurgeryData()

      if (!this.preL2Provider) {
        console.log('Initializing pre GenesisJsonProvider...')
        // Convert the genesis dump into a genesis file format
        const genesis = { ...this.surgeryDataSources.genesis }
        for (const account of this.surgeryDataSources.dump) {
          let nonce = account.nonce
          if (typeof nonce === 'string') {
            if (nonce === '') {
              nonce = 0
            } else {
              nonce = BigNumber.from(nonce).toNumber()
            }
          }
          genesis.alloc[remove0x(account.address).toLowerCase()] = {
            nonce,
            balance: account.balance,
            codeHash: remove0x(account.codeHash),
            root: remove0x(account.root),
            code: remove0x(account.code),
            storage: {},
          }
          // Fill in the storage if it exists
          if (account.storage) {
            for (const [key, value] of Object.entries(account.storage)) {
              genesis.alloc[remove0x(account.address).toLowerCase()].storage[
                remove0x(key)
              ] = remove0x(value)
            }
          }
        }
        // Create the pre L2 provider using the build genesis object
        this.preL2Provider = new GenesisJsonProvider(genesis)
      }

      // Classify the accounts once, this takes a while so it's better to cache it.
      console.log(`Classifying accounts...`)
      for (const account of this.surgeryDataSources.dump) {
        const accountType = classify(account, this.surgeryDataSources)
        this.accounts.push({
          ...account,
          type: accountType,
        })

        if (classifiers[AccountType.ERC20](account, this.surgeryDataSources)) {
          this.erc20s.push(account)
        }
      }
    }
  }

  // isProvider is false when it is not live
  hasLiveProviders(): boolean {
    return this.postL2Provider._isProvider
  }

  getAccountsByType(type: AccountType) {
    return this.accounts.filter((account) => account.type === type)
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
