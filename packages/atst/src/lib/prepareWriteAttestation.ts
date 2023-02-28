import { Address, prepareWriteContract } from '@wagmi/core'
import { formatBytes32String } from 'ethers/lib/utils.js'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { WagmiBytes } from '../types/WagmiBytes'
import { abi } from './abi'
import { stringifyAttestationBytes } from './stringifyAttestationBytes'

export const prepareWriteAttestation = async (
  about: Address,
  key: string,
  value: string | WagmiBytes | number | boolean,
  chainId = 10,
  contractAddress: Address = ATTESTATION_STATION_ADDRESS
) => {
  const formattedKey = formatBytes32String(key) as WagmiBytes
  return prepareWriteContract({
    address: contractAddress,
    abi,
    functionName: 'attest',
    chainId,
    args: [about, formattedKey, stringifyAttestationBytes(value) as WagmiBytes],
  })
}
