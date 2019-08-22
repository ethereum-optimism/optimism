import uuid = require('uuid')

import {
  ONE,
  ZERO,
  ParsedMessage,
  SignedMessage,
  StateChannelMessageDBInterface,
  ImplicationProofItem,
  Decision,
  BigNumber,
} from '../../../types'
import {
  AddressBalance,
  deserializeBuffer,
  deserializeMessage,
  messageToBuffer,
  objectToBuffer,
  parseStateChannelSignedMessage,
  StateChannelExitClaim,
  StateChannelMessage,
  stateChannelMessageDeserializer,
  stateChannelMessageToString,
} from '../../serialization'
import {
  AndDecider,
  CannotDecideError,
  ForAllSuchThatDecider,
  MessageNonceLessThanDecider,
  Utils,
} from '../deciders'
import { SignedByDecider } from '../deciders/signed-by-decider'
import { SignedByQuantifier } from '../quantifiers/signed-by-quantifier'
import { sign } from '../../utils'

/**
 * Client responsible for State Channel communication
 */
export class StateChannelClient {
  public constructor(
    private readonly messageDB: StateChannelMessageDBInterface,
    private readonly signedByDecider: SignedByDecider,
    private readonly signedByQuantifier: SignedByQuantifier,
    private readonly myPrivateKey: Buffer,
    public readonly myAddress: Buffer
  ) {}

  /**
   * Creates a new Signed message with the provided balances on the channel that exists between
   * this address and the provided recipient address. If a channel does not exist, the resulting
   * SignedMessage will represent the first message in a new state channel.
   *
   * @param addressBalance The address balance for the message to create
   * @param recipient The address to send the SignedMessage to
   * @returns The SignedMessage if this client is able to create one
   */
  public async createNewMessage(
    addressBalance: AddressBalance,
    recipient: Buffer
  ): Promise<SignedMessage> {
    let channelID: Buffer = await this.messageDB.getChannelForCounterparty(
      recipient
    )
    let nonce: BigNumber

    if (!!channelID) {
      const [lastValid, lastSigned, exited]: [
        ParsedMessage,
        ParsedMessage,
        boolean
      ] = await Promise.all([
        this.messageDB.getMostRecentValidStateChannelMessage(channelID),
        this.messageDB.getMostRecentMessageSignedBy(channelID, this.myAddress),
        this.messageDB.isChannelExited(channelID),
      ])

      if (
        (!lastSigned && !lastValid) ||
        !lastSigned.message.nonce.equals(lastValid.message.nonce)
      ) {
        throw Error(
          'Cannot create new message when last message is not counter-signed'
        )
      }

      if (exited) {
        throw Error('Cannot create new message for exited channel.')
      }

      nonce = lastValid.message.nonce.add(ONE)
    } else {
      channelID = Buffer.from(uuid.v4())
      nonce = ONE
    }

    return this.signAndSaveMessage({
      sender: this.myAddress,
      recipient,
      message: {
        channelID,
        nonce,
        data: { addressBalance },
      },
      signatures: {},
    })
  }

  /**
   * Exits the state channel with the provided counterparty.
   *
   * @param counterparty The address of the counterparty.
   * @returns The StateChannelClaim representing a valid exit claim for this channel.
   */
  public async exitChannel(
    counterparty: Buffer
  ): Promise<StateChannelExitClaim> {
    const channelID: Buffer = await this.messageDB.getChannelForCounterparty(
      counterparty
    )

    if (!channelID) {
      throw Error('Cannot exit a channel that does not exist.')
    }

    const mostRecent: ParsedMessage = await this.messageDB.getMostRecentValidStateChannelMessage(
      channelID
    )

    await this.messageDB.markChannelExited(channelID)

    return {
      decider: AndDecider.instance(),
      input: {
        properties: [
          {
            decider: this.signedByDecider,
            input: {
              message: messageToBuffer(
                mostRecent.message,
                stateChannelMessageToString
              ),
              publicKey: counterparty,
            },
            witness: {
              signature: mostRecent.signatures[counterparty.toString()],
            },
          },
          {
            decider: ForAllSuchThatDecider.instance(),
            input: {
              quantifier: this.signedByQuantifier,
              quantifierParameters: { address: this.myAddress },
              propertyFactory: (signedMessage: Buffer) => {
                return {
                  decider: MessageNonceLessThanDecider.instance(),
                  input: {
                    messageWithNonce: deserializeBuffer(
                      signedMessage,
                      deserializeMessage,
                      stateChannelMessageDeserializer
                    ),
                    lessThanThis: mostRecent.message.nonce.add(ONE),
                  },
                }
              },
            },
          },
        ],
      },
    }
  }

