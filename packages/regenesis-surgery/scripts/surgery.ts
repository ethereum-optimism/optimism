import {
  StateDump,
  UniswapPoolData,
  SurgeryDataSources,
  EtherscanContract,
} from './types'
import { handlers } from './handlers'
import { classify } from './classifiers'

const main = async () => {
  const dump: StateDump = null as any // TODO
  const genesis: StateDump = null as any // TODO
  const pools: UniswapPoolData[] = null as any // TODO
  const etherscanDump: EtherscanContract[] = null as any // TODO
  const data: SurgeryDataSources = {
    dump,
    genesis,
    pools,
    etherscanDump,
    l1TestnetProvider: null as any, // TODO
    l1MainnetProvider: null as any, // TODO
    l2Provider: null as any, // TODO
  }

  // TODO: Insert any accounts from genesis that aren't in the dump

  const output: StateDump = []
  for (const account of dump) {
    const accountType = classify(account, data)
    const handler = handlers[accountType]
    const newAccount = await handler(account, data)
    if (newAccount !== undefined) {
      output.push(newAccount)
    }
  }
}
