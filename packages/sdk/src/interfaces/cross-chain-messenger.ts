import { Overrides, Signer, BigNumber } from 'ethers'
import {
  TransactionRequest,
  TransactionResponse,
} from '@ethersproject/abstract-provider'

import { MessageLike, NumberLike, CrossChainMessageRequest } from './types'
import { ICrossChainProvider } from './cross-chain-provider'

/**
 * Represents a utility class for making L1/L2 cross-chain transactions.
 */
export interface ICrossChainMessenger {
  /**
   * Provider that will be used to interact with the L1/L2 system.
   */
  provider: ICrossChainProvider

  /**
   * Signer that will carry out L1 transactions.
   */
  l1Signer: Signer

  /**
   * Signer that will carry out L2 transactions.
   */
  l2Signer: Signer

  /**
   * Sends a given cross chain message. Where the message is sent depends on the direction attached
   * to the message itself.
   *
   * @param message Cross chain message to send.
   * @param opts Additional options.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message sending transaction.
   */
  sendMessage(
    message: CrossChainMessageRequest,
    opts?: {
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
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message resending transaction.
   */
  resendMessage(
    message: MessageLike,
    messageGasLimit: NumberLike,
    opts?: {
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Finalizes a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
   * messages. Will throw an error if the message has not completed its challenge period yet.
   *
   * @param message Message to finalize.
   * @param opts Additional options.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the finalization transaction.
   */
  finalizeMessage(
    message: MessageLike,
    opts?: {
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Deposits some ETH into the L2 chain.
   *
   * @param amount Amount of ETH to deposit (in wei).
   * @param opts Additional options.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  depositETH(
    amount: NumberLike,
    opts?: {
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Withdraws some ETH back to the L1 chain.
   *
   * @param amount Amount of ETH to withdraw.
   * @param opts Additional options.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  withdrawETH(
    amount: NumberLike,
    opts?: {
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
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdrawETH(
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
     * @returns Transaction that can be signed and executed to send the message.
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
     * @returns Transaction that can be signed and executed to resend the message.
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
     * @returns Transaction that can be signed and executed to finalize the message.
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
     * @returns Transaction that can be signed and executed to deposit the ETH.
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
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdrawETH(
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>
  }
}
