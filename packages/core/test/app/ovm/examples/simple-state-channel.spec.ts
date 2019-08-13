import '../../../setup'

import MemDown from 'memdown'

import {
  AndDecider,
  ForAllSuchThatDecider,
  MessageNonceLessThanDecider,
  MessageNonceLessThanInput,
  Utils,
} from '../../../../src/app/ovm/deciders'
import { BaseDB } from '../../../../src/app/db'
import { BigNumber, objectsEqual, ONE } from '../../../../src/app/utils'
import { DB } from '../../../../src/types/db'
import {
  ImplicationProofItem,
  StateChannelMessageDB,
} from '../../../../src/types/ovm'
import {
  ParsedMessage,
  SignedMessage,
} from '../../../../src/types/serialization'
import { StateChannelClient } from '../../../../src/app/ovm/examples'
import { SignedByDecider } from '../../../../src/app/ovm/deciders/signed-by-decider'
import { SignedByQuantifier } from '../../../../src/app/ovm/quantifiers/signed-by-quantifier'
import {
  AddressBalance,
  parseStateChannelSignedMessage,
  StateChannelExitClaim,
  StateChannelMessage,
} from '../../../../src/app/serialization/examples'
import * as assert from 'assert'
import {
  messageToBuffer,
  stateChannelMessageToString,
} from '../../../../src/app/serialization'

class TestStateChannelMessageDB implements StateChannelMessageDB {
  private readonly exitedChannels: Set<string> = new Set()
  private readonly conflictingMessageStore: {} = {}
  private readonly messageStore: ParsedMessage[] = []

  public constructor(private readonly myAddress: Buffer) {}

  public async storeMessage(parsedMessage: ParsedMessage): Promise<void> {
    // Check if conflict, and if so, store separately
    const potentialConflict: ParsedMessage = await this.getMessageByChannelIdAndNonce(
      parsedMessage.message.channelId,
      parsedMessage.message.nonce
    )

    if (Utils.stateChannelMessagesConflict(parsedMessage, potentialConflict)) {
      this.putConflict(potentialConflict)
      return
    }

    const channelId: Buffer = await this.getChannelForCounterparty(
      parsedMessage.sender.equals(this.myAddress)
        ? parsedMessage.recipient
        : parsedMessage.sender
    )

    if (channelId && !channelId.equals(parsedMessage.message.channelId)) {
      throw Error(
        'Cannot store message because at least one participant is not a part of the listed channel.'
      )
    }

    for (let i = 0; i < this.messageStore.length; i++) {
      const parsedMsg: ParsedMessage = this.messageStore[i]
      if (
        parsedMsg.message.channelId.equals(parsedMessage.message.channelId) &&
        objectsEqual(parsedMsg.message, parsedMessage.message) &&
        ((!parsedMsg.message.nonce && !parsedMessage.message.nonce) ||
          (parsedMsg.message.nonce &&
            parsedMessage.message.nonce &&
            parsedMsg.message.nonce.eq(parsedMessage.message.nonce)))
      ) {
        this.messageStore[i] = parsedMessage
        break
      }
    }

    this.messageStore.push(parsedMessage)
  }

  public async getMessageByChannelIdAndNonce(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    for (const parsedMsg of this.messageStore) {
      if (
        parsedMsg.message.channelId.equals(channelId) &&
        parsedMsg.message.nonce &&
        parsedMsg.message.nonce.eq(nonce)
      ) {
        return parsedMsg
      }
    }
    return undefined
  }

  public async getMessagesByRecipient(
    recipient: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const parsedMsg of this.messageStore) {
      if (
        parsedMsg.recipient.equals(recipient) &&
        (!channelId || parsedMsg.message.channelId.equals(channelId)) &&
        (!nonce ||
          (parsedMsg.message.nonce && parsedMsg.message.nonce.eq(nonce)))
      ) {
        messages.push(parsedMsg)
      }
    }

