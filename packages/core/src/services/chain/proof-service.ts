/* External Imports */
import { Service } from '@nestd/core'
import BigNum from 'bn.js'
import { MerkleSumTree, Transaction } from 'plasma-utils'
import { validStateTransition } from 'plasma-verifier'

/* Services */
import { ETHProvider } from '../eth/eth-provider'
import { ContractProvider } from '../eth/contract-provider'
import { ChainDB } from '../db/interfaces/chain-db'
import { LoggerService } from '../logger.service'

/* Internal Imports */
import { TransactionProof } from '../../models/chain'
import { StateManager } from './state-manager'

interface PredicateCache {
  [key: string]: string
}

@Service()
export class ProofService {
  private readonly name = 'proof'
  private predicates: PredicateCache = {}

  constructor(
    private readonly logger: LoggerService,
    private readonly eth: ETHProvider,
    private readonly contract: ContractProvider,
    private readonly chaindb: ChainDB
  ) {}

  /**
   * Checks a transaction proof.
   * @param tx The transaction to verify.
   * @param proof A transaction proof for that transaction.
   * @returns the head state at the end of the transaction if valid.
   */
  public async applyProof(proof: TransactionProof): Promise<StateManager> {
    const state = new StateManager()
    const tx = proof.tx

    // Apply deposits.
    this.logger.log(this.name, `Applying deposits for: ${tx.hash}`)
    for (const deposit of proof.deposits) {
      // Validate the deposit.
      const validDeposit = await this.contract.depositValid(deposit)
      if (!validDeposit) {
        throw new Error('Invalid deposit')
      }

      state.addStateObject(deposit)
    }

    // Apply transactions.
    this.logger.log(this.name, `Applying transactions for: ${tx.hash}`)
    for (const transaction of proof.transactions) {
      // Check inclusion proofs.
      try {
        const { start, end } = await this.checkInclusionProof(transaction)
        transaction.newState.implicitStart = start
        transaction.newState.implicitEnd = end
      } catch {
        throw new Error('Invalid transaction inclusion proof')
      }

      const witness = transaction.witness
      const newState = transaction.newState
      const oldStates = state.getOldStates(newState)

      for (const oldState of oldStates) {
        // Validate the state transition using the predicate.
        const bytecode = await this.getPredicateBytecode(oldState.predicate)
        const valid = await validStateTransition(
          oldState.encoded,
          newState.encoded,
          witness,
          bytecode
        )

        // State object is invalid if any transition fails.
        if (!valid) {
          throw new Error('Invalid state transition')
        }
      }

      // Apply the transaction to local state.
      state.applyStateObject(newState)
    }

    // Check that the transaction is in the verified state.
    const validTransaction = state.hasStateObject(tx.newState)
    if (!validTransaction) {
      throw new Error('Invalid transaction')
    }

    return state
  }

  private async getPredicateBytecode(address: string): Promise<string> {
    // Try to pull from cache first.
    if (address in this.predicates) {
      return this.predicates[address]
    }

    let bytecode
    try {
      bytecode = await this.chaindb.getPredicateBytecode(address)
    } catch {
      // Don't have the bytecode stored, pull it and store it.
      bytecode = await this.eth.getContractBytecode(address)
      await this.chaindb.setPredicateBytecode(address, bytecode)
    }

    // Cache the bytecode for later.
    this.predicates[address] = bytecode

    return bytecode
  }

  /**
   * Checks whether a transaction's inclusion proof is valid.
   * @param transaction The transaction to check.
   * @returns `true` if the inclusion proof is valid, `false` otherwise.
   */
  private async checkInclusionProof(
    transaction: Transaction
  ): Promise<{ start: BigNum; end: BigNum }> {
    let root = await this.chaindb.getBlockHeader(transaction.block.toNumber())
    if (root === null) {
      throw new Error(
        `Received transaction for non-existent block #${transaction.block}`
      )
    }

    root = root + 'ffffffffffffffffffffffffffffffff'

    // Return the result of the inclusion proof check.
    const tree = new MerkleSumTree()
    return tree.verify(
      {
        end: transaction.newState.end,
        data: transaction.newState.encoded,
      },
      0,
      transaction.inclusionProof,
      root
    )
  }
}
