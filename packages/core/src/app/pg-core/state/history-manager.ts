/* Internal Imports */
import {
  Transaction,
  TransactionProof,
  ChainDB,
  HistoryManager,
} from '../../../interfaces'

/**
 * HistoryManager implementation for PG's Plasma Cashflow variant.
 */
export class PGHistoryManager implements HistoryManager {
  /**
   * Creates the manager.
   * @param db ChainDB to interact with.
   */
  constructor(private db: ChainDB) {}

  /**
   * Generates a transaction proof for a given transaction.
   * Assumes that the recipient has access to public information (e.g. plasma
   * blocks).
   * @param transaction Transaction to generate a proof for.
   * @returns the transaction proof.
   */
  public async getTransactionProof(
    transaction: Transaction
  ): Promise<TransactionProof> {
    /**
     * 1. Initialize the empty proof.
     */
    const proof: TransactionProof = []

    /**
     * 2. Figure out which range the transaction acts upon.
     */
    const range = transaction.stateUpdate.id

    /**
     * 3. Find all deposits which intersect with the range.
     */
    const deposits = await this.db.getDeposits(range.start, range.end)
    if (deposits.length === 0) {
      throw new Error('Could not find any valid transaction deposits.')
    }

    /**
     * 4. Add each deposit to the proof. Deposits are public, so we can assume
     * that the recipient can check the validity of these deposits by querying
     * Ethereum. No need to attach any proof of validity for deposits.
     */
    for (const deposit of deposits) {
      proof.push({
        transaction: deposit,
      })
    }

    /**
     * 5. Figure out the range of blocks that might have transactions on our
     * range. Earliest block is based on the earliest deposit, latest block is
     * based on the block in which the transaction was included.
     */
    const latestBlock = transaction.block
    const earliestBlock = deposits.reduce((earliest, deposit) => {
      return deposit.block < earliest ? deposit.block : earliest
    }, deposits[0].block)

    /**
     * 6. Find all transactions that impact our range.
     */
    let transactions = []
    for (let block = earliestBlock; block < latestBlock; block++) {
      // Figure out which transactions overlap with our range.
      const overlapping = await this.db.getTransactions(
        block,
        range.start,
        range.end
      )

      // Add those transactions to the list of impacting transactions.
      transactions = transactions.concat(overlapping)
    }

    /**
     * 7. Add our transaction as the last element in the list of transactions
     * that impact our range.
     */
    transactions.push(transaction)

    /**
     * 8. Get an inclusion proof for each transaction that impacted our range.
     * Add each transaction + inclusion proof combination to our
     * transaction proof. Plasma block contents isn't considered public
     * information because it isn't published to Ethereum. We therefore need to
     * provide an inclusion proof for each transaction that shows it's actually
     * part of the block it's supposed to be in.
     */
    for (const tx of transactions) {
      const inclusionProof = await this.db.getInclusionProof(tx)

      proof.push({
        transaction: tx,
        inclusionProof,
      })
    }

    /**
     * 9. Return the proof!
     */
    return proof
  }
}
