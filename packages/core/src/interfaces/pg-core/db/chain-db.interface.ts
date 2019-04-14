/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import { Transaction, InclusionProof } from '../../../interfaces'

export interface ChainDB {
  getTransactions(
    blockNumber: number,
    start: BigNum,
    end: BigNum
  ): Promise<Transaction[]>
  getDeposits(start: BigNum, end: BigNum): Promise<Transaction[]>
  getBlockHash(blockNumber: number): Promise<string>
  getInclusionProof(transaction: Transaction): Promise<InclusionProof>
  addBlockHash(blockNumber: number, blockHash: string): Promise<void>
  addTransaction(transaction: Transaction): Promise<void>
  addDeposit(deposit: Transaction): Promise<void>
}
