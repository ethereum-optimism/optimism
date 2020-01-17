import '../../setup'

/* External Imports */
import {
  IdentityVerifier,
  ONE,
  serializeObject,
  SignatureVerifier,
  strToHexStr,
} from '@pigi/core-utils'

/* Internal Imports */
import {
  QuantifierResult,
  SignedByDBInterface,
  SignedByQuantifier,
  SignedMessage,
} from '../../../src'

/*******************
 * Mocks & Helpers *
 *******************/

class MockedMessageDB implements SignedByDBInterface {
  private readonly signedMessages: Map<string, SignedMessage[]> = new Map<
    string,
    SignedMessage[]
  >()

  public constructor(
    private readonly signatureVerifier: SignatureVerifier = IdentityVerifier.instance()
  ) {}

  public async handleMessage(
    serializedMessage: string,
    signature: string
  ): Promise<void> {
    if (!!signature) {
      await this.storeSignedMessage(serializedMessage, signature)
    }
  }

  public async storeSignedMessage(
    serializedMessage: string,
    signature: string
  ): Promise<void> {
    const signerAddress: string = await this.signatureVerifier.verifyMessage(
      serializedMessage,
      signature
    )

    if (!this.signedMessages.has(signerAddress)) {
      this.signedMessages.set(signerAddress, [])
    }

    this.signedMessages.get(signerAddress).push({
      serializedMessage,
      signature,
    })
  }

  public async getAllSignedBy(publicKey: string): Promise<SignedMessage[]> {
    return this.signedMessages.has(publicKey)
      ? this.signedMessages.get(publicKey)
      : []
  }

  public async getMessageSignature(
    serializedMessage: string,
    signerPublicKey: string
  ): Promise<string | undefined> {
    if (!this.signedMessages.has(signerPublicKey)) {
      return undefined
    }

    for (const signed of this.signedMessages.get(signerPublicKey)) {
      if (signed.serializedMessage === serializedMessage) {
        return signed.signature
      }
    }

    return undefined
  }
}

/*********
 * TESTS *
 *********/

describe('SignedByQuantifier', () => {
  let db: SignedByDBInterface
  const serializedMessage1: string = strToHexStr(
    serializeObject({
      channelID: '10',
      data: { msg: 'a' },
    })
  )
  const serializedMessage2: string = strToHexStr(
    serializeObject({
      channelID: '10',
      data: { msg: 'b' },
    })
  )
  const myAddress: string = '0xMY_ADDRESS =D'
  const notMyAddress: string = '0xNOT_MY_ADDRESS =|'

  beforeEach(() => {
    db = new MockedMessageDB()
  })

  describe('getAllQuantified without channelID', () => {
    it('returns messages from the DB with my address', async () => {
      await db.storeSignedMessage(serializedMessage1, myAddress)
      await db.storeSignedMessage(serializedMessage2, myAddress)
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: myAddress,
      })
      result.allResultsQuantified.should.equal(true)
      result.results.length.should.equal(2)
      result.results[0].serializedMessage.should.equal(serializedMessage1)
      result.results[0].signature.should.equal(myAddress)
      result.results[1].serializedMessage.should.equal(serializedMessage2)
      result.results[1].signature.should.equal(myAddress)
    })

    it('returns messages from the DB not with my address', async () => {
      await db.storeSignedMessage(serializedMessage1, notMyAddress)
      await db.storeSignedMessage(serializedMessage2, notMyAddress)
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: notMyAddress,
      })
      result.allResultsQuantified.should.equal(false)
      result.results.length.should.equal(2)
      result.results[0].serializedMessage.should.equal(serializedMessage1)
      result.results[0].signature.should.equal(notMyAddress)
      result.results[1].serializedMessage.should.equal(serializedMessage2)
      result.results[1].signature.should.equal(notMyAddress)
    })

    it('returns empty list from DB with my address', async () => {
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: myAddress,
      })
      result.allResultsQuantified.should.equal(true)
      result.results.length.should.equal(0)
    })

    it('returns empty list from the DB not with my address', async () => {
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: notMyAddress,
      })
      result.allResultsQuantified.should.equal(false)
      result.results.length.should.equal(0)
    })
  })

  describe('getAllQuantified with channelID', () => {
    const channelID: string = '10'
    it('returns messages from the DB with my address', async () => {
      const serializedMessage: string = strToHexStr(
        serializeObject({
          channelID,
          nonce: ONE,
          data: {},
        })
      )

      const serializedMessageTwo: string = strToHexStr(
        serializeObject({
          channelID: 'not the channel',
          nonce: ONE,
          data: {},
        })
      )

      await db.storeSignedMessage(serializedMessage, myAddress)
      await db.storeSignedMessage(serializedMessageTwo, myAddress)
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: myAddress,
        channelID,
      })
      result.allResultsQuantified.should.equal(true)
      result.results.length.should.equal(1)
      result.results[0].should.deep.equal({
        serializedMessage,
        signature: myAddress,
      })
    })
  })
})
