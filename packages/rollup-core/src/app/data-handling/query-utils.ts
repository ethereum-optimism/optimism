/* External Imports */
import { Block, TransactionResponse } from 'ethers/providers'
import { RollupTransaction } from '../../types'

export const blockInsertStatement = `INSERT INTO block(block_hash, parent_hash, block_number, block_timestamp, gas_limit, gas_used, processed) `
export const txInsertStatement = `INSERT INTO tx(block_number, block_hash, from_address, to_address, nonce, gas_limit, gas_price, calldata, v, r, s) `
export const rollupTxInsertStatement = `INSERT INTO rollup_tx(nonce, gasLimit, sender, target, calldata) `

export const getTransactionInsertValue = (tx: TransactionResponse): string => {
  return `${tx.blockNumber}, '${tx.blockHash}', '${tx.from}', '${tx.to}', ${
    tx.nonce
  }, ${bigNumOrNull(tx.gasLimit)}, ${bigNumOrNull(tx.gasPrice)}, '${
    tx.data
  }', ${numOrNull(tx.v)}, ${numOrNull(tx.r)}, ${numOrNull(tx.s)}`
}

export const getBlockInsertValue = (
  block: Block,
  processed: boolean
): string => {
  return `'${block.hash}', '${block.parentHash}', ${
    block.number
  }, ${bigNumOrNull(block.gasLimit)}, ${bigNumOrNull(block.gasUsed)}, ${bool(
    processed
  )}`
}

export const getRollupTransactionInsertValue = (
  tx: RollupTransaction
): string => {
  return `${tx.nonce}, ${tx.gasLimit}, '${tx.sender}', '${tx.target}', '${tx.calldata}'`
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
