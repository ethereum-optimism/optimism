/* External Imports */
import {
  Address,
  CHAIN_ID,
  GAS_LIMIT,
  getCurrentTime,
  initializeL2Node,
  isErrorEVMRevert,
  L1ToL2Transaction,
  L1ToL2TransactionListener,
  L2NodeContext,
  L2ToL1Message,
} from '@eth-optimism/rollup-core'
import {
  add0x,
  bufToHexString,
  BloomFilter,
  getLogger,
  hexStrToBuf,
  hexStrToNumber,
  logError,
  numberToHexString,
  remove0x,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import {
  convertInternalLogsToOvmLogs,
  internalTxReceiptToOvmTxReceipt,
  OvmTransactionReceipt,
} from '@eth-optimism/ovm'
import {
  executionManagerInterface,
  l2ToL1MessagePasserInterface,
} from '@eth-optimism/rollup-contracts'

import AsyncLock from 'async-lock'
import { utils, Wallet } from 'ethers'
import Web3 from 'web3'
import { JsonRpcProvider, TransactionReceipt } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
  L2ToL1MessageSubmitter,
  UnsupportedMethodError,
  Web3Handler,
  Web3RpcTypes,
  Web3RpcMethods,
  RevertError,
  UnsupportedFilterError,
} from '../types'
import { NoOpL2ToL1MessageSubmitter } from './message-submitter'

const log = getLogger('web3-handler')

export const latestBlock: string = 'latest'
const lockKey: string = 'LOCK'

const EMEvents = executionManagerInterface.events
const ALL_EXECUTION_MANAGER_EVENT_TOPICS = []
for (const eventKey of Object.keys(EMEvents)) {
  ALL_EXECUTION_MANAGER_EVENT_TOPICS.push(EMEvents[eventKey].topic)
}