    return messages
  }

  public async getMessagesBySender(
    sender: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const msg of this.messageStore) {
      if (
        msg.sender.equals(sender) &&
        (!channelId || msg.message.channelId.equals(channelId)) &&
        (!nonce || (msg.message.nonce && msg.message.nonce.eq(nonce)))
      ) {
        messages.push(msg)
      }
    }

    return messages
  }

  public async getMessagesSignedBy(
    signer: Buffer,
    channelId?: Buffer,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const parsedMsg of this.messageStore) {
      if (
        TestStateChannelMessageDB.messageSignedBy(parsedMsg, signer) &&
        (!channelId || parsedMsg.message.channelId.equals(channelId)) &&
        (!nonce ||
          (parsedMsg.message.nonce && parsedMsg.message.nonce.eq(nonce)))
      ) {
        messages.push(parsedMsg)
      }
    }

    return messages
  }

  private static messageSignedBy(
    message: ParsedMessage,
    signer: Buffer
  ): boolean {
    const signerAddress: string = signer.toString()
    for (const [address, signature] of Object.entries(message.signatures)) {
      if (address === signerAddress) {
        // TODO: would check signature, but not right now
        return true
      }
    }
    return false
  }

  public async getConflictingCounterpartyMessage(
    channelId: Buffer,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    return this.getConflict(channelId, nonce)
  }

  public async channelIdExists(channelId: Buffer): Promise<boolean> {
    for (const message of this.messageStore) {
      if (channelId.equals(message.message.channelId)) {
        return true
      }
    }
    return false
  }

  public async conflictsWithAnotherMessage(
    message: ParsedMessage
  ): Promise<ParsedMessage> {
    const conflict: ParsedMessage = this.getConflict(
      message.message.channelId,
      message.message.nonce
    )
    if (!!conflict) {
      return conflict
    }

    for (const msg of this.messageStore) {
      const storedConflict = this.getConflict(
        msg.message.channelId,
        msg.message.nonce
      )
      if (
        !!storedConflict &&
        storedConflict.message.nonce.equals(message.message.nonce)
      ) {
        return msg
      }
    }
  }

  public async getChannelForCounterparty(address: Buffer): Promise<Buffer> {
    for (const message of this.messageStore) {
      if (message.recipient.equals(address) || message.sender.equals(address)) {
        return message.message.channelId
      }
    }
  }

  public async getMostRecentMessageSignedBy(
    channelId: Buffer,
    address: Buffer
  ): Promise<ParsedMessage> {
    const addressString: string = address.toString()
    let mostRecent: ParsedMessage
    for (const message of this.messageStore) {
      if (
        message.message.channelId.equals(channelId) &&
        (!mostRecent || message.message.nonce.gt(mostRecent.message.nonce)) &&
        addressString in message.signatures
      ) {
        mostRecent = message
      }
    }
    return mostRecent
  }

  public async getMostRecentValidStateChannelMessage(
    channelId: Buffer
  ): Promise<ParsedMessage> {
    let mostRecent: ParsedMessage
    for (const message of this.messageStore) {
      if (
        message.message.channelId.equals(channelId) &&
        (!mostRecent || message.message.nonce.gt(mostRecent.message.nonce)) &&
        Object.keys(message.signatures).length === 2
      ) {
        mostRecent = message
      }
    }
    return mostRecent
  }

  public async isChannelExited(channelId: Buffer): Promise<boolean> {
    return this.exitedChannels.has(channelId.toString())
  }

  public async markChannelExited(channelId: Buffer): Promise<void> {
    this.exitedChannels.add(channelId.toString())
  }

  public getMyAddress(): Buffer {
    return this.myAddress
  }

  private getConflict(channelId: Buffer, nonce: BigNumber): ParsedMessage {
    const channelString: string = channelId.toString()
    const nonceString: string = nonce.toString()
    if (
      channelString in this.conflictingMessageStore &&
      nonceString in this.conflictingMessageStore[channelString]
    ) {
      return this.conflictingMessageStore[channelString][nonce]
    }
    return undefined
  }

  private putConflict(message: ParsedMessage): void {
    const channelString: string = message.message.channelId.toString()
    const nonceString: string = message.message.nonce.toString()
    if (!(channelString in this.conflictingMessageStore)) {
      this.conflictingMessageStore[channelString] = {}
    }
    this.conflictingMessageStore[channelString][nonceString] = message
  }
}

