import type { TransactionReceipt } from 'viem'
import { getWithdrawals } from 'viem/op-stack'

export const useWithdrawalMessage = (receipt?: TransactionReceipt) => {
  return receipt ? getWithdrawals(receipt)[0] : undefined
}
