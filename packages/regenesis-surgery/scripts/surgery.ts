import { StateDump, SurgeryDataSources } from './types'
import { handlers } from './handlers'
import { classify } from './classifiers'
import { clone, findAccount } from './utils'
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

  const network = 'kovan' // TODO: make configurable
  const l2Provider = null as any // TODO
  const pools = await getUniswapPoolData(l2Provider, network)
  const genesis = await doGenesisSurgery({} as any)
}

main()
