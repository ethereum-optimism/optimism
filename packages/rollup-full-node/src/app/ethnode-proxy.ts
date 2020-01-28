/* External Imports */
import {
  getLogger,
  abi,
  remove0x,
  ZERO_ADDRESS,
  deployContract,
} from '@pigi/core-utils'
import {
  ExecutionManagerContractDefinition,
  RLPEncodeContractDefinition,
  ContractAddressGeneratorContractDefinition,
  convertInternalLogsToOvmLogs,
} from '@pigi/ovm'
import { utils, ContractFactory, Wallet, Contract } from 'ethers'
import { Web3Provider } from 'ethers/providers'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import { Web3RpcMethods, Web3RpcHandlerFunctions } from '../types'

const getExeCallInternalCalldata = (
  ovmEntrypoint: string,
  ovmCalldata: string
): string => {
  return getExecutionMgrTxData('executeCall', ovmEntrypoint, ovmCalldata)
}

const getExeTxInternalCalldata = (
  ovmEntrypoint: string,
  ovmCalldata: string
): string => {
  return getExecutionMgrTxData('executeCall', ovmEntrypoint, ovmCalldata)
}

/**
 * Generates the calldata for executing either a call or transaction
 */
const getExecutionMgrTxData = (method, ovmEntrypoint, ovmCalldata) => {
  const methodId: string = ethereumjsAbi.methodID(method, []).toString('hex')

  const timestamp: string = '00'.repeat(32)
  const origin: string = '00'.repeat(32)
  const encodedEntrypoint: string = '00'.repeat(12) + remove0x(ovmEntrypoint)
  const txBody: string = `0x${methodId}${timestamp}${origin}${encodedEntrypoint}${remove0x(
    ovmCalldata
  )}`
  return txBody
}

const log = getLogger('mock-rollup-fullnode')

export class EthnodeProxy {
  private executionManager: Contract
  private internalTxHashes: { string: string } | {} = {}

  public constructor(
    readonly provider: Web3Provider,
    readonly wallet: Wallet,
    executionManagerAddress?: string
  ) {
    if (executionManagerAddress !== undefined) {
      // Create a new execution manager contract interface object
      this.executionManager = new Contract(
        executionManagerAddress,
        ExecutionManagerContractDefinition.abi,
        this.wallet
      )
    }
  }

  public async deployExecutionManager(): Promise<string> {
    if (this.executionManager !== undefined) {
      throw new Error('Execution manager already deployed!')
    }

    const purityCheckerContractAddress = ZERO_ADDRESS
    // Now deploy the execution manager!
    this.executionManager = await deployContract(
      ExecutionManagerContractDefinition,
      this.wallet,
      purityCheckerContractAddress,
      this.wallet.address
    )

    log.info(
      'Deployed execution manager to address:',
      this.executionManager.address
    )
    // For now we need to return the execution manager address because it's used in our tests during contract deployment
    return this.executionManager.address
  }

