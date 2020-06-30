/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Transaction } from 'ethers/utils'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import * as ExecutionManager from '../../artifacts/ExecutionManager.json'

const executionManagerInterface = new ethers.utils.Interface(ExecutionManager.abi)

/**********************************
 * Byte String Generation Helpers *
 *********************************/

// Create a byte string of some length in bytes. It repeats the value provided until the
// string hits that length
export function makeRepeatedBytes(value: string, length: number): string {
  const repeated = value.repeat((length * 2) / value.length + 1)
  const sliced = repeated.slice(0, length * 2)
  return '0x' + sliced
}

export function makeRandomBlockOfSize(blockSize: number): string[] {
  const block = []
  for (let i = 0; i < blockSize; i++) {
    block.push(makeRepeatedBytes('' + Math.floor(Math.random() * 500 + 1), 32))
  }
  return block
}

export function makeRandomBatchOfSize(batchSize: number): string[] {
  return makeRandomBlockOfSize(batchSize)
}

/* External Imports */
import {
  ZERO_ADDRESS,
  getLogger,
  add0x,
  abi,
  getCurrentTime,
  keccak256,
  strToHexStr,
  remove0x,
  hexStrToBuf,
  bufToHexString,
  bufferUtils,
  logError,
  BloomFilter,
  numberToHexString,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Wallet, Signer } from 'ethers'
import {
  Provider,
  TransactionReceipt,
  JsonRpcProvider,
  Log,
} from 'ethers/providers'

/* Internal Imports */
import { DEFAULT_ACCOUNTS } from '../../src/constants'
import { Address, CHAIN_ID, GAS_LIMIT } from './core-helpers'

type Signature = [string, string, string]

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

export const revertMessagePrefix: string =
  'VM Exception while processing transaction: revert '

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
    .map((log) => executionManagerInterface.parseLog(log))
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

export const getWallets = (): Wallet[] => {
  return DEFAULT_ACCOUNTS.map((account) => {
    return new ethers.Wallet(account.secretKey)
  })
}

export const signTransaction = async (wallet: Wallet, transaction: any): Promise<string> => {
  return wallet.signTransaction(transaction)
}

