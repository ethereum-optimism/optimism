import { Address, connect, createClient } from '@wagmi/core'
import { isAddress } from 'ethers/lib/utils.js'
import { z } from 'zod'
import { providers, Wallet } from 'ethers'
import { MockConnector } from '@wagmi/core/connectors/mock'

import * as logger from '../lib/logger'
import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { DEFAULT_RPC_URL } from '../constants/defaultRpcUrl'
import { prepareWriteAttestation } from '../lib/prepareWriteAttestation'
import { writeAttestation } from '../lib/writeAttestation'
import { castAsDataType } from '../lib/castAsDataType'
import { dataTypeOptionValidator } from '../types/DataTypeOption'

const zodAddress = () =>
  z
    .string()
    .transform((addr) => addr as Address)
    .refine(isAddress, { message: 'Invalid address' })

const zodWallet = () => z.string().refine((key) => new Wallet(key))

const zodAttestation = () => z.union([z.string(), z.number(), z.boolean()])

export const writeOptionsValidators = {
  privateKey: zodWallet().describe('Address of the creator of the attestation'),
  about: zodAddress().describe('Address of the subject of the attestation'),
  key: z
    .string()
    .describe('Key of the attestation either as string or hex number'),
  value: zodAttestation().describe('Attestation value').default(''),
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
const validators = z.object(writeOptionsValidators)

export type WriteOptions = z.infer<typeof validators>

export const write = async (options: WriteOptions) => {
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

  const network = await provider.getNetwork()

  if (!network) {
    logger.error('Unable to detect chainId')
    process.exit(1)
  }

  await connect({
    // MockConnector is actually a vanilla connector
    // it's called mockConnector because normally they
    // expect us to connect with metamask or something
    // but we're just using a private key
    connector: new MockConnector({
      options: {
        chainId: network.chainId,
        signer: new Wallet(parsedOptions.privateKey, provider),
      },
    }),
  })

  try {
    const preparedTx = await prepareWriteAttestation(
      parsedOptions.about,
      parsedOptions.key,
      castAsDataType(parsedOptions.value, parsedOptions.dataType),
      network.chainId
    )
    const result = await writeAttestation(preparedTx)
    await result.wait()
    logger.log(`txHash: ${result.hash}`)
    return result.hash
  } catch (e) {
    logger.error('Unable to read attestation', e)
    process.exit(1)
  }
}
