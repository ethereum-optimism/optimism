import type { AccessList, Account, Chain, Client, Log, Transport } from 'viem'
import { BaseError } from 'viem'
import { getBlock, getChainId } from 'viem/actions'

import type {
  MessageIdentifier,
  MessagePayload,
} from '@/types/interop/executingMessage.js'
import { encodeAccessList } from '@/utils/interop/encodeAccessList.js'
import { encodeMessagePayload } from '@/utils/interop/encodeMessagePayload.js'

/**
 * @category Types
 */
export type BuildExecutingMessageParameters = { log: Log }

/**
 * @category Types
 */
export type BuildExecutingMessageReturnType = {
  id: MessageIdentifier
  payload: MessagePayload
  accessList: AccessList
}

/**
 * @category Types
 */
export type ExecutingMessagePendingLogErrorType =
  ExecutingMessagePendingLogError & {
    name: 'ExecutingMessagePendingLogError'
  }

export class ExecutingMessagePendingLogError extends BaseError {
  constructor(log: Log) {
    const txHash = log.transactionHash
    super(
      `log from pending tx ${txHash} cannot be constructed into an interop message`,
    )
  }
}

/**
 * Build an executing message from a log
 * @category Actions
 * @param client - client to the chain that emitted the log
 * @param params - {@link BuildExecutingMessageParameters}
 * @returns - {@link BuildExecutingMessageReturnType}
 * @example
 * import { createPublicClient } from 'viem'
 * import { http } from 'viem/transports'
 * import { op } from '@eth-optimism/viem/chains'
 *
 * const publicClientOp = createPublicClient({ chain: op, transport: http() })
 * const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
 * const params = await buildExecutingMessage(publicClientOp, { log: receipt.logs[0] })
 */
export async function buildExecutingMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  params: BuildExecutingMessageParameters,
): Promise<BuildExecutingMessageReturnType> {
  const { log } = params
  if (
    log.blockHash === null ||
    log.blockNumber === null ||
    log.logIndex === null
  ) {
    throw new ExecutingMessagePendingLogError(log)
  }

  const chainId = await getChainId(client)
  const block = await getBlock(client, { blockHash: log.blockHash })
  const id = {
    origin: log.address,
    logIndex: BigInt(log.logIndex),
    blockNumber: block.number,
    timestamp: block.timestamp,
    chainId: BigInt(chainId),
  }

  const payload = encodeMessagePayload(log)
  const accessList = encodeAccessList(id, payload)

  return { id, payload, accessList }
}
