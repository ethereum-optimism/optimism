import { Address, prepareWriteContract } from '@wagmi/core'
import { formatBytes32String } from 'ethers/lib/utils.js'

import { ATTESTATION_STATION_ADDRESS } from '../constants/attestationStationAddress'
import { WagmiBytes } from '../types/WagmiBytes'
import { abi } from './abi'
import { createValue } from './createValue'

export const prepareWriteAttestation = async (
  about: Address,
  key: string,
  value: string | WagmiBytes | number | boolean,
  chainId: number | undefined = undefined,
  contractAddress: Address = ATTESTATION_STATION_ADDRESS
) => {
  let formattedKey: WagmiBytes
  try {
    formattedKey = formatBytes32String(key) as WagmiBytes
  } catch (e) {
    console.error(e)
    throw new Error(
      `key is longer than 32 bytes: ${key}.  Try using a shorter key or using 'encodeRawKey' to encode the key into 32 bytes first`
    )
  }
  return prepareWriteContract({
    address: contractAddress,
    abi,
    functionName: 'attest',
    chainId,
    args: [about, formattedKey, createValue(value) as WagmiBytes],
  })
}
