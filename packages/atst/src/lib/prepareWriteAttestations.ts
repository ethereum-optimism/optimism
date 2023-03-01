import { Address, prepareWriteContract } from '@wagmi/core'
import { formatBytes32String } from 'ethers/lib/utils.js'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { WagmiBytes } from '../types/WagmiBytes'
import { abi } from './abi'
import { stringifyAttestationBytes } from './stringifyAttestationBytes'

type Attestation = {
  about: Address
  key: string
  value: string | WagmiBytes | number | boolean
}

export const prepareWriteAttestations = async (
  attestations: Attestation[],
  chainId = 10,
  contractAddress: Address = ATTESTATION_STATION_ADDRESS
) => {
  const formattedAttestations = attestations.map((attestation) => {
    const formattedKey = formatBytes32String(attestation.key) as WagmiBytes
    const formattedValue = stringifyAttestationBytes(
      attestation.value
    ) as WagmiBytes
    return {
      about: attestation.about,
      key: formattedKey,
      val: formattedValue,
    } as const
  })
  return prepareWriteContract({
    address: contractAddress,
    abi,
    functionName: 'attest',
    chainId,
    args: [formattedAttestations],
  })
}
