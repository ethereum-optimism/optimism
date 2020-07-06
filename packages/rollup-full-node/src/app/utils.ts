/* External Imports */
import {
  ZERO_ADDRESS,
  abi,
  hexStrToBuf,
  logError,
  BloomFilter,
  numberToHexString,
  bufToHexString,
} from '@eth-optimism/core-utils'
import { getLogger } from '@eth-optimism/core-utils/build/src'
import { getContractDefinition } from '@eth-optimism/rollup-contracts'
import { ethers, Contract, ContractFactory, providers, Wallet } from 'ethers'
import { LogDescription } from 'ethers/utils'
import { Web3Provider, TransactionReceipt, Log } from 'ethers/providers'
import * as waffle from 'ethereum-waffle'

/* Internal Imports */
import {
  FullnodeHandler,
  L2ToL1MessageSubmitter,
  OvmTransactionReceipt,
} from '../types'
import { NoOpL2ToL1MessageSubmitter } from './message-submitter'
import { TestWeb3Handler } from './test-web3-rpc-handler'
import { FullnodeRpcServer } from './fullnode-rpc-server'

const logger = getLogger('utils')
const executionManagerInterface = new ethers.utils.Interface(
  getContractDefinition('ExecutionManager').abi
)

export interface OvmTransactionMetadata {
  ovmTxSucceeded: boolean
  ovmTo: string
  ovmFrom: string
  ovmCreatedContractAddress: string
  revertMessage?: string
}

export const revertMessagePrefix: string =
  'VM Exception while processing transaction: revert '

/**
 * Creates a Provider that uses the provided handler to handle `send`s.
 *
 * @param fullnodeHandler The handler to use for the provider's send function.
 * @return The provider.
 */
export const createProviderForHandler = (
  fullnodeHandler: FullnodeHandler
): Web3Provider => {
  // First, we create a mock provider which is identical to a normal ethers "mock provider"
  const provider = waffle.createMockProvider()

  // Then we replace `send()` with our modified send that uses the execution manager as a proxy
  provider.send = async (method: string, params: any) => {
    logger.debug('Sending -- Method:', method, 'Params:', params)

    // Convert the message or response if we need to
    const response = await fullnodeHandler.handleRequest(method, params)

    logger.debug('Received Response --', response)
    return response
  }

  // The return our slightly modified provider & the execution manager address
  return provider
}

/**
 * Creates a fullnodeHandler to handle the given Provider's `send`s.
 *
 * @param provider The provider to modify
 * @return The provider with modified `send`s
 */
export async function addHandlerToProvider(provider: any): Promise<any> {
  const messageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter()
  const fullnodeHandler: FullnodeHandler = await TestWeb3Handler.create(
    messageSubmitter
  )
  // Then we replace `send()` with our modified send that uses the execution manager as a proxy
  provider.send = async (method: string, params: any) => {
    logger.debug('Sending -- Method:', method, 'Params:', params)

    // Convert the message or response if we need to
    const response = await fullnodeHandler.handleRequest(method, params)

    logger.debug('Received Response --', response)
    return response
  }

  // The return our slightly modified provider & the execution manager address
  return provider
}

export async function createMockProvider() {
  const messageSubmitter = new NoOpL2ToL1MessageSubmitter()
  const fullnodeHandler = await TestWeb3Handler.create(messageSubmitter)
  const web3Provider = createProviderForHandler(fullnodeHandler)

  return web3Provider
}

const defaultDeployOptions = {
  gasLimit: 4000000,
  gasPrice: 9000000000,
}

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export async function deployOvmContract(
  wallet: Wallet,
  contractJSON: any,
  args: any[] = [],
  overrideOptions: providers.TransactionRequest = {}
) {
  // Get the factory and deploy the contract
  const factory = new ContractFactory(
    contractJSON.abi,
    contractJSON.bytecode,
    wallet
  )
  const contract = await factory.deploy(...args, {
    ...defaultDeployOptions,
    ...overrideOptions,
  })

  // Now get the deployment tx reciept so we can find the contract address
  // NOTE: We need to get the address manually because we do not have EOAs
  const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
    contract.deployTransaction.hash
  )
  // Create a new contract object with this wallet & the **real** address
  return new Contract(
    deploymentTxReceipt.contractAddress,
    contractJSON.abi,
    wallet
  )
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
