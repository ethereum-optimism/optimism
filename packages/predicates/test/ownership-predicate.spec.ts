import BigNum = require('bn.js')

import './setup'
import { OwnershipPredicatePlugin } from '../src/plugins/ownership-predicate'
import {
  ONE,
  OwnershipBody,
  Range,
  StateObject,
  StateUpdate,
  Transaction,
  stateObjectsEqual,
} from '@pigi/core'
import * as assert from 'assert'

/***********
 * HELPERS *
 ***********/

const defaultDepositAddress: string = '0x11111111111111111'
const defaultPredicateAddress: string = '0x123456789abcdef'
const defaultOwner: string = '0x999999999999999'
const newOwner: string = '0x8888888888888'

// TODO: Change this when we are actually checking signatures
const defaultWitness: string = defaultOwner

const defaultInBlock: BigNum = new BigNum(2)
const defaultOriginBlock: BigNum = defaultInBlock
const defaultMaxBlock: BigNum = new BigNum(10)
const defaultCurrentBlock: BigNum = ONE

const defaultRange: Range = {
  start: ONE,
  end: new BigNum(10),
}

const getStateObject = (
  owner: string = defaultOwner,
  predicateAddress: string = defaultPredicateAddress
): StateObject => {
  return {
    predicateAddress,
    data: {
      owner,
    },
  }
}

const getStateUpdate = (
  stateObject: StateObject,
  plasmaBlockNumber: BigNum,
  range: Range = defaultRange,
  depositAddress: string = defaultDepositAddress
): StateUpdate => {
  return {
    range,
    stateObject,
    depositAddress,
    plasmaBlockNumber,
  }
}

const getTransactionBody = (
  newState: StateObject,
  originBlock: BigNum = defaultOriginBlock,
  maxBlock: BigNum = defaultMaxBlock
): OwnershipBody => {
  return {
    newState,
    originBlock,
    maxBlock,
  }
}

const getTransaction = (
  body: any,
  depositAddress: string = defaultDepositAddress,
  range: Range = defaultRange
) => {
  return {
    depositAddress,
    range,
    body,
  }
}

/*********
 * TESTS *
 *********/

describe('OwnershipPredicate', async () => {
  const ownershipPredicate: OwnershipPredicatePlugin = new OwnershipPredicatePlugin()
  const defaultStateObject: StateObject = {
    predicateAddress: defaultDepositAddress,
    data: {
      owner: defaultOwner,
    },
  }

  describe('getOwner', async () => {
    it('should get owner when present', async () => {
      ownershipPredicate.getOwner(defaultStateObject).should.equal(defaultOwner)
    })

    it('should return undefined when owner not present', async () => {
      const stateObject: any = { predicateAddress: defaultPredicateAddress }

      assert(ownershipPredicate.getOwner(stateObject) === undefined)
    })
  })

  describe('executeStateTransition', async () => {
    const defaultPreviousStateObject: StateObject = getStateObject()
    const defaultPreviousStateUpdate: StateUpdate = getStateUpdate(
      defaultPreviousStateObject,
      defaultCurrentBlock
    )
    const defaultNewState: StateObject = getStateObject(newOwner)
    const defaultTransactionBody = getTransactionBody(defaultNewState)
    const defaultTransaction: Transaction = getTransaction(
      defaultTransactionBody
    )

    it('should return expected StateUpdate with valid input', async () => {
      const stateObject: StateObject = await ownershipPredicate.executeStateTransition(
        defaultPreviousStateUpdate,
        defaultTransaction,
        defaultWitness
      )

      assert(stateObjectsEqual(stateObject, defaultNewState))
    })
  })
})
