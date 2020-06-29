import { Contract, ethers, Wallet } from 'ethers'
import { NULL_ADDRESS } from '@eth-optimism/core-utils'
import {
  FraudProofWitness,
  StateTrieWitness,
  AccountTrieWitness,
  isAccountTrieWitness,
} from './interfaces/witness.interface'
import { ABI } from './utils/abi'

/**
 * Fraud proving utility class. Handles everything necessary to prove that a
 * given transaction was fraudulently executed, given relevant witness data.
 */
export class FraudProver {
  private _preStateTransitionIndex: number
  private _witnesses: FraudProofWitness[]
  private _wallet: Wallet
  private _fraudVerifierContract: Contract
  private _stateTransitionerContract: Contract
  private _stateManagerContract: Contract

  /**
   * Creates the fraud prover.
   * @param preStateTransitionIndex Index of the particular state transition
   * that was fraudulently executed.
   * @param witnesses Witness data for the various inclusion proofs necessary
   * to execute the transaction.
   * @param wallet `ethers` wallet instance to be used for contract calls.
   * @param fraudVerifierContract FraudVerifier contract instance to publish
   * relevant information to.
   */
  constructor(
    preStateTransitionIndex: number,
    witnesses: FraudProofWitness[],
    wallet: Wallet,
    fraudVerifierContract: Contract
  ) {
    this._preStateTransitionIndex = preStateTransitionIndex
    this._witnesses = witnesses
    this._wallet = wallet
    this._fraudVerifierContract = fraudVerifierContract
  }

  /**
   * Executes the full fraud proof process.
   */
  public async prove(): Promise<void> {
    // Prepare to run our proof.
    await this._initializeContracts()

    // Publish all witness data to the state transitioner.
    await this._proveAllWitnessInclusion()

    // Execute the fraudulent transaction.
    await this._applyTransaction()

    // Have the state manager compute the correct root given our updates.
    await this._popAllTrieUpdates()

    // Finalize the transition with the state manager.
    await this._completeTransition()

    // Have the fraud verifier confirm that the state roots do not match.
    await this._verifyFraud()
  }

  /* Contract Initialization */

  /**
   * Pulls the address of the state transitioner contract to interact with. If
   * a contract already exists, it simply pulls the address. Otherwise, it
   * creates a new state transitioner entirely.
   * @returns State transitioner contract address.
   */
  private async _getStateTransitioner(): Promise<string> {
    let stateTransitionerAddress = await this._getExistingStateTransitioner()

    if (stateTransitionerAddress === NULL_ADDRESS) {
      stateTransitionerAddress = await this._createStateTransitioner()
    }

    return stateTransitionerAddress
  }

  /**
   * Gets the address for an existing state transitioner.
   * @returns Existing state transitioner address.
   */
  private async _getExistingStateTransitioner(): Promise<string> {
    return this._fraudVerifierContract.stateTransitioners(
      this._preStateTransitionIndex
    )
  }

  /**
   * Creates a new state transitioner address with our state transition index.
   * @returns Address of the newly created state transitioner.
   */
  private async _createStateTransitioner(): Promise<string> {
    await this._fraudVerifierContract.initNewStateTransitioner(
      this._preStateTransitionIndex
    )
    return this._fraudVerifierContract.stateTransitioners(
      this._preStateTransitionIndex
    )
  }

  /**
   * Gets the address for our state manager.
   * @returns State manager address.
   */
  private async _getStateManager(): Promise<string> {
    return this._stateTransitionerContract.stateManager()
  }

  /**
   * Initializes all of the necessary contracts. Gets (or creates) a state
   * transitioner, then pulls the transitioner's associated state manager.
   */
  private async _initializeContracts(): Promise<void> {
    this._fraudVerifierContract.connect(this._wallet)

    const stateTransitionerAddress = await this._getStateTransitioner()
    this._stateTransitionerContract = new ethers.Contract(
      stateTransitionerAddress,
      ABI.STATE_TRANSITIONER_ABI,
      this._wallet
    )

    const stateManagerAddress = await this._getStateManager()
    this._stateManagerContract = new ethers.Contract(
      stateManagerAddress,
      ABI.STATE_MANAGER_ABI,
      this._wallet
    )
  }

  /* Pre-Execution: Inclusion Proof Publication */

