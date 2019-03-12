/* tslint:disable:no-any */
/* Credit: https://github.com/ethereum/remix */

import { ethers } from 'ethers'

export interface ABIResult {
  [key: number]: string | number | boolean
}

interface ABIType {
  type: string
  name: string
  components?: ABIType[]
}

export interface ABIObject {
  type: string
  name: string
  inputs: ABIType[]
  outputs: ABIType[]
}

const extractSize = (type: string): string => {
  const size = type.match(/([a-zA-Z0-9])(\[.*\])/)
  return size ? size[2] : ''
}

const makeFullTypeDefinition = (typeDef: ABIType): string => {
  if (typeDef && typeDef.type.indexOf('tuple') === 0 && typeDef.components) {
    const innerTypes = typeDef.components.map((innerType) => {
      return makeFullTypeDefinition(innerType)
    })
    return `tuple(${innerTypes.join(',')})${extractSize(typeDef.type)}`
  }
  return typeDef.type
}

export const encodeParams = (types: string[], args: any[]): string => {
  const abiCoder = new ethers.utils.AbiCoder()
  return abiCoder.encode(types, args)
}

const encodeMethodParams = (methodAbi: ABIObject, args: any[]): string => {
  const types: string[] = []
  if (methodAbi.inputs && methodAbi.inputs.length) {
    for (const input of methodAbi.inputs) {
      const type = input.type
      types.push(
        type.indexOf('tuple') === 0 ? makeFullTypeDefinition(input) : type
      )
      if (args.length < types.length) {
        args.push('')
      }
    }
  }

  return encodeParams(types, args)
}

const encodeMethodId = (methodAbi: ABIObject): string => {
  if (methodAbi.type === 'fallback') {
    return '0x'
  }

  const abi = new ethers.utils.Interface([methodAbi])
  const fn = abi.functions[methodAbi.name]
  return fn.sighash
}

export const encodeMethod = (methodAbi: ABIObject, args: any[]): string => {
  const encodedParams = encodeMethodParams(methodAbi, args).replace('0x', '')
  const methodId = encodeMethodId(methodAbi)
  return methodId + encodedParams
}

export const decodeResponse = (
  methodAbi: ABIObject,
  response: Buffer | Uint8Array
): ABIResult => {
  if (!methodAbi.outputs || methodAbi.outputs.length === 0) {
    return {}
  }

  const outputTypes = []
  for (const output of methodAbi.outputs) {
    const type = output.type
    outputTypes.push(
      type.indexOf('tuple') === 0 ? makeFullTypeDefinition(output) : type
    )
  }

  if (!response.length) {
    response = new Uint8Array(32 * methodAbi.outputs.length)
  }

  const abiCoder = new ethers.utils.AbiCoder()
  return abiCoder.decode(outputTypes, response)
}