const equalsSignatureVerifier = async (
  publicKey: Buffer,
  message: Buffer,
  signature: Buffer
): Promise<boolean> => {
  return message.equals(signature)
}

const checkSignedMessage = (
  signedMessage: SignedMessage,
  sender: Buffer,
  nonce?: BigNumber,
  channelId?: Buffer,
  signers?: Buffer[],
  addressBalance?: AddressBalance
) => {
  assert(
    !!signedMessage,
    'Signed Message should not be undefined. Channel should be created'
  )
  assert(
    signedMessage.sender.equals(sender),
    `Sender of message should be ${sender}`
  )

  const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
    signedMessage,
    sender
  )
  if (!!nonce) {
    assert(
      parsedMessage.message.nonce.eq(nonce),
      'First message in a channel should have nonce 1'
    )
  }

  if (!!channelId) {
    assert(
      channelId.equals(parsedMessage.message.channelId),
      `Channel ID should equal ${channelId.toString()}`
    )
  } else {
    assert(
      !!parsedMessage.message.channelId,
      `Channel ID should exist for all messages`
    )
  }

  if (!!signers) {
    const expectedLength: number = Object.keys(parsedMessage.signatures).length
    assert(
      expectedLength === signers.length,
      `There should be ${expectedLength} signature(s) for new message`
    )
    for (const signer of signers) {
      assert(
        signer.toString() in parsedMessage.signatures,
        `The message should be signed by ${signers.toString()}`
      )
    }
  }

  const stateChannelMessage: StateChannelMessage = parsedMessage.message
    .data as StateChannelMessage

  if (!!addressBalance) {
    for (const [address, balance] of Object.entries(addressBalance)) {
      assert(
        stateChannelMessage.addressBalance[address].equals(balance),
        `Address ${address} should start with balance ${balance.toString()}`
      )
    }
  }
}

const getChannelId = (
  signedMessage: SignedMessage,
  myAddress: Buffer = undefined
): Buffer => {
  return parseStateChannelSignedMessage(signedMessage, myAddress).message
    .channelId
}

