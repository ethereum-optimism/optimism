/* Imports: External */
import { getContractFactory } from '@eth-optimism/contracts'
import { EventArgsStateBatchAppended } from '@eth-optimism/core-utils'
import { BigNumber } from 'ethers'

/* Imports: Internal */
import {
  StateRootBatchEntry,
  StateBatchAppendedExtraData,
  StateBatchAppendedParsedEvent,
  StateRootEntry,
  EventHandlerSet,
} from '../../../types'

export const handleEventsStateBatchAppended: EventHandlerSet<
  EventArgsStateBatchAppended,
  StateBatchAppendedExtraData,
  StateBatchAppendedParsedEvent
> = {
  getExtraData: async (event) => {
    const eventBlock = await event.getBlock()
    const l1Transaction = await event.getTransaction()

    return {
      timestamp: eventBlock.timestamp,
      blockNumber: eventBlock.number,
      submitter: l1Transaction.from,
      l1TransactionHash: l1Transaction.hash,
      l1TransactionData: l1Transaction.data,
    }
  },
  parseEvent: (event, extraData) => {
    const stateRoots = getContractFactory(
      'OVM_StateCommitmentChain'
    ).interface.decodeFunctionData(
      'appendStateBatch',
      extraData.l1TransactionData
    )[0]

    const stateRootEntries: StateRootEntry[] = []
    for (let i = 0; i < stateRoots.length; i++) {
      stateRootEntries.push({
        index: event.args._prevTotalElements.add(BigNumber.from(i)).toNumber(),
        batchIndex: event.args._batchIndex.toNumber(),
        value: stateRoots[i],
        confirmed: true,
      })
    }

    // Using .toNumber() here and in other places because I want to move everything to use
    // BigNumber + hex, but that'll take a lot of work. This makes it easier in the future.
    const stateRootBatchEntry: StateRootBatchEntry = {
      index: event.args._batchIndex.toNumber(),
      blockNumber: BigNumber.from(extraData.blockNumber).toNumber(),
      timestamp: BigNumber.from(extraData.timestamp).toNumber(),
      submitter: extraData.submitter,
      size: event.args._batchSize.toNumber(),
      root: event.args._batchRoot,
      prevTotalElements: event.args._prevTotalElements.toNumber(),
      extraData: event.args._extraData,
      l1TransactionHash: extraData.l1TransactionHash,
    }

    return {
      stateRootBatchEntry,
      stateRootEntries,
    }
  },
  storeEvent: async (entry, db) => {
    await db.putStateRootBatchEntries([entry.stateRootBatchEntry])
    await db.putStateRootEntries(entry.stateRootEntries)
  },
}
