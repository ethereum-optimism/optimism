import '../../../setup'

import { MessageDB } from '../../../../src/types/ovm/db'
import { BigNumber, ParsedMessage } from '../../../../src/types'
import { SignedByQuantifier } from '../../../../src/app/ovm/quantifiers/signed-by-quantifier'
import { QuantifierResult } from '../../../../src/types/ovm'
import { Message } from '../../../../src/types/serialization'

/*******************
 * Mocks & Helpers *
 *******************/

class MockedMessageDB implements MessageDB {
  public async getMessageByChannelIdAndNonce(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    return undefined
  }

  public async getMessagesByRecipient(
    address: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    return undefined
  }

  public async getMessagesBySender(
    address: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    return undefined
  }

  public async getMessagesSignedBy(
    signer: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    return undefined
  }

  public async getConflictingCounterpartyMessage(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    return undefined
  }

  public async storeMessage(message: ParsedMessage): Promise<void> {
    return undefined
  }

  public getMyAddress(): Buffer {
    return undefined
  }
}

const getMessageDBThatReturns = (
  messages: ParsedMessage[]
): MockedMessageDB => {
  const db: MockedMessageDB = new MockedMessageDB()
  db.getMessagesSignedBy = async (
    signer: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ) => messages
  return db
}

/*********
 * TESTS *
 *********/

describe('SignedByQuantifier', () => {
  describe('getAllQuantified', () => {
    const myAddress: Buffer = Buffer.from('0xMY_ADDRESS =D')
    const notMyAddress: Buffer = Buffer.from('0xNOT_MY_ADDRESS =|')
    const mySignatures: {} = {}
    mySignatures[myAddress.toString()] = Buffer.from('My Signature')

    it('returns messages from the DB with my address', async () => {
      const message1: ParsedMessage = {
        sender: Buffer.from('sender'),
        recipient: Buffer.from('recipient'),
        message: {
          channelId: Buffer.from('channel'),
          data: {},
        },
        signatures: mySignatures,
      }

      const message2: ParsedMessage = {
        sender: Buffer.from('sender'),
        recipient: Buffer.from('recipient'),
        message: {
          channelId: Buffer.from('channel'),
          data: {},
        },
        signatures: mySignatures,
      }
      const messages: ParsedMessage[] = [message1, message2]
      const db: MockedMessageDB = getMessageDBThatReturns(messages)
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
      const message1: ParsedMessage = {
        sender: Buffer.from('sender'),
        recipient: Buffer.from('recipient'),
        message: {
          channelId: Buffer.from('channel'),
          data: {},
        },
        signatures: mySignatures,
      }

      const message2: ParsedMessage = {
        sender: Buffer.from('sender'),
        recipient: Buffer.from('recipient'),
        message: {
          channelId: Buffer.from('channel'),
          data: {},
        },
        signatures: mySignatures,
      }
      const messages: ParsedMessage[] = [message1, message2]
      const db: MockedMessageDB = getMessageDBThatReturns(messages)
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
      const db: MockedMessageDB = getMessageDBThatReturns([])
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
      const db: MockedMessageDB = getMessageDBThatReturns([])
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
})
