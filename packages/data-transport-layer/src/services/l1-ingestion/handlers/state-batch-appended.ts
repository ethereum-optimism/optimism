/* Imports: External */
import { StateBatchAppendedEvent } from '@eth-optimism/contracts/dist/types/contracts/L1/rollup/StateCommitmentChain'
import { getContractFactory } from '@eth-optimism/contracts'
import { BigNumber } from 'ethers'

/* Imports: Internal */
import { MissingElementError } from './errors'
import {
  StateRootBatchEntry,
  StateBatchAppendedExtraData,
  StateBatchAppendedParsedEvent,
  StateRootEntry,
  EventHandlerSet,
} from '../../../types'

export const handleEventsStateBatchAppended: EventHandlerSet<
  StateBatchAppendedEvent,
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
      'StateCommitmentChain'
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
      type: 'LEGACY', // There is currently only 1 state root batch type
    }

    return {
      stateRootBatchEntry,
      stateRootEntries,
    }
  },
  storeEvent: async (entry, db) => {
    // Defend against situations where we missed an event because the RPC provider
    // (infura/alchemy/whatever) is missing an event.
    if (entry.stateRootBatchEntry.index > 0) {
      const prevStateRootBatchEntry = await db.getStateRootBatchByIndex(
        entry.stateRootBatchEntry.index - 1
      )

      // We should *always* have a previous batch entry here.
      if (prevStateRootBatchEntry === null) {
        throw new MissingElementError('StateBatchAppended')
      }
    }

    await db.putStateRootBatchEntries([entry.stateRootBatchEntry])
    await db.putStateRootEntries(entry.stateRootEntries)
  },
}
