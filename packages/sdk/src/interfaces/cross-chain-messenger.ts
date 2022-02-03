import { Event, BigNumber, Overrides } from 'ethers'
import {
  Provider,
  BlockTag,
  TransactionRequest,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'

import {
  MessageLike,
  MessageRequestLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  CrossChainMessage,
  CrossChainMessageRequest,
  CrossChainMessageProof,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  OEContracts,
  MessageReceipt,
  StateRoot,
  StateRootBatch,
  BridgeAdapters,
} from './types'
import { IBridgeAdapter } from './bridge-adapter'

/**
 * Handles L1/L2 interactions.
 */
export interface ICrossChainMessenger {
  /**
   * Provider connected to the L1 chain.
   */
  l1SignerOrProvider: Signer | Provider

  /**
   * Provider connected to the L2 chain.
   */
  l2SignerOrProvider: Signer | Provider

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
  bridges: BridgeAdapters

  /**
   * Provider connected to the L1 chain.
   */
  l1Provider: Provider

  /**
   * Provider connected to the L2 chain.
   */
  l2Provider: Provider

  /**
   * Signer connected to the L1 chain.
   */
  l1Signer: Signer

  /**
   * Signer connected to the L2 chain.
   */
  l2Signer: Signer

  /**
   * Number of blocks before a deposit is considered confirmed.
   */
  depositConfirmationBlocks: number

  /**
   * Estimated average L1 block time in seconds.
   */
  l1BlockTimeSeconds: number

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
   * Finds the appropriate bridge adapter for a given L1<>L2 token pair. Will throw if no bridges
   * support the token pair or if more than one bridge supports the token pair.
   *
   * @param l1Token L1 token address.
   * @param l2Token L2 token address.
   * @returns The appropriate bridge adapter for the given token pair.
   */
  getBridgeForTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<IBridgeAdapter>

  /**
   * Finds all cross chain messages that correspond to token deposits or withdrawals sent by a
   * particular address. Useful for finding deposits/withdrawals because the sender of the message
   * will appear to be the StandardBridge contract and not the actual end user.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.direction Direction to search for messages in. If not provided, will attempt to
   * find all messages in both directions.
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
   * @param opts Options object.
   * @param opts.bufferPercent Percentage of gas to add to the estimate. Defaults to 20.
   * @param opts.from Address to use as the sender.
   * @returns Estimates L2 gas limit.
   */
  estimateL2MessageGasLimit(
    message: MessageRequestLike,
    opts?: {
      bufferPercent?: number
      from?: string
    }
  ): Promise<BigNumber>

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
   * Queries the current challenge period in seconds from the StateCommitmentChain.
   *
   * @returns Current challenge period in seconds.
   */
  getChallengePeriodSeconds(): Promise<number>

  /**
   * Returns the state root that corresponds to a given message. This is the state root for the
   * block in which the transaction was included, as published to the StateCommitmentChain. If the
   * state root for the given message has not been published yet, this function returns null.
   *
   * @param message Message to find a state root for.
   * @returns State root for the block in which the message was created.
   */
  getMessageStateRoot(message: MessageLike): Promise<StateRoot | null>

  /**
   * Returns the StateBatchAppended event that was emitted when the batch with a given index was
   * created. Returns null if no such event exists (the batch has not been submitted).
   *
   * @param batchIndex Index of the batch to find an event for.
   * @returns StateBatchAppended event for the batch, or null if no such batch exists.
   */
  getStateBatchAppendedEventByBatchIndex(
    batchIndex: number
  ): Promise<Event | null>

  /**
   * Returns the StateBatchAppended event for the batch that includes the transaction with the
   * given index. Returns null if no such event exists.
   *
   * @param transactionIndex Index of the L2 transaction to find an event for.
   * @returns StateBatchAppended event for the batch that includes the given transaction by index.
   */
  getStateBatchAppendedEventByTransactionIndex(
    transactionIndex: number
  ): Promise<Event | null>

  /**
   * Returns information about the state root batch that included the state root for the given
   * transaction by index. Returns null if no such state root has been published yet.
   *
   * @param transactionIndex Index of the L2 transaction to find a state root batch for.
   * @returns State root batch for the given transaction index, or null if none exists yet.
   */
  getStateRootBatchByTransactionIndex(
    transactionIndex: number
  ): Promise<StateRootBatch | null>

  /**
   * Generates the proof required to finalize an L2 to L1 message.
   *
   * @param message Message to generate a proof for.
   * @returns Proof that can be used to finalize the message.
   */
  getMessageProof(message: MessageLike): Promise<CrossChainMessageProof>

  /**
   * Sends a given cross chain message. Where the message is sent depends on the direction attached
   * to the message itself.
   *
   * @param message Cross chain message to send.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message sending transaction.
   */
  sendMessage(
    message: CrossChainMessageRequest,
    opts?: {
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Resends a given cross chain message with a different gas limit. Only applies to L1 to L2
   * messages. If provided an L2 to L1 message, this function will throw an error.
   *
   * @param message Cross chain message to resend.
   * @param messageGasLimit New gas limit to use for the message.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message resending transaction.
   */
  resendMessage(
    message: MessageLike,
    messageGasLimit: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Finalizes a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
   * messages. Will throw an error if the message has not completed its challenge period yet.
   *
   * @param message Message to finalize.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the finalization transaction.
   */
  finalizeMessage(
    message: MessageLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Deposits some ETH into the L2 chain.
   *
   * @param amount Amount of ETH to deposit (in wei).
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  depositETH(
    amount: NumberLike,
    opts?: {
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Withdraws some ETH back to the L1 chain.
   *
   * @param amount Amount of ETH to withdraw.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  withdrawETH(
    amount: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Deposits some ERC20 tokens into the L2 chain.
   *
   * @param l1Token Address of the L1 token.
   * @param l2Token Address of the L2 token.
   * @param amount Amount to deposit.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  depositERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Withdraws some ERC20 tokens back to the L1 chain.
   *
   * @param l1Token Address of the L1 token.
   * @param l2Token Address of the L2 token.
   * @param amount Amount to withdraw.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  withdrawERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Object that holds the functions that generate transactions to be signed by the user.
   * Follows the pattern used by ethers.js.
   */
  populateTransaction: {
    /**
     * Generates a transaction that sends a given cross chain message. This transaction can be signed
     * and executed by a signer.
     *
     * @param message Cross chain message to send.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to send the message.
     */
    sendMessage: (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ) => Promise<TransactionRequest>

    /**
     * Generates a transaction that resends a given cross chain message. Only applies to L1 to L2
     * messages. This transaction can be signed and executed by a signer.
     *
     * @param message Cross chain message to resend.
     * @param messageGasLimit New gas limit to use for the message.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to resend the message.
     */
    resendMessage(
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a message finalization transaction that can be signed and executed. Only
     * applicable for L2 to L1 messages. Will throw an error if the message has not completed
     * its challenge period yet.
     *
     * @param message Message to generate the finalization transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to finalize the message.
     */
    finalizeMessage(
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a transaction for depositing some ETH into the L2 chain.
     *
     * @param amount Amount of ETH to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the ETH.
     */
    depositETH(
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a transaction for withdrawing some ETH back to the L1 chain.
     *
     * @param amount Amount of ETH to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the ETH.
     */
    withdrawETH(
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a transaction for depositing some ERC20 tokens into the L2 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the tokens.
     */
    depositERC20(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a transaction for withdrawing some ERC20 tokens back to the L1 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdrawERC20(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>
  }

  /**
   * Object that holds the functions that estimates the gas required for a given transaction.
   * Follows the pattern used by ethers.js.
   */
  estimateGas: {
    /**
     * Estimates gas required to send a cross chain message.
     *
     * @param message Cross chain message to send.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    sendMessage: (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ) => Promise<BigNumber>

    /**
     * Estimates gas required to resend a cross chain message. Only applies to L1 to L2 messages.
     *
     * @param message Cross chain message to resend.
     * @param messageGasLimit New gas limit to use for the message.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    resendMessage(
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to finalize a cross chain message. Only applies to L2 to L1 messages.
     *
     * @param message Message to generate the finalization transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    finalizeMessage(
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to deposit some ETH into the L2 chain.
     *
     * @param amount Amount of ETH to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    depositETH(
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to withdraw some ETH back to the L1 chain.
     *
     * @param amount Amount of ETH to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    withdrawETH(
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to deposit some ERC20 tokens into the L2 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    depositERC20(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to withdraw some ERC20 tokens back to the L1 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    withdrawERC20(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>
  }
}
