/* Internal Imports */
import {
  JsonRpcErrorResponse,
  JsonRpcRequest,
  JsonRpcResponse,
} from './transport.interface'
import { Range } from './range'
import { ZERO, add0x } from '../app'
import { Address } from 'cluster'

/* External Imports */
import { ethers } from 'ethers'

/**
 * Checks if a JSON-RPC response is an error response.
 * @param response Response to check.
 * @returns `true` if the response has an error, `false` otherwise.
 */
export const isJsonRpcErrorResponse = (
  response: JsonRpcResponse
): response is JsonRpcErrorResponse => {
  return typeof (response as JsonRpcErrorResponse).error !== 'undefined'
}

/**
 * Checks if a request is a valid JSON-RPC request.
 * @param request Request to check.
 * @returns `true` if the request is a valid JSON-RPC request, `false` otherwise.
 */
export const isJsonRpcRequest = (request: any): request is JsonRpcRequest => {
  return (
    request.method !== undefined &&
    request.id !== undefined &&
    request.jsonrpc === '2.0'
  )
}

/**
 * Validates that the provided Range has all of the necessary fields populated with valid data.
 *
 * @param range the Range to inspect
 * @returns true if valid, false otherwise
 */
export const isValidRange = (range: any): range is Range => {
  return (
    !!range &&
    !!range.start &&
    !!range.end &&
    range.start.gte(ZERO) &&
    range.end.gt(range.start)
  )
}

/**
 * Validates that the provided address hex string is the right length.
 *
 * @param address the string which is supposed to be a 20-byte address.
 * @returns true if valid address hex string, false otherwise
 */
export const isValidHexAddress = (address: any): address is Address => {
  try {
    ethers.utils.getAddress(add0x(address))
    return true
  } catch {
    return false
  }
}
