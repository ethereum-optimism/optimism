import * as path from 'path'
import * as fs from 'fs'
import { Transaction } from 'ethers/utils'
import { Log } from 'ethers/providers'
import * as ethereumjsAbi from 'ethereumjs-abi'
import {
  add0x,
  remove0x,
  keccak256,
  abi,
  strToHexStr,
  bufferUtils,
  bufToHexString,
  hexStrToBuf,
} from '@eth-optimism/core-utils'

/**
 * Deterministically computes the smart contract address given
 * the account that will deploy the contract (factory contract)
 * the salt as uint256 and the contract bytecode
 * Source: https://github.com/miguelmota/solidity-create2-example
 * Note: Use this function to generate new tests
 */
export const buildCreate2Address = (
  creatorAddress: string,
  saltHex: string,
  byteCode: string
): string => {
  const preimage: string = `ff${remove0x(creatorAddress)}${remove0x(
    saltHex
  )}${keccak256(byteCode)}`
  return add0x(
    keccak256(preimage)
      .slice(-40)
      .toLowerCase()
  )
}

/**
 * Waits for a transaction to complete and returns the result
 * @param n the Number to convert
 * @returns The buffer
 */
export const getTransactionResult = async (
  provider: any,
  tx: Transaction,
  returnType: string
): Promise<any[]> => {
  const receipt = await provider.waitForTransaction(tx.hash)
  return abi.decode([returnType], receipt.logs.pop().data)
}

/**
 * Builds a ethers.js Log object from it's respective parts
 *
 * @param address The address the logs was sent from
 * @param event The event identifier
 * @param data The event data
 * @returns an ethers.js Log object
 */
export const buildLog = (
  address: string,
  event: string,
  data: string[],
  logIndex: number
): Log => {
  const types = event.match(/\((.+)\)/)
  const encodedData = types ? abi.encode(types[1].split(','), data) : '0x'

  return {
    address,
    topics: [add0x(keccak256(strToHexStr(event)))],
    data: encodedData,
    logIndex,
  }
}

/**
 * Computes the method id of a function name and encodes it as
 * a hexidecimal string.
 * @param The name of the function
 * @returns The hex-encoded methodId
 */
export const encodeMethodId = (functionName: string): string => {
  return ethereumjsAbi.methodID(functionName, []).toString('hex')
}

/**
 * Encodes an array of function arguments into a hex string.
 * @param any[] An array of arguments
 * @returns The hex-encoded function arguments
 */
export const encodeRawArguments = (args: any[]): string => {
  return args
    .map((arg) => {
      if (Number.isInteger(arg)) {
        return bufferUtils.numberToBuffer(arg).toString('hex')
      } else if (Buffer.isBuffer(arg)) {
        return arg.toString('hex')
      } else if (arg && arg.startsWith('0x')) {
        return remove0x(arg)
      } else {
        return arg
      }
    })
    .join('')
}

/**
 * Gets a padded big-endian 32-byte address string from an address string.
 * @param addr The 20-byte address string
 * @returns The 0x-prefixed 32-byte address string
 */
export const addressToBytes32Address = (addr: string): string => {
  return bufToHexString(
    bufferUtils.padLeft(hexStrToBuf(addr), 32)
  ).toLowerCase()
}

export const compile = (
  compiler: any,
  file: string,
  settings: any = {}
): any => {
  const input = {
    language: 'Solidity',
    sources: {
      [path.basename(file)]: {
        content: fs.readFileSync(file, 'utf8'),
      },
    },
    settings: {
      outputSelection: {
        '*': {
          '*': ['*'],
        },
      },
      ...settings,
    },
  }
  return JSON.parse(compiler.compile(JSON.stringify(input)))
}

export const encodeFunctionData = (
  functionName: string,
  functionParams: any[] = []
): string => {
  return add0x(
    encodeMethodId(functionName) + encodeRawArguments(functionParams)
  )
}

export const getCodeHash = async (
  provider: any,
  address: string
): Promise<string> => {
  return keccak256(await provider.getCode(address))
}
