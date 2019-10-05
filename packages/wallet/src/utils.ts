import { Address, TokenType, RollupTransaction } from './index'
import { NULL_ADDRESS, SignatureProvider, SignatureVerifier } from '@pigi/core'

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
