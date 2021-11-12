import { ethers, BigNumber, Contract, Event } from 'ethers'
import {
  TransactionReceipt,
  Provider,
  BlockTag,
} from '@ethersproject/abstract-provider'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'
import {
  ICrossChainProvider,
  AddressLike,
  TransactionLike,
  NumberLike,
  MessageLike,
  MessageStatus,
  TokenBridgeMessage,
  MessageDirection,
  CrossChainMessage,
  ProviderLike,
  OEContracts,
  MessageReceipt,
  MessageReceiptStatus,
  NetworkName,
} from '../base'
import {
  ErrMultipleSuccessfulRelays,
  ErrTimeoutReached,
  ErrSentMessageMultipleTimes,
  ErrSentMessageNotFound,
} from './errors'
import {
  getTransactionHash,
  toProvider,
  hashCrossChainMessage,
  NUM_L2_GENESIS_BLOCKS,
  CHALLENGE_PERIOD_BLOCKS,
  L1_TO_L2_TX_CONFIRMATIONS,
  L1_BLOCK_INTERVAL_SECONDS,
} from './utils'

export class CrossChainProvider implements ICrossChainProvider {
  public l1Provider: Provider
  public l2Provider: Provider
  public contracts: OEContracts
  public network: NetworkName

  constructor(opts: { l1Provider: ProviderLike; l2Provider: ProviderLike }) {
    this.l1Provider = toProvider(opts.l1Provider)
    this.l2Provider = toProvider(opts.l2Provider)

    // Handle contract connections
    this.contracts.l2.L2CrossDomainMessenger = new Contract(
      predeploys.L2CrossDomainMessenger,
      getContractInterface('L2CrossDomainMessenger'),
      this.l2Provider
    )
  }

