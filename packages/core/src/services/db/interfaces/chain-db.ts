/* External Imports */
import { Service, OnStart } from '@nestd/core'
import BigNum from 'bn.js'
import { Transaction, StateObject } from '@pigi/utils'

/* Services */
import { ContractService } from '../../eth/contract.service'
import { DBService } from '../db.service'

/* Internal Imports */
import { Block, Exit, ExitArgs } from '../../../models/chain'
import { Deposit } from '../../../models/chain/deposit'
import { BaseDBProvider } from '../backends/base-db.provider'
import { StateManager } from '../../../utils'

/**
 * Service that exposes an interface to chain-related
 * database calls.
 */
@Service()
export class ChainDB implements OnStart {
  constructor(
    private readonly contract: ContractService,
    private readonly dbservice: DBService
  ) {}

  /**
   * @returns the current db instance.
   */
  get db(): BaseDBProvider {
    const db = this.dbservice.dbs.chain
    if (db === undefined) {
      throw new Error('ChainDB is not yet initialized.')
    }
    return db
  }

  public async onStart(): Promise<void> {
    const address = await this.contract.waitForAddress()
    await this.dbservice.open({ namespace: 'chain', id: address })
  }

  /**
   * Queries a transaction.
   * @param hash Hash of the transaction.
   * @returns the transaction object.
   */
  public async getTransaction(hash: string): Promise<Transaction> {
    const encoded = await this.db.get(`transaction:${hash}`, undefined)
    if (encoded === undefined) {
      throw new Error('Transaction not found in database.')
    }
    return Transaction.from(encoded as string)
  }

  /**
   * Adds a transaction to the database.
   * @param transaction Transaction to store.
   */
  public async setTransaction(transaction: Transaction): Promise<void> {
    await this.db.set(`transaction:${transaction.hash}`, transaction.encoded)
  }

  /**
   * Checks if the chain has stored a specific transaction already.
   * @param hash The transaction hash.
   * @returns `true` if the chain has stored the transaction, `false` otherwise.
   */
  public async hasTransaction(hash: string): Promise<boolean> {
    return this.db.exists(`transaction:${hash}`)
  }

  /**
   * Returns the number of the last known block.
   * @returns the latest block.
   */
  public async getLatestBlock(): Promise<number> {
    return (await this.db.get('latestblock', -1)) as number
  }

  /**
   * Sets the latest block.
   * @param block A block number.
   */
  public async setLatestBlock(block: number): Promise<void> {
    await this.db.set('latestblock', block)
  }

  /**
   * Queries a block header by number.
   * @param block Number of the block to query.
   * @returns the hash of the specified block.
   */
  public async getBlockHeader(block: number): Promise<string | null> {
    return (await this.db.get(`header:${block}`, null)) as string | null
  }

  /**
   * Adds a block header to the database.
   * @param block Number of the block to add.
   * @param hash Hash of the given block.
   */
  public async addBlockHeader(block: number, hash: string): Promise<void> {
    await this.setLatestBlock(block)
    await this.db.set(`header:${block}`, hash)
  }

  /**
   * Adds multiple block headers to the database.
   * @param blocks An array of block objects.
   */
  public async addBlockHeaders(blocks: Block[]): Promise<void> {
    // Set the latest block.
    const latest = blocks.reduce((a, b) => {
      return a.number > b.number ? a : b
    })
    await this.setLatestBlock(latest.number)

    const objects = blocks.map((block) => {
      return { key: `header:${block.number}`, value: block.hash }
    })
    await this.db.bulkPut(objects)
  }

  /**
   * Returns a list of known deposits for an address.
   * @param address Address to query.
   * @returns a list of known deposits.
   */
  public async getDeposits(address: string): Promise<Deposit[]> {
    const deposits = (await this.db.get(`deposits:${address}`, [])) as Deposit[]
    return deposits.map((deposit) => {
      return new Deposit(deposit)
    })
  }

  /**
   * Returns the list of known exits for an address.
   * @param address Address to query.
   * @returns a list of known exits.
   */
  public async getExits(address: string): Promise<Exit[]> {
    const exits = (await this.db.get(`exits:${address}`, [])) as ExitArgs[]
    return exits.map((exit) => {
      return new Exit(exit)
    })
  }

