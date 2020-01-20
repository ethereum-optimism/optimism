import '../../setup'

/* External Imports */
import {
  BigNumber,
  bufToHexString,
  DefaultSignatureProvider,
  DefaultSignatureVerifier,
  objectsEqual,
  ONE,
  SignatureVerifier,
} from '@pigi/core-utils'
import * as assert from 'assert'

/* Internal Imports */
import {
  AndDecider,
  ForAllSuchThatDecider,
  MessageNonceLessThanDecider,
  MessageNonceLessThanInput,
  SignedByDecider,
  Utils,
  ImplicationProofItem,
  StateChannelMessageDBInterface,
  Message,
  ParsedMessage,
  SignedMessage,
  SignedByQuantifier,
  AddressBalance,
  parseStateChannelSignedMessage,
  StateChannelExitClaim,
  StateChannelMessage,
  deserializeMessage,
  messageToString,
  stateChannelMessageDeserializer,
  stateChannelMessageToString,
  StateChannelClient,
} from '../../../src'

class TestStateChannelMessageDB implements StateChannelMessageDBInterface {
  private readonly exitedChannels: Set<string> = new Set()
  private readonly conflictingMessageStore: {} = {}
  private readonly messageStore: ParsedMessage[] = []
  private readonly signedMessages: Map<string, SignedMessage[]> = new Map<
    string,
    SignedMessage[]
  >()

  public constructor(
    private readonly myAddress: string,
    private readonly signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
  ) {}

  public async handleMessage(
    serializedMessage: string,
    signature?: string
  ): Promise<void> {
    try {
      // TODO Look at how this is used. This is probably messed up.
      const message: Message = deserializeMessage(serializedMessage)
      await this.storeMessage(message.data as ParsedMessage)
    } catch (e) {
      // Must not have been a ParsedMessage
    }
  }

  public async storeSignedMessage(
    serializedMessage: string,
    signature: string
  ): Promise<void> {
    const signerPubKey: string = this.signatureVerifier.verifyMessage(
      serializedMessage,
      signature
    )

    if (!this.signedMessages.has(signerPubKey)) {
      this.signedMessages.set(signerPubKey, [])
    }

    this.signedMessages.get(signerPubKey).push({
      signature,
      serializedMessage,
    })
  }

  public async storeMessage(parsedMessage: ParsedMessage): Promise<void> {
    const serializedMessage: string = messageToString(
      parsedMessage.message,
      stateChannelMessageToString
    )

    // Save signed messages.
    await Promise.all(
      Object.keys(parsedMessage.signatures).map((k: string) =>
        this.storeSignedMessage(serializedMessage, parsedMessage.signatures[k])
      )
    )

    // Check if conflict, and if so, store separately
    const potentialConflict: ParsedMessage = await this.getMessageByChannelIdAndNonce(
      parsedMessage.message.channelID,
      parsedMessage.message.nonce
    )

    if (Utils.stateChannelMessagesConflict(parsedMessage, potentialConflict)) {
      this.putConflict(potentialConflict)
      return
    }

    const channelID: string = await this.getChannelForCounterparty(
      parsedMessage.sender === this.myAddress
        ? parsedMessage.recipient
        : parsedMessage.sender
    )

    if (channelID && channelID !== parsedMessage.message.channelID) {
      throw Error(
        'Cannot store message because at least one participant is not a part of the listed channel.'
      )
    }

    for (let i = 0; i < this.messageStore.length; i++) {
      const parsedMsg: ParsedMessage = this.messageStore[i]
      if (
        parsedMsg.message.channelID === parsedMessage.message.channelID &&
        objectsEqual(parsedMsg.message, parsedMessage.message) &&
        ((!parsedMsg.message.nonce && !parsedMessage.message.nonce) ||
          (parsedMsg.message.nonce &&
            parsedMessage.message.nonce &&
            parsedMsg.message.nonce.eq(parsedMessage.message.nonce)))
      ) {
        this.messageStore[i] = parsedMessage
        return
      }
    }

    this.messageStore.push(parsedMessage)
  }

  public async getMessageByChannelIdAndNonce(
    channelID: string,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    for (const parsedMsg of this.messageStore) {
      if (
        parsedMsg.message.channelID === channelID &&
        parsedMsg.message.nonce &&
        parsedMsg.message.nonce.eq(nonce)
      ) {
        return parsedMsg
      }
    }
    return undefined
  }

