import { l2ToL2CrossDomainMessengerAbi } from '@eth-optimism/viem/abis'
import { gasTankAbi } from '@eth-optimism/viem/abis/experimental'
import {
  type Abi,
  BaseError,
  ContractFunctionRevertedError,
  decodeErrorResult,
  type Hex,
  InsufficientFundsError,
  InvalidParamsRpcError,
} from 'viem'

export type RelayStage = 'simulation' | 'broadcast' | 'execution'

export interface DecodedRevert {
  errorName: string
  args: readonly unknown[] | undefined
}

export class RelayError extends Error {
  override readonly name = 'RelayError'
  readonly stage: RelayStage
  readonly reason: string

  constructor(params: {
    stage: RelayStage
    reason: string
    message?: string
    cause?: Error
  }) {
    super(
      params.message ?? `relay failed at ${params.stage}: ${params.reason}`,
      {
        cause: params.cause,
      },
    )
    this.stage = params.stage
    this.reason = params.reason
    Object.setPrototypeOf(this, RelayError.prototype)
  }
}

/**
 * Walks a viem error's cause chain to find a revert, then decodes it against
 * the given ABI. Returns undefined if the error is not a decodable revert.
 */
export function decodeRelayRevert(
  err: unknown,
  abi: Abi,
): DecodedRevert | undefined {
  if (!(err instanceof BaseError)) return undefined

  const reverted = err.walk(
    (e) => e instanceof ContractFunctionRevertedError,
  ) as ContractFunctionRevertedError | null
  if (!reverted) return undefined

  if (reverted.data?.errorName) {
    return {
      errorName: reverted.data.errorName,
      args: reverted.data.args,
    }
  }

  // viem couldn't decode against the call-site ABI (e.g. a nested revert
  // from a contract the outer call delegates to). Depending on viem version
  // the undecoded revert surfaces as `raw` (full revert data) or `signature`
  // (4-byte selector, set when decode fails with AbiErrorSignatureNotFoundError
  // — viem 2.17.x). Try both against the caller-provided ABI. A bare selector
  // only decodes for zero-arg errors, which covers the cases we classify.
  const raw = (reverted as unknown as { raw?: Hex }).raw
  for (const data of [raw, reverted.signature]) {
    if (!data) continue
    try {
      const decoded = decodeErrorResult({ abi, data })
      return { errorName: decoded.errorName, args: decoded.args }
    } catch {
      // fall through to the next candidate
    }
  }

  return undefined
}

export interface RevertClassification {
  errorName: string
  reason: string
}

/**
 * Decodes a viem revert against the given ABI and maps the decoded error name
 * to a relayer reason string. Falls back to `'unknown'` for decoded-but-unmapped
 * error names. Returns `undefined` if the error is not a decodable revert at all.
 */
export function classifyRevert(
  err: unknown,
  abi: Abi,
  reasonMap: Readonly<Record<string, string>>,
): RevertClassification | undefined {
  const decoded = decodeRelayRevert(err, abi)
  if (!decoded) return undefined
  return {
    errorName: decoded.errorName,
    reason: reasonMap[decoded.errorName] ?? 'unknown',
  }
}

/**
 * Known custom errors thrown by the L2ToL2CrossDomainMessenger contract,
 * mapped to relayer reason strings.
 */
export const L2_TO_L2_CDM_ERROR_REASONS: Readonly<Record<string, string>> = {
  MessageAlreadyRelayed: 'already_relayed',
}

/**
 * Detects pre-EVM RPC rejections from op-supervisor / op-geth, returned over
 * JSON-RPC as `-32602 Invalid params` with body `details: "failed tx
 * simulation"`. These are NOT contract reverts — they're emitted before the
 * EVM runs, when the supervisor declines to validate the executing-message
 * access list (e.g. the source log is past the interop expiry window, or the
 * source chain isn't in the destination's dependency set).
 *
 * Returned reason `rpc_rejected` is intentionally coarse: the supersim error
 * doesn't always distinguish "expired" from "supervisor lag" from "unknown
 * source chain." If a more specific signal becomes available upstream,
 * refine here. The umbrella name leaves room to extend this classifier to
 * other RPC-level refusals (e.g. nonce too low, gas price too low) without
 * a follow-up rename.
 */
function isRpcRejection(err: unknown): boolean {
  if (!(err instanceof BaseError)) return false
  const found = err.walk((e) => e instanceof InvalidParamsRpcError)
  if (!found) return false
  // Defense in depth: also confirm the supersim "failed tx simulation"
  // marker, so we don't misclassify a genuinely malformed eth_sendRawTransaction
  // call (which would also surface as InvalidParamsRpcError).
  const message = (found as Error).message ?? ''
  return /failed tx simulation/i.test(message)
}

