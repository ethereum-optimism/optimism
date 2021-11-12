/* External Imports */
import { ethers } from 'ethers'
import { Provider, TransactionReceipt } from '@ethersproject/abstract-provider'
import { getContractInterface } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'

export interface Layer {
  provider: Provider
  messengerAddress: string
}

export interface WatcherOptions {
  l1: Layer
  l2: Layer
  pollInterval?: number
  pollForPending?: boolean
}

/**
 * Utility function for computing the hash of an L1 <> L2 message.
 *
 * @param message Message to hash.
 * @returns Computed hash of the message.
 */
export const computeMessageHash = (message: {
  target: string
  sender: string
  message: string
  messageNonce: number
}): string => {
  const iface = getContractInterface('L2CrossDomainMessenger')
  return ethers.utils.solidityKeccak256(
    ['bytes'],
    [
      iface.encodeFunctionData('relayMessage', [
        message.target,
        message.sender,
        message.message,
        message.messageNonce,
      ]),
    ]
  )
}

export class Watcher {
  public l1: Layer
  public l2: Layer
  public pollInterval = 3000
  public pollForPending = true

  constructor(opts: WatcherOptions) {
    this.l1 = opts.l1
    this.l2 = opts.l2
    if (typeof opts.pollInterval === 'number') {
      this.pollInterval = opts.pollInterval
    }
    if (typeof opts.pollForPending === 'boolean') {
      this.pollForPending = opts.pollForPending
    }
  }

  /**
   * Pulls all L1 => L2 message hashes out of an L1 transaction by hash.
   *
   * @param l1TxHash Hash of the L1 transaction to find messages for.
   * @returns List of message hashes emitted in the transaction.
   */
  public async getMessageHashesFromL1Tx(l1TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l1, l1TxHash)
  }

  /**
   * Pulls all L2 => L1 message hashes out of an L2 transaction by hash.
   *
   * @param l2TxHash Hash of the L2 transaction to find messages for.
   * @returns List of message hashes emitted in the transaction.
   */
  public async getMessageHashesFromL2Tx(l2TxHash: string): Promise<string[]> {
    return this.getMessageHashesFromTx(this.l2, l2TxHash)
  }

  /**
   * Finds the receipt of the L1 transaction that relayed a given L2 => L1 message hash.
   *
   * @param l2ToL1MsgHash Hash of the L2 => L1 message to find the receipt for.
   * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
   * @returns Receipt of the L1 transaction that relayed the message.
   */
  public async getL1TransactionReceipt(
    l2ToL1MsgHash: string,
    pollForPending?: boolean
  ): Promise<TransactionReceipt> {
    return this.getTransactionReceipt(this.l1, l2ToL1MsgHash, pollForPending)
  }

  /**
   * Finds the receipt of the L2 transaction that relayed a given L1 => L2 message hash.
   *
   * @param l1ToL2MsgHash Hash of the L1 => L2 message to find the receipt for.
   * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
   * @returns Receipt of the L2 transaction that relayed the message.
   */
  public async getL2TransactionReceipt(
    l1ToL2MsgHash: string,
    pollForPending?: boolean
  ): Promise<TransactionReceipt> {
    return this.getTransactionReceipt(this.l2, l1ToL2MsgHash, pollForPending)
  }

  /**
   * Generic function for looking for messages emitted by a transaction.
   *
   * @param layer Parameters for the network layer to look for a messages on.
   * @param txHash Transaction to look for message hashes in.
   * @returns List of message hashes emitted by the transaction.
   */
  public async getMessageHashesFromTx(
    layer: Layer,
    txHash: string
  ): Promise<string[]> {
    const receipt = await layer.provider.getTransactionReceipt(txHash)
    if (!receipt) {
      return []
    }

    // We create a reference to the messenger contract to simplify the process of parsing events
    // and whatnot. In this case we use the ICrossDomainMessenger interface because it contains
    // the event interfaces that both messengers use.
    const messenger = new ethers.Contract(
      layer.messengerAddress,
      getContractInterface('ICrossDomainMessenger'),
      layer.provider
    )

    return receipt.logs
      .filter((log) => {
        // Only look at logs emitted by the messenger address
        return log.address === messenger.address
      })
      .filter((log) => {
        // Only look at SentMessage logs specifically
        const parsed = messenger.interface.parseLog(log)
        return parsed.name === 'SentMessage'
      })
      .map((log) => {
        // Convert each SentMessage log into a message hash
        const parsed = messenger.interface.parseLog(log)
        return computeMessageHash({
          target: parsed.args.target,
          sender: parsed.args.sender,
          message: parsed.args.message,
          messageNonce: parsed.args.messageNonce,
        })
      })
  }

  /**
   * Generic function for looking for the receipt of a transaction that relayed a given message.
   *
   * @param layer Parameters for the network layer to look for the transaction on.
   * @param msgHash Hash of the message to find the receipt for.
   * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
   * @returns Receipt of the transaction that relayed the message.
   */
  public async getTransactionReceipt(
    layer: Layer,
    msgHash: string,
    pollForPending?: boolean
  ): Promise<TransactionReceipt> {
    if (typeof pollForPending !== 'boolean') {
      pollForPending = this.pollForPending
    }

    // We create a reference to the messenger contract to simplify the process of parsing events
    // and whatnot. In this case we use the ICrossDomainMessenger interface because it contains
    // the event interfaces that both messengers use.
    const messenger = new ethers.Contract(
      layer.messengerAddress,
      getContractInterface('ICrossDomainMessenger'),
      layer.provider
    )

    let matches: ethers.Event[] = []
    while (matches.length === 0) {
      matches = [
        ...(await messenger.queryFilter(
          messenger.filters.RelayedMessage(msgHash)
        )),
        ...(await messenger.queryFilter(
          messenger.filters.FailedRelayedMessage(msgHash)
        )),
      ]

      if (!pollForPending) {
        break
      }

      await sleep(this.pollInterval)
    }

    if (matches.length === 0) {
      return undefined
    } else if (matches.length > 1) {
      throw new Error(`Found multiple events with the same message hash.`)
    } else {
      return matches[0].getTransactionReceipt()
    }
  }
}
