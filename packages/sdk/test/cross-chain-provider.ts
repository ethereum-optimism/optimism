import { expect } from './setup'
import { CrossChainProvider } from '../src'
import { ethers } from 'hardhat'
import { Contract } from 'ethers'

describe('CrossChainProvider', () => {
  let L1CrossDomainMessenger: Contract
  beforeEach(async () => {
    const L1CrossDomainMessengerFactory = await ethers.getContractFactory(
      'MockMessenger'
    )
    L1CrossDomainMessenger =
      (await L1CrossDomainMessengerFactory.deploy()) as any
  })

  describe('standard construction', () => {
    let provider: CrossChainProvider
    beforeEach(async () => {
      provider = new CrossChainProvider({
        // TODO: Fix this junk.
        l1Provider: ethers.provider as any,
        l2Provider: ethers.provider as any,
        contracts: {
          l1: {
            L1CrossDomainMessenger,
          },
        },
      })
    })

    describe('getMessagesByTransaction', () => {
      it('should throw an error if the transaction hash does not exist', async () => {
        await expect(
          provider.getMessagesByTransaction('0x' + '00'.repeat(32))
        ).to.be.rejectedWith('unable to find transaction receipt')
      })

      it('should be ok if there are no messages in the transaction', async () => {
        const tx = await L1CrossDomainMessenger.doNothing()
        const messages = await provider.getMessagesByTransaction(tx)
        expect(messages).to.deep.equal([])
      })

      it('should be able to find a single message in an L1 transaction', async () => {
        const tx = await L1CrossDomainMessenger.triggerSentMessageEvents([
          {
            target: '0x' + '11'.repeat(20),
            sender: '0x' + '22'.repeat(20),
            message: '0x1234',
            messageNonce: 1234,
            gasLimit: 100_000,
          },
        ])

        const messages = await provider.getMessagesByTransaction(tx)
        console.log(messages)
      })

      it('should be able to find multiple messages in an L1 transaction', async () => {
        const tx = await L1CrossDomainMessenger.triggerSentMessageEvents([
          {
            target: '0x' + '11'.repeat(20),
            sender: '0x' + '22'.repeat(20),
            message: '0x1234',
            messageNonce: 1234,
            gasLimit: 100_000,
          },
          {
            target: '0x' + '11'.repeat(20),
            sender: '0x' + '22'.repeat(20),
            message: '0x1234',
            messageNonce: 1234,
            gasLimit: 100_000,
          },
        ])

        const messages = await provider.getMessagesByTransaction(tx)
        console.log(messages)
      })
    })
  })
})
