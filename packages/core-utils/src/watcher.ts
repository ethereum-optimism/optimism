/* External Imports */
import { ethers } from 'ethers'
import { Provider, TransactionReceipt } from '@ethersproject/abstract-provider'

const SENT_MESSAGE = ethers.utils.id('SentMessage(bytes)')
const RELAYED_MESSAGE = ethers.utils.id(`RelayedMessage(bytes32)`)
const FAILED_RELAYED_MESSAGE = ethers.utils.id(`FailedRelayedMessage(bytes32)`)

export interface Layer {
  provider: Provider
  messengerAddress: string
  blocksToFetch?: number
}

export interface WatcherOptions {
  l1: Layer
  l2: Layer
  pollInterval?: number
  blocksToFetch?: number
  pollForPending?: boolean
}

export class Watcher {
  public l1: Layer
  public l2: Layer
  public pollInterval = 3000
  public blocksToFetch = 1500
  public pollForPending = true

  constructor(opts: WatcherOptions) {
    this.l1 = opts.l1
    this.l2 = opts.l2
    if (typeof opts.pollInterval === 'number') {
      this.pollInterval = opts.pollInterval
    }
    if (typeof opts.blocksToFetch === 'number') {
      this.blocksToFetch = opts.blocksToFetch
    }
    if (typeof opts.pollForPending === 'boolean') {
      this.pollForPending = opts.pollForPending
    }
  }

  public async getMessageHashesFromL1Tx(l1TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l1, l1TxHash)
  }
  public async getMessageHashesFromL2Tx(l2TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l2, l2TxHash)
  }

  public async getL1TransactionReceipt(
    l2ToL1MsgHash: string,
    pollForPending?
  ): Promise<TransactionReceipt> {
    return this.getTransactionReceipt(this.l1, l2ToL1MsgHash, pollForPending)
  }

  public async getL2TransactionReceipt(
    l1ToL2MsgHash: string,
    pollForPending?
  ): Promise<TransactionReceipt> {
    return this.getTransactionReceipt(this.l2, l1ToL2MsgHash, pollForPending)
  }

  public async getMessageHashesFromTx(
    layer: Layer,
    txHash: string
  ): Promise<string[]> {
    const receipt = await layer.provider.getTransactionReceipt(txHash)
    if (!receipt) {
      return []
    }

    const msgHashes = []
    for (const log of receipt.logs) {
      if (
        log.address === layer.messengerAddress &&
        log.topics[0] === SENT_MESSAGE
      ) {
        const [message] = ethers.utils.defaultAbiCoder.decode(
          ['bytes'],
          log.data
        )
        msgHashes.push(ethers.utils.solidityKeccak256(['bytes'], [message]))
      }
    }
    return msgHashes
  }

  public async getTransactionReceipt(
    layer: Layer,
    msgHash: string,
    pollForPending?
  ): Promise<TransactionReceipt> {
    if (typeof pollForPending !== 'boolean') {
      pollForPending = this.pollForPending
    }

    let matches: ethers.providers.Log[] = []

    let blocksToFetch = layer.blocksToFetch
    if (typeof blocksToFetch !== 'number') {
      blocksToFetch = this.blocksToFetch
    }

    // scan for transaction with specified message
    while (matches.length === 0) {
      const blockNumber = await layer.provider.getBlockNumber()
      const startingBlock = Math.max(blockNumber - blocksToFetch, 0)
      const successFilter: ethers.providers.Filter = {
        address: layer.messengerAddress,
        topics: [RELAYED_MESSAGE],
        fromBlock: startingBlock,
      }
      const failureFilter: ethers.providers.Filter = {
        address: layer.messengerAddress,
        topics: [FAILED_RELAYED_MESSAGE],
        fromBlock: startingBlock,
      }
      const successLogs = await layer.provider.getLogs(successFilter)
      const failureLogs = await layer.provider.getLogs(failureFilter)
      const logs = successLogs.concat(failureLogs)
      matches = logs.filter((log: ethers.providers.Log) => log.data === msgHash)

      // exit loop after first iteration if not polling
      if (!pollForPending) {
        break
      }

      // pause awhile before trying again
      await new Promise((r) => setTimeout(r, this.pollInterval))
    }

    // Message was relayed in the past
    if (matches.length > 0) {
      if (matches.length > 1) {
        throw Error(
          'Found multiple transactions relaying the same message hash.'
        )
      }
      return layer.provider.getTransactionReceipt(matches[0].transactionHash)
    } else {
      return Promise.resolve(undefined)
    }
  }
}
