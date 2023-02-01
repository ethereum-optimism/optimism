import {
  readContract,
  prepareWriteContract,
  writeContract,
  Address,
} from '@wagmi/core'

import { abi } from './abi'

type Bytes32 = string | number[]
type Bytes = string

export const readAttestation = async (
  creator: Address,
  about: Address,
  key: Bytes32
) => {
  return readContract({
    address: '0xecb504d39723b0be0e3a9aa33d646642d1051ee1',
    abi,
    functionName: 'attestations',
    args: [creator, about, key],
  })
}

export const prepeareWriteAttestation = async (
  about: Address,
  key: Bytes32,
  value: Bytes
) => {
  return prepareWriteContract({
    address: '0xecb504d39723b0be0e3a9aa33d646642d1051ee1',
    abi,
    functionName: 'attest',
    args: [],
  })
}

export const writeAttestation = async (address: string) => {
  return prepareWriteContract({
    address: '0xecb504d39723b0be0e3a9aa33d646642d1051ee1',
    abi,
    functionName: 'attest',
  })
  const { hash } = await writeContract(config)
}