  /**
   * Handles a channel exit claim by validating it. If it can be disproven, it will return the
   * counter-claim that disproves it.
   *
   * TODO: Improve this signature so that channel ID doesn't need to be passed
   * @param channelID the ChannelID in question
   * @param exitClaim the Exit claim in question
   * @returns the counter-claim that the original claim is invalid
   */
  public async handleChannelExit(
    channelID: Buffer,
    exitClaim: StateChannelExitClaim
  ): Promise<ImplicationProofItem[]> {
    let decision: Decision
    try {
      decision = await exitClaim.decider.decide(exitClaim.input)
    } catch (e) {
      if (!(e instanceof CannotDecideError)) {
        throw e
      }
    }

    if (!decision || decision.outcome) {
      await this.messageDB.markChannelExited(channelID)
      return undefined
    }

    return decision.justification
  }

  /**
   * Handles the provided SignedMessage and responds in the appropriate manner
   *
   * @param message The message to handle
   * @returns The response message, if one exists
   */
  public async handleMessage(message: SignedMessage): Promise<SignedMessage> {
    const parsedMessage: ParsedMessage = parseStateChannelSignedMessage(
      message,
      this.myAddress
    )

    const existingMessage: ParsedMessage = await this.messageDB.getMessageByChannelIdAndNonce(
      parsedMessage.message.channelID,
      parsedMessage.message.nonce
    )

    const mergedMessage: ParsedMessage = this.updateWithReceived(
      existingMessage,
      parsedMessage
    )

    // Store message no matter what
    await this.messageDB.storeMessage(mergedMessage || parsedMessage)

    if (!!existingMessage) {
      return undefined
    }

    if (parsedMessage.message.nonce.equals(ONE)) {
      return this.handleNewChannel(parsedMessage)
    }

    return this.handleNewMessage(parsedMessage)
  }

  /**
   * Handle the case when an address wants to create a new channel with this client.
   *
   * @param message The message to create the new channel
   * @return the SignedMessage countersigning the new channel creation if valid, undefined otherwise
   */
  private async handleNewChannel(
    message: ParsedMessage
  ): Promise<SignedMessage> {
    if (!this.validateStateChannelMessage(message)) {
      // Not going to be a part of this channel
      return undefined
    }

    return this.signAndSaveMessage(message)
  }

  /**
   * Handles a new message for an existing channel.
   *
   * @param message The new message in question
   * @returns The SignedMessage of the countersigned message or undefined if we're disputing this message
   */
  private async handleNewMessage(
    message: ParsedMessage
  ): Promise<SignedMessage> {
    if (!this.validateStateChannelMessage(message)) {
      return undefined
    }

    const [exited, conflicts, previousMessage]: [
      boolean,
      ParsedMessage,
      ParsedMessage
    ] = await Promise.all([
      this.messageDB.isChannelExited(message.message.channelID),
      this.messageDB.conflictsWithAnotherMessage(message),
      this.messageDB.getMostRecentValidStateChannelMessage(
        message.message.channelID
      ),
    ])
    if (!!conflicts || exited) {
      return undefined
    }

    // No previous message or this nonce is invalid
    if (
      !previousMessage ||
      previousMessage.message.nonce.gte(message.message.nonce)
    ) {
      return undefined
    }

    return this.signAndSaveMessage(message)
  }

  /**
   * Creates a ParsedMessage that represents the existing message with updates
   * from the received message.
   * Mainly, this entails adding signatures if the sender is a new signer.
   *
   * @param existing The existing message, if one exists.
   * @param received The received message.
   * @returns The updated message or undefined if one does not exist.
   */
  private updateWithReceived(
    existing: ParsedMessage,
    received: ParsedMessage
  ): ParsedMessage {
    if (!!existing && !Utils.stateChannelMessagesConflict(received, existing)) {
      for (const [address, signature] of Object.entries(received.signatures)) {
        existing.signatures[address] = signature
      }
      return existing
    }
    return undefined
  }

  /**
   * Signs the provided message, stores it, and returns the signed message.
   * @param message The message to sign.
   * @returns The signed message.
   */
  private async signAndSaveMessage(
    message: ParsedMessage
  ): Promise<SignedMessage> {
    message.signatures[this.myAddress.toString()] = sign(
      this.myPrivateKey,
      messageToBuffer(message.message, stateChannelMessageToString)
    )

    await this.messageDB.storeMessage(message)

    return {
      sender: this.myAddress,
      signedMessage: objectToBuffer({
        channelID: message.message.channelID.toString(),
        nonce: message.message.nonce,
        data: stateChannelMessageToString(message.message
          .data as StateChannelMessage),
      }),
    }
  }

  /**
   * Validates that the provided ParsedMessage wraps a valid StateChannelMessage.
   *
   * @param message The message to validate
   * @returns True if it is valid, false otherwise
   */
  private validateStateChannelMessage(message: ParsedMessage): boolean {
    try {
      const stateChannelMessage: StateChannelMessage = message.message
        .data as StateChannelMessage
      return (
        message.message.nonce.gte(ZERO) &&
        Object.keys(stateChannelMessage.addressBalance).length === 2 &&
        stateChannelMessage.addressBalance[this.myAddress.toString()].gte(
          ZERO
        ) &&
        stateChannelMessage.addressBalance[message.sender.toString()].gte(ZERO)
      )
    } catch (e) {
      return false
    }
  }
}
