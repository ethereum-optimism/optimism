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
  console.log('Downloading all required solc versions...')
  await downloadAllSolcVersions()

  // Load the configuration values, will throw if anything is missing.
  console.log('Loading configuration values...')
  const configs: SurgeryConfigs = loadConfigs()

  // Load and validate the state dump.
  console.log('Loading and validating state dump file...')
  const dump: StateDump = await readDumpFile(configs.stateDumpFilePath)
  checkStateDump(dump)

  // Load the genesis file.
  console.log('Loading genesis file...')
  const genesis: GenesisFile = await readGenesisFile(configs.genesisFilePath)
  const genesisDump: StateDump = []
  for (const [address, account] of Object.entries(genesis.alloc)) {
    genesisDump.push({
      address,
      ...account,
    })
  }

  // Load the etherscan dump.
  console.log('Loading etherscan dump file...')
  const etherscanDump: EtherscanContract[] = await readEtherscanFile(
    configs.etherscanFilePath
  )

  // Get a reference to the L2 provider so we can load pool data.
  console.log('Connecting to L2 provider...')
  const l2Provider = new ethers.providers.JsonRpcProvider(configs.l2ProviderUrl)

  // Load the pool data.
  console.log('Loading Uniswap pool data...')
  const pools: UniswapPoolData[] = await getUniswapPoolData(
    l2Provider,
    configs.l2NetworkName
  )

  // Get a reference to the L1 testnet provider and wallet, used for deploying Uniswap pools.
  console.log('Connecting to L1 testnet provider...')
  const l1TestnetProvider = new ethers.providers.JsonRpcProvider(
    configs.l1TestnetProviderUrl
  )
  const l1TestnetWallet = new ethers.Wallet(
    configs.l1TestnetPrivateKey,
    l1TestnetProvider
  )

  // Get a reference to the L1 mainnet provider.
  console.log('Connecting to L1 mainnet provider...')
  const l1MainnetProvider = new ethers.providers.JsonRpcProvider(
    configs.l1MainnetProviderUrl
  )

  // Do the surgery process and get the new genesis dump.
  console.log('Starting surgery process...')
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
  console.log('Converting dump to final format...')
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
  // TODO: This WILL break because the genesis file will be larger than the allowable string size.
  // We'll need to write it in chunks instead. Not sure of the best way to achieve this.
  console.log('Writing final genesis to disk...')
  fs.writeFileSync(
    configs.outputFilePath,
    JSON.stringify(finalGenesis, null, 2)
  )

  console.log('All done!')
}

main()