describe('State Channel Tests', () => {
  const aPrivateKey: Buffer = Buffer.from('A Private Key')
  const aAddress: Buffer = Buffer.from('A Address')

  const bPrivateKey: Buffer = Buffer.from('B Private Key')
  const bAddress: Buffer = Buffer.from('B Address')

  let a: StateChannelClient
  let aMemdown: any
  let aDb: DB
  let aMessageDB: TestStateChannelMessageDB
  let aSignedByDecider: SignedByDecider
  let aSignedByQuantifier: SignedByQuantifier

  let b: StateChannelClient
  let bMemdown: any
  let bDb: DB
  let bMessageDB: TestStateChannelMessageDB
  let bSignedByDecider: SignedByDecider
  let bSignedByQuantifier: SignedByQuantifier

  beforeEach(() => {
    aMemdown = new MemDown('a')
    aDb = new BaseDB(aMemdown, 256)
    aMessageDB = new TestStateChannelMessageDB(aAddress)
    aSignedByDecider = new SignedByDecider(aDb, equalsSignatureVerifier)
    aSignedByQuantifier = new SignedByQuantifier(aMessageDB, aAddress)

    a = new StateChannelClient(
      aMessageDB,
      aSignedByDecider,
      aSignedByQuantifier,
      aPrivateKey,
      aAddress
    )

    bMemdown = new MemDown('b')
    bDb = new BaseDB(bMemdown, 256)
    bMessageDB = new TestStateChannelMessageDB(bAddress)
    bSignedByDecider = new SignedByDecider(bDb, equalsSignatureVerifier)
    bSignedByQuantifier = new SignedByQuantifier(bMessageDB, bAddress)

    b = new StateChannelClient(
      bMessageDB,
      bSignedByDecider,
      bSignedByQuantifier,
      bPrivateKey,
      bAddress
    )
  })

  afterEach(async () => {
    await Promise.all([aDb.close(), bDb.close()])
    aMemdown = undefined
    bMemdown = undefined
  })

  const createChannel = async (): Promise<SignedMessage> => {
    const addressBalance: AddressBalance = {}
    const ten: BigNumber = new BigNumber(10)
    addressBalance[aAddress.toString()] = ten
    addressBalance[bAddress.toString()] = ten
    const signedMessage: SignedMessage = await a.createNewMessage(
      addressBalance,
      bAddress
    )

    checkSignedMessage(
      signedMessage,
      aAddress,
      ONE,
      undefined,
      [aAddress],
      addressBalance
    )
    return signedMessage
  }

  const acknowledgeMessage = async (
    signedMessage: SignedMessage,
    myClient: StateChannelClient,
    nonce: BigNumber = ONE
  ): Promise<void> => {
    const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
      signedMessage,
      myClient.myAddress
    )

    const counterSigned: SignedMessage = await myClient.handleMessage(
      signedMessage
    )
    checkSignedMessage(
      counterSigned,
      myClient.myAddress,
      nonce,
      parsedMessage.message.channelId,
      [myClient.myAddress],
      parsedMessage.message.data['addressBalance']
    )

    const otherClient: StateChannelClient = myClient === a ? b : a
    const res: SignedMessage = await otherClient.handleMessage(counterSigned)
    assert(
      res === undefined,
      'There should be no response when accepting a coutnersigned message'
    )
  }

  describe('channel creation', () => {
    it('handles channel creation', async () => {
      await createChannel()
    })

    it('handles channel creation acknowledgement', async () => {
      const signedMessage: SignedMessage = await createChannel()
      await acknowledgeMessage(signedMessage, b)
    })
  })

  describe('message exchange', () => {
    it('handles new message from A on created channel', async () => {
      const signedMessage: SignedMessage = await createChannel()
      await acknowledgeMessage(signedMessage, b)

      const addressBalance: AddressBalance = {}
      addressBalance[aAddress.toString()] = new BigNumber(5)
      addressBalance[bAddress.toString()] = new BigNumber(15)

      const nextMessage: SignedMessage = await a.createNewMessage(
        addressBalance,
        bAddress
      )

      const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
        nextMessage,
        aAddress
      )

      checkSignedMessage(
        nextMessage,
        aAddress,
        new BigNumber(2),
        parsedMessage.message.channelId,
        [aAddress],
        parsedMessage.message.data['addressBalance']
      )
    })

    it('handles new message from B on created channel', async () => {
      const signedMessage: SignedMessage = await createChannel()
      await acknowledgeMessage(signedMessage, b)

      const addressBalance: AddressBalance = {}
      addressBalance[bAddress.toString()] = new BigNumber(5)
      addressBalance[aAddress.toString()] = new BigNumber(15)

      const nextMessage: SignedMessage = await b.createNewMessage(
        addressBalance,
        aAddress
      )

      const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      checkSignedMessage(
        nextMessage,
        bAddress,
        new BigNumber(2),
        parsedMessage.message.channelId,
        [bAddress],
        parsedMessage.message.data['addressBalance']
      )
    })

    it('acknowledges new message from A on created channel', async () => {
      const signedMessage: SignedMessage = await createChannel()
      await acknowledgeMessage(signedMessage, b)

      const addressBalance: AddressBalance = {}
      addressBalance[aAddress.toString()] = new BigNumber(5)
      addressBalance[bAddress.toString()] = new BigNumber(15)

      const nextMessage: SignedMessage = await a.createNewMessage(
        addressBalance,
        bAddress
      )

      const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      checkSignedMessage(
        nextMessage,
        aAddress,
        new BigNumber(2),
        parsedMessage.message.channelId,
        [aAddress],
        parsedMessage.message.data['addressBalance']
      )

      await acknowledgeMessage(nextMessage, b, new BigNumber(2))
    })

    it('acknowledges new message from B on created channel', async () => {
      const signedMessage: SignedMessage = await createChannel()
      await acknowledgeMessage(signedMessage, b)

      const addressBalance: AddressBalance = {}
      addressBalance[bAddress.toString()] = new BigNumber(5)
      addressBalance[aAddress.toString()] = new BigNumber(15)

      const nextMessage: SignedMessage = await b.createNewMessage(
        addressBalance,
        aAddress
      )

      const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      checkSignedMessage(
        nextMessage,
        bAddress,
        new BigNumber(2),
        parsedMessage.message.channelId,
        [bAddress],
        parsedMessage.message.data['addressBalance']
      )

      await acknowledgeMessage(nextMessage, a, new BigNumber(2))
    })
  })

  describe('channel exit', () => {
    describe('valid exits', () => {
      it('handles A exit right after creation', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const claim: StateChannelExitClaim = await a.exitChannel(b.myAddress)
        assert(!!claim, 'Exist claim should not be null/undefined!')

        const counterClaim: ImplicationProofItem[] = await b.handleChannelExit(
          getChannelId(signedMessage),
          claim
        )
        assert(
          !counterClaim,
          'Exit should be valid, so there should be no counter-claim'
        )
      })

      it('handles B exit right after creation', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const claim: StateChannelExitClaim = await b.exitChannel(a.myAddress)
        assert(!!claim, 'Exist claim should not be null/undefined!')

        const counterClaim: ImplicationProofItem[] = await a.handleChannelExit(
          getChannelId(signedMessage),
          claim
        )
        assert(
          !counterClaim,
          'Exit should be valid, so there should be no counter-claim'
        )
      })

      it('handles A exit of second message', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const addressBalance: AddressBalance = {}
        addressBalance[aAddress.toString()] = new BigNumber(5)
        addressBalance[bAddress.toString()] = new BigNumber(15)

        const nextMessage: SignedMessage = await a.createNewMessage(
          addressBalance,
          bAddress
        )

        const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
          nextMessage,
          bAddress
        )

        checkSignedMessage(
          nextMessage,
          aAddress,
          new BigNumber(2),
          parsedMessage.message.channelId,
          [aAddress],
          parsedMessage.message.data['addressBalance']
        )

        await acknowledgeMessage(nextMessage, b, new BigNumber(2))

        const claim: StateChannelExitClaim = await a.exitChannel(b.myAddress)
        assert(!!claim, 'Exist claim should not be null/undefined!')

        const counterClaim: ImplicationProofItem[] = await b.handleChannelExit(
          getChannelId(signedMessage),
          claim
        )
        assert(
          !counterClaim,
          'Exit should be valid, so there should be no counter-claim'
        )
      })
    })

    describe('invalid exit disputes', () => {
      it('ensures B properly refutes an invalid nonce exit from A', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const mostRecentMessage: ParsedMessage = parseStateChannelSignedMessage(
          signedMessage,
          aAddress
        )

        const refutableClaim: StateChannelExitClaim = {
          decider: AndDecider.instance(),
          input: {
            // Claim that B has signed the message to be exited (this will evaluate to true)
            left: {
              decider: aSignedByDecider,
              input: {
                message: messageToBuffer(
                  mostRecentMessage.message,
                  stateChannelMessageToString
                ),
                publicKey: bAddress,
              },
            },
            leftWitness: {
              signature: mostRecentMessage.signatures[bAddress.toString()],
            },
            // Claim that A has not signed any message with nonce higher than the previous message (wrong)
            right: {
              decider: ForAllSuchThatDecider.instance(),
              input: {
                quantifier: aSignedByQuantifier,
                quantifierParameters: { address: aAddress },
                propertyFactory: (message: ParsedMessage) => {
                  return {
                    decider: MessageNonceLessThanDecider.instance(),
                    input: {
                      messageWithNonce: message,
                      // This will be disputed because mostRecentMessage has been signed by A
                      lessThanThis: mostRecentMessage.message.nonce,
                    },
                  }
                },
              },
            },
            rightWitness: undefined,
          },
        }

        const counterClaimJustification: ImplicationProofItem[] = await b.handleChannelExit(
          getChannelId(signedMessage),
          refutableClaim
        )
        assert(
          !!counterClaimJustification,
          'Exit should not be valid, so there should be a counter-claim'
        )
        assert(
          counterClaimJustification.length === 3,
          `Counter claim should have 3 justification layers: AndDecider, ForAllSuchThatDecider, and MessageNonceLessThanDecider. Received ${counterClaimJustification.length}.`
        )
        assert(
          counterClaimJustification[0].implication.decider instanceof
            AndDecider,
          `First counter-claim decider should be AndDecider. Received: ${JSON.stringify(
            counterClaimJustification[0]
          )}`
        )
        assert(
          counterClaimJustification[1].implication.decider instanceof
            ForAllSuchThatDecider,
          `Second counter-claim decider should be ForAllSuchThatDecider. Received: ${JSON.stringify(
            counterClaimJustification[1]
          )}`
        )
        assert(
          counterClaimJustification[2].implication.decider instanceof
            MessageNonceLessThanDecider,
          `Third counter-claim decider should be MessageNonceLessThanDecider. Received: ${JSON.stringify(
            counterClaimJustification[2]
          )}`
        )
        const nonceLessThanInput: MessageNonceLessThanInput = counterClaimJustification[2]
          .implication.input as MessageNonceLessThanInput
        assert(
          nonceLessThanInput.messageWithNonce.message.nonce.gte(
            nonceLessThanInput.lessThanThis
          ),
          `Counter-claim should be based on a message with a nonce that is NOT less than ${
            nonceLessThanInput.lessThanThis
          }, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
        assert(
          aAddress.toString() in nonceLessThanInput.messageWithNonce.signatures,
          `Counter-claim proof message should be signed by A, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
      })

      it('ensures A properly refutes an invalid nonce exit from B', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const mostRecentMessage: ParsedMessage = parseStateChannelSignedMessage(
          signedMessage,
          bAddress
        )

        const refutableClaim: StateChannelExitClaim = {
          decider: AndDecider.instance(),
          input: {
            // Claim that A has signed the message to be exited (this will evaluate to true)
            left: {
              decider: bSignedByDecider,
              input: {
                message: messageToBuffer(
                  mostRecentMessage.message,
                  stateChannelMessageToString
                ),
                publicKey: aAddress,
              },
            },
            leftWitness: {
              signature: mostRecentMessage.signatures[aAddress.toString()],
            },
            // Claim that B has not signed any message with nonce higher than the previous message (wrong)
            right: {
              decider: ForAllSuchThatDecider.instance(),
              input: {
                quantifier: bSignedByQuantifier,
                quantifierParameters: { address: bAddress },
                propertyFactory: (message: ParsedMessage) => {
                  return {
                    decider: MessageNonceLessThanDecider.instance(),
                    input: {
                      messageWithNonce: message,
                      // This will be disputed because mostRecentMessage has been signed by B
                      lessThanThis: mostRecentMessage.message.nonce,
                    },
                  }
                },
              },
            },
            rightWitness: undefined,
          },
        }

        const counterClaimJustification: ImplicationProofItem[] = await a.handleChannelExit(
          getChannelId(signedMessage),
          refutableClaim
        )
        assert(
          !!counterClaimJustification,
          'Exit should not be valid, so there should be a counter-claim'
        )
        assert(
          counterClaimJustification.length === 3,
          `Counter claim should have 3 justification layers: AndDecider, ForAllSuchThatDecider, and MessageNonceLessThanDecider. Received ${counterClaimJustification.length}.`
        )
        assert(
          counterClaimJustification[0].implication.decider instanceof
            AndDecider,
          `First counter-claim decider should be AndDecider. Received: ${JSON.stringify(
            counterClaimJustification[0]
          )}`
        )
        assert(
          counterClaimJustification[1].implication.decider instanceof
            ForAllSuchThatDecider,
          `Second counter-claim decider should be ForAllSuchThatDecider. Received: ${JSON.stringify(
            counterClaimJustification[1]
          )}`
        )
        assert(
          counterClaimJustification[2].implication.decider instanceof
            MessageNonceLessThanDecider,
          `Third counter-claim decider should be MessageNonceLessThanDecider. Received: ${JSON.stringify(
            counterClaimJustification[2]
          )}`
        )
        const nonceLessThanInput: MessageNonceLessThanInput = counterClaimJustification[2]
          .implication.input as MessageNonceLessThanInput
        assert(
          nonceLessThanInput.messageWithNonce.message.nonce.gte(
            nonceLessThanInput.lessThanThis
          ),
          `Counter-claim should be based on a message with a nonce that is NOT less than ${
            nonceLessThanInput.lessThanThis
          }, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
        assert(
          aAddress.toString() in nonceLessThanInput.messageWithNonce.signatures,
          `Counter-claim proof message should be signed by B, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
      })
    })
  })
})
