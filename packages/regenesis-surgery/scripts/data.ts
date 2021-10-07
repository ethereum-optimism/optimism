import { ethers } from 'ethers'
import {
  computePoolAddress,
  POOL_INIT_CODE_HASH,
  POOL_INIT_CODE_HASH_OPTIMISM,
  POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
} from '@uniswap/v3-sdk'
import { Token } from '@uniswap/sdk-core'
import { UniswapPoolData } from './types'
import { getUniswapV3Factory } from './utils'
import { UNISWAP_V3_FACTORY_ADDRESS } from './constants'

export const getUniswapPoolData = async (
  l2Provider: ethers.providers.BaseProvider,
  network: 'mainnet' | 'kovan'
): Promise<UniswapPoolData[]> => {
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
