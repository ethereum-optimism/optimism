import { Address, TokenType, RollupTransaction, State } from '../index'
import { NULL_ADDRESS } from '@pigi/core-utils'

/* Constants */
export const UNISWAP_ADDRESS = NULL_ADDRESS
export const UNISWAP_STORAGE_SLOT = 0
export const UNI_TOKEN_TYPE = 0
export const PIGI_TOKEN_TYPE = 1

export const NON_EXISTENT_SLOT_INDEX = -1
export const EMPTY_AGGREGATOR_SIGNATURE = 'THIS IS EMPTY'

/* Utilities */
export const generateTransferTx = (
  sender: Address,
  recipient: Address,
  tokenType: TokenType,
  amount: number
): RollupTransaction => {
  return {
    sender,
    tokenType,
    recipient,
    amount,
  }
}

/* Aggregator API */
export const AGGREGATOR_API = {
  getState: 'getState',
  getUniswapState: 'getUniswapBalances',
  applyTransaction: 'applyTransaction',
  requestFaucetFunds: 'requestFaucetFunds',
  getTransactionCount: 'getTxCount',
}

/* Set the initial balances/state */
export const getGenesisState = (
  aggregatorAddress: Address,
  genesisState?: State[]
): State[] => {
  if (!genesisState) {
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

  if (genesisState[0].pubkey === 'UNISWAP_ADDRESS') {
    genesisState[0].pubkey = UNISWAP_ADDRESS
  }
  if (genesisState[1].pubkey === 'AGGREGATOR_ADDRESS') {
    genesisState[1].pubkey = aggregatorAddress
  }

  return genesisState
}
