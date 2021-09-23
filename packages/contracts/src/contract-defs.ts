import {
  ethers,
  ContractFactory,
  Signer,
  providers,
  Contract,
  constants,
} from 'ethers'
import { Interface } from 'ethers/lib/utils'

export const getContractDefinition = (name: string, ovm?: boolean): any => {
  // We import this using `require` because hardhat tries to build this file when compiling
  // the contracts, but we need the contracts to be compiled before the contract-artifacts.ts
  // file can be generated.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const { getContractArtifact } = require('./contract-artifacts')
  const artifact = getContractArtifact(name, ovm)
  if (artifact === undefined) {
    throw new Error(`Unable to find artifact for contract: ${name}`)
  }
  return artifact
}

export const getContractInterface = (
  name: string,
  ovm?: boolean
): Interface => {
  const definition = getContractDefinition(name, ovm)
  return new ethers.utils.Interface(definition.abi)
}

export const getContractFactory = (
  name: string,
  signer?: Signer,
  ovm?: boolean
): ContractFactory => {
  const definition = getContractDefinition(name, ovm)
  const contractInterface = getContractInterface(name, ovm)
  return new ContractFactory(contractInterface, definition.bytecode, signer)
}

export const loadContract = (
  name: string,
  address: string,
  provider: providers.JsonRpcProvider
): Contract => {
  return new Contract(address, getContractInterface(name) as any, provider)
}

export const loadContractFromManager = async (args: {
  name: string
  proxy?: string
  Lib_AddressManager: Contract
  provider: providers.JsonRpcProvider
}): Promise<Contract> => {
  const { name, proxy, Lib_AddressManager, provider } = args
  const address = await Lib_AddressManager.getAddress(proxy ? proxy : name)

  if (address === constants.AddressZero) {
    throw new Error(
      `Lib_AddressManager does not have a record for a contract named: ${name}`
    )
  }

  return loadContract(name, address, provider)
}
