import '../../../setup'

import MemDown from 'memdown'

import {
  CannotDecideError,
  ForAllSuchThatDecider,
  ForAllSuchThatInput,
  HashPreimageExistenceDecider,
} from '../../../../src/app/ovm/deciders'
import { BaseDB } from '../../../../src/app/db'
import { Md5Hash } from '../../../../src/app/utils'
import { DB } from '../../../../src/types/db'
import { IntegerRangeQuantifier } from '../../../../src/app/ovm/quantifiers'
import {
  Decision,
  PropertyFactory,
  WitnessFactory,
} from '../../../../src/types/ovm'
import * as assert from 'assert'

describe('PreimageExistenceOnRangeOfHashes', () => {
  const forAllDecider: ForAllSuchThatDecider = new ForAllSuchThatDecider()
  const rangeQuantifier: IntegerRangeQuantifier = new IntegerRangeQuantifier()

  let hashDecider: HashPreimageExistenceDecider
  let db: DB
  let memdown: any

  beforeEach(() => {
    memdown = new MemDown('')
    db = new BaseDB(memdown, 256)
    hashDecider = new HashPreimageExistenceDecider(db, Md5Hash)
  })

  afterEach(async () => {
    await db.close()
    memdown = undefined
  })

  describe('decide', () => {
    it('should decide true when preimages produce hashes', async () => {
      const witnessFactory: WitnessFactory = (num: number) => {
        return { preimage: Buffer.of(num) }
      }
      const propertyFactory: PropertyFactory = (num: number) => {
        return {
          decider: hashDecider,
          input: {
            hash: Md5Hash(Buffer.of(num)),
          },
        }
      }

      const forAllInput: ForAllSuchThatInput = {
        quantifier: rangeQuantifier,
        quantifierParameters: { start: 2, end: 8 },
        witnessFactory,
        propertyFactory,
      }

      const decision: Decision = await forAllDecider.decide(
        forAllInput,
        undefined
      )
      decision.outcome.should.eq(true)
    })

    it('should return cannot decide when a single preimage does not produce the correct hash', async () => {
      const witnessFactory: WitnessFactory = (num: number) => {
        return {
          preimage:
            num !== 4 ? Buffer.of(num) : Buffer.from('Definitely not 4'),
        }
      }
      const propertyFactory: PropertyFactory = (num: number) => {
        return {
          decider: hashDecider,
          input: {
            hash: Md5Hash(Buffer.of(num)),
          },
        }
      }

      const forAllInput: ForAllSuchThatInput = {
        quantifier: rangeQuantifier,
        quantifierParameters: { start: 2, end: 8 },
        witnessFactory,
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
