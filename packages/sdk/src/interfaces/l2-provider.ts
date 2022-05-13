import {
  Provider,
  TransactionRequest,
  TransactionResponse,
  Block,
  BlockWithTransactions,
} from '@ethersproject/abstract-provider'
import { BigNumber } from 'ethers'

/**
 * JSON transaction representation when returned by L2Geth nodes. This is simply an extension to
 * the standard transaction response type. You do NOT need to use this type unless you care about
 * having typed access to L2-specific fields.
 */
export interface L2Transaction extends TransactionResponse {
  l1BlockNumber: number
  l1TxOrigin: string
  queueOrigin: string
  rawTransaction: string
}

/**
 * JSON block representation when returned by L2Geth nodes. Just a normal block but with
 * an added stateRoot field.
 */
export interface L2Block extends Block {
  stateRoot: string
}

/**
 * JSON block representation when returned by L2Geth nodes. Just a normal block but with
 * L2Transaction objects instead of the standard transaction response object.
 */
export interface L2BlockWithTransactions extends BlockWithTransactions {
  stateRoot: string
  transactions: [L2Transaction]
}

/**
 * Represents an extended version of an normal ethers Provider that returns additional L2 info and
 * has special functions for L2-specific interactions.
 */
export type L2Provider<TProvider extends Provider> = TProvider & {
  /**
   * Gets the current L1 (data) gas price.
   *
   * @returns Current L1 data gas price in wei.
   */
  getL1GasPrice(): Promise<BigNumber>

  /**
   * Estimates the L1 (data) gas required for a transaction.
   *
   * @param tx Transaction to estimate L1 gas for.
   * @returns Estimated L1 gas.
   */
  estimateL1Gas(tx: TransactionRequest): Promise<BigNumber>

  /**
   * Estimates the L1 (data) gas cost for a transaction in wei by multiplying the estimated L1 gas
   * cost by the current L1 gas price.
   *
   * @param tx Transaction to estimate L1 gas cost for.
   * @returns Estimated L1 gas cost.
   */
  estimateL1GasCost(tx: TransactionRequest): Promise<BigNumber>

  /**
   * Estimates the L2 (execution) gas cost for a transaction in wei by multiplying the estimated L1
   * gas cost by the current L2 gas price. This is a simple multiplication of the result of
   * getGasPrice and estimateGas for the given transaction request.
   *
   * @param tx Transaction to estimate L2 gas cost for.
   * @returns Estimated L2 gas cost.
   */
  estimateL2GasCost(tx: TransactionRequest): Promise<BigNumber>

  /**
   * Estimates the total gas cost for a transaction in wei by adding the estimated the L1 gas cost
   * and the estimated L2 gas cost.
   *
   * @param tx Transaction to estimate total gas cost for.
   * @returns Estimated total gas cost.
   */
  estimateTotalGasCost(tx: TransactionRequest): Promise<BigNumber>

  /**
   * Internal property to determine if a provider is a L2Provider
   * You are likely looking for the isL2Provider function
   */
  _isL2Provider: true
}
