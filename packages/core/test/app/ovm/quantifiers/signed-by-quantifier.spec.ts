import '../../../setup'

import { Message, ONE, SignedMessage } from '../../../../src/types'
import { SignedByQuantifier } from '../../../../src/app/ovm/quantifiers/signed-by-quantifier'
import { QuantifierResult } from '../../../../src/types/ovm'
import { SignedByDBInterface } from '../../../../src/types/ovm/db/signed-by-db.interface'
import { decryptWithPublicKey, sign } from '../../../../src/app/utils'
import { messageToBuffer } from '../../../../src/app/serialization'

/*******************
 * Mocks & Helpers *
 *******************/

class MockedMessageDB implements SignedByDBInterface {
  private readonly signedMessages: {} = {}

  public async handleMessage(
    message: Message,
    signedMessage?: SignedMessage
  ): Promise<void> {
    if (!!signedMessage) {
      await this.storeSignedMessage(
        signedMessage.signedMessage,
        signedMessage.sender
      )
    }
  }

  public async storeSignedMessage(
    signature: Buffer,
    signerPublicKey: Buffer
  ): Promise<void> {
    const keyString: string = signerPublicKey.toString()
    if (!(keyString in this.signedMessages)) {
      this.signedMessages[keyString] = []
    }

    this.signedMessages[keyString].push(signature)
  }

  public async getAllSignedBy(publicKey: Buffer): Promise<Buffer[]> {
    const keyString: string = publicKey.toString()
    return keyString in this.signedMessages
      ? this.signedMessages[keyString]
      : []
  }

  public async getMessageSignature(
    message: Buffer,
    signerPublicKey
  ): Promise<Buffer | undefined> {
    const keyString: string = signerPublicKey.toString()
    if (!(keyString in this.signedMessages)) {
      return undefined
    }

    for (const signed of this.signedMessages[keyString]) {
      if (decryptWithPublicKey(signerPublicKey, signed).equals(message)) {
        return signed
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
  const myPrivateKey: Buffer = Buffer.from('my PK')
  const message1: Buffer = Buffer.from('a')
  const message2: Buffer = Buffer.from('b')
  const myAddress: Buffer = Buffer.from('0xMY_ADDRESS =D')
  const notMyAddress: Buffer = Buffer.from('0xNOT_MY_ADDRESS =|')

  beforeEach(() => {
    db = new MockedMessageDB()
  })

  describe('getAllQuantified without channelID', () => {
    it('returns messages from the DB with my address', async () => {
      await db.storeSignedMessage(message1, myAddress)
      await db.storeSignedMessage(message2, myAddress)
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: myAddress,
      })
      result.allResultsQuantified.should.equal(true)
      result.results.length.should.equal(2)
      result.results[0].should.equal(message1)
      result.results[1].should.equal(message2)
    })

    it('returns messages from the DB not with my address', async () => {
      await db.storeSignedMessage(message1, notMyAddress)
      await db.storeSignedMessage(message2, notMyAddress)
      const quantifier: SignedByQuantifier = new SignedByQuantifier(
        db,
        myAddress
      )

      const result: QuantifierResult = await quantifier.getAllQuantified({
        address: notMyAddress,
      })
      result.allResultsQuantified.should.equal(false)
      result.results.length.should.equal(2)
      result.results[0].should.equal(message1)
      result.results[1].should.equal(message2)
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
    const channelID: Buffer = Buffer.from('ChannelID')
    it('returns messages from the DB with my address', async () => {
      const message: Message = {
        channelID,
        nonce: ONE,
        data: {},
      }

      const secondMessage: Message = {
        channelID: Buffer.from('not the channel'),
        nonce: ONE,
        data: {},
      }

      const signedMessage: Buffer = sign(myPrivateKey, messageToBuffer(message))
      const secondSignedMessage: Buffer = sign(
        myPrivateKey,
        messageToBuffer(secondMessage)
      )

      await db.storeSignedMessage(signedMessage, myAddress)
      await db.storeSignedMessage(secondSignedMessage, myAddress)
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
      result.results[0].should.equal(signedMessage)
    })
  })
})
