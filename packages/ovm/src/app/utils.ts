/* External Imports */
import {
  abi,
  getLogger,
  hexStrToBuf,
  bufToHexString,
  numberToHexString,
  logError,
  ZERO_ADDRESS,
  LOG_NEWLINE_STRING,
  BloomFilter,
} from '@eth-optimism/core-utils'
import { Address } from '@eth-optimism/rollup-core'

import { ethers } from 'ethers'
import { LogDescription } from 'ethers/utils'
import { Log, TransactionReceipt } from 'ethers/providers'

/* Contract Imports */

import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as L2ExecutionManager from '../../build/contracts/L2ExecutionManager.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as L2ToL1MessageReceiver from '../../build/contracts/L2ToL1MessageReceiver.json'
import * as L2ToL1MessagePasser from '../../build/contracts/L2ToL1MessagePasser.json'
import * as L1ToL2TransactionPasser from '../../build/contracts/L1ToL2TransactionPasser.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

/* Internal Imports */
import { OvmTransactionReceipt } from '../types'

// Contract Exports
export const L2ExecutionManagerContractDefinition = L2ExecutionManager
export const ContractAddressGeneratorContractDefinition = ContractAddressGenerator
export const RLPEncodeContractDefinition = RLPEncode
export const L2ToL1MessageReceiverContractDefinition = L2ToL1MessageReceiver
export const L2ToL1MessagePasserContractDefinition = L2ToL1MessagePasser
export const L1ToL2TransactionPasserContractDefinition = L1ToL2TransactionPasser

export const revertMessagePrefix: string =
  'VM Exception while processing transaction: revert '

export const executionManagerInterface = new ethers.utils.Interface(
  ExecutionManager.interface
)

export const l2ExecutionManagerInterface = new ethers.utils.Interface(
  L2ExecutionManager.interface
)
export const l2ToL1MessagePasserInterface = new ethers.utils.Interface(
  L2ToL1MessagePasser.interface
)

const logger = getLogger('utils')

export interface OvmTransactionMetadata {
  ovmTxSucceeded: boolean
  ovmTo: string
  ovmFrom: string
  ovmCreatedContractAddress: string
  revertMessage?: string
}

/**
 * Convert internal transaction logs into OVM logs. Or in other words, take the logs which
 * are emitted by a normal Ganache or Geth node (this will include logs from the ExecutionManager),
 * parse them, and then convert them into logs which look like they would if you were running this tx
 * using an OVM backend.
 *
 * NOTE: The input logs MUST NOT be stripped of any Execution Manager events, or this function will break.
 *
 * @param logs An array of internal transaction logs which we will parse and then convert.
 * @param executionManagerAddress The address of the Execution Manager contract for log parsing.
 * @return the converted logs
 */
export const convertInternalLogsToOvmLogs = (
  logs: Log[],
  executionManagerAddress: string
): Log[] => {
  const uppercaseExecutionMangerAddress: Address = executionManagerAddress.toUpperCase()
  let activeContractAddress: Address = logs[0] ? logs[0].address : ZERO_ADDRESS
  const stringsToDebugLog = [`Parsing internal logs ${JSON.stringify(logs)}: `]
  const ovmLogs = []
  let numberOfEMLogs = 0
  let prevEMLogIndex = 0
  logs.forEach((log) => {
    if (log.address.toUpperCase() === uppercaseExecutionMangerAddress) {
      if (log.logIndex <= prevEMLogIndex) {
        // This indicates a new TX, so reset number of EM logs to 0
        numberOfEMLogs = 0
      }
      numberOfEMLogs++
      prevEMLogIndex = log.logIndex
      const executionManagerLog = executionManagerInterface.parseLog(log)
      if (!executionManagerLog) {
        stringsToDebugLog.push(
          `Execution manager emitted log with topics: ${log.topics}.  These were unrecognized by the interface parser-but definitely not an ActiveContract event, ignoring...`
        )
      } else if (executionManagerLog.name === 'ActiveContract') {
        activeContractAddress = executionManagerLog.values['_activeContract']
      }
    } else {
      const newIndex = log.logIndex - numberOfEMLogs
      ovmLogs.push({
        ...log,
        address: activeContractAddress,
        logIndex: newIndex,
      })
    }
  })
  logger.debug(stringsToDebugLog.join(LOG_NEWLINE_STRING))
  return ovmLogs
}

/**
 * Gets ovm transaction metadata from an internal transaction receipt.
 *
 * @param internalTxReceipt the internal transaction receipt
 * @return ovm transaction metadata
 */
