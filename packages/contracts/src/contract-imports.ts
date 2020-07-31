import * as path from 'path'
import { ethers, ContractFactory, Signer } from 'ethers'
import { Interface } from 'ethers/lib/utils'

export const getContractDefinition = (name: string): any => {
  return require(path.join(__dirname, '../artifacts', `${name}.json`))
}

export const getContractInterface = (name: string): Interface => {
  const definition = getContractDefinition(name)
  return new ethers.utils.Interface(definition.abi)
}

export const getContractFactory = (
  name: string,
  signer?: Signer
): ContractFactory => {
  const definition = getContractDefinition(name)
  const contractInterface = getContractInterface(name)
  return new ContractFactory(contractInterface, definition.bytecode, signer)
}
