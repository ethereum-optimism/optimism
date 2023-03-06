import { ethers } from 'ethers'
import { Address } from 'wagmi'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { abi } from '../lib/abi'
import { AttestationCreatedEvent } from '../types/AttestationCreatedEvent'
import { encodeRawKey } from './createKey'

export const getEvents = async ({
  creator = null,
  about = null,
  key = null,
  value = null,
  provider,
  fromBlockOrBlockhash,
  toBlock,
}: {
  creator?: Address | null
  about?: Address | null
  key?: string | null
  value?: string | null
  provider: ethers.providers.JsonRpcProvider
  fromBlockOrBlockhash?: ethers.providers.BlockTag | undefined
  toBlock?: ethers.providers.BlockTag | undefined
}) => {
  const contract = new ethers.Contract(
    ATTESTATION_STATION_ADDRESS,
    abi,
    provider
  )
  return contract.queryFilter(
    contract.filters.AttestationCreated(
      creator,
      about,
      key && encodeRawKey(key),
      value
    ),
    fromBlockOrBlockhash,
    toBlock
  ) as Promise<AttestationCreatedEvent[]>
}
