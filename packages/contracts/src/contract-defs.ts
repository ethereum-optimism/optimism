import { ethers } from 'ethers'
/** 
 * Gets the contract's artifact.
 * 
 * @param name Given contract's name. 
 * @returns Hardhat's artifact for the given contract. 
 */ 
export const getContractArtifact = (name: string): any => {
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
 * Gets the contract's interface.
 * 
 * @param name Given contract's name. 
 * @returns Interface for the given contract. 
 */ 
export const getContractInterface = (name: string): ethers.utils.Interface => {
  const definition = getContractDefinition(name)
  return new ethers.utils.Interface(definition.abi)
}

/** 
 * Gets the contract to deploy.
 * 
 * @param name Given contract's name. 
 * @param signer Signer of contract. 
 * @returns Contract's Factory.
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
