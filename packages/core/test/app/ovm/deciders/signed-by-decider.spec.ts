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
import { SignedByDecider } from '../../../../src/app/ovm/deciders/signed-by-decider'
import { CannotDecideError } from '../../../../src/app/ovm/deciders'
import { SignedByDB } from '../../../../src/app/ovm/db'

describe('SignedByDecider', () => {
  const myPublicKey: Buffer = Buffer.from('key')
  const publicKey: Buffer = Buffer.from('not my key')
  const message: Buffer = Buffer.from('m')
  const signature: Buffer = Buffer.from('m')

  describe('decide', () => {
    let decider: Decider
    let db: DB
    let memdown: any
    let signedByDb: SignedByDB

    beforeEach(() => {
      memdown = new MemDown('')
      db = new BaseDB(memdown, 256)
      signedByDb = new SignedByDB(db)
      decider = new SignedByDecider(signedByDb, myPublicKey)
    })

    afterEach(async () => {
      await db.close()
      memdown = undefined
    })

    it('should return true when signature is verified', async () => {
      await signedByDb.storeSignedMessage(signature, publicKey)

      const decision: Decision = await decider.decide({ publicKey, message })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(justification.implication.input['publicKey'].equals(publicKey))
      assert(justification.implication.input['message'].equals(message))
      assert(justification.implicationWitness['signature'].equals(signature))
    })

    it('should return false if not signed and is my signature', async () => {
      const decision: Decision = await decider.decide({
        publicKey: myPublicKey,
        message,
      })

      decision.outcome.should.equal(false)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(justification.implication.input['publicKey'].equals(myPublicKey))
      assert(justification.implication.input['message'].equals(message))
      assert(justification.implicationWitness['signature'] === undefined)
    })

    it('should throw cannot decide when signature is not verified', async () => {
      try {
        await decider.decide({ publicKey, message })
        assert(false, 'This should have thrown CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          throw e
        }
      }
    })
  })
})