export const getSuccessfulOvmTransactionMetadata = (
  internalTxReceipt: TransactionReceipt
): OvmTransactionMetadata => {
  let ovmTo
  let ovmFrom
  let ovmCreatedContractAddress
  let ovmTxSucceeded

  if (!internalTxReceipt) {
    return undefined
  }

  const logs = internalTxReceipt.logs
    .map((log) => executionManagerInterface.parseLog(log))
    .filter((log) => log != null)
  const callingWithEoaLog = logs.find((log) => log.name === 'CallingWithEOA')

  const revertEvents: LogDescription[] = logs.filter(
    (x) => x.name === 'EOACallRevert'
  )
  ovmTxSucceeded = !revertEvents.length

  if (callingWithEoaLog) {
    ovmFrom = callingWithEoaLog.values._ovmFromAddress
    ovmTo = callingWithEoaLog.values._ovmToAddress
  }

  const eoaContractCreatedLog = logs.find(
    (log) => log.name === 'EOACreatedContract'
  )
  if (eoaContractCreatedLog) {
    ovmCreatedContractAddress = eoaContractCreatedLog.values._ovmContractAddress
    ovmTo = ovmCreatedContractAddress
  }

  const metadata: OvmTransactionMetadata = {
    ovmTxSucceeded,
    ovmTo,
    ovmFrom,
    ovmCreatedContractAddress,
  }

  if (!ovmTxSucceeded) {
    try {
      if (
        !revertEvents[0].values['_revertMessage'] ||
        revertEvents[0].values['_revertMessage'].length <= 2
      ) {
        metadata.revertMessage = revertMessagePrefix
      } else {
        // decode revert message from event
        const msgBuf: any = abi.decode(
          ['bytes'],
          // Remove the first 4 bytes of the revert message that is a sighash
          ethers.utils.hexDataSlice(revertEvents[0].values['_revertMessage'], 4)
        )
        const revertMsg: string = hexStrToBuf(msgBuf[0]).toString('utf8')
        metadata.revertMessage = `${revertMessagePrefix}${revertMsg}`
        logger.debug(`Decoded revert message: [${metadata.revertMessage}]`)
      }
    } catch (e) {
      logError(logger, `Error decoding revert event!`, e)
    }
  }

  return metadata
}

/**
 * Converts an EVM receipt to an OVM receipt.
 *
 * @param internalTxReceipt The EVM tx receipt to convert to an OVM tx receipt
 * @param ovmTxHash The OVM tx hash to replace the internal tx hash with.
 * @returns The converted receipt
 */
export const internalTxReceiptToOvmTxReceipt = async (
  internalTxReceipt: TransactionReceipt,
  executionManagerAddress: string,
  ovmTxHash?: string
): Promise<OvmTransactionReceipt> => {
  const ovmTransactionMetadata = getSuccessfulOvmTransactionMetadata(
    internalTxReceipt
  )
  // Construct a new receipt

  // Start off with the internalTxReceipt
  const ovmTxReceipt: OvmTransactionReceipt = internalTxReceipt
  // Add the converted logs
  ovmTxReceipt.logs = convertInternalLogsToOvmLogs(
    internalTxReceipt.logs,
    executionManagerAddress
  )
  // Update the to and from fields if necessary
  if (ovmTransactionMetadata.ovmTo) {
    ovmTxReceipt.to = ovmTransactionMetadata.ovmTo
  }
  // Also update the contractAddress in case we deployed a new contract
  ovmTxReceipt.contractAddress = !!ovmTransactionMetadata.ovmCreatedContractAddress
    ? ovmTransactionMetadata.ovmCreatedContractAddress
    : null

  ovmTxReceipt.status = ovmTransactionMetadata.ovmTxSucceeded ? 1 : 0

  if (!!ovmTxReceipt.transactionHash && !!ovmTxHash) {
    ovmTxReceipt.transactionHash = ovmTxHash
  }

  if (ovmTransactionMetadata.revertMessage !== undefined) {
    ovmTxReceipt.revertMessage = ovmTransactionMetadata.revertMessage
  }

  logger.debug('Ovm parsed logs:', ovmTxReceipt.logs)
  const logsBloom = new BloomFilter()
  ovmTxReceipt.logs.forEach((log, index) => {
    logsBloom.add(hexStrToBuf(log.address))
    log.topics.forEach((topic) => logsBloom.add(hexStrToBuf(topic)))
    log.transactionHash = ovmTxReceipt.transactionHash
    log.logIndex = numberToHexString(index) as any
  })
  ovmTxReceipt.logsBloom = bufToHexString(logsBloom.bitvector)

  // Return!
  return ovmTxReceipt
}
