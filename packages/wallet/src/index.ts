export * from './rollup-aggregator'
export * from './rollup-client'
export * from './rollup-state-machine'
export * from './unipig-wallet'
export * from './types/'
export * from './utils'

/* Constants */
export const AGGREGATOR_ADDRESS = '0xAc001762c6424F4959852A516368DBf970C835a7'
export const UNISWAP_ADDRESS = '0x' + 'ff'.repeat(32)
export const UNI_TOKEN_TYPE = 'uni'
export const PIGI_TOKEN_TYPE = 'pigi'

/* Aggregator API */
export const AGGREGATOR_API = {
  getState: 'getState',
  getUniswapState: 'getUniswapBalances',
  applyTransaction: 'applyTransaction',
  requestFaucetFunds: 'requestFaucetFunds',
}
