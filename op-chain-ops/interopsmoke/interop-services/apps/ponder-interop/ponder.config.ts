import type { Transport } from 'viem'
import { http } from 'viem'
import { z } from 'zod'

import { createPonderConfig } from '@/createPonderConfig.js'

// Parse Endpoints from Environment
const endpoints: Record<string, { id: number; rpc: Transport }> = {}
for (const [key, value] of Object.entries(process.env)) {
  if (!key.startsWith('PONDER_INTEROP_ENDPOINT_') || value === undefined) {
    continue
  }

  const chainIdStr = key.replace('PONDER_INTEROP_ENDPOINT_', '')
  const chainIdSchema = z.string().regex(/^\d+$/)
  const chainIdResult = chainIdSchema.safeParse(chainIdStr)
  if (!chainIdResult.success) {
    throw new Error(
      `invalid chain id for ${key}. Use format PONDER_INTEROP_ENDPOINT_<chainId>=<url>: ${chainIdStr}`,
    )
  }

  const urlSchema = z.string().url()
  const result = urlSchema.safeParse(value)
  if (!result.success) {
    throw new Error(`invalid endpoint url for ${key}: ${value}`)
  }

  endpoints[chainIdStr] = {
    id: parseInt(chainIdStr),
    rpc: http(value),
  }
}

if (Object.keys(endpoints).length === 0) {
  throw new Error(
    'No endpoints in environment found. Please set `PONDER_INTEROP_ENDPOINT_<chainId>=<url>` urls for each chain to index.',
  )
}

export default createPonderConfig(endpoints)
