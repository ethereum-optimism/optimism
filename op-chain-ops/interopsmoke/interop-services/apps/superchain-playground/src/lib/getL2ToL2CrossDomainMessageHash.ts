import type { Hex } from 'viem'
import { encodeAbiParameters, keccak256 } from 'viem'

import type { L2ToL2CrossDomainMessage } from '@/types/L2ToL2CrossDomainMessage'

/**
 * Generates a unique hash for cross L2 messages. This hash is used to identify
 * the message and ensure it is not relayed more than once.
 * @param messageParams Object containing all parameters for the cross-domain message.
 * @returns Hash of the encoded message parameters, used to uniquely identify the message.
 */
export function getL2ToL2CrossDomainMessageHash(
  messageParams: L2ToL2CrossDomainMessage,
  source: bigint,
): Hex {
  const { destination, messageNonce, sender, target, message } = messageParams

  const encodedParams = encodeAbiParameters(
    [
      { type: 'uint256' },
      { type: 'uint256' },
      { type: 'uint256' },
      { type: 'address' },
      { type: 'address' },
      { type: 'bytes' },
    ],
    [destination, source, messageNonce, sender, target, message],
  )

  return keccak256(encodedParams)
}
