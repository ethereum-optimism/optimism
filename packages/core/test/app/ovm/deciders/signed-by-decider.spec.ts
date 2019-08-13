import '../../../setup'

import MemDown from 'memdown'
import { BaseDB } from '../../../../src/app/db'
import {
  Decider,
  Decision,
  ImplicationProofItem,
} from '../../../../src/types/ovm'
import * as assert from 'assert'
import { DB } from '../../../../src/types/db'
import {
  SignatureVerifier,
  SignedByDecider,
} from '../../../../src/app/ovm/deciders/signed-by-decider'
import { CannotDecideError } from '../../../../src/app/ovm/deciders'

describe('SignedByDecider', () => {
  const publicKey: Buffer = Buffer.from('key')
  const message: Buffer = Buffer.from('m')
  const signature: Buffer = Buffer.from('s')
  const trueSignatureVerifier: SignatureVerifier = async (
    a: any,
    b: any,
    c: any
  ) => true
  const falseSignatureVerifier: SignatureVerifier = async (
    a: any,
    b: any,
    c: any
  ) => false
  const throwSignatureVerifier: SignatureVerifier = async (
    a: any,
    b: any,
    c: any
  ) => {
    throw Error('Whooooops!')
  }

  describe('decide', () => {
    let decider: Decider
    let db: DB
    let memdown: any

    beforeEach(() => {
      memdown = new MemDown('')
      db = new BaseDB(memdown, 256)
    })

    afterEach(async () => {
      await db.close()
      memdown = undefined
    })

    it('should return true when signature is verified', async () => {
      decider = new SignedByDecider(db, trueSignatureVerifier)
      const decision: Decision = await decider.decide(
        { publicKey, message },
        { signature }
      )

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      justification.implication.input['publicKey'].should.equal(publicKey)
      justification.implication.input['message'].should.equal(message)
      justification.implicationWitness['signature'].should.equal(signature)
    })

    it('should throw cannot decide when signature is not verified', async () => {
      decider = new SignedByDecider(db, falseSignatureVerifier)
      try {
        await decider.decide({ publicKey, message }, { signature })
        assert(false, 'This should have thrown CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(false, 'This should have thrown CannotDecideError')
        }
      }
    })

    it('should throw if signature checker throws', async () => {
      decider = new SignedByDecider(db, throwSignatureVerifier)

      try {
        await decider.decide({ publicKey, message }, { signature })
        assert(false, 'this should have thrown')
      } catch (e) {
        // success!
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
    })

    afterEach(async () => {
      await db.close()
      memdown = undefined
    })

    it('should return saved decision if true', async () => {
      decider = new SignedByDecider(db, trueSignatureVerifier)
      const decision: Decision = await decider.decide(
        { publicKey, message },
        { signature }
      )

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      justification.implication.input['publicKey'].should.equal(publicKey)
      justification.implication.input['message'].should.equal(message)
      justification.implicationWitness['signature'].should.equal(signature)

      const checkedDecision: Decision = await decider.decide({
        publicKey,
        message,
      })

      checkedDecision.outcome.should.equal(true)
      checkedDecision.justification.length.should.equal(1)

      const checkedJustification: ImplicationProofItem =
        checkedDecision.justification[0]
      checkedJustification.implication.decider.should.equal(decider)
      assert(
        checkedJustification.implication.input['publicKey'].equals(publicKey)
      )
      assert(checkedJustification.implication.input['message'].equals(message))
      assert(
        checkedJustification.implicationWitness['signature'].equals(signature)
      )
    })

    it('should throw cannot decide when signature is not verified', async () => {
      decider = new SignedByDecider(db, falseSignatureVerifier)
      try {
        await decider.decide({ publicKey, message }, { signature })
        assert(false, 'This should have thrown CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(false, 'This should have thrown CannotDecideError')
        }
      }

      try {
        await decider.decide({ publicKey, message })
        assert(false, 'This should have thrown a CannotDecideError.')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          throw Error(`Expected CannotDecideError. Got: ${e}`)
        }
      }
    })
  })
})
