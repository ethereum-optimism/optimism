/* External Imports */
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import { RollupTransaction, TransactionOutput } from '../../types'
import { remove0x } from '@eth-optimism/core-utils/build'
export const l1BlockInsertStatement = `INSERT INTO l1_block(block_hash, parent_hash, block_number, block_timestamp, gas_limit, gas_used, processed) `
export const getL1BlockInsertValue = (
  block: Block,
  processed: boolean
): string => {
  return `'${block.hash}', '${block.parentHash}', ${
    block.number
  }, ${bigNumOrNull(block.gasLimit)}, ${bigNumOrNull(block.gasUsed)}, ${bool(
    processed
  )}`
}

export const l1TxInsertStatement = `INSERT INTO l1_tx(block_number, block_hash, tx_hash, from_address, to_address, nonce, gas_limit, gas_price, calldata, signature) `
export const getL1TransactionInsertValue = (
  tx: TransactionResponse
): string => {
  return `${tx.blockNumber}, '${tx.blockHash}', '${tx.hash}', '${tx.from}', '${
    tx.to
  }', ${tx.nonce}, ${bigNumOrNull(tx.gasLimit)}, ${bigNumOrNull(
    tx.gasPrice
  )}, '${tx.data}', '${tx.r}${remove0x(tx.s)}${tx.v.toString(16)}'`
}

export const l1RollupTxInsertStatement = `INSERT INTO l1_rollup_tx(sender, l1_message_sender, target, calldata, queue_origin, nonce, gas_limit, signature, geth_submission_queue_index, index_within_submission, l1_tx_hash, l1_tx_index, l1_tx_log_index) `
export const getL1RollupTransactionInsertValue = (
  tx: RollupTransaction,
  batchNumber?: number
): string => {
  const batchNum = batchNumber || 'NULL'
  return `${stringOrNull(tx.sender)}, ${stringOrNull(tx.l1MessageSender)}, '${
    tx.target
  }', '${tx.calldata}', ${tx.queueOrigin}, ${numOrNull(tx.nonce)}, ${numOrNull(
    tx.gasLimit
  )}, ${stringOrNull(tx.signature)}, ${batchNum}, ${
    tx.indexWithinSubmission
  }, '${tx.l1TxHash}', ${tx.l1TxIndex}, ${numOrNull(tx.l1TxLogIndex)}`
}

export const l1RollupStateRootInsertStatement = `INSERT into l1_rollup_state_root(state_root, batch_number, batch_index) `
export const getL1RollupStateRootInsertValue = (
  stateRoot: string,
  batchNumber: number,
  batchIndex: number
): string => {
  return `'${stateRoot}', ${batchNumber}, ${batchIndex}`
}

export const l2TransactionOutputInsertStatement = `INSERT INTO l2_tx_output(block_number, block_timestamp, tx_index, tx_hash, sender, l1_message_sender, target, calldata, nonce, signature, state_root) `
export const getL2TransactionOutputInsertValue = (
  tx: TransactionOutput
): string => {
  return `'${tx.blockNumber}', ${tx.timestamp}, ${tx.transactionIndex}, '${
    tx.transactionHash
  }' ${stringOrNull(tx.from)}, ${stringOrNull(tx.l1MessageSender)}, '${
    tx.to
  }', '${tx.calldata}', ${tx.nonce}, ${stringOrNull(tx.signature)}, '${
    tx.stateRoot
  }'`
}

export const bigNumOrNull = (bn: any): string => {
  return !!bn ? `'${bn.toString()}'` : 'NULL'
}

export const numOrNull = (num: any): any => {
  return typeof num === 'number' ? num : 'NULL'
}

export const bool = (b: boolean): string => {
  return b ? 'TRUE' : 'FALSE'
}

export const stringOrNull = (s: string): string => {
  return !!s ? `'${s}'` : 'NULL'
}
