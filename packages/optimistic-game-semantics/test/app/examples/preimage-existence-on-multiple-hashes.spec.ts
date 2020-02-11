import '../../setup'

import {
  HashAlgorithm,
  HashFunction,
  keccak256,
} from '@eth-optimism/core-utils'
import { newInMemoryDB } from '@eth-optimism/core-db'
import * as assert from 'assert'

import {
  CannotDecideError,
  ForAllSuchThatDecider,
  ForAllSuchThatInput,
  HashPreimageExistenceDecider,
  IntegerRangeQuantifier,
  HashPreimageDBInterface,
  HashPreimageDB,
  Decision,
  PropertyFactory,
} from '../../../src'

describe('PreimageExistenceOnRangeOfHashes', () => {
  const forAllDecider: ForAllSuchThatDecider = new ForAllSuchThatDecider()
  const rangeQuantifier: IntegerRangeQuantifier = new IntegerRangeQuantifier()
  const hashAlgorithm: HashAlgorithm = HashAlgorithm.KECCAK256
  const hashFunction: HashFunction = keccak256

  let hashDecider: HashPreimageExistenceDecider
  let preimageDB: HashPreimageDBInterface

  beforeEach(() => {
    preimageDB = new HashPreimageDB(newInMemoryDB())
    hashDecider = new HashPreimageExistenceDecider(preimageDB, hashAlgorithm)
  })

  const savePreimages = async (numbers: number[]) => {
    for (const num of numbers) {
      await preimageDB.storePreimage(num.toString(), hashAlgorithm)
    }
  }

  describe('decide', () => {
    it('should decide true when preimages produce hashes', async () => {
      await savePreimages([2, 3, 4, 5, 6, 7, 8])

      const propertyFactory: PropertyFactory = (num: number) => {
        return {
          decider: hashDecider,
          input: {
            hash: hashFunction(num.toString()),
          },
        }
      }

      const forAllInput: ForAllSuchThatInput = {
        quantifier: rangeQuantifier,
        quantifierParameters: { start: 2, end: 8 },
        propertyFactory,
      }

      const decision: Decision = await forAllDecider.decide(forAllInput)
      decision.outcome.should.eq(true)
    })

    it('should return cannot decide when a single preimage does not produce the correct hash', async () => {
      await savePreimages([2, 3, 4, /* no 5 */ 6, 7, 8])
      const propertyFactory: PropertyFactory = (num: number) => {
        return {
          decider: hashDecider,
          input: {
            hash: hashFunction(num.toString()),
          },
        }
      }

      const forAllInput: ForAllSuchThatInput = {
        quantifier: rangeQuantifier,
        quantifierParameters: { start: 2, end: 8 },
        propertyFactory,
      }

      try {
        await forAllDecider.decide(forAllInput, undefined)
        assert(false, 'Should have thrown.')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          throw e
        }
      }
    })
  })
})
