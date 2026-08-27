import type { Hash } from 'viem'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type TransactionEntry = {
  chainId: number
  hash: Hash
}

type TransactionStore = {
  transactionEntryByHash: Record<Hash, TransactionEntry>

  addTransaction: (entry: TransactionEntry) => void
}

export const useTransactionStore = create<TransactionStore>()(
  persist(
    (set) => ({
      transactionEntryByHash: {},

      addTransaction: (entry: TransactionEntry) => {
        set((state) => ({
          transactionEntryByHash: {
            [entry.hash]: entry,
            ...state.transactionEntryByHash,
          },
        }))
      },
    }),
    {
      name: 'superchain-tools-transaction-store',
    },
  ),
)
