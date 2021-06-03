/* Imports: External */
import { ethers, providers } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'
import { sleep, NUM_L2_GENESIS_BLOCKS } from '@eth-optimism/core-utils'
import { BaseService } from '@eth-optimism/common-ts'

/* Imports: Internal */
import {
  getCrossDomainMessageHash,
  getMessagesAndProofsForL2Transaction,
  getStateRootBatchByBatchIndex,
} from './relay-tx'

interface MessageRelayerOptions {
  // Providers for interacting with L1 and L2.
  l1RpcProvider: providers.JsonRpcProvider | string
  l2RpcProvider: providers.JsonRpcProvider | string

  // Address of the OVM_StateCommitmentChain.
  stateCommitmentChainAddress: string

  // Address of the L1CrossDomainMessenger.
  l1CrossDomainMessengerAddress: string

  // Address of the L2CrossDomainMessenger.
  l2CrossDomainMessengerAddress: string

  // Private key for the account that will relay transactions.
  relayerPrivateKey: string

  // Interval in milliseconds to wait between loops when waiting for new transactions to scan.
  pollingIntervalMs?: number
}

export class MessageRelayerService extends BaseService<MessageRelayerOptions> {
  constructor(options: MessageRelayerOptions) {
    super('Message_Relayer', options, {
      pollingIntervalMs: { default: 5000 },
    })
  }

  private state: {
    l1RpcProvider: ethers.providers.JsonRpcProvider
    l2RpcProvider: ethers.providers.JsonRpcProvider
    relayerWallet: ethers.Wallet
    stateCommitmentChain: ethers.Contract
    l1CrossDomainMessenger: ethers.Contract
    l2CrossDomainMessenger: ethers.Contract
    nextUnsyncedStateRootBatchIndex: number
  } = {
    nextUnsyncedStateRootBatchIndex: 0,
  } as any

  protected async _init(): Promise<void> {
    this.logger.info('Initializing message relayer', {
      pollingInterval: this.options.pollingIntervalMs,
    })

    // Set up our providers.
    if (typeof this.options.l1RpcProvider === 'string') {
      this.state.l1RpcProvider = new ethers.providers.JsonRpcProvider(
        this.options.l1RpcProvider
      )
    } else {
      this.state.l1RpcProvider = this.options.l1RpcProvider
    }
    if (typeof this.options.l2RpcProvider === 'string') {
      this.state.l2RpcProvider = new ethers.providers.JsonRpcProvider(
        this.options.l2RpcProvider
      )
    } else {
      this.state.l2RpcProvider = this.options.l2RpcProvider
    }

    // Set up our contract references.
    this.state.stateCommitmentChain = new ethers.Contract(
      this.options.stateCommitmentChainAddress,
      getContractInterface('OVM_StateCommitmentChain'),
      this.state.l1RpcProvider
    )
    this.state.l1CrossDomainMessenger = new ethers.Contract(
      this.options.l1CrossDomainMessengerAddress,
      getContractInterface('OVM_L1CrossDomainMessenger'),
      this.state.l1RpcProvider
    )
    this.state.l2CrossDomainMessenger = new ethers.Contract(
      this.options.l2CrossDomainMessengerAddress,
      getContractInterface('OVM_L2CrossDomainMessenger'),
      this.state.l2RpcProvider
    )

    // And finally set up our wallet object.
    this.state.relayerWallet = new ethers.Wallet(
      this.options.relayerPrivateKey,
      this.state.l1RpcProvider
    )
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      try {
        await this._main()
      } catch (err) {
        // Log the error but don't throw.
      }
    }
  }

  private async _main(): Promise<void> {
    const nextUnsyncedStateRootBatch = await getStateRootBatchByBatchIndex(
      this.state.l1RpcProvider,
      this.options.stateCommitmentChainAddress,
      this.state.nextUnsyncedStateRootBatchIndex
    )

    if (nextUnsyncedStateRootBatch === null) {
      await sleep(this.options.pollingIntervalMs)
      return
    }

    const isBatchUnfinalized = await this.state.stateCommitmentChain.insideFraudProofWindow(
      nextUnsyncedStateRootBatch.header
    )

    if (isBatchUnfinalized) {
      await sleep(this.options.pollingIntervalMs)
      return
    }

    const batchPrevTotalElements = nextUnsyncedStateRootBatch.header.prevTotalElements.toNumber()
    const batchSize = nextUnsyncedStateRootBatch.header.batchSize.toNumber()
    const messageEvents = await this.state.l2CrossDomainMessenger.queryFilter(
      this.state.l2CrossDomainMessenger.filters.SentMessage(),
      batchPrevTotalElements + NUM_L2_GENESIS_BLOCKS,
      batchPrevTotalElements + batchSize + NUM_L2_GENESIS_BLOCKS
    )

    this.logger.info('found next finalized transaction batch', {
      batchIndex: this.state.nextUnsyncedStateRootBatchIndex,
      batchPrevTotalElements,
      batchSize,
      numSentMessages: messageEvents.length,
    })

    for (const messageEvent of messageEvents) {
      this.logger.info('generating proof data for message', {
        transactionHash: messageEvent.transactionHash,
        eventIndex: messageEvent.logIndex,
      })

      const messagePairs = await getMessagesAndProofsForL2Transaction(
        this.state.l1RpcProvider,
        this.state.l2RpcProvider,
        this.options.stateCommitmentChainAddress,
        this.options.l2CrossDomainMessengerAddress,
        messageEvent.transactionHash
      )

      for (const { message, proof } of messagePairs) {
        const messageHash = getCrossDomainMessageHash(message)

        this.logger.info('relaying message', {
          transactionHash: messageEvent.transactionHash,
          messageHash,
          message,
        })

        try {
          const result = await this.state.l1CrossDomainMessenger
            .connect(this.state.relayerWallet)
            .relayMessage(
              message.target,
              message.sender,
              message.message,
              message.messageNonce,
              proof
            )

          const receipt = await result.wait()

          this.logger.info('relayed message successfully', {
            messageHash,
            relayTransactionHash: receipt.transactionHash,
          })
        } catch (err) {
          const wasAlreadyRelayed = await this.state.l1CrossDomainMessenger.successfulMessages(
            messageHash
          )

          if (wasAlreadyRelayed) {
            this.logger.info('message was already relayed', {
              messageHash,
            })
          } else {
            this.logger.error('caught an error while relaying a message', {
              message: err.message,
              stack: err.stack,
              code: err.code,
            })
          }
        }
      }
    }

    this.state.nextUnsyncedStateRootBatchIndex += 1
  }
}
