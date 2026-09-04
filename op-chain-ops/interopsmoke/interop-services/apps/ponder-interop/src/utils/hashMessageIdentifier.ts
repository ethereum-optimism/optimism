import type { MessageIdentifier } from '@eth-optimism/viem'
import type { Hash } from 'viem'
import { encodeAbiParameters, keccak256, parseAbiParameters } from 'viem'

/**
 * Hash an L2 to L2 cross domain message identifier
 * @param message {@link MessageIdentifier}
 * @returns Hash of the cross domain message identifier
 */
export function hashMessageIdentifier(message: MessageIdentifier): Hash {
  const encoded = encodeAbiParameters(
    parseAbiParameters('address,uint256,uint256,uint256,uint256'),
    [
      message.origin,
      message.blockNumber,
      message.logIndex,
      message.timestamp,
      message.chainId,
    ],
  )

  return keccak256(encoded)
}
