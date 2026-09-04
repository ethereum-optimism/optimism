import type { AccessList, Hex } from 'viem'
import {
  encodePacked,
  keccak256,
  maxUint32,
  maxUint64,
  toBytes,
  toHex,
} from 'viem'

import { contracts } from '@/contracts.js'
import type { MessageIdentifier } from '@/types/interop/executingMessage.js'

function lookupEntry(id: MessageIdentifier): Hex {
  const out = new Uint8Array(32)
  const prefixLookup = 1
  out[0] = prefixLookup

  // For chainId, we need to handle values larger than uint64 so we'll only use the lower 64 bits
  // for the lookup entry
  const chainIdForLookup = id.chainId & maxUint64
  const chainIdBytes = toBytes(toHex(chainIdForLookup, { size: 8 }))
  out.set(chainIdBytes, 4)

  // Convert blockNumber to uint64 and place in bytes 12-20
  const blockNumberBytes = toBytes(toHex(id.blockNumber, { size: 8 }))
  out.set(blockNumberBytes, 12)

  // Convert timestamp to uint64 and place in bytes 20-28
  const timestampBytes = toBytes(toHex(id.timestamp, { size: 8 }))
  out.set(timestampBytes, 20)

  // Convert logIndex to uint32 and place in bytes 28-32
  const logIndexBytes = toBytes(toHex(id.logIndex, { size: 4 }))
  out.set(logIndexBytes, 28)

  // Convert the byte array to a hex string
  return `0x${Array.from(out)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')}`
}

function chainIDExtensionEntry(chainId: bigint): Hex {
  const out = new Uint8Array(32)
  const prefixChainIDExtension = 2
  out[0] = prefixChainIDExtension

  // Convert chainId to bytes32 representation
  const chainIdBytes = toBytes(toHex(chainId, { size: 32 }))

  // Copy bytes from chainId to output, starting at position 8
  // This copies 24 bytes from the start of chainIdBytes to out[8:32]
  out.set(chainIdBytes.slice(0, 24), 8)

  return `0x${Array.from(out)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')}`
}

function calculateChecksum(
  id: MessageIdentifier,
  payloadHash: `0x${string}`,
): Hex {
  const logHash = keccak256(
    encodePacked(['address', 'bytes32'], [id.origin, payloadHash]),
  )

  const idPacked = encodePacked(
    ['uint96', 'uint64', 'uint64', 'uint32'],
    [0n, id.blockNumber, id.timestamp, Number(id.logIndex)],
  )

  const idLogHash = keccak256(
    encodePacked(['bytes32', 'bytes32'], [logHash, idPacked]),
  )

  const bareChecksum = keccak256(
    encodePacked(['bytes32', 'uint256'], [idLogHash, id.chainId]),
  )

  const MSB_MASK =
    '0x00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
  const TYPE_3_MASK =
    '0x0300000000000000000000000000000000000000000000000000000000000000'
  const checksum =
    (BigInt(bareChecksum) & BigInt(MSB_MASK)) | BigInt(TYPE_3_MASK)

  return toHex(checksum, { size: 32 })
}

/**
 * @description Encodes an access list for validating a cross-chain message in the CrossL2Inbox contract
 * @category Utils
 * @param id - The message identifier containing origin, blockNumber, logIndex, timestamp, and chainId
 * @param messagePayload - The encoded payload of the message
 * @returns An AccessList containing the CrossL2Inbox contract address and storage keys needed for message validation
 * @example
 * const id = {
 *   origin: '0x4200000000000000000000000000000000000023',
 *   blockNumber: 14n,
 *   logIndex: 2n,
 *   timestamp: 1743801675n,
 *   chainId: 901n,
 * }
 *
 * const payload = '0x...' // encoded message payload
 *
 * const accessList = encodeAccessList(id, payload)
 * // Returns an AccessList to include in the transaction
 * // [{
 * //   address: '0x4200000000000000000000000000000000000022',
 * //   storageKeys: ['0x...', '0x...', '0x...']
 * // }]
 */
export function encodeAccessList(
  id: MessageIdentifier,
  messagePayload: `0x${string}`,
): AccessList {
  // Validate size constraints
  if (id.blockNumber > maxUint64) throw new Error('BlockNumberTooHigh')
  if (id.logIndex > maxUint32) throw new Error('LogIndexTooHigh')
  if (id.timestamp > maxUint64) throw new Error('TimestampTooHigh')

  const storageKeys: Hex[] = []

  storageKeys.push(lookupEntry(id))

  const isChainIdUint64 = id.chainId <= maxUint64
  if (!isChainIdUint64) {
    storageKeys.push(chainIDExtensionEntry(id.chainId))
  }

  const prefixChecksum = 3
  const checksum = calculateChecksum(id, keccak256(messagePayload))
  if (Number(checksum[3]) != prefixChecksum) {
    throw new Error('invalid checksum entry')
  }
  storageKeys.push(checksum)

  return [
    {
      address: contracts.crossL2Inbox.address,
      storageKeys,
    },
  ]
}
