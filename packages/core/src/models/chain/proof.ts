import { Transaction } from 'plasma-utils'
import { Deposit } from './deposit'

export interface TransactionProof {
  tx: Transaction
  deposits: Deposit[]
  transactions: Transaction[]
}
