import { l2ToL2CrossDomainMessengerAbi } from '@eth-optimism/viem/abis'
import {
  encodeAbiParameters,
  encodeEventTopics,
  encodePacked,
  toHex,
} from 'viem'

import type { L2ToL2CrossDomainMessage } from '@/types/L2ToL2CrossDomainMessage'

export const encodeL2ToL2CrossDomainSentMessageEvent = (
  message: L2ToL2CrossDomainMessage,
) => {
  const topics = encodeEventTopics({
    abi: l2ToL2CrossDomainMessengerAbi,
    eventName: 'SentMessage',
    args: {
      destination: message.destination,
      target: message.target,
      messageNonce: message.messageNonce,
    },
  })

  const data = encodeAbiParameters(
    [{ type: 'address' }, { type: 'bytes' }],
    [message.sender, message.message],
  )

  if (topics === null || !Array.isArray(topics)) {
    throw new Error('Failed to encode event topics')
  }

  // TODO: MAYBE is a bug in viem? but prob not. either way when messageNonce is 0n, the topic is null
  if (topics[3] === null) {
    topics[3] = toHex(message.messageNonce, { size: 32 })
  }

  return encodePacked(
    ['bytes32[]', 'bytes'],
    [topics as Array<`0x${string}`>, data],
  )
}
