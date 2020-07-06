import * as path from 'path'
import { ethers } from 'ethers'
import { Interface } from 'ethers/lib/utils'

export const getContractDefinition = (name: string): any => {
  return require(path.join(__dirname, 'artifacts', `${name}.json`))
}

export const getContractInterface = (name: string): Interface => {
  const definition = getContractDefinition(name)
  return new ethers.utils.Interface(definition.abi)
}