export class DefaultWeb3Handler
  implements Web3Handler, FullnodeHandler, L1ToL2TransactionListener {
  private readonly ovmHashToOvmTransactionCache: Object = {}
  protected blockTimestamps: Object = {}
  private lock: AsyncLock

  /**
   * Creates a local node, deploys the L2ExecutionManager to it, and returns a
   * Web3Handler that handles Web3 requests to it.
   *
   * @param messageSubmitter The messageSubmitter to use to pass messages to L1. Will be replaced by block submitter.
   * @param web3Provider (optional) The web3 provider to use.
   * @param l2NodeContext (optional) The L2NodeContext to use.
   * @returns The constructed Web3 handler.
   */
  public static async create(
    messageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter(),
    web3Provider?: JsonRpcProvider,
    l2NodeContext?: L2NodeContext
  ): Promise<DefaultWeb3Handler> {
    log.info(
      `Creating Web3 Handler with provider: ${
        !!web3Provider
          ? web3Provider.connection.url
          : 'undefined -- will create.'
      }`
    )

    const timestamp = getCurrentTime()
    const nodeContext: L2NodeContext =
      l2NodeContext || (await initializeL2Node(web3Provider))

    const handler = new DefaultWeb3Handler(messageSubmitter, nodeContext)
    const blockNumber = await nodeContext.provider.getBlockNumber()
    handler.blockTimestamps[blockNumber] = timestamp
    return handler
  }

  protected constructor(
    protected readonly messageSubmitter: L2ToL1MessageSubmitter,
    protected readonly context: L2NodeContext
  ) {
    this.lock = new AsyncLock()
  }

  public getL2ToL1MessagePasserAddress(): Address {
    return this.context.l2ToL1MessagePasser.address
  }

  /**
   * Handles generic Web3 requests.
   *
   * @param method The Web3 method being requested.
   * @param params The parameters for the method in question.
   *
   * @returns The response if the method is supported and properly formatted.
   * @throws If the method is not supported or request is improperly formatted.
   */
  public async handleRequest(method: string, params: any[]): Promise<string> {
    log.debug(
      `Handling request, method: [${method}], params: [${JSON.stringify(
        params
      )}]`
    )

    // Make sure the method is available
    let response: any
    switch (method) {
      case Web3RpcMethods.blockNumber:
        this.assertParameters(params, [])
        response = await this.blockNumber()
        break
      case Web3RpcMethods.call:
        this.assertParameters(params, [
          Web3RpcTypes.object,
          Web3RpcTypes.quantityOrTag,
        ])
        response = await this.call(params[0], params[1] || latestBlock)
        break
      case Web3RpcMethods.estimateGas:
        this.assertParameters(params, [
          Web3RpcTypes.object,
          Web3RpcTypes.quantityOrTag,
        ])
        response = await this.estimateGas(params[0], params[1] || latestBlock)
        break
      case Web3RpcMethods.gasPrice:
        this.assertParameters(params, [])
        response = await this.gasPrice()
        break
      case Web3RpcMethods.getBlockByNumber:
        this.assertParameters(params, [
          Web3RpcTypes.quantityOrTag,
          Web3RpcTypes.boolean,
        ])
        response = await this.getBlockByNumber(params[0], params[1])
        break
      case Web3RpcMethods.getBlockByHash:
        this.assertParameters(params, [Web3RpcTypes.data, Web3RpcTypes.boolean])
        response = await this.getBlockByHash(params[0], params[1])
        break
      case Web3RpcMethods.getBalance:
        this.assertParameters(
          params,
          [Web3RpcTypes.address, Web3RpcTypes.quantityOrTag],
          latestBlock
        )
        response = await this.getBalance()
        break
      case Web3RpcMethods.getCode:
        this.assertParameters(params, [
          Web3RpcTypes.data,
          Web3RpcTypes.quantityOrTag,
        ])
        response = await this.getCode(params[0], params[1] || latestBlock)
        break
      case Web3RpcMethods.getExecutionManagerAddress:
        this.assertParameters(params, [])
        response = await this.getExecutionManagerAddress()
        break
      case Web3RpcMethods.getLogs:
        this.assertParameters(params, [Web3RpcTypes.object])
        response = await this.getLogs(params[0])
        break
      case Web3RpcMethods.getTransactionByHash:
        this.assertParameters(params, [Web3RpcTypes.data])
        response = await this.getTransactionByHash(params[0])
        break
      case Web3RpcMethods.getTransactionCount:
        this.assertParameters(params, [
          Web3RpcTypes.data,
          Web3RpcTypes.quantityOrTag,
        ])
        response = await this.getTransactionCount(
          params[0],
          params[1] || latestBlock
        )
        break
      case Web3RpcMethods.getTransactionReceipt:
        this.assertParameters(params, [Web3RpcTypes.data])
        response = await this.getTransactionReceipt(params[0])
        break
      case Web3RpcMethods.sendRawTransaction:
        this.assertParameters(params, [Web3RpcTypes.data])
        response = await this.sendRawTransaction(params[0])
        break
      case Web3RpcMethods.networkVersion:
        this.assertParameters(params, [])
        response = await this.networkVersion()
        break
      case Web3RpcMethods.chainId:
        this.assertParameters(params, [])
        response = await this.chainId()
        break
      default:
        const msg: string = `Method / params [${method} / ${JSON.stringify(
          params
        )}] is not supported by this Web3 handler!`
        log.debug(msg)
        throw new UnsupportedMethodError(msg)
    }

    log.debug(
      `Request: method [${method}], params: [${JSON.stringify(
        params
      )}], got result: [${JSON.stringify(response)}]`
    )
    return response
  }

  public async blockNumber(): Promise<string> {
    log.debug(`Requesting block number.`)
    const response = await this.context.provider.send(
      Web3RpcMethods.blockNumber,
      []
    )
    // For now we will just use the internal node's blocknumber.
    // TODO: Add rollup block tracking
    log.debug(`Received block number [${response}].`)
    return response
  }

  public async call(txObject: {}, defaultBlock: string): Promise<string> {
    log.debug(
      `Making eth_call: [${JSON.stringify(
        txObject
      )}], defaultBlock: [${defaultBlock}]`
    )
    // TODO allow executing a call without a from address
    // Currently using a dummy default from_address
    if (!txObject['from']) {
      txObject['from'] = '0x' + '88'.repeat(20)
    }
    // First generate the internalTx calldata
    const internalCalldata = this.getTransactionCalldata(
      this.getTimestamp(),
      0,
      txObject['to'],
      txObject['data'],
      txObject['from'],
      ZERO_ADDRESS,
      true
    )

    log.debug(`calldata: ${internalCalldata}`)

    let response
    try {
      // Then actually make the call and get the response
      response = await this.context.provider.send(Web3RpcMethods.call, [
        {
          from: this.context.wallet.address,
          to: this.context.executionManager.address,
          data: internalCalldata,
        },
        defaultBlock,
      ])
    } catch (e) {
      log.debug(
        `Internal error executing call: ${JSON.stringify(
          txObject
        )}, default block: ${defaultBlock}, error: ${JSON.stringify(e)}`
      )
      if (isErrorEVMRevert(e)) {
        log.debug(
          `Internal error appears to be an EVM revert, surfacing revert message up...`
        )
        throw new RevertError(e.message as string)
      }
      throw e
    }

    // Now just return the response!
    log.debug(
      `eth_call with request: [${JSON.stringify(
        txObject
      )}] default block: ${defaultBlock} got response [${response}]`
    )
    return response
  }

  public async estimateGas(
    txObject: {},
    defaultBlock: string
  ): Promise<string> {
    log.debug(
      `Estimating gas: [${JSON.stringify(
        txObject
      )}], defaultBlock: [${defaultBlock}]`
    )
    // First generate the internalTx calldata
    const internalCalldata = this.getTransactionCalldata(
      this.getTimestamp(),
      0,
      txObject['to'],
      txObject['data'],
      txObject['from'],
      ZERO_ADDRESS,
      true
    )

    log.debug(internalCalldata)
    // Then estimate the gas
    const response = await this.context.provider.send(
      Web3RpcMethods.estimateGas,
      [
        {
          from: this.context.wallet.address,
          to: this.context.executionManager.address,
          data: internalCalldata,
        },
      ]
    )
    // TODO: Make sure gas limit is below max
    log.debug(
      `Estimated gas: request: [${JSON.stringify(
        txObject
      )}] default block: ${defaultBlock} got response [${response}]`
    )
    return add0x(GAS_LIMIT.toString(16))
  }

  public async gasPrice(): Promise<string> {
    // Gas price is always zero
    return '0x0'
  }

  public async getBalance(): Promise<string> {
    // Balances are always zero
    return '0x0'
  }

  public async getBlockByNumber(
    defaultBlock: string,
    fullObjects: boolean
  ): Promise<any> {
    log.debug(`Got request to get block ${defaultBlock}.`)
    const res: object = await this.context.provider.send(
      Web3RpcMethods.getBlockByNumber,
      [defaultBlock, fullObjects]
    )
    const block = this.parseInternalBlock(res, fullObjects)

    log.debug(
      `Returning block ${defaultBlock} (fullObj: ${fullObjects}): ${JSON.stringify(
        block
      )}`
    )

    return block
  }

  public async getBlockByHash(
    blockHash: string,
    fullObjects: boolean
  ): Promise<any> {
    log.debug(`Got request to get block ${blockHash}.`)
    const res: object = await this.context.provider.send(
      Web3RpcMethods.getBlockByHash,
      [blockHash, fullObjects]
    )
    const block = this.parseInternalBlock(res, fullObjects)

    log.debug(
      `Returning block ${blockHash} (fullObj: ${fullObjects}): ${JSON.stringify(
        block
      )}`
    )

    return block
  }

  public async parseInternalBlock(
    block: object,
    fullObjects: boolean
  ): Promise<object> {
    if (!block) {
      return block
    }

    log.debug(`Parsing block #${block['number']}: ${JSON.stringify(block)}`)

    if (this.blockTimestamps[block['number']]) {
      block['timestamp'] = numberToHexString(
        this.blockTimestamps[block['number']]
      )
    }
    if (fullObjects) {
      block['transactions'] = (
        await Promise.all(
          block['transactions'].map(async (transaction) => {
            transaction['hash'] = await this.getOvmTxHash(transaction['hash'])
            const ovmTx = await this.getTransactionByHash(transaction['hash'])
            Object.keys(transaction).forEach((key) => {
              if (ovmTx && ovmTx[key]) {
                transaction[key] = utils.BigNumber.isBigNumber(ovmTx[key])
                  ? ovmTx[key].toNumber()
                  : ovmTx[key]
              }
              if (typeof transaction[key] === 'number') {
                transaction[key] = numberToHexString(transaction[key])
              }
            })

            return transaction
          })
        )
      )
        // Filter transactions that aren't included in the execution manager
        .filter((transaction) => transaction['hash'] !== add0x('00'.repeat(32)))
    } else {
      block['transactions'] = await Promise.all(
        block['transactions'].map(async (transactionHash) =>
          this.getOvmTxHash(transactionHash)
        )
      )
    }

    const logsBloom = new BloomFilter()
    await Promise.all(
      block['transactions'].map(async (transactionOrHash) => {
        const transactionHash = fullObjects
          ? transactionOrHash.hash
          : transactionOrHash
        if (transactionHash) {
          const receipt = await this.getTransactionReceipt(transactionHash)
          if (receipt && receipt.logsBloom) {
            logsBloom.or(new BloomFilter(hexStrToBuf(receipt.logsBloom)))
          }
        }
      })
    )
    block['logsBloom'] = bufToHexString(logsBloom.bitvector)

    log.debug(
      `Transforming block #${block['number']} complete: ${JSON.stringify(
        block
      )}`
    )

    return block
  }
  public async getCode(
    address: Address,
    defaultBlock: string
  ): Promise<string> {
    const curentBlockNumber = await this.context.provider.getBlockNumber()
    if (
      !['latest', numberToHexString(curentBlockNumber)].includes(defaultBlock)
    ) {
      log.debug(
        `Historical code lookups aren't supported. defaultBlock: [${hexStrToNumber(
          defaultBlock
        )}] curentBlockNumber:[${curentBlockNumber}]`
      )
      throw new InvalidParametersError(
        `Historical code lookups aren't supported. Requested Block: ${hexStrToNumber(
          defaultBlock
        )} Current Block: ${curentBlockNumber}`
      )
    }
    log.debug(
      `Getting code for address: [${address}], defaultBlock: [${defaultBlock}]`
    )
    // First get the code contract address at the requested OVM address
    const codeContractAddress = await this.context.executionManager.getCodeContractAddress(
      address
    )
    const response = await this.context.provider.send(Web3RpcMethods.getCode, [
      codeContractAddress,
      'latest',
    ])
    log.debug(
      `Got code for address [${address}], block [${defaultBlock}]: [${response}]`
    )
    return response
  }

  public async getExecutionManagerAddress(): Promise<Address> {
    return this.context.executionManager.address
  }

  public async getLogs(ovmFilter: any): Promise<any[]> {
    const filter = JSON.parse(JSON.stringify(ovmFilter))
    // We cannot filter out execution manager events or else convertInternalLogsToOvmLogs will break.  So add EM address to address filter
    if (filter['address']) {
      if (!Array.isArray(filter['address'])) {
        filter['address'] = [filter['address']]
      }
      const codeContractAddresses = []
      for (const address of filter['address']) {
        codeContractAddresses.push(
          await this.context.executionManager.getCodeContractAddress(address)
        )
      }
      filter['address'] = [
        ...codeContractAddresses,
        this.context.executionManager.address,
      ]
    }
    // We cannot filter out execution manager events or else convertInternalLogsToOvmLogs will break.  So add EM topics to topics filter
    if (filter['topics']) {
      if (filter['topics'].length > 1) {
        // todo make this proper error
        const msg = `The provided filter ${JSON.stringify(
          filter
        )} has multiple levels of topic filter.  Multi-level topic filters are currently unsupported by the OVM.`
        throw new UnsupportedFilterError(msg)
      }
      if (!Array.isArray(filter['topics'][0])) {
        filter['topics'][0] = [JSON.parse(JSON.stringify(filter['topics'][0]))]
      }
      filter['topics'][0].push(...ALL_EXECUTION_MANAGER_EVENT_TOPICS)
    }
    log.debug(
      `Converted ovm filter ${JSON.stringify(
        ovmFilter
      )} to internal filter ${JSON.stringify(filter)}`
    )

    const res = await this.context.provider.send(Web3RpcMethods.getLogs, [
      filter,
    ])

    let logs = JSON.parse(
      JSON.stringify(
        convertInternalLogsToOvmLogs(res, this.context.executionManager.address)
      )
    )
    log.debug(
      `Log result: [${JSON.stringify(logs)}], filter: [${JSON.stringify(
        filter
      )}].`
    )
    logs = await Promise.all(
      logs.map(async (logItem, index) => {
        logItem['transactionHash'] = await this.getOvmTxHash(
          logItem['transactionHash']
        )
        const transaction = await this.getTransactionByHash(
          logItem['transactionHash']
        )
        if (transaction['to'] === null) {
          const receipt = await this.getTransactionReceipt(transaction.hash)
          transaction['to'] = receipt.contractAddress
        }
        if (typeof logItem['logIndex'] === 'number') {
          logItem['logIndex'] = numberToHexString(logItem['logIndex'])
        }
        return logItem
      })
    )

    return logs
  }

  public async getTransactionByHash(ovmTxHash: string): Promise<any> {
    log.debug('Getting tx for ovm tx hash:', ovmTxHash)
    // First convert our ovmTxHash into an internalTxHash
    const signedOvmTx: string = await this.getOvmTransactionByHash(ovmTxHash)

    if (!remove0x(signedOvmTx)) {
      log.debug(`There is no OVM tx associated with OVM tx hash [${ovmTxHash}]`)
      return null
    }

    log.debug(
      `OVM tx hash [${ovmTxHash}] is associated with signed OVM tx [${signedOvmTx}]`
    )

    const ovmTx = utils.parseTransaction(signedOvmTx)

    log.debug(
      `OVM tx hash [${ovmTxHash}] is associated with parsed OVM tx [${JSON.stringify(
        ovmTx
      )}]`
    )

    return ovmTx
  }

  public async getTransactionCount(
    address: Address,
    defaultBlock: string
  ): Promise<string> {
    log.debug(
      `Requesting transaction count. Address [${address}], block: [${defaultBlock}].`
    )
    const ovmContractNonce = await this.context.executionManager.getOvmContractNonce(
      address
    )
    const response = add0x(ovmContractNonce.toNumber().toString(16))
    log.debug(
      `Received transaction count for Address [${address}], block: [${defaultBlock}]: [${response}].`
    )
    return response
  }

  public async getTransactionReceipt(
    ovmTxHash: string,
    includeRevertMessage: boolean = false
  ): Promise<any> {
    log.debug('Getting tx receipt for ovm tx hash:', ovmTxHash)
    // First convert our ovmTxHash into an internalTxHash
    const internalTxHash = await this.getInternalTxHash(ovmTxHash)

    log.debug(
      `Got internal hash [${internalTxHash}] for ovm hash [${ovmTxHash}]`
    )

    const internalTxReceipt = await this.context.provider.send(
      Web3RpcMethods.getTransactionReceipt,
      [internalTxHash]
    )

    if (!internalTxReceipt) {
      log.debug(`No tx receipt found for ovm tx hash [${ovmTxHash}]`)
      return null
    }

    log.debug(
      `Converting internal tx receipt to ovm receipt, internal receipt is:`,
      internalTxReceipt
    )

    // if there are no logs, the tx must have failed, as the Execution Mgr always logs stuff
    const txSucceeded: boolean = internalTxReceipt.logs.length !== 0
    let ovmTxReceipt
    if (txSucceeded) {
      log.debug(
        `The internal tx previously succeeded for this OVM tx, converting internal receipt to OVM receipt...`
      )
      ovmTxReceipt = await internalTxReceiptToOvmTxReceipt(
        internalTxReceipt,
        this.context.executionManager.address,
        ovmTxHash
      )
    } else {
      log.debug(
        `Internal tx previously failed for this OVM tx, creating receipt from the OVM tx itself.`
      )
      ovmTxReceipt = internalTxReceipt
      ovmTxReceipt.transactionHash = ovmTxHash
      ovmTxReceipt.logs = []
    }
    const ovmTx = await this.getTransactionByHash(ovmTxReceipt.transactionHash)
    log.debug(`got OVM tx from hash: [${JSON.stringify(ovmTx)}]`)
    ovmTxReceipt.to = ovmTx.to ? ovmTx.to : ovmTxReceipt.to
    ovmTxReceipt.from = ovmTx.from

    if (ovmTxReceipt.revertMessage !== undefined && !includeRevertMessage) {
      delete ovmTxReceipt.revertMessage
    }
    if (typeof ovmTxReceipt.status === 'number') {
      ovmTxReceipt.status = numberToHexString(ovmTxReceipt.status)
    }

    log.debug(
      `Returning tx receipt for ovm tx hash [${ovmTxHash}]: [${JSON.stringify(
        ovmTxReceipt
      )}]`
    )
    return ovmTxReceipt
  }

  public async networkVersion(): Promise<string> {
    log.debug('Getting network version')
    // Return our internal chain_id
    // TODO: Add getter for chainId that is not just imported
    const response = CHAIN_ID
    log.debug(`Got network version: [${response}]`)
    return response.toString()
  }

  public async chainId(): Promise<string> {
    log.debug('Getting chain ID')
    // Return our internal chain_id
    // TODO: Add getter for chainId that is not just imported
    const response = add0x(CHAIN_ID.toString(16))
    log.debug(`Got chain ID: [${response}]`)
    return response
  }

  public async sendRawTransaction(
    rawOvmTx: string,
    fromAddressOverride?: string
  ): Promise<string> {
    const debugTime = new Date().getTime()
    log.debug('Sending raw transaction with params:', rawOvmTx)
    return this.lock.acquire(lockKey, async () => {
      log.debug(
        `Send tx lock acquired. Waited ${new Date().getTime() -
          debugTime}ms for lock.`
      )
      const blockTimestamp = this.getTimestamp()

      // Decode the OVM transaction -- this will be used to construct our internal transaction
      const ovmTx = utils.parseTransaction(rawOvmTx)
      // override the from address if in testing mode
      if (!!fromAddressOverride) {
        ovmTx.from = fromAddressOverride
      }
      log.debug(
        `OVM Transaction being parsed ${rawOvmTx}, with from address override of [${fromAddressOverride}], parsed: ${JSON.stringify(
          ovmTx
        )}`
      )

      // Convert the OVM transaction into an "internal" tx which we can use for our execution manager
      const internalTx = await this.ovmTxToInternalTx(ovmTx)
      // Now compute the hash of the OVM transaction which we will return
      const ovmTxHash = await utils.keccak256(rawOvmTx)
      const internalTxHash = await utils.keccak256(internalTx)

      log.debug(
        `OVM tx hash: ${ovmTxHash}, internal tx hash: ${internalTxHash}, signed internal tx: ${JSON.stringify(
          internalTx
        )}. Elapsed time: ${new Date().getTime() - debugTime}ms`
      )

      // Make sure we have a way to look up our internal tx hash from the ovm tx hash.
      await this.storeOvmTransaction(ovmTxHash, internalTxHash, rawOvmTx)

      let returnedInternalTxHash: string
      try {
        // Then apply our transaction
        returnedInternalTxHash = await this.context.provider.send(
          Web3RpcMethods.sendRawTransaction,
          [internalTx]
        )
      } catch (e) {
        if (isErrorEVMRevert(e)) {
          log.debug(
            `Internal EVM revert for Ovm tx hash: ${ovmTxHash} and internal hash: ${internalTxHash}.  Incrementing nonce Incrementing nonce for sender (${ovmTx.from}) and surfacing revert message up...`
          )
          await this.context.executionManager.incrementNonce(add0x(ovmTx.from))
          log.debug(`Nonce incremented successfully for ${ovmTx.from}.`)
          throw new RevertError(e.message as string)
        }
        logError(
          log,
          `Non-revert error executing internal transaction! Ovm tx hash: ${ovmTxHash}, internal hash: ${internalTxHash}. Returning generic internal error.`,
          e
        )
        throw e
      }

      if (remove0x(internalTxHash) !== remove0x(returnedInternalTxHash)) {
        const msg: string = `Internal Transaction hashes do not match for OVM Hash: [${ovmTxHash}]. Calculated: [${internalTxHash}], returned from tx: [${returnedInternalTxHash}]`
        log.error(msg)
        throw Error(msg)
      }

      log.debug(
        `OVM tx with hash ${ovmTxHash} sent. Elapsed time: ${new Date().getTime() -
          debugTime}ms`
      )

      this.context.provider
        .waitForTransaction(internalTxHash)
        .then(async () => {
          const receipt: OvmTransactionReceipt = await this.getTransactionReceipt(
            ovmTxHash,
            true
          )
          log.debug(
            `Transaction receipt for ${rawOvmTx}: ${JSON.stringify(receipt)}`
          )
          if (!receipt) {
            log.error(`Unable to find receipt for raw ovm tx: ${rawOvmTx}`)
            return
          } else if (!receipt.status) {
            log.debug(`Transaction reverted: ${rawOvmTx}`)
          } else {
            log.debug(`Transaction mined successfully: ${rawOvmTx}`)
            await this.processTransactionEvents(receipt)
          }
          this.blockTimestamps[receipt.blockNumber] = blockTimestamp
        })

      log.debug(
        `Completed send raw tx [${rawOvmTx}]. Response: [${ovmTxHash}]. Total time: ${new Date().getTime() -
          debugTime}ms`
      )
      // Return the *OVM* tx hash. We can do this because we store a mapping to the ovmTxHashs in the EM contract.
      return ovmTxHash
    })
  }

  /**
   * @inheritDoc
   */
  public async handleL1ToL2Transaction(
    transaction: L1ToL2Transaction
  ): Promise<void> {
    log.debug(`Executing L1 to L2 Transaction ${JSON.stringify(transaction)}`)

    const calldata = this.context.executionManager.interface.functions[
      'executeTransaction'
    ].encode([
      this.getTimestamp(),
      0,
      transaction.target,
      transaction.callData,
      ZERO_ADDRESS,
      transaction.sender,
      false,
    ])

    const signedTx = await this.getSignedTransaction(
      calldata,
      this.context.executionManager.address
    )
    const receipt = await this.context.provider.sendTransaction(signedTx)

    log.debug(
      `L1 to L2 Transaction submitted. Tx hash: ${
        receipt.hash
      }. Tx: ${JSON.stringify(transaction)}`
    )
    let txReceipt: TransactionReceipt
    try {
      txReceipt = await this.context.provider.waitForTransaction(receipt.hash)
    } catch (e) {
      logError(
        log,
        `Error submitting L1 to L2 transaction to L2 node. Tx Hash: ${
          receipt.hash
        }, Tx: ${JSON.stringify(transaction)}`,
        e
      )
      throw e
    }
    log.debug(`L1 to L2 Transaction applied to L2. Tx hash: ${receipt.hash}`)

    try {
      const ovmTxReceipt: OvmTransactionReceipt = await internalTxReceiptToOvmTxReceipt(
        txReceipt,
        this.context.executionManager.address
      )
      await this.processTransactionEvents(ovmTxReceipt)
    } catch (e) {
      logError(
        log,
        `Error processing L1 to L2 transaction events. Tx Hash: ${
          receipt.hash
        }, Tx: ${JSON.stringify(transaction)}`,
        e
      )
    }
  }

  /**
   * Gets the current number of seconds since the epoch.
   *
   * @returns The seconds since epoch.
   */
  protected getTimestamp(): number {
    return getCurrentTime()
  }

  protected getNewWallet(): Wallet {
    return Wallet.createRandom().connect(this.context.provider)
  }

  private async processTransactionEvents(
    receipt: OvmTransactionReceipt
  ): Promise<void> {
    const messagePromises: Array<Promise<void>> = []
    for (const logEntry of receipt.logs.filter(
      (x) =>
        remove0x(x.address) ===
        remove0x(this.context.l2ToL1MessagePasser.address)
    )) {
      const parsedLog = l2ToL1MessagePasserInterface.parseLog(logEntry)
      log.debug(`parsed log: ${JSON.stringify(parsedLog)}.`)
      if (!parsedLog || parsedLog.name !== 'L2ToL1Message') {
        continue
      }

      const nonce: number = parsedLog.values['_nonce'].toNumber()
      const ovmSender: string = parsedLog.values['_ovmSender']
      const callData: string = parsedLog.values['_callData']
      const message: L2ToL1Message = {
        nonce,
        ovmSender,
        callData,
      }
      log.debug(`Submitting L2 to L1 Message: ${JSON.stringify(message)}`)
      messagePromises.push(this.messageSubmitter.submitMessage(message))
    }

    if (!!messagePromises.length) {
      await Promise.all(messagePromises)
    }
  }

  /**
   * Maps the provided OVM transaction hash to the provided internal transaction hash by storing it in our
   * L2 Execution Manager contract.
   *
   * @param ovmTxHash The OVM transaction's hash.
   * @param internalTxHash Our internal transactions's hash.
   * @throws if not stored properly
   */
  private async storeOvmTransaction(
    ovmTxHash: string,
    internalTxHash: string,
    signedOvmTransaction: string
  ): Promise<void> {
    log.debug(
      `Mapping ovmTxHash: ${ovmTxHash} to internal tx hash: ${internalTxHash}.`
    )

    const calldata: string = this.context.executionManager.interface.functions[
      'storeOvmTransaction'
    ].encode([
      add0x(ovmTxHash),
      add0x(internalTxHash),
      add0x(signedOvmTransaction),
    ])

    const signedTx = this.getSignedTransaction(
      calldata,
      this.context.executionManager.address
    )

    const res = await this.context.provider.sendTransaction(signedTx)
    this.ovmHashToOvmTransactionCache[ovmTxHash] = signedOvmTransaction

    this.context.provider
      .waitForTransaction(res.hash)
      .then((receipt) => {
        log.debug(
          `Got receipt mapping ovm tx hash ${ovmTxHash} to internal tx hash ${internalTxHash}: ${JSON.stringify(
            receipt
          )}`
        )
        delete this.ovmHashToOvmTransactionCache[ovmTxHash]
      })
      .catch((e) => {
        logError(
          log,
          `Error mapping ovmTxHash: ${ovmTxHash} to internal tx hash: ${internalTxHash}. This should never happen!`,
          e
        )
        throw e
      })
  }

  /**
   * Gets the internal EVM transaction hash for the provided OVM transaction hash, if one exists.
   *
   * @param ovmTxHash The OVM transaction hash
   * @returns The EVM tx hash if one exists, else undefined.
   */
  private async getInternalTxHash(ovmTxHash: string): Promise<string> {
    return this.context.executionManager.getInternalTransactionHash(
      add0x(ovmTxHash)
    )
  }

  /**
   * Gets the external OVM transaction hash for the provided EVM transaction hash, if one exists.
   *
   * @param evmTxHash The EVM transaction hash
   * @returns The OVM tx hash if one exists, else undefined.
   */
  private async getOvmTxHash(evmTxHash: string): Promise<string> {
    return this.context.executionManager.getOvmTransactionHash(add0x(evmTxHash))
  }

  /**
   * Gets the signed OVM transaction that we received by its hash.
   *
   * @param ovmTxHash The hash of the signed tx.
   * @returns The signed OVM transaction if one exists, else undefined.
   */
  private async getOvmTransactionByHash(ovmTxHash: string): Promise<string> {
    if (ovmTxHash in this.ovmHashToOvmTransactionCache) {
      return this.ovmHashToOvmTransactionCache[ovmTxHash]
    }
    return this.context.executionManager.getOvmTransaction(add0x(ovmTxHash))
  }

  /**
   * Wraps the provided OVM transaction in a signed EVM transaction capable
   * of execution within the L2 node.
   *
   * @param ovmTx The OVM transaction to wrap
   * @returns The wrapped, signed EVM transaction.
   */
  private async ovmTxToInternalTx(ovmTx: any): Promise<string> {
    // Verify that the transaction is not accidentally sending to the ZERO_ADDRESS
    if (ovmTx.to === ZERO_ADDRESS) {
      throw new InvalidParametersError('Sending to Zero Address disallowed')
    }
    // Get the nonce of the account that we will use to send everything
    // Note: + 1 because all transactions will have a tx hash mapping tx sent before them.
    // Check that this is an EOA transaction, if not we throw until we've
    // implemented non-EOA transactions
    if (ovmTx.v === 0) {
      log.error(
        'Transaction does not have a valid signature! For now we only support calls from EOAs'
      )
      throw new InvalidParametersError('Non-EOA transaction detected')
    }
    // Generate the calldata which we'll use to call our internal execution manager
    // First pull out the `to` field (we just need to check if it's null & if so set ovmTo to the zero address as that's how we deploy contracts)
    const ovmTo = ovmTx.to === null ? ZERO_ADDRESS : ovmTx.to
    const ovmFrom = ovmTx.from === undefined ? ZERO_ADDRESS : ovmTx.from
    // Check the nonce
    const expectedNonce = (
      await this.context.executionManager.getOvmContractNonce(ovmFrom)
    ).toNumber()
    if (expectedNonce !== ovmTx.nonce) {
      throw new InvalidParametersError(
        `Incorrect nonce! Expected nonce: ${expectedNonce} but received nonce: ${ovmTx.nonce}`
      )
    }
    // Construct the raw transaction calldata
    const internalCalldata = this.getTransactionCalldata(
      this.getTimestamp(),
      0,
      ovmTo,
      ovmTx.data,
      ovmFrom,
      ZERO_ADDRESS,
      true
    )

    log.debug(`EOA calldata: [${internalCalldata}]`)

    return this.getSignedTransaction(
      internalCalldata,
      this.context.executionManager.address
    )
  }

  private async getSignedTransaction(
    calldata: string,
    to: string,
    nonce: number = 0,
    gasLimit?: number
  ): Promise<string> {
    const tx = {
      nonce,
      gasPrice: 0,
      gasLimit: GAS_LIMIT,
      to,
      value: 0,
      data: add0x(calldata),
      chainId: CHAIN_ID,
    }
    if (gasLimit !== undefined) {
      tx['gasLimit'] = gasLimit
    }

    return this.getNewWallet().sign(tx)
  }

  /**
   * Get the calldata for an EVM transaction to the ExecutionManager.
   */
  private getTransactionCalldata(
    timestamp: number,
    queueOrigin: number,
    ovmEntrypoint: string,
    callBytes: string,
    fromAddress: string,
    l1TxSenderAddress: string,
    allowRevert: boolean
  ): string {
    // Update the ovmEntrypoint to be the ZERO_ADDRESS if this is a contract creation
    if (ovmEntrypoint === null || ovmEntrypoint === undefined) {
      ovmEntrypoint = ZERO_ADDRESS
    }
    return this.context.executionManager.interface.functions[
      'executeTransaction'
    ].encode([
      timestamp,
      queueOrigin,
      ovmEntrypoint,
      callBytes,
      fromAddress,
      l1TxSenderAddress,
      allowRevert,
    ])
  }

  protected assertParameters(
    params: any[],
    expected: Web3RpcTypes[],
    defaultLast?: any
  ) {
    if (
      !(
        !params ||
        params.length === expected.length - 1 ||
        params.length === expected.length
      )
    ) {
      throw new InvalidParametersError(
        `Expected ${expected} parameters but received ${params.length}.`
      )
    }
    expected.forEach((expectedType, index) => {
      const param = params[index]
      const typeChecks = {
        [Web3RpcTypes.quantityOrTag]: (value) => {
          return (
            value === undefined ||
            !isNaN(value) ||
            ['latest', 'earliest', 'pending'].includes(value)
          )
        },
        [Web3RpcTypes.boolean]: (value) => [true, false].includes(value),
        [Web3RpcTypes.quantity]: (value) => !isNaN(value),
        [Web3RpcTypes.data]: Web3.utils.isHex,
        [Web3RpcTypes.address]: Web3.utils.isAddress,
        [Web3RpcTypes.object]: (value) => {
          return value instanceof Object
        },
      }

      if (!typeChecks[expectedType](param)) {
        throw new InvalidParametersError(
          `Expected ${expectedType} but got ${param}`
        )
      }
    })
  }
}
