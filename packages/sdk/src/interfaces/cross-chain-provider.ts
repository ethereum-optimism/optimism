import { BigNumber } from 'ethers'
import { Provider, BlockTag } from '@ethersproject/abstract-provider'
import {
  MessageLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  CrossChainMessage,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  OEContracts,
  MessageReceipt,
  CustomBridges,
} from './types'

/**
 * Represents the L1/L2 connection. Only handles read requests. If you want to send messages, use
 * the CrossChainMessenger contract which takes a CrossChainProvider and a signer as inputs.
 */
export interface ICrossChainProvider {
  /**
   * Provider connected to the L1 chain.
   */
  l1Provider: Provider

  /**
   * Provider connected to the L2 chain.
   */
  l2Provider: Provider

  /**
   * Chain ID for the L1 network.
   */
  l1ChainId: number

  /**
   * Contract objects attached to their respective providers and addresses.
   */
  contracts: OEContracts

  /**
   * List of custom bridges for the given network.
   */
  bridges: CustomBridges

  /**
   * Retrieves all cross chain messages sent within a given transaction.
   *
   * @param transaction Transaction hash or receipt to find messages from.
   * @param opts Options object.
   * @param opts.direction Direction to search for messages in. If not provided, will attempt to
   * automatically search both directions under the assumption that a transaction hash will only
   * exist on one chain. If the hash exists on both chains, will throw an error.
   * @returns All cross chain messages sent within the transaction.
   */
  getMessagesByTransaction(
    transaction: TransactionLike,
    opts?: {
      direction?: MessageDirection
    }
  ): Promise<CrossChainMessage[]>

  /**
   * Retrieves all cross chain messages sent by a particular address.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.direction Direction to search for messages in. If not provided, will attempt to
   * find all messages in both directions.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All cross chain messages sent by the particular address.
   */
  getMessagesByAddress(
    address: AddressLike,
    opts?: {
      direction?: MessageDirection
      fromBlock?: NumberLike
      toBlock?: NumberLike
    }
  ): Promise<CrossChainMessage[]>

  /**
   * Finds all cross chain messages that correspond to token deposits or withdrawals sent by a
   * particular address. Useful for finding deposits/withdrawals because the sender of the message
   * will appear to be the StandardBridge contract and not the actual end user.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.direction Direction to search for messages in. If not provided, will attempt to
   * find all messages in both directions.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All token bridge messages sent by the given address.
   */
  getTokenBridgeMessagesByAddress(
    address: AddressLike,
    opts?: {
      direction?: MessageDirection
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]>

  /**
   * Alias for getTokenBridgeMessagesByAddress with a drection of L1_TO_L2.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All deposit token bridge messages sent by the given address.
   */
  getDepositsByAddress(
    address: AddressLike,
    opts?: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]>

  /**
   * Alias for getTokenBridgeMessagesByAddress with a drection of L2_TO_L1.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All withdrawal token bridge messages sent by the given address.
   */
  getWithdrawalsByAddress(
    address: AddressLike,
    opts?: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]>

  /**
   * Resolves a MessageLike into a CrossChainMessage object.
   * Unlike other coercion functions, this function is stateful and requires making additional
   * requests. For now I'm going to keep this function here, but we could consider putting a
   * similar function inside of utils/coercion.ts if people want to use this without having to
   * create an entire CrossChainProvider object.
   *
   * @param message MessageLike to resolve into a CrossChainMessage.
   * @returns Message coerced into a CrossChainMessage.
   */
  toCrossChainMessage(message: MessageLike): Promise<CrossChainMessage>

  /**
   * Retrieves the status of a particular message as an enum.
   *
   * @param message Cross chain message to check the status of.
   * @returns Status of the message.
   */
  getMessageStatus(message: MessageLike): Promise<MessageStatus>

  /**
   * Finds the receipt of the transaction that executed a particular cross chain message.
   *
   * @param message Message to find the receipt of.
   * @returns CrossChainMessage receipt including receipt of the transaction that relayed the
   * given message.
   */
  getMessageReceipt(message: MessageLike): Promise<MessageReceipt>

  /**
   * Waits for a message to be executed and returns the receipt of the transaction that executed
   * the given message.
   *
   * @param message Message to wait for.
   * @param opts Options to pass to the waiting function.
   * @param opts.confirmations Number of transaction confirmations to wait for before returning.
   * @param opts.pollIntervalMs Number of milliseconds to wait between polling for the receipt.
   * @param opts.timeoutMs Milliseconds to wait before timing out.
   * @returns CrossChainMessage receipt including receipt of the transaction that relayed the
   * given message.
   */
  waitForMessageReceipt(
    message: MessageLike,
    opts?: {
      confirmations?: number
      pollIntervalMs?: number
      timeoutMs?: number
    }
  ): Promise<MessageReceipt>

  /**
   * Estimates the amount of gas required to fully execute a given message on L2. Only applies to
   * L1 => L2 messages. You would supply this gas limit when sending the message to L2.
   *
   * @param message Message get a gas estimate for.
   * @returns Estimates L2 gas limit.
   */
  estimateL2MessageGasLimit(message: MessageLike): Promise<BigNumber>

  /**
   * Returns the estimated amount of time before the message can be executed. When this is a
   * message being sent to L1, this will return the estimated time until the message will complete
   * its challenge period. When this is a message being sent to L2, this will return the estimated
   * amount of time until the message will be picked up and executed on L2.
   *
   * @param message Message to estimate the time remaining for.
   * @returns Estimated amount of time remaining (in seconds) before the message can be executed.
   */
  estimateMessageWaitTimeSeconds(message: MessageLike): Promise<number>

  /**
   * Returns the estimated amount of time before the message can be executed (in L1 blocks).
   * When this is a message being sent to L1, this will return the estimated time until the message
   * will complete its challenge period. When this is a message being sent to L2, this will return
   * the estimated amount of time until the message will be picked up and executed on L2.
   *
   * @param message Message to estimate the time remaining for.
   * @returns Estimated amount of time remaining (in blocks) before the message can be executed.
   */
  estimateMessageWaitTimeBlocks(message: MessageLike): Promise<number>
}
