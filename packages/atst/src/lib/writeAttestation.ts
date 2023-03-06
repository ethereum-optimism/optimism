import { writeContract } from '@wagmi/core'
export { prepareWriteAttestation } from './prepareWriteAttestation'

export { abi } from './abi'

/**
 * Writes an attestation to the blockchain
 * Same function as `writeContract` from @wagmi/core
 * To use first use prepareWriteContract
 *
 * @example
 * const config = await prepareAttestation(about, key, value)
 * const tx = await writeAttestation(config)
 */
export const writeAttestation = writeContract
