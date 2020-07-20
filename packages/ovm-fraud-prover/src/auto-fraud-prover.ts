/* External Imports */
import * as rlp from 'rlp'
import { Contract, ethers, Wallet } from 'ethers'
import { NULL_ADDRESS } from '@eth-optimism/core-utils'
import { BaseTrie, SecureTrie } from 'merkle-patricia-tree'

/* Internal Imports */
import {
  FraudProofWitness,
  StateTrieWitness,
  AccountTrieWitness,
  OVMStateElementInclusionProof,
  OVMTransactionElementInclusionProof,
  isAccountTrieWitness,
  OVMTransactionData
} from './interfaces'
import {
  ABI,
  GAS_LIMIT,
  toHexBuffer,
  encodeAccountState,
  decodeAccountState,
  updateAndProve,
  toHexString
} from './utils'

/**
 * Fraud proving utility class. Handles everything necessary to prove that a
 * given transaction was fraudulently executed, given relevant witness data.
 */
export class AutoFraudProver {
  private _preStateTransitionIndex: number
  private _preStateRoot: string
  private _preStateInclusionProof: OVMStateElementInclusionProof
  private _postStateRoot: string
  private _postStateInclusionProof: OVMStateElementInclusionProof
  private _transaction: OVMTransactionData
  private _transactionInclusionProof: OVMTransactionElementInclusionProof
  private _witnesses: FraudProofWitness[]
  private _wallet: Wallet

