import type { Address, Hex } from 'viem'

/**
 * Spec for [`MessageIdentifier`](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/messaging.md#message-identifier).
 * @category Types
 */
export type MessageIdentifier = {
  /** Account that emits log */
  origin: Address
  /** Block number in which the log was emitted */
  blockNumber: bigint
  /** The index of the log in the array of all logs emitted in the block */
  logIndex: bigint
  /** The timestamp that the log was emitted. Used to enforce the timestamp invariant */
  timestamp: bigint
  /** The chain that emitted the log */
  chainId: bigint
}

/**
 * Spec for [`MessagePayload`](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/messaging.md#message-payload).
 * @category Types
 */
export type MessagePayload = Hex
