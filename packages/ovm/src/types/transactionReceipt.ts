import { TransactionReceipt } from 'ethers/providers'

export interface OvmTransactionReceipt extends TransactionReceipt {
  revertMessage?: string
}
