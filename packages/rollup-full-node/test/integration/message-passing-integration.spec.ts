import { Address, L2ToL1Message } from '@eth-optimism/rollup-core'
import { add0x } from '@eth-optimism/core-utils'

import {
  createProviderForHandler,
  deployContract,
  getWallets,
  TestWeb3Handler,
} from '../../src/app'
import { L2ToL1MessageSubmitter } from '../../src/types'

import * as SimpleStorage from '../contracts/build/transpiled/SimpleStorage.json'
import * as SimpleCaller from '../contracts/build/transpiled/SimpleCaller.json'
import * as MessagePasserUtil from '../contracts/build/transpiled/L2ToL1MessageUtil.json'

class QueuedMessageSubmitter implements L2ToL1MessageSubmitter {
  private readonly receivedMessages: L2ToL1Message[] = []
  public async submitMessage(l2ToL1Message: L2ToL1Message): Promise<void> {
    this.receivedMessages.push(l2ToL1Message)
  }

  public getReceivedMessages(): L2ToL1Message[] {
    return this.receivedMessages
  }

  public clearMessages(): void {
    this.receivedMessages.length = 0
  }
}

describe('Message Passing Integration Tests', () => {
  let messageSubmitter: QueuedMessageSubmitter
  let handler: TestWeb3Handler
  let provider
  let wallet

  beforeEach(async () => {
    messageSubmitter = new QueuedMessageSubmitter()
    handler = await TestWeb3Handler.create(messageSubmitter)
    provider = createProviderForHandler(handler)
    wallet = getWallets(provider)[0]
  })

  describe('L1 Message Passing Tests', () => {
    it('Should not queue any messages if ', async () => {
      const simpleStorage = await deployContract(wallet, SimpleStorage, [], [])
      const simpleCaller = await deployContract(wallet, SimpleCaller, [], [])

      const storageKey = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '02'.repeat(32)

      await simpleStorage.setStorage(storageKey, storageValue)

      const res = await simpleCaller.doGetStorageCall(
        simpleStorage.address,
        storageKey
      )
      res.should.equal(storageValue)

      messageSubmitter
        .getReceivedMessages()
        .length.should.equal(0, 'Should not receive any L2ToL1Messages!')
    })

    it('Should not queue any messages if event is not from message passer', async () => {
      const messagePasserFraud = await deployContract(
        wallet,
        MessagePasserUtil,
        [],
        []
      )

      await messagePasserFraud.emitFraudulentMessage()

      messageSubmitter
        .getReceivedMessages()
        .length.should.equal(0, 'Should not receive any L2ToL1Messages!')
    })

    it('Should queue messages from message passer', async () => {
      const messagePasserFraud = await deployContract(
        wallet,
        MessagePasserUtil,
        [],
        []
      )
      const messagePasserAddress: Address = handler.getL2ToL1MessagePasserAddress()

      const message: string = add0x(
        Buffer.from(`Roads? Where we're going, we don't need roads.`).toString(
          'hex'
        )
      )
      await messagePasserFraud.callMessagePasser(messagePasserAddress, message)
      await messagePasserFraud.callMessagePasser(messagePasserAddress, message)

      const receivedMessages = messageSubmitter.getReceivedMessages()
      receivedMessages.length.should.equal(
        2,
        'Should receive any L2ToL1Message!'
      )
      const msg: L2ToL1Message = receivedMessages[0]
      msg.callData.should.equal(message, 'Message mismatch!')
      msg.nonce.should.equal(0, 'Nonce mismatch!')
      msg.ovmSender.should.equal(
        messagePasserFraud.address,
        'Address mismatch!'
      )

      receivedMessages[1].nonce.should.equal(1, 'Nonce Mismatch!')
    })
  })
})
