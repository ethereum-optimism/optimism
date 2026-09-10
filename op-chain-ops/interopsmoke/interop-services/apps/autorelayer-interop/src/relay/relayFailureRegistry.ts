import * as fs from 'node:fs'
import * as path from 'node:path'

import Database from 'better-sqlite3'

const SCHEMA = `
CREATE TABLE IF NOT EXISTS relay_failures (
  message_hash      TEXT PRIMARY KEY NOT NULL,
  source            INTEGER NOT NULL,
  destination       INTEGER NOT NULL,
  count             INTEGER NOT NULL,
  first_failed_at   INTEGER NOT NULL,
  last_failed_at    INTEGER NOT NULL,
  last_reason       TEXT NOT NULL,
  permanent         INTEGER NOT NULL DEFAULT 0,
  permanent_reason  TEXT
);

CREATE INDEX IF NOT EXISTS idx_relay_failures_route
  ON relay_failures (source, destination);

CREATE INDEX IF NOT EXISTS idx_relay_failures_permanent
  ON relay_failures (permanent);
`

/**
 * Reasons that get marked permanent on first failure. Once a reason is here,
 * the message is never retried regardless of count. Reasons not in this set
 * fall back to count-based promotion (after MAX_FAILURES, marked permanent).
 *
 * `rpc_rejected` covers op-supervisor refusals — most commonly because the
 * source log is past the interop expiry window. By the time the relayer
 * sees these (i.e. ponder-interop's server-side filter missed it because the
 * message expired in the broadcast window), retrying is useless.
 */
const PERMANENT_REASONS: ReadonlySet<string> = new Set([
  'rpc_rejected',
  'expired',
])

const MAX_FAILURES = 10
const BACKOFF_BASE_MS = 5_000
const BACKOFF_MAX_MS = 60 * 60_000

/**
 * Exponential backoff schedule. count=1 → 5s, count=2 → 10s, ..., capped at
 * BACKOFF_MAX_MS. The schedule's floor is well above a typical cycle interval
 * (loopIntervalMs ≥ 2s in config) so a fail-then-retry on consecutive cycles
 * is naturally suppressed.
 */
export function backoffMs(count: number): number {
  if (count <= 0) return 0
  const ms = BACKOFF_BASE_MS * 2 ** (count - 1)
  return Math.min(ms, BACKOFF_MAX_MS)
}

export interface PermanentEntry {
  messageHash: string
  source: number
  destination: number
  reason: string
  count: number
}

export interface RegistryRouteStat {
  source: number
  destination: number
  reason: string
  count: number
  oldestLastFailedAt: number
}

export type RelayStatus = 'ready' | 'in_backoff' | 'abandoned'

/**
 * SQLite-backed durable store for per-message relay failure history.
 *
 * Replaces the in-process Set<string> previously held in BaseRelayModule. Two
 * primary uses by the circuit breaker:
 *
 * 1. shouldAttempt(hash, now) — gate that consults backoff + permanent flag.
 * 2. gc(stillPendingHashes) — drop entries that ponder-interop no longer
 * reports as pending (relayed elsewhere or expired server-side).
 *
 * Schema lives in a dedicated file (default `${cwd}/.relay-failure-registry.sqlite`,
 * override via RELAY_FAILURE_REGISTRY_DB_PATH) — separate from RelayFundsDeposited's
 * file because the failure registry has a different operational lifecycle (frequent
 * GC vs stable deposit state) and recovery story.
 */
export class RelayFailureRegistry {
  private readonly db: Database.Database
  private readonly upsertFailure: Database.Statement
  private readonly getEntry: Database.Statement
  private readonly hasEntry: Database.Statement
  private readonly clearOne: Database.Statement
  private readonly listPermanent: Database.Statement
  private readonly statsByRoute: Database.Statement

  constructor(dbPath: string) {
    if (dbPath !== ':memory:') {
      const dir = path.dirname(dbPath)
      if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    }

    this.db = new Database(dbPath)
    this.db.pragma('journal_mode = WAL')
    this.db.pragma('synchronous = NORMAL')
    this.db.exec(SCHEMA)

    this.upsertFailure = this.db.prepare(`
      INSERT INTO relay_failures (
        message_hash, source, destination, count,
        first_failed_at, last_failed_at, last_reason,
        permanent, permanent_reason
      ) VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?)
      ON CONFLICT(message_hash) DO UPDATE SET
        count = count + 1,
        last_failed_at = excluded.last_failed_at,
        last_reason = excluded.last_reason,
        permanent = CASE
          WHEN permanent = 1 THEN 1
          WHEN excluded.permanent = 1 THEN 1
          WHEN count + 1 >= ${MAX_FAILURES} THEN 1
          ELSE 0
        END,
        permanent_reason = CASE
          WHEN permanent = 1 THEN permanent_reason
          WHEN excluded.permanent = 1 THEN excluded.permanent_reason
          ELSE permanent_reason
        END
    `)

    this.getEntry = this.db.prepare(
      `SELECT count, last_failed_at, permanent, permanent_reason
       FROM relay_failures WHERE message_hash = ?`,
    )

    this.hasEntry = this.db.prepare(
      `SELECT 1 FROM relay_failures WHERE message_hash = ? LIMIT 1`,
    )

    this.clearOne = this.db.prepare(
      `DELETE FROM relay_failures WHERE message_hash = ?`,
    )

    this.listPermanent = this.db.prepare(
      `SELECT message_hash, source, destination, last_reason,
              permanent_reason, count
       FROM relay_failures WHERE permanent = 1`,
    )

    this.statsByRoute = this.db.prepare(
      `SELECT source,
              destination,
              COALESCE(permanent_reason, last_reason) AS reason,
              COUNT(*) AS count,
              MIN(last_failed_at) AS oldest_last_failed_at
       FROM relay_failures
       WHERE permanent = 1
       GROUP BY source, destination, reason`,
    )
  }

