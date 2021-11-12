import { expect } from './setup'
import { Contract } from 'ethers'
import { ethers } from 'hardhat'
import { computeMessageHash, Watcher } from '../src'

describe('Watcher', () => {
  let watcher: Watcher
  let l1Messenger: Contract
  let l2Messenger: Contract
  beforeEach(async () => {
    const messengerFactory = await ethers.getContractFactory('MockMessenger')
    l1Messenger = await messengerFactory.deploy()
    l2Messenger = await messengerFactory.deploy()
    watcher = new Watcher({
      l1: {
        provider: ethers.provider,
        messengerAddress: l1Messenger.address,
      },
      l2: {
        provider: ethers.provider,
        messengerAddress: l2Messenger.address,
      },
    })
  })

  describe('computeMessageHash', () => {
    // TODO
  })

  describe('getMessageHashesFromL1Tx', () => {
    for (const numHashes of [1, 2, 4, 8, 16]) {
      it(`should be able to find ${numHashes} message hash(es) in a transaction`, async () => {
        // Arbitrary values
        const target = '0x' + '11'.repeat(20)
        const sender = '0x' + '22'.repeat(20)
        const message = '0x' + '33'.repeat(64)
        const messageNonce = 1234
        const gasLimit = 100000

        const tx = await l1Messenger.triggerSentMessageEvents(
          [...Array(numHashes)].map(() => {
            return {
              target,
              sender,
              message,
              messageNonce,
              gasLimit,
            }
          })
        )

        const messageHashes = await watcher.getMessageHashesFromL1Tx(tx.hash)
        expect(messageHashes).to.deep.equal(
          [...Array(numHashes)].map(() => {
            return computeMessageHash({
              target,
              sender,
              message,
              messageNonce,
            })
          })
        )
      })
    }

    it('should return empty if the transaction contains no messages', async () => {
      const tx = await l1Messenger.doNothing()
      const messageHashes = await watcher.getMessageHashesFromL1Tx(tx.hash)
      expect(messageHashes).to.deep.equal([])
    })

    it('should return empty if the transaction does not exist', async () => {
      const messageHashes = await watcher.getMessageHashesFromL1Tx(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      expect(messageHashes).to.deep.equal([])
    })
  })

  describe('getL1TransactionReceipt', () => {
    it('should get the L1 transaction receipt if the message has been relayed', async () => {
      // Arbitrary values
      const target = '0x' + '11'.repeat(20)
      const sender = '0x' + '22'.repeat(20)
      const message = '0x' + '33'.repeat(64)
      const messageNonce = 1234
      const gasLimit = 100000

      // Make it look like a message was sent on L2
      const l2Tx = await l2Messenger.triggerSentMessageEvents([
        {
          target,
          sender,
          message,
          messageNonce,
          gasLimit,
        },
      ])

      // Make it look like the message was relayed on L1
      const [messageHash] = await watcher.getMessageHashesFromL2Tx(l2Tx.hash)
      const l1Tx = await l1Messenger.triggerRelayedMessageEvents([messageHash])

      // Get the receipt found by the watcher
      const actualL1TxReceipt = await watcher.getL1TransactionReceipt(
        messageHash
      )

      // Get the actual relay receipt
      const expectedl1TxReceipt =
        await watcher.l2.provider.getTransactionReceipt(l1Tx.hash)

      // Compare
      expect(expectedl1TxReceipt).to.deep.equal(actualL1TxReceipt)
    })

    it('should get the L1 transaction receipt if the message has been executed but failed', async () => {
      // Arbitrary values
      const target = '0x' + '11'.repeat(20)
      const sender = '0x' + '22'.repeat(20)
      const message = '0x' + '33'.repeat(64)
      const messageNonce = 1234
      const gasLimit = 100000

      // Make it look like a message was sent on L2
      const l2Tx = await l2Messenger.triggerSentMessageEvents([
        {
          target,
          sender,
          message,
          messageNonce,
          gasLimit,
        },
      ])

      // Make it look like the message was relayed (but failed) on L1
      const [messageHash] = await watcher.getMessageHashesFromL2Tx(l2Tx.hash)
      const l1Tx = await l1Messenger.triggerFailedRelayedMessageEvents([
        messageHash,
      ])

      // Get the receipt found by the watcher
      const actualL1TxReceipt = await watcher.getL1TransactionReceipt(
        messageHash
      )

      // Get the actual relay receipt
      const expectedl1TxReceipt =
        await watcher.l2.provider.getTransactionReceipt(l1Tx.hash)

      // Compare
      expect(expectedl1TxReceipt).to.deep.equal(actualL1TxReceipt)
    })

    it('should wait for the message to be relayed if polling is enabled', async () => {
      // Arbitrary values
      const target = '0x' + '11'.repeat(20)
      const sender = '0x' + '22'.repeat(20)
      const message = '0x' + '33'.repeat(64)
      const messageNonce = 1234
      const gasLimit = 100000

      // Make it look like a message was sent on L2
      const l2Tx = await l2Messenger.triggerSentMessageEvents([
        {
          target,
          sender,
          message,
          messageNonce,
          gasLimit,
        },
      ])

      const [messageHash] = await watcher.getMessageHashesFromL2Tx(l2Tx.hash)

      // Relay the transaction after 10 seconds
      let l1Tx: any
      setTimeout(async () => {
        // Make it look like the message was relayed on L1
        l1Tx = await l1Messenger.triggerRelayedMessageEvents([messageHash])
      }, 10000)

      // Get the receipt found by the watcher
      const actualL1TxReceipt = await watcher.getL1TransactionReceipt(
        messageHash
      )

      // Get the actual relay receipt
      const expectedl1TxReceipt =
        await watcher.l2.provider.getTransactionReceipt(l1Tx.hash)

      // Compare
      expect(expectedl1TxReceipt).to.deep.equal(actualL1TxReceipt)
    }).timeout(50_000)

    it('should throw if it detects the same message relayed twice', async () => {
      // Arbitrary values
      const target = '0x' + '11'.repeat(20)
      const sender = '0x' + '22'.repeat(20)
      const message = '0x' + '33'.repeat(64)
      const messageNonce = 1234
      const gasLimit = 100000

      // Make it look like a message was sent on L2
      const l2Tx = await l2Messenger.triggerSentMessageEvents([
        {
          target,
          sender,
          message,
          messageNonce,
          gasLimit,
        },
      ])

      // Relay the message twice
      const [messageHash] = await watcher.getMessageHashesFromL2Tx(l2Tx.hash)
      await l1Messenger.triggerRelayedMessageEvents([messageHash])
      await l1Messenger.triggerRelayedMessageEvents([messageHash])

      // Get the receipt found by the watcher
      await expect(watcher.getL1TransactionReceipt(messageHash)).to.be.rejected
    })
  })
})