  /**
   * Adds an exit to the database.
   * @param exit Exit to add to database.
   */
  public async addExits(exits: Exit[]): Promise<void> {
    const objects = exits.map((exit) => {
      return { key: `exits:${exit}`, value: exit }
    })
    await this.markExited(exit)
    await this.db.push(`exits:${exit.owner}`, exit)
  }

  /**
   * Adds an "exitable end" to the database.
   * For more information, see:
   * https://github.com/plasma-group/plasma-contracts/issues/44.
   * @param end End of the range.
   */
  public async addExitableEnd(end: BigNum): Promise<void> {
    await this.addExitableEnds([end])
  }

  /**
   * Adds multiple "exitable ends" to the database in bulk.
   * For more information, see:
   * https://github.com/plasma-group/plasma-contracts/issues/44.
   * @param exitable Ends to add to the database.
   */
  public async addExitableEnds(ends: BigNum[]): Promise<void> {
    const objects = ends.map((end) => {
      return { key: `exitable:${end}`, value: end.toString('hex') }
    })

    await this.db.bulkPut(objects)
  }

  /**
   * Returns the correct exitable end for a range.
   * @param end End of the range.
   * @returns the exitable end.
   */
  public async getExitableEnd(end: BigNum): Promise<BigNum> {
    const nextKey = await this.db.findNextKey(`exitable:${end}`)
    const exitableEnd = (await this.db.get(nextKey)) as string
    return new BigNum(exitableEnd, 'hex')
  }

  /**
   * Marks a range as exited.
   * @param range Range to mark.
   */
  public async markExited(range: {
    start: BigNum
    end: BigNum
  }): Promise<void> {
    await this.db.set(`exited:${range.start}:${range.end}`, true)
  }

  /**
   * Checks if a range is marked as exited.
   * @param range Range to check.
   * @returns `true` if the range is exited, `false` otherwise.
   */
  public async checkExited(range: {
    start: BigNum
    end: BigNum
  }): Promise<boolean> {
    return (await this.db.get(
      `exited:${range.start}:${range.end}`,
      false
    )) as boolean
  }

  /**
   * Marks an exit as finalized.
   * @param exit Exit to mark.
   */
  public async markFinalized(exit: {
    start: BigNum
    end: BigNum
  }): Promise<void> {
    await this.db.set(`finalized:${exit.start}:${exit.end}`, true)
  }

  /**
   * Checks if an exit is marked as finalized.
   * @param exit Exit to check.
   * @returns `true` if the exit is finalized, `false` otherwise.
   */
  public async checkFinalized(exit: {
    start: BigNum
    end: BigNum
  }): Promise<boolean> {
    return (await this.db.get(
      `finalized:${exit.start}:${exit.end}`,
      false
    )) as boolean
  }

  /**
   * Returns the latest state.
   * @returns a list of snapshots.
   */
  public async getState(): Promise<StateManager> {
    const snapshots = (await this.db.get(`state:latest`, [])) as StateObject[]
    const state = snapshots.map((snapshot) => {
      return new StateObject(snapshot)
    })
    return new StateManager(state)
  }

  /**
   * Sets the latest state.
   * @param state A list of snapshots.
   */
  public async setState(stateManager: StateManager): Promise<void> {
    await this.db.set('state:latest', stateManager.state)
  }

  /**
   * Returns the bytecode of a given predicate.
   * @param address Address of the predicate.
   * @returns the predicate bytecode.
   */
  public async getPredicateBytecode(address: string): Promise<string> {
    const bytecode = await this.db.get(`predicate:${address}`, undefined)
    if (bytecode === undefined) {
      throw new Error('Predicate not found in database.')
    }

    return bytecode as string
  }

  /**
   * Sets the bytecode for a given predicate.
   * @param address Address of the predicate.
   * @param bytecode Bytecode of the predicate.
   */
  public async setPredicateBytecode(
    address: string,
    bytecode: string
  ): Promise<void> {
    await this.db.set(`predicate:${address}`, bytecode)
  }
}
