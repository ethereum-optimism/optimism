/* Imports: External */
import { Wallet } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'
import { Logger, BaseService, Metrics } from '@eth-optimism/common-ts'
import {
  CrossChainMessenger,
  MessageStatus,
  ProviderLike,
} from '@eth-optimism/sdk'

interface MessageRelayerOptions {
  /**
   * Provider for interacting with L2.
   */
  l2RpcProvider: ProviderLike

  /**
   * Wallet used to interact with L1.
   */
  l1Wallet: Wallet

  /**
   * Gas to relay transactions with. If not provided, will use the estimated gas for the relay
   * transaction.
   */
  relayGasLimit?: number

  /**
   * Index of the first L2 transaction to start processing from.
   */
  fromL2TransactionIndex?: number

  /**
   * Waiting interval between loops when the service is at the tip.
   */
  pollingInterval?: number

  /**
   * Size of the block range to query when looking for new SentMessage events.
   */
  getLogsInterval?: number

  /**
   * Logger to transport logs. Defaults to STDOUT.
   */
  logger?: Logger

  /**
   * Metrics object to use. Defaults to no metrics.
   */
  metrics?: Metrics
}

export class MessageRelayerService extends BaseService<MessageRelayerOptions> {
  constructor(options: MessageRelayerOptions) {
    super('Message_Relayer', options, {
      relayGasLimit: {
        default: 4_000_000,
      },
      fromL2TransactionIndex: {
        default: 0,
      },
      pollingInterval: {
        default: 5000,
      },
      getLogsInterval: {
        default: 2000,
      },
    })
  }

  private state: {
    messenger: CrossChainMessenger
    highestCheckedL2Tx: number
  } = {} as any

  protected async _init(): Promise<void> {
    this.logger.info('Initializing message relayer', {
      relayGasLimit: this.options.relayGasLimit,
      fromL2TransactionIndex: this.options.fromL2TransactionIndex,
      pollingInterval: this.options.pollingInterval,
      getLogsInterval: this.options.getLogsInterval,
    })

    const l1Network = await this.options.l1Wallet.provider.getNetwork()
    const l1ChainId = l1Network.chainId
    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: this.options.l1Wallet,
      l2SignerOrProvider: this.options.l2RpcProvider,
      l1ChainId,
    })

    this.state.highestCheckedL2Tx = this.options.fromL2TransactionIndex || 1
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      await sleep(this.options.pollingInterval)

      try {
        // Loop strategy is as follows:
        // 1. Get the current L2 tip
        // 2. While we're not at the tip:
        //    2.1. Get the transaction for the next L2 block to parse.
        //    2.2. Find any messages sent in the L2 block.
        //    2.3. Make sure all messages are ready to be relayed.
        //    3.4. Relay the messages.
        const l2BlockNumber =
          await this.state.messenger.l2Provider.getBlockNumber()

        while (this.state.highestCheckedL2Tx <= l2BlockNumber) {
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
            continue
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
            break
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
      } catch (err) {
        this.logger.error('Caught an unhandled error', {
          message: err.toString(),
          stack: err.stack,
          code: err.code,
        })
      }
    }
  }
}
