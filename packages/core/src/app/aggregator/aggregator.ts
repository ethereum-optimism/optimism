import { Aggregator } from '../../types/aggregator'
import { StateManager } from '../../types/ovm'
import {
  BlockTransaction,
  BlockTransactionCommitment,
  Transaction,
  TransactionResult,
} from '../../types/serialization'
import { BigNumber, doRangesSpanRange } from '../utils'
import { BlockManager } from '../../types/block-production'
import { SignatureProvider } from '../../types/keystore'
import { serializeObject } from '../serialization'

export class DefaultAggregator implements Aggregator {
  public constructor(
    private readonly stateManager: StateManager,
    private readonly blockManager: BlockManager,
    private readonly publicKey: string,
    private readonly signatureProvider: SignatureProvider
  ) {}

  public async ingestTransaction(
    transaction: Transaction
  ): Promise<BlockTransactionCommitment> {
    const blockNumber: BigNumber = await this.blockManager.getNextBlockNumber()

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

    const serializedTransaction: string = serializeObject(blockTransaction)

    return {
      blockTransaction,
      witness: await this.signatureProvider.sign(
        this.publicKey,
        serializedTransaction
      ),
    }
  }

  public async getPublicKey(): Promise<any> {
    return this.publicKey
  }
}
