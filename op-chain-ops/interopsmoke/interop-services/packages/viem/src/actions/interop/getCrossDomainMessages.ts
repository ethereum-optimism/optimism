import type { Account, Chain, Client, Log, Transport } from 'viem'
import { getChainId } from 'viem/actions'

import type { CrossDomainMessage } from '@/types/interop/cdm.js'
import { extractSentMessageLogs } from '@/utils/interop/extractSentMessageLogs.js'

/**
 * @category Types
 */
export type GetCrossDomainMessagesParameters = { logs: Log[] }

/**
 * @category Types
 */
export type GetCrossDomainMessagesReturnType = CrossDomainMessage[]

/**
 * Get all cross domain messages from a set of logs
 * @category Actions
 * @param client - The client to use
 * @param parameters - {@link GetCrossDomainMessagesParameters}
 * @returns cross domain messages - {@link GetCrossDomainMessagesReturnType}
 * @example
 * import { createPublicClient } from 'viem'
 * import { http } from 'viem/transports'
 * import { op } from '@eth-optimism/viem/chains'
 * import { getCrossDomainMessages } from '@eth-optimism/viem/actions/interop'
 *
 * const publicClientOp = createPublicClient({ chain: op, transport: http() })
 * const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
 * const messages = await getCrossDomainMessages(publicClientOp, { logs: receipt.logs })
 */
export async function getCrossDomainMessages<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: GetCrossDomainMessagesParameters,
): Promise<GetCrossDomainMessagesReturnType> {
  const { logs } = parameters

  const chainId = await getChainId(client)
  const sentMessages = extractSentMessageLogs({ logs })

  return sentMessages.map((log) => {
    return {
      source: BigInt(chainId),
      destination: log.args.destination,
      nonce: log.args.messageNonce,
      sender: log.args.sender,
      target: log.args.target,
      message: log.args.message,
      log,
    } satisfies CrossDomainMessage
  })
}