  private web3RpcMethodHandlers: Web3RpcHandlerFunctions = {
    [Web3RpcMethods.getTransactionCount]: async (
      params: any[]
    ): Promise<string> => {
      const response = await this.provider.send(
        Web3RpcMethods.getTransactionCount,
        params
      )
      return response
    },
    [Web3RpcMethods.sendRawTransaction]: async (
      params: any[]
    ): Promise<string> => {
      log.info('Sending raw transaction with params:', params)
      const rawOvmTx = params[0]
      // Convert the OVM transaction into an "internal" tx which we can use for our execution manager
      const internalTx = await this.ovmTxToInternalTx(rawOvmTx)
      // Then apply our transaction
      const internalTxHash = await this.provider.send(
        Web3RpcMethods.sendRawTransaction,
        internalTx
      )
      // Now compute the hash of the OVM transaction which we will return
      const ovmTxHash = await utils.keccak256(rawOvmTx)
      // Store the ovmTxHash as a reference to the internalTxHash
      this.internalTxHashes[ovmTxHash] = internalTxHash
      log.info('Completed send raw tx. Response:' + ovmTxHash)
      // Return the *OVM* tx hash. We can do this because we store a mapping to the ovmTxHashs locally.
      return ovmTxHash
    },
    [Web3RpcMethods.call]: async (params: any[]): Promise<string> => {
      log.info('Making eth_call:', params)
      // First get the internal calldata for our internal call
      const internalCalldata = getExeCallInternalCalldata(
        params[0].to,
        params[0].data
      )
      // Then actually make the call and get the response
      const response = await this.provider.send(Web3RpcMethods.call, [
        {
          from: ZERO_ADDRESS,
          to: this.executionManager.address,
          data: internalCalldata,
        },
        params[1],
      ])
      // Now just return the response!
      log.info('Finished eth_call and got response:', response)
      return response
    },
    [Web3RpcMethods.getTransactionReceipt]: async (
      params: any[]
    ): Promise<string> => {
      log.info('Getting tx receipt for hash:', params[0])
      // First convert our ovmTxHash into an internalTxHash
      const internalTxHash = this.internalTxHashes[params[0]]
      const internalTxReceipt = await this.provider.send(
        Web3RpcMethods.getTransactionReceipt,
        [internalTxHash]
      )
      // Now let's parse the internal transaction reciept
      const ovmTxReceipt = await this.internalTxReceiptToOvmTxReceipt(
        internalTxReceipt
      )
      log.info('Returning tx receipt:', internalTxReceipt)
      return ovmTxReceipt
    },
    [Web3RpcMethods.blockNumber]: async (params: any[]): Promise<string> => {
      const response = await this.provider.send(
        Web3RpcMethods.blockNumber,
        params
      )
      // For now we will just use the internal node's blocknumber.
      // TODO: Add rollup block tracking
      return response
    },
    [Web3RpcMethods.gasPrice]: async (params: any[]): Promise<string> => {
      // Gas price is always zero
      return '0x0'
    },
    [Web3RpcMethods.estimateGas]: async (params: any[]): Promise<string> => {
      log.info('Started estimating')
      // First convert the calldata
      const internalCalldata = getExeTxInternalCalldata(
        params[0].to,
        params[0].data
      )
      // Then estimate the gas
      const response = await this.provider.send(Web3RpcMethods.estimateGas, [
        {
          from: ZERO_ADDRESS,
          to: this.executionManager.address,
          data: internalCalldata,
        },
      ])
      log.info('Estimated Gas Cost. Response:', response)
      return response
    },
    [Web3RpcMethods.getCode]: async (params: any[]): Promise<string> => {
      if (params[1] !== 'latest') {
        throw new Error('No support for historical code lookups!')
      }
      // First get the code contract address at the requested OVM address
      const codeContractAddress = await this.executionManager.getCodeContractAddress(
        params[0]
      )
      const response = await this.provider.send(Web3RpcMethods.getCode, [
        codeContractAddress,
        'latest',
      ])
      return response
    },
  }

  /**
   * OVM tx to EVM tx converter
   */
  public async ovmTxToInternalTx(rawOvmTx: string): Promise<string> {
    // Decode the OVM transaction -- this will be used to construct our internal transaction
    const ovmTx = utils.parseTransaction(rawOvmTx)
    log.info(ovmTx)
    // Get the nonce of the account that we will use to send everything
    // TODO: Make sure we lock this function with this nonce so we don't send to txs with the same nonce
    const nonce = await this.wallet.getTransactionCount()
    // Generate the calldata which we'll use to call our internal execution manager
    // First pull out the ovmEntrypoint (we just need to check if it's null & if so set ovmEntrypoint to the zero address as that's how we deploy contracts)
    const ovmEntrypoint = ovmTx.to === null ? ZERO_ADDRESS : ovmTx.to
    // Then construct the internal calldata
    const internalCalldata = getExeTxInternalCalldata(ovmEntrypoint, ovmTx.data)
    // Construct the transaction
    const internalTx = {
      nonce,
      gasPrice: 0,
      gasLimit: ovmTx.gasLimit,
      to: this.executionManager.address,
      value: 0,
      data: internalCalldata,
    }
    log.info('The internal')
    log.info(internalTx)
    // Sign
    const rawInternalTx = await this.wallet.sign(internalTx)
    // And return!
    return rawInternalTx
  }

  /**
   * EVM receipt to OVM receipt converter
   */
  public async internalTxReceiptToOvmTxReceipt(
    internalTxReceipt: any
  ): Promise<any> {
    const convertedOvmLogs = convertInternalLogsToOvmLogs(
      this.executionManager,
      internalTxReceipt.logs
    )

    // Construct a new receipt
    //
    // Start off with the internalTxReceipt
    const ovmTxReceipt = internalTxReceipt
    // Add the converted logs
    ovmTxReceipt.logs = convertedOvmLogs.ovmLogs
    // Update the to and from fields
    ovmTxReceipt.to = convertedOvmLogs.ovmEntrypoint
    // TODO: Update this to use some default account abstraction library potentially.
    ovmTxReceipt.from = ZERO_ADDRESS
    // Also update the contractAddress in case we deployed a new contract
    ovmTxReceipt.contractAddress = convertedOvmLogs.ovmCreatedContractAddress
    // TODO: Fix the logsBloom to remove the txs we just removed

    // Return!
    return ovmTxReceipt
  }

  public async handleRequest(
    method: string,
    params: string[]
  ): Promise<string> {
    // Make sure the method is available
    if (typeof this.web3RpcMethodHandlers[method] === 'undefined') {
      throw new Error('Method ' + method + ' is not supported by this mock!')
    }
    // Run the function! We won't throw an error
    const result = await this.web3RpcMethodHandlers[method](params)
    // Return!
    return result
  }
}
