/* External Imports */
import { ethers } from 'ethers'

interface Layer {
  provider: any
  messengerAddress: string
}

interface WatcherOptions {
  l1: Layer
  l2: Layer
}

export class Watcher {
  public l1: Layer
  public l2: Layer
  public NUM_BLOCKS_TO_FETCH: number = 10_000_000

  constructor(opts: WatcherOptions) {
    this.l1 = opts.l1
    this.l2 = opts.l2
  }

  public async getMessageHashesFromL1Tx(l1TxHash: string): Promise<string[]> {
    return this._getMessageHashesFromTx(true, l1TxHash)
  }
  public async getMessageHashesFromL2Tx(l2TxHash: string): Promise<string[]> {
    return this._getMessageHashesFromTx(false, l2TxHash)
  }

  public async getL1TransactionReceipt(l2ToL1MsgHash: string): Promise<any> {
    return this._getLXTransactionReceipt(true, l2ToL1MsgHash)
  }

  public async getL2TransactionReceipt(l1ToL2MsgHash: string): Promise<any> {
    return this._getLXTransactionReceipt(false, l1ToL2MsgHash)
  }

  private async _getMessageHashesFromTx(
    isL1: boolean,
    txHash: string
  ): Promise<string[]> {
    const layer = isL1 ? this.l1 : this.l2
    const receipt = await layer.provider.getTransactionReceipt(txHash)
    const msgHashes = []
    for (const log of receipt.logs) {
      if (
        log.address === layer.messengerAddress &&
        log.topics[0] === ethers.utils.id('SentMessage(bytes)')
      ) {
        msgHashes.push(ethers.utils.solidityKeccak256(['bytes'], [log.data]))
      }
    }
    return msgHashes
  }

  public async _getLXTransactionReceipt(
    isL1: boolean,
    msgHash: string
  ): Promise<any> {
    const layer = isL1 ? this.l1 : this.l2
    const blockNumber = await layer.provider.getBlockNumber()
    const startingBlock = Math.max(blockNumber - this.NUM_BLOCKS_TO_FETCH, 0)
    const filter = {
      address: layer.messengerAddress,
      topics: [ethers.utils.id(`RelayedMessage(bytes32)`)],
      fromBlock: startingBlock,
    }
    const logs = await layer.provider.getLogs(filter)
    const matches = logs.filter((log: any) => log.data === msgHash)

    // Message was relayed in the past
    if (matches.length > 0) {
      if (matches.length > 1) {
        throw Error(
          'Found multiple transactions relaying the same message hash.'
        )
      }
      return layer.provider.getTransactionReceipt(matches[0].transactionHash)
    }

    // Message has yet to be relayed
    return new Promise(async (resolve, reject) => {
      layer.provider.on(filter, async (log: any) => {
        if (log.data === msgHash) {
          try {
            const txReceipt = await layer.provider.getTransactionReceipt(
              log.transactionHash
            )
            resolve(txReceipt)
          } catch (e) {
            reject(e)
          }
        }
      })
    })
  }
}
