// constants
export { ATTESTATION_STATION_ADDRESS } from './constants/attestationStationAddress'
// lib
export {
  readAttestation,
  readAttestationAddress,
  readAttestationBool,
  readAttestationNumber,
  readAttestationString,
} from './lib/readAttestation'
export { readAttestations } from './lib/readAttestations'
export { prepareWriteAttestation } from './lib/prepareWriteAttestation'
export { prepareWriteAttestations } from './lib/prepareWriteAttestations'
export { writeAttestation } from './lib/writeAttestation'
export { abi } from './lib/abi'
export { stringifyAttestationBytes } from './lib/stringifyAttestationBytes'
export {
  parseAttestationBytes,
  parseAddress,
  parseNumber,
  parseBool,
  parseString,
} from './lib/parseAttestationBytes'
// types
export type { AttestationReadParams } from './types/AttestationReadParams'
export type { WagmiBytes } from './types/WagmiBytes'
export type { DataTypeOption } from './types/DataTypeOption'
