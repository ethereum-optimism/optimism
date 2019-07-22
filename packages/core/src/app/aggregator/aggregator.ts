import BigNum = require('bn.js')

import { Aggregator } from '../../types/aggregator'
import { StateManager } from '../../types/ovm'
import {
  BlockTransaction,
  BlockTransactionCommitment,
  Transaction,
  TransactionResult,
} from '../../types/serialization'
import { doRangesSpanRange, sign } from '../utils'
import { BlockManager } from '../../types/block-production'

export class DefaultAggregator implements Aggregator {
  private readonly publicKey: string =
    'TODO: figure out public key storage and access'
  private readonly privateKey: string =
    'TODO: figure out private key storage and access'

  public constructor(
    private readonly stateManager: StateManager,
    private readonly blockManager: BlockManager
  ) {}

  public async ingestTransaction(
    transaction: Transaction
  ): Promise<BlockTransactionCommitment> {
    const blockNumber: BigNum = await this.blockManager.getNextBlockNumber()

    const {
      stateUpdate,
      validRanges,
    }: TransactionResult = await this.stateManager.executeTransaction(
      transaction,
      blockNumber,
      '' // Note: This function call will change, so just using '' so it compiles
    )

    if (!doRangesSpanRange(validRanges, transaction.range)) {
      throw Error(
        `Cannot ingest Transaction that is not valid across its entire range. 
        Valid Ranges: ${JSON.stringify(validRanges)}. 
        Transaction: ${JSON.stringify(transaction)}.`
      )
    }

    await this.blockManager.addPendingStateUpdate(stateUpdate)

    const blockTransaction: BlockTransaction = {
      blockNumber,
      transaction,
    }

    return {
      blockTransaction,
      witness: sign(this.privateKey, blockTransaction),
    }
  }

  public async getPublicKey(): Promise<any> {
    return this.publicKey
  }
}