  public async getMessagesByTransaction(
    transaction: TransactionLike,
    direction?: MessageDirection
  ): Promise<CrossChainMessage[]> {
    const txHash = getTransactionHash(transaction)

    let receipt: TransactionReceipt
    if (direction !== undefined) {
      // Get the receipt for the requested direction.
      if (direction === MessageDirection.L1_TO_L2) {
        receipt = await this.l1Provider.getTransactionReceipt(txHash)
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
      }
    } else {
      // Try both directions, starting with L1 => L2.
      receipt = await this.l1Provider.getTransactionReceipt(txHash)
      if (receipt) {
        direction = MessageDirection.L1_TO_L2
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
        direction = MessageDirection.L2_TO_L1
      }
    }

    if (!receipt) {
      throw new Error(`unable to find transaction receipt for ${txHash}`)
    }

    const messenger =
      direction === MessageDirection.L1_TO_L2
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
          direction,
          target: parsed.args.target,
          sender: parsed.args.sender,
          message: parsed.args.message,
          messageNonce: parsed.args.messageNonce,
        }
      })
  }

  public async getMessagesByAddress(
    address: AddressLike,
    direction?: MessageDirection,
    fromBlock?: NumberLike,
    toBlock?: NumberLike
  ): Promise<CrossChainMessage[]> {
    throw new Error('Method not implemented.')
  }

  public async getTokenBridgeMessagesByAddress(
    address: AddressLike,
    direction?: MessageDirection,
    fromBlock?: BlockTag,
    toBlock?: BlockTag
  ): Promise<TokenBridgeMessage[]> {
    const parseTokenEvent = async (
      event: Event,
      dir: MessageDirection
    ): Promise<TokenBridgeMessage> => {
      const raws = await this.getMessagesByTransaction(
        event.transactionHash,
        dir
      )

      // TODO: Correctly handle multiple messages by filtering out for the message with the
      // correct input data. Or figure out another way to get the correct message. Either way,
      // we need to be able to handle cases where more than one deposit is triggered in the same
      // transaction.
      if (raws.length !== 1) {
        throw new Error('expected number of messages in transaction')
      }

      return {
        direction: dir,
        from: event.args._from,
        to: event.args._to,
        l1Token: event.args._l1Token || ethers.constants.AddressZero,
        l2Token: event.args._l2Token || predeploys.OVM_ETH,
        amount: event.args._amount,
        raw: raws[0],
      }
    }

    // Keep track of all of the messages triggered by the address in question.
    // We'll add messages to this list as we find them, based on the direction that the user has
    // requested we find messages in. If the user hasn't requested a direction, we find messages in
    // both directions.
    const messages: TokenBridgeMessage[] = []

    // First find all messages in the L1 to L2 direction.
    if (direction === undefined || direction === MessageDirection.L1_TO_L2) {
      // Find all ETH deposit events and push them into the messages array.
      const ethDepositEvents =
        await this.contracts.l1.L1StandardBridge.queryFilter(
          this.contracts.l1.L1StandardBridge.filters.ETHDepositInitiated(
            address
          ),
          fromBlock,
          toBlock
        )
      for (const event of ethDepositEvents) {
        messages.push(await parseTokenEvent(event, MessageDirection.L1_TO_L2))
      }

      // Find all token deposit events and push them into the messages array.
      const erc20DepositEvents =
        await this.contracts.l1.L1StandardBridge.queryFilter(
          this.contracts.l1.L1StandardBridge.filters.ERC20DepositInitiated(
            undefined,
            undefined,
            address
          ),
          fromBlock,
          toBlock
        )
      for (const event of erc20DepositEvents) {
        messages.push(await parseTokenEvent(event, MessageDirection.L1_TO_L2))
      }
    }

    // Next find all messages in the L2 to L1 direction.
    if (direction === undefined || direction === MessageDirection.L2_TO_L1) {
      // ETH withdrawals and ERC20 withdrawals are the same event on L2.
      // Find all withdrawal events and push them into the messages array.
      const withdrawEvents =
        await this.contracts.l2.L2StandardBridge.queryFilter(
          this.contracts.l2.L2StandardBridge.filters.WithdrawalInitiated(
            undefined,
            undefined,
            address
          ),
          fromBlock,
          toBlock
        )

      for (const event of withdrawEvents) {
        messages.push(await parseTokenEvent(event, MessageDirection.L2_TO_L1))
      }
    }

    return messages
  }

  public async getMessageStatus(message: MessageLike): Promise<MessageStatus> {
    // First, simply check if the message has already been relayed
    const receipt = await this.getMessageReceipt(message)
    if (receipt) {
      if (receipt.receiptStatus === MessageReceiptStatus.RELAYED_SUCCEEDED) {
        return MessageStatus.RELAYED
      } else {
        // If the message has been relayed, but the receipt indicates that it failed, then we can
        // effectively treat it as if it's ready to be relayed but hasn't been.
        return MessageStatus.READY_FOR_RELAY
      }
    }

    // L1 to L2 messages get relayed automatically once they're underneath enough confirmations.
    // Since we know that the message hasn't been relayed yet (from the above checks), then if
    // this is an L1 to L2 message it must be unconfirmed.
    const resolved = await this.resolveMessage(message)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      return MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE
    }

    // Since we didn't return above we must be dealing with an L2 to L1 message.
    // Now we need to check if the message is still within its challenge period or not. This
    // requires that we find the state root batch that includes the transaction in which the
    // message was triggered.
    const sendingTxReceipt = await this.getMessageSendReceipt(message)
    const sendingTxIndex = sendingTxReceipt.blockNumber - NUM_L2_GENESIS_BLOCKS
    const stateBatchAppendedEvent =
      await this.getStateBatchAppendedEventForTxIndex(sendingTxIndex)

    // Special case if a state root hasn't been published for this transaction yet.
    if (stateBatchAppendedEvent === null) {
      return MessageStatus.STATE_ROOT_NOT_PUBLISHED
    }

    // Finally we simply check whether the state root batch is old enough that the challenge
    // period has fully elapsed.
    const currentL1Block = await this.l1Provider.getBlockNumber()
    if (
      stateBatchAppendedEvent.blockNumber +
        CHALLENGE_PERIOD_BLOCKS[this.network] <
      currentL1Block
    ) {
      return MessageStatus.READY_FOR_RELAY
    } else {
      return MessageStatus.IN_CHALLENGE_PERIOD
    }
  }

  public async getMessageReceipt(
    message: MessageLike
  ): Promise<MessageReceipt> {
    const resolved = await this.resolveMessage(message)
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
        messageHash,
        receiptStatus: MessageReceiptStatus.RELAYED_SUCCEEDED,
        transactionReceipt:
          await relayedMessageEvents[0].getTransactionReceipt(),
      }
    } else if (relayedMessageEvents.length > 1) {
      // Should never happen!
      throw new ErrMultipleSuccessfulRelays(messageHash)
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
        messageHash,
        receiptStatus: MessageReceiptStatus.RELAYED_FAILED,
        transactionReceipt: await failedRelayedMessageEvents[
          failedRelayedMessageEvents.length - 1
        ].getTransactionReceipt(),
      }
    }

    // Just return null if we didn't find a receipt. Slightly nicer than throwing an error.
    return null
  }

  public async waitForMessageReciept(
    message: MessageLike,
    opts: {
      confirmations?: number
      pollIntervalMs?: number
      loopsBeforeTimeout?: number
    } = {}
  ): Promise<MessageReceipt> {
    let totalLoops = 0
    while (totalLoops < opts.loopsBeforeTimeout) {
      const receipt = await this.getMessageReceipt(message)
      if (receipt !== null) {
        return receipt
      } else {
        totalLoops += 1
        await sleep(opts.pollIntervalMs)
      }
    }

    // We timed out, throw an error so the user knows.
    throw new ErrTimeoutReached()
  }

  public async estimateMessageExecutionGas(
    message: MessageLike
  ): Promise<BigNumber> {}

  public async estimateMessageWaitTimeSeconds(
    message: MessageLike
  ): Promise<number> {
    const waitTimeBlocks = await this.estimateMessageWaitTimeBlocks(message)
    const blockIntervalSeconds = L1_BLOCK_INTERVAL_SECONDS[this.network]
    return waitTimeBlocks * blockIntervalSeconds
  }

  public async estimateMessageWaitTimeBlocks(
    message: MessageLike
  ): Promise<number> {
    // First, simply check if the message has already been relayed
    const receipt = await this.getMessageReceipt(message)
    if (receipt) {
      return 0
    }

    // L1 to L2 messages get relayed automatically once they're underneath enough confirmations.
    // Since we know that the message hasn't been relayed yet (from the above checks), then if
    // this is an L1 to L2 message it must be unconfirmed.
    const resolved = await this.resolveMessage(message)
    const sendingTxReceipt = await this.getMessageSendReceipt(message)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      return Math.max(
        L1_TO_L2_TX_CONFIRMATIONS[this.network] -
          sendingTxReceipt.confirmations,
        0
      )
    }

    // Since we didn't return above we must be dealing with an L2 to L1 message.
    // Now we need to check if the message is still within its challenge period or not. This
    // requires that we find the state root batch that includes the transaction in which the
    // message was triggered.
    const sendingTxIndex = sendingTxReceipt.blockNumber - NUM_L2_GENESIS_BLOCKS
    const stateBatchAppendedEvent =
      await this.getStateBatchAppendedEventForTxIndex(sendingTxIndex)

    // Special case if a state root hasn't been published for this transaction yet.
    if (stateBatchAppendedEvent === null) {
      // TODO: Maybe need to return undefined here? Or throw? Idk.
      return null
    }

    // Number of blocks remaining is simply the block at which the challenge period expires minus
    // the current L1 block number.
    const currentL1Block = await this.l1Provider.getBlockNumber()
    return Math.max(
      stateBatchAppendedEvent.blockNumber +
        CHALLENGE_PERIOD_BLOCKS[this.network] -
        currentL1Block,
      0
    )
  }

  /**
   * Resolves a MessageLike object into a message. USeful so we can treat transactions like
   * messages as long as they only emit one SentMessage event.
   *
   * @param message MessageLike object to resolve into a CrossChainMessage.
   * @returns Resolved CrossChainMessage.
   */
  private async resolveMessage(
    message: MessageLike
  ): Promise<CrossChainMessage> {
    if ((message as CrossChainMessage).message) {
      return message as CrossChainMessage
    } else if ((message as TokenBridgeMessage).raw) {
      return (message as TokenBridgeMessage).raw
    } else {
      const messages = await this.getMessagesByTransaction(
        message as TransactionLike
      )

      if (messages.length !== 1) {
        throw new Error(`expected 1 message, got ${messages.length}`)
      }

      return messages[0]
    }
  }

  /**
   * Gets the receipt of the transaction that sent a given message.
   * TODO: Maybe make this public?
   *
   * @param message Message to find a sending transaction for.
   * @returns Receipt of the transaction that sent the message.
   */
  private async getMessageSendReceipt(
    message: MessageLike
  ): Promise<TransactionReceipt> {
    const resolved = await this.resolveMessage(message)
    const messageHash = hashCrossChainMessage(resolved)

    // Here we want the messenger that sent the message.
    const messenger =
      resolved.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l1.L1CrossDomainMessenger
        : this.contracts.l2.L2CrossDomainMessenger

    const sentMessageEvents = await messenger.queryFilter(
      messenger.filters.SentMessage(messageHash)
    )

    // Great, we found the message. Convert it into a transaction receipt.
    if (sentMessageEvents.length === 1) {
      return sentMessageEvents[0].getTransactionReceipt()
    } else if (sentMessageEvents.length > 1) {
      // Should never happen!
      throw new ErrSentMessageMultipleTimes(messageHash)
    } else {
      throw new ErrSentMessageNotFound(messageHash)
    }
  }

  private async getStateBatchAppendedEventForTxIndex(
    transactionIndex: number
  ): Promise<Event> {
    const getStateBatchAppendedEventByBatchIndex = async (
      index: number
    ): Promise<ethers.Event | null> => {
      const eventQueryResult =
        await this.contracts.l1.StateCommitmentChain.queryFilter(
          this.contracts.l1.StateCommitmentChain.filters.StateBatchAppended(
            index
          )
        )
      if (eventQueryResult.length === 0) {
        return null
      } else {
        return eventQueryResult[0]
      }
    }

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
      await getStateBatchAppendedEventByBatchIndex(upperBound)

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
      batchEvent = await getStateBatchAppendedEventByBatchIndex(middleOfBounds)

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
}
