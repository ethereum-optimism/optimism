/* External Imports */
import { BigNumber, remove0x, rsvToSignature } from '@eth-optimism/core-utils'
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import { RollupTransaction, TransactionOutput } from '../../types'

export const l1BlockInsertStatement = `INSERT INTO l1_block(block_hash, parent_hash, block_number, block_timestamp, gas_limit, gas_used, processed) `
export const getL1BlockInsertValue = (
  block: Block,
  processed: boolean
): string => {
  return `${stringOrNull(block.hash)}, ${stringOrNull(
    block.parentHash
  )}, ${numOrNull(block.number)}, ${numOrNull(block.timestamp)}, ${bigNumOrNull(
    block.gasLimit
  )}, ${bigNumOrNull(block.gasUsed)}, ${bool(processed)}`
}

export const l1TxInsertStatement = `INSERT INTO l1_tx(block_number, tx_index, tx_hash, from_address, to_address, nonce, gas_limit, gas_price, calldata, signature) `
export const getL1TransactionInsertValue = (
  tx: TransactionResponse,
  index: number
): string => {
  return `${numOrNull(tx.blockNumber)}, ${numOrNull(index)}, ${stringOrNull(
    tx.hash
  )}, ${stringOrNull(tx.from)}, ${stringOrNull(tx.to)}, ${numOrNull(
    tx.nonce
  )}, ${bigNumOrNull(tx.gasLimit)}, ${bigNumOrNull(
    tx.gasPrice
  )}, ${stringOrNull(tx.data)}, '${rsvToSignature(tx.r, tx.s, tx.v)}'`
}

export const l1RollupTxInsertStatement = `INSERT INTO l1_rollup_tx(sender, l1_message_sender, target, calldata, queue_origin, nonce, gas_limit, signature, geth_submission_queue_index, index_within_submission, l1_tx_hash, l1_tx_index, l1_tx_log_index) `
export const getL1RollupTransactionInsertValue = (
  tx: RollupTransaction,
  batchNumber?: number
): string => {
  const batchNum = batchNumber === undefined ? 'NULL' : batchNumber
  return `${stringOrNull(tx.sender)}, ${stringOrNull(
    tx.l1MessageSender
  )}, ${stringOrNull(tx.target)}, ${stringOrNull(tx.calldata)}, ${numOrNull(
    tx.queueOrigin
  )}, ${numOrNull(tx.nonce)}, ${numOrNull(tx.gasLimit)}, ${stringOrNull(
    tx.signature
  )}, ${numOrNull(batchNum)}, ${numOrNull(
    tx.indexWithinSubmission
  )}, ${stringOrNull(tx.l1TxHash)}, ${numOrNull(tx.l1TxIndex)}, ${numOrNull(
    tx.l1TxLogIndex
  )}`
}

export const l1RollupStateRootInsertStatement = `INSERT into l1_rollup_state_root(state_root, batch_number, batch_index) `
export const getL1RollupStateRootInsertValue = (
  stateRoot: string,
  batchNumber: number,
  batchIndex: number
): string => {
  return `${stringOrNull(stateRoot)}, ${numOrNull(batchNumber)}, ${numOrNull(
    batchIndex
  )}`
}

export const l2TransactionOutputInsertStatement = `INSERT INTO l2_tx_output(block_number, block_timestamp, tx_index, tx_hash, sender, l1_message_sender, target, calldata, nonce, gas_limit, gas_price, signature, state_root, l1_rollup_tx_id) `
export const getL2TransactionOutputInsertValue = (
  tx: TransactionOutput
): string => {
  return `${numOrNull(tx.blockNumber)}, ${numOrNull(tx.timestamp)}, ${numOrNull(
    tx.transactionIndex
  )}, ${stringOrNull(tx.transactionHash)}, ${stringOrNull(
    tx.from
  )}, ${stringOrNull(tx.l1MessageSender)}, ${stringOrNull(
    tx.to
  )}, ${stringOrNull(tx.calldata)}, ${numOrNull(tx.nonce)}, ${bigNumberOrNull(
    tx.gasLimit
  )}, ${bigNumberOrNull(tx.gasPrice)}, ${stringOrNull(
    tx.signature
  )}, ${stringOrNull(tx.stateRoot)}, ${numOrNull(tx.l1RollupTransactionId)}`
}

export const bigNumberOrNull = (bigNumber: BigNumber): string => {
  return !!bigNumber ? `'${bigNumber.toString(10)}'` : 'NULL'
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
