/* External Imports */
import {
  abi,
  add0x,
  getLogger,
  hexStrToBuf,
  logError,
  remove0x,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { ethers } from 'ethers'
import { LogDescription } from 'ethers/utils'
import { Log, TransactionReceipt } from 'ethers/providers'

/* Contract Imports */

import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as L2ExecutionManager from '../../build/contracts/L2ExecutionManager.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as L2ToL1MessageReceiver from '../../build/contracts/L2ToL1MessageReceiver.json'
import * as L2ToL1MessagePasser from '../../build/contracts/L2ToL1MessagePasser.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

/* Internal Imports */
import { OvmTransactionReceipt } from '../types'

// Contract Exports
export const L2ExecutionManagerContractDefinition = L2ExecutionManager
export const ContractAddressGeneratorContractDefinition = ContractAddressGenerator
export const RLPEncodeContractDefinition = RLPEncode
export const L2ToL1MessageReceiverContractDefinition = L2ToL1MessageReceiver
export const L2ToL1MessagePasserContractDefinition = L2ToL1MessagePasser

export const revertMessagePrefix: string =
  'VM Exception while processing transaction: revert '

export const executionManagerInterface = new ethers.utils.Interface(
  ExecutionManager.interface
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
 * Convert internal logs into OVM logs. Or in other words, take the logs which
 * are emitted by a normal Ganache or Geth node (this will include logs from the ExecutionManager),
 * parse them, and then convert them into logs which look like they would if you were running this tx
 * using an OVM backend.
 *
 *
 * @param logs an array of internal logs which we will parse and then convert.
 * @return the converted logs
 */
export const convertInternalLogsToOvmLogs = (logs: Log[]): Log[] => {
  let activeContract = ZERO_ADDRESS
  const ovmLogs = []
  logs.forEach((log) => {
    const executionManagerLog = executionManagerInterface.parseLog(log)
    if (executionManagerLog) {
      if (executionManagerLog.name === 'ActiveContract') {
        activeContract = executionManagerLog.values['_activeContract']
      } else {
        logger.debug(
          `${
            executionManagerLog.name
          }, values: ${JSON.stringify(executionManagerLog.values)}`
        )
      }
    } else {
      logger.debug(`Non-EM log: ${JSON.stringify(log)}`)
      ovmLogs.push({ ...log, address: activeContract })
    }
  })
  return ovmLogs
}

/**
 * Gets ovm transaction metadata from an internal transaction receipt.
 *
 * @param internalTxReceipt the internal transaction receipt
 * @return ovm transaction metadata
 */
export const getOvmTransactionMetadata = (
  internalTxReceipt: TransactionReceipt
): OvmTransactionMetadata => {
  let ovmTo
  let ovmFrom
  let ovmCreatedContractAddress
  let ovmTxSucceeded
  const logs = internalTxReceipt.logs
    .map((log) => executionManagerInterface.parseLog(log))
    .filter((log) => log != null)
  const callingWithEoaLog = logs.find((log) => log.name === 'CallingWithEOA')
  const eoaContractCreatedLog = logs.find(
    (log) => log.name === 'EOACreatedContract'
  )

  const revertEvents: LogDescription[] = logs.filter(
    (x) => x.name === 'EOACallRevert'
  )
  ovmTxSucceeded = !revertEvents.length

  if (callingWithEoaLog) {
    ovmFrom = callingWithEoaLog.values._ovmFromAddress
  }
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
 * @returns The converted receipt
 */
export const internalTxReceiptToOvmTxReceipt = async (
  internalTxReceipt: TransactionReceipt
): Promise<OvmTransactionReceipt> => {
  const ovmTransactionMetadata = getOvmTransactionMetadata(internalTxReceipt)
  // Construct a new receipt
  //
  // Start off with the internalTxReceipt
  const ovmTxReceipt: OvmTransactionReceipt = internalTxReceipt
  // Add the converted logs
  ovmTxReceipt.logs = convertInternalLogsToOvmLogs(internalTxReceipt.logs)
  // Update the to and from fields
  ovmTxReceipt.to = ovmTransactionMetadata.ovmTo
  // TODO: Update this to use some default account abstraction library potentially.
  ovmTxReceipt.from = ovmTransactionMetadata.ovmFrom
  // Also update the contractAddress in case we deployed a new contract
  ovmTxReceipt.contractAddress =
    ovmTransactionMetadata.ovmCreatedContractAddress

  ovmTxReceipt.status = ovmTransactionMetadata.ovmTxSucceeded ? 1 : 0

  if (ovmTransactionMetadata.revertMessage !== undefined) {
    ovmTxReceipt.revertMessage = ovmTransactionMetadata.revertMessage
  }

  logger.debug('Ovm parsed logs:', ovmTxReceipt.logs)
  // TODO: Fix the logsBloom to remove the txs we just removed

  // Return!
  return ovmTxReceipt
}
