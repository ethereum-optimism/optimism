import type { Address } from '@wagmi/core'

import { DataTypeOption, DEFAULT_DATA_TYPE } from '../types/DataTypeOption'
import { readAttestations } from './readAttestations'

/**
 * reads attestation from the attestation station contract
 *
 * @param attestationRead - the parameters for reading an attestation
 * @returns attestation result
 * @throws Error if key is longer than 32 bytes
 * @example
 * const attestation = await readAttestation(
 * {
 *  creator: creatorAddress,
 * about: aboutAddress,
 * key: 'my_key',
 * },
 */
export const readAttestation = async (
  creator: Address,
  about: Address,
  key: string,
  dataType: DataTypeOption = DEFAULT_DATA_TYPE,
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  const [result] = await readAttestations({
    creator,
    about,
    key,
    dataType,
    contractAddress,
  })
  return result
}
