/* External Imports */
import { TransactionLog, TransactionReceipt } from '@eth-optimism/rollup-core'
import { BigNumber, getLogger } from '@eth-optimism/core-utils'

/* Internal imports */
import { abi, logAbiTypes, transactionReceiptAbiTypes } from './common'

const log = getLogger('abiEncoders')

/**
 * Creates a TransactionLog from an ABI-encoded log.
 *
 * @param abiEncoded The ABI-encoded TransactionLog.
 * @returns the TransactionLog.
 */
export const abiDecodeLog = (abiEncoded: string): TransactionLog => {
  const [
    data,
    topics,
    logIndex,
    transactionIndex,
    transactionHash,
    blockHash,
    blockNumber,
    address,
  ] = abi.decode(logAbiTypes, abiEncoded)
  return {
    data,
    topics,
    logIndex: new BigNumber(logIndex),
    transactionIndex: new BigNumber(transactionIndex),
    transactionHash,
    blockHash,
    blockNumber: new BigNumber(blockNumber),
    address,
  }
}

/**
 * Creates a TransactionReceipt from an ABI-encoded receipt.
 *
 * @param abiEncoded The ABI-encoded TransactionReceipt.
 * @returns the TransactionReceipt.
 */
export const abiDecodeTransactionReceipt = (
  abiEncoded: string
): TransactionReceipt => {
  const [
    status,
    transactionHash,
    transactionIndex,
    blockHash,
    blockNumber,
    contractAddress,
    cumulativeGasUsed,
    gasUsed,
    logsEncoded,
  ] = abi.decode(transactionReceiptAbiTypes, abiEncoded)

  const logs: TransactionLog[] = []
  for (const l of logsEncoded) {
    logs.push(abiDecodeLog(l))
  }

  return {
    status,
    transactionHash,
    transactionIndex: new BigNumber(transactionIndex),
    blockHash,
    blockNumber: new BigNumber(blockNumber),
    contractAddress,
    cumulativeGasUsed: new BigNumber(cumulativeGasUsed),
    gasUsed: new BigNumber(gasUsed),
    logs,
  }
}
