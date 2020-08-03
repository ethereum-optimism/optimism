/* External Imports */
import { ethers, Signer, Contract, ContractFactory } from 'ethers'
import { Log, TransactionReceipt, JsonRpcProvider, Provider } from 'ethers/providers'
import {
  abi,
  add0x,
  hexStrToBuf,
  getLogger,
  logError,
  BloomFilter,
  numberToHexString,
  bufToHexString,
  getCurrentTime,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import { ZERO_ADDRESS, GAS_LIMIT } from './constants'
import { Address } from './types'
import { encodeMethodId, encodeRawArguments } from './ethereum-helpers'

/* Contract Imports */
import { getContractInterface } from '../../index'

const ExecutionManagerInterface = getContractInterface('ExecutionManager')
const logger = getLogger('contracts:test-helpers', true)
const revertMessagePrefix: string =
  'VM Exception while processing transaction: revert '

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
  const uppercaseExecutionMangerAddress: string = executionManagerAddress.toUpperCase()
  let activeContractAddress: string = logs[0] ? logs[0].address : ZERO_ADDRESS
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
      const executionManagerLog = ExecutionManagerInterface.parseLog(log)
      if (!executionManagerLog) {
        stringsToDebugLog.push(
          `Execution manager emitted log with topics: ${log.topics}.  These were unrecognized by the interface parser-but definitely not an ActiveContract event, ignoring...`
        )
      } else if (executionManagerLog.name === 'ActiveContract') {
        activeContractAddress = executionManagerLog.args['_activeContract']
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
): any => {
  let ovmTo
  let ovmFrom
  let ovmCreatedContractAddress
  let ovmTxSucceeded

  if (!internalTxReceipt) {
    return undefined
  }

  const logs = internalTxReceipt.logs
    .map((log) => ExecutionManagerInterface.parseLog(log))
    .filter((log) => log != null)
  const callingWithEoaLog = logs.find((log) => log.name === 'CallingWithEOA')

  const revertEvents: any[] = logs.filter((x) => x.name === 'EOACallRevert')
  ovmTxSucceeded = !revertEvents.length

  if (callingWithEoaLog) {
    ovmFrom = callingWithEoaLog.args._ovmFromAddress
    ovmTo = callingWithEoaLog.args._ovmToAddress
  }

  const eoaContractCreatedLog = logs.find(
    (log) => log.name === 'EOACreatedContract'
  )
  if (eoaContractCreatedLog) {
    ovmCreatedContractAddress = eoaContractCreatedLog.args._ovmContractAddress
    ovmTo = ovmCreatedContractAddress
  }

  const metadata: any = {
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
): Promise<any> => {
  const ovmTransactionMetadata = getSuccessfulOvmTransactionMetadata(
    internalTxReceipt
  )
  // Construct a new receipt

  // Start off with the internalTxReceipt
  const ovmTxReceipt: any = internalTxReceipt
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

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export const manuallyDeployOvmContractReturnReceipt = async (
  wallet: Signer,
  provider: any,
  executionManager: Contract,
  contractDefinition: ContractFactory,
  constructorArguments: any[],
  timestamp: number = getCurrentTime()
): Promise<any> => {
  const initCode = contractDefinition.getDeployTransaction(
    ...constructorArguments
  ).data as string

  const receipt: TransactionReceipt = await executeTransaction(
    executionManager,
    wallet,
    undefined,
    initCode,
    false,
    timestamp
  )

  return internalTxReceiptToOvmTxReceipt(receipt, executionManager.address)
}

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export const manuallyDeployOvmContract = async (
  wallet: Signer,
  provider: any,
  executionManager: Contract,
  contractDefinition: any,
  constructorArguments: any[],
  timestamp: number = getCurrentTime()
): Promise<Address> => {
  const receipt = await manuallyDeployOvmContractReturnReceipt(
    wallet,
    provider,
    executionManager,
    contractDefinition,
    constructorArguments,
    timestamp
  )
  return receipt.contractAddress
}

export const executeTransaction = async (
  executionManager: Contract,
  wallet: Signer,
  to: Address,
  data: string,
  allowRevert: boolean,
  timestamp: number = getCurrentTime(),
  provider: any = false
): Promise<any> => {

  // Verify that the transaction is not accidentally sending to the ZERO_ADDRESS
  if (to === ZERO_ADDRESS) {
    throw new Error('Sending to Zero Address disallowed')
  }

  // Get the `to` field -- NOTE: We have to set `to` to equal ZERO_ADDRESS if this is a contract create
  const ovmTo = to === null || to === undefined ? ZERO_ADDRESS : to

  // get the max gas limit allowed by this EM
  const getMaxGasLimitCalldata =
    executionManager.interface.encodeFunctionData('ovmBlockGasLimit')
  const maxTxGasLimit = await wallet.provider.call({
    to: executionManager.address,
    data: getMaxGasLimitCalldata,
    gasLimit: GAS_LIMIT,
  })

  // Actually make the call
  const tx = await executionManager.executeTransaction(
    getCurrentTime(),
    0,
    ovmTo,
    data,
    await wallet.getAddress(),
    ZERO_ADDRESS,
    maxTxGasLimit,
    allowRevert
  )
  // Return the parsed transaction values
  if (provider) {
    return provider.waitForTransaction(tx.hash)
  } else {
    return executionManager.provider.waitForTransaction(tx.hash)
  }
}

/**
 * Returns whether the provided Create transaction succeeded.
 *
 * @param executionManager The ExecutionManager contract.
 * @param createTxHash The transaction hash in question.
 * @returns True if there was a successful create in this tx, false otherwise.
 */
export const didCreateSucceed = async (
  executionManager: Contract,
  createTxHash: string
): Promise<boolean> => {
  const receipt = await executionManager.provider.waitForTransaction(
    createTxHash
  )
  return (
    receipt.logs
      .map((x) => executionManager.interface.parseLog(x))
      .filter((x) => x.name === 'CreatedContract').length > 0
  )
}

/**
 * Executes a call in the OVM
 * @param The name of the function to call
 * @param The function arguments
 * @returns The return value of the function executed
 */
export const executeOVMCall = async (
  executionManager: Contract,
  functionName: string,
  args: any[]
): Promise<string> => {
  const data: string = add0x(
    encodeMethodId(functionName) + encodeRawArguments(args)
  )

  return executionManager.provider.call({
    to: executionManager.address,
    data,
    gasLimit: GAS_LIMIT,
  })
}
