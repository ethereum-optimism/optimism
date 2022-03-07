/* eslint-disable @typescript-eslint/no-unused-vars */
import {
  Provider,
  BlockTag,
  TransactionReceipt,
  TransactionResponse,
  TransactionRequest,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { ethers, BigNumber, Overrides } from 'ethers'
import { sleep, remove0x } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'

import {
  ICrossChainMessenger,
  OEContracts,
  OEContractsLike,
  MessageLike,
  MessageRequestLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  SignerOrProviderLike,
  CrossChainMessage,
  CrossChainMessageRequest,
  CrossChainMessageProof,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  MessageReceipt,
  MessageReceiptStatus,
  BridgeAdapterData,
  BridgeAdapters,
  StateRoot,
  StateRootBatch,
  IBridgeAdapter,
} from './interfaces'
import {
  toSignerOrProvider,
  toNumber,
  toTransactionHash,
  DeepPartial,
  getAllOEContracts,
  getBridgeAdapters,
  hashCrossChainMessage,
  makeMerkleTreeProof,
  makeStateTrieProof,
  encodeCrossChainMessage,
  DEPOSIT_CONFIRMATION_BLOCKS,
  CHAIN_BLOCK_TIMES,
} from './utils'

export class CrossChainMessenger implements ICrossChainMessenger {
  public l1SignerOrProvider: Signer | Provider
  public l2SignerOrProvider: Signer | Provider
  public l1ChainId: number
  public contracts: OEContracts
  public bridges: BridgeAdapters
  public depositConfirmationBlocks: number
  public l1BlockTimeSeconds: number

  /**
   * Creates a new CrossChainProvider instance.
   *
   * @param opts Options for the provider.
   * @param opts.l1SignerOrProvider Signer or Provider for the L1 chain, or a JSON-RPC url.
   * @param opts.l2SignerOrProvider Signer or Provider for the L2 chain, or a JSON-RPC url.
   * @param opts.l1ChainId Chain ID for the L1 chain.
   * @param opts.depositConfirmationBlocks Optional number of blocks before a deposit is confirmed.
   * @param opts.l1BlockTimeSeconds Optional estimated block time in seconds for the L1 chain.
   * @param opts.contracts Optional contract address overrides.
   * @param opts.bridges Optional bridge address list.
   */
  constructor(opts: {
    l1SignerOrProvider: SignerOrProviderLike
    l2SignerOrProvider: SignerOrProviderLike
    l1ChainId: NumberLike
    depositConfirmationBlocks?: NumberLike
    l1BlockTimeSeconds?: NumberLike
    contracts?: DeepPartial<OEContractsLike>
    bridges?: BridgeAdapterData
  }) {
    this.l1SignerOrProvider = toSignerOrProvider(opts.l1SignerOrProvider)
    this.l2SignerOrProvider = toSignerOrProvider(opts.l2SignerOrProvider)
    this.l1ChainId = toNumber(opts.l1ChainId)

    this.depositConfirmationBlocks =
      opts?.depositConfirmationBlocks !== undefined
        ? toNumber(opts.depositConfirmationBlocks)
        : DEPOSIT_CONFIRMATION_BLOCKS[this.l1ChainId] || 0

    this.l1BlockTimeSeconds =
      opts?.l1BlockTimeSeconds !== undefined
        ? toNumber(opts.l1BlockTimeSeconds)
        : CHAIN_BLOCK_TIMES[this.l1ChainId] || 1

    this.contracts = getAllOEContracts(this.l1ChainId, {
      l1SignerOrProvider: this.l1SignerOrProvider,
      l2SignerOrProvider: this.l2SignerOrProvider,
      overrides: opts.contracts,
    })

    this.bridges = getBridgeAdapters(this.l1ChainId, this, {
      overrides: opts.bridges,
    })
  }

  get l1Provider(): Provider {
    if (Provider.isProvider(this.l1SignerOrProvider)) {
      return this.l1SignerOrProvider
    } else {
      return this.l1SignerOrProvider.provider as any
    }
  }

  get l2Provider(): Provider {
    if (Provider.isProvider(this.l2SignerOrProvider)) {
      return this.l2SignerOrProvider
    } else {
      return this.l2SignerOrProvider.provider as any
    }
  }

  get l1Signer(): Signer {
    if (Provider.isProvider(this.l1SignerOrProvider)) {
      throw new Error(`messenger has no L1 signer`)
    } else {
      return this.l1SignerOrProvider
    }
  }

  get l2Signer(): Signer {
    if (Provider.isProvider(this.l2SignerOrProvider)) {
      throw new Error(`messenger has no L2 signer`)
    } else {
      return this.l2SignerOrProvider
    }
  }

  public async getMessagesByTransaction(
    transaction: TransactionLike,
    opts: {
      direction?: MessageDirection
    } = {}
  ): Promise<CrossChainMessage[]> {
    // Wait for the transaction receipt if the input is waitable.
    // TODO: Maybe worth doing this with more explicit typing but whatever for now.
    if (typeof (transaction as any).wait === 'function') {
      await (transaction as any).wait()
    }

    // Convert the input to a transaction hash.
    const txHash = toTransactionHash(transaction)

    let receipt: TransactionReceipt
    if (opts.direction !== undefined) {
      // Get the receipt for the requested direction.
      if (opts.direction === MessageDirection.L1_TO_L2) {
        receipt = await this.l1Provider.getTransactionReceipt(txHash)
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
      }
    } else {
      // Try both directions, starting with L1 => L2.
      receipt = await this.l1Provider.getTransactionReceipt(txHash)
      if (receipt) {
        opts.direction = MessageDirection.L1_TO_L2
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
        opts.direction = MessageDirection.L2_TO_L1
      }
    }

    if (!receipt) {
      throw new Error(`unable to find transaction receipt for ${txHash}`)
    }

    // By this point opts.direction will always be defined.
    const messenger =
      opts.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l1.L1CrossDomainMessenger
        : this.contracts.l2.L2CrossDomainMessenger

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
        // Convert each SentMessage log into a message object
        const parsed = messenger.interface.parseLog(log)
        return {
          direction: opts.direction,
          target: parsed.args.target,
          sender: parsed.args.sender,
          message: parsed.args.message,
          messageNonce: parsed.args.messageNonce,
          gasLimit: parsed.args.gasLimit,
          logIndex: log.logIndex,
          blockNumber: log.blockNumber,
          transactionHash: log.transactionHash,
        }
      })
  }

  public async getMessagesByAddress(
    address: AddressLike,
    opts?: {
      direction?: MessageDirection
      fromBlock?: NumberLike
      toBlock?: NumberLike
    }
  ): Promise<CrossChainMessage[]> {
    throw new Error(`
      The function getMessagesByAddress is currently not enabled because the sender parameter of
      the SentMessage event is not indexed within the CrossChainMessenger contracts.
      getMessagesByAddress will be enabled by plugging in an Optimism Indexer (coming soon).
      See the following issue on GitHub for additional context:
      https://github.com/ethereum-optimism/optimism/issues/2129
    `)
  }

  public async getBridgeForTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<IBridgeAdapter> {
    const bridges: IBridgeAdapter[] = []
    for (const bridge of Object.values(this.bridges)) {
      if (await bridge.supportsTokenPair(l1Token, l2Token)) {
        bridges.push(bridge)
      }
    }

    if (bridges.length === 0) {
      throw new Error(`no supported bridge for token pair`)
    }

    if (bridges.length > 1) {
      throw new Error(`found more than one bridge for token pair`)
    }

    return bridges[0]
  }

  public async getDepositsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return (
      await Promise.all(
        Object.values(this.bridges).map(async (bridge) => {
          return bridge.getDepositsByAddress(address, opts)
        })
      )
    )
      .reduce((acc, val) => {
        return acc.concat(val)
      }, [])
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  public async getWithdrawalsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return (
      await Promise.all(
        Object.values(this.bridges).map(async (bridge) => {
          return bridge.getWithdrawalsByAddress(address, opts)
        })
      )
    )
      .reduce((acc, val) => {
        return acc.concat(val)
      }, [])
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  public async toCrossChainMessage(
    message: MessageLike
  ): Promise<CrossChainMessage> {
    // TODO: Convert these checks into proper type checks.
    if ((message as CrossChainMessage).message) {
      return message as CrossChainMessage
    } else if (
      (message as TokenBridgeMessage).l1Token &&
      (message as TokenBridgeMessage).l2Token &&
      (message as TokenBridgeMessage).transactionHash
    ) {
      const messages = await this.getMessagesByTransaction(
        (message as TokenBridgeMessage).transactionHash
      )

      // The `messages` object corresponds to a list of SentMessage events that were triggered by
      // the same transaction. We want to find the specific SentMessage event that corresponds to
      // the TokenBridgeMessage (either a ETHDepositInitiated, ERC20DepositInitiated, or
      // WithdrawalInitiated event). We expect the behavior of bridge contracts to be that these
      // TokenBridgeMessage events are triggered and then a SentMessage event is triggered. Our
      // goal here is therefore to find the first SentMessage event that comes after the input
      // event.
      const found = messages
        .sort((a, b) => {
          // Sort all messages in ascending order by log index.
          return a.logIndex - b.logIndex
        })
        .find((m) => {
          return m.logIndex > (message as TokenBridgeMessage).logIndex
        })

      if (!found) {
        throw new Error(`could not find SentMessage event for message`)
      }

      return found
    } else {
      // TODO: Explicit TransactionLike check and throw if not TransactionLike
      const messages = await this.getMessagesByTransaction(
        message as TransactionLike
      )

      // We only want to treat TransactionLike objects as MessageLike if they only emit a single
      // message (very common). It's unintuitive to treat a TransactionLike as a MessageLike if
      // they emit more than one message (which message do you pick?), so we throw an error.
      if (messages.length !== 1) {
        throw new Error(`expected 1 message, got ${messages.length}`)
      }

      return messages[0]
    }
  }

  public async getMessageStatus(message: MessageLike): Promise<MessageStatus> {
    const resolved = await this.toCrossChainMessage(message)
    const receipt = await this.getMessageReceipt(resolved)

    if (resolved.direction === MessageDirection.L1_TO_L2) {
      if (receipt === null) {
        return MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE
      } else {
        if (receipt.receiptStatus === MessageReceiptStatus.RELAYED_SUCCEEDED) {
          return MessageStatus.RELAYED
        } else {
          return MessageStatus.FAILED_L1_TO_L2_MESSAGE
        }
      }
    } else {
      if (receipt === null) {
        const stateRoot = await this.getMessageStateRoot(resolved)
        if (stateRoot === null) {
          return MessageStatus.STATE_ROOT_NOT_PUBLISHED
        } else {
          const challengePeriod = await this.getChallengePeriodSeconds()
          const targetBlock = await this.l1Provider.getBlock(
            stateRoot.batch.blockNumber
          )
          const latestBlock = await this.l1Provider.getBlock('latest')
          if (targetBlock.timestamp + challengePeriod > latestBlock.timestamp) {
            return MessageStatus.IN_CHALLENGE_PERIOD
          } else {
            return MessageStatus.READY_FOR_RELAY
          }
        }
      } else {
        if (receipt.receiptStatus === MessageReceiptStatus.RELAYED_SUCCEEDED) {
          return MessageStatus.RELAYED
        } else {
          return MessageStatus.READY_FOR_RELAY
        }
      }
    }
  }

  public async getMessageReceipt(
    message: MessageLike
  ): Promise<MessageReceipt> {
    const resolved = await this.toCrossChainMessage(message)
    const messageHash = hashCrossChainMessage(resolved)

    // Here we want the messenger that will receive the message, not the one that sent it.
    const messenger =
      resolved.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l2.L2CrossDomainMessenger
        : this.contracts.l1.L1CrossDomainMessenger

    const relayedMessageEvents = await messenger.queryFilter(
      messenger.filters.RelayedMessage(messageHash)
    )

    // Great, we found the message. Convert it into a transaction receipt.
    if (relayedMessageEvents.length === 1) {
      return {
        receiptStatus: MessageReceiptStatus.RELAYED_SUCCEEDED,
        transactionReceipt:
          await relayedMessageEvents[0].getTransactionReceipt(),
      }
    } else if (relayedMessageEvents.length > 1) {
      // Should never happen!
      throw new Error(`multiple successful relays for message`)
    }

    // We didn't find a transaction that relayed the message. We now attempt to find
    // FailedRelayedMessage events instead.
    const failedRelayedMessageEvents = await messenger.queryFilter(
      messenger.filters.FailedRelayedMessage(messageHash)
    )

    // A transaction can fail to be relayed multiple times. We'll always return the last
    // transaction that attempted to relay the message.
    // TODO: Is this the best way to handle this?
    if (failedRelayedMessageEvents.length > 0) {
      return {
        receiptStatus: MessageReceiptStatus.RELAYED_FAILED,
        transactionReceipt: await failedRelayedMessageEvents[
          failedRelayedMessageEvents.length - 1
        ].getTransactionReceipt(),
      }
    }

    // TODO: If the user doesn't provide enough gas then there's a chance that FailedRelayedMessage
    // will never be triggered. We should probably fix this at the contract level by requiring a
    // minimum amount of input gas and designing the contracts such that the gas will always be
    // enough to trigger the event. However, for now we need a temporary way to find L1 => L2
    // transactions that fail but don't alert us because they didn't provide enough gas.
    // TODO: Talk with the systems and protocol team about coordinating a hard fork that fixes this
    // on both L1 and L2.

    // Just return null if we didn't find a receipt. Slightly nicer than throwing an error.
    return null
  }

  public async waitForMessageReceipt(
    message: MessageLike,
    opts: {
      confirmations?: number
      pollIntervalMs?: number
      timeoutMs?: number
    } = {}
  ): Promise<MessageReceipt> {
    // Resolving once up-front is slightly more efficient.
    const resolved = await this.toCrossChainMessage(message)

    let totalTimeMs = 0
    while (totalTimeMs < (opts.timeoutMs || Infinity)) {
      const tick = Date.now()
      const receipt = await this.getMessageReceipt(resolved)
      if (receipt !== null) {
        return receipt
      } else {
        await sleep(opts.pollIntervalMs || 4000)
        totalTimeMs += Date.now() - tick
      }
    }

    throw new Error(`timed out waiting for message receipt`)
  }

  public async waitForMessageStatus(
    message: MessageLike,
    status: MessageStatus,
    opts: {
      pollIntervalMs?: number
      timeoutMs?: number
    } = {}
  ): Promise<void> {
    // Resolving once up-front is slightly more efficient.
    const resolved = await this.toCrossChainMessage(message)

    let totalTimeMs = 0
    while (totalTimeMs < (opts.timeoutMs || Infinity)) {
      const tick = Date.now()
      const currentStatus = await this.getMessageStatus(resolved)

      // Handle special cases for L1 to L2 messages.
      if (resolved.direction === MessageDirection.L1_TO_L2) {
        // If we're at the expected status, we're done.
        if (currentStatus === status) {
          return
        }

        if (
          status === MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE &&
          currentStatus > status
        ) {
          // Anything other than UNCONFIRMED_L1_TO_L2_MESSAGE implies that the message was at one
          // point "unconfirmed", so we can stop waiting.
          return
        }

        if (
          status === MessageStatus.FAILED_L1_TO_L2_MESSAGE &&
          currentStatus === MessageStatus.RELAYED
        ) {
          throw new Error(
            `incompatible message status, expected FAILED_L1_TO_L2_MESSAGE got RELAYED`
          )
        }

        if (
          status === MessageStatus.RELAYED &&
          currentStatus === MessageStatus.FAILED_L1_TO_L2_MESSAGE
        ) {
          throw new Error(
            `incompatible message status, expected RELAYED got FAILED_L1_TO_L2_MESSAGE`
          )
        }
      }

      // Handle special cases for L2 to L1 messages.
      if (resolved.direction === MessageDirection.L2_TO_L1) {
        if (currentStatus >= status) {
          // For L2 to L1 messages, anything after the expected status implies the previous status,
          // so we can safely return if the current status enum is larger than the expected one.
          return
        }
      }

      await sleep(opts.pollIntervalMs || 4000)
      totalTimeMs += Date.now() - tick
    }

    throw new Error(`timed out waiting for message status change`)
  }

  public async estimateL2MessageGasLimit(
    message: MessageRequestLike,
    opts?: {
      bufferPercent?: number
      from?: string
    }
  ): Promise<BigNumber> {
    let resolved: CrossChainMessage | CrossChainMessageRequest
    let from: string
    if ((message as CrossChainMessage).messageNonce === undefined) {
      resolved = message as CrossChainMessageRequest
      from = opts?.from
    } else {
      resolved = await this.toCrossChainMessage(message as MessageLike)
      from = opts?.from || (resolved as CrossChainMessage).sender
    }

    // L2 message gas estimation is only used for L1 => L2 messages.
    if (resolved.direction === MessageDirection.L2_TO_L1) {
      throw new Error(`cannot estimate gas limit for L2 => L1 message`)
    }

    const estimate = await this.l2Provider.estimateGas({
      from,
      to: resolved.target,
      data: resolved.message,
    })

    // Return the estimate plus a buffer of 20% just in case.
    const bufferPercent = opts?.bufferPercent || 20
    return estimate.mul(100 + bufferPercent).div(100)
  }

  public async estimateMessageWaitTimeSeconds(
    message: MessageLike
  ): Promise<number> {
    const resolved = await this.toCrossChainMessage(message)
    const status = await this.getMessageStatus(resolved)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      if (
        status === MessageStatus.RELAYED ||
        status === MessageStatus.FAILED_L1_TO_L2_MESSAGE
      ) {
        // Transactions that are relayed or failed are considered completed, so the wait time is 0.
        return 0
      } else {
        // Otherwise we need to estimate the number of blocks left until the transaction will be
        // considered confirmed by the Layer 2 system. Then we multiply this by the estimated
        // average L1 block time.
        const receipt = await this.l1Provider.getTransactionReceipt(
          resolved.transactionHash
        )
        const blocksLeft = Math.max(
          this.depositConfirmationBlocks - receipt.confirmations,
          0
        )
        return blocksLeft * this.l1BlockTimeSeconds
      }
    } else {
      if (
        status === MessageStatus.RELAYED ||
        status === MessageStatus.READY_FOR_RELAY
      ) {
        // Transactions that are relayed or ready for relay are considered complete.
        return 0
      } else if (status === MessageStatus.STATE_ROOT_NOT_PUBLISHED) {
        // If the state root hasn't been published yet, just assume it'll be published relatively
        // quickly and return the challenge period for now. In the future we could use more
        // advanced techniques to figure out average time between transaction execution and
        // state root publication.
        return this.getChallengePeriodSeconds()
      } else if (status === MessageStatus.IN_CHALLENGE_PERIOD) {
        // If the message is still within the challenge period, then we need to estimate exactly
        // the amount of time left until the challenge period expires. The challenge period starts
        // when the state root is published.
        const stateRoot = await this.getMessageStateRoot(resolved)
        const challengePeriod = await this.getChallengePeriodSeconds()
        const targetBlock = await this.l1Provider.getBlock(
          stateRoot.batch.blockNumber
        )
        const latestBlock = await this.l1Provider.getBlock('latest')
        return Math.max(
          challengePeriod - (latestBlock.timestamp - targetBlock.timestamp),
          0
        )
      } else {
        // Should not happen
        throw new Error(`unexpected message status`)
      }
    }
  }

  public async getChallengePeriodSeconds(): Promise<number> {
    const challengePeriod =
      await this.contracts.l1.StateCommitmentChain.FRAUD_PROOF_WINDOW()
    return challengePeriod.toNumber()
  }

  public async getMessageStateRoot(
    message: MessageLike
  ): Promise<StateRoot | null> {
    const resolved = await this.toCrossChainMessage(message)

    // State roots are only a thing for L2 to L1 messages.
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`cannot get a state root for an L1 to L2 message`)
    }

    // We need the block number of the transaction that triggered the message so we can look up the
    // state root batch that corresponds to that block number.
    const messageTxReceipt = await this.l2Provider.getTransactionReceipt(
      resolved.transactionHash
    )

    // Every block has exactly one transaction in it. Since there's a genesis block, the
    // transaction index will always be one less than the block number.
    const messageTxIndex = messageTxReceipt.blockNumber - 1

    // Pull down the state root batch, we'll try to pick out the specific state root that
    // corresponds to our message.
    const stateRootBatch = await this.getStateRootBatchByTransactionIndex(
      messageTxIndex
    )

    // No state root batch, no state root.
    if (stateRootBatch === null) {
      return null
    }

    // We have a state root batch, now we need to find the specific state root for our transaction.
    // First we need to figure out the index of the state root within the batch we found. This is
    // going to be the original transaction index offset by the total number of previous state
    // roots.
    const indexInBatch =
      messageTxIndex - stateRootBatch.header.prevTotalElements.toNumber()

    // Just a sanity check.
    if (stateRootBatch.stateRoots.length <= indexInBatch) {
      // Should never happen!
      throw new Error(`state root does not exist in batch`)
    }

    return {
      stateRoot: stateRootBatch.stateRoots[indexInBatch],
      stateRootIndexInBatch: indexInBatch,
      batch: stateRootBatch,
    }
  }

  public async getStateBatchAppendedEventByBatchIndex(
    batchIndex: number
  ): Promise<ethers.Event | null> {
    const events = await this.contracts.l1.StateCommitmentChain.queryFilter(
      this.contracts.l1.StateCommitmentChain.filters.StateBatchAppended(
        batchIndex
      )
    )

    if (events.length === 0) {
      return null
    } else if (events.length > 1) {
      // Should never happen!
      throw new Error(`found more than one StateBatchAppended event`)
    } else {
      return events[0]
    }
  }

  public async getStateBatchAppendedEventByTransactionIndex(
    transactionIndex: number
  ): Promise<ethers.Event | null> {
    const isEventHi = (event: ethers.Event, index: number) => {
      const prevTotalElements = event.args._prevTotalElements.toNumber()
      return index < prevTotalElements
    }

    const isEventLo = (event: ethers.Event, index: number) => {
      const prevTotalElements = event.args._prevTotalElements.toNumber()
      const batchSize = event.args._batchSize.toNumber()
      return index >= prevTotalElements + batchSize
    }

    const totalBatches: ethers.BigNumber =
      await this.contracts.l1.StateCommitmentChain.getTotalBatches()
    if (totalBatches.eq(0)) {
      return null
    }

    let lowerBound = 0
    let upperBound = totalBatches.toNumber() - 1
    let batchEvent: ethers.Event | null =
      await this.getStateBatchAppendedEventByBatchIndex(upperBound)

    // Only happens when no batches have been submitted yet.
    if (batchEvent === null) {
      return null
    }

    if (isEventLo(batchEvent, transactionIndex)) {
      // Upper bound is too low, means this transaction doesn't have a corresponding state batch yet.
      return null
    } else if (!isEventHi(batchEvent, transactionIndex)) {
      // Upper bound is not too low and also not too high. This means the upper bound event is the
      // one we're looking for! Return it.
      return batchEvent
    }

    // Binary search to find the right event. The above checks will guarantee that the event does
    // exist and that we'll find it during this search.
    while (lowerBound < upperBound) {
      const middleOfBounds = Math.floor((lowerBound + upperBound) / 2)
      batchEvent = await this.getStateBatchAppendedEventByBatchIndex(
        middleOfBounds
      )

      if (isEventHi(batchEvent, transactionIndex)) {
        upperBound = middleOfBounds
      } else if (isEventLo(batchEvent, transactionIndex)) {
        lowerBound = middleOfBounds
      } else {
        break
      }
    }

    return batchEvent
  }

  public async getStateRootBatchByTransactionIndex(
    transactionIndex: number
  ): Promise<StateRootBatch | null> {
    const stateBatchAppendedEvent =
      await this.getStateBatchAppendedEventByTransactionIndex(transactionIndex)
    if (stateBatchAppendedEvent === null) {
      return null
    }

    const stateBatchTransaction = await stateBatchAppendedEvent.getTransaction()
    const [stateRoots] =
      this.contracts.l1.StateCommitmentChain.interface.decodeFunctionData(
        'appendStateBatch',
        stateBatchTransaction.data
      )

    return {
      blockNumber: stateBatchAppendedEvent.blockNumber,
      stateRoots,
      header: {
        batchIndex: stateBatchAppendedEvent.args._batchIndex,
        batchRoot: stateBatchAppendedEvent.args._batchRoot,
        batchSize: stateBatchAppendedEvent.args._batchSize,
        prevTotalElements: stateBatchAppendedEvent.args._prevTotalElements,
        extraData: stateBatchAppendedEvent.args._extraData,
      },
    }
  }

  public async getMessageProof(
    message: MessageLike
  ): Promise<CrossChainMessageProof> {
    const resolved = await this.toCrossChainMessage(message)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`can only generate proofs for L2 to L1 messages`)
    }

    const stateRoot = await this.getMessageStateRoot(resolved)
    if (stateRoot === null) {
      throw new Error(`state root for message not yet published`)
    }

    // We need to calculate the specific storage slot that demonstrates that this message was
    // actually included in the L2 chain. The following calculation is based on the fact that
    // messages are stored in the following mapping on L2:
    // https://github.com/ethereum-optimism/optimism/blob/c84d3450225306abbb39b4e7d6d82424341df2be/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol#L23
    // You can read more about how Solidity storage slots are computed for mappings here:
    // https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#mappings-and-dynamic-arrays
    const messageSlot = ethers.utils.keccak256(
      ethers.utils.keccak256(
        encodeCrossChainMessage(resolved) +
          remove0x(this.contracts.l2.L2CrossDomainMessenger.address)
      ) + '00'.repeat(32)
    )

    const stateTrieProof = await makeStateTrieProof(
      this.l2Provider as any,
      resolved.blockNumber,
      this.contracts.l2.OVM_L2ToL1MessagePasser.address,
      messageSlot
    )

    return {
      stateRoot: stateRoot.stateRoot,
      stateRootBatchHeader: stateRoot.batch.header,
      stateRootProof: {
        index: stateRoot.stateRootIndexInBatch,
        siblings: makeMerkleTreeProof(
          stateRoot.batch.stateRoots,
          stateRoot.stateRootIndexInBatch
        ),
      },
      stateTrieWitness: stateTrieProof.accountProof,
      storageTrieWitness: stateTrieProof.storageProof,
    }
  }

  public async sendMessage(
    message: CrossChainMessageRequest,
    opts?: {
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    const tx = await this.populateTransaction.sendMessage(message, opts)
    if (message.direction === MessageDirection.L1_TO_L2) {
      return (opts?.signer || this.l1Signer).sendTransaction(tx)
    } else {
      return (opts?.signer || this.l2Signer).sendTransaction(tx)
    }
  }

  public async resendMessage(
    message: MessageLike,
    messageGasLimit: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.resendMessage(
        message,
        messageGasLimit,
        opts
      )
    )
  }

  public async finalizeMessage(
    message: MessageLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.finalizeMessage(message, opts)
    )
  }

  public async depositETH(
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.depositETH(amount, opts)
    )
  }

  public async withdrawETH(
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l2Signer).sendTransaction(
      await this.populateTransaction.withdrawETH(amount, opts)
    )
  }

  public async approval(
    l1Token: AddressLike,
    l2Token: AddressLike,
    opts?: {
      signer?: Signer
    }
  ): Promise<BigNumber> {
    const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
    return bridge.approval(l1Token, l2Token, opts?.signer || this.l1Signer)
  }

  public async approveERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.approveERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  public async depositERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.depositERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  public async withdrawERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l2Signer).sendTransaction(
      await this.populateTransaction.withdrawERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  populateTransaction = {
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.contracts.l1.L1CrossDomainMessenger.populateTransaction.sendMessage(
          message.target,
          message.message,
          opts?.l2GasLimit || (await this.estimateL2MessageGasLimit(message)),
          opts?.overrides || {}
        )
      } else {
        return this.contracts.l2.L2CrossDomainMessenger.populateTransaction.sendMessage(
          message.target,
          message.message,
          0, // Gas limit goes unused when sending from L2 to L1
          opts?.overrides || {}
        )
      }
    },

    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const resolved = await this.toCrossChainMessage(message)
      if (resolved.direction === MessageDirection.L2_TO_L1) {
        throw new Error(`cannot resend L2 to L1 message`)
      }

      return this.contracts.l1.L1CrossDomainMessenger.populateTransaction.replayMessage(
        resolved.target,
        resolved.sender,
        resolved.message,
        resolved.messageNonce,
        resolved.gasLimit,
        messageGasLimit,
        opts?.overrides || {}
      )
    },

    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const resolved = await this.toCrossChainMessage(message)
      if (resolved.direction === MessageDirection.L1_TO_L2) {
        throw new Error(`cannot finalize L1 to L2 message`)
      }

      const proof = await this.getMessageProof(resolved)
      return this.contracts.l1.L1CrossDomainMessenger.populateTransaction.relayMessage(
        resolved.target,
        resolved.sender,
        resolved.message,
        resolved.messageNonce,
        proof,
        opts?.overrides || {}
      )
    },

    depositETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      return this.bridges.ETH.populateTransaction.deposit(
        ethers.constants.AddressZero,
        predeploys.OVM_ETH,
        amount,
        opts
      )
    },

    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      return this.bridges.ETH.populateTransaction.withdraw(
        ethers.constants.AddressZero,
        predeploys.OVM_ETH,
        amount,
        opts
      )
    },

    approveERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      return bridge.populateTransaction.approve(l1Token, l2Token, amount, opts)
    },

    depositERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      return bridge.populateTransaction.deposit(l1Token, l2Token, amount, opts)
    },

    withdrawERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      return bridge.populateTransaction.withdraw(l1Token, l2Token, amount, opts)
    },
  }

  estimateGas = {
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.sendMessage(message, opts)
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.l1Provider.estimateGas(tx)
      } else {
        return this.l2Provider.estimateGas(tx)
      }
    },

    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.resendMessage(
          message,
          messageGasLimit,
          opts
        )
      )
    },

    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.finalizeMessage(message, opts)
      )
    },

    depositETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.depositETH(amount, opts)
      )
    },

    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l2Provider.estimateGas(
        await this.populateTransaction.withdrawETH(amount, opts)
      )
    },

    approveERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.approveERC20(
          l1Token,
          l2Token,
          amount,
          opts
        )
      )
    },

    depositERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.depositERC20(
          l1Token,
          l2Token,
          amount,
          opts
        )
      )
    },

    withdrawERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      return this.l2Provider.estimateGas(
        await this.populateTransaction.withdrawERC20(
          l1Token,
          l2Token,
          amount,
          opts
        )
      )
    },
  }
}
