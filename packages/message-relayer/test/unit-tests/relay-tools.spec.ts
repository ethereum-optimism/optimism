import { expect } from '../setup'

/* Imports: External */
import hre from 'hardhat'
import { Contract, ethers } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import { getSentMessages, parseSentMessageEvent } from '../../src/relay-tools'
import { SentMessage } from '../../src/types'

const encodeSentMessage = (
  target: string,
  sender: string,
  message: string,
  messageNonce: number
): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [target, sender, message, messageNonce])
}

const makeSentMessageStruct = (
  target: string,
  sender: string,
  message: string,
  messageNonce: number,
  blockNumber: number,
  l2BlockOffset: number,
  transactionHash: string
): SentMessage => {
  const encodedMessage = encodeSentMessage(
    target,
    sender,
    message,
    messageNonce
  )

  return {
    target,
    sender,
    message,
    messageNonce,
    encodedMessage,
    encodedMessageHash: ethers.utils.keccak256(encodedMessage),
    parentTransactionIndex: blockNumber - l2BlockOffset,
    parentTransactionHash: transactionHash,
  }
}

describe('message relayer utilities', () => {
  describe('parseSentMessageEvent', () => {
    it('should throw if the event has no "message" argument', () => {
      expect(() => {
        parseSentMessageEvent(
          {
            args: {},
          } as any,
          0
        )
      }).to.throw('event is not a SentMessage event')
    })

    it('should throw if the attached message is not correctly encoded', () => {
      expect(() => {
        parseSentMessageEvent(
          {
            args: {
              message: '0x12341234', // not valid
            },
          } as any,
          0
        )
      }).to.throw('unable to parse SentMessage event')
    })

    it('should successfully parse a valid SentMessage event', () => {
      const target = `0x${'11'.repeat(20)}`
      const sender = `0x${'22'.repeat(20)}`
      const message = `0x1234567890`
      const messageNonce = 1234
      const blockNumber = 5678
      const l2BlockOffset = 0
      const transactionHash = ethers.constants.HashZero

      expect(
        parseSentMessageEvent(
          {
            args: {
              message: encodeSentMessage(target, sender, message, messageNonce),
            },
            blockNumber,
            transactionHash,
          } as any,
          l2BlockOffset
        )
      ).to.deep.equal(
        makeSentMessageStruct(
          target,
          sender,
          message,
          messageNonce,
          blockNumber,
          l2BlockOffset,
          transactionHash
        )
      )
    })

    it('should account for the l2BlockOffset', () => {
      const target = `0x${'11'.repeat(20)}`
      const sender = `0x${'22'.repeat(20)}`
      const message = `0x1234567890`
      const messageNonce = 1234
      const blockNumber = 5678
      const l2BlockOffset = 1234
      const transactionHash = ethers.constants.HashZero

      expect(
        parseSentMessageEvent(
          {
            args: {
              message: encodeSentMessage(target, sender, message, messageNonce),
            },
            blockNumber,
            transactionHash,
          } as any,
          l2BlockOffset
        )
      ).to.deep.equal(
        makeSentMessageStruct(
          target,
          sender,
          message,
          messageNonce,
          blockNumber,
          l2BlockOffset,
          transactionHash
        )
      )
    })
  })

  describe('getSentMessages', () => {
    let mockL2CrossDomainMessenger: Contract
    beforeEach(async () => {
      const factory = await (hre as any).ethers.getContractFactory(
        'MockL2CrossDomainMessenger'
      )
      mockL2CrossDomainMessenger = await factory.deploy()
    })

    it('should throw if end height is less than start height', async () => {
      const startHeight = 1234
      const endHeight = startHeight - 1

      await expect(
        getSentMessages(mockL2CrossDomainMessenger, 0, startHeight, endHeight)
      ).to.be.rejectedWith('end height must be greater than start height')
    })

    it('should throw if end height is equal to start height', async () => {
      const startHeight = 1234
      const endHeight = startHeight

      await expect(
        getSentMessages(mockL2CrossDomainMessenger, 0, startHeight, endHeight)
      ).to.be.rejectedWith('end height must be greater than start height')
    })

    it('should throw if provided a contract without a SentMessage event', async () => {
      const startHeight = 1234
      const endHeight = startHeight + 1

      const factory = await (hre as any).ethers.getContractFactory(
        'EmptyContract'
      )
      const contract = await factory.deploy()

      await expect(
        getSentMessages(contract, 0, startHeight, endHeight)
      ).to.be.rejectedWith('SentMessage filter not found on provided contract')
    })

    it('should find a single event in a single block', async () => {
      const startHeight =
        (await (hre as any).ethers.provider.getBlockNumber()) + 1
      const endHeight = startHeight + 1

      const target = `0x${'11'.repeat(20)}`
      const sender = `0x${'22'.repeat(20)}`
      const message = `0x1234567890`
      const messageNonce = 1234

      const result = await mockL2CrossDomainMessenger.emitSentMessageEvent({
        target,
        sender,
        message,
        messageNonce,
      })

      expect(
        await getSentMessages(
          mockL2CrossDomainMessenger,
          0,
          startHeight,
          endHeight
        )
      ).to.deep.equal([
        makeSentMessageStruct(
          target,
          sender,
          message,
          messageNonce,
          startHeight,
          0,
          result.hash
        ),
      ])
    })

    it('should find a multiple events in a single block', async () => {
      const startHeight =
        (await (hre as any).ethers.provider.getBlockNumber()) + 1
      const endHeight = startHeight + 1

      const messages = [
        {
          target: `0x${'11'.repeat(20)}`,
          sender: `0x${'22'.repeat(20)}`,
          message: `0x1234567890`,
          messageNonce: 1234,
        },
        {
          target: `0x${'33'.repeat(20)}`,
          sender: `0x${'44'.repeat(20)}`,
          message: `0x112233445566778899`,
          messageNonce: 120595,
        },
        {
          target: `0x${'55'.repeat(20)}`,
          sender: `0x${'66'.repeat(20)}`,
          message: `0x95191359010915905123919040912901230230125125`,
          messageNonce: 1205512551,
        },
      ]

      const result = await mockL2CrossDomainMessenger.emitMultipleSentMessageEvents(
        messages
      )

      expect(
        await getSentMessages(
          mockL2CrossDomainMessenger,
          0,
          startHeight,
          endHeight
        )
      ).to.deep.equal(
        messages.map((message) => {
          return makeSentMessageStruct(
            message.target,
            message.sender,
            message.message,
            message.messageNonce,
            startHeight,
            0,
            result.hash
          )
        })
      )
    })

    it('should find a single event in multiple blocks', async () => {
      const factory = await (hre as any).ethers.getContractFactory(
        'DummyContract'
      )
      const dummy = await factory.deploy()

      const startHeight =
        (await (hre as any).ethers.provider.getBlockNumber()) + 1

      const target = `0x${'11'.repeat(20)}`
      const sender = `0x${'22'.repeat(20)}`
      const message = `0x1234567890`
      const messageNonce = 1234

      const result = await mockL2CrossDomainMessenger.emitSentMessageEvent({
        target,
        sender,
        message,
        messageNonce,
      })

      for (let i = 0; i < 10; i++) {
        await dummy.triggerBlockProduction()
      }

      const endHeight = await (hre as any).ethers.provider.getBlockNumber()

      expect(
        await getSentMessages(
          mockL2CrossDomainMessenger,
          0,
          startHeight,
          endHeight
        )
      ).to.deep.equal([
        makeSentMessageStruct(
          target,
          sender,
          message,
          messageNonce,
          startHeight,
          0,
          result.hash
        ),
      ])
    })
  })
})
