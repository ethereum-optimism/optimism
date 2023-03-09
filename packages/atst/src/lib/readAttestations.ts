import { readContracts } from '@wagmi/core'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import type { AttestationReadParams } from '../types/AttestationReadParams'
import { DEFAULT_DATA_TYPE } from '../types/DataTypeOption'
import type { WagmiBytes } from '../types/WagmiBytes'
import { abi } from './abi'
import { createKey } from './createKey'
import { parseAttestationBytes } from './parseAttestationBytes'

/**
 * reads attestations from the attestation station contract
 *
 * @returns an array of attestation values
 * @throws Error if key is longer than 32 bytes
 * @example
 * const attestations = await readAttestations(
 *  {
 *    creator: creatorAddress,
 *    about: aboutAddress,
 *    key: 'my_key',
 *  },
 *  {
 *    creator: creatorAddress2,
 *    about: aboutAddress2,
 *    key: 'my_key',
 *    dataType: 'number',
 *    contractAddress: '0x1234',
 *   },
 * )
 */
export const readAttestations = async (
  ...attestationReads: Array<AttestationReadParams>
) => {
  const calls = attestationReads.map((attestation) => {
    const {
      creator,
      about,
      key,
      contractAddress = ATTESTATION_STATION_ADDRESS,
    } = attestation
    return {
      address: contractAddress,
      abi,
      functionName: 'attestations',
      args: [creator, about, createKey(key) as WagmiBytes],
    } as const
  })

  const results = await readContracts({
    contracts: calls,
  })

  return results.map((dataBytes, i) => {
    const dataType = attestationReads[i].dataType ?? DEFAULT_DATA_TYPE
    return parseAttestationBytes(dataBytes, dataType)
  })
}
