import { Provider, TransactionRequest } from '@ethersproject/abstract-provider'
import { BigNumber } from 'ethers'

/**
 * Represents an extended version of an normal ethers Provider that returns additional L2 info and
 * has special functions for L2-specific interactions.
 */
export interface L2Provider extends Provider {
  /**
   * Gets the current L1 (data) gas price.
   *
   * @returns Current L1 data gas price in wei.
   */
  getL1GasPrice(): Promise<BigNumber>

  /**
   * Gets the current L2 (execution) gas price.
   *
   * @returns Current L2 execution gas price in wei.
   */
  getL2GasPrice(): Promise<BigNumber>

  /**
   * Estimates the L1 (data) gas required for a transaction.
   *
   * @param tx Transaction to estimate L1 gas for.
   * @returns Estimated L1 gas.
   */
  estimateL1Gas(tx: TransactionRequest): Promise<BigNumber>

  /**
   * Estimates the L2 (execution) gas required for a transaction.
   *
   * @param tx Transaction to estimate L2 gas for.
   * @returns Estimated L2 gas.
   */
  estimateL2Gas(tx: TransactionRequest): Promise<BigNumber>

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
   * gas cost by the current L2 gas price.
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
}

/**
 * Converts a normal ethers Provider into an L2Provider.
 *
 * @param provider Provider to convert into an L2Provider or JSON-RPC url.
 * @returns Provider as an L2Provider.
 */
export const toL2Provider = (provider: string | Provider): L2Provider => {
  throw new Error('Not implemented')
}
