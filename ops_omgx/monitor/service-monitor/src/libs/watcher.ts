/* External Imports */
import { ethers } from 'ethers'
import { Provider, TransactionReceipt } from '@ethersproject/abstract-provider'
import logger from '../logger'

export interface Layer {
  provider: Provider
  messengerAddress: string
}

export interface WatcherOptions {
  l1: Layer
  l2: Layer
}

export class Watcher {

  public NUM_BLOCKS_TO_FETCH: number = 10000

  public l1: Layer
  public l2: Layer

  constructor (opts: WatcherOptions) {
    this.l1 = opts.l1
    this.l2 = opts.l2
  }

  public async getMessageHashesFromL1Tx (l1TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l1, l1TxHash)
  }

  public async getMessageHashesFromL2Tx (l2TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l2, l2TxHash)
  }

  public async getL1TransactionReceipt (
    l2ToL1MsgHash: string,
    pollForPending: boolean = true
  ): Promise<TransactionReceipt> {
    logger.debug(' Calling getL1TransactionReceipt')
    return this.getTransactionReceipt(this.l1, l2ToL1MsgHash, pollForPending)
  }

  public async getL2TransactionReceipt (
    l1ToL2MsgHash: string,
    pollForPending: boolean = true
  ): Promise<TransactionReceipt> {
    logger.debug('Calling getL2TransactionReceipt')
    return this.getTransactionReceipt(this.l2, l1ToL2MsgHash, pollForPending)
  }

  public async getMessageHashesFromTx (
    layer: Layer,
    txHash: string
  ): Promise<string[]> {

    const receipt = await layer.provider.getTransactionReceipt(txHash)

    if (!receipt) {
      logger.debug('No receipt for txHash', { txHash })
      return []
    }

    const msgHashes = []

    logger.debug('Interate over ' + receipt.logs.length + ' of logs for txHash ' + txHash)
    for (const log of receipt.logs) {
      logger.debug('log.address', { address: log.address, messengerAddress: layer.messengerAddress })
      if (
        log.address === layer.messengerAddress &&
        log.topics[0] === ethers.utils.id('SentMessage(bytes)')
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

  public async getTransactionReceipt (
    layer: Layer,
    msgHash: string,
    pollForPending: boolean = true
  ): Promise<TransactionReceipt> {

    logger.debug('Watcher::getTransactionReceipt')

    const blockNumber = await layer.provider.getBlockNumber()
    const startingBlock = Math.max(blockNumber - this.NUM_BLOCKS_TO_FETCH, 0)

    logger.debug('Layer:', { layer })

    logger.debug('Address: ' + layer.messengerAddress)
    logger.debug('topic: ' + ethers.utils.id(`RelayedMessage(bytes32)`))
    logger.debug('fromBlock: ' + startingBlock)

    const filter = {
      address: layer.messengerAddress,
      topics: [ethers.utils.id(`RelayedMessage(bytes32)`)],
      fromBlock: startingBlock,
    }

    const logs = await layer.provider.getLogs(filter)
    logger.debug('Looking for: ' + msgHash)
    // logger.debug("Current logs:", logs)

    const matches = logs.filter((log: any) => log.data === msgHash)

    // Message was relayed in the past
    if (matches.length > 0) {
      if (matches.length > 1) {
        throw Error(
          ' Found multiple transactions relaying the same message hash.'
        )
      }
      return layer.provider.getTransactionReceipt(matches[0].transactionHash)
    }

    if (!pollForPending) {
      return Promise.resolve(undefined)
    }

    // Message has yet to be relayed, poll until it is found
    return new Promise(async (resolve, reject) => {
      logger.debug('Watcher polling::layer.provider.getTransactionReceipt pre filter')

      // check timeout
      const timeout = parseInt(process.env.DUMMY_TIMEOUT_MINS, 10) || 5
      let isFound = false
      setTimeout(() => {
        if (!isFound) reject(Error('Timeout'))
      }, timeout * 60 * 1000)

      // listener that triggers on filter event
      layer.provider.on(filter, async (log: any) => {
        logger.debug('Watcher polling::layer.provider.getTransactionReceipt post filter')
        logger.debug(log)
        if (log.data === msgHash) {
          logger.debug('FOUND')
          isFound = true
          try {
            const txReceipt = await layer.provider.getTransactionReceipt(log.transactionHash)
            layer.provider.off(filter)
            resolve(txReceipt)
          } catch (e) {
            reject(e)
          }
        }
      })
    })
  }
}
