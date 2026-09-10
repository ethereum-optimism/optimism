import type { Log } from 'viem'
import { parseEventLogs } from 'viem'

import { l2ToL2CrossDomainMessengerAbi } from '@/abis/index.js'

export type ExtractSentMessageLogsParameters = { logs: Log[] }
export type ExtractSentMessageLogsReturnType = Array<
  Log<
    bigint,
    number,
    false,
    undefined,
    true,
    typeof l2ToL2CrossDomainMessengerAbi,
    'SentMessage'
  >
>

/**
 * @description Extract all L2ToL2CrossDomainMessenger SentMessage logs from a receipt
 * @category Utils
 * @param params - {@link ExtractSentMessageLogsParameters}
 * @returns - {@link ExtractSentMessageLogsReturnType}
 */
export function extractSentMessageLogs(
  params: ExtractSentMessageLogsParameters,
): ExtractSentMessageLogsReturnType {
  return parseEventLogs({
    abi: l2ToL2CrossDomainMessengerAbi,
    eventName: 'SentMessage',
    logs: params.logs,
  })
}
