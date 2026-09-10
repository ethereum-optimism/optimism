import 'dotenv/config'
import { vi } from 'vitest'
import './types.d.ts'

// Mock environment variables for tests
process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test_db'
process.env.DATABASE_SCHEMA = 'test_schema'
process.env.GAS_TANK_CONTRACT_ADDRESS = '0x1234567890123456789012345678901234567890'

// Global test helpers
globalThis.mockEventLog = (overrides = {}) => ({
  logIndex: 0,
  transactionHash: '0xabcdef1234567890',
  blockNumber: 1000n,
  topics: ['0x1234567890abcdef'],
  data: '0x0000000000000000000000000000000000000000000000000000000000000020',
  ...overrides,
} as any)

globalThis.mockTransaction = (overrides = {}) => ({
  hash: '0xabcdef1234567890',
  from: '0x1234567890123456789012345678901234567890',
  ...overrides,
} as any)

globalThis.mockBlock = (overrides = {}) => ({
  number: 1000n,
  timestamp: 1700000000n,
  ...overrides,
} as any)

// Suppress console warnings during tests unless explicitly needed
const originalWarn = console.warn
console.warn = vi.fn()

// Restore console.warn for specific tests that need it
globalThis.restoreConsoleWarn = () => {
  console.warn = originalWarn
}