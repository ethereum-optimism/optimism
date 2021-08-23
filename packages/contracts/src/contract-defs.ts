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

export const loadContract = (
  name: string,
  address: string,
  provider: ethers.providers.JsonRpcProvider
): ethers.Contract => {
  return new ethers.Contract(
    address,
    getContractInterface(name) as any,
    provider
  )
}

export const loadContractFromManager = async (args: {
  name: string
  proxy?: string
  Lib_AddressManager: ethers.Contract
  provider: ethers.providers.JsonRpcProvider
}): Promise<ethers.Contract> => {
  const { name, proxy, Lib_AddressManager, provider } = args
  const address = await Lib_AddressManager.getAddress(proxy ? proxy : name)
  if (address === ethers.constants.AddressZero) {
    throw new Error(
      `Lib_AddressManager does not have a record for a contract named: ${name}`
    )
  }
  return loadContract(name, address, provider)
}
