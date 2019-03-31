import { should } from '../../../setup'

/* External Imports */
import BigNum = require('bn.js')
import { StateObject } from '@pigi/utils'

/* Internal Imports */
import { StateManager } from '../../../../src/app/common/utils/state-manager'

/**
 * Checks if two StateManager objects are equal.
 * @param a First StateManger.
 * @param b Second StateManager.
 * @returns `true` if the two are equal, `false` otherwise.
 */
const equals = (a: StateManager, state: StateObject[]): boolean => {
  for (const elA of a.state) {
    for (const elB of state) {
      if (elA.equals(elB)) {
        return false
      }
    }
  }
  return true
}

describe('StateManager', () => {
  const deposit = new StateObject({
    start: new BigNum(0),
    end: new BigNum(100),
    block: new BigNum(0),
    predicate: null,
    state: null,
  })

  let stateManager: StateManager
  beforeEach(() => {
    stateManager = new StateManager()
  })

  describe('addStateObject', () => {
    it('should be able to apply a deposit', () => {
      stateManager.addStateObject(deposit)
      equals(stateManager, [deposit]).should.be.true
    })

    it('should not apply a deposit with start greater than end', () => {
      const badDeposit = new StateObject({
        start: new BigNum(100),
        end: new BigNum(0),
        block: new BigNum(0),
        predicate: null,
        state: null,
      })

      should.Throw(() => {
        stateManager.addStateObject(badDeposit)
      }, 'Invalid StateObject')
    })
  })

  describe('applyStateObject', () => {
    it('should be able to apply a valid transaction', () => {
      const expected = new StateObject({
        start: new BigNum(0),
        end: new BigNum(100),
        block: new BigNum(1),
        predicate: null,
        state: null,
      })

      stateManager.addStateObject(deposit)
      stateManager.applyStateObject(expected)

      equals(stateManager, [expected]).should.be.true
    })

    it('should apply a transaction that goes under an existing range', () => {
      const stateObject = new StateObject({
        start: new BigNum(0),
        end: new BigNum(50),
        block: new BigNum(1),
        predicate: null,
        state: null,
      })
      const expected = [
        new StateObject({
          start: new BigNum(0),
          end: new BigNum(50),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
        new StateObject({
          start: new BigNum(50),
          end: new BigNum(100),
          block: new BigNum(0),
          predicate: null,
          state: null,
        }),
      ]

      stateManager.addStateObject(deposit)
      stateManager.applyStateObject(stateObject)

      equals(stateManager, expected).should.be.true
    })

    it('should apply a transaction with implicit start and ends', () => {
      const stateObject = new StateObject({
        start: new BigNum(25),
        end: new BigNum(75),
        implicitStart: new BigNum(0),
        implicitEnd: new BigNum(100),
        block: new BigNum(1),
        predicate: null,
        state: null,
      })
      const expected = [
        new StateObject({
          start: new BigNum(0),
          end: new BigNum(25),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
        new StateObject({
          start: new BigNum(25),
          end: new BigNum(75),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
        new StateObject({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
      ]

      stateManager.addStateObject(deposit)
      stateManager.applyStateObject(stateObject)

      equals(stateManager, expected).should.be.true
    })

    it('should apply a transaction with only an implicit end', () => {
      const stateObject = new StateObject({
        start: new BigNum(0),
        end: new BigNum(75),
        implicitEnd: new BigNum(100),
        block: new BigNum(1),
        predicate: null,
        state: null,
      })
      const expected = [
        new StateObject({
          start: new BigNum(0),
          end: new BigNum(75),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
        new StateObject({
          start: new BigNum(75),
          end: new BigNum(100),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
      ]

      stateManager.addStateObject(deposit)
      stateManager.applyStateObject(stateObject)

      equals(stateManager, expected).should.be.true
    })

    it('should apply a transaction where only an implicit part overlaps', () => {
      const deposit2 = new StateObject({
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
        predicate: null,
        state: null,
      })
      const transaction = new StateObject({
        start: new BigNum(100),
        end: new BigNum(200),
        implicitStart: new BigNum(0),
        block: new BigNum(1),
        predicate: null,
        state: null,
      })
      const expected = [
        new StateObject({
          start: new BigNum(0),
          end: new BigNum(100),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
        new StateObject({
          start: new BigNum(100),
          end: new BigNum(200),
          block: new BigNum(1),
          predicate: null,
          state: null,
        }),
      ]

      stateManager.addStateObject(deposit)
      stateManager.addStateObject(deposit2)
      stateManager.applyStateObject(transaction)

      equals(stateManager, expected).should.be.true
    })
  })
})