  /**
   * Records a failure. Increments count, updates timestamp/reason, and trips
   * permanent if the reason is in PERMANENT_REASONS or count reaches MAX_FAILURES.
   * `now` is exposed for testability; production callers pass Date.now().
   */
  recordFailure(
    messageHash: string,
    source: number,
    destination: number,
    reason: string,
    now: number = Date.now(),
  ): void {
    const isPermanentReason = PERMANENT_REASONS.has(reason)
    this.upsertFailure.run(
      messageHash.toLowerCase(),
      source,
      destination,
      now,
      now,
      reason,
      isPermanentReason ? 1 : 0,
      isPermanentReason ? reason : null,
    )
  }

  /**
   * Whether any failure has been recorded for this hash. Used as a "this is a
   * retry" signal for metric labels — distinct from shouldAttempt, which gates
   * actually attempting the message.
   */
  hasFailed(messageHash: string): boolean {
    return this.hasEntry.get(messageHash.toLowerCase()) !== undefined
  }

  /**
   * Classify a hash's relay eligibility:
   *
   * - 'abandoned'  — entry is marked permanent; will never relay.
   * - 'in_backoff' — entry is within its backoff window (last_failed_at + backoff > now).
   * - 'ready'      — no entry, or the backoff has elapsed.
   *
   * Replaces the boolean shouldAttempt() for call sites that want to attribute
   * skips to the specific cause; shouldAttempt() is kept as a convenience
   * derived from this.
   */
  statusFor(messageHash: string, now: number = Date.now()): RelayStatus {
    const row = this.getEntry.get(messageHash.toLowerCase()) as
      | {
          count: number
          last_failed_at: number
          permanent: number
        }
      | undefined
    if (!row) return 'ready'
    if (row.permanent === 1) return 'abandoned'
    if (now - row.last_failed_at < backoffMs(row.count)) return 'in_backoff'
    return 'ready'
  }

  /**
   * Whether the relayer should attempt this message right now. False if the
   * entry is marked permanent or within its backoff window.
   */
  shouldAttempt(messageHash: string, now: number = Date.now()): boolean {
    return this.statusFor(messageHash, now) === 'ready'
  }

  /**
   * Drop the entry. Called on a successful relay so a future re-emission of the
   * same hash (extremely unlikely; mostly a tidy-up) is treated as fresh.
   */
  clearFailure(messageHash: string): void {
    this.clearOne.run(messageHash.toLowerCase())
  }

  /**
   * Drop entries whose hash is not in the still-pending set. Called from
   * pruneAndEmitInFlight so the book auto-shrinks as ponder-interop stops
   * reporting messages as pending (relayed via another path, or — post the
   * ponder expiry filter — filtered server-side).
   *
   * SQLite has a ~999 parameter limit per statement (SQLITE_LIMIT_VARIABLE_NUMBER),
   * which the autorelayer's 10k-per-cycle cap can blow through. Implementation
   * uses a temp table so the DELETE is one statement regardless of pending size.
   *
   * Returns the number of rows deleted.
   */
  gc(stillPendingHashes: ReadonlySet<string>): number {
    if (stillPendingHashes.size === 0) {
      // Nothing pending → drop everything.
      const info = this.db.prepare(`DELETE FROM relay_failures`).run()
      return info.changes
    }

    const tx = this.db.transaction((hashes: ReadonlySet<string>) => {
      this.db.exec(
        `CREATE TEMP TABLE IF NOT EXISTS _gc_pending (h TEXT PRIMARY KEY) WITHOUT ROWID`,
      )
      this.db.exec(`DELETE FROM _gc_pending`)
      const ins = this.db.prepare(
        `INSERT OR IGNORE INTO _gc_pending (h) VALUES (?)`,
      )
      for (const h of hashes) ins.run(h.toLowerCase())
      const info = this.db
        .prepare(
          `DELETE FROM relay_failures
           WHERE message_hash NOT IN (SELECT h FROM _gc_pending)`,
        )
        .run()
      return info.changes
    })

    return tx(stillPendingHashes)
  }

  /**
   * Per-route × per-reason aggregates over the permanent entries. Drives the
   * failure-registry observability metrics: `count` populates
   * relayer_module_failure_registry_size; `oldestLastFailedAt` lets the
   * relayer derive oldest_age_seconds = now - oldestLastFailedAt at scrape
   * time, so the dashboard never has to reason about clock skew.
   */
  getStats(): RegistryRouteStat[] {
    const rows = this.statsByRoute.all() as Array<{
      source: number
      destination: number
      reason: string
      count: number
      oldest_last_failed_at: number
    }>
    return rows.map((r) => ({
      source: r.source,
      destination: r.destination,
      reason: r.reason,
      count: r.count,
      oldestLastFailedAt: r.oldest_last_failed_at,
    }))
  }

  getPermanent(): PermanentEntry[] {
    const rows = this.listPermanent.all() as Array<{
      message_hash: string
      source: number
      destination: number
      last_reason: string
      permanent_reason: string | null
      count: number
    }>
    return rows.map((r) => ({
      messageHash: r.message_hash,
      source: r.source,
      destination: r.destination,
      // permanent_reason may be null for count-based promotions (max_failures)
      // — fall back to last_reason in that case.
      reason: r.permanent_reason ?? r.last_reason,
      count: r.count,
    }))
  }

  close(): void {
    this.db.close()
  }
}
