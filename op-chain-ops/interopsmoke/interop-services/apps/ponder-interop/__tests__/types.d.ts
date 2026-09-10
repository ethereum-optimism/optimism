import type { Log, Transaction, Block } from 'viem'

declare global {
  function mockEventLog(overrides?: Partial<Log>): Log
  function mockTransaction(overrides?: Partial<Transaction>): Transaction
  function mockBlock(overrides?: Partial<Block>): Block
  function restoreConsoleWarn(): void
}