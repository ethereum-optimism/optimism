import BigNum = require('bn.js')

import {
  JsonRpcResponse,
  JsonRpcErrorResponse,
  JsonRpcRequest,
  Transaction,
  VerifiedStateUpdate,
  Range,
  StateUpdate,
  StateObject,
} from '../../../interfaces'

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

const zero: BigNum = new BigNum(0)

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
    range.start.gte(zero) &&
    range.end.gt(range.start)
  )
}

/**
 * Validates that the provided StateObject has all of the necessary fields populated with valid data.
 *
 * @param stateObject the StateObject to inspect
 * @returns true if valid, false otherwise
 */
export const isValidStateObject = (
  stateObject: any
): stateObject is StateObject => {
  return !!stateObject.predicate && !!stateObject.parameters
}

/**
 * Validates that the provided StateUpdate has all of the necessary fields populated with valid data.
 *
 * @param stateUpdate the StateUpdate to inspect
 * @returns true if valid, false otherwise
 */
export const isValidStateUpdate = (
  stateUpdate: any
): stateUpdate is StateUpdate => {
  return (
    !!stateUpdate &&
    !!stateUpdate.newState &&
    !!stateUpdate.id &&
    isValidRange(stateUpdate.id) &&
    isValidStateObject(stateUpdate.newState)
  )
}

/**
 * Validates that the provided Transaction has all of the necessary fields populated with valid data.
 *
 * @param transaction the Transaction to inspect
 * @returns true if valid, false otherwise
 */
export const isValidTransaction = (
  transaction: any
): transaction is Transaction => {
  return (
    !!transaction &&
    transaction.block > 0 &&
    !!transaction.witness &&
    !!transaction.stateUpdate &&
    isValidStateUpdate(transaction.stateUpdate)
  )
}

/**
 * Validates that the provided VerifiedStateUpdate has all of the necessary fields populated with valid data.
 *
 * @param verifiedUpdate the VerifiedStateUpdate to inspect
 * @returns true if valid, false otherwise
 */
export const isValidVerifiedStateUpdate = (
  verifiedUpdate: any
): verifiedUpdate is VerifiedStateUpdate => {
  return (
    !!verifiedUpdate &&
    !!verifiedUpdate.start &&
    !!verifiedUpdate.end &&
    verifiedUpdate.verifiedBlockNumber >= 0 &&
    verifiedUpdate.start.lt(verifiedUpdate.end) &&
    isValidStateUpdate(verifiedUpdate.stateUpdate)
  )
}
