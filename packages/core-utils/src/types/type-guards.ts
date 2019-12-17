/* Internal Imports */
import {
  JsonRpcErrorResponse,
  JsonRpcRequest,
  JsonRpcResponse,
} from './transport.interface'
import { Range } from './range'
import { ZERO, remove0x } from '../app'
import { Address } from 'cluster'

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

export const isHexStringAddress = (address: any): address is Address => {
  return (
    remove0x(address).length === 40
  )
}