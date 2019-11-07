import '../../setup'

import { IdentityVerifier, serializeObject } from '@pigi/core-utils'
import { newInMemoryDB } from '@pigi/core-db'
import * as assert from 'assert'

import {
  SignedByDBInterface,
  Decider,
  Decision,
  ImplicationProofItem,
  SignedByDecider,
  CannotDecideError,
  SignedByDB,
} from '../../../src'

describe('SignedByDecider', () => {
  const publicKey: string = 'not my key'
  const serializedMessage: string = serializeObject({ msg: 'm' })

  describe('decide', () => {
    let decider: Decider
    let signedByDb: SignedByDBInterface

    beforeEach(() => {
      signedByDb = new SignedByDB(newInMemoryDB(), IdentityVerifier.instance())
      decider = new SignedByDecider(signedByDb, publicKey)
    })

    it('should return true when signature is verified', async () => {
      await signedByDb.storeSignedMessage(serializedMessage, publicKey)

      const decision: Decision = await decider.decide({
        publicKey,
        serializedMessage,
      })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(justification.implication.input['publicKey'] === publicKey)
      assert(
        justification.implication.input['serializedMessage'] ===
          serializedMessage
      )
      assert(justification.implicationWitness['signature'] === publicKey)
    })

    it('should return false if not signed and is my signature', async () => {
      const decision: Decision = await decider.decide({
        publicKey,
        serializedMessage,
      })

      decision.outcome.should.equal(false)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(justification.implication.input['publicKey'] === publicKey)
      assert(justification.implicationWitness['signature'] === undefined)
      assert(
        justification.implication.input['serializedMessage'] ===
          serializedMessage
      )
    })

    it('should throw cannot decide when signature is not verified', async () => {
      try {
        await decider.decide({ publicKey: '', serializedMessage })
        assert(false, 'This should have thrown CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          throw e
        }
      }
    })
  })
})
