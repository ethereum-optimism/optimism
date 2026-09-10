import * as fs from 'node:fs'
import * as path from 'node:path'

import Database from 'better-sqlite3'

const SCHEMA = `
CREATE TABLE IF NOT EXISTS budget_consumed (
  address      TEXT PRIMARY KEY NOT NULL,
  consumed_wei TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS budget_blocked (
  message_hash TEXT PRIMARY KEY NOT NULL,
  tx_origin    TEXT NOT NULL,
  blocked_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_budget_blocked_origin
  ON budget_blocked (tx_origin);
`

/**
 * SQLite-backed store for relay-fund deposits and per-message blocked state.
 * Each write is durable on return (WAL + synchronous=NORMAL).
 */
export class RelayFundsDeposited {
  private readonly db: Database.Database
  private readonly upsertConsumed: Database.Statement
  private readonly getConsumedStmt: Database.Statement
  private readonly insertBlocked: Database.Statement
  private readonly listBlockedForOrigin: Database.Statement
  private readonly deleteBlocked: Database.Statement

  constructor(dbPath: string) {
    if (dbPath !== ':memory:') {
      const dir = path.dirname(dbPath)
      if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    }

    this.db = new Database(dbPath)
    this.db.pragma('journal_mode = WAL')
    this.db.pragma('synchronous = NORMAL')
    this.db.exec(SCHEMA)

    this.upsertConsumed = this.db.prepare(`
      INSERT INTO budget_consumed (address, consumed_wei) VALUES (?, ?)
      ON CONFLICT(address) DO UPDATE SET consumed_wei = excluded.consumed_wei
    `)
    this.getConsumedStmt = this.db.prepare(
      `SELECT consumed_wei FROM budget_consumed WHERE address = ?`,
    )
    this.insertBlocked = this.db.prepare(`
      INSERT INTO budget_blocked (message_hash, tx_origin, blocked_at)
      VALUES (?, ?, ?)
      ON CONFLICT(message_hash) DO NOTHING
    `)
    this.listBlockedForOrigin = this.db.prepare(
      `SELECT message_hash FROM budget_blocked WHERE tx_origin = ?`,
    )
    this.deleteBlocked = this.db.prepare(
      `DELETE FROM budget_blocked WHERE message_hash = ?`,
    )
  }

  recordConsumption(address: string, gasUsed: bigint): void {
    const key = address.toLowerCase()
    const current = this.getConsumed(key)
    this.upsertConsumed.run(key, (current + gasUsed).toString())
  }

  getConsumed(address: string): bigint {
    const row = this.getConsumedStmt.get(address.toLowerCase()) as
      | { consumed_wei: string }
      | undefined
    return row ? BigInt(row.consumed_wei) : 0n
  }

  getRemainingBudget(address: string, totalDeposits: bigint): bigint {
    const remaining = totalDeposits - this.getConsumed(address)
    return remaining > 0n ? remaining : 0n
  }

  hasEnoughBudget(
    address: string,
    totalDeposits: bigint,
    estimatedGasCost: bigint,
  ): boolean {
    return this.getRemainingBudget(address, totalDeposits) >= estimatedGasCost
  }

  markBlocked(messageHash: string, txOrigin: string): void {
    this.insertBlocked.run(
      messageHash.toLowerCase(),
      txOrigin.toLowerCase(),
      Date.now(),
    )
  }

  getBlockedForOrigin(txOrigin: string): string[] {
    const rows = this.listBlockedForOrigin.all(
      txOrigin.toLowerCase(),
    ) as Array<{ message_hash: string }>
    return rows.map((r) => r.message_hash)
  }

  clearBlocked(messageHash: string): void {
    this.deleteBlocked.run(messageHash.toLowerCase())
  }

  close(): void {
    this.db.close()
  }
}
