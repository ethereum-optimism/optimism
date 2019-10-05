import { Address, State } from './types'
import { PIGI_TOKEN_TYPE, UNI_TOKEN_TYPE, UNISWAP_ADDRESS } from './utils'

export * from './aggregator'
export * from './rollup-block-submitter'
export * from './rollup-client'
export * from './rollup-state-solver'
export * from './rollup-state-machine'
export * from './serialization/'
export * from './types/'
export * from './unipig-transitioner'
export * from './utils'
export * from './validator'

/* Aggregator API */
export const AGGREGATOR_API = {
  getState: 'getState',
  getUniswapState: 'getUniswapBalances',
  applyTransaction: 'applyTransaction',
  requestFaucetFunds: 'requestFaucetFunds',
  getTransactionCount: 'getTxCount',
}

/* Set the initial balances/state */
export const getGenesisState = (aggregatorAddress: Address): State[] => {
  return [
    {
      pubkey: UNISWAP_ADDRESS,
      balances: {
        [UNI_TOKEN_TYPE]: 1_000_000_000,
        [PIGI_TOKEN_TYPE]: 1_000_000_000,
      },
    },
    {
      pubkey: aggregatorAddress,
      balances: {
        [UNI_TOKEN_TYPE]: 500_000,
        [PIGI_TOKEN_TYPE]: 500_000,
      },
    },
  ]
}
