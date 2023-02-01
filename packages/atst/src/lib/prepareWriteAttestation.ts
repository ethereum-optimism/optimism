import { Address, prepareWriteContract } from '@wagmi/core'
import { formatBytes32String, toUtf8String } from 'ethers/lib/utils.js'

import { abi } from './abi'

type Bytes = `0x${string}`
type Bytes32 = `0x${string}`

export const prepeareWriteAttestation = async (
  about: Address,
  key: Bytes32 | string,
  value: Bytes | string,
  contractAddress: Address
) => {
  // TODO throw a friendly error message if key is bigger than bytes32
  return prepareWriteContract({
    address: contractAddress,
    abi,
    functionName: 'attest',
    args: [
      about,
      formatBytes32String(key) as Bytes32,
      toUtf8String(value) as Bytes,
    ],
  })
}
