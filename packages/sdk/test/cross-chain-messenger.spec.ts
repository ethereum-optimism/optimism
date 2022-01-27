import { Contract } from 'ethers'
import { ethers } from 'hardhat'
import { predeploys } from '@eth-optimism/contracts'

import { expect } from './setup'
import {
  CrossChainProvider,
  CrossChainMessenger,
  MessageDirection,
} from '../src'

describe('CrossChainMessenger', () => {
  let l1Signer: any
  let l2Signer: any
  before(async () => {
    ;[l1Signer, l2Signer] = await ethers.getSigners()
  })

  describe('sendMessage', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let provider: CrossChainProvider
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })

      messenger = new CrossChainMessenger({
        provider,
        l1Signer,
        l2Signer,
      })
    })

    describe('when the message is an L1 to L2 message', () => {
      describe('when no l2GasLimit is provided', () => {
        it('should send a message with an estimated l2GasLimit', async () => {
          const message = {
            direction: MessageDirection.L1_TO_L2,
            target: '0x' + '11'.repeat(20),
            message: '0x' + '22'.repeat(32),
          }

          const estimate = await provider.estimateL2MessageGasLimit(message)
          await expect(messenger.sendMessage(message))
            .to.emit(l1Messenger, 'SentMessage')
            .withArgs(
              message.target,
              await l1Signer.getAddress(),
              message.message,
              0,
              estimate
            )
        })
      })

      describe('when an l2GasLimit is provided', () => {
        it('should send a message with the provided l2GasLimit', async () => {
          const message = {
            direction: MessageDirection.L1_TO_L2,
            target: '0x' + '11'.repeat(20),
            message: '0x' + '22'.repeat(32),
          }

          await expect(
            messenger.sendMessage(message, {
              l2GasLimit: 1234,
            })
          )
            .to.emit(l1Messenger, 'SentMessage')
            .withArgs(
              message.target,
              await l1Signer.getAddress(),
              message.message,
              0,
              1234
            )
        })
      })
    })

    describe('when the message is an L2 to L1 message', () => {
      it('should send a message', async () => {
        const message = {
          direction: MessageDirection.L2_TO_L1,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        await expect(messenger.sendMessage(message))
          .to.emit(l2Messenger, 'SentMessage')
          .withArgs(
            message.target,
            await l2Signer.getAddress(),
            message.message,
            0,
            0
          )
      })
    })
  })

  describe('resendMessage', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let provider: CrossChainProvider
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })

      messenger = new CrossChainMessenger({
        provider,
        l1Signer,
        l2Signer,
      })
    })

    describe('when resending an L1 to L2 message', () => {
      it('should resend the message with the new gas limit', async () => {
        const message = {
          direction: MessageDirection.L1_TO_L2,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        const sent = await messenger.sendMessage(message, {
          l2GasLimit: 1234,
        })

        await expect(messenger.resendMessage(sent, 10000))
          .to.emit(l1Messenger, 'SentMessage')
          .withArgs(
            message.target,
            await l1Signer.getAddress(),
            message.message,
            1, // nonce is now 1
            10000
          )
      })
    })

    describe('when resending an L2 to L1 message', () => {
      it('should throw an error', async () => {
        const message = {
          direction: MessageDirection.L2_TO_L1,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        const sent = await messenger.sendMessage(message, {
          l2GasLimit: 1234,
        })

        await expect(messenger.resendMessage(sent, 10000)).to.be.rejected
      })
    })
  })

  describe('finalizeMessage', () => {
    describe('when the message being finalized exists', () => {
      describe('when the message is ready to be finalized', () => {
        it('should finalize the message')
      })

      describe('when the message is not ready to be finalized', () => {
        it('should throw an error')
      })

      describe('when the message has already been finalized', () => {
        it('should throw an error')
      })
    })

    describe('when the message being finalized does not exist', () => {
      it('should throw an error')
    })
  })

  describe('depositETH', () => {
    let l1Messenger: Contract
    let l1Bridge: Contract
    let provider: CrossChainProvider
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l1Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l1Messenger.address)) as any

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            L1StandardBridge: l1Bridge.address,
          },
        },
      })

      messenger = new CrossChainMessenger({
        provider,
        l1Signer,
        l2Signer,
      })
    })

    it('should trigger the deposit ETH function with the given amount', async () => {
      await expect(messenger.depositETH(100000))
        .to.emit(l1Bridge, 'ETHDepositInitiated')
        .withArgs(
          await l1Signer.getAddress(),
          await l1Signer.getAddress(),
          100000,
          '0x'
        )
    })
  })

  describe('withdrawETH', () => {
    let l2Messenger: Contract
    let l2Bridge: Contract
    let provider: CrossChainProvider
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l2Messenger.address)) as any

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
        contracts: {
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
            L2StandardBridge: l2Bridge.address,
          },
        },
      })

      messenger = new CrossChainMessenger({
        provider,
        l1Signer,
        l2Signer,
      })
    })

    it('should trigger the deposit ETH function with the given amount', async () => {
      await expect(messenger.withdrawETH(100000))
        .to.emit(l2Bridge, 'WithdrawalInitiated')
        .withArgs(
          ethers.constants.AddressZero,
          predeploys.OVM_ETH,
          await l2Signer.getAddress(),
          await l2Signer.getAddress(),
          100000,
          '0x'
        )
    })
  })
})
