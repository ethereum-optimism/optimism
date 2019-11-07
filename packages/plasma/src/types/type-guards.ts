/* External Imports */
import { isValidRange, ZERO } from '@pigi/core-utils'

/* Internal Imports */
import {
  StateObject,
  StateUpdate,
  Transaction,
  VerifiedStateUpdate,
} from './state.interface'

/**
 * Validates that the provided StateObject has all of the necessary fields populated with valid data.
 *
 * @param stateObject the StateObject to inspect
 * @returns true if valid, false otherwise
 */
export const isValidStateObject = (
  stateObject: any
): stateObject is StateObject => {
  return !!stateObject.predicateAddress && !!stateObject.data
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
    !!stateUpdate.stateObject &&
    !!stateUpdate.range &&
    !!stateUpdate.depositAddress &&
    stateUpdate.plasmaBlockNumber.gt(ZERO) &&
    isValidRange(stateUpdate.range) &&
    isValidStateObject(stateUpdate.stateObject)
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
    !!transaction.range &&
    !!transaction.depositAddress &&
    isValidRange(transaction.range)
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
    !!verifiedUpdate.range &&
    verifiedUpdate.verifiedBlockNumber.gte(ZERO) &&
    isValidRange(verifiedUpdate.range) &&
    isValidStateUpdate(verifiedUpdate.stateUpdate)
  )
}
