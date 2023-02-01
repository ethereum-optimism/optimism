import { Address, readContract } from '@wagmi/core'
import { BigNumber } from 'ethers'
import { formatBytes32String, toUtf8String } from 'ethers/lib/utils.js'
import { z } from 'zod'

import * as logger from '../logger'
import { abi } from './abi'

type Bytes32 = `0x${string}`

export const dataTypeOptions = z
  .union([
    z.literal('string'),
    z.literal('bytes'),
    z.literal('number'),
    z.literal('bool'),
    z.literal('address'),
  ])
  .optional()
  .default('string')

export const readAttestation = async (
  creator: Address,
  about: Address,
  // TODO allow bytes array too
  key: Bytes32 | string,
  dataType: z.infer<typeof dataTypeOptions> = 'bytes',
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  if (key.length > 32) {
    throw new Error(
      'Key is longer than the max length of 32 for attestation keys'
    )
  }
  const dataBytes = await readContract({
    address: contractAddress,
    abi,
    functionName: 'attestations',
    args: [creator, about, formatBytes32String(key) as Bytes32],
  })
  if (dataType === 'bytes') {
    return dataBytes
  }
  if (dataType === 'number') {
    return BigNumber.from(dataBytes).toString()
  }
  if (dataType === 'address') {
    return BigNumber.from(dataBytes).toHexString()
  }
  if (dataType === 'bool') {
    return BigNumber.from(dataBytes).gt(0) ? 'true' : 'false'
  }
  if (dataType === 'string') {
    return toUtf8String(dataBytes)
  }
  logger.warn(`unrecognized dataType ${dataType satisfies never}`)
  return dataBytes
}
