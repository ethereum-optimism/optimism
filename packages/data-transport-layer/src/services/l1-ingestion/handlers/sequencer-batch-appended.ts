/* Imports: External */
import { BigNumber, ethers, constants } from 'ethers'
import { serialize, Transaction } from '@ethersproject/transactions'
import { getContractFactory } from '@eth-optimism/contracts'
import {
  toHexString,
  toRpcHexString,
  BatchType,
  SequencerBatch,
} from '@eth-optimism/core-utils'
import { SequencerBatchAppendedEvent } from '@eth-optimism/contracts/dist/types/contracts/L1/rollup/CanonicalTransactionChain'

/* Imports: Internal */
import { MissingElementError } from './errors'
import {
  DecodedSequencerBatchTransaction,
  SequencerBatchAppendedExtraData,
  SequencerBatchAppendedParsedEvent,
  TransactionBatchEntry,
  TransactionEntry,
  EventHandlerSet,
} from '../../../types'
import { parseSignatureVParam } from '../../../utils'

export const handleEventsSequencerBatchAppended: EventHandlerSet<
  SequencerBatchAppendedEvent,
  SequencerBatchAppendedExtraData,
  SequencerBatchAppendedParsedEvent
> = {
  getExtraData: async (event, l1RpcProvider) => {
    const l1Transaction = await event.getTransaction()
    const eventBlock = await event.getBlock()

    // TODO: We need to update our events so that we actually have enough information to parse this
    // batch without having to pull out this extra event. For the meantime, we need to find this
    // "TransactonBatchAppended" event to get the rest of the data.
    const CanonicalTransactionChain = getContractFactory(
      'CanonicalTransactionChain'
    )
      .attach(event.address)
      .connect(l1RpcProvider)

    const batchSubmissionEvent = (
      await CanonicalTransactionChain.queryFilter(
        CanonicalTransactionChain.filters.TransactionBatchAppended(),
        eventBlock.number,
        eventBlock.number
      )
    ).find((foundEvent: ethers.Event) => {
      // We might have more than one event in this block, so we specifically want to find a
      // "TransactonBatchAppended" event emitted immediately before the event in question.
      return (
        foundEvent.transactionHash === event.transactionHash &&
        foundEvent.logIndex === event.logIndex - 1
      )
    })

    if (!batchSubmissionEvent) {
      throw new Error(
        `Well, this really shouldn't happen. A SequencerBatchAppended event doesn't have a corresponding TransactionBatchAppended event.`
      )
    }

    return {
      timestamp: eventBlock.timestamp,
      blockNumber: eventBlock.number,
      submitter: l1Transaction.from,
      l1TransactionHash: l1Transaction.hash,
      l1TransactionData: l1Transaction.data,

      prevTotalElements: batchSubmissionEvent.args._prevTotalElements,
      batchIndex: batchSubmissionEvent.args._batchIndex,
      batchSize: batchSubmissionEvent.args._batchSize,
      batchRoot: batchSubmissionEvent.args._batchRoot,
      batchExtraData: batchSubmissionEvent.args._extraData,
    }
  },
  parseEvent: (event, extraData, l2ChainId) => {
    const transactionEntries: TransactionEntry[] = []

    // 12 * 2 + 2 = 26
    if (extraData.l1TransactionData.length < 26) {
      throw new Error(
        `Block ${extraData.blockNumber} transaction data is too small: ${extraData.l1TransactionData.length}`
      )
    }

    // TODO: typings not working?
    const decoded = (SequencerBatch as any).fromHex(extraData.l1TransactionData)

    // Keep track of the CTC index
    let transactionIndex = 0
    // Keep track of the number of deposits
    let enqueuedCount = 0
    // Keep track of the tx index in the current batch
    let index = 0

    for (const context of decoded.contexts) {
      for (let j = 0; j < context.numSequencedTransactions; j++) {
        const buf = decoded.transactions[index]
        if (!buf) {
          throw new Error(
            `Invalid batch context, tx count: ${decoded.transactions.length}, attempting to parse ${index}`
          )
        }

        const tx = buf.toTransaction()

        transactionEntries.push({
          index: extraData.prevTotalElements
            .add(BigNumber.from(transactionIndex))
            .toNumber(),
          batchIndex: extraData.batchIndex.toNumber(),
          blockNumber: BigNumber.from(context.blockNumber).toNumber(),
          timestamp: BigNumber.from(context.timestamp).toNumber(),
          gasLimit: BigNumber.from(0).toString(),
          target: constants.AddressZero,
          origin: null,
          data: serialize(
            {
              nonce: tx.nonce,
              gasPrice: tx.gasPrice,
              gasLimit: tx.gasLimit,
              to: tx.to,
              value: tx.value,
              data: tx.data,
            },
            {
              v: tx.v,
              r: tx.r,
              s: tx.s,
            }
          ),
          queueOrigin: 'sequencer',
          value: toRpcHexString(tx.value),
          queueIndex: null,
          decoded: mapSequencerTransaction(tx, l2ChainId),
          confirmed: true,
        })
        transactionIndex++
        index++
      }

      for (let j = 0; j < context.numSubsequentQueueTransactions; j++) {
        const queueIndex = event.args._startingQueueIndex.add(
          BigNumber.from(enqueuedCount)
        )

        // Okay, so. Since events are processed in parallel, we don't know if the Enqueue
        // event associated with this queue element has already been processed. So we'll ask
        // the api to fetch that data for itself later on and we use fake values for some
        // fields. The real TODO here is to make sure we fix this data structure to avoid ugly
        // "dummy" fields.
        transactionEntries.push({
          index: extraData.prevTotalElements
            .add(BigNumber.from(transactionIndex))
            .toNumber(),
          batchIndex: extraData.batchIndex.toNumber(),
          blockNumber: BigNumber.from(0).toNumber(),
          timestamp: context.timestamp,
          gasLimit: BigNumber.from(0).toString(),
          target: constants.AddressZero,
          origin: constants.AddressZero,
          data: '0x',
          queueOrigin: 'l1',
          value: '0x0',
          queueIndex: queueIndex.toNumber(),
          decoded: null,
          confirmed: true,
        })

        enqueuedCount++
        transactionIndex++
      }
    }

    const transactionBatchEntry: TransactionBatchEntry = {
      index: extraData.batchIndex.toNumber(),
      root: extraData.batchRoot,
      size: extraData.batchSize.toNumber(),
      prevTotalElements: extraData.prevTotalElements.toNumber(),
      extraData: extraData.batchExtraData,
      blockNumber: BigNumber.from(extraData.blockNumber).toNumber(),
      timestamp: BigNumber.from(extraData.timestamp).toNumber(),
      submitter: extraData.submitter,
      l1TransactionHash: extraData.l1TransactionHash,
      type: BatchType[decoded.type],
    }

    return {
      transactionBatchEntry,
      transactionEntries,
    }
  },
  storeEvent: async (entry, db) => {
    // Defend against situations where we missed an event because the RPC provider
    // (infura/alchemy/whatever) is missing an event.
    if (entry.transactionBatchEntry.index > 0) {
      const prevTransactionBatchEntry = await db.getTransactionBatchByIndex(
        entry.transactionBatchEntry.index - 1
      )

      // We should *always* have a previous transaction batch here.
      if (prevTransactionBatchEntry === null) {
        throw new MissingElementError('SequencerBatchAppended')
      }
    }

    // Same consistency checks but for transaction entries.
    if (
      entry.transactionEntries.length > 0 &&
      entry.transactionEntries[0].index > 0
    ) {
      const prevTransactionEntry = await db.getTransactionByIndex(
        entry.transactionEntries[0].index - 1
      )

      // We should *always* have a previous transaction here.
      if (prevTransactionEntry === null) {
        throw new MissingElementError('SequencerBatchAppendedTransaction')
      }
    }

    await db.putTransactionEntries(entry.transactionEntries)

    // Add an additional field to the enqueued transactions in the database
    // if they have already been confirmed
    for (const transactionEntry of entry.transactionEntries) {
      if (transactionEntry.queueOrigin === 'l1') {
        await db.putTransactionIndexByQueueIndex(
          transactionEntry.queueIndex,
          transactionEntry.index
        )
      }
    }

    await db.putTransactionBatchEntries([entry.transactionBatchEntry])
  },
}

const mapSequencerTransaction = (
  tx: Transaction,
  l2ChainId: number
): DecodedSequencerBatchTransaction => {
  return {
    nonce: BigNumber.from(tx.nonce).toString(),
    gasPrice: BigNumber.from(tx.gasPrice).toString(),
    gasLimit: BigNumber.from(tx.gasLimit).toString(),
    value: toRpcHexString(tx.value),
    target: tx.to ? toHexString(tx.to) : null,
    data: toHexString(tx.data),
    sig: {
      v: parseSignatureVParam(tx.v, l2ChainId),
      r: toHexString(tx.r),
      s: toHexString(tx.s),
    },
  }
}
