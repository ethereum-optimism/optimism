import '../../../setup'

import MemDown from 'memdown'
import {
  CannotDecideError,
  HashPreimageExistenceDecider,
} from '../../../../src/app/ovm/deciders'
import { BaseDB } from '../../../../src/app/db'
import { Md5Hash } from '../../../../src/app/utils'
import {
  Decider,
  Decision,
  ImplicationProofElement,
} from '../../../../src/types/ovm/decider.interface'
import * as assert from 'assert'
import { DB } from '../../../../src/types/db'

/*********
 * TESTS *
 *********/

describe('HashPreimageExistenceDecider', () => {
  const preimage: Buffer = Buffer.from('really great preimage')
  const hash: Buffer = Md5Hash(preimage)

  describe('Constructor', () => {
    it('should initialize', async () => {
      new HashPreimageExistenceDecider(
        new BaseDB(new MemDown('') as any, 256),
        Md5Hash
      )
    })
  })

  describe('decide', () => {
    let decider: Decider
    let db: DB
    let memdown: any

    beforeEach(() => {
      memdown = new MemDown('')
      db = new BaseDB(memdown, 256)
      decider = new HashPreimageExistenceDecider(db, Md5Hash)
    })

    afterEach(async () => {
      await db.close()
      memdown = undefined
    })

    it('should decide true for valid preimage', async () => {
      const decision: Decision = await decider.decide({ hash }, { preimage })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofElement = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      justification.implication.input['hash'].should.equal(hash)
      justification.implicationWitness['preimage'].should.equal(preimage)
    })

    it('should throw invalid preimage', async () => {
      try {
        await decider.decide({ hash }, { preimage: Buffer.from('womp womp') })
      } catch (e) {
        if (e instanceof CannotDecideError) {
          assert(true, 'This is expected to happen')
        } else {
          assert(false, 'CannotDecideError was expected.')
        }
      }
    })
  })

  describe('checkDecision', () => {
    let decider: Decider
    let db: DB
    let memdown: any

    beforeEach(() => {
      memdown = new MemDown('')
      db = new BaseDB(memdown, 256)
      decider = new HashPreimageExistenceDecider(db, Md5Hash)
    })

    afterEach(async () => {
      await db.close()
      memdown = undefined
    })

    it('should not return anything if no decision', async () => {
      assert(
        (await decider.checkDecision({ hash })) === undefined,
        '' +
          'No decision should exist for input on which a decision has not been made.'
      )
    })

    it('should not return anything if no decision with previous attempt', async () => {
      try {
        await decider.decide({ hash }, { preimage: Buffer.from('womp womp') })
      } catch (e) {
        // No-Op
      }

      assert(
        (await decider.checkDecision({ hash })) === undefined,
        'No decision should exist for input on which a decision has not been made.'
      )
    })

    it('should return Decisions that have been made', async () => {
      await decider.decide({ hash }, { preimage })
      const checkedDecision: Decision = await decider.checkDecision({ hash })

      checkedDecision.outcome.should.equal(true)
      checkedDecision.justification.length.should.equal(1)

      const justification: ImplicationProofElement =
        checkedDecision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        justification.implication.input['hash'].equals(hash),
        'decided hash is not what it should be'
      )
      assert(
        justification.implicationWitness['preimage'].equals(preimage),
        'decided preimage is not what it should be'
      )
    })

    it('should work with multiple Decisions that have been made', async () => {
      await decider.decide({ hash }, { preimage })
      const secondPreimage: Buffer = Buffer.from('Another great preimage!')
      const secondHash: Buffer = Md5Hash(secondPreimage)

      await decider.decide({ hash }, { preimage })
      await decider.decide({ hash: secondHash }, { preimage: secondPreimage })

      const checkedDecision: Decision = await decider.checkDecision({ hash })

      checkedDecision.outcome.should.equal(true)
      checkedDecision.justification.length.should.equal(1)

      let justification: ImplicationProofElement =
        checkedDecision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        justification.implication.input['hash'].equals(hash),
        'decided hash is not what it should be'
      )
      assert(
        justification.implicationWitness['preimage'].equals(preimage),
        'decided preimage is not what it should be'
      )

      const secondCheckedDecision: Decision = await decider.checkDecision({
        hash: secondHash,
      })

      secondCheckedDecision.outcome.should.equal(true)
      secondCheckedDecision.justification.length.should.equal(1)

      justification = secondCheckedDecision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        justification.implication.input['hash'].equals(secondHash),
        'second decided hash is not what it should be'
      )
      assert(
        justification.implicationWitness['preimage'].equals(secondPreimage),
        'second decided preimage is not what it should be'
      )
    })
  })
})
