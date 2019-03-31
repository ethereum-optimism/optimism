import { Transaction } from '@pigi/utils'
import { Deposit } from './deposit'

export interface TransactionProof {
  tx: Transaction
  deposits: Deposit[]
  transactions: Transaction[]
}
