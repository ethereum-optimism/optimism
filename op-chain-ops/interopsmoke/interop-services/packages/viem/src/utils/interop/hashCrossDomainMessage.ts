import type { Hash } from 'viem'
import { encodeAbiParameters, keccak256, parseAbiParameters } from 'viem'

import type { CrossDomainMessage } from '@/types/interop/cdm.js'

/**
 * Hash an L2 to L2 cross domain message
 * @category Utils
 * @param message {@link CrossDomainMessage}
 * @returns Hash of the cross domain message
 */
export function hashCrossDomainMessage(message: CrossDomainMessage): Hash {
  const encoded = encodeAbiParameters(
    parseAbiParameters('uint256,uint256,uint256,address,address,bytes'),
    [
      message.destination,
      message.source,
      message.nonce,
      message.sender,
      message.target,
      message.message,
    ],
  )

  return keccak256(encoded)
}
