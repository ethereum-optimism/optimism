/* eslint-disable @typescript-eslint/no-unused-vars */
import {
  Provider,
  BlockTag,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { ethers, BigNumber, Event } from 'ethers'
import { sleep } from '@eth-optimism/core-utils'

import {
  ICrossChainProvider,
  OEContracts,
  OEContractsLike,
  MessageLike,
  MessageRequestLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  ProviderLike,
  CrossChainMessage,
  CrossChainMessageRequest,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  MessageReceipt,
  MessageReceiptStatus,
  CustomBridges,
  CustomBridgesLike,
  StateRoot,
  StateRootBatch,
} from './interfaces'
import {
  toProvider,
  toBigNumber,
  toTransactionHash,
  DeepPartial,
  getAllOEContracts,
  getCustomBridges,
  hashCrossChainMessage,
} from './utils'

export class CrossChainProvider implements ICrossChainProvider {
  public l1Provider: Provider
  public l2Provider: Provider
  public l1ChainId: number
  public contracts: OEContracts
  public bridges: CustomBridges

  /**
   * Creates a new CrossChainProvider instance.
   *
   * @param opts Options for the provider.
   * @param opts.l1Provider Provider for the L1 chain, or a JSON-RPC url.
   * @param opts.l2Provider Provider for the L2 chain, or a JSON-RPC url.
   * @param opts.l1ChainId Chain ID for the L1 chain.
   * @param opts.contracts Optional contract address overrides.
   * @param opts.bridges Optional bridge address list.
   */
  constructor(opts: {
    l1Provider: ProviderLike
    l2Provider: ProviderLike
    l1ChainId: NumberLike
    contracts?: DeepPartial<OEContractsLike>
    bridges?: Partial<CustomBridgesLike>
  }) {
    this.l1Provider = toProvider(opts.l1Provider)
    this.l2Provider = toProvider(opts.l2Provider)
    this.l1ChainId = toBigNumber(opts.l1ChainId).toNumber()
    this.contracts = getAllOEContracts(this.l1ChainId, {
      l1SignerOrProvider: this.l1Provider,
      l2SignerOrProvider: this.l2Provider,
      overrides: opts.contracts,
    })
    this.bridges = getCustomBridges(this.l1ChainId, {
      l1SignerOrProvider: this.l1Provider,
      l2SignerOrProvider: this.l2Provider,
      overrides: opts.bridges,
    })
  }

  public async getMessagesByTransaction(
    transaction: TransactionLike,
    opts: {
      direction?: MessageDirection
    } = {}
  ): Promise<CrossChainMessage[]> {
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
    throw new Error('Not implemented')
  }

  public async getTokenBridgeMessagesByAddress(
    address: AddressLike,
    opts: {
      direction?: MessageDirection
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    const parseTokenEvent = (
      event: Event,
      dir: MessageDirection
    ): TokenBridgeMessage => {
      return {
        direction: dir,
        from: event.args._from,
        to: event.args._to,
        l1Token: event.args._l1Token || ethers.constants.AddressZero,
        l2Token: event.args._l2Token || this.contracts.l2.OVM_ETH.address,
        amount: event.args._amount,
        data: event.args._data,
        logIndex: event.logIndex,
        blockNumber: event.blockNumber,
        transactionHash: event.transactionHash,
      }
    }

    // Make sure you provide a direction if you specify a block range. Block ranges don't make
    // sense to use on both chains at the same time.
    if (opts.fromBlock !== undefined || opts.toBlock !== undefined) {
      if (opts.direction === undefined) {
        throw new Error('direction must be specified when using a block range')
      }
    }

    // Keep track of all of the messages triggered by the address in question.
    // We'll add messages to this list as we find them, based on the direction that the user has
    // requested we find messages in. If the user hasn't requested a direction, we find messages in
    // both directions.
    const messages: TokenBridgeMessage[] = []

    // First find all messages in the L1 to L2 direction.
    if (
      opts.direction === undefined ||
      opts.direction === MessageDirection.L1_TO_L2
    ) {
      // Find all ETH deposit events and push them into the messages array.
      const ethDepositEvents =
        await this.contracts.l1.L1StandardBridge.queryFilter(
          this.contracts.l1.L1StandardBridge.filters.ETHDepositInitiated(
            address
          ),
          opts.fromBlock,
          opts.toBlock
        )
      for (const event of ethDepositEvents) {
        messages.push(parseTokenEvent(event, MessageDirection.L1_TO_L2))
      }

      // Send an event query for every L1 bridge, this will return an array of arrays.
      const erc20DepositEventSets = await Promise.all(
        [
          this.contracts.l1.L1StandardBridge,
          ...Object.values(this.bridges.l1),
        ].map(async (bridge) => {
          return bridge.queryFilter(
            bridge.filters.ERC20DepositInitiated(undefined, undefined, address),
            opts.fromBlock,
            opts.toBlock
          )
        })
      )

      for (const erc20DepositEvents of erc20DepositEventSets) {
        for (const event of erc20DepositEvents) {
          messages.push(parseTokenEvent(event, MessageDirection.L1_TO_L2))
        }
      }
    }

    // Next find all messages in the L2 to L1 direction.
    if (
      opts.direction === undefined ||
      opts.direction === MessageDirection.L2_TO_L1
    ) {
      // ETH withdrawals and ERC20 withdrawals are the same event on L2.
      // Send an event query for every L2 bridge, this will return an array of arrays.
      const withdrawalEventSets = await Promise.all(
        [
          this.contracts.l2.L2StandardBridge,
          ...Object.values(this.bridges.l2),
        ].map(async (bridge) => {
          return bridge.queryFilter(
            bridge.filters.WithdrawalInitiated(undefined, undefined, address),
            opts.fromBlock,
            opts.toBlock
          )
        })
      )

      for (const withdrawalEvents of withdrawalEventSets) {
        for (const event of withdrawalEvents) {
          messages.push(parseTokenEvent(event, MessageDirection.L2_TO_L1))
        }
      }
    }

    return messages
  }

  public async getDepositsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return this.getTokenBridgeMessagesByAddress(address, {
      ...opts,
      direction: MessageDirection.L1_TO_L2,
    })
  }

  public async getWithdrawalsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return this.getTokenBridgeMessagesByAddress(address, {
      ...opts,
      direction: MessageDirection.L2_TO_L1,
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
            stateRoot.blockNumber
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
    let totalTimeMs = 0
    while (totalTimeMs < (opts.timeoutMs || Infinity)) {
      const tick = Date.now()
      const receipt = await this.getMessageReceipt(message)
      if (receipt !== null) {
        return receipt
      } else {
        await sleep(opts.pollIntervalMs || 4000)
        totalTimeMs += Date.now() - tick
      }
    }

    throw new Error(`timed out waiting for message receipt`)
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
    throw new Error('Not implemented')
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
      blockNumber: stateRootBatch.blockNumber,
      header: stateRootBatch.header,
      stateRoot: stateRootBatch.stateRoots[indexInBatch],
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
}
