import { BigNumber } from 'ethers'
import { toUtf8String } from 'ethers/lib/utils.js'
import type { Address } from '@wagmi/core'

import type { DataTypeOption } from '../types/DataTypeOption'
import type { WagmiBytes } from '../types/WagmiBytes'
import { ParseBytesReturn } from '../types/ParseBytesReturn'

/**
 * Parses a string attestion
 */
export const parseString = (rawAttestation: WagmiBytes): string => {
  rawAttestation = rawAttestation === '0x0' ? '0x' : rawAttestation
  return rawAttestation ? toUtf8String(rawAttestation) : ''
}

/**
 * Parses a boolean attestion
 */
export const parseBool = (rawAttestation: WagmiBytes): boolean => {
  rawAttestation = rawAttestation === '0x' ? '0x0' : rawAttestation
  return rawAttestation ? BigNumber.from(rawAttestation).gt(0) : false
}

/**
 * Parses a number attestion
 */
export const parseNumber = (rawAttestation: WagmiBytes): BigNumber => {
  rawAttestation = rawAttestation === '0x' ? '0x0' : rawAttestation
  return rawAttestation ? BigNumber.from(rawAttestation) : BigNumber.from(0)
}

/**
 * Parses a address attestion
 */
export const parseAddress = (rawAttestation: WagmiBytes): Address => {
  rawAttestation = rawAttestation === '0x' ? '0x0' : rawAttestation
  return rawAttestation
    ? (BigNumber.from(rawAttestation).toHexString() as Address)
    : '0x0000000000000000000000000000000000000000'
}

/**
 * @deprecated use parseString, parseBool, parseNumber, or parseAddress instead
 * Will be removed in v1.0.0
 * @internal
 * Parses a raw attestation
 */
export const parseAttestationBytes = <TDataType extends DataTypeOption>(
  attestationBytes: WagmiBytes,
  dataType: TDataType
): ParseBytesReturn<TDataType> => {
  if (dataType === 'bytes') {
    return attestationBytes as ParseBytesReturn<TDataType>
  }
  if (dataType === 'number') {
    return parseNumber(attestationBytes) as ParseBytesReturn<TDataType>
  }
  if (dataType === 'address') {
    return parseAddress(attestationBytes) as ParseBytesReturn<TDataType>
  }
  if (dataType === 'bool') {
    return parseBool(attestationBytes) as ParseBytesReturn<TDataType>
  }
  if (dataType === 'string') {
    return parseString(attestationBytes) as ParseBytesReturn<TDataType>
  }
  console.warn(`unrecognized dataType ${dataType satisfies never}`)
  return attestationBytes as never
}
