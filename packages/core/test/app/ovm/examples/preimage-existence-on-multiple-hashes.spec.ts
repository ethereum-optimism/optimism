import '../../../setup'

import MemDown from 'memdown'

import {
  CannotDecideError,
  ForAllSuchThatDecider,
  ForAllSuchThatInput,
  HashPreimageExistenceDecider,
} from '../../../../src/app/ovm/deciders'
import { BaseDB } from '../../../../src/app/db'
import { keccak256, Md5Hash } from '../../../../src/app/utils'
import { DB } from '../../../../src/types/db'
import { IntegerRangeQuantifier } from '../../../../src/app/ovm/quantifiers'
import {
  Decision,
  HashPreimageDbInterface,
  PropertyFactory,
  WitnessFactory,
} from '../../../../src/types/ovm'
import * as assert from 'assert'
import { HashAlgorithm, HashFunction } from '../../../../src/types/utils'
import { HashPreimageDb } from '../../../../src/app/ovm/db/hash-preimage-db'

describe('PreimageExistenceOnRangeOfHashes', () => {
  const forAllDecider: ForAllSuchThatDecider = new ForAllSuchThatDecider()
  const rangeQuantifier: IntegerRangeQuantifier = new IntegerRangeQuantifier()
  const hashAlgorithm: HashAlgorithm = HashAlgorithm.KECCAK256
  const hashFunction: HashFunction = keccak256

  let hashDecider: HashPreimageExistenceDecider
  let preimageDB: HashPreimageDbInterface
  let db: DB
  let memdown: any

  beforeEach(() => {
    memdown = new MemDown('')
    db = new BaseDB(memdown, 256)
    preimageDB = new HashPreimageDb(db)
    hashDecider = new HashPreimageExistenceDecider(preimageDB, hashAlgorithm)
  })

  afterEach(async () => {
    await db.close()
    memdown = undefined
  })

  const savePreimages = async (numbers: number[]) => {
    for (const num of numbers) {
      await preimageDB.storePreimage(Buffer.of(num), hashAlgorithm)
    }
  }

  describe('decide', () => {
    it('should decide true when preimages produce hashes', async () => {
      await savePreimages([2, 3, 4, 5, 6, 7, 8])

      const propertyFactory: PropertyFactory = (num: number) => {
        return {
          decider: hashDecider,
          input: {
            hash: hashFunction(Buffer.of(num)),
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
            hash: hashFunction(Buffer.of(num)),
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
