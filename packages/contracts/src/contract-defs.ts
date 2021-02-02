import * as path from 'path'
import * as glob from 'glob'
import { ethers, ContractFactory, Signer } from 'ethers'
import { Interface } from 'ethers/lib/utils'

export const getContractDefinition = (name: string, ovm?: boolean): any => {
  const match = glob.sync(
    path.resolve(__dirname, `../artifacts`) +
      `/**/${name}${ovm ? '.ovm' : ''}.json`
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
