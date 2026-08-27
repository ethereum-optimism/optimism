import 'dotenv/config'

import { getAddress, isAddress } from 'viem'
import { inferSchemas, parseEnv } from 'znv'
import { z } from 'zod'

const optionalAddress = z
  .string()
  .optional()
  .refine(
    (address) => address === undefined || isAddress(address),
    'must be a valid address',
  )
  .transform((address) =>
    address === undefined ? address : getAddress(address),
  )

export const envVarsSchema = inferSchemas({
  GAS_TANK_CONTRACT_ADDRESS: { schema: optionalAddress },
  DEPOSIT_CONTRACT_ADDRESS: { schema: optionalAddress },
  PROMISE_CONTRACT_ADDRESS: { schema: optionalAddress },
  CALLBACK_CONTRACT_ADDRESS: { schema: optionalAddress },
})

export const envVars = parseEnv(process.env, envVarsSchema)
