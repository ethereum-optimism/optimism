// constants
export { ATTESTATION_STATION_ADDRESS } from './constants/attestationStationAddress'
// lib
export { encodeRawKey, createKey } from './lib/createKey'
export { createValue, stringifyAttestationBytes } from './lib/createValue'
export {
  readAttestation,
  readAttestationAddress,
  readAttestationBool,
  readAttestationNumber,
  readAttestationString,
} from './lib/readAttestation'
export { readAttestations } from './lib/readAttestations'
export { getEvents } from './lib/getEvents'
export { prepareWriteAttestation } from './lib/prepareWriteAttestation'
export { prepareWriteAttestations } from './lib/prepareWriteAttestations'
export { writeAttestation } from './lib/writeAttestation'
export { abi } from './lib/abi'
export {
  parseAttestationBytes,
  parseAddress,
  parseNumber,
  parseBool,
  parseString,
} from './lib/parseAttestationBytes'
// types
export type { AttestationCreatedEvent } from './types/AttestationCreatedEvent'
export type { AttestationReadParams } from './types/AttestationReadParams'
export type { DataTypeOption } from './types/DataTypeOption'
export type { WagmiBytes } from './types/WagmiBytes'
// react
export * from './react'