/**
 * Detects the relayer EOA running out of ETH to cover gas. Surfaces as an
 * `InsufficientFundsError` (-32000 "insufficient funds for gas * price +
 * value") returned by the destination node when it admits the broadcast tx.
 * Returns `relayer_insufficient_funds` so dashboards and alerts can
 * distinguish "we can't pay" from "user can't pay" (which is
 * `insufficient_deposit_bal` upstream).
 */
function isRelayerInsufficientFunds(err: unknown): boolean {
  if (!(err instanceof BaseError)) return false
  return !!err.walk((e) => e instanceof InsufficientFundsError)
}

/**
 * Convenience wrapper: classify a revert specifically against the L2ToL2CDM ABI.
 * Recognizes two pre-EVM error classes that aren't contract reverts and would
 * otherwise show up as `unknown`: relayer-EOA gas exhaustion (priority — caused
 * by us, fixable by topping up) and op-supervisor RPC rejections
 * (`rpc_rejected` — permanent; expired source log or unknown source chain).
 */
export function classifyL2ToL2CdmRevert(
  err: unknown,
): RevertClassification | undefined {
  if (isRelayerInsufficientFunds(err)) {
    return {
      errorName: 'InsufficientFunds',
      reason: 'relayer_insufficient_funds',
    }
  }
  if (isRpcRejection(err)) {
    return {
      errorName: 'RpcRejected',
      reason: 'rpc_rejected',
    }
  }
  return classifyRevert(
    err,
    l2ToL2CrossDomainMessengerAbi,
    L2_TO_L2_CDM_ERROR_REASONS,
  )
}

/**
 * Known custom errors thrown by the GasTank contract, mapped to relayer
 * reason strings. Names follow the gasTankAbi the relayer simulates against.
 * All are count-based (retry with backoff, abandon at MAX_FAILURES) — none
 * are in the registry's PERMANENT_REASONS set, because each is at least
 * plausibly recoverable (a gas provider can top up, a pending withdrawal can
 * finalize). Supervisor refusals are handled separately via isRpcRejection.
 */
export const GAS_TANK_ERROR_REASONS: Readonly<Record<string, string>> = {
  AlreadyClaimed: 'already_claimed',
  InsufficientBalance: 'insufficient_gas_tank_balance',
  InvalidLength: 'invalid_length',
  InvalidOrigin: 'invalid_origin',
  InvalidPayer: 'invalid_payer',
  InvalidPayload: 'invalid_payload',
  InvalidRootMessage: 'invalid_root_message',
  MaxDepositExceeded: 'max_deposit_exceeded',
  WithdrawPending: 'withdraw_pending',
}

/**
 * GasTank.relayMessage calls MESSENGER.relayMessage internally, so reverts
 * from the L2ToL2CDM (e.g. MessageAlreadyRelayed) bubble up through gas-tank
 * call sites as raw data that gasTankAbi alone can't decode. Classify
 * against the union of both ABIs with a merged reason map.
 */
const GAS_TANK_CLASSIFY_ABI: Abi = [
  ...gasTankAbi,
  ...l2ToL2CrossDomainMessengerAbi,
] as Abi

const GAS_TANK_CLASSIFY_REASONS: Readonly<Record<string, string>> = {
  ...GAS_TANK_ERROR_REASONS,
  ...L2_TO_L2_CDM_ERROR_REASONS,
}

/**
 * Convenience wrapper: classify a revert against the GasTank ABI (plus the
 * L2ToL2CDM ABI for nested messenger reverts). Mirrors the CDM classifier's
 * pre-EVM checks: relayer-EOA gas exhaustion (priority — caused by us,
 * fixable by topping up) and op-supervisor RPC rejections (`rpc_rejected` —
 * permanent; expired source log or unknown source chain). Before the
 * rpc_rejected check, supervisor refusals on this path classified 'unknown'
 * and burned ~43 minutes of count-based retries (R4).
 */
export function classifyGasTankRevert(
  err: unknown,
): RevertClassification | undefined {
  if (isRelayerInsufficientFunds(err)) {
    return {
      errorName: 'InsufficientFunds',
      reason: 'relayer_insufficient_funds',
    }
  }
  if (isRpcRejection(err)) {
    return {
      errorName: 'RpcRejected',
      reason: 'rpc_rejected',
    }
  }
  return classifyRevert(err, GAS_TANK_CLASSIFY_ABI, GAS_TANK_CLASSIFY_REASONS)
}

/**
 * Classifies errors returned by the Ponder backend.
 */
export function classifyPonderError(err: unknown): string {
  const msg = err instanceof Error ? err.message : String(err)
  if (msg.includes('API response validation failed')) return 'parse'
  const httpMatch = msg.match(/^HTTP (\d{3})/)
  if (httpMatch) {
    const status = parseInt(httpMatch[1], 10)
    if (status >= 500) return 'http_5xx'
    if (status >= 400) return 'http_4xx'
  }
  if (
    /\btimeout\b/i.test(msg) ||
    (err as { name?: string })?.name === 'AbortError'
  ) {
    return 'timeout'
  }
  return 'network'
}
