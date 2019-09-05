import { Range } from '../../types/db'
import {
  BlockTransaction,
  StateObject,
  StateUpdate,
  Transaction,
} from '../../types/serialization'

/**
 * All of the below functions check whether or not the two provided objects are equal,
 * returning true if they are and false otherwise
 */

export const areEqual = (one: any, two: any): boolean => {
  if (
    (one === undefined && two === undefined) ||
    (one === null && two === null) ||
    (one === false && two === false)
  ) {
    return true
  }

  if (Array.isArray(one) || Array.isArray(two)) {
    if (
      Array.isArray(one) !== Array.isArray(two) ||
      one.length !== two.length
    ) {
      return false
    }
    for (let i = 0; i < one.length; i++) {
      if (!areEqual(one[i], two[i])) {
        return false
      }
    }
    return true
  }

  if (typeof one === 'object' && typeof two === 'object') {
    return objectsEqual(one, two)
  }

  return one === two
}

export const objectsEqual = (obj1: {}, obj2: {}): boolean => {
  if (!obj1 && !obj2) {
    return true
  }

  if (!obj1 || !obj2) {
    return false
  }

  if (obj1.hasOwnProperty('equals')) {
    return obj1['equals'](obj2)
  }

  const props: string[] = Object.getOwnPropertyNames(obj1)
  if (props.length !== Object.getOwnPropertyNames(obj2).length) {
    return false
  }

  for (const prop of props) {
    if (!obj2.hasOwnProperty(prop)) {
      return false
    }

    if (typeof obj1[prop] === 'object' && typeof obj2[prop] === 'object') {
      if (objectsEqual(obj1[prop], obj2[prop])) {
        continue
      } else {
        return false
      }
    }

    // TODO: This won't work for reference types, but it'll work for now
    if (obj1[prop] !== obj2[prop]) {
      return false
    }
  }

  return true
}

export const rangesEqual = (range1: Range, range2: Range): boolean => {
  return (
    range1 !== undefined &&
    range2 !== undefined &&
    range1.start.eq(range2.start) &&
    range1.end.eq(range2.end)
  )
}

export const stateObjectsEqual = (
  stateObject1: StateObject,
  stateObject2: StateObject
): boolean => {
  return (
    stateObject1 !== undefined &&
    stateObject2 !== undefined &&
    stateObject1.predicateAddress === stateObject2.predicateAddress &&
    objectsEqual(stateObject1.data, stateObject2.data)
  )
}

export const stateUpdatesEqual = (
  stateUpdate1: StateUpdate,
  stateUpdate2: StateUpdate
): boolean => {
  return (
    stateUpdate1 !== undefined &&
    stateUpdate2 !== undefined &&
    stateUpdate1.plasmaBlockNumber.eq(stateUpdate2.plasmaBlockNumber) &&
    stateUpdate1.depositAddress === stateUpdate2.depositAddress &&
    rangesEqual(stateUpdate1.range, stateUpdate2.range) &&
    stateObjectsEqual(stateUpdate1.stateObject, stateUpdate2.stateObject)
  )
}

export const transactionsEqual = (
  tx1: Transaction,
  tx2: Transaction
): boolean => {
  return (
    tx1 !== undefined &&
    tx2 !== undefined &&
    tx1.depositAddress === tx2.depositAddress &&
    rangesEqual(tx1.range, tx2.range) &&
    objectsEqual(tx1.body, tx2.body)
  )
}

export const blockTransactionsEqual = (
  blockTx1: BlockTransaction,
  blockTx2: BlockTransaction
): boolean => {
  return (
    blockTx1 !== undefined &&
    blockTx2 !== undefined &&
    blockTx1.blockNumber.eq(blockTx2.blockNumber) &&
    transactionsEqual(blockTx1.transaction, blockTx2.transaction)
  )
}
