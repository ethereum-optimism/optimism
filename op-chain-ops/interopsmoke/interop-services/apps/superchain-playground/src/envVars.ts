import type { Address } from 'viem'
import { z } from 'zod'

const envVarSchema = z.object({
  VITE_WALLET_CONNECT_PROJECT_ID: z.string().transform((x) => x as Address),
})

export const envVars = envVarSchema.parse(import.meta.env)
