import { ethers } from 'ethers'
import fs from 'fs'
import {
  StateDump,
  UniswapPoolData,
  SurgeryDataSources,
  EtherscanContract,
  SurgeryConfigs,
  GenesisFile,
  AccountType,
} from './types'
import {
  loadConfigs,
  checkStateDump,
  readDumpFile,
  readEtherscanFile,
  readGenesisFile,
  clone,
  findAccount,
} from './utils'
import { handlers } from './handlers'
import { classify } from './classifiers'
import { downloadAllSolcVersions } from './download-solc'
import { getUniswapPoolData } from './data'

const doGenesisSurgery = async (
  data: SurgeryDataSources
): Promise<StateDump> => {
  // We'll generate the final genesis file from this output.
  const output: StateDump = []

  const size = data.dump.length
  // Handle each account in the state dump.
  for (const [i, account] of data.dump.entries()) {
    if (i >= data.startIndex && i <= data.endIndex) {
      const accountType = classify(account, data)
      const handler = handlers[accountType]
      console.log(
        `${i}/${size} - Handling type ${AccountType[accountType]} - ${account.address} `
      )
      const newAccount = await handler(clone(account), data)
      if (newAccount !== undefined) {
        output.push(newAccount)
      }
    }
  }

  // Injest any accounts in the genesis that aren't already in the state dump.
  // TODO: this needs to be able to be deduplicated if running in parallel
  for (const account of data.genesis) {
    if (findAccount(data.dump, account.address) === undefined) {
      output.push(account)
    }
  }

  return output
}

const main = async () => {
  // First download every solc version that we'll need during this surgery.
  await downloadAllSolcVersions()

  // Load the configuration values, will throw if anything is missing.
  const configs: SurgeryConfigs = loadConfigs()

  // Load and validate the state dump.
  const dump: StateDump = await readDumpFile(configs.stateDumpFilePath)
  checkStateDump(dump)

  // Load the genesis file.
  const genesis: GenesisFile = await readGenesisFile(configs.genesisFilePath)
  const genesisDump: StateDump = []
  for (const [address, account] of Object.entries(genesis.alloc)) {
    genesisDump.push({
      address,
      ...account,
    })
  }

  // Load the etherscan dump.
  const etherscanDump: EtherscanContract[] = await readEtherscanFile(
    configs.etherscanFilePath
  )

  // Get a reference to the L2 provider and load all revelant pool data.
  const l2Provider = new ethers.providers.JsonRpcProvider(configs.l2ProviderUrl)
  const pools: UniswapPoolData[] = await getUniswapPoolData(
    l2Provider,
    configs.l2NetworkName
  )

  // Get a reference to the L1 testnet provider and wallet, used for deploying Uniswap pools.
  const l1TestnetProvider = new ethers.providers.JsonRpcProvider(
    configs.l1TestnetProviderUrl
  )
  const l1TestnetWallet = new ethers.Wallet(
    configs.l1TestnetPrivateKey,
    l1TestnetProvider
  )

  // Get a reference to the L1 mainnet provider.
  const l1MainnetProvider = new ethers.providers.JsonRpcProvider(
    configs.l1MainnetProviderUrl
  )

  // Do the surgery process and get the new genesis dump.
  const finalGenesisDump = await doGenesisSurgery({
    dump,
    genesis: genesisDump,
    pools,
    etherscanDump,
    l1TestnetProvider,
    l1TestnetWallet,
    l1MainnetProvider,
    l2Provider,
    l2NetworkName: configs.l2NetworkName,
    startIndex: configs.startIndex,
    endIndex: configs.endIndex,
  })

  // Convert to the format that Geth expects.
  const finalGenesisAlloc = {}
  for (const account of finalGenesisDump) {
    const address = account.address
    delete account.address
    finalGenesisAlloc[address] = account
  }

  // Attach all of the original genesis configuration values.
  const finalGenesis = {
    ...genesis,
    alloc: finalGenesisAlloc,
  }

  // Write the final genesis file to disk.
  fs.writeFileSync(
    configs.outputFilePath,
    JSON.stringify(finalGenesis, null, 2)
  )
}

main()
