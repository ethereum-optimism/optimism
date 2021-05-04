import * as path from 'path'
import * as glob from 'glob'
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
  const match = glob.sync(
    path.resolve(__dirname, `../artifacts${ovm ? '-ovm' : ''}`) +
      `/**/${name.split('-').join(':')}.json`
  )

  if (match.length > 0) {
    return require(match[0])
  } else {
    throw new Error(`Unable to find artifact for contract: ${name}`)
  }
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
