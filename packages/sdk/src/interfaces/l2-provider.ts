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
}
