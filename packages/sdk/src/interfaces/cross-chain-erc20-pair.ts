import { Overrides, Contract } from 'ethers'
import {
  TransactionRequest,
  TransactionResponse,
} from '@ethersproject/abstract-provider'

import { NumberLike } from './types'
import { ICrossChainMessenger } from './cross-chain-messenger'

/**
 * Represents an L1<>L2 ERC20 token pair.
 */
export interface ICrossChainERC20Pair {
  /**
   * Messenger that will be used to carry out cross-chain iteractions.
   */
  messenger: ICrossChainMessenger

  /**
   * Ethers Contract object connected to the L1 token.
   */
  l1Token: Contract

  /**
   * Ethers Contract object connected to the L2 token.
   */
  l2Token: Contract

  /**
   * Deposits some tokens into the L2 chain.
   *
   * @param amount Amount of the token to deposit.
   * @param opts Additional options.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  deposit(
    amount: NumberLike,
    opts?: {
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse>

  /**
   * Withdraws some tokens back to the L1 chain.
   *
   * @param amount Amount of the token to withdraw.
   * @param opts Additional options.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  withdraw(
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
     * Generates a transaction for depositing some tokens into the L2 chain.
     *
     * @param amount Amount of the token to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the tokens.
     */
    deposit(
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionResponse>

    /**
     * Generates a transaction for withdrawing some tokens back to the L1 chain.
     *
     * @param amount Amount of the token to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdraw(
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
     * @param amount Amount of the token to deposit.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the tokens.
     */
    deposit(
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionResponse>

    /**
     * Estimates gas required to withdraw some tokens back to the L1 chain.
     *
     * @param amount Amount of the token to withdraw.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdraw(
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest>
  }
}
