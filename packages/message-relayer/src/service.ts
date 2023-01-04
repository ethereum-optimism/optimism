/* Imports: External */
import { Signer } from 'ethers'
import { getChainId, sleep } from '@eth-optimism/core-utils'
import {
  BaseServiceV2,
  validators,
  Gauge,
  Counter,
} from '@eth-optimism/common-ts'
import { CrossChainMessenger, MessageStatus } from '@eth-optimism/sdk'
import { Provider } from '@ethersproject/abstract-provider'

import { version } from '../package.json'

type MessageRelayerOptions = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  l1Wallet: Signer
  fromL2TransactionIndex?: number
}

type MessageRelayerMetrics = {
  highestCheckedL2Tx: Gauge
  highestKnownL2Tx: Gauge
  numRelayedMessages: Counter
}

type MessageRelayerState = {
  wallet: Signer
  messenger: CrossChainMessenger
  highestCheckedL2Tx: number
  highestKnownL2Tx: number
}

export class MessageRelayerService extends BaseServiceV2<
  MessageRelayerOptions,
  MessageRelayerMetrics,
  MessageRelayerState
> {
  constructor(options?: Partial<MessageRelayerOptions>) {
    super({
      version,
      name: 'message-relayer',
      options,
      optionsSpec: {
        l1RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L1.',
        },
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L2.',
        },
        l1Wallet: {
          validator: validators.wallet,
          desc: 'Wallet used to interact with L1.',
        },
        fromL2TransactionIndex: {
          validator: validators.num,
          desc: 'Index of the first L2 transaction to start processing from.',
          default: 0,
          public: true,
        },
      },
      metricsSpec: {
        highestCheckedL2Tx: {
          type: Gauge,
          desc: 'Highest L2 tx that has been scanned for messages',
        },
        highestKnownL2Tx: {
          type: Gauge,
          desc: 'Highest known L2 transaction',
        },
        numRelayedMessages: {
          type: Counter,
          desc: 'Number of messages relayed by the service',
        },
      },
    })
  }

  protected async init(): Promise<void> {
    this.state.wallet = this.options.l1Wallet.connect(
      this.options.l1RpcProvider
    )

    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: this.state.wallet,
      l2SignerOrProvider: this.options.l2RpcProvider,
      l1ChainId: await getChainId(this.state.wallet.provider),
      l2ChainId: await getChainId(this.options.l2RpcProvider),
    })

    this.state.highestCheckedL2Tx = this.options.fromL2TransactionIndex || 1
    this.state.highestKnownL2Tx =
      await this.state.messenger.l2Provider.getBlockNumber()
  }

  protected async main(): Promise<void> {
    // Update metrics
    this.metrics.highestCheckedL2Tx.set(this.state.highestCheckedL2Tx)
    this.metrics.highestKnownL2Tx.set(this.state.highestKnownL2Tx)

    // If we're already at the tip, then update the latest tip and loop again.
    if (this.state.highestCheckedL2Tx > this.state.highestKnownL2Tx) {
      this.state.highestKnownL2Tx =
        await this.state.messenger.l2Provider.getBlockNumber()

      // Sleeping for 1000ms is good enough since this is meant for development and not for live
      // networks where we might want to restrict the number of requests per second.
      await sleep(1000)
      return
    }

    this.logger.info(`checking L2 block ${this.state.highestCheckedL2Tx}`)

    const block =
      await this.state.messenger.l2Provider.getBlockWithTransactions(
        this.state.highestCheckedL2Tx
      )

    // Should never happen.
    if (block.transactions.length !== 1) {
      throw new Error(
        `got an unexpected number of transactions in block: ${block.number}`
      )
    }

    const messages = await this.state.messenger.getMessagesByTransaction(
      block.transactions[0].hash
    )

    // No messages in this transaction so we can move on to the next one.
    if (messages.length === 0) {
      this.state.highestCheckedL2Tx++
      return
    }

    // Make sure that all messages sent within the transaction are finalized. If any messages
    // are not finalized, then we're going to break the loop which will trigger the sleep and
    // wait for a few seconds before we check again to see if this transaction is finalized.
    let isFinalized = true
    for (const message of messages) {
      const status = await this.state.messenger.getMessageStatus(message)
      if (
        status === MessageStatus.IN_CHALLENGE_PERIOD ||
        status === MessageStatus.STATE_ROOT_NOT_PUBLISHED
      ) {
        isFinalized = false
      }
    }

    if (!isFinalized) {
      this.logger.info(
        `tx not yet finalized, waiting: ${this.state.highestCheckedL2Tx}`
      )
      return
    } else {
      this.logger.info(
        `tx is finalized, relaying: ${this.state.highestCheckedL2Tx}`
      )
    }

    // If we got here then all messages in the transaction are finalized. Now we can relay
    // each message to L1.
    for (const message of messages) {
      try {
        const tx = await this.state.messenger.finalizeMessage(message)
        this.logger.info(`relayer sent tx: ${tx.hash}`)
        this.metrics.numRelayedMessages.inc()
      } catch (err) {
        if (err.message.includes('message has already been received')) {
          // It's fine, the message was relayed by someone else
        } else {
          throw err
        }
      }
      await this.state.messenger.waitForMessageReceipt(message)
    }

    // All messages have been relayed so we can move on to the next block.
    this.state.highestCheckedL2Tx++
  }
}

if (require.main === module) {
  const service = new MessageRelayerService()
  service.run()
}
