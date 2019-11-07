import { Address, SignedStateReceipt, StateReceipt } from './types'
import { ImplicationProofItem } from '@pigi/ovm'

export interface RollupStateSolver {
  /**
   * Stores the SignedStateReceipt
   * @param signedReceipt The signed receipt
   */
  storeSignedStateReceipt(signedReceipt: SignedStateReceipt): Promise<void>

  /**
   * Determines whether or not the provided StateReceipt is valid, checking that
   * there is a signature for it, and it has a valid inclusion proof.
   * @param stateReceipt The state receipt to check validity
   * @param aggregator The aggregator from which the receipt was received
   * @returns True if the receipt is provably valid, false otherwise
   */
  isStateReceiptProvablyValid(
    stateReceipt: StateReceipt,
    aggregator: Address
  ): Promise<boolean>

  /**
   * Gets the proof that the provided state receipt is valid.
   * @param stateReceipt The State Receipt in question
   * @param signer The Signer of the StateReceipt
   * @returns The implication proof items of state receipt being valid, else undefined
   */
  getFraudProof(
    stateReceipt: StateReceipt,
    signer: Address
  ): Promise<ImplicationProofItem[]>
}
