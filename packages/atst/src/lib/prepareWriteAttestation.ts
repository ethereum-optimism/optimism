import { Address, prepareWriteContract } from '@wagmi/core'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { WagmiBytes } from '../types/WagmiBytes'
import { abi } from './abi'
import { createKey } from './createKey'
import { createValue } from './createValue'

export const prepareWriteAttestation = async (
  about: Address,
  key: string,
  value: string | WagmiBytes | number | boolean,
  chainId: number | undefined = undefined,
  contractAddress: Address = ATTESTATION_STATION_ADDRESS
) => {
  const formattedKey = createKey(key) as WagmiBytes
  return prepareWriteContract({
    address: contractAddress,
    abi,
    functionName: 'attest',
    chainId,
    args: [about, formattedKey, createValue(value) as WagmiBytes],
  })
}
