/* External Imports */
import { getLogger } from '@pigi/core-utils'

/* Internal imports */
import { RollupBlock } from '../../types'

const log = getLogger('abiEncoders')

export const abiDecodeRollupBlock = (abiEncoded: string): RollupBlock => {
  // TODO: actually fill this out
  return {
    blockNumber: 1,
    stateRoot: '',
    transactions: [],
  }
}
