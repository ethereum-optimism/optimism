import { Address, SignedTransaction, StateSnapshot, StateUpdate, RollupBlock } from './types'
import { RollupTransitionPosition, FraudCheckResult } from './types'



export interface RollupStateGuard {
  /**
   * Gets the most recent transition and block number wich the guard has verified so far.
   *
   * @returns The RollupTransitionPosition up to which the guard has currently verified.
   */
  getCurrentVerifiedPosition(): Promise<RollupTransitionPosition>

  /**
   * Gets the state for the provided address, if one exists.
   *
   * @param nextSignedTransaction The next transaction which was rolled up.
   * @param nextRolledupRoot The next root which was rolled up, which should be compared.
   * @returns The FraudCheckResult resulting from the check
   */
  checkNextTransition(nextSignedTransaction: SignedTransaction, nextRolledUpRoot: string): Promise<FraudCheckResult>

  /**
   * Gets the state for the provided address, if one exists.
   *
   * @param nextBlock The block to be checked for fraud
   * @returns The FraudCheckResult resulting from the check
   */
  checkNextBlock(nextBlock: RollupBlock): Promise<FraudCheckResult>
}
