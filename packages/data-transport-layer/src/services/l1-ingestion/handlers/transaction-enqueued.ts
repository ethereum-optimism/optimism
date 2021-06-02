import { EventArgsTransactionEnqueued } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { BigNumber } from 'ethers'
import { EnqueueEntry, EventHandlerSet } from '../../../types'

export const handleEventsTransactionEnqueued: EventHandlerSet<
  EventArgsTransactionEnqueued,
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
    await db.putEnqueueEntries([entry])
  },
}