  private _stateTrie: BaseTrie
  private _accountTries: {
    [account: string]: BaseTrie
  }

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
    preStateRoot: string,
    preStateInclusionProof: OVMStateElementInclusionProof,
    postStateRoot: string,
    postStateInclusionProof: OVMStateElementInclusionProof,
    transaction: OVMTransactionData,
    transactionInclusionProof: OVMTransactionElementInclusionProof,
    witnesses: FraudProofWitness[],
    wallet: Wallet,
    fraudVerifierContract: Contract
  ) {
    this._preStateTransitionIndex = preStateTransitionIndex
    this._preStateRoot = preStateRoot
    this._preStateInclusionProof = preStateInclusionProof
    this._postStateRoot = postStateRoot
    this._postStateInclusionProof = postStateInclusionProof
    this._transaction = transaction
    this._transactionInclusionProof = transactionInclusionProof
    this._witnesses = witnesses
    this._wallet = wallet
    this._fraudVerifierContract = fraudVerifierContract
    this._accountTries = {}
  }

  /**
   * Executes the full fraud proof process.
   */
  public async prove(): Promise<void> {
    // Set up our tries.
    await this._makeStateTrie()
    await this._makeAccountTries()

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
    await this._fraudVerifierContract.initializeFraudVerification(
      this._preStateTransitionIndex,
      this._preStateRoot,
      this._preStateInclusionProof,
      this._transaction,
      this._transactionInclusionProof,
      {
        gasLimit: GAS_LIMIT
      }
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
    await this._stateTransitionerContract.proveContractInclusion(
      witness.ovmContractAddress,
      witness.codeContractAddress,
      witness.value.nonce,
      rlp.encode(witness.proof),
      {
        gasLimit: GAS_LIMIT
      }
    )
  }

  /**
   * Proves inclusion for a given account trie witness (storage slots).
   * @param witness Account trie witness to prove and insert.
   */
  private async _proveAccountTrieInclusion(
    witness: AccountTrieWitness
  ): Promise<void> {
    await this._stateTransitionerContract.proveStorageSlotInclusion(
      witness.stateTrieWitness.ovmContractAddress,
      witness.accountTrieWitness.key,
      witness.accountTrieWitness.value,
      rlp.encode(witness.stateTrieWitness.proof),
      rlp.encode(witness.accountTrieWitness.proof),
      {
        gasLimit: GAS_LIMIT
      }
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
    const [
      ovmContractAddress,
      updatedNonce,
      updatedCodeHash
    ] = await this._stateManagerContract.peekUpdatedContract()

    const oldAccountState = decodeAccountState(
      await this._stateTrie.get(
        toHexBuffer(ethers.utils.keccak256(ovmContractAddress))
      )
    )

    const proof = await updateAndProve(
      this._stateTrie,
      toHexBuffer(ethers.utils.keccak256(ovmContractAddress)),
      encodeAccountState({
        ...oldAccountState,
        ...{
          nonce: updatedNonce.toNumber(),
          codeHash: updatedCodeHash
        }
      })
    )

    await this._stateTransitionerContract.proveUpdatedContract(proof,
      {
        gasLimit: GAS_LIMIT
      }
    )
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
    const [
      ovmContractAddress,
      updatedSlotKey,
      updatedSlotValue
    ] = await this._stateManagerContract.peekUpdatedStorageSlot()

    if (!this._accountTries[ovmContractAddress]) {
      this._accountTries[ovmContractAddress] = new BaseTrie()
    }

    const trie = this._accountTries[ovmContractAddress]

    const storageTrieProof = await updateAndProve(
      trie,
      toHexBuffer(ethers.utils.keccak256(updatedSlotKey)),
      toHexBuffer(updatedSlotValue),
    )
    const oldAccountState = decodeAccountState(
      await this._stateTrie.get(
        toHexBuffer(ethers.utils.keccak256(ovmContractAddress))
      )
    )

    const newAccountState = {
      ...oldAccountState,
      codeHash: toHexString(trie.root)
    }

    const stateTrieProof = await updateAndProve(
      this._stateTrie,
      toHexBuffer(ethers.utils.keccak256(ovmContractAddress)),
      encodeAccountState(newAccountState)
    )
  
    await this._stateTransitionerContract.proveUpdatedStorageSlot(
      stateTrieProof,
      storageTrieProof,
      {
        gasLimit: GAS_LIMIT
      }
    )
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
    await this._popAllStateTrieUpdates()
    await this._popAllAccountTrieUpdates()
  }

  /* Process Advancement */

  /**
   * Applies the fraudulent transaction to the state transitioner. Called once
   * all inclusion proofs have been successfully applied. Moves us from the
   * pre-execution phase to the post-execution phase.
   */
  private async _applyTransaction(): Promise<void> {
    await this._stateTransitionerContract.applyTransaction(this._transaction,
      {
        gasLimit: GAS_LIMIT
      }
    )
  }

  /**
   * Completes the transition process. Called after all updates have been
   * reflected in the new root. Allows us to finally verify that the transition
   * was fraudulent.
   */
  private async _completeTransition(): Promise<void> {
    await this._stateTransitionerContract.completeTransition(
      {
        gasLimit: GAS_LIMIT
      }
    )
  }

  /**
   * Verifies that the transition was indeed fraudulent. Called once the
   * transition process has been completed.
   */
  private async _verifyFraud(): Promise<void> {
    await this._fraudVerifierContract.finalizeFraudVerification(
      this._preStateTransitionIndex,
      this._preStateRoot,
      this._preStateInclusionProof,
      this._postStateRoot,
      this._postStateInclusionProof,
      {
        gasLimit: GAS_LIMIT
      }
    )
  }

  /**
   * Generates the state trie from the provided witnesses.
   */
  private async _makeStateTrie(): Promise<void> {
    const witnesses = this.getStateTrieWitnesses();

    const firstRootNode = witnesses[0].proof[0];
    const allNonRootNodes: Buffer[] = witnesses.reduce((nodes, witness) => {
      return nodes.concat(witness.proof.slice(1));
    }, [])
    const allNodes = [firstRootNode].concat(allNonRootNodes)

    this._stateTrie = await BaseTrie.fromProof(allNodes)
  }

  /**
   * Generates all account tries from the provided witnesses.
   */
  private async _makeAccountTries(): Promise<void> {
    const witnesses = this.getAccountTrieWitnesses();
    const witnessMap = witnesses.reduce((map: {
      [address: string]: Buffer[]
    }, witness) => {
      const ovmContractAddress = witness.stateTrieWitness.ovmContractAddress
      if (!(ovmContractAddress in map)) {
        map[ovmContractAddress] = [
          toHexBuffer(witness.stateTrieWitness.value.storageRoot)
        ]
      }

      map[ovmContractAddress] = map[ovmContractAddress].concat(witness.accountTrieWitness.proof.slice(1))
      return map;
    }, {})

    for (const ovmContractAddress in witnessMap) {
      const proof = witnessMap[ovmContractAddress]
      this._accountTries[ovmContractAddress] = new SecureTrie((await BaseTrie.fromProof(proof)).db)
    }
  }

  /**
   * Picks out the state trie witnesses from the list of witnesses.
   * @returns List of state trie witnesses.
   */
  private getStateTrieWitnesses(): StateTrieWitness[] {
    const witnesses: StateTrieWitness[] = []

    for (const witness of this._witnesses) {
      if (!isAccountTrieWitness(witness)) {
        witnesses.push(witness)
      }
    }

    return witnesses;
  }

  /**
   * Picks out the account trie witnesses from the list of witnesses.
   * @returns List of account trie witnesses.
   */
  private getAccountTrieWitnesses(): AccountTrieWitness[] {
    const witnesses: AccountTrieWitness[] = []

    for (const witness of this._witnesses) {
      if (isAccountTrieWitness(witness)) {
        witnesses.push(witness)
      }
    }

    return witnesses;
  }
}