  public async getMessagesByRecipient(
    recipient: string,
    channelID?: string,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const parsedMsg of this.messageStore) {
      if (
        parsedMsg.recipient === recipient &&
        (!channelID || parsedMsg.message.channelID === channelID) &&
        (!nonce ||
          (parsedMsg.message.nonce && parsedMsg.message.nonce.eq(nonce)))
      ) {
        messages.push(parsedMsg)
      }
    }

    return messages
  }

  public async getMessagesBySender(
    sender: string,
    channelID?: string,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const msg of this.messageStore) {
      if (
        msg.sender === sender &&
        (!channelID || msg.message.channelID === channelID) &&
        (!nonce || (msg.message.nonce && msg.message.nonce.eq(nonce)))
      ) {
        messages.push(msg)
      }
    }

    return messages
  }

  public async getMessagesSignedBy(
    signer: string,
    channelID?: string,
    nonce?: BigNumber
  ): Promise<ParsedMessage[]> {
    // passes back live references to messages, but that doesn't matter for these tests.
    const messages = []
    for (const parsedMsg of this.messageStore) {
      if (
        (await this.messageSignedBy(parsedMsg, signer)) &&
        (!channelID || parsedMsg.message.channelID === channelID) &&
        (!nonce ||
          (parsedMsg.message.nonce && parsedMsg.message.nonce.eq(nonce)))
      ) {
        messages.push(parsedMsg)
      }
    }

    return messages
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

  private async messageSignedBy(
    message: ParsedMessage,
    pubKey: string
  ): Promise<boolean> {
    for (const [address, signature] of Object.entries(message.signatures)) {
      if (address === pubKey) {
        const serializedMessage: string = messageToString(
          message.message,
          stateChannelMessageToString
        )
        const messageSigner: string = await this.signatureVerifier.verifyMessage(
          serializedMessage,
          signature
        )

        return messageSigner === pubKey
      }
    }
    return false
  }

  public async getConflictingCounterpartyMessage(
    channelID: string,
    nonce: BigNumber
  ): Promise<ParsedMessage> {
    return this.getConflict(channelID, nonce)
  }

  public async channelIDExists(channelID: string): Promise<boolean> {
    for (const message of this.messageStore) {
      if (channelID === message.message.channelID) {
        return true
      }
    }
    return false
  }

  public async conflictsWithAnotherMessage(
    message: ParsedMessage
  ): Promise<ParsedMessage> {
    const conflict: ParsedMessage = this.getConflict(
      message.message.channelID,
      message.message.nonce
    )
    if (!!conflict) {
      return conflict
    }

    for (const msg of this.messageStore) {
      const storedConflict = this.getConflict(
        msg.message.channelID,
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

  public async getChannelForCounterparty(address: string): Promise<string> {
    for (const message of this.messageStore) {
      if (message.recipient === address || message.sender === address) {
        return message.message.channelID
      }
    }
  }

  public async getMostRecentMessageSignedBy(
    channelID: string,
    address: string
  ): Promise<ParsedMessage> {
    const addressString: string = address.toString()
    let mostRecent: ParsedMessage
    for (const message of this.messageStore) {
      if (
        message.message.channelID === channelID &&
        (!mostRecent || message.message.nonce.gt(mostRecent.message.nonce)) &&
        addressString in message.signatures
      ) {
        mostRecent = message
      }
    }
    return mostRecent
  }

  public async getMostRecentValidStateChannelMessage(
    channelID: string
  ): Promise<ParsedMessage> {
    let mostRecent: ParsedMessage
    for (const message of this.messageStore) {
      if (
        message.message.channelID === channelID &&
        (!mostRecent || message.message.nonce.gt(mostRecent.message.nonce)) &&
        Object.keys(message.signatures).length === 2
      ) {
        mostRecent = message
      }
    }
    return mostRecent
  }

  public async isChannelExited(channelID: string): Promise<boolean> {
    return this.exitedChannels.has(channelID)
  }

  public async markChannelExited(channelID: string): Promise<void> {
    this.exitedChannels.add(channelID)
  }

  public getMyAddress(): string {
    return this.myAddress
  }

  private getConflict(channelID: string, nonce: BigNumber): ParsedMessage {
    const nonceString: string = nonce.toString()
    if (
      channelID in this.conflictingMessageStore &&
      nonceString in this.conflictingMessageStore[channelID]
    ) {
      return this.conflictingMessageStore[channelID][nonce]
    }
    return undefined
  }

  private putConflict(message: ParsedMessage): void {
    const nonceString: string = message.message.nonce.toString()
    if (!(message.message.channelID in this.conflictingMessageStore)) {
      this.conflictingMessageStore[message.message.channelID] = {}
    }
    this.conflictingMessageStore[message.message.channelID][
      nonceString
    ] = message
  }
}

const checkSignedMessage = async (
  signedMessage: SignedMessage,
  sender: string,
  nonce?: BigNumber,
  channelID?: string,
  signers?: string[],
  addressBalance?: AddressBalance,
  signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
): Promise<void> => {
  assert(
    !!signedMessage,
    'Signed Message should not be undefined. Channel should be created'
  )

  const signer: string = signatureVerifier.verifyMessage(
    signedMessage.serializedMessage,
    signedMessage.signature
  )

  assert(
    signer === sender,
    `Sender of message should be ${sender} but is ${signer}`
  )

  const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
    signedMessage,
    sender
  )
  if (!!nonce) {
    assert(
      parsedMessage.message.nonce.eq(nonce),
      'First message in a channel should have nonce 1'
    )
  }

  if (!!channelID) {
    assert(
      channelID === parsedMessage.message.channelID,
      `Channel ID should equal ${channelID.toString()}`
    )
  } else {
    assert(
      !!parsedMessage.message.channelID,
      `Channel ID should exist for all messages`
    )
  }

  if (!!signers) {
    const expectedLength: number = Object.keys(parsedMessage.signatures).length
    assert(
      expectedLength === signers.length,
      `There should be ${expectedLength} signature(s) for new message`
    )
    for (const addr of signers) {
      assert(
        addr.toString() in parsedMessage.signatures,
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

const getChannelId = async (
  signedMessage: SignedMessage,
  myAddress: string = undefined
): Promise<string> => {
  const message: ParsedMessage = await parseStateChannelSignedMessage(
    signedMessage,
    myAddress
  )
  return message.message.channelID
}

describe('State Channel Tests', () => {
  let aAddress: string
  const aSigner: DefaultSignatureProvider = new DefaultSignatureProvider()

  let bAddress: string
  const bSigner: DefaultSignatureProvider = new DefaultSignatureProvider()

  let a: StateChannelClient
  let aMessageDB: TestStateChannelMessageDB
  let aSignedByDecider: SignedByDecider
  let aSignedByQuantifier: SignedByQuantifier

  let b: StateChannelClient
  let bMessageDB: TestStateChannelMessageDB
  let bSignedByDecider: SignedByDecider
  let bSignedByQuantifier: SignedByQuantifier

  beforeEach(async () => {
    aAddress = await aSigner.getAddress()
    aMessageDB = new TestStateChannelMessageDB(aAddress)
    aSignedByDecider = new SignedByDecider(aMessageDB, aAddress)
    aSignedByQuantifier = new SignedByQuantifier(aMessageDB, aAddress)

    a = new StateChannelClient(
      aMessageDB,
      aSignedByDecider,
      aSignedByQuantifier,
      aAddress,
      aSigner
    )

    bAddress = await bSigner.getAddress()
    bMessageDB = new TestStateChannelMessageDB(bAddress)
    bSignedByDecider = new SignedByDecider(bMessageDB, bAddress)
    bSignedByQuantifier = new SignedByQuantifier(bMessageDB, bAddress)

    b = new StateChannelClient(
      bMessageDB,
      bSignedByDecider,
      bSignedByQuantifier,
      bAddress,
      bSigner
    )
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

    await checkSignedMessage(
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
    const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
      signedMessage,
      myClient.myAddress
    )

    const counterSigned: SignedMessage = await myClient.handleMessage(
      signedMessage
    )
    await checkSignedMessage(
      counterSigned,
      myClient.myAddress,
      nonce,
      parsedMessage.message.channelID,
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

      const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
        nextMessage,
        aAddress
      )

      await checkSignedMessage(
        nextMessage,
        aAddress,
        new BigNumber(2),
        parsedMessage.message.channelID,
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

      const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      await checkSignedMessage(
        nextMessage,
        bAddress,
        new BigNumber(2),
        parsedMessage.message.channelID,
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

      const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      await checkSignedMessage(
        nextMessage,
        aAddress,
        new BigNumber(2),
        parsedMessage.message.channelID,
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

      const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
        nextMessage,
        bAddress
      )

      await checkSignedMessage(
        nextMessage,
        bAddress,
        new BigNumber(2),
        parsedMessage.message.channelID,
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
          await getChannelId(signedMessage),
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
          await getChannelId(signedMessage),
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

        const parsedMessage: ParsedMessage = await parseStateChannelSignedMessage(
          nextMessage,
          bAddress
        )

        await checkSignedMessage(
          nextMessage,
          aAddress,
          new BigNumber(2),
          parsedMessage.message.channelID,
          [aAddress],
          parsedMessage.message.data['addressBalance']
        )

        await acknowledgeMessage(nextMessage, b, new BigNumber(2))

        const claim: StateChannelExitClaim = await a.exitChannel(b.myAddress)
        assert(!!claim, 'Exist claim should not be null/undefined!')

        const counterClaim: ImplicationProofItem[] = await b.handleChannelExit(
          await getChannelId(signedMessage),
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

        const mostRecentMessage: ParsedMessage = await parseStateChannelSignedMessage(
          signedMessage,
          aAddress
        )

        const refutableClaim: StateChannelExitClaim = {
          decider: AndDecider.instance(),
          input: {
            // Claim that B has signed the message to be exited (this will evaluate to true)
            properties: [
              {
                decider: bSignedByDecider,
                input: {
                  serializedMessage: messageToString(
                    mostRecentMessage.message,
                    stateChannelMessageToString
                  ),
                  publicKey: bAddress,
                },
                witness: {
                  signature: mostRecentMessage.signatures[bAddress.toString()],
                },
              },
              // Claim that A has not signed any message with nonce higher than the previous message (wrong)
              {
                decider: ForAllSuchThatDecider.instance(),
                input: {
                  quantifier: bSignedByQuantifier,
                  quantifierParameters: {
                    address: aAddress,
                    channelID: mostRecentMessage.message.channelID,
                  },
                  propertyFactory: (signed: SignedMessage) => {
                    return {
                      decider: MessageNonceLessThanDecider.instance(),
                      input: {
                        messageWithNonce: deserializeMessage(
                          signed.serializedMessage,
                          stateChannelMessageDeserializer
                        ),
                        // This will be disputed because mostRecentMessage has been signed by A
                        lessThanThis: mostRecentMessage.message.nonce,
                      },
                    }
                  },
                },
              },
            ],
          },
        }

        const counterClaimJustification: ImplicationProofItem[] = await b.handleChannelExit(
          await getChannelId(signedMessage),
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
          nonceLessThanInput.messageWithNonce.nonce.gte(
            nonceLessThanInput.lessThanThis
          ),
          `Counter-claim should be based on a message with a nonce that is NOT less than ${
            nonceLessThanInput.lessThanThis
          }, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
      })

      it('ensures A properly refutes an invalid nonce exit from B', async () => {
        const signedMessage: SignedMessage = await createChannel()
        await acknowledgeMessage(signedMessage, b)

        const mostRecentMessage: ParsedMessage = await parseStateChannelSignedMessage(
          signedMessage,
          bAddress
        )

        const refutableClaim: StateChannelExitClaim = {
          decider: AndDecider.instance(),
          input: {
            // Claim that B has signed the message to be exited (this will evaluate to true)
            properties: [
              {
                decider: aSignedByDecider,
                input: {
                  serializedMessage: messageToString(
                    mostRecentMessage.message,
                    stateChannelMessageToString
                  ),
                  publicKey: aAddress,
                },
                witness: {
                  signature: mostRecentMessage.signatures[aAddress.toString()],
                },
              },
              // Claim that A has not signed any message with nonce higher than the previous message (wrong)
              {
                decider: ForAllSuchThatDecider.instance(),
                input: {
                  quantifier: aSignedByQuantifier,
                  quantifierParameters: {
                    address: bAddress,
                    channelID: mostRecentMessage.message.channelID,
                  },
                  propertyFactory: (signed: SignedMessage) => {
                    return {
                      decider: MessageNonceLessThanDecider.instance(),
                      input: {
                        messageWithNonce: deserializeMessage(
                          signed.serializedMessage,
                          stateChannelMessageDeserializer
                        ),
                        // This will be disputed because mostRecentMessage has been signed by A
                        lessThanThis: mostRecentMessage.message.nonce,
                      },
                    }
                  },
                },
              },
            ],
          },
        }

        const counterClaimJustification: ImplicationProofItem[] = await a.handleChannelExit(
          await getChannelId(signedMessage),
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
          nonceLessThanInput.messageWithNonce.nonce.gte(
            nonceLessThanInput.lessThanThis
          ),
          `Counter-claim should be based on a message with a nonce that is NOT less than ${
            nonceLessThanInput.lessThanThis
          }, but received message: ${JSON.stringify(
            nonceLessThanInput.messageWithNonce
          )}`
        )
      })
    })
  })
})
