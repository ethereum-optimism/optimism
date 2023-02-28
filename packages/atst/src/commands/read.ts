import { Address, createClient } from '@wagmi/core'
import { isAddress } from 'ethers/lib/utils.js'
import { z } from 'zod'
import { providers } from 'ethers'

import * as logger from '../lib/logger'
import { dataTypeOptionValidator } from '../types/DataTypeOption'
import type { WagmiBytes } from '../types/WagmiBytes'
import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { DEFAULT_RPC_URL } from '../constants/defaultRpcUrl'
import { readAttestation } from '../lib/readAttestation'

const zodAddress = () =>
  z
    .string()
    .transform((addr) => addr as Address)
    .refine(isAddress, { message: 'Invalid address' })

export const readOptionsValidators = {
  creator: zodAddress().describe('Address of the creator of the attestation'),
  about: zodAddress().describe('Address of the subject of the attestation'),
  key: z
    .string()
    .describe('Key of the attestation either as string or hex number'),
  dataType: dataTypeOptionValidator,
  rpcUrl: z
    .string()
    .url()
    .optional()
    .default(DEFAULT_RPC_URL)
    .describe('Rpc url to use'),
  contract: zodAddress()
    .optional()
    .default(ATTESTATION_STATION_ADDRESS)
    .describe('Contract address to read from'),
}
const validators = z.object(readOptionsValidators)

export type ReadOptions = z.infer<typeof validators>

export const read = async (options: ReadOptions) => {
  // TODO make these errors more user friendly
  const parsedOptions = await validators.parseAsync(options).catch((e) => {
    logger.error(e)
    process.exit(1)
  })

  const provider = new providers.JsonRpcProvider({
    url: parsedOptions.rpcUrl,
    headers: {
      'User-Agent': '@eth-optimism/atst',
    },
  })

  createClient({
    provider,
  })

  try {
    const result = await readAttestation(
      parsedOptions.creator,
      parsedOptions.about,
      parsedOptions.key as WagmiBytes,
      parsedOptions.dataType,
      parsedOptions.contract
    )
    logger.log(result?.toString())
    return result?.toString()
  } catch (e) {
    logger.error('Unable to read attestation', e)
    process.exit(1)
  }
}
