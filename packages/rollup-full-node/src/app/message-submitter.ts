/* External Imports */
import { DB } from '@eth-optimism/core-db'
import {
  abiDecodeL2ToL1Message,
  abiEncodeL2ToL1Message,
  L2ToL1Message,
} from '@eth-optimism/rollup-core'
import {
  add0x,
  BigNumber,
  getLogger,
  logError,
  ONE,
  ZERO,
} from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

import { L2ToL1MessageSubmitter } from '../types'
import { TransactionReceipt } from 'ethers/providers/abstract-provider'

const log = getLogger('rollup-message-submitter')

export class NoOpL2ToL1MessageSubmitter implements L2ToL1MessageSubmitter {
  public async submitMessage(l2ToL1Message: L2ToL1Message): Promise<void> {
    log.debug(
      `L2ToL1Message received by NoOpL2ToL1MessageSubmitter: ${JSON.stringify(
        NoOpL2ToL1MessageSubmitter
      )}`
    )
    return
  }
}

/**
 *  Default Message Submitter implementation. This will be deprecated when message submission works properly.
 */
export class DefaultL2ToL1MessageSubmitter implements L2ToL1MessageSubmitter {
  public static readonly LAST_CONFIRMED_KEY: Buffer = Buffer.from(
    'last_confirmed'
  )
  public static readonly LAST_SUBMITTED_KEY: Buffer = Buffer.from(
    'last_submitted'
  )
  public static readonly LAST_QUEUED_KEY: Buffer = Buffer.from('last_queued')

  private lastSubmitted: BigNumber
  private lastConfirmed: BigNumber
  private lastQueued: BigNumber
  private messageQueue: L2ToL1Message[]

  public static async create(
    db: DB,
    messageReceiverContract: Contract
  ): Promise<DefaultL2ToL1MessageSubmitter> {
    const submitter = new DefaultL2ToL1MessageSubmitter(
      db,
      messageReceiverContract
    )

    await submitter.init()

    return submitter
  }

  private constructor(
    private db: DB,
    private readonly messageReceiverContract: Contract
  ) {
    this.messageQueue = []
  }

  /**
   * Initializes this MessageSubmitter, loading any previously-stored state including:
   * * Last Submitted Message Number
   * * Last Confirmed Message Number
   * * Last Queued Message Number
   * * The Message Queue
   */
  private async init(): Promise<void> {
    try {
      const [
        lastSubmittedBuffer,
        lastConfirmedBuffer,
        lastQueuedBuffer,
      ] = await Promise.all([
        this.db.get(DefaultL2ToL1MessageSubmitter.LAST_SUBMITTED_KEY),
        this.db.get(DefaultL2ToL1MessageSubmitter.LAST_CONFIRMED_KEY),
        this.db.get(DefaultL2ToL1MessageSubmitter.LAST_QUEUED_KEY),
      ])

      this.lastSubmitted = !!lastSubmittedBuffer
        ? new BigNumber(lastSubmittedBuffer)
        : ZERO
      this.lastConfirmed = !!lastConfirmedBuffer
        ? new BigNumber(lastConfirmedBuffer)
        : ZERO
      this.lastQueued = !!lastQueuedBuffer
        ? new BigNumber(lastQueuedBuffer)
        : ZERO

      // We're up to date, return
      if (
        this.lastSubmitted.eq(this.lastConfirmed) &&
        this.lastConfirmed.eq(this.lastQueued)
      ) {
        log.info(
          `Last queued was confirmed (message ${this.lastQueued}). Done initializing.`
        )
        return
      }

      // We need to populate the queue from storage
      if (this.lastConfirmed !== this.lastQueued) {
        let i: BigNumber = this.lastConfirmed.add(ONE)
        const promises: Array<Promise<Buffer>> = []
        for (; i.lte(this.lastQueued); i = i.add(ONE)) {
          promises.push(
            this.db.get(DefaultL2ToL1MessageSubmitter.getMessageKey(i))
          )
        }

        const messages: Buffer[] = await Promise.all(promises)
        this.messageQueue = messages.map((x) =>
          DefaultL2ToL1MessageSubmitter.deserializeRollupMessageFromStorage(x)
        )
      }

      await this.trySubmitNextMessage()
      log.info(
        `Initialized Message submitter. Last Submitted: [${this.lastSubmitted.toString(
          'hex'
        )}], Last Confirmed: [${this.lastConfirmed.toString(
          'hex'
        )}], Last Queued: [${this.lastQueued.toString('hex')}]`
      )
    } catch (e) {
      logError(log, `Error initializing Message Submitter!`, e)
      throw e
    }
  }

