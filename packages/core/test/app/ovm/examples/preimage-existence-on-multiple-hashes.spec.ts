import '../../../setup'

import {
  CannotDecideError,
  ForAllSuchThatDecider,
  ForAllSuchThatInput,
  HashPreimageExistenceDecider,
} from '../../../../src/app/ovm/deciders'
import { newInMemoryDB } from '../../../../src/app/db'
import { keccak256 } from '../../../../src/app/utils'
import { IntegerRangeQuantifier } from '../../../../src/app/ovm/quantifiers'
import {
  Decision,
  HashPreimageDBInterface,
  PropertyFactory,
} from '../../../../src/types/ovm'
import * as assert from 'assert'
import { HashAlgorithm, HashFunction } from '../../../../src/types/utils'
import { HashPreimageDB } from '../../../../src/app/ovm/db/hash-preimage-db'

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
