/* External Imports */
import {
  add0x,
  BigNumber,
  getLogger,
  logError,
  remove0x,
} from '@eth-optimism/core-utils'
import { Log, TransactionResponse } from 'ethers/providers/abstract-provider'
import { ethers } from 'ethers'

/* Internal Imports */
import { L1DataService, QueueOrigin, RollupTransaction } from '../../../types'

const abi = new ethers.utils.AbiCoder()
const log = getLogger('log-handler')

/**
 * Handles the L1ToL2TxEnqueued event by parsing a RollupTransaction
 * from the event data and storing it in the DB.
 *
 * Assumed Log Data Format:
 *  - sender: 20-byte address    0-20
 *  - target: 20-byte address	   20-40
 *  - gasLimit: 32-byte uint 	   40-72
 *  - calldata: bytes            72-end
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const L1ToL2TxEnqueuedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `L1ToL2TxEnqueued event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}. Log Data: ${l.data}`
  )

  const data: string = remove0x(l.data)

  let rollupTransaction: RollupTransaction
  try {
    rollupTransaction = {
      l1BlockNumber: tx.blockNumber,
      l1Timestamp: tx.timestamp,
      l1TxHash: l.transactionHash,
      l1TxIndex: l.transactionIndex,
      l1TxLogIndex: l.transactionLogIndex,
      queueOrigin: QueueOrigin.L1_TO_L2_QUEUE,
      batchIndex: 0,
      sender: l.address,
      l1MessageSender: add0x(data.substr(0, 40)),
      target: add0x(data.substr(40, 40)),
      // TODO: Change gasLimit to a BigNumber so it can support 256 bits
      gasLimit: new BigNumber(data.substr(80, 64), 'hex').toNumber(),
      calldata: add0x(data.substr(144)),
    }
  } catch (e) {
    // This is, by definition, just an ill-formatted, and therefore invalid, tx.
    log.debug(
      `Error parsing calldata tx from CalldataTxEnqueued event. Calldata: ${tx.data}. Error: ${e.message}. Stack: ${e.stack}.`
    )
    return
  }

  await ds.insertL1RollupTransactions(l.transactionHash, [rollupTransaction])
}

/**
 * Handles the CalldataTxEnqueued event by parsing a RollupTransaction
 * from the transaction calldata and storing it in the DB.
 *
 * Assumed calldata format:
 *   - sender: 20-byte address    0-20
 *   - target: 20-byte address	  20-40
 *   - nonce: 32-byte uint 	      40-72
 *   - gasLimit: 32-byte uint	    72-104
 *   - signature: 65-byte bytes   104-169
 *   - calldata: bytes    		    169-end
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const CalldataTxEnqueuedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `CalldataTxEnqueued event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}. Calldata: ${tx.data}`
  )

  let rollupTransaction: RollupTransaction
  try {
    // Skip the 4 bytes of MethodID
    const calldata = remove0x(ethers.utils.hexDataSlice(tx.data, 4))
    rollupTransaction = {
      l1BlockNumber: tx.blockNumber,
      l1Timestamp: tx.timestamp,
      l1TxHash: l.transactionHash,
      l1TxIndex: l.transactionIndex,
      l1TxLogIndex: l.transactionLogIndex,
      queueOrigin: QueueOrigin.SAFETY_QUEUE,
      batchIndex: 0,
      sender: add0x(calldata.substr(0, 40)),
      target: add0x(calldata.substr(40, 40)),
      // TODO Change nonce to a BigNumber so it can support 256 bits
      nonce: new BigNumber(calldata.substr(80, 64), 'hex').toNumber(),
      // TODO: Change gasLimit to a BigNumber so it can support 256 bits
      gasLimit: new BigNumber(calldata.substr(144, 64), 'hex').toNumber(),
      signature: add0x(calldata.substr(208, 130)),
      calldata: add0x(calldata.substr(338)),
    }
  } catch (e) {
    // This is, by definition, just an ill-formatted, and therefore invalid, tx.
    log.debug(
      `Error parsing calldata tx from CalldataTxEnqueued event. Calldata: ${tx.data}. Error: ${e.message}. Stack: ${e.stack}.`
    )
    return
  }

  await ds.insertL1RollupTransactions(l.transactionHash, [rollupTransaction])
}

/**
 * Handles the L1ToL2BatchAppended event by parsing a RollupTransaction
 * from the log event and storing it in the DB.
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const L1ToL2BatchAppendedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `L1ToL2BatchAppended event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}`
  )
  let batchNumber: number
  try {
    batchNumber = await ds.createNextL1ToL2Batch()
  } catch (e) {
    logError(
      log,
      `Error creating next L1ToL2Batch after receiving an event to do so!`,
      e
    )
    throw e
  }

  if (!batchNumber) {
    const msg = `Attempted to create L1 to L2 Batch upon receiving L1ToL2BatchAppended log, but no tx was available for batching!`
    log.error(msg)
    throw Error(msg)
  } else {
    log.debug(
      `Successfully created L1 to L2 Batch! Batch number: ${batchNumber}`
    )
  }
}

/**
 * Handles the SafetyQueueBatchAppended event by parsing a RollupTransaction
 * from the transaction calldata and storing it in the DB.
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const SafetyQueueBatchAppendedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `SafetyQueueBatchAppended event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}`
  )
  let batchNumber: number

  try {
    batchNumber = await ds.createNextSafetyQueueBatch()
  } catch (e) {
    logError(
      log,
      `Error creating next L1ToL2Batch after receiving an event to do so!`,
      e
    )
    throw e
  }

  if (!batchNumber) {
    const msg = `Attempted to create Safety Queue Batch upon receiving L1ToL2BatchAppended log, but no tx was available for batching!`
    log.error(msg)
    throw Error(msg)
  } else {
    log.debug(
      `Successfully created Safety Queue Batch! Batch number: ${batchNumber}`
    )
  }
}

/**
 * Handles the SequencerBatchAppended event by parsing:
 *    - a list of RollupTransactions
 *    - L1 Block Timestamp at the time of L2 Execution
 * from the transaction calldata and storing it in the DB.
 *
 * Assumed calldata format:
 *   - sender: 20-byte address    0-20
 *   - target: 20-byte address	  20-40
 *   - nonce: 32-byte uint 	      40-72
 *   - gasLimit: 32-byte uint	    72-104
 *   - signature: 65-byte bytes   104-169
 *   - calldata: bytes    		    169-end
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const SequencerBatchAppendedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `SequencerBatchAppended event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}. Calldata: ${tx.data}`
  )

  const rollupTransactions: RollupTransaction[] = []
  let timestamp: any
  try {
    let transactionsBytes: string[]
    ;[transactionsBytes, timestamp] = abi.decode(
      ['bytes[]', 'uint256'],
      ethers.utils.hexDataSlice(tx.data, 4)
    )

    for (let i = 0; i < transactionsBytes.length; i++) {
      const txBytes = remove0x(transactionsBytes[i])
      rollupTransactions.push({
        l1BlockNumber: tx.blockNumber,
        l1Timestamp: timestamp.toNumber(),
        l1TxHash: l.transactionHash,
        l1TxIndex: l.transactionIndex,
        l1TxLogIndex: l.transactionLogIndex,
        queueOrigin: QueueOrigin.SEQUENCER,
        batchIndex: i,
        sender: add0x(txBytes.substr(0, 40)),
        target: add0x(txBytes.substr(40, 40)),
        // TODO Change nonce to a BigNumber so it can support 256 bits
        nonce: new BigNumber(txBytes.substr(80, 64), 'hex').toNumber(),
        // TODO: Change gasLimit to a BigNumber so it can support 256 bits
        gasLimit: new BigNumber(txBytes.substr(144, 64), 'hex').toNumber(),
        signature: add0x(txBytes.substr(208, 130)),
        calldata: add0x(txBytes.substr(338)),
      })
    }
  } catch (e) {
    // This is, by definition, just an ill-formatted, and therefore invalid, tx.
    log.debug(
      `Error parsing calldata tx from CalldataTxEnqueued event. Calldata: ${tx.data}. Error: ${e.message}. Stack: ${e.stack}.`
    )
    return
  }

  const batchNumber = await ds.insertL1RollupTransactions(
    l.transactionHash,
    rollupTransactions,
    true
  )
  log.debug(`Sequencer batch number ${batchNumber} successfully created!`)
}

/**
 * Handles the StateBatchAppended event by parsing a batch of state roots
 * from the provided transaction calldata and storing it in the DB.
 *
 * @param ds The L1DataService to use for persistence.
 * @param l The log event that was emitted.
 * @param tx The transaction that emitted the event.
 * @throws Error if there's an error with persistence.
 */
export const StateBatchAppendedLogHandler = async (
  ds: L1DataService,
  l: Log,
  tx: TransactionResponse
): Promise<void> => {
  log.debug(
    `StateBatchAppended event received at block ${tx.blockNumber}, tx ${l.transactionIndex}, log: ${l.transactionLogIndex}. TxHash: ${tx.hash}. Calldata: ${tx.data}`
  )

  let stateRoots: string[]
  try {
    ;[stateRoots] = abi.decode(
      ['bytes32[]'],
      ethers.utils.hexDataSlice(tx.data, 4)
    )
  } catch (e) {
    // This is, by definition, just an ill-formatted, and therefore invalid, tx.
    log.debug(
      `Error parsing calldata tx from CalldataTxEnqueued event. Calldata: ${tx.data}. Error: ${e.message}. Stack: ${e.stack}.`
    )
    return
  }

  await ds.insertL1RollupStateRoots(l.transactionHash, stateRoots)
}
