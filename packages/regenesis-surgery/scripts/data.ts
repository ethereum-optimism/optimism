import { ethers } from 'ethers'
import {
  computePoolAddress,
  POOL_INIT_CODE_HASH,
  POOL_INIT_CODE_HASH_OPTIMISM,
  POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
} from '@uniswap/v3-sdk'
import { Token } from '@uniswap/sdk-core'

import { UNISWAP_V3_FACTORY_ADDRESS } from './constants'
import { downloadAllSolcVersions } from './solc'
import {
  PoolHashCache,
  StateDump,
  UniswapPoolData,
  SurgeryDataSources,
  EtherscanContract,
  SurgeryConfigs,
  GenesisFile,
} from './types'
import {
  loadConfigs,
  checkStateDump,
  readDumpFile,
  readEtherscanFile,
  readGenesisFile,
  getUniswapV3Factory,
  getMappingKey,
} from './utils'

export const getUniswapPoolData = async (
  l2Provider: ethers.providers.BaseProvider,
  network: 'mainnet' | 'kovan'
): Promise<UniswapPoolData[]> => {
  if (!network) {
    throw new Error('Must provide network "mainnet" or "kovan"')
  }
  const UniswapV3Factory = getUniswapV3Factory(l2Provider)

  const pools: UniswapPoolData[] = []
  const poolEvents = await UniswapV3Factory.queryFilter('PoolCreated' as any)
  for (const event of poolEvents) {
    // Compute the old pool address using the OVM init code hash.
    const oldPoolAddress = computePoolAddress({
      factoryAddress: UNISWAP_V3_FACTORY_ADDRESS,
      tokenA: new Token(0, event.args.token0, 18),
      tokenB: new Token(0, event.args.token1, 18),
      fee: event.args.fee,
      initCodeHashManualOverride:
        network === 'mainnet'
          ? POOL_INIT_CODE_HASH_OPTIMISM
          : POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
    }).toLowerCase()

    // Compute the new pool address using the EVM init code hash.
    const newPoolAddress = computePoolAddress({
      factoryAddress: UNISWAP_V3_FACTORY_ADDRESS,
      tokenA: new Token(0, event.args.token0, 18),
      tokenB: new Token(0, event.args.token1, 18),
      fee: event.args.fee,
      initCodeHashManualOverride: POOL_INIT_CODE_HASH,
    }).toLowerCase()

    pools.push({
      oldAddress: oldPoolAddress,
      newAddress: newPoolAddress,
      token0: event.args.token0,
      token1: event.args.token1,
      fee: event.args.fee,
    })
  }

  return pools
}

export const makePoolHashCache = (pools: UniswapPoolData[]): PoolHashCache => {
  const cache: PoolHashCache = {}
  for (const pool of pools) {
    for (let i = 0; i < 1000; i++) {
      cache[getMappingKey([pool.oldAddress], i)] = {
        pool,
        index: i,
      }
    }
  }
  return cache
}

const getChainId = async (
  provider: ethers.providers.JsonRpcProvider
): Promise<number> => {
  const response = await provider.send('eth_chainId', [])
  return ethers.BigNumber.from(response).toNumber()
}

export const loadSurgeryData = async (
  configs?: SurgeryConfigs
): Promise<SurgeryDataSources> => {
  // First download every solc version that we'll need during this surgery.
  console.log('Downloading all required solc versions...')
  await downloadAllSolcVersions()

  // Load the configuration values, will throw if anything is missing.
  if (configs === undefined) {
    console.log('Loading configuration values...')
    configs = loadConfigs()
  }

  // Get a reference to an ETH (mainnet) provider.
  console.log('Connecting to ETH provider...')
  const ethProvider = new ethers.providers.JsonRpcProvider(
    configs.ethProviderUrl
  )
  const mainnetChainId = await getChainId(ethProvider)
  if (mainnetChainId !== 1) {
    throw new Error(
      `Mainnet chain id incorrect, got ${mainnetChainId} and expected 1`
    )
  }

  // Get a reference to the L2 provider so we can load pool data.
  // Do validation on the chain id before reading data from disk
  // because that is slow
  console.log('Connecting to L2 provider...')
  const l2Provider = new ethers.providers.JsonRpcProvider(configs.l2ProviderUrl)
  const l2ChainId = await getChainId(l2Provider)
  if (l2ChainId === 10) {
    configs.l2NetworkName = 'mainnet'
  } else if (l2ChainId === 69) {
    configs.l2NetworkName = 'kovan'
  } else {
    throw new Error(`Unknown l2 chain id: ${l2ChainId}`)
  }
  console.log(`Using network ${configs.l2NetworkName}`)

  // Load and validate the state dump.
  console.log('Loading and validating state dump file...')
  const dump: StateDump = await readDumpFile(configs.stateDumpFilePath)
  checkStateDump(dump)
  console.log(`${dump.length} entries in state dump`)

  // Load the genesis file.
  console.log('Loading genesis file...')
  const genesis: GenesisFile = await readGenesisFile(configs.genesisFilePath)
  if (genesis.config.chainId !== l2ChainId) {
    // Don't throw here because we might need to do a staging environment with a different chain ID
    console.log(
      `WARNING: Genesis File at ${configs.genesisFilePath} has chain id mismatch with remote L2 node` +
        ` got ${genesis.config.chainId} locally and ${l2ChainId} remotely`
    )
  }
  const genesisDump: StateDump = []
  for (const [address, account] of Object.entries(genesis.alloc)) {
    genesisDump.push({
      address,
      ...account,
    })
  }
  console.log(`${genesisDump.length} entries in genesis file`)

  // Load the etherscan dump.
  console.log('Loading etherscan dump file...')
  const etherscanDump: EtherscanContract[] = await readEtherscanFile(
    configs.etherscanFilePath
  )
  console.log(`${etherscanDump.length} entries in etherscan dump`)

  // Load the pool data.
  console.log('Loading Uniswap pool data...')
  const pools: UniswapPoolData[] = await getUniswapPoolData(
    l2Provider,
    configs.l2NetworkName
  )
  console.log(`${pools.length} uniswap pools`)

  console.log('Generating pool cache...')
  const poolHashCache = makePoolHashCache(pools)

  // Get a reference to the ropsten provider and wallet, used for deploying Uniswap pools.
  console.log('Connecting to ropsten provider...')
  const ropstenProvider = new ethers.providers.JsonRpcProvider(
    configs.ropstenProviderUrl
  )

  const ropstenWallet = new ethers.Wallet(
    configs.ropstenPrivateKey,
    ropstenProvider
  )
  const ropstenChainId = await ropstenWallet.getChainId()
  if (ropstenChainId !== 3) {
    throw new Error(
      `Ropsten chain id incorrect, got ${ropstenChainId} and expected 3`
    )
  }

  return {
    configs,
    dump,
    genesis,
    genesisDump,
    pools,
    poolHashCache,
    etherscanDump,
    ropstenProvider,
    ropstenWallet,
    l2Provider,
    ethProvider,
  }
}
