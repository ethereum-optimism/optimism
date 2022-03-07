import { ethers } from 'ethers'

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

export const getContractInterface = (name: string): ethers.utils.Interface => {
  const definition = getContractDefinition(name)
  return new ethers.utils.Interface(definition.abi)
}

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
