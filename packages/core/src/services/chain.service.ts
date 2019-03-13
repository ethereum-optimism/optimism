/* External Imports */
import { Service } from '@nestd/core'
import AsyncLock from 'async-lock'
import { Transaction } from '@pigi/utils'

/* Services */
import { LoggerService, SyncLogger } from './logging'
import { EthService } from './eth/eth.service'
import { ContractService } from './eth/contract.service'
import { OperatorService } from './operator.service'
import { ChainDB } from './db/interfaces/chain-db'
import { ProofVerificationService } from './proof/proof-verification.service'

/* Internal Imports */
import { Exit, TransactionProof, Deposit } from '../models/chain'
import { StateManager } from '../utils'

/**
 * Service that manages core state-related functionality.
 */
@Service()
export class ChainService {
  private lock = new AsyncLock()
  private readonly logger = new SyncLogger('chain', this.logs)

  constructor(
    private readonly logs: LoggerService,
    private readonly eth: EthService,
    private readonly contract: ContractService,
    private readonly operator: OperatorService,
    private readonly chaindb: ChainDB,
    private readonly verifier: ProofVerificationService
  ) {}

  /**
   * Adds deposit records to the database.
   * @param deposits Deposits to add.
   */
  public async addDeposits(deposits: Deposit[]): Promise<void> {
    // Filter out any ranges that have already been exited.
    const isNotExited = await Promise.all(
      deposits.map(async (deposit) => {
        return !(await this.chaindb.checkExited(deposit))
      })
    )
    deposits = deposits.filter((_, i) => isNotExited[i])

    // Add the deposit to the head state.
    await this.lock.acquire('state', async () => {
      const stateManager = await this.loadState()
      for (const deposit of deposits) {
        stateManager.addStateObject(deposit)
      }
      await this.saveState(stateManager)
    })

    // Add exitable ends to database.
    const ends = deposits.map((deposit) => {
      return deposit.end
    })
    await this.chaindb.addExitableEnds(ends)

    for (const deposit of deposits) {
      this.logger.log(
        `Added deposit to database: ${deposit.start}, ${deposit.end}`
      )
    }
  }

  /**
   * Returns the list of known exits for an address
   * along with its status (challenge period completed, exit finalized).
   * This method makes contract calls and is therefore slower than `getExits`.
   * @param address Address to query.
   * @returns a list of known exits.
   */
  public async getExitsWithStatus(address: string): Promise<Exit[]> {
    const exits = await this.chaindb.getExits(address)

    const currentBlock = await this.eth.getCurrentBlock()
    // const challengePeriod = await
    // this.eth.contract.getChallengePeriod()
    const challengePeriod = 20

    for (const exit of exits) {
      exit.completed = exit.block.addn(challengePeriod).ltn(currentBlock)
      exit.finalized = await this.chaindb.checkFinalized(exit)
    }

    return exits
  }

  /**
   * Adds an exit to the database.
   * @param exit Exit to add to database.
   */
  public async addExit(exit: Exit): Promise<void> {
    await this.chaindb.addExit(exit)

    await this.lock.acquire('state', async () => {
      const stateManager = await this.loadState()
      stateManager.addStateObject(exit)
      await this.saveState(stateManager)
    })
  }

  /**
   * Attempts to finalized exits for a user.
   * @param address Address to finalize exits for.
   * @returns the transaction hashes for each finalization.
   */
  public async finalizeExits(address: string): Promise<string[]> {
    const exits = await this.getExitsWithStatus(address)
    const completed = exits.filter((exit) => {
      return exit.completed && !exit.finalized
    })

    const finalized = []
    const finalizedTxHashes = []
    for (const exit of completed) {
      try {
        const exitableEnd = await this.chaindb.getExitableEnd(exit.end)
        const finalizeTx = await this.contract.finalizeExit(
          exit.id.toString(10),
          exitableEnd,
          address
        )
        finalizedTxHashes.push(finalizeTx.transactionHash)
        finalized.push(exit)
      } catch (err) {
        this.logger.error('Could not finalize exit', err)
      }
    }

    return finalizedTxHashes
  }

  /**
   * Adds a new transaction to a history if it's valid.
   * @param transaction A Transaction object.
   * @param deposits A list of deposits for the transaction.
   * @param proof A Proof object.
   */
  public async addTransaction(proof: TransactionProof): Promise<void> {
    const tx = proof.tx

    this.logger.log(`Verifying transaction proof for: ${tx.hash}`)
    let tempManager: StateManager
    try {
      tempManager = await this.verifier.applyProof(proof)
    } catch (err) {
      this.logger.error(
        `Rejecting transaction proof for: ${tx.hash}`,
        err
      )
      throw new Error(`Invalid transaction proof: ${err}`)
    }
    this.logger.log(`Verified transaction proof for: ${tx.hash}`)

    // Merge and save the new head state.
    this.logger.log(`Saving head state for: ${tx.hash}`)
    await this.lock.acquire('state', async () => {
      const stateManager = await this.loadState()
      stateManager.merge(tempManager)
      this.saveState(stateManager)
    })
    this.logger.log(`Saved head state for: ${tx.hash}`)

    // Store the transaction.
    this.logger.log(`Adding transaction to database: ${tx.hash}`)
    await this.chaindb.setTransaction(tx)
    this.logger.log(`Added transaction to database: ${tx.hash}`)
  }

  /**
   * Sends a transaction to the operator.
   * @param transaction A signed transaction.
   * @returns the transaction receipt.
   */
  public async sendTransaction(transaction: Transaction): Promise<string> {
    // TODO: Check that the transaction receipt is valid.
    this.logger.log(
      `Sending transaction to operator: ${transaction.hash}.`
    )
    const receipt = await this.operator.sendTransaction(transaction)
    this.logger.log(
      `Sent transaction to operator: ${transaction.hash}.`
    )

    return receipt
  }

  /**
   * Loads the current head state as a SnapshotManager.
   * @returns Current head state.
   */
  public async loadState(): Promise<StateManager> {
    const state = await this.chaindb.getState()
    return new StateManager(state)
  }

  /**
   * Saves the current head state from a SnapshotManager.
   * @param stateManager A SnapshotManager.
   */
  public async saveState(stateManager: StateManager): Promise<void> {
    const state = stateManager.state
    await this.chaindb.setState(state)
  }
}
