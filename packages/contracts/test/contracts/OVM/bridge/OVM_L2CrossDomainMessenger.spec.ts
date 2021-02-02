import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  NON_NULL_BYTES32,
  ZERO_ADDRESS,
  NON_ZERO_ADDRESS,
} from '../../../helpers'
import { getContractInterface } from '../../../../src'

const getXDomainCalldata = (
  sender: string,
  target: string,
  message: string,
  messageNonce: number
): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [target, sender, message, messageNonce])
}

describe('OVM_L2CrossDomainMessenger', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__TargetContract: MockContract
  let Mock__OVM_L1CrossDomainMessenger: MockContract
  let Mock__OVM_L1MessageSender: MockContract
  let Mock__OVM_L2ToL1MessagePasser: MockContract
  before(async () => {
    Mock__TargetContract = await smockit(
      await ethers.getContractFactory('Helper_SimpleProxy')
    )
    Mock__OVM_L1CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L1CrossDomainMessenger')
    )
    Mock__OVM_L1MessageSender = await smockit(
      await ethers.getContractFactory('OVM_L1MessageSender')
    )
    Mock__OVM_L2ToL1MessagePasser = await smockit(
      await ethers.getContractFactory('OVM_L2ToL1MessagePasser')
    )

    await AddressManager.setAddress(
      'OVM_L1CrossDomainMessenger',
      Mock__OVM_L1CrossDomainMessenger.address
    )

    await setProxyTarget(
      AddressManager,
      'OVM_L1MessageSender',
      Mock__OVM_L1MessageSender
    )
    await setProxyTarget(
      AddressManager,
      'OVM_L2ToL1MessagePasser',
      Mock__OVM_L2ToL1MessagePasser
    )
  })

  let Factory__OVM_L2CrossDomainMessenger: ContractFactory
  before(async () => {
    Factory__OVM_L2CrossDomainMessenger = await ethers.getContractFactory(
      'OVM_L2CrossDomainMessenger'
    )
  })

  let OVM_L2CrossDomainMessenger: Contract
  beforeEach(async () => {
    OVM_L2CrossDomainMessenger = await Factory__OVM_L2CrossDomainMessenger.deploy(
      AddressManager.address
    )
  })

  describe('sendMessage', () => {
    const target = NON_ZERO_ADDRESS
    const message = NON_NULL_BYTES32
    const gasLimit = 100_000

    it('should be able to send a single message', async () => {
      await expect(
        OVM_L2CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted

      expect(
        Mock__OVM_L2ToL1MessagePasser.smocked.passMessageToL1.calls[0]
      ).to.deep.equal([
        getXDomainCalldata(await signer.getAddress(), target, message, 0),
      ])
    })

    it('should be able to send the same message twice', async () => {
      await OVM_L2CrossDomainMessenger.sendMessage(target, message, gasLimit)

      await expect(
        OVM_L2CrossDomainMessenger.sendMessage(target, message, gasLimit)
      ).to.not.be.reverted
    })
  })

  describe('relayMessage', () => {
    let target: string
    let message: string
    let sender: string
    before(async () => {
      target = Mock__TargetContract.address
      message = Mock__TargetContract.interface.encodeFunctionData('setTarget', [
        NON_ZERO_ADDRESS,
      ])
      sender = await signer.getAddress()
    })

    beforeEach(async () => {
      Mock__OVM_L1MessageSender.smocked.getL1MessageSender.will.return.with(
        Mock__OVM_L1CrossDomainMessenger.address
      )
    })

    it('should revert if the L1 message sender is not the OVM_L1CrossDomainMessenger', async () => {
      Mock__OVM_L1MessageSender.smocked.getL1MessageSender.will.return.with(
        ZERO_ADDRESS
      )

      await expect(
        OVM_L2CrossDomainMessenger.relayMessage(target, sender, message, 0)
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should send a call to the target contract', async () => {
      await OVM_L2CrossDomainMessenger.relayMessage(target, sender, message, 0)

      expect(Mock__TargetContract.smocked.setTarget.calls[0]).to.deep.equal([
        NON_ZERO_ADDRESS,
      ])
    })

    it('should revert if trying to send the same message twice', async () => {
      Mock__OVM_L1MessageSender.smocked.getL1MessageSender.will.return.with(
        Mock__OVM_L1CrossDomainMessenger.address
      )

      await OVM_L2CrossDomainMessenger.relayMessage(target, sender, message, 0)

      await expect(
        OVM_L2CrossDomainMessenger.relayMessage(target, sender, message, 0)
      ).to.be.revertedWith('Provided message has already been received.')
    })
  })
})
