/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

/* Internal Imports */
import { RollupBlock, Transaction } from '../../types'

const log = getLogger('abiEncoders')

export const abiEncodeRollupBlock = (rollupBlock: RollupBlock): string => {
  // TODO: actually ABI encode blocks when they are solidified.
  return ''
}

export const abiEncodeTransaction = (transaction: Transaction): string => {
  // TODO: actually ABI encode transactions when they are solidified
  return ''
}
