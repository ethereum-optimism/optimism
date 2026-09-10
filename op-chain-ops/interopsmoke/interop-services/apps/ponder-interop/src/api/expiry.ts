/**
 * Interop message expiry helpers for the /messages/pending API surface.
 *
 * A SentMessage whose source-log timestamp is older than (now - WINDOW + MARGIN)
 * seconds is dropped from /messages/pending — the op-supervisor will reject any
 * executing message that references a source log older than its window, so
 * surfacing them as "pending" just churns relayer retries. MARGIN shrinks the
 * relayable window so the relayer doesn't pick up edge-of-window messages.
 *
 * Per-network values come from the orchestrator manifest via render.mjs;
 * defaults match supersim's ~1h cutoff.
 */

const DEFAULT_EXPIRY_WINDOW_SECONDS = 3600
const DEFAULT_EXPIRY_SAFETY_MARGIN_SECONDS = 60

export function parseExpiryEnv(
  raw: string | undefined,
  fallback: number,
): number {
  if (!raw) return fallback
  const n = parseInt(raw, 10)
  return Number.isFinite(n) && n >= 0 ? n : fallback
}

export const EXPIRY_WINDOW_SECONDS = parseExpiryEnv(
  process.env.INTEROP_EXPIRY_WINDOW_SECONDS,
  DEFAULT_EXPIRY_WINDOW_SECONDS,
)

export const EXPIRY_SAFETY_MARGIN_SECONDS = parseExpiryEnv(
  process.env.INTEROP_EXPIRY_SAFETY_MARGIN_SECONDS,
  DEFAULT_EXPIRY_SAFETY_MARGIN_SECONDS,
)

/**
 * Cutoff timestamp (seconds since epoch). SentMessages with timestamp <= this
 * are considered expired. Computed per call so it tracks wall-clock as it
 * advances. Parameters are exposed for unit testing — production callers
 * should rely on the defaults.
 */
export function expiryCutoffSeconds(
  windowSeconds: number = EXPIRY_WINDOW_SECONDS,
  marginSeconds: number = EXPIRY_SAFETY_MARGIN_SECONDS,
  nowMs: number = Date.now(),
): bigint {
  const nowSec = Math.floor(nowMs / 1000)
  return BigInt(nowSec - windowSeconds + marginSeconds)
}
