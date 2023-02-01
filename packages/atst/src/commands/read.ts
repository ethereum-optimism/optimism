import { Address, createClient } from '@wagmi/core'
import { isAddress } from 'ethers/lib/utils.js'
import { z } from 'zod'
// TODO don't import all of ethers
import { providers } from 'ethers'

import * as logger from '../logger'
import { dataTypeOptions } from '../lib/readAttestation'

const zodAddress = () =>
  z
    .string()
    .transform((addr) => addr as Address)
    .refine(isAddress, { message: 'Invalid address' })

export const optionsValidators = {
  creator: zodAddress().describe('Address of the creator of the attestation'),
  about: zodAddress().describe('Address of the subject of the attestation'),
  key: z
    .string()
    .describe('Key of the attestation either as string or hex number'),
  dataType: dataTypeOptions,
  rpcUrl: z
    .string()
    .url()
    .optional()
    .default('https://mainnet.optimism.io')
    .describe('Rpc url to use'),
  contract: zodAddress()
    .optional()
    .default('0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77')
    .describe('Contract address to read from'),
}
const validators = z.object(optionsValidators)

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

  const { readAttestation } = await import('../lib/readAttestation')

  try {
    const result: string = await readAttestation(
      parsedOptions.creator,
      parsedOptions.about,
      parsedOptions.key,
      parsedOptions.dataType,
      parsedOptions.contract
    )
    logger.log(result.toString())
  } catch (e) {
    logger.error('Unable to read attestation', e)
    process.exit(1)
  }
}
