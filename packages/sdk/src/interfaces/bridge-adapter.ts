import { Contract, Overrides, Signer, BigNumber } from 'ethers'
import {
  TransactionRequest,
  TransactionResponse,
  BlockTag,
} from '@ethersproject/abstract-provider'

import {
  NumberLike,
  AddressLike,
  MessageDirection,
  TokenBridgeMessage,
} from './types'
import { ICrossChainMessenger } from './cross-chain-messenger'

/**
 * Represents an adapter for an L1<>L2 token bridge. Each custom bridge currently needs its own
 * adapter because the bridge interface is not standardized. This may change in the future.
 */
export interface IBridgeAdapter {
  /**
   * Provider used to make queries related to cross-chain interactions.
   */
  messenger: ICrossChainMessenger

  /**
   * L1 bridge contract.
   */
  l1Bridge: Contract

  /**
   * L2 bridge contract.
   */
  l2Bridge: Contract

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
    }
  ): Promise<TokenBridgeMessage[]>

  /**
   * Gets all deposits for a given address.
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
   * Gets all withdrawals for a given address.
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
   * Checks whether the given token pair is supported by the bridge.
   *
   * @param l1Token The L1 token address.
   * @param l2Token The L2 token address.
   * @returns Whether the given token pair is supported by the bridge.
   */
  supportsTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<boolean>

  /**
   * Deposits some tokens into the L2 chain.
   *
   * @param l1Token The L1 token address.
   * @param l2Token The L2 token address.
   * @param amount Amount of the token to deposit.
   * @param signer Signer used to sign and send the transaction.
   * @param opts Additional options.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  deposit(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    signer: Signer,
    opts?: {
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Withdraws some tokens back to the L1 chain.
   *
   * @param l1Token The L1 token address.
   * @param l2Token The L2 token address.
   * @param amount Amount of the token to withdraw.
   * @param signer Signer used to sign and send the transaction.
   * @param opts Additional options.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  withdraw(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    signer: Signer,
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
     * Generates a transaction for depositing some tokens into the L2 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the tokens.
     */
    deposit(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>

    /**
     * Generates a transaction for withdrawing some tokens back to the L1 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdraw(
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
     * Estimates gas required to deposit some tokens into the L2 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    deposit(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber>

    /**
     * Estimates gas required to withdraw some tokens back to the L1 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    withdraw(
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber>
  }
}
