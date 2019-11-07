import {
  SignedTransaction,
  RollupBlock,
  RollupTransitionPosition,
  RollupTransition,
  LocalFraudProof,
  StateSnapshot,
} from './types'
import { ContractFraudProof } from '../validator'

export interface RollupStateValidator {
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
  getTransactionFromTransitionAndSnapshots(
    transition: RollupTransition,
    snapshots: StateSnapshot[]
  ): Promise<SignedTransaction>
  /**
   * Applies the next transition as a transaction to the rollup state machine.
   *
   * @param nextTransition The next transition which was rolled up.
   * @returns The LocalFraudProof resulting from the check
   */
  checkNextTransition(
    nextTransition: RollupTransition
  ): Promise<LocalFraudProof>

  /**
   * Checks the next block of stored transitions
   *
   * @param blockNumber The block nunmber of the stored block to be ingested
   * @returns The ContractFraudProof resulting from the check
   */
  validateStoredBlock(blockNumber: number): Promise<ContractFraudProof>

  /**
   * Stores a block of transitions for provessing
   *
   * @param newBlock The block to be stored for later ingestion
   * @returns The LocalFraudProof resulting from the check
   */
  storeBlock(newBlock: RollupBlock): Promise<void>
}

export class LocalMachineError extends Error {
  constructor() {
    super(
      'Transaction application failed for a reason other than the tx being invalid!'
    )
  }
}

export class ValidationOutOfOrderError extends Error {
  constructor() {
    super('Blocks were fed to the validator out of sync.')
  }
}

export class AggregatorUnsupportedError extends Error {
  constructor() {
    super(
      'We are currently unable to guard addresses which are created non-sequentially.'
    )
  }
}
