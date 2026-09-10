import { z } from 'zod'

import type { JsonRpcRequest } from '@/jsonrpc/types.js'

type ParseRequestBodyResult = {
  data: JsonRpcRequest | JsonRpcRequest[]
  error?: Error
}

const JsonRpcRequestZodSchema = z
  .object({
    jsonrpc: z.literal('2.0'),
    id: z.union([z.number().int(), z.string()]),
    method: z.string(),
    params: z.unknown().optional(),
  })
  .strict()

/**
 * Parse out the json rpcs out of the request body.
 * @param body - The request body.
 * @returns A single json rpc request or an array of json rpc requests - {@link JsonRpcRequest}
 */
export function parseRequestBody(body: unknown): ParseRequestBodyResult {
  const result = JsonRpcRequestZodSchema.safeParse(body)
  if (result.success) {
    return { data: result.data }
  }

  return { data: [], error: result.error }
}
