import '../../../setup'

import MemDown from 'memdown'
import {
  CannotDecideError,
  HashPreimageExistenceDecider,
} from '../../../../src/app/ovm/deciders'
import { BaseDB } from '../../../../src/app/db'
import { Md5Hash } from '../../../../src/app/utils'
import { Decision, ImplicationProofItem } from '../../../../src/types/ovm'
import * as assert from 'assert'
import { DB } from '../../../../src/types/db'

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
    let decider: HashPreimageExistenceDecider
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

      const justification: ImplicationProofItem = decision.justification[0]
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

  describe('decide with cache', () => {
    let decider: HashPreimageExistenceDecider
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

    it('should throw if no decision', async () => {
      try {
        await decider.decide({ hash })
        assert(
          false,
          'No decision should exist for input on which a decision has not been made.'
        )
      } catch (e) {
        assert(
          e instanceof CannotDecideError,
          `Expected error, but got ${JSON.stringify(e)}`
        )
      }
    })

    it('should not return anything if no decision with previous attempt', async () => {
      try {
        await decider.decide({ hash }, { preimage: Buffer.from('womp womp') })
      } catch (e) {
        // No-Op
      }

      try {
        await decider.decide({ hash })
        assert(
          false,
          'No decision should exist for input on which a decision has not been made.'
        )
      } catch (e) {
        assert(
          e instanceof CannotDecideError,
          `Expected error, but not ${JSON.stringify(e)}`
        )
      }
    })

    it('should return Decisions that have been made', async () => {
      await decider.decide({ hash }, { preimage })
      const checkedDecision: Decision = await decider.decide({ hash })

      checkedDecision.outcome.should.equal(true)
      checkedDecision.justification.length.should.equal(1)

      const justification: ImplicationProofItem =
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

      const checkedDecision: Decision = await decider.decide({ hash })

      checkedDecision.outcome.should.equal(true)
      checkedDecision.justification.length.should.equal(1)

      let justification: ImplicationProofItem = checkedDecision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        justification.implication.input['hash'].equals(hash),
        'decided hash is not what it should be'
      )
      assert(
        justification.implicationWitness['preimage'].equals(preimage),
        'decided preimage is not what it should be'
      )

      const secondCheckedDecision: Decision = await decider.decide({
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