  /**
   * Proves inclusion for a given state trie witness (account state).
   * @param witness State trie witness to prove and insert.
   */
  private async _proveStateTrieInclusion(
    witness: StateTrieWitness
  ): Promise<void> {
    await this._fraudVerifierContract.proveContractInclusion(
      witness.key,
      witness.root,
      witness.proof,
      witness.value.nonce,
      witness.value.balance,
      witness.value.storageRoot,
      witness.value.codeHash
    )
  }

  /**
   * Proves inclusion for a given account trie witness (storage slots).
   * @param witness Account trie witness to prove and insert.
   */
  private async _proveAccountTrieInclusion(
    witness: AccountTrieWitness
  ): Promise<void> {
    await this._fraudVerifierContract.proveStorageSlotInclusion(
      witness.stateTrieWitness.key,
      witness.stateTrieWitness.root,
      witness.stateTrieWitness.proof,
      witness.accountTrieWitness.key,
      witness.accountTrieWitness.proof,
      witness.accountTrieWitness.value
    )
  }

  /**
   * Proves inclusion of all provided witnesses sequentially.
   */
  private async _proveAllWitnessInclusion(): Promise<void> {
    for (const witness of this._witnesses) {
      if (isAccountTrieWitness(witness)) {
        await this._proveAccountTrieInclusion(witness)
      } else {
        await this._proveStateTrieInclusion(witness)
      }
    }
  }

  /* Post-Execution: Root Updates */

  /**
   * Gets the total number of state trie updates remaining to be processed.
   * @returns Remaining state trie updates.
   */
  private async _getRemainingStateTrieUpdates(): Promise<number> {
    return this._stateManagerContract.updatedContractsCounter()
  }

  /**
   * Gets the total number of account trie updates remaining to be processed.
   * @returns Remaining account trie updates.
   */
  private async _getRemainingAccountTrieUpdates(): Promise<number> {
    return this._stateManagerContract.updatedStorageSlotCounter()
  }

  /**
   * Processes a single state trie update and modifies root accordingly.
   */
  private async _popStateTrieUpdate(): Promise<void> {
    await this._stateTransitionerContract.proveUpdatedContract()
  }

  /**
   * Processes all state trie updates and modifies root accordingly.
   */
  private async _popAllStateTrieUpdates(): Promise<void> {
    let remainingStateTrieUpdates = await this._getRemainingStateTrieUpdates()

    while (remainingStateTrieUpdates > 0) {
      await this._popStateTrieUpdate()
      remainingStateTrieUpdates = await this._getRemainingStateTrieUpdates()
    }
  }

  /**
   * Processes a single account trie update and modifies root accordingly.
   */
  private async _popAccountTrieUpdate(): Promise<void> {
    await this._stateTransitionerContract.proveUpdatedStorageSlot()
  }

  /**
   * Processes all account trie updates and modifies root accordingly.
   */
  private async _popAllAccountTrieUpdates(): Promise<void> {
    let remainingAccountTrieUpdates = await this._getRemainingAccountTrieUpdates()

    while (remainingAccountTrieUpdates > 0) {
      await this._popAccountTrieUpdate()
      remainingAccountTrieUpdates = await this._getRemainingAccountTrieUpdates()
    }
  }

  /**
   * Processes all state/account trie updates and modifies root accordingly.
   */
  private async _popAllTrieUpdates(): Promise<void> {
    await this._popAllAccountTrieUpdates()
    await this._popAllStateTrieUpdates()
  }

  /* Process Advancement */

  /**
   * Applies the fraudulent transaction to the state transitioner. Called once
   * all inclusion proofs have been successfully applied. Moves us from the
   * pre-execution phase to the post-execution phase.
   */
  private async _applyTransaction(): Promise<void> {
    await this._stateTransitionerContract.applyTransaction()
  }

  /**
   * Completes the transition process. Called after all updates have been
   * reflected in the new root. Allows us to finally verify that the transition
   * was fraudulent.
   */
  private async _completeTransition(): Promise<void> {
    await this._stateTransitionerContract.completeTransition()
  }

  /**
   * Verifies that the transition was indeed fraudulent. Called once the
   * transition process has been completed.
   */
  private async _verifyFraud(): Promise<void> {
    await this._fraudVerifierContract.verifyFraud()
  }
}
