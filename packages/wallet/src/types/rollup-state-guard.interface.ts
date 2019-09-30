import {
  SignedTransaction,
  RollupBlock,
  RollupTransitionPosition,
  RollupTransition,
  RollupTransaction,
  FraudCheckResult,
} from './types'

export interface RollupStateGuard {
  /**
   * Gets the most recent transition and block number wich the guard has verified so far.
   *
   * @returns The RollupTransitionPosition up to which the guard has currently verified.
   */
  getCurrentVerifiedPosition(): Promise<RollupTransitionPosition>

  /**
   * Converts a transition into a transaction to be parsed by the transitioner.
   *
   * @returns The RollupTransitionPosition up to which the guard has currently verified.
   */
  getTransactionFromTransition(transition: RollupTransition): Promise<SignedTransaction> 

  /**
   * Applies the next transition as a transaction to the rollup state machine.
   *
   * @param nextTransition The next transition which was rolled up.
   * @param nextRolledupRoot The next root which was rolled up, which should be compared.
   * @returns The FraudCheckResult resulting from the check
   */
  checkNextEncodedTransition(
    encodedNextTransition: string,
    nextRolledUpRoot: Buffer
  ): Promise<FraudCheckResult>

  /**
   * Checks a block of transitions once downloaded from ethereum
   *
   * @param nextBlock The block to be checked for fraud
   * @returns The FraudCheckResult resulting from the check
   */
  checkNextBlock(nextBlock: RollupBlock): Promise<FraudCheckResult>
}

export class LocalMachineError extends Error {
  constructor() {
    super(
      'Transaction application failed for a reason other than the tx being invalid!'
    )
  }
}
