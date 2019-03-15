/* External Imports */
import { Service } from '@nestd/core'
import { Transaction } from '@pigi/utils'

/* Services */
import { ChainDB } from '../../db/interfaces/chain-db'
import { ChainService } from '../../chain.service'

/* Internal Imports */
import { BaseRpcModule } from './base-rpc-module'
import { Exit } from '../../../models/chain'

/**
 * Subdispatcher that handles chain-related requests.
 */
@Service()
export class ChainRpcModule extends BaseRpcModule {
  public readonly prefix = 'pg_'

  constructor(
    private readonly chaindb: ChainDB,
    private readonly chain: ChainService
  ) {
    super()
  }

  /**
   * Queries a block hash by block number.
   * @param block Block number to query.
   * @returns the hash of that block.
   */
  public async getBlockHeader(block: number): Promise<string> {
    return this.chaindb.getBlockHeader(block)
  }

  /**
   * @returns the latest stored plasma block number.
   */
  public async getLastestPlasmaBlock(): Promise<number> {
    return this.chaindb.getLatestBlock()
  }

  /**
   * Queries a transaction by its hash.
   * @param hash Hash of the transaction.
   * @returns the transaction with that hash.
   */
  public async getTransaction(hash: string): Promise<Transaction> {
    return this.chaindb.getTransaction(hash)
  }

  /**
   * Finalizes all exist for a given address.
   * The given address must be unlocked because it's used to
   * make the finalization transactions.
   * @param address Address to finalize exits for.
   * @returns the transaction hashes for each finalization.
   */
  public async finalizeExits(address: string): Promise<string[]> {
    return this.chain.finalizeExits(address)
  }

  /**
   * Queries all exits for a given address.
   * @param address Address to query.
   * @returns all exits for that address.
   */
  public async getExits(address: string): Promise<Exit[]> {
    return this.chain.getExits(address)
  }
}
