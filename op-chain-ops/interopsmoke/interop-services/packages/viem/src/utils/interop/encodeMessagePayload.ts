import type { Log } from 'viem'
import { concat } from 'viem'

import type { MessagePayload } from '@/types/interop/executingMessage.js'

/**
 * @description Create an executing message payload from a log
 * @category Utils
 * @param log - The log to create the payload from
 * @returns The payload of the executing message
 */
export function encodeMessagePayload(log: Log): MessagePayload {
  return concat([...log.topics, log.data])
}