export const getSignedComponents = (signed: string): any[] => {
  return ethers.utils.RLP.decode(signed).slice(-3)
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

export const ZERO_UINT = '00'.repeat(32)

export const gasLimit = 6_700_000
const logger = getLogger('helpers', true)

/**
 * Helper function to ensure GoVM is connected
 */
export const ensureGovmIsConnected = async (provider: JsonRpcProvider) => {
  let connected
  try {
    connected = (await provider.send('web3_clientVersion', [])).startsWith(
      'govm'
    )
  } catch {
    connected = false
  }
  connected.should.be.equal(
    true,
    'Govm is not connected. Please run govm as described [here](https://github.com/op-optimism/go-ethereum/blob/master/OPTIMISM_README.md)'
  )
}
/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export const manuallyDeployOvmContractReturnReceipt = async (
  wallet: Signer,
  provider: any,
  executionManager: Contract,
  contractDefinition: ContractFactory,
  constructorArguments: any[]
): Promise<any> => {
  const initCode = contractDefinition.getDeployTransaction(...constructorArguments).data as string

  const receipt: TransactionReceipt = await executeTransaction(
    executionManager,
    wallet,
    undefined,
    initCode,
    false
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
  constructorArguments: any[]
): Promise<Address> => {
  const receipt = await manuallyDeployOvmContractReturnReceipt(
    wallet,
    provider,
    executionManager,
    contractDefinition,
    constructorArguments
  )
  return receipt.contractAddress
}

export const executeTransaction = async (
  executionManager: Contract,
  wallet: Signer,
  to: Address,
  data: string,
  allowRevert: boolean
): Promise<any> => {
  // Verify that the transaction is not accidentally sending to the ZERO_ADDRESS
  if (to === ZERO_ADDRESS) {
    throw new Error('Sending to Zero Address disallowed')
  }

  // Get the `to` field -- NOTE: We have to set `to` to equal ZERO_ADDRESS if this is a contract create
  const ovmTo = to === null || to === undefined ? ZERO_ADDRESS : to

  // Actually make the call
  const tx = await executionManager.executeTransaction(
    getCurrentTime(),
    0,
    ovmTo,
    data,
    await wallet.getAddress(),
    ZERO_ADDRESS,
    allowRevert
  )
  // Return the parsed transaction values
  return executionManager.provider.waitForTransaction(tx.hash)
}

export const executeEOACall = async (
  executionManager: Contract,
  wallet: Wallet,
  to: Address,
  data: string
): Promise<any> => {
  // Verify that the transaction is not accidentally sending to the ZERO_ADDRESS
  if (to === ZERO_ADDRESS) {
    throw new Error('Sending to Zero Address disallowed')
  }

  // Get the nonce
  const nonce = await executionManager.getOvmContractNonce(wallet.address)
  // Create the transaction
  const transaction = {
    nonce,
    gasLimit: GAS_LIMIT,
    gasPrice: 0,
    to,
    value: 0,
    data,
    chainId: CHAIN_ID,
  }
  // Sign the transaction
  const signedTransaction = await signTransaction(wallet, transaction)

  // Parse the tx that we just signed to get the signature
  const ovmTx = ethers.utils.parseTransaction(signedTransaction)
  // Get the to field -- NOTE: We have to set `to` to equal ZERO_ADDRESS if this is a contract create
  const ovmTo = to === null || to === undefined ? ZERO_ADDRESS : to

  // Actually make the call
  const tx = await executionManager.executeEOACall(
    0,
    0,
    ovmTx.nonce,
    ovmTo,
    ovmTx.data,
    ovmTx.v,
    ovmTx.r,
    ovmTx.s
  )
  // Return the parsed transaction values
  return executionManager.provider.waitForTransaction(tx.hash)
}

/**
 * Deterministically computes the smart contract address given
 * the account that will deploy the contract (factory contract)
 * the salt as uint256 and the contract bytecode
 * Source: https://github.com/miguelmota/solidity-create2-example
 * Note: Use this function to generate new tests
 */
export const buildCreate2Address = (
  creatorAddress,
  saltHex,
  byteCode
): Address => {
  const preimage: string = `ff${remove0x(creatorAddress)}${remove0x(
    saltHex
  )}${keccak256(byteCode)}`
  return add0x(
    keccak256(preimage)
      .slice(-40)
      .toLowerCase()
  )
}

/**
 * Gets an address string from a bytes32 big-endian Address.
 * @param bytes32Address The 32-byte address string
 * @returns The 0x-prefixed 20-byte address string
 */
export const bytes32AddressToAddress = (bytes32Address: string): Address => {
  return bufToHexString(hexStrToBuf(bytes32Address).slice(12)).toLowerCase()
}

/**
 * Gets a padded big-endian 32-byte address string from an address string.
 * @param addr The 20-byte address string
 * @returns The 0x-prefixed 32-byte address string
 */
export const addressToBytes32Address = (addr: Address): string => {
  return bufToHexString(
    bufferUtils.padLeft(hexStrToBuf(addr), 32)
  ).toLowerCase()
}

/**
 * Converts a number to a 32-byte word hex string
 * @param n the number to convert
 * @returns The 0x-prefixed 32-byte address string
 */
export const numberToHexWord = (n: number): string => {
  return bufToHexString(bufferUtils.padLeft(numberToBuf(n), 32)).toLowerCase()
}

/**
 * Converts a Number to a Buffer of bytes
 * @param n the Number to convert
 * @returns The buffer
 */
export const numberToBuf = (n: number): Buffer => {
  const arr = new ArrayBuffer(4)
  const view = new DataView(arr)
  view.setUint32(0, n, false)
  return Buffer.from(arr)
}

/**
 * Waits for a transaction to complete and returns the result
 * @param n the Number to convert
 * @returns The buffer
 */
export const getTransactionResult = async (
  provider: any,
  tx: Transaction,
  returnType: string
): Promise<any[]> => {
  const receipt = await provider.waitForTransaction(tx.hash)
  return abi.decode([returnType], receipt.logs.pop().data)
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
 * Builds a ethers.js Log object from it's respective parts
 *
 * @param address The address the logs was sent from
 * @param event The event identifier
 * @param data The event data
 * @returns an ethers.js Log object
 */
export const buildLog = (
  address: string,
  event: string,
  data: string[],
  logIndex: number
): Log => {
  const types = event.match(/\((.+)\)/)
  const encodedData = types ? abi.encode(types[1].split(','), data) : '0x'

  return {
    address,
    topics: [add0x(keccak256(strToHexStr(event)))],
    data: encodedData,
    logIndex,
  }
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
    gasLimit,
  })
}

/**
 * Computes the method id of a function name and encodes it as
 * a hexidecimal string.
 * @param The name of the function
 * @returns The hex-encoded methodId
 */
export const encodeMethodId = (functionName: string): string => {
  return ethereumjsAbi.methodID(functionName, []).toString('hex')
}

/**
 * Encodes an array of function arguments into a hex string.
 * @param any[] An array of arguments
 * @returns The hex-encoded function arguments
 */
export const encodeRawArguments = (args: any[]): string => {
  return args
    .map((arg) => {
      if (Number.isInteger(arg)) {
        return bufferUtils.numberToBuffer(arg).toString('hex')
      } else if (Buffer.isBuffer(arg)) {
        return arg.toString('hex')
      } else if (arg && arg.startsWith('0x')) {
        return remove0x(arg)
      } else {
        return arg
      }
    })
    .join('')
}
