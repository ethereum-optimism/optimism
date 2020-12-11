import * as path from 'path'
import { ethers, ContractFactory, Signer } from 'ethers'
import { Interface } from 'ethers/lib/utils'

export const getContractDefinition = (name: string, ovm?: boolean): any => {
  return require(path.join(
    __dirname,
    `../artifacts${ovm ? '/ovm' : ''}`,
    `${name}.json`
  ))
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
