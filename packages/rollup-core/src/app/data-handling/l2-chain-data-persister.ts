/* External Imports */
import { DB } from '@eth-optimism/core-db'
import { getLogger, Logger } from '@eth-optimism/core-utils'

import {
  Block,
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from 'ethers/providers'

/* Internal Imports */
import {
  L2DataService,
  LogHandlerContext,
  TransactionAndRoot,
} from '../../types'
import { ChainDataProcessor } from './chain-data-processor'
import { monkeyPatchL2Provider } from '../utils'
import { BigNumber, remove0x } from '@eth-optimism/core-utils/build'

const log: Logger = getLogger('l2-chain-data-persister')

/**
 * This class subscribes to and syncs L2, processing all transactions
 * so that it may be more easily accessed in the context of Optimistic Rollup.
 */
export class L2ChainDataPersister extends ChainDataProcessor {
  public static readonly persistenceKey = 'L2ChainDataPersister'

  /**
   * Creates a L2ChainDataPersister that subscribes to blocks, processes all
   * transactions, and inserts relevant data into the provided RDB.
   *
   * @param db The DB to use to persist the queue of Block objects.
   * @param dataService The L2 Data Service handling persistence of relevant data.
   * @param l2Provider The provider to use to connect to L2 to subscribe & fetch block / tx data.
   * @param persistenceKey The persistence key to use for this instance within the provided DB.
   */
  public static async create(
    db: DB,
    dataService: L2DataService,
    l2Provider: Provider,
    persistenceKey: string = L2ChainDataPersister.persistenceKey
  ): Promise<L2ChainDataPersister> {
    const processor = new L2ChainDataPersister(
      db,
      dataService,
      monkeyPatchL2Provider(l2Provider),
      persistenceKey
    )
    await processor.init()
    return processor
  }

  private constructor(
    db: DB,
    private readonly l2DataService: L2DataService,
    private readonly l2Provider: Provider,
    persistenceKey: string
  ) {
    super(db, persistenceKey)
  }

  /**
   * @inheritDoc
   */
  protected async handleNextItem(index: number, block: Block): Promise<void> {
    log.debug(`handling block ${block.number}.`)

    if (!block.transactions || !block.transactions.length) {
      log.error(`Received L2 block #${index} with 0 transactions!`)
      return this.markProcessed(index)
    }

    if (block.transactions.length > 1) {
      log.error(
        `Received ${block.transactions.length} transactions for block #${block.number}`
      )
    }

    const txHashes: string[] = block.transactions.map((x) =>
      typeof x !== 'string' ? x['hash'] : x
    )

    const txs: any[] = await Promise.all([
      ...txHashes.map(
        (hash) => this.l2Provider.getTransaction(hash) as Promise<any>
      ),
      ...txHashes.map((hash) => this.l2Provider.getTransactionReceipt(hash)),
    ])

    for (let i = 0; i < block.transactions.length; i++) {
      const txAndRoot: TransactionAndRoot = L2ChainDataPersister.getTransactionAndRoot(
        block,
        txs[i],
        txs[i + block.transactions.length]
      )
      await this.l2DataService.insertL2Transaction(txAndRoot)
    }

    return this.markProcessed(index)
  }

  /**
   * TransactionResponse and TransactionReceipt don't return all the info needed from a Tx, so
   * this function takes in one of each and outputs the full tx data.
   *
   * @param block The Block object.
   * @param response The TransactionResponse object.
   * @param receipt The TransactionReceipt object.
   * @returns The combined TransactionAndRoot object.
   */
  public static getTransactionAndRoot(
    block: Block,
    response: TransactionResponse,
    receipt: TransactionReceipt
  ): TransactionAndRoot {
    const res: TransactionAndRoot = {
      timestamp: block.timestamp,
      blockNumber: receipt.blockNumber,
      transactionIndex: receipt.transactionIndex,
      transactionHash: receipt.transactionHash,
      to: receipt.to,
      nonce: response.nonce,
      calldata: response.data,
      from: response.from || receipt.from,
      stateRoot: receipt.root,
    }

    if (!!response['l1MessageSender']) {
      res.l1MessageSender = response['l1MessageSender']
    }
    if (!!response.r && !!response.s && response.v !== undefined) {
      res.signature = `${response.r}${remove0x(
        response.s
      )}${response.v.toString(16)}`
    }

    return res
  }
}
