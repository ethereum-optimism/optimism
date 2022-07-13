/* Imports: External */
import { BigNumber } from 'ethers'
import { TransactionEnqueuedEvent } from '@eth-optimism/contracts/dist/types/contracts/L1/rollup/CanonicalTransactionChain'

/* Imports: Internal */
import { MissingElementError } from './errors'
import { EnqueueEntry, EventHandlerSet } from '../../../types'

export const handleEventsTransactionEnqueued: EventHandlerSet<
  TransactionEnqueuedEvent,
  null,
  EnqueueEntry
> = {
  getExtraData: async () => {
    return null
  },
  parseEvent: (event) => {
    return {
      index: event.args._queueIndex.toNumber(),
      target: event.args._target,
      data: event.args._data,
      gasLimit: event.args._gasLimit.toString(),
      origin: event.args._l1TxOrigin,
      blockNumber: BigNumber.from(event.blockNumber).toNumber(),
      timestamp: event.args._timestamp.toNumber(),
      ctcIndex: null,
    }
  },
  storeEvent: async (entry, db) => {
    // Defend against situations where we missed an event because the RPC provider
    // (infura/alchemy/whatever) is missing an event.
    if (entry.index > 0) {
      const prevEnqueueEntry = await db.getEnqueueByIndex(entry.index - 1)

      // We should *alwaus* have a previous enqueue entry here.
      if (prevEnqueueEntry === null) {
        throw new MissingElementError('TransactionEnqueued')
      }
    }

    await db.putEnqueueEntries([entry])
  },
}
