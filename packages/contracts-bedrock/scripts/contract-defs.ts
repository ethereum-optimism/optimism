import { ethers } from 'ethers'

/**
 * Gets the hardhat artifact for the given contract name.
 * Will throw an error if the contract artifact is not found.
 *
 * @param name Contract name.
 * @returns The artifact for the given contract name.
 */
export const getContractDefinition = (name: string): any => {
  // We import this using `require` because hardhat tries to build this file when compiling
  // the contracts, but we need the contracts to be compiled before the contract-artifacts.ts
  // file can be generated.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const { getContractArtifact } = require('./contract-artifacts')
  const artifact = getContractArtifact(name)
  if (artifact === undefined) {
    throw new Error(`Unable to find artifact for contract: ${name}`)
  }
  return artifact
}

/**
 * Gets an ethers Interface instance for the given contract name.
 *
 * @param name Contract name.
 * @returns The interface for the given contract name.
 */
export const getContractInterface = (name: string): ethers.utils.Interface => {
  const definition = getContractDefinition(name)
  return new ethers.utils.Interface(definition.abi)
}

/**
 * Gets an ethers ContractFactory instance for the given contract name.
 *
 * @param name Contract name.
 * @param signer The signer for the ContractFactory to use.
 * @returns The contract factory for the given contract name.
 */
export const getContractFactory = (
  name: string,
  signer?: ethers.Signer
): ethers.ContractFactory => {
  const definition = getContractDefinition(name)
  const contractInterface = getContractInterface(name)
  return new ethers.ContractFactory(
    contractInterface,
    definition.bytecode,
    signer
  )
}
