import { writeContract } from '@wagmi/core'
export { prepeareWriteAttestation } from './lib/prepareWriteAttestation'
export { readAttestation } from './lib/readAttestation'

export { abi } from './lib/abi'

/**
 * Will type this properly later
 *
 * @example
 * const config = await prepareAttestation(about, key, value)
 * const tx = await writeAttestation(config)
 */
export const writeAttestation = writeContract