  public async submitMessage(rollupMessage: L2ToL1Message): Promise<void> {
    if (rollupMessage.nonce <= this.lastQueued) {
      log.error(
        `submitMessage(...) called on old message. Last Queued: ${
          this.lastQueued
        }, received: ${JSON.stringify(rollupMessage)}`
      )
      return
    }

    log.info(`Queueing rollup message: ${JSON.stringify(rollupMessage)}}`)
    this.messageQueue.push(rollupMessage)
    await this.db.put(
      DefaultL2ToL1MessageSubmitter.getMessageKey(rollupMessage.nonce),
      DefaultL2ToL1MessageSubmitter.serializeRollupMessageForStorage(
        rollupMessage
      )
    )

    this.lastQueued = rollupMessage.nonce
    await this.db.put(
      DefaultL2ToL1MessageSubmitter.LAST_QUEUED_KEY,
      this.lastQueued.toBuffer()
    )

    await this.trySubmitNextMessage()
  }

  /**
   * Tries to submit the next message.
   * This will succeed if there is no pending message that has been submitted but not confirmed.
   */
  private async trySubmitNextMessage(): Promise<void> {
    // If message has been submitted and is pending, return
    if (
      this.lastSubmitted > this.lastConfirmed ||
      this.lastSubmitted >= this.lastQueued ||
      !this.messageQueue.length
    ) {
      if (!this.messageQueue.length) {
        log.info(`No messages queued for submission.`)
      } else {
        log.debug(
          `Next message queued but not submitted because message ${this.lastSubmitted} was submitted but not yet confirmed.`
        )
      }

      return
    }

    const message: L2ToL1Message = this.messageQueue[0]

    log.info(
      `Submitting message number ${message.nonce}: ${JSON.stringify(message)}.`
    )

    let receipt
    try {
      receipt = await this.messageReceiverContract.enqueueL2ToL1Message(
        DefaultL2ToL1MessageSubmitter.serializeRollupMessageForSubmission(
          message
        )
      )
      // TODO: do something with receipt?
    } catch (e) {
      logError(
        log,
        `Error submitting rollup message: ${JSON.stringify(message)}`,
        e
      )
      throw e
    }

    this.lastSubmitted = message.nonce
    await this.db.put(
      DefaultL2ToL1MessageSubmitter.LAST_SUBMITTED_KEY,
      this.lastSubmitted.toBuffer()
    )

    this.messageReceiverContract.provider
      .waitForTransaction(receipt)
      .then((txReceipt: TransactionReceipt) => {
        log.debug(
          `L2 -> L1 Message with nonce ${message.nonce.toString(
            'hex'
          )} was confirmed on L1!`
        )
      })
      .catch((error) => {
        logError(log, 'Error submitting L2 -> L1 message transaction', error)
      })
      .finally(async () => {
        this.lastConfirmed = message.nonce
        await this.db.put(
          DefaultL2ToL1MessageSubmitter.LAST_CONFIRMED_KEY,
          this.lastConfirmed.toBuffer()
        )

        // If we failed after submission but before storing submitted, update lastSubmitted
        if (this.lastSubmitted.lt(this.lastConfirmed)) {
          this.lastSubmitted = message.nonce
          await this.db.put(
            DefaultL2ToL1MessageSubmitter.LAST_SUBMITTED_KEY,
            this.lastSubmitted.toBuffer()
          )
        }
        // purposefully don't await
        this.trySubmitNextMessage()
      })
  }

  public static serializeRollupMessageForSubmission(
    rollupMessage: L2ToL1Message
  ): string {
    return abiEncodeL2ToL1Message(rollupMessage)
  }

  public static serializeRollupMessageForStorage(
    rollupMessage: L2ToL1Message
  ): Buffer {
    const rollupMessageString: string = abiEncodeL2ToL1Message(rollupMessage)
    return Buffer.from(rollupMessageString)
  }

  public static deserializeRollupMessageFromStorage(
    rollupMessageBuffer: Buffer
  ): L2ToL1Message {
    const serializedRollupMessage: string = rollupMessageBuffer.toString()
    return abiDecodeL2ToL1Message(serializedRollupMessage)
  }

  public static getMessageKey(messageNumber: BigNumber): Buffer {
    return Buffer.from(`MESSAGE_${add0x(messageNumber.toString('hex'))}`)
  }

  /***********
   * GETTERS *
   ***********/
  public getLastSubmitted(): BigNumber {
    return this.lastSubmitted
  }
  public getLastConfirmed(): BigNumber {
    return this.lastConfirmed
  }
  public getLastQueued(): BigNumber {
    return this.lastQueued
  }
}
