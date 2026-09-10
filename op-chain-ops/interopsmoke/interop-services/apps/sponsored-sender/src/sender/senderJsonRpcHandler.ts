import { Address as zodAddress } from 'abitype/zod'
import type { Hex, PublicClient } from 'viem'
import { isHex, toHex } from 'viem'
import type { Account } from 'viem/accounts'
import { sendTransaction } from 'viem/actions'
import { z } from 'zod'

import { JsonRpcHandler } from '@/jsonrpc/handler.js'

const zodHex = z.string().refine((val): val is Hex => isHex(val), {
  message: 'invalid hex string',
})

const zodTx = z.object(
  {
    to: zodAddress,
    data: zodHex,
    value: z.bigint().optional(),
  },
  {
    message: 'invalid transaction',
  },
)

/**
 * Create a JSON RPC handler that sponsors `eth_sendTransaction` requests. Also
 * supports `eth_chainId` and `eth_accounts` for contextual information.
 * @param sender - The sponsored sender's account.
 * @param client - The client.
 * @returns A JSON RPC handler.
 */
export function senderJsonRpcHandler(
  sender: Account,
  client: PublicClient,
): JsonRpcHandler {
  const handler = new JsonRpcHandler()

  handler.method(
    'eth_chainId',
    z.union([z.undefined(), z.tuple([])]),
    async () => {
      return toHex(await client.getChainId())
    },
  )

  handler.method('eth_accounts', z.undefined(), async () => {
    return [sender.address]
  })

  handler.method('eth_sendTransaction', z.tuple([zodTx]), async (params) => {
    const request = params[0]
    return sendTransaction(client, {
      account: sender,
      chain: client.chain,
      to: request.to,
      data: request.data,
      value: request.value,
    })
  })

  return handler
}
