import type { Account, Chain, Client, Transport } from 'viem'
import { BaseError } from 'viem'
import { getChainId, readContract } from 'viem/actions'

import { l2ToL2CrossDomainMessengerAbi } from '@/abis/index.js'
import { contracts } from '@/contracts.js'
import type { CrossDomainMessage } from '@/types/interop/cdm.js'
import { hashCrossDomainMessage } from '@/utils/interop/hashCrossDomainMessage.js'

/**
 * @category Types
 */
export type GetCrossDomainMessageStatusParameters = {
  message: CrossDomainMessage
}

/**
 * @category Types
 */
export type GetCrossDomainMessageStatusReturnType = 'ready-to-relay' | 'relayed'

/**
 * @category Types
 */
export type InvalidDestinationChainErrorType = InvalidDestinationChainError & {
  name: 'InvalidDestinationChainError'
}
export class InvalidDestinationChainError extends BaseError {
  constructor(destination: bigint, chainId: bigint) {
    super(`Invalid destination chain: ${destination} !== ${chainId}`)
  }
}

/**
 * Get the status of a cross domain message
 * @category Actions
 * @param client - The client to use
 * @param parameters - {@link GetCrossDomainMessageStatusParameters}
 * @returns status -{@link GetCrossDomainMessageStatusReturnType}
 * @example
 * import { createPublicClient } from 'viem'
 * import { op, unichain } from '@eth-optimism/viem/chains'
 *
 * const publicClientOp = createPublicClient({ chain: op, transport: http() })
 * const publicClientUnichain = createPublicClient({ chain: unichain, transport: http() })
 *
 * const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
 * const messages = await getCrossDomainMessages(publicClientOp, { logs: receipt.logs })
 *
 * const message = messages.filter((message) => message.destination === unichain.id)[0]
 * const status = await getCrossDomainMessageStatus(publicClientUnichain, { message })
 */
export async function getCrossDomainMessageStatus<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: GetCrossDomainMessageStatusParameters,
): Promise<GetCrossDomainMessageStatusReturnType> {
  const { message } = parameters

  const chainId = BigInt(await getChainId(client))
  if (chainId !== message.destination) {
    throw new InvalidDestinationChainError(message.destination, chainId)
  }

  const messageHash = hashCrossDomainMessage(message)
  const relayed = await readContract(client, {
    address: contracts.l2ToL2CrossDomainMessenger.address,
    abi: l2ToL2CrossDomainMessengerAbi,
    functionName: 'successfulMessages',
    args: [messageHash],
  })

  return relayed ? 'relayed' : 'ready-to-relay'
}
