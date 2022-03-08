/* External Imports */
import hre, { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { applyL1ToL2Alias } from '@eth-optimism/core-utils'
import {
  smock,
  MockContractFactory,
  FakeContract,
} from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../../setup'
import {
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
  encodeXDomainCalldata,
} from '../../../helpers'
import { predeploys } from '../../../../src'

describe('L2CrossDomainMessenger', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Fake__TargetContract: FakeContract
  let Fake__L1CrossDomainMessenger: FakeContract
  let Fake__OVM_L2ToL1MessagePasser: FakeContract
  before(async () => {
    Fake__TargetContract = await smock.fake<Contract>(
      await ethers.getContractFactory('Helper_SimpleProxy')
    )
    Fake__L1CrossDomainMessenger = await smock.fake<Contract>(
      await ethers.getContractFactory('L1CrossDomainMessenger')
    )
    Fake__OVM_L2ToL1MessagePasser = await smock.fake<Contract>(
      await ethers.getContractFactory('OVM_L2ToL1MessagePasser'),
      { address: predeploys.OVM_L2ToL1MessagePasser }
    )
  })

  let impersonatedL1CrossDomainMessengerSender: Signer
  before(async () => {
    const impersonatedAddress = applyL1ToL2Alias(
      Fake__L1CrossDomainMessenger.address
    )
    await hre.network.provider.request({
      method: 'hardhat_impersonateAccount',
      params: [impersonatedAddress],
    })
    await hre.network.provider.request({
      method: 'hardhat_setBalance',
      params: [impersonatedAddress, '0xFFFFFFFFFFFFFFFFF'],
    })
    impersonatedL1CrossDomainMessengerSender = await ethers.getSigner(
      impersonatedAddress
    )
  })

  let Factory__L2CrossDomainMessenger: ContractFactory
  before(async () => {
    Factory__L2CrossDomainMessenger = await ethers.getContractFactory(
      'L2CrossDomainMessenger'
    )
  })

  let L2CrossDomainMessenger: Contract
  beforeEach(async () => {
    L2CrossDomainMessenger = await Factory__L2CrossDomainMessenger.deploy(
      Fake__L1CrossDomainMessenger.address
    )
  })

  describe('xDomainMessageSender', () => {
    let Mock__Factory__L2CrossDomainMessenger: MockContractFactory<ContractFactory>
    let Mock__L2CrossDomainMessenger
    before(async () => {
      Mock__Factory__L2CrossDomainMessenger = await smock.mock(
        'L2CrossDomainMessenger'
      )
      Mock__L2CrossDomainMessenger =
        await Mock__Factory__L2CrossDomainMessenger.deploy(
          Fake__L1CrossDomainMessenger.address
        )
    })

    it('should return the xDomainMsgSender address', async () => {
      await Mock__L2CrossDomainMessenger.setVariable(
        'xDomainMsgSender',
        '0x0000000000000000000000000000000000000000'
      )
      expect(
        await Mock__L2CrossDomainMessenger.xDomainMessageSender()
      ).to.equal('0x0000000000000000000000000000000000000000')
    })
  })

  describe('sendMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should be able to send a single message', async () => {
      await expect(
        L2CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted

      expect(
        Fake__OVM_L2ToL1MessagePasser.passMessageToL1.getCall(0).args[0]
      ).to.deep.equal(
        encodeXDomainCalldata(target, await signer.getAddress(), message, 0)
      )
    })

    it('should be able to send the same message twice', async () => {
      await L2CrossDomainMessenger.sendMessage(target, message, gasLimit)

      await expect(
        L2CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted
    })
  })

  describe('relayMessage', () => {
    let target: string
    let message: string
    let sender: string
    before(async () => {
      target = Fake__TargetContract.address
      message = Fake__TargetContract.interface.encodeFunctionData('setTarget', [
        NON_ZERO_ADDRESS,
      ])
      sender = await signer.getAddress()
    })

    it('should revert if the L1 message sender is not the L1CrossDomainMessenger', async () => {
      await expect(
        L2CrossDomainMessenger.connect(signer).relayMessage(
          target,
          sender,
          message,
          0
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should send a call to the target contract', async () => {
      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(target, sender, message, 0)

      expect(Fake__TargetContract.setTarget.getCall(0).args[0]).to.deep.equal(
        NON_ZERO_ADDRESS
      )
    })

    it('the xDomainMessageSender is reset to the original value', async () => {
      await expect(
        L2CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')

      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(target, sender, message, 0)

      await expect(
        L2CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
    })

    it('should revert if trying to send the same message twice', async () => {
      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(target, sender, message, 0)

      await expect(
        L2CrossDomainMessenger.connect(
          impersonatedL1CrossDomainMessengerSender
        ).relayMessage(target, sender, message, 0)
      ).to.be.revertedWith('Provided message has already been received.')
    })

    it('should not make a call if the target is the L2 MessagePasser', async () => {
      target = predeploys.OVM_L2ToL1MessagePasser
      message = Fake__OVM_L2ToL1MessagePasser.interface.encodeFunctionData(
        'passMessageToL1(bytes)',
        [NON_NULL_BYTES32]
      )

      const resProm = L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(target, sender, message, 0)

      // The call to relayMessage() should succeed.
      await expect(resProm).to.not.be.reverted

      // There should be no 'relayedMessage' event logged in the receipt.
      const logs = (
        await Fake__OVM_L2ToL1MessagePasser.provider.getTransactionReceipt(
          (
            await resProm
          ).hash
        )
      ).logs
      expect(logs).to.deep.equal([])

      // The message should be registered as successful.
      expect(
        await L2CrossDomainMessenger.successfulMessages(
          ethers.utils.solidityKeccak256(
            ['bytes'],
            [encodeXDomainCalldata(target, sender, message, 0)]
          )
        )
      ).to.be.true
    })
  })
})
