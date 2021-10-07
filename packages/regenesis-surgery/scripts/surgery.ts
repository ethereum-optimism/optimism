import { parseChunked } from '@discoveryjs/json-ext'
import { createReadStream } from 'fs'
import { ethers } from 'ethers'
import {
  StateDump,
  UniswapPoolData,
  SurgeryDataSources,
  EtherscanContract,
  SurgeryConfigs,
} from './types'
import { loadConfigs, checkStateDump, readDumpFile, clone, findAccount } from './utils'
import { handlers } from './handlers'
import { classify } from './classifiers'
import { downloadAllSolcVersions } from './download-solc'
import { getUniswapPoolData } from './data'

const doGenesisSurgery = async (
  data: SurgeryDataSources
): Promise<StateDump> => {
  // We'll generate the final genesis file from this output.
  const output: StateDump = []

  // Handle each account in the state dump.
  for (const account of data.dump) {
    const accountType = classify(account, data)
    const handler = handlers[accountType]
    const newAccount = await handler(clone(account), data)
    if (newAccount !== undefined) {
      output.push(newAccount)
    }
  }

  // Injest any accounts in the genesis that aren't already in the state dump.
  for (const account of data.genesis) {
    if (findAccount(data.dump, account.address) === undefined) {
      output.push(account)
    }
  }

  return output
}

const main = async () => {
  await downloadAllSolcVersions()

  const configs: SurgeryConfigs = loadConfigs()

  const dump: StateDump = await readDumpFile(configs.stateDumpFilePath)
  // Validate state dump
  checkStateDump(dump)
  const genesis: StateDump = null as any
  const etherscanDump: EtherscanContract[] = await parseChunked(
    createReadStream(configs.etherscanFilePath)
  )

  const l1TestnetProvider = new ethers.providers.JsonRpcProvider(
    configs.l1TestnetProviderUrl
  )
  const l2Provider = new ethers.providers.JsonRpcProvider(configs.l2ProviderUrl)
  const pools: UniswapPoolData[] = await getUniswapPoolData(
    l2Provider,
    configs.l2NetworkName
  )
  const data: SurgeryDataSources = {
    dump,
    genesis,
    pools,
    etherscanDump,
    l1TestnetProvider,
    l1TestnetWallet: new ethers.Wallet(
      configs.l1TestnetPrivateKey,
      l1TestnetProvider
    ),
    l1MainnetProvider: new ethers.providers.JsonRpcProvider(
      configs.l1MainnetProviderUrl
    ),
    l2Provider,
  }

  const nextGenesis = await doGenesisSurgery(data)
}

main()
