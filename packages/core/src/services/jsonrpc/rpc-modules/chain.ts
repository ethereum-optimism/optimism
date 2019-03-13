/* External Imports */
import { Service } from '@nestd/core'
import { Transaction } from '@pigi/utils'

/* Services */
import { LoggerService, SyncLogger } from '../../logging'
import { ChainDB } from '../../db/interfaces/chain-db'
import { EthService } from '../../eth/eth.service'
import { ContractService } from '../../eth/contract.service'

/* Internal Imports */
import { BaseRpcModule } from './base-rpc-module'
import { Exit } from '../../../models/chain'

/**
 * Subdispatcher that handles chain-related requests.
 */
@Service()
export class ChainRpcModule extends BaseRpcModule {
  public readonly prefix = 'pg_'
  private readonly logger = new SyncLogger('ChainRpcModule', this.logs)

  constructor(
    private readonly logs: LoggerService,
    private readonly chaindb: ChainDB,
    private readonly eth: EthService,
    private readonly contract: ContractService
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
    const exits = await this.getExits(address)
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
   * Queries all exits for a given address.
   * @param address Address to query.
   * @returns all exits for that address.
   */
  public async getExits(address: string): Promise<Exit[]> {
    const exits = await this.chaindb.getExits(address)

    const currentBlock = await this.eth.getCurrentBlock()
    // const challengePeriod = await this.contract.getChallengePeriod()
    const challengePeriod = 20

    for (const exit of exits) {
      exit.completed = exit.block.addn(challengePeriod).ltn(currentBlock)
      exit.finalized = await this.chaindb.checkFinalized(exit)
    }

    return exits
  }
}
